package kea

import (
	"context"
	"encoding/json"

	pkgerrors "github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	config "isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
)

var _ config.TransactionStateAccessor = (*config.TransactionState[ConfigRecipe])(nil)

// Contains a Kea command along with an app instance to which the
// command should be sent to apply configuration changes. Multiple
// such commands can occur in a config recipe.
type ConfigCommand struct {
	Command *keactrl.Command
	App     *dbmodel.App
}

// A structure embedded in the ConfigRecipe grouping parameters used
// in transactions adding, updating and deleting host reservations.
type HostConfigRecipeParams struct {
	// An instance of the host (reservation) before an update. It is
	// typically fetched at the beginning of the host update (e.g., when a
	// user clicks the host edit button).
	HostBeforeUpdate *dbmodel.Host
	// An instance of the host (reservation) after it has been added or
	// updated. This instance is held in the context until it is committed
	// or scheduled for committing later. It is set when a new host is
	// added or an existing host is updated.
	HostAfterUpdate *dbmodel.Host
	// Edited or deleted host ID.
	HostID *int64
}

// Represents a Kea config change recipe. A recipe is associated with
// each config update and may comprise several commands sent to different
// Kea servers. Other data stored in the recipe structure are used in the
// Kea config module to pass the information between various configuration
// stages (begin, apply, commit/schedule). This structure is meant to be
// generic for different configuration use cases in Kea. Should we need to
// support a subnet configuration we will extend it with a new embedded
// structure holding parameters appropriate for this new use case.
type ConfigRecipe struct {
	// A list of commands and the corresponding targets to be sent to
	// apply a configuration update.
	Commands []ConfigCommand
	// Embedded structure holding the parameters appropriate for the
	// host management.
	HostConfigRecipeParams
}

// A configuration manager module responsible for the Kea configuration.
type ConfigModule struct {
	// A configuration manager owning the module.
	manager config.ModuleManager
}

// Creates an instance of the Kea config update from the config update
// represented in the database.
func NewConfigUpdateFromDBModel(dbupdate *dbmodel.ConfigUpdate) *config.Update[ConfigRecipe] {
	update := &config.Update[ConfigRecipe]{
		Target:    dbupdate.Target,
		Operation: dbupdate.Operation,
		DaemonIDs: dbupdate.DaemonIDs,
	}
	if dbupdate.Recipe != nil {
		if err := json.Unmarshal(*dbupdate.Recipe, &update.Recipe); err != nil {
			return nil
		}
	}
	return update
}

// Creates new instance of the Kea configuration module.
func NewConfigModule(manager config.ModuleManager) *ConfigModule {
	return &ConfigModule{
		manager: manager,
	}
}

// Commits the Kea configuration changes.
func (module *ConfigModule) Commit(ctx context.Context) (context.Context, error) {
	var err error
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
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
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "host_add")
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Applies new host reservation. It prepares necessary commands to be sent
// to Kea upon commit.
func (module *ConfigModule) ApplyHostAdd(ctx context.Context, host *dbmodel.Host) (context.Context, error) {
	if len(host.LocalHosts) == 0 {
		return ctx, pkgerrors.Errorf("applied host %d is not associated with any daemon", host.ID)
	}
	var commands []ConfigCommand
	for _, lh := range host.LocalHosts {
		if lh.Daemon == nil {
			return ctx, pkgerrors.Errorf("applied host %d is associated with nil daemon", host.ID)
		}
		if lh.Daemon.App == nil {
			return ctx, pkgerrors.Errorf("applied host %d is associated with nil app", host.ID)
		}
		// Convert the host information to Kea reservation.
		lookup := module.manager.GetDHCPOptionDefinitionLookup()
		reservation, err := keaconfig.CreateHostCmdsReservation(lh.DaemonID, lookup, host)
		if err != nil {
			return ctx, err
		}
		// Create command arguments.
		arguments := make(map[string]interface{})
		arguments["reservation"] = reservation
		// Associate the command with an app receiving this command.
		appCommand := ConfigCommand{
			Command: keactrl.NewCommand("reservation-add", []string{lh.Daemon.Name}, arguments),
			App:     lh.Daemon.App,
		}
		commands = append(commands, appCommand)
	}
	var err error
	recipe := &ConfigRecipe{
		HostConfigRecipeParams: HostConfigRecipeParams{
			HostAfterUpdate: host,
		},
		Commands: commands,
	}
	if ctx, err = config.SetRecipeForUpdate(ctx, 0, recipe); err != nil {
		return ctx, err
	}
	return ctx, nil
}

// Create the host reservation in the Kea servers.
func (module *ConfigModule) commitHostAdd(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, pkgerrors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitHostChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.HostAfterUpdate == nil {
			return ctx, pkgerrors.New("server logic error: the update.Recipe.HostAfterUpdate cannot be nil when committing host creation")
		}
		err = dbmodel.AddHostWithLocalHosts(module.manager.GetDB(), update.Recipe.HostAfterUpdate)
		if err != nil {
			return ctx, pkgerrors.WithMessagef(err, "host has been successfully added to Kea but adding to the Stork database failed")
		}
	}
	return ctx, nil
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
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "host_update", daemonIDs...)
	recipe := &ConfigRecipe{
		HostConfigRecipeParams: HostConfigRecipeParams{
			HostBeforeUpdate: host,
		},
	}
	if err := state.SetRecipeForUpdate(0, recipe); err != nil {
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
	// Retrieve existing host from the context. We will need it for sending
	// the reservation-del commands, in case the DHCP identifier changes.
	recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, 0)
	if err != nil {
		return ctx, err
	}
	existingHost := recipe.HostBeforeUpdate
	if existingHost == nil {
		return ctx, pkgerrors.New("internal server error: host instance cannot be nil when committing host update")
	}

	var commands []ConfigCommand
	// First, delete all instances of the host on all Kea servers.
	for _, lh := range existingHost.LocalHosts {
		if lh.Daemon == nil {
			return ctx, pkgerrors.Errorf("updated host %d is associated with nil daemon", host.ID)
		}
		if lh.Daemon.App == nil {
			return ctx, pkgerrors.Errorf("updated host %d is associated with nil app", host.ID)
		}
		// Convert the host information to Kea reservation.
		deleteArguments, err := keaconfig.CreateHostCmdsDeletedReservation(lh.DaemonID, existingHost)
		if err != nil {
			return ctx, err
		}
		// Associate the command with an app receiving this command.
		appCommand := ConfigCommand{}
		appCommand.Command = keactrl.NewCommand("reservation-del", []string{lh.Daemon.Name}, deleteArguments)
		appCommand.App = lh.Daemon.App
		commands = append(commands, appCommand)
	}
	// Re-create the host reservations.
	for _, lh := range host.LocalHosts {
		if lh.Daemon == nil {
			return ctx, pkgerrors.Errorf("applied host %d is associated with nil daemon", host.ID)
		}
		if lh.Daemon.App == nil {
			return ctx, pkgerrors.Errorf("applied host %d is associated with nil app", host.ID)
		}
		// Convert the updated host information to Kea reservation.
		lookup := module.manager.GetDHCPOptionDefinitionLookup()
		reservation, err := keaconfig.CreateHostCmdsReservation(lh.DaemonID, lookup, host)
		if err != nil {
			return ctx, err
		}
		// Create command arguments.
		addArguments := make(map[string]any)
		addArguments["reservation"] = reservation
		appCommand := ConfigCommand{}
		appCommand.Command = keactrl.NewCommand("reservation-add", []string{lh.Daemon.Name}, addArguments)
		appCommand.App = lh.Daemon.App
		commands = append(commands, appCommand)
	}
	recipe.HostAfterUpdate = host
	recipe.Commands = commands
	return config.SetRecipeForUpdate(ctx, 0, recipe)
}

// Create the updated host reservation in the Kea servers.
func (module *ConfigModule) commitHostUpdate(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, pkgerrors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitHostChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.HostAfterUpdate == nil {
			return ctx, pkgerrors.New("server logic error: the update.Recipe.HostAfterUpdate cannot be nil when committing the host update")
		}
		err = dbmodel.UpdateHostWithLocalHosts(module.manager.GetDB(), update.Recipe.HostAfterUpdate)
		if err != nil {
			return ctx, pkgerrors.WithMessagef(err, "host has been successfully updated in Kea but updating it in the Stork database failed")
		}
	}
	return ctx, nil
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
	var commands []ConfigCommand
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
		appCommand := ConfigCommand{}
		appCommand.Command = keactrl.NewCommand("reservation-del", []string{lh.Daemon.Name}, arguments)
		appCommand.App = lh.Daemon.App
		commands = append(commands, appCommand)
	}
	daemonIDs, _ := ctx.Value(config.DaemonsContextKey).([]int64)
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "host_delete", daemonIDs...)
	recipe := ConfigRecipe{
		Commands: commands,
		HostConfigRecipeParams: HostConfigRecipeParams{
			HostID: &host.ID,
		},
	}
	if err := state.SetRecipeForUpdate(0, &recipe); err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Delete host reservation from the Kea servers.
func (module *ConfigModule) commitHostDelete(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, pkgerrors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitHostChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.HostID == nil {
			return ctx, pkgerrors.New("server logic error: the host ID cannot be nil when committing host deletion")
		}
		err = dbmodel.DeleteHost(module.manager.GetDB(), *update.Recipe.HostID)
		if err != nil {
			return ctx, pkgerrors.WithMessagef(err, "host has been successfully deleted in Kea but deleting in the Stork database failed")
		}
	}
	return ctx, nil
}

// Generic function used to commit host changes (i.e., delete,  add or update host reservation)
// using the data stored in the context.
func (module *ConfigModule) commitHostChanges(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, pkgerrors.New("context lacks state")
	}
	for _, update := range state.Updates {
		// Retrieve associations between the commands and apps.
		// Iterate over the associations.
		for _, acs := range update.Recipe.Commands {
			// Send the command to Kea.
			var response keactrl.ResponseList
			result, err := module.manager.GetConnectedAgents().ForwardToKeaOverHTTP(context.Background(), acs.App, []keactrl.SerializableCommand{acs.Command}, &response)
			// There was no error in communication between the server and the agent but
			// the agent could have issues with the Kea response.
			if err == nil {
				// Let's check if the agent found errors in communication with Kea.
				// If not, the individual Kea instances could return error codes as
				// a result of processing the commands.
				if err = result.GetFirstError(); err == nil {
					for _, r := range response {
						// Let's check if the individual Kea servers returned error
						// codes for the processed commands.
						if err = keactrl.GetResponseError(r); err != nil {
							break
						}
					}
				}
			}
			if err != nil {
				err = pkgerrors.WithMessagef(err, "%s command to %s failed", acs.Command.GetCommand(), acs.App.GetName())
				return ctx, err
			}
		}
	}
	return ctx, nil
}
