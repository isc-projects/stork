package entitymigrator

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/config"
	"isc.org/stork/server/configmigrator"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Implements the configmigrator.Migrator interface. Migrates the hosts from
// the Kea JSON configuration to the host database.
type hostMigrator struct {
	db               *pg.DB
	filter           dbmodel.HostsByPageFilters
	items            []dbmodel.Host
	hostErrs         map[int64]configmigrator.MigrationError
	daemonErrs       map[int64]configmigrator.MigrationError
	limit            int64
	totalItemsLoaded int64
	dhcpOptionLookup keaconfig.DHCPOptionDefinitionLookup
	connectedAgents  agentcomm.ConnectedAgents
	daemonLocker     config.DaemonLocker
	// The daemons are unlocked at the end of the migration because we expect
	// the same daemons will appear in many chunks.
	lockedDemonIDs map[int64]config.LockKey
	// The global pullers that fetch the hosts.
	// Expected to be the state puller (fetches hosts from JSON) and the host
	// puller (fetches hosts from DB).
	pullers []Pauser
}

var _ configmigrator.Migrator = &hostMigrator{}

// Creates a new host migrator.
func NewHostMigrator(
	filter dbmodel.HostsByPageFilters,
	db *pg.DB,
	connectedAgents agentcomm.ConnectedAgents,
	dhcpOptionLookup keaconfig.DHCPOptionDefinitionLookup,
	locker config.DaemonLocker,
	statePuller Pauser,
	hostPuller Pauser,
) configmigrator.Migrator {
	// Migrating the conflicted hosts is not supported.
	filter.DHCPDataConflict = storkutil.Ptr(false)
	return &hostMigrator{
		db:               db,
		filter:           filter,
		limit:            100,
		totalItemsLoaded: 0,
		dhcpOptionLookup: dhcpOptionLookup,
		connectedAgents:  connectedAgents,
		daemonLocker:     locker,
		pullers:          []Pauser{statePuller, hostPuller},
		lockedDemonIDs:   make(map[int64]config.LockKey),
	}
}

// Begins the migration. Returns an error if the migration cannot be started.
// Stops the hosts puller.
func (m *hostMigrator) Begin() error {
	for _, puller := range m.pullers {
		puller.Pause()
	}
	return nil
}

// Ends the migration. Restarts the hosts puller.
func (m *hostMigrator) End() error {
	var errs []error
	for daemonID, lockKey := range m.lockedDemonIDs {
		err := m.daemonLocker.Unlock(lockKey, daemonID)
		if err != nil {
			errs = append(errs, err)
		}
		delete(m.lockedDemonIDs, daemonID)
	}

	for _, puller := range m.pullers {
		puller.Unpause()
	}

	return storkutil.CombineErrors(
		"some errors occurred while finishing the migration",
		errs,
	)
}

// Returns a total number of hosts to migrate.
func (m *hostMigrator) CountTotal() (int64, error) {
	_, count, err := dbmodel.GetHostsByPage(m.db, 0, 0, m.filter, "", dbmodel.SortDirAny)
	return count, err
}

// Loads a chunk of hosts from the database. Returns the number of loaded
// hosts.
func (m *hostMigrator) LoadItems() (int64, error) {
	items, _, err := dbmodel.GetHostsByPage(m.db, m.totalItemsLoaded, m.limit, m.filter, "", dbmodel.SortDirAsc)
	if err != nil {
		return 0, err
	}
	m.items = items
	itemsLoaded := int64(len(items))
	m.totalItemsLoaded += itemsLoaded
	return itemsLoaded, nil
}

// Adds the hosts to the database and removes them from the Kea configuration.
//
// Algorithm.
//
//   - Check the transaction lock for a given daemon. Lock it before the
//     migration and unlock it after the migration.
//   - Send the reservation-add command
//   - Send the reservation-local-del command
//   - Send the config-write command
//   - Handle the insufficient permissions to write the configuration.
func (m *hostMigrator) Migrate() []configmigrator.MigrationError {
	// Clean up the error map.
	m.hostErrs = make(map[int64]configmigrator.MigrationError)
	m.daemonErrs = make(map[int64]configmigrator.MigrationError)

	// Collect the unique daemons to send all the commands to the same daemon
	// in a single batch.
	daemonsByIDs := make(map[int64]*dbmodel.Daemon)
	for _, host := range m.items {
		for _, localHost := range host.LocalHosts {
			daemon := localHost.Daemon
			if !daemon.Active {
				// Skip inactive daemons.
				continue
			}

			daemonsByIDs[daemon.ID] = daemon
		}
	}

	daemons := make([]*dbmodel.Daemon, 0, len(daemonsByIDs))
	for _, daemon := range daemonsByIDs {
		daemons = append(daemons, daemon)
	}
	sort.Slice(daemons, func(i, j int) bool {
		return daemons[i].ID < daemons[j].ID
	})

	// Iterate over the daemons in the ascending order of their IDs.
	for _, daemon := range daemons {
		m.migrateDaemonHosts(daemon)
	}

	sliceErrs := make([]configmigrator.MigrationError, 0, len(m.hostErrs)+len(m.daemonErrs))
	for _, err := range m.hostErrs {
		sliceErrs = append(sliceErrs, err)
	}
	for _, err := range m.daemonErrs {
		sliceErrs = append(sliceErrs, err)
	}
	return sliceErrs
}

// Migrates the hosts related to the given daemon.
func (m *hostMigrator) migrateDaemonHosts(daemon *dbmodel.Daemon) {
	daemonID := daemon.ID

	// Lock the daemon for modification. Do it only if the daemon has not
	// been locked yet.
	if _, ok := m.lockedDemonIDs[daemonID]; !ok {
		lockKey, err := m.daemonLocker.Lock(daemonID)
		if err != nil {
			// Skip the daemon if it cannot be locked.
			m.setDaemonError(daemon, err)
			return
		}
		m.lockedDemonIDs[daemonID] = lockKey
	}

	// Insert the reservations to the host database.
	m.prepareAndSendHostCommands(daemon, func(host *dbmodel.Host, localHosts map[dbmodel.HostDataSource]*dbmodel.LocalHost) (keactrl.SerializableCommand, error) {
		if _, ok := localHosts[dbmodel.HostDataSourceConfig]; !ok {
			// Nothing to add. The host is not stored in the JSON
			// configuration.
			return nil, nil
		}

		// Skip if the host is already in the database.
		// Disclaimer: We expect all conflicts to be resolved before the
		// migration.
		if _, ok := localHosts[dbmodel.HostDataSourceAPI]; ok {
			// Nothing to migrate. The host is already in the database.
			return nil, nil
		}

		// Add the reservation to the Kea host database.
		reservationAdd, err := keaconfig.CreateHostCmdsReservation(
			daemonID,
			m.dhcpOptionLookup,
			*host,
		)
		if err != nil {
			err = errors.WithMessagef(err,
				"failed to create a reservation for host '%d' of daemon '%d'",
				host.ID, daemonID,
			)
			return nil, err
		}

		commandAdd := keactrl.NewCommandReservationAdd(reservationAdd, daemon.Name)

		return commandAdd, nil
	})

	// Delete the reservations from the Kea configuration. It is done only
	// for the hosts that have been properly added to the host database or
	// already existed in the host database.
	m.prepareAndSendHostCommands(daemon, func(host *dbmodel.Host, localHosts map[dbmodel.HostDataSource]*dbmodel.LocalHost) (keactrl.SerializableCommand, error) {
		if _, ok := localHosts[dbmodel.HostDataSourceConfig]; !ok {
			// Nothing to remove. The host is not stored in the JSON
			// configuration.
			return nil, nil
		}

		// Remove the reservation from the Kea configuration.
		reservationDel, err := keaconfig.CreateHostCmdsDeletedReservation(
			daemonID, host, keaconfig.HostCmdsOperationTargetMemory,
		)
		if err != nil {
			err = errors.WithMessagef(err,
				"failed to create a deleted reservation for host '%d' of daemon '%d'",
				host.ID, daemonID,
			)
			return nil, err
		}

		commandDel := keactrl.NewCommandReservationDel(reservationDel, daemon.Name)
		return commandDel, nil
	})

	// Make the configuration persistent.
	m.saveConfigChanges(daemon)
}

// It is general-purpose function to create a command for each host and send
// all created commands to the same daemon in a single batch.
// Accepts the target daemon and a function that produces a command. The
// command accepts a host to which the command is related and a map of local
// hosts that are related to the host indexed by the data source. The function
// returns a command to send, or nil if there is nothing to send, and an error.
//
// It skips the hosts that have already been marked as errored or don't belong
// to the daemon. If the provided function returns an error or the command
// execution fails, the host is marked as errored. If the daemon communication
// fails, all hosts related to the daemon are marked as errored.
func (m *hostMigrator) prepareAndSendHostCommands(daemon *dbmodel.Daemon, f func(*dbmodel.Host, map[dbmodel.HostDataSource]*dbmodel.LocalHost) (keactrl.SerializableCommand, error)) {
	if m.isDaemonErrored(daemon.ID) {
		// Skip errored daemons.
		return
	}

	daemonID := daemon.ID

	var commands []keactrl.SerializableCommand
	var commandHostIDs []int64
	var commandHostLabels []string

	for _, host := range m.items {
		if m.isHostErrored(host.ID) {
			// Skip errored hosts.
			continue
		}

		// There is a unique index on the local host table that forces
		// there is only one local host for the same daemon ID, host ID,
		// and data source.
		localHosts := map[dbmodel.HostDataSource]*dbmodel.LocalHost{}
		for _, localHost := range host.LocalHosts {
			if localHost.DaemonID != daemonID {
				continue
			}

			localHosts[localHost.DataSource] = &localHost
		}

		if len(localHosts) == 0 {
			// Skip hosts that don't belong to the daemon.
			continue
		}

		hostLabel := getHostLabel(host)

		command, err := f(&host, localHosts)
		if err != nil {
			err = errors.WithMessagef(err, "failed to create a command for host '%d' of daemon '%d'", host.ID, daemonID)
			m.hostErrs[host.ID] = configmigrator.MigrationError{
				ID:          host.ID,
				Label:       hostLabel,
				CauseEntity: configmigrator.ErrorCauseEntityHost,
				Error:       err,
			}
			continue
		}
		if command == nil {
			// Nothing to send.
			continue
		}

		commands = append(commands, command)
		commandHostIDs = append(commandHostIDs, host.ID)
		commandHostLabels = append(commandHostLabels, hostLabel)
	}

	if len(commands) == 0 {
		// Nothing to send.
		return
	}

	// Send the command.
	responses := make([]*keactrl.ResponseList, 0, len(commands))
	for range commands {
		responses = append(responses, &keactrl.ResponseList{})
	}

	responsesAny := make([]any, 0, len(responses))
	for i := range responses {
		responsesAny = append(responsesAny, responses[i])
	}

	result, err := m.connectedAgents.ForwardToKeaOverHTTP(
		context.Background(), daemon.App, commands, responsesAny...,
	)
	if err == nil {
		err = result.Error
	}
	if err != nil {
		err = errors.WithMessagef(err, "failed to send a command to daemon '%d'", daemonID)
		// Communication error between the server and the agent.
		// Mark all related hosts as errored.
		m.setDaemonError(daemon, err)
		return
	}

	// Communication error between the Kea CA and the Kea DHCP daemon.
	for i, err := range result.CmdsErrors {
		hostID := commandHostIDs[i]
		if m.isHostErrored(hostID) {
			continue
		}

		if err == nil {
			continue
		}
		err = errors.WithMessagef(err,
			"command %d/%d execution failed for host '%d' of daemon '%d'",
			i+1, len(result.CmdsErrors),
			hostID, daemonID)
		m.hostErrs[hostID] = configmigrator.MigrationError{
			ID:          hostID,
			Label:       commandHostLabels[i],
			CauseEntity: configmigrator.ErrorCauseEntityHost,
			Error:       err,
		}
	}

	// Execution error of the command.
	for i, responsePerDaemon := range responses {
		hostID := commandHostIDs[i]
		if m.isHostErrored(hostID) {
			continue
		}

		if len(*responsePerDaemon) == 0 {
			// Should not happen.
			continue
		}
		response := (*responsePerDaemon)[0]

		if err := response.GetError(); err != nil {
			err = errors.WithMessagef(err,
				"command %d/%d returned an error for host '%d' of daemon '%d'",
				i+1, len(responses),
				hostID, daemonID)
			m.hostErrs[hostID] = configmigrator.MigrationError{
				ID:          hostID,
				Label:       commandHostLabels[i],
				Error:       err,
				CauseEntity: configmigrator.ErrorCauseEntityHost,
			}
		}
	}
}

// Makes the changes provided in the Kea JSON configuration persistent by
// sending the config-write command. Handles the error if the command fails.
func (m *hostMigrator) saveConfigChanges(daemon *dbmodel.Daemon) {
	// Send the config-write command.
	commandWrite := keactrl.NewCommandBase(keactrl.ConfigWrite, daemon.Name)

	var response keactrl.ResponseList
	result, err := m.connectedAgents.ForwardToKeaOverHTTP(
		context.Background(), daemon.App,
		[]keactrl.SerializableCommand{commandWrite}, &response,
	)
	if err == nil {
		err = result.GetFirstError()
		if err == nil && len(response) > 0 {
			err = response[0].GetError()
		}
	}
	if err != nil {
		err = errors.WithMessagef(err, "failed to send config-write to daemon '%d'", daemon.ID)
		m.setDaemonError(daemon, err)
	}
}

// Marks all hosts related to the daemon as errored.
func (m *hostMigrator) setDaemonError(daemon *dbmodel.Daemon, err error) {
	m.daemonErrs[daemon.ID] = configmigrator.MigrationError{
		ID:          daemon.ID,
		Label:       getDaemonLabel(daemon),
		Error:       err,
		CauseEntity: configmigrator.ErrorCauseEntityDaemon,
	}
}

// Returns true if the host is errored.
func (m *hostMigrator) isHostErrored(hostID int64) bool {
	_, ok := m.hostErrs[hostID]
	return ok
}

// Returns true if the daemon is errored.
func (m *hostMigrator) isDaemonErrored(daemonID int64) bool {
	_, ok := m.daemonErrs[daemonID]
	return ok
}

// Creates a label for the host.
func getHostLabel(host dbmodel.Host) string {
	if len(host.HostIdentifiers) == 0 {
		// Host must have at least one identifier.
		return "unknown"
	}
	identifier := host.HostIdentifiers[0]
	return fmt.Sprintf(
		"%s=%s",
		identifier.Type,
		storkutil.BytesToHexWithSeparator(identifier.Value, ":"),
	)
}

// Creates a label for the daemon.
func getDaemonLabel(daemon *dbmodel.Daemon) string {
	return daemon.Name
}
