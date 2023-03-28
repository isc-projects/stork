package apps

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	kea "isc.org/stork/server/apps/kea"
	"isc.org/stork/server/configreview"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
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
	_, err := db.Model(&setting).Insert()
	require.NoError(t, err)

	// Fake agents, event center and config review dispatcher.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}

	sp, err := NewStatePuller(db, fa, fec, fd, dbmodel.NewDHCPOptionDefinitionLookup())
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
				Type: datamodel.AppTypeKea.String(),
				// access point is changing from 1.1.1.1 to 1.2.3.4
				AccessPoints: agentcomm.MakeAccessPoint(dbmodel.AccessPointControl, "1.2.3.4", "", 1234),
			},
			{
				Type:         datamodel.AppTypeBind9.String(),
				AccessPoints: agentcomm.MakeAccessPoint(dbmodel.AccessPointControl, "1.2.3.4", "abcd", 124),
			},
		},
	}

	// prepare fake event center
	fec := &storktest.FakeEventCenter{}

	// fake config review dispatcher
	fd := &storktest.FakeDispatcher{}

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

	config, err := dbmodel.NewKeaConfigFromJSON(`{"Dhcp4": { }}`)
	require.NoError(t, err)

	var ap []*dbmodel.AccessPoint
	a := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.AppTypeKea,
		Active:    true,
		// initial access point is 1.1.1.1
		AccessPoints: dbmodel.AppendAccessPoint(ap, dbmodel.AccessPointControl, "1.1.1.1", "", 1234, false),
		Daemons: []*dbmodel.Daemon{
			{
				Active: true,
				Name:   "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config:        config,
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
	_, err = db.Model(&setting).Insert()
	require.NoError(t, err)

	// prepare stats puller
	sp, err := NewStatePuller(db, fa, fec, fd, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)
	// shutdown state puller at the end
	defer sp.Shutdown()

	// invoke pulling state
	err = sp.pullData()
	require.NoError(t, err)

	// check if apps have been updated correctly
	apps, err := dbmodel.GetAllApps(db, true)
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

	// Ensure that the puller initiated configuration review for the Kea daemon.
	require.Len(t, fd.CallLog, 1)
	require.Equal(t, "BeginReview", fd.CallLog[0].CallName)
}

// Check appCompare.
func TestAppCompare(t *testing.T) {
	// no access points so not equal
	dbApp := &dbmodel.App{}
	app := &agentcomm.App{}
	require.False(t, appCompare(dbApp, app))

	// access point only in dbApp so not equal
	var ap []*dbmodel.AccessPoint
	dbApp.AccessPoints = dbmodel.AppendAccessPoint(ap, dbmodel.AccessPointControl, "1.1.1.1", "", 1234, true)
	require.False(t, appCompare(dbApp, app))

	// the same access points so equal
	app.AccessPoints = agentcomm.MakeAccessPoint(dbmodel.AccessPointControl, "1.2.3.4", "abcd", 1234)
	require.True(t, appCompare(dbApp, app))

	// different ports so not equal
	dbApp.AccessPoints[0].Port = 4321
	require.False(t, appCompare(dbApp, app))
}

// Test that new configuration review is scheduled when a daemon's
// configuration has changed or when review dispatcher's checkers
// have changed.
func TestConditionallyBeginKeaConfigReviews(t *testing.T) {
	config, err := dbmodel.NewKeaConfigFromJSON(`{"Dhcp4": { }}`)
	require.NoError(t, err)

	app := &dbmodel.App{
		Daemons: []*dbmodel.Daemon{
			{
				Name: "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: config,
				},
				ConfigReview: &dbmodel.ConfigReview{
					Signature: "",
				},
			},
		},
	}

	state := &kea.AppStateMeta{
		SameConfigDaemons: make(map[string]bool),
	}

	dispatcher := &storktest.FakeDispatcher{}

	// New daemon. The review should be initiated.
	conditionallyBeginKeaConfigReviews(app, state, dispatcher, false)
	require.Len(t, dispatcher.CallLog, 1)
	require.Equal(t, "BeginReview", dispatcher.CallLog[0].CallName)

	// There are no "same daemons". The review should be
	// performed again.
	conditionallyBeginKeaConfigReviews(app, state, dispatcher, false)
	require.Len(t, dispatcher.CallLog, 2)
	require.Equal(t, "BeginReview", dispatcher.CallLog[1].CallName)

	// Neither daemon's configuration nor dispatcher's signature
	// have changed. The review should not be performed.
	state.SameConfigDaemons["dhcp4"] = true
	conditionallyBeginKeaConfigReviews(app, state, dispatcher, false)
	require.Len(t, dispatcher.CallLog, 3)
	require.Equal(t, "GetSignature", dispatcher.CallLog[2].CallName)

	// Modify the dispatcher's signature. It should result in
	// another config review.
	dispatcher.Signature = "new signature"
	conditionallyBeginKeaConfigReviews(app, state, dispatcher, false)
	require.Len(t, dispatcher.CallLog, 5)
	require.Equal(t, "GetSignature", dispatcher.CallLog[3].CallName)
	require.Equal(t, "BeginReview", dispatcher.CallLog[4].CallName)

	// Stork Agent configuration changed. The review should be performed again.
	conditionallyBeginKeaConfigReviews(app, state, dispatcher, true)
	require.Len(t, dispatcher.CallLog, 7)
	require.Equal(t, "GetSignature", dispatcher.CallLog[5].CallName)
	require.Equal(t, "BeginReview", dispatcher.CallLog[6].CallName)
	require.Len(t, dispatcher.CallLog[6].Triggers, 2)
	require.Equal(t, configreview.StorkAgentConfigModified, dispatcher.CallLog[6].Triggers[0])
	require.Equal(t, configreview.ConfigModified, dispatcher.CallLog[6].Triggers[1])
}
