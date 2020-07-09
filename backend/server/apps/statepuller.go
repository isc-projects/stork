package apps

import (
	"context"

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

// Gets the status of the Kea apps and stores useful information in the database.
// The High Availability status is stored in the database for those apps which
// have the HA enabled.
func (puller *StatePuller) pullData() (int, error) {
	// get list of all apps from database
	dbApps, err := dbmodel.GetAllApps(puller.Db)
	if err != nil {
		return 0, err
	}

	// get ...TODO
	var lastErr error
	appsOkCnt := 0
	for _, dbApp := range dbApps {
		dbApp2 := dbApp
		err := puller.getAppState(&dbApp2)
		if err != nil {
			lastErr = err
			log.Errorf("error occurred while getting stats from app %d: %+v", dbApp.ID, err)
		} else {
			appsOkCnt++
		}
	}
	log.Printf("completed pulling lease stats from Kea apps: %d/%d succeeded", appsOkCnt, len(dbApps))
	return appsOkCnt, lastErr
}

func (puller *StatePuller) getAppState(dbApp *dbmodel.App) error {
	ctx := context.Background()

	var err error
	switch dbApp.Type {
	case dbmodel.AppTypeKea:
		events := kea.GetAppState(ctx, puller.Agents, dbApp, puller.EventCenter)
		err = kea.CommitAppIntoDB(puller.Db, dbApp, puller.EventCenter, events)
	case dbmodel.AppTypeBind9:
		bind9.GetAppState(ctx, puller.Agents, dbApp, puller.EventCenter)
		err = bind9.CommitAppIntoDB(puller.Db, dbApp, puller.EventCenter)
	default:
		err = nil
	}
	return err
}
