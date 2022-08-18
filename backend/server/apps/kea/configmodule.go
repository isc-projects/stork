package kea

import (
	"context"

	pkgerrors "github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	agentcomm "isc.org/stork/server/agentcomm"
	config "isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
)

// A configuration manager module responsible for Kea configuration.
type ConfigModule struct {
	// A configuration manager owning the module.
	manager config.ManagerAccessors
}

// Creates new instance of the Kea configuration module.
func NewConfigModule(manager config.ManagerAccessors) *ConfigModule {
	return &ConfigModule{
		manager: manager,
	}
}

// Commits the Kea configuration changes.
func (module *ConfigModule) Commit(ctx context.Context) (context.Context, error) {
	var err error
	state, ok := config.GetTransactionState(ctx)
	if !ok {
		return ctx, pkgerrors.Errorf("context lacks state")
	}
	for _, pu := range state.Updates {
		switch pu.Operation {
		case "host_add":
			ctx, err = module.commitHostAdd(ctx)
		case "host_update":
			ctx, err = module.commitHostUpdate(ctx)
		case "host_delete":
			ctx, err = module.commitHostDelete(ctx)
		default:
			err = pkgerrors.Errorf("unknown operation %s when called Commit()", pu.Operation)
		}
		if err != nil {
			return ctx, err
		}
	}
	return ctx, err
}

// Begins adding a new host reservation. It initializes transaction state.
func (module *ConfigModule) BeginHostAdd(ctx context.Context) (context.Context, error) {
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate("kea", "host_add")
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Applies new host reservation. It prepares necessary commands to be sent
// to Kea upon commit.
func (module *ConfigModule) ApplyHostAdd(ctx context.Context, host *dbmodel.Host) (context.Context, error) {
	if len(host.LocalHosts) == 0 {
		return ctx, pkgerrors.Errorf("applied host %d is not associated with any daemon", host.ID)
	}
	var (
		commands []interface{}
		lookup   dbmodel.DHCPOptionDefinitionLookup
	)
	for _, lh := range host.LocalHosts {
		if lh.Daemon == nil {
			return ctx, pkgerrors.Errorf("applied host %d is associated with nil daemon", host.ID)
		}
		if lh.Daemon.App == nil {
			return ctx, pkgerrors.Errorf("applied host %d is associated with nil app", host.ID)
		}
		// Convert the host information to Kea reservation.
		reservation, err := keaconfig.CreateHostCmdsReservation(lh.DaemonID, lookup, host)
		if err != nil {
			return ctx, err
		}
		// Create command arguments.
		arguments := make(map[string]interface{})
		arguments["reservation"] = reservation
		// Associate the command with an app receiving this command.
		appCommand := make(map[string]interface{})
		appCommand["command"] = keactrl.NewCommand("reservation-add", []string{lh.Daemon.Name}, arguments)
		appCommand["app"] = lh.Daemon.App
		commands = append(commands, appCommand)
	}
	var err error
	if ctx, err = config.SetValueForUpdate(ctx, 0, "commands", commands); err != nil {
		return ctx, err
	}
	return ctx, nil
}

// Create the host reservation in the Kea servers.
func (module *ConfigModule) commitHostAdd(ctx context.Context) (context.Context, error) {
	return module.commitHostChanges(ctx)
}

// Begins a host reservation update. It fetches the specified host reservation
// from the database and stores it in the context state. Then, it locks the
// daemons associated with the host for updates.
func (module *ConfigModule) BeginHostUpdate(ctx context.Context, hostID int64) (context.Context, error) {
	// Try to get the host to be updated from the database.
	host, err := dbmodel.GetHost(module.manager.GetDB(), hostID)
	if err != nil {
		// Internal database error.
		return ctx, err
	}
	// Host does not exist.
	if host == nil {
		return ctx, pkgerrors.WithStack(config.NewHostNotFoundError(hostID))
	}
	// Get the list of daemons for whose configurations must be locked for
	// updates.
	var daemonIDs []int64
	for _, lh := range host.LocalHosts {
		daemonIDs = append(daemonIDs, lh.DaemonID)
	}
	// Try to lock configurations.
	ctx, err = module.manager.Lock(ctx, daemonIDs...)
	if err != nil {
		return ctx, pkgerrors.WithStack(config.NewLockError())
	}
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate("kea", "host_update", daemonIDs...)
	if err := state.SetValueForUpdate(0, "host_before_update", *host); err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Applies updated host reservation. It prepares necessary commands to be sent
// to Kea upon commit. Kea does not provide a command to update host reservations.
// Therefore, it sends reservation-del followed by reservation-add to each
// daemon owning the reservation.
func (module *ConfigModule) ApplyHostUpdate(ctx context.Context, host *dbmodel.Host) (context.Context, error) {
	if len(host.LocalHosts) == 0 {
		return ctx, pkgerrors.Errorf("applied host %d is not associated with any daemon", host.ID)
	}
	var (
		commands []any
		lookup   dbmodel.DHCPOptionDefinitionLookup
	)
	for _, lh := range host.LocalHosts {
		if lh.Daemon == nil {
			return ctx, pkgerrors.Errorf("applied host %d is associated with nil daemon", host.ID)
		}
		if lh.Daemon.App == nil {
			return ctx, pkgerrors.Errorf("applied host %d is associated with nil app", host.ID)
		}
		// Convert the host information to Kea reservation.
		deletedReservation, err := keaconfig.CreateHostCmdsDeletedReservation(lh.DaemonID, host)
		if err != nil {
			return ctx, err
		}
		// Create command arguments.
		deleteArguments := deletedReservation
		// Associate the command with an app receiving this command.
		appCommand := make(map[string]any)
		appCommand["command"] = keactrl.NewCommand("reservation-del", []string{lh.Daemon.Name}, deleteArguments)
		appCommand["app"] = lh.Daemon.App
		commands = append(commands, appCommand)

		// Convert the updated host information to Kea reservation.
		reservation, err := keaconfig.CreateHostCmdsReservation(lh.DaemonID, lookup, host)
		if err != nil {
			return ctx, err
		}
		// Create command arguments.
		addArguments := make(map[string]any)
		addArguments["reservation"] = reservation
		appCommand = make(map[string]any)
		appCommand["command"] = keactrl.NewCommand("reservation-add", []string{lh.Daemon.Name}, addArguments)
		appCommand["app"] = lh.Daemon.App
		commands = append(commands, appCommand)
	}
	return config.SetValueForUpdate(ctx, 0, "commands", commands)
}

// Create the updated host reservation in the Kea servers.
func (module *ConfigModule) commitHostUpdate(ctx context.Context) (context.Context, error) {
	return module.commitHostChanges(ctx)
}

// Begins deleting a host reservation. Currently it is no-op but may evolve
// in the future.
func (module *ConfigModule) BeginHostDelete(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// Creates requests to delete host reservation. It prepares necessary commands to be sent
// to Kea upon commit.
func (module *ConfigModule) ApplyHostDelete(ctx context.Context, host *dbmodel.Host) (context.Context, error) {
	if len(host.LocalHosts) == 0 {
		return ctx, pkgerrors.Errorf("deleted host %d is not associated with any daemon", host.ID)
	}
	var commands []interface{}
	for _, lh := range host.LocalHosts {
		if lh.Daemon == nil {
			return ctx, pkgerrors.Errorf("deleted host %d is associated with nil daemon", host.ID)
		}
		if lh.Daemon.App == nil {
			return ctx, pkgerrors.Errorf("deleted host %d is associated with nil app", host.ID)
		}
		// Convert the host information to Kea reservation.
		reservation, err := keaconfig.CreateHostCmdsDeletedReservation(lh.DaemonID, host)
		if err != nil {
			return ctx, err
		}
		// Create command arguments.
		arguments := reservation
		// Associate the command with an app receiving this command.
		appCommand := make(map[string]interface{})
		appCommand["command"] = keactrl.NewCommand("reservation-del", []string{lh.Daemon.Name}, arguments)
		appCommand["app"] = lh.Daemon.App
		commands = append(commands, appCommand)
	}
	daemonIDs, _ := ctx.Value(config.DaemonsContextKey).([]int64)
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate("kea", "host_delete", daemonIDs...)
	if err := state.SetValueForUpdate(0, "commands", commands); err != nil {
		return ctx, err
	}
	if err := state.SetValueForUpdate(0, "id", host.ID); err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Delete host reservation from the Kea servers.
func (module *ConfigModule) commitHostDelete(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState(ctx)
	if !ok {
		return ctx, pkgerrors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitHostChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		hostID, err := update.GetRecipeValueAsInt64("id")
		if err != nil {
			return ctx, err
		}
		err = dbmodel.DeleteHost(module.manager.GetDB(), hostID)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// Generic function used to commit host changes (i.e., delete,  add or update host reservation)
// using the data stored in the context.
func (module *ConfigModule) commitHostChanges(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState(ctx)
	if !ok {
		return ctx, pkgerrors.New("context lacks state")
	}
	for _, update := range state.Updates {
		// Retrieve associations between the commands and apps.
		appCommands, ok := update.Recipe["commands"]
		if !ok {
			return ctx, pkgerrors.New("Kea commands not found in the context")
		}
		// Iterate over the associations.
		for _, acs := range appCommands.([]interface{}) {
			// Split the commands and apps.
			var (
				command keactrl.Command
				app     agentcomm.ControlledApp
			)
			if state.Scheduled {
				// If the context has been re-created after scheduling the config
				// change in the database we use a simplified structure holding the
				// App information.
				var commandApp struct {
					Command keactrl.Command
					App     config.App
				}
				if err := config.DecodeContextData(acs, &commandApp); err != nil {
					return ctx, err
				}
				app = commandApp.App
				command = commandApp.Command
			} else {
				// We didn't schedule the change so we have an original context
				// with an app represented using dbmodel.App structure.
				var commandApp struct {
					Command keactrl.Command
					App     dbmodel.App
				}
				if err := config.DecodeContextData(acs, &commandApp); err != nil {
					return ctx, err
				}
				app = commandApp.App
				command = commandApp.Command
			}
			// Send the command to Kea.
			var response keactrl.ResponseList
			result, err := module.manager.GetConnectedAgents().ForwardToKeaOverHTTP(context.Background(), app, []keactrl.SerializableCommand{command}, &response)
			if err == nil {
				if err = result.GetFirstError(); err == nil {
					for _, r := range response {
						if err = keactrl.GetResponseError(r); err != nil {
							break
						}
					}
				}
			}
			if err != nil {
				err = pkgerrors.WithMessagef(err, "%s command to %s failed", command.GetCommand(), app.GetName())
				return ctx, err
			}
		}
	}
	return ctx, nil
}
