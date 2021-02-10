package apps

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

// Check creating and shutting down StatePuller.
func TestStatsPullerBasic(t *testing.T) {
	// prepare db
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "apps_state_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err := db.Insert(&setting)
	require.NoError(t, err)

	// prepare fake agents and event center
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}

	sp, err := NewStatePuller(db, fa, fec)
	require.NoError(t, err)
	require.NotNil(t, sp.PeriodicPuller)

	sp.Shutdown()
}

// Check if puller correctly pulls data.
func TestStatePullerPullData(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fa.MachineState = &agentcomm.State{
		Apps: []*agentcomm.App{
			{
				Type: dbmodel.AppTypeKea,
				// access point is changing from 1.1.1.1 to 1.2.3.4
				AccessPoints: agentcomm.MakeAccessPoint(dbmodel.AccessPointControl, "1.2.3.4", "", 1234),
			},
			{
				Type:         dbmodel.AppTypeBind9,
				AccessPoints: agentcomm.MakeAccessPoint(dbmodel.AccessPointControl, "1.2.3.4", "abcd", 124),
			},
		},
	}

	// prepare fake event center
	fec := &storktest.FakeEventCenter{}

	// add one machine with one kea app
	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	var ap []*dbmodel.AccessPoint
	a := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.AppTypeKea,
		Active:    true,
		// initial access point is 1.1.1.1
		AccessPoints: dbmodel.AppendAccessPoint(ap, dbmodel.AccessPointControl, "1.1.1.1", "", 1234),
		Daemons: []*dbmodel.Daemon{
			{
				Active: true,
				Name:   "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, a)
	require.NoError(t, err)
	require.NotEqual(t, 0, a.ID)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "apps_state_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err = db.Insert(&setting)
	require.NoError(t, err)

	// prepare stats puller
	sp, err := NewStatePuller(db, fa, fec)
	require.NoError(t, err)
	// shutdown state puller at the end
	defer sp.Shutdown()

	// invoke pulling state
	appsOkCnt, err := sp.pullData()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// check if apps have been updated correctly
	apps, err := dbmodel.GetAllApps(db)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	var keaApp dbmodel.App
	if apps[0].Type == dbmodel.AppTypeKea {
		keaApp = apps[0]
	} else {
		keaApp = apps[1]
	}
	require.Len(t, keaApp.AccessPoints, 1)
	require.EqualValues(t, keaApp.AccessPoints[0].Address, "1.2.3.4")
}

// Check appCompare.
func TestAppCompare(t *testing.T) {
	// no access points so not equal
	dbApp := &dbmodel.App{}
	app := &agentcomm.App{}
	require.False(t, appCompare(dbApp, app))

	// access point only in dbApp so not equal
	var ap []*dbmodel.AccessPoint
	dbApp.AccessPoints = dbmodel.AppendAccessPoint(ap, dbmodel.AccessPointControl, "1.1.1.1", "", 1234)
	require.False(t, appCompare(dbApp, app))

	// the same access points so equal
	app.AccessPoints = agentcomm.MakeAccessPoint(dbmodel.AccessPointControl, "1.2.3.4", "abcd", 1234)
	require.True(t, appCompare(dbApp, app))

	// different ports so not equal
	dbApp.AccessPoints[0].Port = 4321
	require.False(t, appCompare(dbApp, app))
}
