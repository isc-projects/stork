package apps

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps/bind9"
	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/configreview"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Instance of the puller which periodically checks the status of the Kea apps.
// Besides basic status information the High Availability status is fetched.
type StatePuller struct {
	*agentcomm.PeriodicPuller
	EventCenter                eventcenter.EventCenter
	ReviewDispatcher           configreview.Dispatcher
	DHCPOptionDefinitionLookup keaconfig.DHCPOptionDefinitionLookup
}

// Create an instance of the puller which periodically checks the status of
// the Kea apps.
func NewStatePuller(db *dbops.PgDB, agents agentcomm.ConnectedAgents, eventCenter eventcenter.EventCenter, reviewDispatcher configreview.Dispatcher, lookup keaconfig.DHCPOptionDefinitionLookup) (*StatePuller, error) {
	puller := &StatePuller{
		EventCenter:                eventCenter,
		ReviewDispatcher:           reviewDispatcher,
		DHCPOptionDefinitionLookup: lookup,
	}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Apps State puller",
		"apps_state_puller_interval", puller.pullData)
	if err != nil {
		return nil, err
	}
	puller.PeriodicPuller = periodicPuller
	return puller, nil
}

// Stops the timer triggering status checks.
func (puller *StatePuller) Shutdown() {
	puller.PeriodicPuller.Shutdown()
}

// Gets the status of machines and their apps and stores useful information in the database.
func (puller *StatePuller) pullData() error {
	// get list of all authorized machines from database
	authorized := true
	dbMachines, err := dbmodel.GetAllMachines(puller.DB, &authorized)
	if err != nil {
		return err
	}

	// get state from machines and their apps
	var lastErr error
	okCnt := 0
	for _, dbM := range dbMachines {
		dbM2 := dbM
		ctx := context.Background()
		errStr := UpdateMachineAndAppsState(ctx, puller.DB, &dbM2, puller.Agents, puller.EventCenter, puller.ReviewDispatcher, puller.DHCPOptionDefinitionLookup)
		if errStr != "" {
			lastErr = errors.New(errStr)
			log.Errorf("Error occurred while getting info from machine %d: %s", dbM2.ID, errStr)
		} else {
			okCnt++
		}
	}
	log.Printf("Completed pulling information from machines: %d/%d succeeded", okCnt, len(dbMachines))
	return lastErr
}

// Store updated machine fields in to database.
func updateMachineFields(db *dbops.PgDB, dbMachine *dbmodel.Machine, m *agentcomm.State) error {
	// update state fields in machine
	dbMachine.State.AgentVersion = m.AgentVersion
	dbMachine.State.Cpus = m.Cpus
	dbMachine.State.CpusLoad = m.CpusLoad
	dbMachine.State.Memory = m.Memory
	dbMachine.State.Hostname = m.Hostname
	dbMachine.State.Uptime = m.Uptime
	dbMachine.State.UsedMemory = m.UsedMemory
	dbMachine.State.Os = m.Os
	dbMachine.State.Platform = m.Platform
	dbMachine.State.PlatformFamily = m.PlatformFamily
	dbMachine.State.PlatformVersion = m.PlatformVersion
	dbMachine.State.KernelVersion = m.KernelVersion
	dbMachine.State.KernelArch = m.KernelArch
	dbMachine.State.VirtualizationSystem = m.VirtualizationSystem
	dbMachine.State.VirtualizationRole = m.VirtualizationRole
	dbMachine.State.HostID = m.HostID
	dbMachine.LastVisitedAt = m.LastVisitedAt
	dbMachine.Error = m.Error
	err := dbmodel.UpdateMachine(db, dbMachine)
	if err != nil {
		return errors.Wrapf(err, "problem updating machine %+v", dbMachine)
	}
	return nil
}

// appCompare compares two apps for equality.  Two apps are considered equal if
// their type matches and if they have the same control port.  Return true if
// equal, false otherwise.
func appCompare(dbApp *dbmodel.App, app *agentcomm.App) bool {
	if dbApp.Type.String() != app.Type {
		return false
	}

	var controlPortEqual bool
	for _, pt1 := range dbApp.AccessPoints {
		if pt1.Type != dbmodel.AccessPointControl {
			continue
		}
		for _, pt2 := range app.AccessPoints {
			if pt2.Type != dbmodel.AccessPointControl {
				continue
			}

			if pt1.Port == pt2.Port {
				controlPortEqual = true
				break
			}
		}

		// If a match is found, we can break.
		if controlPortEqual {
			break
		}
	}

	return controlPortEqual
}

// Get old apps from the machine db object and new apps retrieved from the machine remotely
// and merge them into one list of all, unique apps.
func mergeNewAndOldApps(db *dbops.PgDB, dbMachine *dbmodel.Machine, discoveredApps []*agentcomm.App) ([]*dbmodel.App, string) {
	// If there are any new apps then get their state and add to db.
	// Old ones are just updated. Use GetAppsByMachine to retrieve
	// machine's apps with their daemons.
	oldAppsList, err := dbmodel.GetAppsByMachine(db, dbMachine.ID)
	if err != nil {
		log.Error(err)
		return nil, "Cannot get machine's apps from db"
	}

	// count old apps
	oldKeaAppsCnt := 0
	oldBind9AppsCnt := 0
	for _, dbApp := range oldAppsList {
		if dbApp.Type == dbmodel.AppTypeKea {
			oldKeaAppsCnt++
		} else if dbApp.Type == dbmodel.AppTypeBind9 {
			oldBind9AppsCnt++
		}
	}

	// count new apps
	newKeaAppsCnt := 0
	newBind9AppsCnt := 0
	for _, app := range discoveredApps {
		if app.Type == dbmodel.AppTypeKea.String() {
			newKeaAppsCnt++
		} else if app.Type == dbmodel.AppTypeBind9.String() {
			newBind9AppsCnt++
		}
	}

	// new and old apps
	allApps := []*dbmodel.App{}

	// old apps found in new apps fetched from the machine
	matchedApps := []*dbmodel.App{}
	for _, app := range discoveredApps {
		// try to match apps on machine with old apps from database
		var dbApp *dbmodel.App
		for _, dbAppOld := range oldAppsList {
			// If there is one app of a given type detected on the machine and one app recorded in the database
			// we assume that this is the same app. If there are more apps of a given type than used to be,
			// or there are less apps than it used to be we have to compare their access control information
			// to identify matching ones.
			if (app.Type == dbmodel.AppTypeKea.String() && dbAppOld.Type.IsKea() && oldKeaAppsCnt == 1 && newKeaAppsCnt == 1) ||
				(app.Type == dbmodel.AppTypeBind9.String() && dbAppOld.Type.IsBind9() && oldBind9AppsCnt == 1 && newBind9AppsCnt == 1) ||
				appCompare(dbAppOld, app) {
				dbApp = dbAppOld
				matchedApps = append(matchedApps, dbApp)
				break
			}
		}
		// if no old app in db then prepare new record
		if dbApp == nil {
			dbApp = &dbmodel.App{
				ID:        0,
				MachineID: dbMachine.ID,
				Machine:   dbMachine,
				Type:      dbmodel.AppType(app.Type),
			}
		} else {
			dbApp.Machine = dbMachine
		}
		allApps = append(allApps, dbApp)

		// add or update access points
		var accessPoints []*dbmodel.AccessPoint
		for _, point := range app.AccessPoints {
			accessPoints = append(accessPoints, &dbmodel.AccessPoint{
				Type:              point.Type,
				Address:           point.Address,
				Port:              point.Port,
				Key:               point.Key,
				UseSecureProtocol: point.UseSecureProtocol,
			})
		}
		dbApp.AccessPoints = accessPoints
	}

	// add old, not matched apps to all apps
	for _, dbApp := range oldAppsList {
		toAdd := true
		for _, app := range matchedApps {
			if dbApp == app {
				toAdd = false
				break
			}
		}
		if toAdd {
			dbApp.Machine = dbMachine
			allApps = append(allApps, dbApp)
		}
	}

	return allApps, ""
}

// Retrieve remotely machine and its apps state, and store it in the database.
func UpdateMachineAndAppsState(ctx context.Context, db *dbops.PgDB, dbMachine *dbmodel.Machine, agents agentcomm.ConnectedAgents, eventCenter eventcenter.EventCenter, reviewDispatcher configreview.Dispatcher, lookup keaconfig.DHCPOptionDefinitionLookup) string {
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// get state of machine from agent
	state, err := agents.GetState(ctx2, dbMachine)
	if err != nil {
		log.Warn(err)
		dbMachine.Error = "Cannot get state of machine"
		err = dbmodel.UpdateMachine(db, dbMachine)
		if err != nil {
			log.Error(err)
			return "Problem updating record in database"
		}
		return ""
	}

	// The Stork server doesn't gather the Stork agent configuration, so we cannot
	// detect its change. It used to compare the current agent state and the database
	// entry to merely recognise the HTTP credentials state change but this
	// parameter has been removed from the agent state. The following variable is
	// a placeholder for the possible future implementation of the Stork agent
	// configuration change detection.
	isStorkAgentChanged := false

	// store machine's state in db
	err = updateMachineFields(db, dbMachine, state)
	if err != nil {
		log.Error(err)
		return "Cannot update machine in db"
	}

	// take old apps from db and new apps fetched from the machine
	// and match them and prepare a list of all apps
	allApps, errStr := mergeNewAndOldApps(db, dbMachine, state.Apps)
	if errStr != "" {
		return errStr
	}

	// go through all apps and store their changes in database
	for _, dbApp := range allApps {
		// get app state from the machine
		switch dbApp.Type {
		case dbmodel.AppTypeKea:
			state := kea.GetAppState(ctx2, agents, dbApp, eventCenter)
			err = kea.CommitAppIntoDB(db, dbApp, eventCenter, state, lookup)
			if err == nil {
				// Let's now identify new daemons or the daemons with updated
				// configurations and schedule configuration reviews for them
				conditionallyBeginKeaConfigReviews(dbApp, state, reviewDispatcher, isStorkAgentChanged)
			}
		case dbmodel.AppTypeBind9:
			bind9.GetAppState(ctx2, agents, dbApp, eventCenter)
			err = bind9.CommitAppIntoDB(db, dbApp, eventCenter)
		default:
			err = nil
		}

		if err != nil {
			log.Errorf("Cannot store application state: %+v", err)
			return "Problem storing application state in the database"
		}
	}

	// add all apps to machine's apps list - it will be used in ReST API functions
	// to return state of machine and its apps
	dbMachine.Apps = allApps

	return ""
}

// This function iterates over the app's daemons and checks if a new config
// review should be performed. It is performed when daemon's configuration
// or dispatcher's signature has changed.
func conditionallyBeginKeaConfigReviews(dbApp *dbmodel.App, state *kea.AppStateMeta, reviewDispatcher configreview.Dispatcher, storkAgentConfigChanged bool) {
	for i, daemon := range dbApp.Daemons {
		// Let's make sure that the config pointer is set. It can be nil
		// when the daemon is inactive.
		if daemon.KeaDaemon == nil || daemon.KeaDaemon.Config == nil {
			continue
		}

		var triggers configreview.Triggers
		if storkAgentConfigChanged {
			triggers = append(triggers, configreview.StorkAgentConfigModified)
		}

		isConfigModified := true
		if state != nil && state.SameConfigDaemons != nil {
			if isSame, ok := state.SameConfigDaemons[daemon.Name]; ok && isSame {
				if daemon.ConfigReview != nil &&
					daemon.ConfigReview.Signature == reviewDispatcher.GetSignature() {
					// Configuration of this daemon hasn't changed and the dispatcher has
					// no checkers modified since the last review. Skip the config modified trigger.
					isConfigModified = false
				}
			}
		}
		if isConfigModified {
			triggers = append(triggers, configreview.ConfigModified)
		}

		if len(triggers) != 0 {
			_ = reviewDispatcher.BeginReview(dbApp.Daemons[i], triggers, nil)
		}
	}
}
