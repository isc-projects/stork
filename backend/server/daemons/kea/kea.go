package kea

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keaconfig "isc.org/stork/daemoncfg/kea"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
	storkutil "isc.org/stork/util"
)

// Get list of hooks for the given Kea daemon.
func GetDaemonHooks(dbDaemon *dbmodel.Daemon) (hooks []string) {
	if dbDaemon.KeaDaemon == nil || dbDaemon.KeaDaemon.Config == nil {
		return
	}
	libraries := dbDaemon.KeaDaemon.Config.GetHookLibraries()
	for _, library := range libraries {
		hooks = append(hooks, library.Library)
	}
	return
}

// The arguments of the version-get command response.
type VersionGetRespArgs struct {
	Extended string
}

// The response of the version-get command.
type VersionGetResponse struct {
	keactrl.ResponseHeader
	Arguments *VersionGetRespArgs `json:"arguments,omitempty"`
}

// Struct containing the events related to changes in the daemon state and
// the change status.
type DaemonStateMeta struct {
	Events          []*dbmodel.Event
	IsConfigChanged bool
}

// Get configuration from Kea daemon using ForwardToKeaOverHTTP function.
// Return a config, its hash and an error if any.
func GetConfig(ctx context.Context, agents agentcomm.ConnectedAgents, daemon agentcomm.ControlledDaemon) (*keaconfig.Config, error) {
	// prepare the command to get config and version from CA
	commands := []keactrl.SerializableCommand{
		keactrl.NewCommandBase(keactrl.ConfigGet, daemon.GetName()),
	}

	var configGetResponse keactrl.Response

	result, err := agents.ForwardToKeaOverHTTP(ctx, daemon, commands, &configGetResponse)
	if err != nil {
		return nil, errors.WithMessage(err, "problem communicating with Stork agent")
	}
	if err := result.GetFirstError(); err != nil {
		return nil, errors.WithMessage(err, "problem with config-get response")
	}
	if err = configGetResponse.GetError(); err != nil {
		return nil, err
	}
	if configGetResponse.Arguments == nil {
		return nil, errors.Errorf("empty arguments")
	}

	return keaconfig.NewConfig(configGetResponse.Arguments)
}

// Returns a new instance of Kea daemon with a refreshed state fetched from Kea.
// It doesn't modify the provided daemon.
func getDaemonWithRefreshedState(ctx context.Context, agents agentcomm.ConnectedAgents, inDaemon *dbmodel.Daemon) (daemon *dbmodel.Daemon, err error) {
	// Output daemon.
	daemon = dbmodel.ShallowCopyKeaDaemon(inDaemon)
	daemon.Active = true

	defer func() {
		if err != nil {
			// In case of error, mark the daemon as inactive.
			daemon.Active = false
		}
	}()

	now := storkutil.UTCNow()

	isDHCPDaemon := daemon.Name.IsDHCP()

	var versionGetResponse VersionGetResponse
	var configGetResponse keactrl.Response
	var statusGetResponse StatusGetResponse

	cmds := []keactrl.SerializableCommand{
		keactrl.NewCommandBase(keactrl.VersionGet, daemon.Name),
		keactrl.NewCommandBase(keactrl.ConfigGet, daemon.Name),
	}
	responses := []any{&versionGetResponse, &configGetResponse}

	if isDHCPDaemon {
		cmds = append(cmds, keactrl.NewCommandBase(keactrl.StatusGet, daemon.Name))
		responses = append(responses, &statusGetResponse)
	}

	var cmdsResult *agentcomm.KeaCmdsResult
	cmdsResult, err = agents.ForwardToKeaOverHTTP(ctx, daemon, cmds, responses...)
	if err != nil {
		return daemon, err
	}
	if err = cmdsResult.GetFirstError(); err != nil {
		return daemon, err
	}

	// process version-get responses
	if err = versionGetResponse.GetError(); err != nil {
		err = errors.WithMessage(err, "problem with version-get response")
		return daemon, err
	}
	daemon.Version = versionGetResponse.Text
	if versionGetResponse.Arguments != nil {
		daemon.ExtendedVersion = versionGetResponse.Arguments.Extended
	}

	// process config-get responses
	if err = configGetResponse.GetError(); err != nil {
		err = errors.WithMessage(err, "problem with config-get and kea daemon")
		return daemon, err
	}

	// Set the configuration for the daemon and populate selected configuration
	// information to the respective structures, e.g. logging information.
	// It does nothing if the configuration has not changed.
	err = daemon.SetKeaConfigFromJSON(configGetResponse.Arguments)
	if err != nil {
		return daemon, err
	}

	if isDHCPDaemon {
		if err = statusGetResponse.GetError(); err != nil {
			err = errors.WithMessage(err, "problem with status-get and kea daemon")
			return daemon, err
		}

		if statusGetResponse.Arguments != nil {
			daemon.Uptime = statusGetResponse.Arguments.Uptime
			daemon.ReloadedAt = now.Add(time.Second * time.Duration(-statusGetResponse.Arguments.Reload))
		}
	}

	return daemon, err
}

// Returns a new instance of Kea daemon with a refreshed state fetched from Kea,
// and an object representing the detected changes.
// It doesn't modify the provided daemon.
func GetDaemonWithRefreshedState(ctx context.Context, agents agentcomm.ConnectedAgents, inDaemon *dbmodel.Daemon) (outDaemon *dbmodel.Daemon, meta DaemonStateMeta) {
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// if no problems then now get state from the rest of Kea daemons
	var err error
	outDaemon, err = getDaemonWithRefreshedState(ctx2, agents, inDaemon)
	meta = findChangesAndRaiseEvents(inDaemon, outDaemon, err)
	return outDaemon, meta
}

// Checks if the configuration of the daemon has changed compared to the
// previous state.
func isDaemonConfigChanged(daemonOld, daemonNew *dbmodel.Daemon) bool {
	if daemonOld.KeaDaemon == nil && daemonNew.KeaDaemon == nil {
		return false
	}
	if daemonOld.KeaDaemon == nil || daemonNew.KeaDaemon == nil {
		return true
	}
	return daemonOld.KeaDaemon.ConfigHash != daemonNew.KeaDaemon.ConfigHash
}

// Detects changes in the daemon before and after the fetching state from Kea.
// It raises events when a daemon changes its state between active and
// inactive state. It also raises events about detected daemon restarts and when
// configuration change was detected.
func findChangesAndRaiseEvents(daemonOld, daemonNew *dbmodel.Daemon, err error) DaemonStateMeta {
	meta := DaemonStateMeta{
		IsConfigChanged: isDaemonConfigChanged(daemonOld, daemonNew),
	}

	if daemonNew.ID == 0 {
		// Daemon is newly added. It wasn't committed to the database yet.
		// Its events will be raised when the daemon is committed to the
		// database.
		return meta
	}

	if daemonOld.Active && !daemonNew.Active {
		// Kea daemon was not found in the response or it is inactive.
		ev := eventcenter.CreateEvent(dbmodel.EvError, "{daemon} is unreachable", err, daemonOld.Machine, daemonOld)
		meta.Events = append(meta.Events, ev)
	} else if !daemonOld.Active && daemonNew.Active {
		// Kea daemon is now active.
		ev := eventcenter.CreateEvent(dbmodel.EvInfo, "{daemon} is reachable now", daemonNew.Machine, daemonNew)
		meta.Events = append(meta.Events, ev)
	}

	if daemonOld.Uptime > daemonNew.Uptime {
		// Check if daemon has been restarted.
		text := "{daemon} has been restarted"
		ev := eventcenter.CreateEvent(dbmodel.EvWarning, text, daemonNew.Machine, daemonNew)
		meta.Events = append(meta.Events, ev)
	}

	if daemonOld.Version != daemonNew.Version {
		// Check if daemon version has changed.
		text := fmt.Sprintf("{daemon} version changed from %s to %s",
			daemonOld.Version, daemonNew.Version)
		ev := eventcenter.CreateEvent(dbmodel.EvWarning, text, daemonNew.Machine, daemonNew)
		meta.Events = append(meta.Events, ev)
	}

	if daemonOld.KeaDaemon != nil && daemonNew.KeaDaemon != nil {
		if meta.IsConfigChanged {
			// Raise this event only if we're certain that the configuration has
			// changed based on the comparison of the hash values.
			text := "Configuration change detected for {daemon}"
			ev := eventcenter.CreateEvent(dbmodel.EvInfo, text, daemonNew.Machine, daemonNew)
			meta.Events = append(meta.Events, ev)
		} else {
			log.Infof("Configuration of Kea: machine %d, daemon: %d has not changed since last fetch; skipping database update for that daemon", daemonNew.MachineID, daemonNew.ID)
		}
	}

	return meta
}

// Removes associations between the daemon, shared networks, subnets and hosts.
func deleteDaemonAssociations(tx *pg.Tx, daemon *dbmodel.Daemon) error {
	// Remove associations between the daemon and the existing hosts.
	// We will recreate the associations using new configuration.
	_, err := dbmodel.DeleteDaemonFromHosts(tx, daemon.ID, dbmodel.HostDataSourceConfig)
	if err != nil {
		return err
	}

	// Remove associations between the daemon and the subnets. We will
	// recreate the associations using new configuration.
	_, err = dbmodel.DeleteDaemonFromSubnets(tx, daemon.ID)
	if err != nil {
		return err
	}

	// Remove associations between the daemon and the subnets. We will
	// recreate the associations using new configuration.
	_, err = dbmodel.DeleteDaemonFromSharedNetworks(tx, daemon.ID)
	if err != nil {
		return err
	}

	// Remove associations between the daemon and the services. We will
	// recreate the associations using new configuration.
	_, err = dbmodel.DeleteDaemonFromServices(tx, daemon.ID)
	if err != nil {
		return err
	}

	return nil
}

// Deletes empty shared networks and orphaned subnets and hosts.
func deleteEmptyAndOrphanedObjects(tx *pg.Tx) error {
	// Removed the hosts that no longer belong to any daemon.
	_, err := dbmodel.DeleteOrphanedHosts(tx)
	if err != nil {
		return err
	}

	// Remove the subnets that no longer belong to any daemon.
	_, err = dbmodel.DeleteOrphanedSubnets(tx)
	if err != nil {
		return err
	}

	// Delete the shared networks that no longer belong to any daemon.
	_, err = dbmodel.DeleteOrphanedSharedNetworks(tx)
	if err != nil {
		return err
	}
	return nil
}

// Detects and commits the discovered services into the database.
func detectAndCommitServices(tx *pg.Tx, daemon *dbmodel.Daemon) error {
	// Check what HA services the daemon belongs to.
	services, err := DetectHAServices(tx, daemon)
	if err != nil {
		return err
	}

	// For the given daemon, iterate over the services and add/update them in the
	// database.
	err = dbmodel.CommitServicesIntoDB(tx, services, daemon)
	if err != nil {
		return err
	}
	return nil
}

// Adds events specific to the recent daemon subnets updates.
func addOnCommitSubnetEvents(daemon *dbmodel.Daemon, addedSubnets []*dbmodel.Subnet, eventCenter eventcenter.EventCenter) {
	if len(addedSubnets) > 0 {
		// add event per subnet only if there is not more than 10 subnets
		if len(addedSubnets) < 10 {
			for _, sn := range addedSubnets {
				eventCenter.AddInfoEvent("added {subnet} to {daemon} on {machine}", sn, daemon, daemon.Machine)
			}
		}
		t := fmt.Sprintf("added %d subnets to {daemon} on {machine}", len(addedSubnets))
		eventCenter.AddInfoEvent(t, daemon, daemon.Machine)
	}
}

// Inserts or updates information about Kea daemons in the database. Next, it extracts
// Kea's configurations and uses to either update or create new shared networks,
// subnets and pools. Finally, the relations between the subnets and the Kea daemon
// are created. Note that multiple daemons can be associated with the same subnet.
func CommitDaemonsIntoDB(db *dbops.PgDB, daemons []*dbmodel.Daemon, eventCenter eventcenter.EventCenter, states []DaemonStateMeta, lookup keaconfig.DHCPOptionDefinitionLookup) error {
	err := db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		// Add or update the daemon in the database. It must be done
		// before detecting the subnets and shared networks because we need to
		// know daemon IDs to associate the subnets and shared networks with.
		// The daemon IDs are assigned by the database when the daemons are
		// first added.
		for i, daemon := range daemons {
			if daemon.ID == 0 {
				// New daemon, insert it.
				if err := dbmodel.AddDaemon(tx, daemon); err != nil {
					return err
				}
				eventCenter.AddInfoEvent("added {daemon} to {machine}", daemon, daemon.Machine)
			} else {
				// Existing daemon, update it if needed.
				if err := dbmodel.UpdateDaemon(tx, daemon); err != nil {
					return err
				}
			}

			// Broadcast daemon-related events.
			state := states[i]
			for _, ev := range state.Events {
				eventCenter.AddEvent(ev)
			}
		}

		for i, daemon := range daemons {
			if !states[i].IsConfigChanged {
				// There are quite frequent cases when the daemons' configurations haven't
				// changed since last update. If that's the case, this map contains the
				// names of these daemons. For such daemons we should safely skip processing
				// subnets and shared networks. This saves many CPU cycles.
				continue
			}

			// Remove daemon associations with hosts, subnets and shared networks.
			err := deleteDaemonAssociations(tx, daemon)
			if err != nil {
				return err
			}

			// Go over the shared networks and subnets stored in the Kea configuration
			// and match them with the existing entries in the database. If some of
			// the shared networks or subnets do not exist they are instantiated and
			// returned here.
			networks, subnets, err := detectDaemonNetworks(tx, daemon, lookup)
			if err != nil {
				err = errors.WithMessagef(err, "unable to detect subnets and shared networks for Kea daemon %d", daemon.ID)
				return err
			}

			globalHosts, err := detectGlobalHostsFromConfig(tx, daemon, lookup)
			if err != nil {
				err = errors.WithMessagef(err, "unable to detect global host reservations for Kea daemon %d", daemon.ID)
				return err
			}

			// For the given daemon, iterate over the networks and subnets and update their
			// global instances accordingly in the database.
			addedSubnets, err := dbmodel.CommitNetworksIntoDB(tx, networks, subnets)
			if err != nil {
				return err
			}

			// For the given daemon, iterate over the global hosts and update their instances
			// in the database or insert them into the database.
			if err = dbmodel.CommitGlobalHostsIntoDB(tx, globalHosts); err != nil {
				return err
			}

			// Add subnet related events to the database.
			addOnCommitSubnetEvents(daemon, addedSubnets, eventCenter)

			if err = detectAndCommitServices(tx, daemon); err != nil {
				return err
			}
		}

		// Remove empty shared networks and orphaned subnets and hosts.
		err := deleteEmptyAndOrphanedObjects(tx)
		return err
	})
	return errors.WithMessage(err, "problem daemon committing updates")
}
