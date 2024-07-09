package agentcomm

import (
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Structure representing a periodic puller which is configured to
// execute a function specified by a caller according to the timer
// interval specified in the database. The user's function typically
// pulls and manipulates the data from multiple apps.
type PeriodicPuller struct {
	*storkutil.PeriodicExecutor
	intervalSettingName string
	lastInvokedAt       *atomic.Value
	lastFinishedAt      *atomic.Value
	DB                  *dbops.PgDB
	Agents              ConnectedAgents
}

// Creates an instance of a new periodic puller. The periodic puller offers a mechanism
// to periodically trigger an action. This action is supplied as a function instance.
// This function is executed within a goroutine periodically according to the timer
// interval available in the database. The intervalSettingName is a name of this
// setting in the database. The pullerName is used for logging purposes.
func NewPeriodicPuller(db *dbops.PgDB, agents ConnectedAgents, pullerName, intervalSettingName string, pullFunc func() error) (*PeriodicPuller, error) {
	var lastInvokedAt atomic.Value
	var lastFinishedAt atomic.Value
	lastInvokedAt.Store(time.Time{})
	lastFinishedAt.Store(time.Time{})

	periodicExecutor, err := storkutil.NewPeriodicExecutor(
		pullerName,
		func() error {
			lastInvokedAt.Store(time.Now())
			err := pullFunc()
			lastFinishedAt.Store(time.Now())
			return err
		},
		func() (time.Duration, error) {
			interval, err := dbmodel.GetSettingInt(db, intervalSettingName)
			return time.Duration(interval) * time.Second,
				errors.WithMessagef(err,
					"Problem getting interval setting %s from db",
					intervalSettingName)
		},
	)
	if err != nil {
		return nil, err
	}

	periodicPuller := &PeriodicPuller{
		PeriodicExecutor:    periodicExecutor,
		intervalSettingName: intervalSettingName,
		lastInvokedAt:       &lastInvokedAt,
		lastFinishedAt:      &lastFinishedAt,
		DB:                  db,
		Agents:              agents,
	}

	return periodicPuller, nil
}

// Returns the interval setting name used by the puller.
func (p *PeriodicPuller) GetIntervalSettingName() string {
	return p.intervalSettingName
}

// Return time when the last execution finished.
func (p *PeriodicPuller) GetLastFinishedAt() time.Time {
	return p.lastFinishedAt.Load().(time.Time)
}

// Return time when the last execution started.
func (p *PeriodicPuller) GetLastInvokedAt() time.Time {
	return p.lastInvokedAt.Load().(time.Time)
}
