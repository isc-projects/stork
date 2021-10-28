package agentcomm

import (
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
	DB                  *dbops.PgDB
	Agents              ConnectedAgents
}

// Creates an instance of a new periodic puller. The periodic puller offers a mechanism
// to periodically trigger an action. This action is supplied as a function instance.
// This function is executed within a goroutine periodically according to the timer
// interval available in the database. The intervalSettingName is a name of this
// setting in the database. The pullerName is used for logging purposes.
func NewPeriodicPuller(db *dbops.PgDB, agents ConnectedAgents, pullerName, intervalSettingName string, pullFunc func() error) (*PeriodicPuller, error) {
	periodicExecutor, err := storkutil.NewPeriodicExecutor(
		pullerName, pullFunc,
		func() (int64, error) {
			interval, err := dbmodel.GetSettingInt(db, intervalSettingName)
			return interval, errors.WithMessagef(err, "problem with getting interval setting %s from db",
				intervalSettingName)
		},
	)
	if err != nil {
		return nil, err
	}

	periodicPuller := &PeriodicPuller{
		periodicExecutor,
		intervalSettingName,
		db,
		agents,
	}

	return periodicPuller, nil
}
