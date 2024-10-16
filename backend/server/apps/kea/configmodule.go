package kea

import (
	"context"
	"encoding/json"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	config "isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

var (
	_ config.KeaModule                = (*ConfigModule)(nil)
	_ config.TransactionStateAccessor = (*config.TransactionState[ConfigRecipe])(nil)
)

// Contains a Kea command along with an app instance to which the
// command should be sent to apply configuration changes. Multiple
// such commands can occur in a config recipe.
type ConfigCommand struct {
	Command *keactrl.Command
	App     *dbmodel.App
}

type GlobalConfigRecipeParams struct {
	KeaDaemonsBeforeConfigUpdate []dbmodel.Daemon
	KeaDaemonsAfterConfigUpdate  []dbmodel.Daemon
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

// A structure embedded in the ConfigRecipe grouping parameters used
// in transactions adding, updating and deleting shared networks.
type SharedNetworkConfigRecipeParams struct {
	// An instance of the shared network before an update. It is typically fetched
	// at the beginning of the shared network update (e.g., when a user clicks the
	// shared network edit button).
	SharedNetworkBeforeUpdate *dbmodel.SharedNetwork
	// An instance of the shared network after it has been added or updated. This
	// instance is held in the context until it is committed  or scheduled
	// for committing later. It is set when a new subnet is added or an
	// existing subnet is updated.
	SharedNetworkAfterUpdate *dbmodel.SharedNetwork
	// Edited or deleted shared network ID.
	SharedNetworkID *int64
}

// A structure embedded in the ConfigRecipe grouping parameters used
// in transactions adding, updating and deleting subnets.
type SubnetConfigRecipeParams struct {
	// An instance of the subnet before an update. It is typically fetched
	// at the beginning of the subnet update (e.g., when a user clicks the
	// subnet edit button).
	SubnetBeforeUpdate *dbmodel.Subnet
	// An instance of the subnet after it has been added or updated. This
	// instance is held in the context until it is committed  or scheduled
	// for committing later. It is set when a new subnet is added or an
	// existing subnet is updated.
	SubnetAfterUpdate *dbmodel.Subnet
	// Edited or deleted subnet ID.
	SubnetID *int64
}

// Represents a Kea config change recipe. A recipe is associated with
// each config update and may comprise several commands sent to different
// Kea servers. Other data stored in the recipe structure are used in the
// Kea config module to pass the information between various configuration
// stages (begin, apply, commit/schedule). This structure is meant to be
// generic for different configuration use cases in Kea.
type ConfigRecipe struct {
	// A list of commands and the corresponding targets to be sent to
	// apply a configuration update.
	Commands []ConfigCommand
	// Embedded structure holding the parameters appropriate for the
	// global parameters management.
	GlobalConfigRecipeParams
	// Embedded structure holding the parameters appropriate for the
	// host management.
	HostConfigRecipeParams
	// Embedded structure holding the parameters appropriate for the
	// shared network management.
	SharedNetworkConfigRecipeParams
	// Embedded structure holding the parameters appropriate for the
	// subnet management.
	SubnetConfigRecipeParams
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
		return ctx, errors.Errorf("context lacks state")
	}
	for _, pu := range state.Updates {
		switch pu.Operation {
		case "global_parameters_update":
			ctx, err = module.commitGlobalParametersUpdate(ctx)
		case "host_add":
			ctx, err = module.commitHostAdd(ctx)
		case "host_update":
			ctx, err = module.commitHostUpdate(ctx)
		case "host_delete":
			ctx, err = module.commitHostDelete(ctx)
		case "shared_network_add":
			ctx, err = module.commitSharedNetworkAdd(ctx)
		case "shared_network_update":
			ctx, err = module.commitSharedNetworkUpdate(ctx)
		case "shared_network_delete":
			ctx, err = module.commitSharedNetworkDelete(ctx)
		case "subnet_add":
			ctx, err = module.commitSubnetAdd(ctx)
		case "subnet_update":
			ctx, err = module.commitSubnetUpdate(ctx)
		case "subnet_delete":
			ctx, err = module.commitSubnetDelete(ctx)
		default:
			err = errors.Errorf("unknown operation %s when called Commit()", pu.Operation)
		}
		if err != nil {
			return ctx, err
		}
	}
	return ctx, err
}

// Begins updating global configuration parameters for one or more Kea servers.
// Since the configuration update can be performed for multiple daemons in a
// single transaction it is possible to specify multiple daemon IDs for which
// the transaction should be created.
func (module *ConfigModule) BeginGlobalParametersUpdate(ctx context.Context, daemonIDs []int64) (context.Context, error) {
	// Get the daemons with their configurations from the database.
	daemons, err := dbmodel.GetDaemonsByIDs(module.manager.GetDB(), daemonIDs)
	if err != nil {
		// Internal database error.
		return ctx, err
	}
	// Some daemons do not exist.
	if len(daemons) != len(daemonIDs) {
		return ctx, errors.WithStack(config.NewSomeDaemonsNotFoundError(daemonIDs...))
	}
	// Try to lock configurations.
	ctx, err = module.manager.Lock(ctx, daemonIDs...)
	if err != nil {
		return ctx, errors.WithStack(config.NewLockError())
	}
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "global_parameters_update", daemonIDs...)
	recipe := &ConfigRecipe{
		GlobalConfigRecipeParams: GlobalConfigRecipeParams{
			KeaDaemonsBeforeConfigUpdate: daemons,
		},
	}
	if err := state.SetRecipeForUpdate(0, recipe); err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Applies global configuration parameters update. It prepares necessary commands to be sent
// to Kea upon commit.
func (module *ConfigModule) ApplyGlobalParametersUpdate(ctx context.Context, daemonSettableConfigs []config.AnnotatedEntity[*keaconfig.SettableConfig]) (context.Context, error) {
	// Retrieve existing configs from the context.
	recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, 0)
	if err != nil {
		return ctx, err
	}
	// Retrieve the existing daemons with their configurations. The partial
	// configurations will be merged into them.
	existingDaemons := recipe.KeaDaemonsBeforeConfigUpdate
	if existingDaemons == nil {
		return ctx, errors.New("internal server error - existing Kea configs cannot be nil when committing global parameters update")
	}
	var (
		commands         []ConfigCommand
		updatedDaemonIDs []int64
	)
	// Iterate over the received partial configs and match them with the daemons
	// for which the config update is performed.
	for _, daemonSettableConfig := range daemonSettableConfigs {
		for _, existingDaemon := range existingDaemons {
			if daemonSettableConfig.GetID() == existingDaemon.ID {
				// Merge the partial configuration into the existing configuration.
				err = existingDaemon.KeaDaemon.Config.Merge(daemonSettableConfig.GetEntity())
				if err != nil {
					return ctx, err
				}
				appCommand := ConfigCommand{
					Command: keactrl.NewCommandConfigSet(existingDaemon.KeaDaemon.Config.Config, existingDaemon.Name),
					App:     existingDaemon.App,
				}
				commands = append(commands, appCommand)
				updatedDaemonIDs = append(updatedDaemonIDs, daemonSettableConfig.GetID())
			}
		}
	}
	if len(existingDaemons) != len(updatedDaemonIDs) {
		return ctx, config.NewInvalidConfigsError(updatedDaemonIDs...)
	}
	// Each config-set must come with config-write to persist the configuration.
	for _, existingDaemon := range existingDaemons {
		appCommand := ConfigCommand{
			Command: keactrl.NewCommandBase(keactrl.ConfigWrite, existingDaemon.Name),
			App:     existingDaemon.App,
		}
		commands = append(commands, appCommand)
	}
	// Remember the modified configurations.
	recipe.KeaDaemonsAfterConfigUpdate = existingDaemons
	recipe.Commands = commands
	return config.SetRecipeForUpdate(ctx, 0, recipe)
}

// Sends commands to Kea to update the global configuration parameters.
// It also updates the respective configurations in the Stork database.
func (module *ConfigModule) commitGlobalParametersUpdate(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, errors.New("context lacks state")
	}
	// Send the commands to the Kea servers.
	var err error
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.KeaDaemonsAfterConfigUpdate == nil {
			return ctx, errors.New("server logic error: the update.Recipe.KeaDaemonsAfterConfigUpdate cannot be nil when committing the global config update")
		}
		// Update the daemons with their configurations in the Stork database.
		for _, daemon := range update.Recipe.KeaDaemonsAfterConfigUpdate {
			err = module.manager.GetDB().RunInTransaction(context.Background(), func(tx *pg.Tx) error {
				err := dbmodel.UpdateDaemon(tx, &daemon)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return ctx, errors.WithMessage(err, "global configuration updated in Kea but updating it in the Stork database failed")
			}
		}
	}
	return ctx, nil
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
		return ctx, errors.New("applied host is not associated with any daemon")
	}
	var commands []ConfigCommand
	for _, lh := range host.LocalHosts {
		if lh.Daemon == nil {
			return ctx, errors.New("applied host is associated with nil daemon")
		}
		if lh.Daemon.App == nil {
			return ctx, errors.New("applied host is associated with nil app")
		}
		// Convert the host information to Kea reservation.
		lookup := module.manager.GetDHCPOptionDefinitionLookup()
		reservation, err := keaconfig.CreateHostCmdsReservation(lh.DaemonID, lookup, host)
		if err != nil {
			return ctx, err
		}
		// Associate the command with an app receiving this command.
		appCommand := ConfigCommand{
			Command: keactrl.NewCommandReservationAdd(reservation, lh.Daemon.Name),
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
		return ctx, errors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.HostAfterUpdate == nil {
			return ctx, errors.New("server logic error: the update.Recipe.HostAfterUpdate cannot be nil when committing host creation")
		}
		err = dbmodel.AddHost(module.manager.GetDB(), update.Recipe.HostAfterUpdate)
		if err != nil {
			return ctx, errors.WithMessagef(err, "host has been successfully added to Kea but adding to the Stork database failed")
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
		return ctx, errors.WithStack(config.NewHostNotFoundError(hostID))
	}
	// Get the list of daemons whose configurations must be locked for updates.
	var daemonIDs []int64
	for _, lh := range host.LocalHosts {
		// Skip the local hosts from the configuration file.
		if lh.DataSource.IsAPI() {
			daemonIDs = append(daemonIDs, lh.DaemonID)
		}
	}
	// Try to lock configurations.
	ctx, err = module.manager.Lock(ctx, daemonIDs...)
	if err != nil {
		return ctx, errors.Wrap(config.NewLockError(), err.Error())
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
		return ctx, errors.Errorf("applied host %d is not associated with any daemon", host.ID)
	}
	// Retrieve existing host from the context. We will need it for sending
	// the reservation-del commands, in case the DHCP identifier changes.
	recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, 0)
	if err != nil {
		return ctx, err
	}
	existingHost := recipe.HostBeforeUpdate
	if existingHost == nil {
		return ctx, errors.New("internal server error - host instance cannot be nil when committing host update")
	}

	var commands []ConfigCommand
	// First, delete all instances of the host on all Kea servers.
	for _, lh := range existingHost.LocalHosts {
		if !lh.DataSource.IsAPI() {
			continue
		}
		if lh.Daemon == nil {
			return ctx, errors.Errorf("updated host %d is associated with nil daemon", host.ID)
		}
		if lh.Daemon.App == nil {
			return ctx, errors.Errorf("updated host %d is associated with nil app", host.ID)
		}
		// Convert the host information to Kea reservation.
		deleteArguments, err := keaconfig.CreateHostCmdsDeletedReservation(lh.DaemonID, existingHost)
		if err != nil {
			return ctx, err
		}
		// Associate the command with an app receiving this command.
		appCommand := ConfigCommand{
			Command: keactrl.NewCommandReservationDel(deleteArguments, lh.Daemon.Name),
			App:     lh.Daemon.App,
		}
		commands = append(commands, appCommand)
	}
	// Re-create the host reservations.
	for _, lh := range host.LocalHosts {
		if !lh.DataSource.IsAPI() {
			continue
		}
		if lh.Daemon == nil {
			return ctx, errors.Errorf("applied host %d is associated with nil daemon", host.ID)
		}
		if lh.Daemon.App == nil {
			return ctx, errors.Errorf("applied host %d is associated with nil app", host.ID)
		}
		// Convert the updated host information to Kea reservation.
		lookup := module.manager.GetDHCPOptionDefinitionLookup()
		reservation, err := keaconfig.CreateHostCmdsReservation(lh.DaemonID, lookup, host)
		if err != nil {
			return ctx, err
		}
		// Create command arguments.
		appCommand := ConfigCommand{
			Command: keactrl.NewCommandReservationAdd(reservation, lh.Daemon.Name),
			App:     lh.Daemon.App,
		}
		commands = append(commands, appCommand)
	}

	// Append the local hosts from the configuration file to the edited host.
	// The edit form doesn't attach them to the data sent to the server.
	// It must be done after re-creating the host reservations to avoid
	// duplicating the DHCP options.
	for _, lh := range existingHost.LocalHosts {
		if !lh.DataSource.IsAPI() {
			host.LocalHosts = append(host.LocalHosts, lh)
		}
	}

	recipe.HostAfterUpdate = host
	recipe.Commands = commands
	return config.SetRecipeForUpdate(ctx, 0, recipe)
}

// Create the updated host reservation in the Kea servers.
func (module *ConfigModule) commitHostUpdate(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, errors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.HostAfterUpdate == nil {
			return ctx, errors.New("server logic error: the update.Recipe.HostAfterUpdate cannot be nil when committing the host update")
		}

		// Update the host in the database.
		err = dbmodel.UpdateHost(module.manager.GetDB(), update.Recipe.HostAfterUpdate)
		if err != nil {
			return ctx, errors.WithMessagef(err, "host has been successfully updated in Kea but updating it in the Stork database failed")
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
		return ctx, errors.Errorf("deleted host %d is not associated with any daemon", host.ID)
	}
	var commands []ConfigCommand
	for _, lh := range host.LocalHosts {
		if !lh.DataSource.IsAPI() {
			continue
		}
		if lh.Daemon == nil {
			return ctx, errors.Errorf("deleted host %d is associated with nil daemon", host.ID)
		}
		if lh.Daemon.App == nil {
			return ctx, errors.Errorf("deleted host %d is associated with nil app", host.ID)
		}
		// Convert the host information to Kea reservation.
		reservation, err := keaconfig.CreateHostCmdsDeletedReservation(lh.DaemonID, host)
		if err != nil {
			return ctx, err
		}
		// Associate the command with an app receiving this command.
		appCommand := ConfigCommand{
			Command: keactrl.NewCommandReservationDel(reservation, lh.Daemon.Name),
			App:     lh.Daemon.App,
		}
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
		return ctx, errors.New("context lacks state")
	}

	var err error
	db := module.manager.GetDB()
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.HostID == nil {
			return ctx, errors.New("server logic error: the host ID cannot be nil when committing host deletion")
		}

		host, err := dbmodel.GetHost(db, *update.Recipe.HostID)
		if err != nil {
			return ctx, errors.WithMessagef(err, "could not retrieve host %d from the database", *update.Recipe.HostID)
		}

		// If there is any local host from the configuration, keep the host and
		// remove the local hosts from the API.
		hasLocalHostFromConfig := false
		for _, lh := range host.LocalHosts {
			if lh.DataSource.IsConfig() {
				hasLocalHostFromConfig = true
				break
			}
		}

		if !hasLocalHostFromConfig {
			err = dbmodel.DeleteHost(module.manager.GetDB(), *update.Recipe.HostID)
		} else {
			// TODO: The removal of the local hosts from the JSON configuration
			// is not implemented yet. But it should be done soon because this
			// feature is needed to clean up the configuration after migrating
			// the hosts from the configuration to the database.
			// After the implementation, the following code should be changed
			// to always remove the hosts, not only when there are no local
			// hosts from the configuration.
			_, err = dbmodel.DeleteDaemonsFromHost(db, *update.Recipe.HostID, dbmodel.HostDataSourceAPI)
		}
		if err != nil {
			return ctx, errors.WithMessagef(err, "host has been successfully deleted in Kea but deleting in the Stork database failed")
		}
	}
	return ctx, nil
}

// Generic function used to commit configuration changes (e.g., delete, add or update host reservation)
// using the data stored in the context.
func (module *ConfigModule) commitChanges(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, errors.New("context lacks state")
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
				err = errors.WithMessagef(err, "%s command to %s failed", acs.Command.GetCommand(), acs.App.GetName())
				return ctx, err
			}
		}
	}
	return ctx, nil
}

// Begins adding a new shared network. It initializes transaction state.
func (module *ConfigModule) BeginSharedNetworkAdd(ctx context.Context) (context.Context, error) {
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "shared_network_add")
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Applies new shared network. It prepares necessary commands to be sent to Kea upon commit.
func (module *ConfigModule) ApplySharedNetworkAdd(ctx context.Context, sharedNetwork *dbmodel.SharedNetwork) (context.Context, error) {
	if len(sharedNetwork.LocalSharedNetworks) == 0 {
		return ctx, errors.Errorf("applied shared network %s is not associated with any daemon", sharedNetwork.Name)
	}
	// Retrieve existing shared network from the context. We need it for sending
	// the network4-del or network6-del commands.
	recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, 0)
	if err != nil {
		return ctx, err
	}

	var commands []ConfigCommand
	// Add the shared network instances.
	for _, lsn := range sharedNetwork.LocalSharedNetworks {
		if lsn.Daemon == nil {
			return ctx, errors.Errorf("applied shared network %s is associated with nil daemon", sharedNetwork.Name)
		}
		if lsn.Daemon.App == nil {
			return ctx, errors.Errorf("applied shared network %s is associated with nil app", sharedNetwork.Name)
		}
		// Convert the shared network information to Kea shared network.
		lookup := module.manager.GetDHCPOptionDefinitionLookup()
		appCommand := ConfigCommand{
			App: lsn.Daemon.App,
		}
		switch sharedNetwork.Family {
		case 4:
			sharedNetwork4, err := keaconfig.CreateSharedNetwork4(lsn.DaemonID, lookup, sharedNetwork)
			if err != nil {
				return ctx, err
			}
			appCommand.Command = keactrl.NewCommandNetwork4Add(sharedNetwork4, lsn.Daemon.Name)
			commands = append(commands, appCommand)

		default:
			sharedNetwork6, err := keaconfig.CreateSharedNetwork6(lsn.DaemonID, lookup, sharedNetwork)
			if err != nil {
				return ctx, err
			}
			appCommand.Command = keactrl.NewCommandNetwork6Add(sharedNetwork6, lsn.Daemon.Name)
			commands = append(commands, appCommand)
		}
	}
	// Create the commands to write the updated configuration to files. The shared network
	// changes won't persist across the servers' restarts otherwise.
	for _, lsn := range sharedNetwork.LocalSharedNetworks {
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandBase(keactrl.ConfigWrite, lsn.Daemon.Name),
			App:     lsn.Daemon.App,
		})
		// Kea versions up to 2.6.0 do not update statistics after modifying pools with the
		// subnet_cmds hook library. Therefore, for these versions we send the config-reload
		// command to force the statistics update. There is no lighter command to force the
		// statistics update unfortunately.
		version := storkutil.ParseSemanticVersionOrLatest(lsn.Daemon.Version)
		if version.LessThan(storkutil.NewSemanticVersion(2, 6, 0)) {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandBase(keactrl.ConfigReload, lsn.Daemon.Name),
				App:     lsn.Daemon.App,
			})
		}
	}

	// Store the data in the existing recipe.
	recipe.SharedNetworkAfterUpdate = sharedNetwork
	recipe.Commands = commands
	return config.SetRecipeForUpdate(ctx, 0, recipe)
}

// Create the shared network in the Kea servers.
func (module *ConfigModule) commitSharedNetworkAdd(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, errors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for i, update := range state.Updates {
		if update.Recipe.SharedNetworkAfterUpdate == nil {
			return ctx, errors.New("server logic error: the update.Recipe.SharedNetworkAfterUpdate cannot be nil when committing the shared network")
		}
		sharedNetworks := []dbmodel.SharedNetwork{*update.Recipe.SharedNetworkAfterUpdate}
		_, err = dbmodel.CommitNetworksIntoDB(module.manager.GetDB(), sharedNetworks, []dbmodel.Subnet{})
		if err != nil {
			return ctx, errors.WithMessagef(err, "shared network has been successfully created in Kea but creating it in the Stork database failed")
		}
		recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, i)
		if err != nil {
			return ctx, err
		}
		recipe.SharedNetworkID = storkutil.Ptr(sharedNetworks[0].ID)
		if ctx, err = config.SetRecipeForUpdate(ctx, 0, recipe); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// Begins a shared network update. It fetches the specified shared network from
// the database and stores it in the context state. Then, it locks the daemons
// associated with the shared network for updates.
func (module *ConfigModule) BeginSharedNetworkUpdate(ctx context.Context, sharedNetworkID int64) (context.Context, error) {
	// Try to get the shared network to be updated from the database.
	sharedNetwork, err := dbmodel.GetSharedNetwork(module.manager.GetDB(), sharedNetworkID)
	if err != nil {
		// Internal database error.
		return ctx, err
	}
	// Shared network does not exist.
	if sharedNetwork == nil {
		return ctx, errors.WithStack(config.NewSharedNetworkNotFoundError(sharedNetworkID))
	}
	// Fill the daemon details.
	err = sharedNetwork.PopulateDaemons(module.manager.GetDB())
	if err != nil {
		return ctx, err
	}
	// Get the list of daemons for whose configurations must be locked for
	// updates.
	var daemonIDs []int64
	for _, ls := range sharedNetwork.LocalSharedNetworks {
		if ls.Daemon.KeaDaemon.Config == nil {
			return ctx, errors.Errorf("configuration not found for daemon %d", ls.DaemonID)
		}
		if _, _, exists := ls.Daemon.KeaDaemon.Config.GetHookLibrary("libdhcp_subnet_cmds"); !exists {
			return ctx, errors.WithStack(config.NewNoSubnetCmdsHookError())
		}
		daemonIDs = append(daemonIDs, ls.DaemonID)
	}
	// Try to lock configurations.
	ctx, err = module.manager.Lock(ctx, daemonIDs...)
	if err != nil {
		return ctx, errors.WithStack(config.NewLockError())
	}
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "shared_network_update", daemonIDs...)
	recipe := &ConfigRecipe{
		SharedNetworkConfigRecipeParams: SharedNetworkConfigRecipeParams{
			SharedNetworkBeforeUpdate: sharedNetwork,
		},
	}
	if err := state.SetRecipeForUpdate(0, recipe); err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Applies updated shared network. It prepares necessary commands to be sent to Kea
// upon commit.
func (module *ConfigModule) ApplySharedNetworkUpdate(ctx context.Context, sharedNetwork *dbmodel.SharedNetwork) (context.Context, error) {
	if len(sharedNetwork.LocalSharedNetworks) == 0 {
		return ctx, errors.Errorf("applied shared network %s is not associated with any daemon", sharedNetwork.Name)
	}
	// Retrieve existing shared network from the context. We need it for sending
	// the network4-del or network6-del commands.
	recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, 0)
	if err != nil {
		return ctx, err
	}
	existingSharedNetwork := recipe.SharedNetworkBeforeUpdate
	if existingSharedNetwork == nil {
		return ctx, errors.New("internal server error - shared network instance cannot be nil when committing shared network update")
	}

	var commands []ConfigCommand
	// Update the shared network instances.
	for _, lsn := range sharedNetwork.LocalSharedNetworks {
		if lsn.Daemon == nil {
			return ctx, errors.Errorf("applied shared network %s is associated with nil daemon", sharedNetwork.Name)
		}
		if lsn.Daemon.App == nil {
			return ctx, errors.Errorf("applied shared network %s is associated with nil app", sharedNetwork.Name)
		}
		// Convert the updated shared network information to Kea shared network.
		lookup := module.manager.GetDHCPOptionDefinitionLookup()
		appCommand := ConfigCommand{}
		switch sharedNetwork.Family {
		case 4:
			deletedSharedNetwork4 := keaconfig.CreateSubnetCmdsDeletedSharedNetwork(lsn.DaemonID, existingSharedNetwork, keaconfig.SharedNetworkSubnetsActionDelete)
			appCommand.Command = keactrl.NewCommandNetwork4Del(deletedSharedNetwork4, lsn.Daemon.Name)
			appCommand.App = lsn.Daemon.App
			commands = append(commands, appCommand)

			sharedNetwork4, err := keaconfig.CreateSharedNetwork4(lsn.DaemonID, lookup, sharedNetwork)
			if err != nil {
				return ctx, err
			}
			appCommand.Command = keactrl.NewCommandNetwork4Add(sharedNetwork4, lsn.Daemon.Name)
			commands = append(commands, appCommand)

		default:
			deletedSharedNetwork6 := keaconfig.CreateSubnetCmdsDeletedSharedNetwork(lsn.DaemonID, existingSharedNetwork, keaconfig.SharedNetworkSubnetsActionDelete)
			appCommand.Command = keactrl.NewCommandNetwork6Del(deletedSharedNetwork6, lsn.Daemon.Name)
			appCommand.App = lsn.Daemon.App
			commands = append(commands, appCommand)

			sharedNetwork6, err := keaconfig.CreateSharedNetwork6(lsn.DaemonID, lookup, sharedNetwork)
			if err != nil {
				return ctx, err
			}
			appCommand.Command = keactrl.NewCommandNetwork6Add(sharedNetwork6, lsn.Daemon.Name)
			commands = append(commands, appCommand)
		}
	}
	// Identify the daemons which no longer exist in the updated shared network.
	// Remove the shared network from these daemons.
	var deletedLocalSharedNetworks []*dbmodel.LocalSharedNetwork
	for i, exln := range existingSharedNetwork.LocalSharedNetworks {
		deletedLocalSharedNetwork := existingSharedNetwork.LocalSharedNetworks[i]
		for _, ln := range sharedNetwork.LocalSharedNetworks {
			if exln.DaemonID == ln.DaemonID {
				// Daemon still exists. Do not remove.
				deletedLocalSharedNetwork = nil
				break
			}
		}
		if deletedLocalSharedNetwork != nil {
			appCommand := ConfigCommand{}
			deletedKeaSharedNetwork := keaconfig.CreateSubnetCmdsDeletedSharedNetwork(deletedLocalSharedNetwork.DaemonID, existingSharedNetwork, keaconfig.SharedNetworkSubnetsActionDelete)
			if sharedNetwork.Family == 6 {
				appCommand.Command = keactrl.NewCommandNetwork6Del(deletedKeaSharedNetwork, deletedLocalSharedNetwork.Daemon.Name)
			} else {
				appCommand.Command = keactrl.NewCommandNetwork4Del(deletedKeaSharedNetwork, deletedLocalSharedNetwork.Daemon.Name)
			}
			appCommand.App = deletedLocalSharedNetwork.Daemon.App
			commands = append(commands, appCommand)
			deletedLocalSharedNetworks = append(deletedLocalSharedNetworks, deletedLocalSharedNetwork)
		}
	}
	// Create the commands to write the updated configuration to files. The shared network
	// changes won't persist across the servers' restarts otherwise.
	for _, lsn := range append(sharedNetwork.LocalSharedNetworks, deletedLocalSharedNetworks...) {
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandBase(keactrl.ConfigWrite, lsn.Daemon.Name),
			App:     lsn.Daemon.App,
		})
		// Kea versions up to 2.6.0 do not update statistics after modifying pools with the
		// subnet_cmds hook library. Therefore, for these versions we send the config-reload
		// command to force the statistics update. There is no lighter command to force the
		// statistics update unfortunately.
		version := storkutil.ParseSemanticVersionOrLatest(lsn.Daemon.Version)
		if version.LessThan(storkutil.NewSemanticVersion(2, 6, 0)) {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandBase(keactrl.ConfigReload, lsn.Daemon.Name),
				App:     lsn.Daemon.App,
			})
		}
	}

	// Store the data in the existing recipe.
	recipe.SharedNetworkAfterUpdate = sharedNetwork
	recipe.Commands = commands
	return config.SetRecipeForUpdate(ctx, 0, recipe)
}

// Create the updated shared network in the Kea servers.
func (module *ConfigModule) commitSharedNetworkUpdate(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, errors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.SharedNetworkAfterUpdate == nil {
			return ctx, errors.New("server logic error: the update.Recipe.SharedNetworkAfterUpdate cannot be nil when committing the shared network update")
		}
		err = module.manager.GetDB().RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			// Remove associations between the daemons and the shared network because it is possible
			// that user has removed some of the associations. The ones left will be recreated.
			err := dbmodel.DeleteDaemonsFromSharedNetwork(tx, update.Recipe.SharedNetworkAfterUpdate.ID)
			if err == nil {
				// Recreate the associations.
				_, err = dbmodel.CommitNetworksIntoDB(tx, []dbmodel.SharedNetwork{*update.Recipe.SharedNetworkAfterUpdate}, []dbmodel.Subnet{})
			}
			return errors.WithMessagef(err, "shared network has been successfully updated in Kea but updating it in the Stork database failed")
		})
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// Begins deleting a shared network. Currently it is no-op but may evolve
// in the future.
func (module *ConfigModule) BeginSharedNetworkDelete(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// Creates requests to delete a shared network. It prepares necessary commands to be sent
// to Kea upon commit.
func (module *ConfigModule) ApplySharedNetworkDelete(ctx context.Context, sharedNetwork *dbmodel.SharedNetwork) (context.Context, error) {
	if len(sharedNetwork.LocalSharedNetworks) == 0 {
		return ctx, errors.Errorf("deleted shared network %d is not associated with any daemon", sharedNetwork.ID)
	}
	var commands []ConfigCommand
	for _, lsn := range sharedNetwork.LocalSharedNetworks {
		if lsn.Daemon == nil {
			return ctx, errors.Errorf("deleted shared network %d is associated with nil daemon", sharedNetwork.ID)
		}
		if lsn.Daemon.App == nil {
			return ctx, errors.Errorf("deleted shared network %d is associated with nil app", sharedNetwork.ID)
		}
		// Convert the shared network information to Kea shared network.
		deletedSharedNetwork := keaconfig.CreateSubnetCmdsDeletedSharedNetwork(lsn.DaemonID, sharedNetwork, keaconfig.SharedNetworkSubnetsActionDelete)

		// Create command arguments.
		arguments := deletedSharedNetwork
		// Associate the command with an app receiving this command.
		appCommand := ConfigCommand{}
		switch sharedNetwork.Family {
		case 4:
			appCommand.Command = keactrl.NewCommandNetwork4Del(arguments, lsn.Daemon.Name)
		default:
			appCommand.Command = keactrl.NewCommandNetwork6Del(arguments, lsn.Daemon.Name)
		}
		appCommand.App = lsn.Daemon.App
		commands = append(commands, appCommand)
	}
	// Persist the configuration changes.
	for _, ls := range sharedNetwork.LocalSharedNetworks {
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandBase(keactrl.ConfigWrite, ls.Daemon.Name),
			App:     ls.Daemon.App,
		})
	}
	daemonIDs, _ := ctx.Value(config.DaemonsContextKey).([]int64)
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "shared_network_delete", daemonIDs...)
	recipe := ConfigRecipe{
		Commands: commands,
		SharedNetworkConfigRecipeParams: SharedNetworkConfigRecipeParams{
			SharedNetworkID: &sharedNetwork.ID,
		},
	}
	if err := state.SetRecipeForUpdate(0, &recipe); err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Delete shared network from the Kea servers.
func (module *ConfigModule) commitSharedNetworkDelete(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, errors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.SharedNetworkID == nil {
			return ctx, errors.New("server logic error: the shared network ID cannot be nil when committing shared network deletion")
		}
		err = dbmodel.DeleteSharedNetworkWithSubnets(module.manager.GetDB(), *update.Recipe.SharedNetworkID)
		if err != nil {
			return ctx, errors.WithMessagef(err, "shared network has been successfully deleted in Kea but deleting in the Stork database failed")
		}
	}
	return ctx, nil
}

// Begins adding a new subnet. It initializes transaction state.
func (module *ConfigModule) BeginSubnetAdd(ctx context.Context) (context.Context, error) {
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "subnet_add")
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Applies new subnet. It prepares necessary commands to be sent to Kea upon commit.
func (module *ConfigModule) ApplySubnetAdd(ctx context.Context, subnet *dbmodel.Subnet) (context.Context, error) {
	if len(subnet.LocalSubnets) == 0 {
		return ctx, errors.Errorf("applied subnet %s is not associated with any daemon", subnet.Prefix)
	}
	// Get the highest local subnet ID from the database.
	localSubnetID, err := dbmodel.GetMaxLocalSubnetID(module.manager.GetDB())
	if err != nil {
		return ctx, errors.WithMessagef(err, "failed querying the database to generate ID for the new subnet")
	}
	// Next subnet ID should be available. Assign it to all local subnets.
	localSubnetID++
	for i := range subnet.LocalSubnets {
		subnet.LocalSubnets[i].LocalSubnetID = localSubnetID
	}

	var sharedNetworkNameAfterUpdate string
	if subnet.SharedNetwork != nil {
		sharedNetworkNameAfterUpdate = subnet.SharedNetwork.Name
	}

	var commands []ConfigCommand
	// Update the subnet instances.
	for _, ls := range subnet.LocalSubnets {
		if ls.Daemon == nil {
			return ctx, errors.Errorf("applied subnet %s is associated with nil daemon", subnet.Prefix)
		}
		if ls.Daemon.App == nil {
			return ctx, errors.Errorf("applied subnet %s is associated with nil app", subnet.Prefix)
		}
		// Convert the updated subnet information to Kea subnet.
		lookup := module.manager.GetDHCPOptionDefinitionLookup()
		appCommand := ConfigCommand{}
		switch subnet.GetFamily() {
		case 4:
			// Create subnet4-add command.
			subnet4, err := keaconfig.CreateSubnet4(ls.DaemonID, lookup, subnet)
			if err != nil {
				return ctx, err
			}
			appCommand.Command = keactrl.NewCommandSubnet4Add(subnet4, ls.Daemon.Name)
			appCommand.App = ls.Daemon.App
			commands = append(commands, appCommand)

			// If the subnet is associated with a shared network, add this association
			// in Kea.
			if sharedNetworkNameAfterUpdate != "" {
				appCommand.Command = keactrl.NewCommandNetwork4SubnetAdd(sharedNetworkNameAfterUpdate, ls.LocalSubnetID, ls.Daemon.Name)
				commands = append(commands, appCommand)
			}
		default:
			// Create subnet6-add command.
			subnet6, err := keaconfig.CreateSubnet6(ls.DaemonID, lookup, subnet)
			if err != nil {
				return ctx, err
			}
			appCommand.Command = keactrl.NewCommandSubnet6Add(subnet6, ls.Daemon.Name)
			appCommand.App = ls.Daemon.App
			commands = append(commands, appCommand)

			// If the subnet is associated with a new shared network, add this association
			// in Kea.
			if sharedNetworkNameAfterUpdate != "" {
				appCommand.Command = keactrl.NewCommandNetwork6SubnetAdd(sharedNetworkNameAfterUpdate, ls.LocalSubnetID, ls.Daemon.Name)
				commands = append(commands, appCommand)
			}
		}
	}
	// Create the commands to write the updated configuration to files. The subnet
	// changes won't persist across the servers' restarts otherwise.
	for _, ls := range subnet.LocalSubnets {
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandBase(keactrl.ConfigWrite, ls.Daemon.Name),
			App:     ls.Daemon.App,
		})
		// Kea versions up to 2.6.0 do not update statistics after modifying pools with the
		// subnet_cmds hook library. Therefore, for these versions we send the config-reload
		// command to force the statistics update. There is no lighter command to force the
		// statistics update unfortunately.
		version := storkutil.ParseSemanticVersionOrLatest(ls.Daemon.Version)
		if version.LessThan(storkutil.NewSemanticVersion(2, 6, 0)) {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandBase(keactrl.ConfigReload, ls.Daemon.Name),
				App:     ls.Daemon.App,
			})
		}
	}

	// Store the data in the recipe.
	recipe := &ConfigRecipe{
		SubnetConfigRecipeParams: SubnetConfigRecipeParams{
			SubnetAfterUpdate: subnet,
		},
		Commands: commands,
	}
	recipe.SubnetAfterUpdate = subnet
	recipe.Commands = commands
	return config.SetRecipeForUpdate(ctx, 0, recipe)
}

// Create the subnet in the Kea servers.
func (module *ConfigModule) commitSubnetAdd(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, errors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for i, update := range state.Updates {
		if update.Recipe.SubnetAfterUpdate == nil {
			return ctx, errors.New("server logic error: the update.Recipe.SubnetAfterUpdate cannot be nil when committing the subnet creation")
		}
		addedSubnets, err := dbmodel.CommitNetworksIntoDB(module.manager.GetDB(), []dbmodel.SharedNetwork{}, []dbmodel.Subnet{*update.Recipe.SubnetAfterUpdate})
		if err != nil {
			return ctx, errors.WithMessagef(err, "subnet has been successfully created in Kea but updating it in the Stork database failed")
		}
		if len(addedSubnets) != 1 {
			return ctx, errors.Errorf("subnet has been successfully created in Kea but Stork was unable to determine its new identifier")
		}
		recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, i)
		if err != nil {
			return ctx, err
		}
		recipe.SubnetID = storkutil.Ptr(addedSubnets[0].ID)
		if ctx, err = config.SetRecipeForUpdate(ctx, 0, recipe); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// Begins a subnet update. It fetches the specified subnet from the database
// and stores it in the context state. Then, it locks the daemons associated
// with the subnet for updates.
func (module *ConfigModule) BeginSubnetUpdate(ctx context.Context, subnetID int64) (context.Context, error) {
	// Try to get the subnet to be updated from the database.
	subnet, err := dbmodel.GetSubnet(module.manager.GetDB(), subnetID)
	if err != nil {
		// Internal database error.
		return ctx, err
	}
	// Subnet does not exist.
	if subnet == nil {
		return ctx, errors.WithStack(config.NewSubnetNotFoundError(subnetID))
	}

	// Get the list of daemons for whose configurations must be locked for
	// updates.
	var daemonIDs []int64
	for _, ls := range subnet.LocalSubnets {
		if ls.Daemon.KeaDaemon.Config == nil {
			return ctx, errors.Errorf("configuration not found for daemon %d", ls.DaemonID)
		}
		if _, _, exists := ls.Daemon.KeaDaemon.Config.GetHookLibrary("libdhcp_subnet_cmds"); !exists {
			return ctx, errors.WithStack(config.NewNoSubnetCmdsHookError())
		}
		daemonIDs = append(daemonIDs, ls.DaemonID)
	}
	// Try to lock configurations.
	ctx, err = module.manager.Lock(ctx, daemonIDs...)
	if err != nil {
		return ctx, errors.WithStack(config.NewLockError())
	}
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "subnet_update", daemonIDs...)
	recipe := &ConfigRecipe{
		SubnetConfigRecipeParams: SubnetConfigRecipeParams{
			SubnetBeforeUpdate: subnet,
		},
	}
	if err := state.SetRecipeForUpdate(0, recipe); err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Applies updated subnet. It prepares necessary commands to be sent to Kea upon commit.
func (module *ConfigModule) ApplySubnetUpdate(ctx context.Context, subnet *dbmodel.Subnet) (context.Context, error) {
	if len(subnet.LocalSubnets) == 0 {
		return ctx, errors.Errorf("applied subnet %s is not associated with any daemon", subnet.Prefix)
	}
	// Retrieve existing subnet from the context. We may need it for sending
	// the subnet4-del or subnet6-del commands.
	recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, 0)
	if err != nil {
		return ctx, err
	}
	existingSubnet := recipe.SubnetBeforeUpdate
	if existingSubnet == nil {
		return ctx, errors.New("internal server error - subnet instance cannot be nil when committing subnet update")
	}

	var (
		sharedNetworkNameBeforeUpdate string
		sharedNetworkNameAfterUpdate  string
	)
	if existingSubnet.SharedNetwork != nil {
		sharedNetworkNameBeforeUpdate = existingSubnet.SharedNetwork.Name
	}
	if subnet.SharedNetwork != nil {
		sharedNetworkNameAfterUpdate = subnet.SharedNetwork.Name
	}

	var commands []ConfigCommand
	// Update the subnet instances.
	for _, ls := range subnet.LocalSubnets {
		if ls.Daemon == nil {
			return ctx, errors.Errorf("applied subnet %s is associated with nil daemon", subnet.Prefix)
		}
		if ls.Daemon.App == nil {
			return ctx, errors.Errorf("applied subnet %s is associated with nil app", subnet.Prefix)
		}
		// Check if this is a new association.
		existingAssociation := false
		for _, exls := range existingSubnet.LocalSubnets {
			if exls.DaemonID == ls.DaemonID {
				existingAssociation = true
				break
			}
		}
		// Convert the updated subnet information to Kea subnet.
		lookup := module.manager.GetDHCPOptionDefinitionLookup()
		appCommand := ConfigCommand{}
		switch subnet.GetFamily() {
		case 4:
			// Create subnet4-add or subnet4-update depending on whether it is a new
			// subnet or an updated subnet.
			subnet4, err := keaconfig.CreateSubnet4(ls.DaemonID, lookup, subnet)
			if err != nil {
				return ctx, err
			}
			if existingAssociation {
				appCommand.Command = keactrl.NewCommandSubnet4Update(subnet4, ls.Daemon.Name)
			} else {
				appCommand.Command = keactrl.NewCommandSubnet4Add(subnet4, ls.Daemon.Name)
			}
			appCommand.App = ls.Daemon.App
			commands = append(commands, appCommand)

			// If the association of the subnet with the shared network hasn't changed we
			// move on to the next local subnet.
			if sharedNetworkNameBeforeUpdate == sharedNetworkNameAfterUpdate {
				continue
			}

			// If the subnet association with a shared network existed, we need to remove
			// this association first.
			if sharedNetworkNameBeforeUpdate != "" {
				appCommand.Command = keactrl.NewCommandNetwork4SubnetDel(sharedNetworkNameBeforeUpdate, ls.LocalSubnetID, ls.Daemon.Name)
				commands = append(commands, appCommand)
			}

			// If the subnet is associated with a new shared network, add this association
			// in Kea.
			if sharedNetworkNameAfterUpdate != "" {
				appCommand.Command = keactrl.NewCommandNetwork4SubnetAdd(sharedNetworkNameAfterUpdate, ls.LocalSubnetID, ls.Daemon.Name)
				commands = append(commands, appCommand)
			}
		default:
			// Create subnet6-add or subnet6-update depending on whether it is a new
			// subnet or an updated subnet.
			subnet6, err := keaconfig.CreateSubnet6(ls.DaemonID, lookup, subnet)
			if err != nil {
				return ctx, err
			}
			if existingAssociation {
				appCommand.Command = keactrl.NewCommandSubnet6Update(subnet6, ls.Daemon.Name)
			} else {
				appCommand.Command = keactrl.NewCommandSubnet6Add(subnet6, ls.Daemon.Name)
			}
			appCommand.App = ls.Daemon.App
			commands = append(commands, appCommand)

			if sharedNetworkNameBeforeUpdate == sharedNetworkNameAfterUpdate {
				continue
			}

			// If the subnet association with a shared network existed, we need to remove
			// this association first.
			if sharedNetworkNameBeforeUpdate != "" {
				appCommand.Command = keactrl.NewCommandNetwork6SubnetDel(sharedNetworkNameBeforeUpdate, ls.LocalSubnetID, ls.Daemon.Name)
				commands = append(commands, appCommand)
			}

			// If the subnet is associated with a new shared network, add this association
			// in Kea.
			if sharedNetworkNameAfterUpdate != "" {
				appCommand.Command = keactrl.NewCommandNetwork6SubnetAdd(sharedNetworkNameAfterUpdate, ls.LocalSubnetID, ls.Daemon.Name)
				commands = append(commands, appCommand)
			}
		}
	}
	// Identify the daemons which no longer exist in the updated subnet.
	// Remove the subnet from these daemons.
	var removedLocalSubnets []*dbmodel.LocalSubnet
	for i, exls := range existingSubnet.LocalSubnets {
		removedLocalSubnet := existingSubnet.LocalSubnets[i]
		for _, ls := range subnet.LocalSubnets {
			if exls.DaemonID == ls.DaemonID {
				// Daemon still exists. Do not remove.
				removedLocalSubnet = nil
				break
			}
		}
		if removedLocalSubnet != nil {
			if sharedNetworkNameBeforeUpdate != "" {
				// If the deleted subnet belongs to a shared network we first need to remove
				// this subnet from a shared network. This is a limitation of Kea 2.6.0.
				commands = append(commands, ConfigCommand{
					Command: keactrl.NewCommandNetworkSubnetDel(subnet.GetFamily(), sharedNetworkNameBeforeUpdate, removedLocalSubnet.LocalSubnetID, removedLocalSubnet.Daemon.Name),
					App:     removedLocalSubnet.Daemon.App,
				})
			}
			// Delete the subnet.
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandSubnetDel(subnet.GetFamily(), &keaconfig.SubnetCmdsDeletedSubnet{ID: removedLocalSubnet.LocalSubnetID}, removedLocalSubnet.Daemon.Name),
				App:     removedLocalSubnet.Daemon.App,
			})
			removedLocalSubnets = append(removedLocalSubnets, removedLocalSubnet)
		}
	}

	// Create the commands to write the updated configuration to files. The subnet
	// changes won't persist across the servers' restarts otherwise.
	for _, ls := range append(subnet.LocalSubnets, removedLocalSubnets...) {
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandBase(keactrl.ConfigWrite, ls.Daemon.Name),
			App:     ls.Daemon.App,
		})
		// Kea versions up to 2.6.0 do not update statistics after modifying pools with the
		// subnet_cmds hook library. Therefore, for these versions we send the config-reload
		// command to force the statistics update. There is no lighter command to force the
		// statistics update unfortunately.
		version := storkutil.ParseSemanticVersionOrLatest(ls.Daemon.Version)
		if version.LessThan(storkutil.NewSemanticVersion(2, 6, 0)) {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandBase(keactrl.ConfigReload, ls.Daemon.Name),
				App:     ls.Daemon.App,
			})
		}
	}

	// Store the data in the existing recipe.
	recipe.SubnetAfterUpdate = subnet
	recipe.Commands = commands
	return config.SetRecipeForUpdate(ctx, 0, recipe)
}

// Create the updated subnet in the Kea servers.
func (module *ConfigModule) commitSubnetUpdate(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, errors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.SubnetAfterUpdate == nil {
			return ctx, errors.New("server logic error: the update.Recipe.SubnetAfterUpdate cannot be nil when committing the subnet update")
		}
		_, err := dbmodel.CommitNetworksIntoDB(module.manager.GetDB(), []dbmodel.SharedNetwork{}, []dbmodel.Subnet{*update.Recipe.SubnetAfterUpdate})
		if err != nil {
			return ctx, errors.WithMessagef(err, "subnet has been successfully updated in Kea but updating it in the Stork database failed")
		}
	}
	return ctx, nil
}

// Creates requests to delete a subnet. It prepares necessary commands to be sent
// to Kea upon commit.
func (module *ConfigModule) ApplySubnetDelete(ctx context.Context, subnet *dbmodel.Subnet) (context.Context, error) {
	if len(subnet.LocalSubnets) == 0 {
		return ctx, errors.Errorf("deleted subnet %d is not associated with any daemon", subnet.ID)
	}
	var commands []ConfigCommand
	for _, ls := range subnet.LocalSubnets {
		if ls.Daemon == nil {
			return ctx, errors.Errorf("deleted subnet %d is associated with nil daemon", subnet.ID)
		}
		if ls.Daemon.App == nil {
			return ctx, errors.Errorf("deleted subnet %d is associated with nil app", subnet.ID)
		}
		// Convert the host information to Kea reservation.
		deletedSubnet, err := keaconfig.CreateSubnetCmdsDeletedSubnet(ls.DaemonID, subnet)
		if err != nil {
			return ctx, err
		}
		// If the deleted subnet belongs to a shared network we first need to remove
		// this subnet from a shared network. This is a Kea limitation described in
		// https://gitlab.isc.org/isc-projects/kea/-/issues/3455.
		if subnet.SharedNetwork != nil && subnet.SharedNetwork.Name != "" {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandNetworkSubnetDel(subnet.GetFamily(), subnet.SharedNetwork.Name, ls.LocalSubnetID, ls.Daemon.Name),
				App:     ls.Daemon.App,
			})
		}
		// Delete the subnet.
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandSubnetDel(subnet.GetFamily(), deletedSubnet, ls.Daemon.Name),
			App:     ls.Daemon.App,
		})
	}
	// Persist the configuration changes.
	for _, ls := range subnet.LocalSubnets {
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandBase(keactrl.ConfigWrite, ls.Daemon.Name),
			App:     ls.Daemon.App,
		})
	}
	daemonIDs, _ := ctx.Value(config.DaemonsContextKey).([]int64)
	// Create transaction state.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe]("kea", "subnet_delete", daemonIDs...)
	recipe := ConfigRecipe{
		Commands: commands,
		SubnetConfigRecipeParams: SubnetConfigRecipeParams{
			SubnetID: &subnet.ID,
		},
	}
	if err := state.SetRecipeForUpdate(0, &recipe); err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, config.StateContextKey, *state)
	return ctx, nil
}

// Delete subnet from the Kea servers.
func (module *ConfigModule) commitSubnetDelete(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	if !ok {
		return ctx, errors.New("context lacks state")
	}
	var err error
	ctx, err = module.commitChanges(ctx)
	if err != nil {
		return ctx, err
	}
	for _, update := range state.Updates {
		if update.Recipe.SubnetID == nil {
			return ctx, errors.New("server logic error: the subnet ID cannot be nil when committing subnet deletion")
		}
		err = dbmodel.DeleteSubnet(module.manager.GetDB(), *update.Recipe.SubnetID)
		if err != nil {
			return ctx, errors.WithMessagef(err, "subnet has been successfully deleted in Kea but deleting in the Stork database failed")
		}
	}
	return ctx, nil
}
