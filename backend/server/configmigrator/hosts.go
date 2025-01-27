package configmigrator

import (
	"context"
	"slices"

	"github.com/go-pg/pg/v10"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

type hostMigrator struct {
	db               *pg.DB
	filter           dbmodel.HostsByPageFilters
	items            []dbmodel.Host
	errs             map[int64]error
	limit            int64
	dhcpOptionLookup keaconfig.DHCPOptionDefinitionLookup
	connectedAgents  agentcomm.ConnectedAgents
}

var _ Migrator = &hostMigrator{}

// Creates a new host migrator.
func NewHostMigrator(filter dbmodel.HostsByPageFilters, db *pg.DB, connectedAgents agentcomm.ConnectedAgents, dhcpOptionLookup keaconfig.DHCPOptionDefinitionLookup) Migrator {
	// Migrating the conflicted hosts is not supported.
	filter.DHCPDataConflict = storkutil.Ptr(false)
	return &hostMigrator{
		db:               db,
		filter:           filter,
		limit:            100,
		dhcpOptionLookup: dhcpOptionLookup,
		connectedAgents:  connectedAgents,
	}
}

// Returns a total number of hosts to migrate.
func (m *hostMigrator) CountTotal() (int64, error) {
	_, count, err := dbmodel.GetHostsByPage(m.db, 0, 0, m.filter, "", dbmodel.SortDirAny)
	return count, err
}

// Loads a chunk of hosts from the database.
func (m *hostMigrator) LoadItems(offset int64) (int64, error) {
	items, _, err := dbmodel.GetHostsByPage(m.db, offset, m.limit, m.filter, "id", dbmodel.SortDirAsc)
	if err != nil {
		// Returns the number of items tried to load.
		return m.limit, err
	}
	m.items = items
	return int64(len(m.items)), nil
}

// Adds the hosts to the database and removes them from the Kea configuration.
//
// Algorithm.
//
// TODO: 0. Check if the config is writable.
// 1. Send the reservation-add command
// TODO: 2. Detect unexpected changes in the Kea JSON configuration (before config-set)
// 3. Send the reservation-local-del command
// 4. Send the config-write command
// TODO: 5. Handle the insufficient permissions to write the configuration.
func (m *hostMigrator) Migrate() map[int64]error {
	// Clean up the error map.
	m.errs = make(map[int64]error)

	// Collect the daemon IDs to send all the commands to the same daemon in
	// a single batch.
	daemonsByIDs := make(map[int64]*dbmodel.Daemon)
	for _, host := range m.items {
		for _, localHost := range host.LocalHosts {
			daemon := localHost.Daemon
			if !daemon.Active {
				// Skip inactive daemons.
				continue
			}

			daemonsByIDs[localHost.DaemonID] = localHost.Daemon
		}
	}

	// Iterate over the daemons in the ascending order of their IDs.
	daemonIDs := make([]int64, 0, len(daemonsByIDs))
	for daemonID := range daemonsByIDs {
		daemonIDs = append(daemonIDs, daemonID)
	}
	slices.Sort(daemonIDs)

	for _, daemonID := range daemonIDs {
		daemon := daemonsByIDs[daemonID]
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
				return nil, err
			}

			commandAdd := keactrl.NewCommandReservationAdd(reservationAdd, daemon.Name)

			return commandAdd, nil
		})

		// Delete the reservations from the Kea configuration. It is done only
		// for the hosts that has been properly added to the host database or
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
				return nil, err
			}

			commandDel := keactrl.NewCommandReservationDel(reservationDel, daemon.Name)
			return commandDel, nil
		})

		// Make the configuration persistent.
		m.saveConfigChanges(daemon)
	}

	return m.errs
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
	daemonID := daemon.ID

	var commands []keactrl.SerializableCommand
	var commandHostIDs []int64

	for _, host := range m.items {
		if m.errs[host.ID] != nil {
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

		command, err := f(&host, localHosts)
		if err != nil {
			m.errs[host.ID] = err
			continue
		}
		if command == nil {
			// Nothing to send.
			continue
		}

		commands = append(commands, command)
		commandHostIDs = append(commandHostIDs, host.ID)
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
		// Communication error between the server and the agent.
		// Mark all related hosts as errored.
		m.setDaemonError(daemonID, err)
		return
	}

	// Communication error between the Kea CA and the Kea DHCP daemon.
	for i, err := range result.CmdsErrors {
		hostID := commandHostIDs[i]
		if m.errs[hostID] != nil {
			continue
		}

		if err == nil {
			continue
		}
		m.errs[hostID] = err
	}

	// Execution error of the command.
	for i, responsePerDaemon := range responses {
		hostID := commandHostIDs[i]
		if m.errs[hostID] != nil {
			continue
		}

		if len(*responsePerDaemon) == 0 {
			// Should not happen.
			continue
		}
		response := (*responsePerDaemon)[0]

		if err := response.GetError(); err != nil {
			m.errs[hostID] = err
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
		m.setDaemonError(daemon.ID, err)
	}
}

// Marks all hosts related to the daemon as errored.
func (m *hostMigrator) setDaemonError(daemonID int64, err error) {
	for _, host := range m.items {
		if m.errs[host.ID] != nil {
			continue
		}

		for _, localHost := range host.LocalHosts {
			if localHost.DaemonID == daemonID {
				m.errs[host.ID] = err
				break
			}
		}
	}
}
