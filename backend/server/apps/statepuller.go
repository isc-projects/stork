package apps

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps/bind9"
	"isc.org/stork/server/apps/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Instance of the puller which periodically checks the status of the Kea apps.
// Besides basic status information the High Availability status is fetched.
type StatePuller struct {
	*agentcomm.PeriodicPuller
	EventCenter eventcenter.EventCenter
}

// Create an instance of the puller which periodically checks the status of
// the Kea apps.
func NewStatePuller(db *dbops.PgDB, agents agentcomm.ConnectedAgents, eventCenter eventcenter.EventCenter) (*StatePuller, error) {
	puller := &StatePuller{
		EventCenter: eventCenter,
	}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Apps State",
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
func (puller *StatePuller) pullData() (int, error) {
	// get list of all authorized machines from database
	authorized := true
	dbMachines, err := dbmodel.GetAllMachines(puller.DB, &authorized)
	if err != nil {
		return 0, err
	}

	// get state from machines and their apps
	var lastErr error
	okCnt := 0
	for _, dbM := range dbMachines {
		dbM2 := dbM
		ctx := context.Background()
		errStr := GetMachineAndAppsState(ctx, puller.DB, &dbM2, puller.Agents, puller.EventCenter)
		if errStr != "" {
			lastErr = errors.New(errStr)
			log.Errorf("error occurred while getting info from machine %d: %s", dbM2.ID, errStr)
		} else {
			okCnt++
		}
	}
	log.Printf("completed pulling information from machines: %d/%d succeeded", okCnt, len(dbMachines))
	return okCnt, lastErr
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
	err := db.Update(dbMachine)
	if err != nil {
		return errors.Wrapf(err, "problem with updating machine %+v", dbMachine)
	}
	return nil
}

// appCompare compares two apps for equality.  Two apps are considered equal if
// their type matches and if they have the same control port.  Return true if
// equal, false otherwise.
func appCompare(dbApp *dbmodel.App, app *agentcomm.App) bool {
	if dbApp.Type != app.Type {
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
		return nil, "cannot get machine's apps from db"
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
		if app.Type == dbmodel.AppTypeKea {
			newKeaAppsCnt++
		} else if app.Type == dbmodel.AppTypeBind9 {
			newBind9AppsCnt++
		}
	}

	// new and old apps
	allApps := []*dbmodel.App{}

	// old apps found in new apps fetched from the machine
	matchedApps := []*dbmodel.App{}
	for _, app := range discoveredApps {
		// try to match apps on machine with old apps from database
		var dbApp *dbmodel.App = nil
		for _, dbAppOld := range oldAppsList {
			// If there is one app of a given type detected on the machine and one app recorded in the database
			// we assume that this is the same app. If there are more apps of a given type than used to be,
			// or there are less apps than it used to be we have to compare their access control information
			// to identify matching ones.
			if (app.Type == dbmodel.AppTypeKea && dbAppOld.Type == dbmodel.AppTypeKea && oldKeaAppsCnt == 1 && newKeaAppsCnt == 1) ||
				(app.Type == dbmodel.AppTypeBind9 && dbAppOld.Type == dbmodel.AppTypeBind9 && oldBind9AppsCnt == 1 && newBind9AppsCnt == 1) ||
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
				Type:      app.Type,
			}
		} else {
			dbApp.Machine = dbMachine
		}
		allApps = append(allApps, dbApp)

		// add or update access points
		var accessPoints []*dbmodel.AccessPoint
		for _, point := range app.AccessPoints {
			accessPoints = append(accessPoints, &dbmodel.AccessPoint{
				Type:    point.Type,
				Address: point.Address,
				Port:    point.Port,
				Key:     point.Key,
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
func GetMachineAndAppsState(ctx context.Context, db *dbops.PgDB, dbMachine *dbmodel.Machine, agents agentcomm.ConnectedAgents, eventCenter eventcenter.EventCenter) string {
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// get state of machine from agent
	state, err := agents.GetState(ctx2, dbMachine.Address, dbMachine.AgentPort)
	if err != nil {
		log.Warn(err)
		dbMachine.Error = "cannot get state of machine"
		err = db.Update(dbMachine)
		if err != nil {
			log.Error(err)
			return "problem with updating record in database"
		}
		return ""
	}

	// store machine's state in db
	err = updateMachineFields(db, dbMachine, state)
	if err != nil {
		log.Error(err)
		return "cannot update machine in db"
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
			err = kea.CommitAppIntoDB(db, dbApp, eventCenter, state)
		case dbmodel.AppTypeBind9:
			bind9.GetAppState(ctx2, agents, dbApp, eventCenter)
			err = bind9.CommitAppIntoDB(db, dbApp, eventCenter)
		default:
			err = nil
		}

		if err != nil {
			log.Errorf("cannot store application state: %+v", err)
			return "problem with storing application state in the database"
		}
	}

	// add all apps to machine's apps list - it will be used in ReST API functions
	// to return state of machine and its apps
	dbMachine.Apps = allApps

	return ""
}
