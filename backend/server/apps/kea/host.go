package kea

import (
	"context"
	"fmt"

	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

const (
	defaultHostCmdsPageLimit int64 = 1000
)

// Structure reflecting "next" map of the Kea response to the
// reservation-get-page command.
type ReservationGetPageNext struct {
	From        int64
	SourceIndex int64 `json:"source-index"`
}

// Structure reflecting arguments of the Kea response to the
// reservation-get-page command.
type ReservationGetPageArgs struct {
	Count int64
	Hosts []dbmodel.KeaConfigReservation
	Next  ReservationGetPageNext
}

// Structure reflecting a Kea response to the reservation-get-page
// command.
type ReservationGetPageResponse struct {
	agentcomm.KeaResponseHeader
	Arguments *ReservationGetPageArgs `json:"arguments,omitempty"`
}

// Instance of the puller which periodically fetches host reservations from
// the Kea apps.
type HostsPuller struct {
	*agentcomm.PeriodicPuller
}

// Create an instance of the puller which periodically fetches host reservations
// from monitored Kea apps via control channel.
func NewHostsPuller(db *dbops.PgDB, agents agentcomm.ConnectedAgents) (*HostsPuller, error) {
	hostsPuller := &HostsPuller{}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Kea Hosts", "kea_hosts_puller_interval",
		hostsPuller.pullData)
	if err != nil {
		return nil, err
	}
	hostsPuller.PeriodicPuller = periodicPuller
	return hostsPuller, nil
}

// Stops the timer triggering hosts fetching from apps.
func (puller *HostsPuller) Shutdown() {
	puller.PeriodicPuller.Shutdown()
}

// Triggers fetch of the host reservations from the monitored Kea apps.
func (puller *HostsPuller) pullData() (int, error) {
	// Get the list of all Kea apps from the database.
	apps, err := dbmodel.GetAppsByType(puller.Db, dbmodel.AppTypeKea)
	if err != nil {
		return 0, err
	}

	// Get sequence number to be associated with updated and inserted hosts.
	seq, err := dbmodel.GetNextBulkUpdateSeq(puller.Db)
	if err != nil {
		err = errors.WithMessagef(err, "problem with getting next bulk update sequence number fetching hosts from Kea apps")
		return 0, err
	}

	// Synchronize hosts from all Kea apps.
	var lastErr error
	appsOkCnt := 0
	for i := range apps {
		err := updateHostsFromHostCmds(puller.Db, puller.Agents, &apps[i], seq)
		if err != nil {
			lastErr = err
			log.Errorf("error occurred while fetching hosts from app %d: %+v", apps[i].ID, err)
		} else {
			appsOkCnt++
		}
	}

	// Remove all associations between the hosts and tha apps that are no longer
	// present.
	err = dbmodel.DeleteLocalHostsWithOtherSeq(puller.Db, seq, "api")
	if err != nil {
		log.Errorf("error occurred while deleting old hosts after update from Kea apps: %+v", err)
	}

	log.Printf("completed pulling hosts from Kea apps: %d/%d succeeded", appsOkCnt, len(apps))
	return appsOkCnt, lastErr
}

// Structure reflecting a state of fetching host reservations from Kea
// via the reservation-get-page command. This allows for fetching hosts
// in chunks to avoid large bulk of data to be generated on the Kea side
// and transmitted over the network to Stork. The paging mechanism allows
// for controlling how many hosts are returned in a single transaction.
// The client side (Stork in this case) has to has to remember two values
// returned in the last response to the command, i.e. "from" and
// "source-index". These values mark the last retrieved host and should
// be specified in subsequent commands to inform the Kea server where
// the next page of data starts. These two values along with a bulk of
// other values constitute a state of hosts fetching. A collection of
// these values are maintained by the "iterator".
// The current limitation of the Kea server is that the reservation-get-page
// command is required to contain subnet-id parameter. In other words,
// this command allows only for fetching the reservations for the given
// subnet in the given transaction. That's why the iterator also maintains
// the current subnet for which the hosts are being fetched. It also
// holds the family (DHCPv4 or DHCPv6) to indicate from which Kea
// daemons the reservations are currently fetched. It is not easy to fetch
// from both servers at the same time, because they contain different
// subnets, different number of reservations. That's why the iterator
// fetches hosts from these two servers sequentially, i.e. gets all
// hosts from one server and then gets all hosts from the other.
type HostDetectionIterator struct {
	db          *dbops.PgDB
	app         *dbmodel.App
	agents      agentcomm.ConnectedAgents
	limit       int64
	serverIndex int
	family      int
	from        int64
	sourceIndex int64
	subnets     []dbmodel.Subnet
	subnetIndex int
}

// Creates new iterator instance.
func NewHostDetectionIterator(db *dbops.PgDB, app *dbmodel.App, agents agentcomm.ConnectedAgents, limit int64) *HostDetectionIterator {
	it := &HostDetectionIterator{
		db:          db,
		app:         app,
		agents:      agents,
		limit:       limit,
		serverIndex: 0,
		family:      0,
		from:        0,
		sourceIndex: 1,
		subnets:     []dbmodel.Subnet{},
		subnetIndex: -1,
	}
	return it
}

// Resets iterator's state to make it possible to start over fetching the
// hosts if necessary.
func (iterator *HostDetectionIterator) reset() {
	iterator.serverIndex = 0
	iterator.family = 0
	iterator.from = 0
	iterator.sourceIndex = 1
	iterator.subnets = make([]dbmodel.Subnet, 0)
	iterator.subnetIndex = -1
}

// Iterates over the hosts fetched from the Kea server and converts them to
// the Stork's format for hosts. It also associates the hosts with their
// subnet using the data stored as iterator's state.
func (iterator *HostDetectionIterator) convertAndAssignHosts(fetchedHosts []dbmodel.KeaConfigReservation) (hosts []dbmodel.Host) {
	for _, fetchedHost := range fetchedHosts {
		host, err := dbmodel.NewHostFromKeaConfigReservation(fetchedHost)
		if err != nil {
			continue
		}
		if iterator.subnetIndex >= 0 {
			host.SubnetID = iterator.subnets[iterator.subnetIndex].ID
		}
		hosts = append(hosts, *host)
	}
	return hosts
}

// Sends the reservation-get-page command to Kea. If there is an error it is
// returned. Otherwise, the "from" and "source-index" are updated in the
// iterator's state. Finally the list of hosts is retrieved and returned.
func (iterator *HostDetectionIterator) sendReservationGetPage() (hosts []dbmodel.KeaConfigReservation, result int, canRetry bool, err error) {
	// Depending on the family we should set the service parameter to
	// dhcp4 or dhcp6.
	daemons, err := agentcomm.NewKeaDaemons(fmt.Sprintf("dhcp%d", iterator.family))
	if err != nil {
		return hosts, agentcomm.KeaResponseError, false, err
	}
	// We need to set subnet-id. This required extracting the local subnet-id
	// for the given app.
	subnetID := int64(0)
	subnet := iterator.GetCurrentSubnet()
	// The returned subnet will be nil if we're fetching global host reservations.
	if subnet != nil {
		for _, ls := range subnet.LocalSubnets {
			if ls.AppID == iterator.app.ID {
				subnetID = ls.LocalSubnetID
				break
			}
		}
	}
	// The subnet-id and limit are mandatory.
	arguments := map[string]interface{}{
		"subnet-id": subnetID,
		"limit":     iterator.limit,
	}
	// The from and source-index values are not present in a first call to
	// fetch the hosts for a given subnet.
	if iterator.from > 0 {
		arguments["from"] = iterator.from
	}
	if iterator.sourceIndex > 0 {
		arguments["source-index"] = iterator.sourceIndex
	}
	// Prepare the command.
	command, err := agentcomm.NewKeaCommand("reservation-get-page", daemons, &arguments)
	if err != nil {
		return hosts, agentcomm.KeaResponseError, false, err
	}

	commands := []*agentcomm.KeaCommand{command}
	response := make([]ReservationGetPageResponse, 1)
	ctx := context.Background()
	respResult, err := iterator.agents.ForwardToKeaOverHTTP(ctx, iterator.app, commands, &response)
	if err != nil {
		// Can retry because the error may go away upon the next attempt.
		return hosts, agentcomm.KeaResponseError, true, err
	}

	if respResult.Error != nil {
		// Can retry beause the error may go away.
		return hosts, agentcomm.KeaResponseError, true, respResult.Error
	}

	if len(respResult.CmdsErrors) > 0 && respResult.CmdsErrors[0] != nil {
		// Can retry beause the error may go away.
		return hosts, agentcomm.KeaResponseError, true, respResult.CmdsErrors[0]
	}

	// The following two would rather be fatal errors and retrying wouldn't help.
	if len(response) == 0 {
		return hosts, agentcomm.KeaResponseError, false, errors.Errorf("invalid response to reservation-get-page command received")
	}

	// An error is likely to be a communication problem between Kea Control
	// Agent and some other daemon.
	if response[0].Result == agentcomm.KeaResponseError {
		return hosts, response[0].Result, false, errors.Errorf("error returned by Kea in response to reservation-get-page command")
	}

	// If the command is not supported by this Kea server, simply stop.
	if response[0].Result == agentcomm.KeaResponseCommandUnsupported {
		return hosts, response[0].Result, false, nil
	}

	if response[0].Arguments == nil {
		return hosts, response[0].Result, false, errors.Errorf("response to reservation-get-page command lacks arguments")
	}

	// Response received, update the iterator's state.
	iterator.from = response[0].Arguments.Next.From
	iterator.sourceIndex = response[0].Arguments.Next.SourceIndex

	// Return hosts to the caller.
	hosts = response[0].Arguments.Hosts
	return hosts, response[0].Result, false, nil
}

// Returns a pointer to the subnet for which the last chunk of hosts have been
// returned by the DetectHostsFromHostCmds function. This allows for correlating
// the returned hosts with the subnet.
func (iterator *HostDetectionIterator) GetCurrentSubnet() *dbmodel.Subnet {
	if iterator.subnetIndex < 0 || iterator.subnetIndex >= len(iterator.subnets) {
		return nil
	}
	return &iterator.subnets[iterator.subnetIndex]
}

// Returns the next chunk of host reservations. The first returned value is a slice
// containing the next chunk of hosts. The second value, done, indicates if the
// returned chunk of hosts was the last available one for the given app. If this
// value is equal to false the caller should continue calling this function to
// fetch subsequent hosts. If this value is set to true the caller should stop
// calling this function. Further calling this function would return the first
// chunk of hosts again.
func (iterator *HostDetectionIterator) DetectHostsPageFromHostCmds() (hosts []dbmodel.Host, done bool, err error) {
	retry := false

	// The default behavior is that an error terminates hosts fetching from
	// the particular app. It is possible to override this in some cases
	// by setting the retry value.
	defer func() {
		if done || (!retry && err != nil) {
			iterator.reset()
			done = true
		}
	}()

	// If this is not Kea application there is nothing to do.
	appKea := iterator.app
	if appKea.Type != dbmodel.AppTypeKea {
		err = errors.Errorf("attempted to fetch host reservations for non Kea app")
		return hosts, done, err
	}

	// Count the servers we have iterated over to make sure we use the one we used
	// previously.
	serverIndex := 0
	for _, d := range appKea.Daemons {
		if d.KeaDaemon == nil || d.KeaDaemon.Config == nil || !d.Active {
			continue
		}

		var family int
		switch d.Name {
		case dhcp4:
			family = 4
		case dhcp6:
			family = 6
		default:
			continue
		}

		// We have been already getting hosts from this daemon, so let's get to
		// the next one.
		if serverIndex < iterator.serverIndex {
			serverIndex++
			continue
		}

		// Remember the current server's family because it will be required to
		// set a service value for the commnd being sent.
		iterator.family = family

		// If this is the first time we're getting hosts for this server we should
		// first get all corresponding subnets.
		if len(iterator.subnets) == 0 {
			iterator.subnets, err = dbmodel.GetSubnetsByAppID(iterator.db, iterator.app.ID, family)
			if err != nil {
				return hosts, done, errors.WithMessagef(err, "problem with getting Kea subnets upon an attempt to detect host reservations over the host_cmds hooks library")
			}
		}

		// Iterate over the subnets and for each subnet fetch the hosts.
		for i := iterator.subnetIndex; i < len(iterator.subnets); i++ {
			// Send reservation-get-page command to fetch the next chunk of host
			// reservations from Kea.
			var returnedHosts []dbmodel.KeaConfigReservation
			var result int
			returnedHosts, result, retry, err = iterator.sendReservationGetPage()
			if err != nil {
				err = errors.WithMessagef(err, "problem with sending reservation-get-page command upon attempt to detect host reservations over the host_cmds hooks library")
				return hosts, done, err
			}

			// If the command is not supported for this app there is nothing more to do.
			if result == agentcomm.KeaResponseCommandUnsupported {
				return hosts, true, nil
			}

			// If the number of hosts returned is 0, it means that we have hit the
			// end of the hosts list for this subnet. Let's move to the next one.
			if len(returnedHosts) == 0 {
				iterator.from = 0
				iterator.sourceIndex = 1
				iterator.subnetIndex++
				continue
			}

			// There are some hosts for this subnet so let's convert them from
			// Kea to Stork format.
			hosts = iterator.convertAndAssignHosts(returnedHosts)

			// We return one chunk of hosts for one subnet. So let's get out
			// of this loop.
			break
		}

		// If there are some hosts fetched, let's return them.
		if len(hosts) > 0 {
			break
		}

		if len(iterator.subnets) <= iterator.subnetIndex {
			// If we went over all hosts in all subnets but there is potentially
			// one more server available, let's try this server.
			iterator.reset()
			serverIndex++
			iterator.serverIndex = serverIndex
			continue
		}
	}

	// If we got here and there are no hosts it means that we have reached the
	// end of all hosts lists for all servers and all subnets.
	if len(hosts) == 0 {
		done = true
	}
	return hosts, done, err
}

// Merges global or subnet specific hosts and returns the slice with merged
// hosts. When subnetID of 0 is specified it indicates that the global hosts
// are being merged. If the given host already exists in the database and a
// new host is equal to it, the new host is not added. If none of the existing
// hosts equals the new host, the new host is appended to the returned slice.
// The combineHosts boolean flag indicates if the returned hosts are the
// combination of the existing hosts and new hosts (if true) or only new
// hosts are returned (if false). This function is called by mergeGlobalHosts
// and mergeSubnetHosts.
func mergeHosts(db *dbops.PgDB, subnetID int64, newHosts []dbmodel.Host, combineHosts bool, app *dbmodel.App) (hosts []dbmodel.Host, err error) {
	if len(newHosts) == 0 {
		return hosts, err
	}

	existingHosts, err := dbmodel.GetHostsBySubnetID(db, subnetID)
	if err != nil {
		return hosts, errors.WithMessagef(err, "problem with merging hosts for subnet %d", subnetID)
	}

	// Merge each new host.
	for i := range newHosts {
		newHost := &newHosts[i]
		found := false
		// Iterate over the existing hosts to check if the new hosts are there already.
		for ie := range existingHosts {
			host := &existingHosts[ie]
			if newHost.Equal(host) {
				// This host already exist. It will be updated.
				found = true
				newHost = host

				// Indicate that the host should be updated and that the new app should
				// be associated with it upon the call to dbmodel.CommitSubnetHostsIntoDB
				// or dbmodel.CommitGlobalHostsIntoDB.
				newHost.UpdateOnCommit = true

				// Check if there is already an association between this app and the
				// given host. If there is, we need to reset the sequence number for
				// it to force of the sequence number. Otherwise, the old sequence
				// number will remain.
				for j, lh := range host.LocalHosts {
					if lh.AppID == app.ID {
						newHost.LocalHosts[j].UpdateSeq = 0
					}
				}
				break
			}
		}
		if !found || !combineHosts {
			// Host doesn't exist yet, so let's add it.
			newHost.UpdateOnCommit = true
			hosts = append(hosts, *newHost)
		}
	}
	// Also include existing hosts if we're asked to do so.
	if combineHosts {
		hosts = append(existingHosts, hosts...)
	}

	return hosts, err
}

// Merges hosts belonging to the new subnet into the hosts within existing subnet.
// A host from the new subnet is added to the slice of returned hosts if such
// host doesn't exist. If the host with exactly the same set of of identifiers
// and IP reservation exists, it is not added to the slice of returned hosts to
// avoid duplication. As a result, the returned slice of hosts is a collection of
// existing hosts plus the hosts from the new subnet which do not exist in the
// database.
func mergeSubnetHosts(db *dbops.PgDB, existingSubnet, newSubnet *dbmodel.Subnet, app *dbmodel.App) (hosts []dbmodel.Host, err error) {
	return mergeHosts(db, existingSubnet.ID, newSubnet.Hosts, true, app)
}

// For a given Kea application it detects host reservations configured in the
// configuration file.
func detectGlobalHostsFromConfig(db *dbops.PgDB, app *dbmodel.App) (hosts []dbmodel.Host, err error) {
	// If this is not Kea application there is nothing to do.
	if app.Type != dbmodel.AppTypeKea {
		return hosts, nil
	}

	// Detect reservation for each Kea daemon.
	for _, d := range app.Daemons {
		if d.KeaDaemon == nil || d.KeaDaemon.Config == nil {
			continue
		}
		// Get the top level (global) reservations.
		if reservationsList, ok := d.KeaDaemon.Config.GetTopLevelList("reservations"); ok {
			if len(reservationsList) == 0 {
				continue
			}
			// Iterate over the reservations found.
			for _, r := range reservationsList {
				if reservationMap, ok := r.(map[string]interface{}); ok {
					// Parse the reservation.
					host, err := dbmodel.NewHostFromKea(&reservationMap)
					if err != nil {
						log.Warnf("skipping invalid host reservation: %v", reservationMap)
						continue
					}
					hosts = append(hosts, *host)
				}
			}
		}
	}
	// Merge new hosts into the existing global hosts.
	return mergeHosts(db, int64(0), hosts, false, app)
}

// Fetches all host reservations stored in the hosts backend for the particular
// Kea app. The app must have the host_cmds hooks library loaded. The function
// uses HostDetectionIterator mechanism to fetch the hosts, which will in
// most cases result in multiple reservation-get-page commands sent to Kea
// instance.
func updateHostsFromHostCmds(db *dbops.PgDB, agents agentcomm.ConnectedAgents, app *dbmodel.App, seq int64) error {
	tx, rollback, commit, err := dbops.Transaction(db)
	if err != nil {
		err = errors.WithMessagef(err, "problem with starting transaction for committing new hosts from host_cmds hooks library for app id %d", app.ID)
		return err
	}
	defer rollback()
	it := NewHostDetectionIterator(db, app, agents, defaultHostCmdsPageLimit)
	var (
		hosts []dbmodel.Host
		done  bool
	)
	// Fetch the hosts as long as they are returned by Kea.
	for !done {
		hosts, done, err = it.DetectHostsPageFromHostCmds()
		if err != nil {
			break
		}
		// This condition is unlikely but let's make sure.
		if len(hosts) == 0 {
			continue
		}
		subnet := it.GetCurrentSubnet()
		// The subnet is nil when we're dealing with the global hosts.
		if subnet == nil {
			mergedHosts, err := mergeHosts(db, int64(0), hosts, true, app)
			if err != nil {
				break
			}
			err = dbmodel.CommitGlobalHostsIntoDB(tx, mergedHosts, app, "api", seq)
			if err != nil {
				break
			}
			// We're done with global hosts, so let's get the next chunk of
			// hosts. They can be both global or subnet specific.
			continue
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
		// hosts are merged into the existing hosts for this subnet and
		// returned as mergedHosts.
		mergedHosts, err := mergeSubnetHosts(db, subnet, subnet, app)
		if err != nil {
			break
		}
		// Now we have to assign the combined set of existing hosts and
		// new hosts into the subnet instance and commit everything to the
		// database.
		subnet.Hosts = mergedHosts
		err = dbmodel.CommitSubnetHostsIntoDB(tx, subnet, app, "api", seq)
		if err != nil {
			break
		}
	}

	if err == nil {
		err = commit()
		if err != nil {
			err = errors.WithMessagef(err, "problem with committing transaction adding new hosts from host_cmds hooks library for app id %d", app.ID)
		}
	}
	return err
}
