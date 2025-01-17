package configmigrator

import (
	"context"

	"github.com/go-pg/pg/v10"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps/kea"
	dbmodel "isc.org/stork/server/database/model"
)

type hostMigrator struct {
	db               *pg.DB
	filter           dbmodel.HostsByPageFilters
	items            []dbmodel.Host
	limit            int64
	dhcpOptionLookup keaconfig.DHCPOptionDefinitionLookup
	configModule     *kea.ConfigModule
	connectedAgents  agentcomm.ConnectedAgents
}

var _ Migrator = &hostMigrator{}

func (m *hostMigrator) CountTotal() (int64, error) {
	_, count, err := dbmodel.GetHostsByPage(m.db, 0, 0, m.filter, "", dbmodel.SortDirAny)
	return count, err
}

func (m *hostMigrator) LoadItems(offset int64) (int64, error) {
	items, count, err := dbmodel.GetHostsByPage(m.db, offset, m.limit, m.filter, "id", dbmodel.SortDirAsc)
	if err != nil {
		// Returns the number of items tried to load.
		return m.limit, err
	}
	m.items = items
	return count, nil
}

// Adds the hosts to the database. Sends the delete command to Kea.
//
// Algorithm:
//
// 1. Send the reservation-add command
// TODO: 2. Detect unexpected changes in the Kea JSON configuration (before config-set)
// 3. Send the reservation-local-del command
// 4. Send the config-write command
// 5. Handle the insufficient permissions to write the configuration
func (m *hostMigrator) Migrate() map[int64]error {
	errs := make(map[int64]error)

	// Collect the daemon IDs to send all the commands to the same daemon in
	// a single batch.
	daemonsByIDs := make(map[int64]*dbmodel.Daemon)
	for _, host := range m.items {
		for _, localHost := range host.LocalHosts {
			daemonsByIDs[localHost.DaemonID] = localHost.Daemon
		}
	}

	// 1. Send the reservation-add command.
	for daemonID, daemon := range daemonsByIDs {
		var commands []keactrl.SerializableCommand

		for _, host := range m.items {
			if errs[host.ID] != nil {
				continue
			}

			// To do group local hosts by daemon ID.
			for _, localHost := range host.LocalHosts {
				if localHost.DaemonID != daemonID {
					continue
				}
				if !localHost.DataSource.IsConfig() {
					continue
				}

				// Add the reservation to the Kea host database.
				// TODO: Check if the host is already in the Kea database.
				reservationAdd, err := keaconfig.CreateHostCmdsReservation(
					daemonID,
					m.dhcpOptionLookup,
					host,
				)

				if err != nil {
					errs[host.ID] = err
					break
				}

				commandAdd := keactrl.NewCommandReservationAdd(reservationAdd, daemon.Name)

				// Remove the reservation from the Kea configuration.
				reservationDel, err := keaconfig.CreateHostCmdsDeletedReservation(
					daemonID, host, keaconfig.HostCmdsOperationTargetMemory,
				)

				if err != nil {
					errs[host.ID] = err
					break
				}

				commandDel := keactrl.NewCommandReservationDel(reservationDel, daemon.Name)

				commands = append(commands, commandAdd)
				commands = append(commands, commandDel)
			}
		}

		// Write the changes to the Kea configuration.
		command := keactrl.NewCommandBase(keactrl.ConfigWrite, daemon.Name)
		commands = append(commands, command)

		// Send the commands.
		if err := m.sendCommands(daemon.App, commands); err != nil {
			errs[daemonID] = err
		}
	}

	return errs
}

func (m *hostMigrator) sendCommands(app *dbmodel.App, commands []keactrl.SerializableCommand) error {
	var response keactrl.ResponseList
	result, err := m.connectedAgents.ForwardToKeaOverHTTP(
		context.Background(), app, commands, &response,
	)

	// Communication error between the server and the agent.
	if err != nil {
		return err
	}

	// Communication error between the agent and the command handler.
	if err = result.GetFirstError(); err != nil {
		return err
	}

	// Execution error of the command.
	for _, response := range response {
		if err = response.GetError(); err != nil {
			return err
		}
	}

	return nil
}
