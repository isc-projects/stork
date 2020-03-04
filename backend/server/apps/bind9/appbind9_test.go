package bind9

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

func TestGetAppState(t *testing.T) {
	ctx := context.Background()

	dummyFn := func(c int, r []interface{}) {
	}

	fa := storktest.NewFakeAgents(dummyFn)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "127.0.0.1", "abcd", 953)
	dbApp := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	GetAppState(ctx, fa, &dbApp)

	require.Equal(t, "127.0.0.1", fa.RecordedAddress)
	require.EqualValues(t, 953, fa.RecordedPort)
	require.Equal(t, "abcd", fa.RecordedKey)

	require.True(t, dbApp.Active)
	require.Equal(t, dbApp.Meta.Version, "9.9.9")

	daemon := dbApp.Details.(dbmodel.AppBind9).Daemon
	require.True(t, daemon.Active)
	require.Equal(t, "named", daemon.Name)
	require.Equal(t, "9.9.9", daemon.Version)
	reloadedAt, _ := time.Parse(namedLongDateFormat, "Mon, 03 Feb 2020 14:39:36 GMT")
	require.Equal(t, reloadedAt, daemon.ReloadedAt)
	require.EqualValues(t, 5, daemon.ZoneCount)
	require.EqualValues(t, 96, daemon.AutomaticZoneCount)
}

// Tests that BIND 9 can be added and then updated in the database.
func TestCommitAppIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = CommitAppIntoDB(db, app)
	require.NoError(t, err)

	accessPoints = []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 2345)
	app.AccessPoints = accessPoints
	err = CommitAppIntoDB(db, app)
	require.NoError(t, err)

	returned, err := dbmodel.GetAppByID(db, app.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Len(t, returned.AccessPoints, 1)
	require.EqualValues(t, 2345, returned.AccessPoints[0].Port)
}
