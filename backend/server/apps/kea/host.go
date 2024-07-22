package kea

import (
	"context"

	"github.com/go-pg/pg/v10"
	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/configreview"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

const (
	// A limit for returned hosts number in the reservation-get-page command.
	defaultHostCmdsPageLimit int64 = 1000
)

// A structure reflecting "next" map of the Kea response to the
// reservation-get-page command.
type ReservationGetPageNext struct {
	From        int64
	SourceIndex int64 `json:"source-index"`
}

// A structure reflecting arguments of the Kea response to the
// reservation-get-page command.
type ReservationGetPageArgs struct {
	Count int64
	Hosts []keaconfig.Reservation
	Next  ReservationGetPageNext
}

// A structure reflecting Kea response to the reservation-get-page command.
type ReservationGetPageResponse struct {
	keactrl.ResponseHeader
	Arguments *ReservationGetPageArgs `json:"arguments,omitempty"`
}

// Instance of the puller that periodically pulls host reservations from
// the Kea apps.
type HostsPuller struct {
	*agentcomm.PeriodicPuller
	ReviewDispatcher           configreview.Dispatcher
	DHCPOptionDefinitionLookup keaconfig.DHCPOptionDefinitionLookup
	traces                     map[int64]*hostIteratorTrace
}

// Create an instance of the puller that periodically fetches host reservations
// from the monitored Kea apps via control channel.
func NewHostsPuller(db *dbops.PgDB, agents agentcomm.ConnectedAgents, reviewDispatcher configreview.Dispatcher, lookup keaconfig.DHCPOptionDefinitionLookup) (*HostsPuller, error) {
	hostsPuller := &HostsPuller{
		ReviewDispatcher:           reviewDispatcher,
		DHCPOptionDefinitionLookup: lookup,
	}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Kea Hosts puller", "kea_hosts_puller_interval",
		hostsPuller.pull)
	if err != nil {
		return nil, err
	}
	hostsPuller.PeriodicPuller = periodicPuller
	hostsPuller.traces = make(map[int64]*hostIteratorTrace)
	return hostsPuller, nil
}

// Stops the timer triggering pulling host reservations from the monitored apps.
func (puller *HostsPuller) Shutdown() {
	puller.PeriodicPuller.Shutdown()
}

// Pulls host reservations from the monitored Kea apps and updates them in
// the Stork database.
func (puller *HostsPuller) pull() error {
	// Get all Kea apps from the database.
	apps, err := dbmodel.GetAppsByType(puller.DB, dbmodel.AppTypeKea)
	if err != nil {
		return err
	}

	var (
		successCount int
		skippedCount int
		erredCount   int
	)

	// Iterate over the Kea apps and attempt to pull host reservations
	// from them via the host_cmds hooks library. Next, update the
	// hosts in the Stork database.
	for i := range apps {
		success, skip, e := puller.pullFromApp(&apps[i])
		successCount += success
		skippedCount += skip
		erredCount += e
	}

	// Remove the hosts that no longer belong to any app.
	_, err = dbmodel.DeleteOrphanedHosts(puller.DB)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"success_count": successCount,
		"skipped_count": skippedCount,
		"erred_count":   erredCount,
	}).Info("Completed pulling hosts from Kea daemons")

	return nil
}

// Pulls all host reservations stored in the hosts backend for the particular
// Kea app. The returned values are the daemon counters for which the pull
// was successful, skipped and/or erred. They are used for logging purposes.
func (puller *HostsPuller) pullFromApp(app *dbmodel.App) (successCount, skippedCount, erredCount int) {
	for _, daemon := range app.Daemons {
		if daemon.KeaDaemon.KeaDHCPDaemon == nil {
			continue
		}
		pulled, err := puller.pullFromDaemon(app, daemon)
		if err != nil {
			erredCount++
			log.WithError(err).Errorf("Problem pulling Kea hosts from daemon %d", daemon.ID)
			continue
		}
		if !pulled {
			skippedCount++
			continue
		}
		successCount++
	}
	return
}

// Pulls all host reservations stored in the hosts backend for the particular
// Kea daemon. The first returned value indicates whether or not the function
// attempted to pull the reservations (i.e., host_cmds hook library is used by
// the daemon and the daemon is active). The function uses the iterator mechanism
// to pull the hosts. It can result in sending multiple reservation-get-page
// commands to each Kea instance.
func (puller *HostsPuller) pullFromDaemon(app *dbmodel.App, daemon *dbmodel.Daemon) (bool, error) {
	// Hosts update is performed in a transaction but we don't begin the
	// transaction until we detect that there were some changes in the
	// host reservations.
	var (
		tx  *pg.Tx
		err error
	)

	if !daemon.Active {
		log.Infof("Skip pulling host reservations for inactive daemon %d", daemon.ID)
		return false, nil
	}

	if daemon.KeaDaemon.Config == nil {
		err = errors.Errorf("daemon %d lacks configuration", daemon.ID)
		return false, err
	}

	// Ensure that the daemon has host_cmds hooks library loaded. Otherwise,
	// we're unable to get the hosts from this daemon.
	config := daemon.KeaDaemon.Config
	if _, _, ok := config.GetHookLibrary("libdhcp_host_cmds"); !ok {
		log.Infof("Skip pulling host reservations for daemon %d because it lacks host_cmds hook", daemon.ID)
		return false, nil
	}

	// Fetch the hosts as long as they are returned by Kea.
	it := newHostIterator(puller.DB, app, daemon, puller.Agents, defaultHostCmdsPageLimit)
	var done bool
	for !done {
		// Send reservation-get-page to Kea.
		var reservations []keaconfig.Reservation
		if reservations, done, err = it.getPageFromHostCmds(); err != nil {
			return true, err
		}
		// We're probably done getting reservations.
		if len(reservations) == 0 {
			continue
		}
		// The puller holds traces from the previous attempts to fetch the host
		// reservations. The traces don't exist when this is the first time
		// we pull the reservations.
		if trace, ok := puller.traces[daemon.ID]; ok {
			// Compare the hash created from the received reservations with the
			// corresponding hash in the saved trace. If they are equal it means
			// that there was no change in the host reservations since the last
			// pull.
			if it.trace.hasEqualHashes(trace) {
				continue
			}
		}
		// If we have detected host changes right now, there is no transaction yet.
		// Begin new transaction.
		if tx == nil {
			if tx, err = puller.DB.Begin(); err != nil {
				err = errors.Wrapf(err, "problem starting transaction to add hosts from host_cmds hooks library for daemon %d", daemon.ID)
				return true, err
			}
			defer dbops.RollbackOnError(tx, &err)

			// Remove associations between existing host reservations and the
			// daemon. Some associations will be re-created and some possibly
			// not. The orphaned hosts will be later removed.
			if _, err = dbmodel.DeleteDaemonFromHosts(tx, daemon.ID, dbmodel.HostDataSourceAPI); err != nil {
				return true, err
			}

			// We possibly cached some responses for which the hashes haven't
			// changed. Since we created new transaction and had to remove the
			// associations with the daemon we now have to process these cached
			// responses to add the associations with the hosts that haven't
			// changed.
			for _, traceResponse := range it.trace.responses {
				if err = convertAndUpdateHosts(tx, daemon, it.getSubnet(traceResponse.subnetIndex), traceResponse.hosts, puller.DHCPOptionDefinitionLookup); err != nil {
					return true, err
				}
			}
		}
		// Add the host reservations from the current response.
		if err = convertAndUpdateHosts(tx, daemon, it.getCurrentSubnet(), reservations, puller.DHCPOptionDefinitionLookup); err != nil {
			return true, err
		}
	}

	// Only commit if we have started transaction. It means that we
	// detected host changes.
	if tx != nil {
		if err = tx.Commit(); err != nil {
			err = errors.Wrapf(err, "problem committing transaction to add new hosts from host_cmds hooks library for daemon %d", daemon.ID)
			return true, err
		}
		_ = puller.ReviewDispatcher.BeginReview(daemon, []configreview.Trigger{configreview.DBHostsModified}, nil)
	}

	// We no longer need the hosts.
	for i := range it.trace.responses {
		it.trace.responses[i].hosts = nil
	}

	// Remember the current trace.
	puller.traces[daemon.ID] = it.trace

	return true, nil
}

// A structure used by the host iterator to remember host reservations
// and hash values created from the host reservations for each
// reservation-get-page command. The hashes can be later used to check
// that Kea returns the same (or different) responses when Stork next
// attempts to fetch hosts, without making a detailed comparison of the
// responses. If the current hashes match the hashes from the previous
// iteration, there is no need to update the hosts in the database,
// perform config reviews etc.
type hostIteratorTrace struct {
	responses []hostIteratorTraceResponse
}

// A structure representing a single hosts page returned by Kea. It
// includes a hash created from the returned hosts, an index of the
// subnet (within the iterator) for which the hosts have been returned
// and the returned reservations.
type hostIteratorTraceResponse struct {
	hash        string
	subnetIndex int
	hosts       []keaconfig.Reservation
}

// Creates new trace instance.
func newHostIteratorTrace() *hostIteratorTrace {
	return &hostIteratorTrace{}
}

// Appends the specified response increasing the response count.
func (trace *hostIteratorTrace) addResponse(hash string, subnetIndex int, hosts []keaconfig.Reservation) {
	trace.responses = append(trace.responses, hostIteratorTraceResponse{hash, subnetIndex, hosts})
}

// Returns a number of responses currently held.
func (trace *hostIteratorTrace) getResponseCount() int {
	return len(trace.responses)
}

// Checks if all hashes in the trace are equal to the hashes at the
// same positions in the other trace.
func (trace *hostIteratorTrace) hasEqualHashes(other *hostIteratorTrace) bool {
	// If there are more hashes in this trace than in the other
	// trace those additional hashes are unequal.
	if len(trace.responses) > len(other.responses) {
		return false
	}
	// Iterate from the newest to the oldest response. It should
	// marginally improve performance because in a typical case
	// we have checked that the older hashes are equal.
	for i := len(trace.responses) - 1; i >= 0; i-- {
		if trace.responses[i].hash != other.responses[i].hash {
			return false
		}
	}
	return true
}

// Structure reflecting a state of fetching host reservations from Kea
// via the reservation-get-page command. It allows for fetching hosts
// in chunks to avoid large bulk of data to be generated on the Kea side
// and transmitted over the network to Stork. The paging mechanism allows
// for controlling how many hosts are returned in a single transaction.
// The client side (Stork in this case) has to remember two values
// returned in the last response to the command, i.e. "from" and
// "source-index". These values mark the last retrieved host and should
// be specified in subsequent commands to inform the Kea server where
// the next page of data starts. These two values along with a bulk of
// other values constitute a state of hosts fetching. A collection of
// these values are maintained by the "iterator".
// Kea versions older than 1.9.0 require subnet-id parameter in the
// reservation-get-page command allowing to fetch host reservations
// for the specified subnet. That's why the iterator also maintains
// the current subnet for which the hosts are fetched.
type hostIterator struct {
	db          dbops.DBI
	app         *dbmodel.App
	daemon      *dbmodel.Daemon
	agents      agentcomm.ConnectedAgents
	limit       int64
	from        int64
	sourceIndex int64
	subnets     []dbmodel.Subnet
	subnetIndex int
	trace       *hostIteratorTrace
}

// Creates new iterator instance.
func newHostIterator(dbi dbops.DBI, app *dbmodel.App, daemon *dbmodel.Daemon, agents agentcomm.ConnectedAgents, limit int64) *hostIterator {
	it := &hostIterator{
		db:          dbi,
		app:         app,
		daemon:      daemon,
		agents:      agents,
		limit:       limit,
		from:        0,
		sourceIndex: 1,
		subnets:     []dbmodel.Subnet{},
		subnetIndex: -1,
		trace:       newHostIteratorTrace(),
	}
	return it
}

// Sends the reservation-get-page command to Kea daemon. If there is an error
// it is returned. Otherwise, the "from" and "source-index" are updated in the
// iterator's state. Finally the list of hosts is retrieved and returned.
func (iterator *hostIterator) sendReservationGetPage() ([]keaconfig.Reservation, int, error) {
	// Depending on the family we should set the service parameter to
	// dhcp4 or dhcp6.
	daemons := []string{iterator.daemon.Name}
	// We need to set subnet-id. It requires extracting the local subnet-id
	// for the given app.
	subnetID := int64(0)
	subnet := iterator.getCurrentSubnet()
	// The returned subnet will be nil if we're fetching global host reservations.
	if subnet != nil {
		for _, ls := range subnet.LocalSubnets {
			if ls.Daemon != nil && ls.Daemon.ID == iterator.daemon.ID {
				subnetID = ls.LocalSubnetID
				break
			}
		}
	}
	// Prepare the command.
	command := keactrl.NewCommandReservationGetPage(subnetID, iterator.sourceIndex, iterator.from, iterator.limit, daemons...)
	commands := []keactrl.SerializableCommand{command}
	response := make([]ReservationGetPageResponse, 1)
	ctx := context.Background()
	respResult, err := iterator.agents.ForwardToKeaOverHTTP(ctx, iterator.app, commands, &response)
	if err != nil {
		return []keaconfig.Reservation{}, keactrl.ResponseError, err
	}

	if err = respResult.GetFirstError(); err != nil {
		return []keaconfig.Reservation{}, keactrl.ResponseError, err
	}

	if len(response) == 0 {
		return []keaconfig.Reservation{}, keactrl.ResponseError, errors.Errorf("invalid response to reservation-get-page command received")
	}

	// An error is likely to be a communication problem between Kea Control
	// Agent and some other daemon.
	if err := response[0].GetError(); err != nil {
		// If the command is not supported by this Kea server, simply stop.
		if errors.As(err, &keactrl.UnsupportedOperationKeaError{}) {
			return []keaconfig.Reservation{}, keactrl.ResponseCommandUnsupported, nil
		}

		return []keaconfig.Reservation{}, response[0].Result,
			errors.WithMessage(err,
				"error returned by Kea in response to reservation-get-page command",
			)
	}

	if response[0].Arguments == nil {
		return []keaconfig.Reservation{}, response[0].Result, errors.Errorf("response to reservation-get-page command lacks arguments")
	}

	// Response received, update the iterator's state.
	iterator.from = response[0].Arguments.Next.From
	iterator.sourceIndex = response[0].Arguments.Next.SourceIndex

	// Return hosts to the caller.
	return response[0].Arguments.Hosts, response[0].Result, nil
}

// Returns a pointer to the subnet at the specified position index in the iterator.
func (iterator *hostIterator) getSubnet(index int) *dbmodel.Subnet {
	if index < 0 || index >= len(iterator.subnets) {
		return nil
	}
	return &iterator.subnets[index]
}

// Returns a pointer to the subnet for which the last chunk of hosts have been
// returned by the getPageFromHostCmds function. This allows for correlating
// the returned hosts with the subnet.
func (iterator *hostIterator) getCurrentSubnet() *dbmodel.Subnet {
	return iterator.getSubnet(iterator.subnetIndex)
}

// Returns the next chunk of host reservations. The first returned value is a slice
// containing the next chunk of hosts. The second value, done, indicates if the
// returned chunk of hosts was the last available one for the given daemon. If this
// value is equal to false the caller should continue calling this function to
// fetch subsequent hosts. If this value is set to true the caller should stop
// calling this function.
func (iterator *hostIterator) getPageFromHostCmds() (hosts []keaconfig.Reservation, done bool, err error) {
	// The default behavior is that an error terminates hosts fetching from
	// the particular app.
	defer func() {
		if done || err != nil {
			done = true
		}
	}()

	// If this is the first time we're getting hosts for this server we should
	// first get all corresponding subnets.
	if len(iterator.subnets) == 0 {
		iterator.subnets, err = dbmodel.GetSubnetsByDaemonID(iterator.db, iterator.daemon.ID)
		if err != nil {
			return hosts, done, errors.WithMessagef(err, "problem getting Kea subnets upon an attempt to detect host reservations over the host_cmds hook library")
		}
	}

	// Iterate over the subnets and, for each subnet, fetch the hosts.
	for i := iterator.subnetIndex; i < len(iterator.subnets); i++ {
		// Send reservation-get-page command to fetch the next chunk of host
		// reservations from Kea.
		var result int
		hosts, result, err = iterator.sendReservationGetPage()
		if err != nil {
			err = errors.WithMessagef(err, "problem sending reservation-get-page command upon attempt to detect host reservations over the host_cmds hook library")
			return hosts, done, err
		}

		// If the command is not supported for this daemon there is nothing more to do.
		if result == keactrl.ResponseCommandUnsupported {
			return hosts, true, nil
		}

		// If the number of hosts returned is 0, it means that we have hit the
		// end of the hosts list for this subnet. Let's move to the next one.
		if len(hosts) == 0 {
			iterator.from = 0
			iterator.sourceIndex = 1
			iterator.subnetIndex++
			continue
		}

		// Hash the returned hosts and remember the hash in the iterator.
		hash := keaconfig.NewHasher().Hash(hosts)
		iterator.trace.addResponse(hash, iterator.subnetIndex, hosts)

		// We return one chunk of hosts for one subnet. So let's get out
		// of this loop.
		break
	}

	if len(hosts) == 0 {
		// If we got here and there are no hosts it means that we have reached the
		// end of all hosts lists for all servers and all subnets.
		done = true
	}
	return hosts, done, err
}

// Iterates over the reservations pulled from the Kea server and converts them
// to the host format in Stork. It also associates the hosts with their
// subnets. The converted hosts are merged into the existing hosts and
// stored in the database. Note that the function modifies the subnet
// specified in this function (modifies its host reservations).
func convertAndUpdateHosts(tx *pg.Tx, daemon *dbmodel.Daemon, subnet *dbmodel.Subnet, reservations []keaconfig.Reservation, lookup keaconfig.DHCPOptionDefinitionLookup) (err error) {
	var hosts []dbmodel.Host
	for _, reservation := range reservations {
		host, err := dbmodel.NewHostFromKeaConfigReservation(reservation, daemon, dbmodel.HostDataSourceAPI, lookup)
		if err != nil {
			log.WithError(err).Warnf("Failed to parse the host reservation")
			continue
		}
		if subnet != nil {
			host.SubnetID = subnet.ID
		}
		hosts = append(hosts, *host)
	}

	var overriddenHosts []dbmodel.Host
	// The subnet is nil when we're dealing with the global hosts.
	if subnet == nil {
		if overriddenHosts, err = overrideIntoDatabaseHosts(tx, int64(0), hosts); err != nil {
			return
		}
		if err = dbmodel.CommitGlobalHostsIntoDB(tx, overriddenHosts); err != nil {
			return
		}
		// We're done with global hosts, so let's get the next chunk of
		// hosts. They can be both global or subnet specific.
		return nil
	}
	// The returned hosts belong to the subnet, but the subnet instance
	// doesn't contain them yet (they are new hosts), so let's assign
	// them explicitly to the current subnet.
	subnet.Hosts = hosts
	// Now, there is a tricky part. The second part argument is the
	// existing subnet. It is merely used to extract the ID of the
	// given subnet and then fetch this subnet along with all the
	// hosts it has in the database. The second parameter specifies
	// the subnet with the new hosts (fetched via the Kea API). These
	// hosts are overridden into the existing hosts for this subnet and
	// returned as overriddenHosts.
	if overriddenHosts, err = overrideIntoDatabaseSubnetHosts(tx, subnet, subnet); err != nil {
		return
	}
	// Now we have to assign the combined set of existing hosts and
	// new hosts into the subnet instance and commit everything to the
	// database.
	subnet.Hosts = overriddenHosts
	if err = dbmodel.CommitSubnetHostsIntoDB(tx, subnet); err != nil {
		return
	}
	return nil
}

// Overrides global or subnet specific hosts into existing entries from database
// with preserving the database ID and some fixed values (e.g. creation timestamp),
// and returns the slice with combined hosts.
// When subnetID of 0 is specified it indicates that the global hosts
// are being merged. If the given host already exists in the database the
// new host is joined to it, i.e., its local host instances are appended.
// If the host does not exist yet, the new host is appended to the returned
// slice.
func overrideIntoDatabaseHosts(dbi dbops.DBI, subnetID int64, newHosts []dbmodel.Host) (hosts []dbmodel.Host, err error) {
	// If there are no new hosts there is nothing to do.
	if len(newHosts) == 0 {
		return
	}
	// Get the hosts from the Stork database for this subnet.
	existingHosts, err := dbmodel.GetHostsBySubnetID(dbi, subnetID)
	if err != nil {
		err = errors.WithMessagef(err, "problem overriding hosts for subnet %d", subnetID)
		return
	}
	// Override each new host into the existing hosts.
	for i := range newHosts {
		newHost := &newHosts[i]
		// Iterate over the existing hosts to check if the new hosts are there already.
		for ie := range existingHosts {
			host := &existingHosts[ie]
			// Joining the hosts will only pass when both instances point to the
			// same host. In that case, the resulting host is used instead of the
			// newHost instance.
			if ok := host.Join(newHost); ok {
				newHost = host
				break
			}
		}
		hosts = append(hosts, *newHost)
	}

	return hosts, err
}

// Overrides hosts belonging to the new subnet into the hosts within existing subnet.
// A host from the new subnet is added to the slice of returned hosts if such
// host doesn't exist. If the host exists, the new host is joined to it by appending
// the LocalHost instances.
func overrideIntoDatabaseSubnetHosts(dbi dbops.DBI, existingSubnet, newSubnet *dbmodel.Subnet) (hosts []dbmodel.Host, err error) {
	return overrideIntoDatabaseHosts(dbi, existingSubnet.ID, newSubnet.Hosts)
}

// For a given Kea daemon it detects host reservations configured in the
// configuration file.
func detectGlobalHostsFromConfig(dbi dbops.DBI, daemon *dbmodel.Daemon, lookup keaconfig.DHCPOptionDefinitionLookup) (hosts []dbmodel.Host, err error) {
	if daemon.KeaDaemon == nil || daemon.KeaDaemon.Config == nil {
		return hosts, err
	}
	// Get the top level (global) reservations.
	reservationsList := daemon.KeaDaemon.Config.GetReservations()
	if len(reservationsList) == 0 {
		return hosts, err
	}
	for _, reservation := range reservationsList {
		host, err := dbmodel.NewHostFromKeaConfigReservation(reservation, daemon, dbmodel.HostDataSourceConfig, lookup)
		if err != nil {
			log.Warnf("Skipping invalid host reservation: %v", reservation)
			continue
		}
		hosts = append(hosts, *host)
	}
	// Overrides new hosts into the existing global hosts.
	return overrideIntoDatabaseHosts(dbi, int64(0), hosts)
}
