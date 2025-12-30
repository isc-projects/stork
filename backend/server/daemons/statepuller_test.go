package daemons

import (
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	"isc.org/stork/server/configreview"
	kea "isc.org/stork/server/daemons/kea"
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
		Name:    "state_puller_interval",
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
	fa := agentcommtest.NewFakeAgents(func(i int, response []any) {
		r := response[1].(*keactrl.Response)
		r.Arguments = []byte(`{ "Dhcp4": {} }`)
	}, nil)
	fa.MachineState = &agentcomm.State{
		AgentVersion: "2.4.0",
		Daemons: []*agentcomm.Daemon{
			{
				Name: daemonname.DHCPv4,
				// access point is changing from 1.1.1.1 to 1.2.3.4
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "1.2.3.4",
					Port:    1234,
				}},
			},
			{
				Name: daemonname.Bind9,
				AccessPoints: []dbmodel.AccessPoint{
					{
						Type:    dbmodel.AccessPointControl,
						Address: "1.2.3.4",
						Port:    124,
						Key:     "abcd",
					},
					{
						Type:    dbmodel.AccessPointStatistics,
						Address: "1.2.3.4",
						Port:    5678,
					},
				},
			},
			{
				Name: daemonname.PDNS,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "1.2.3.4",
					Port:    134,
					Key:     "abcd",
				}},
			},
			{
				Name: daemonname.CA,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "1.1.1.1",
					Port:    5678,
				}},
			},
		},
	}

	// prepare fake event center
	fec := &storktest.FakeEventCenter{}

	// fake config review dispatcher
	fd := &storktest.FakeDispatcher{}

	// add one machine with one kea daemon
	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	d := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{{
		Type:    dbmodel.AccessPointControl,
		Address: "1.1.1.1",
		Port:    1234,
		Key:     "",
	}})
	err = d.SetKeaConfigFromJSON([]byte(`{"Dhcp4": { }}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, d)
	require.NoError(t, err)
	require.NotEqual(t, 0, d.ID)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "state_puller_interval",
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

	// check if daemons have been updated correctly
	daemons, err := dbmodel.GetAllDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 4)

	var keaDaemon dbmodel.Daemon
	for _, daemon := range daemons {
		if daemon.Name == daemonname.DHCPv4 {
			keaDaemon = daemon
		}
	}
	require.Len(t, keaDaemon.AccessPoints, 1)
	require.EqualValues(t, keaDaemon.AccessPoints[0].Address, "1.2.3.4")

	// Ensure that the puller initiated configuration review for the Kea daemons.
	require.Len(t, fd.CallLog, 2)
	require.Equal(t, "BeginReview", fd.CallLog[0].CallName)
	require.Equal(t, "BeginReview", fd.CallLog[1].CallName)
}

// Check if puller correctly pulls data from an agent that can communicate only
// with the Kea CA and cannot connect directly to the daemons.
func TestStatePullerPullDataFromLegacyAgent(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	fa := agentcommtest.NewFakeAgents(func(callNo int, response []any) {
		switch callNo {
		case 0:
			// Call to Kea CA to retrieve daemons.
			r := response[0].(*keactrl.Response)
			r.Arguments = []byte(`{ "Control-agent": {
				"control-sockets": {
					"dhcp4": {
						"socket-type": "unix",
						"socket-name": "/var/run/kea/kea4-ctrl-socket"
					}
				}
			} }`)
		case 2:
			// Call to Kea CA to retrieve its state.
			versionResponse := response[0].(*kea.VersionGetResponse)
			versionResponse.Result = keactrl.ResponseSuccess
			versionResponse.Text = "2.4.0"
			configGetResponse := response[1].(*keactrl.Response)
			configGetResponse.Result = keactrl.ResponseSuccess
			configGetResponse.Arguments = []byte(`{ "Control-agent": {} }`)
		case 3:
			// Call to Kea DHCPv4 to retrieve its state.
			versionResponse := response[0].(*kea.VersionGetResponse)
			versionResponse.Result = keactrl.ResponseSuccess
			versionResponse.Text = "2.4.0"
			configGetResponse := response[1].(*keactrl.Response)
			configGetResponse.Result = keactrl.ResponseSuccess
			configGetResponse.Arguments = []byte(`{ "Dhcp4": {} }`)
			statusGetResponse := response[2].(*kea.StatusGetResponse)
			statusGetResponse.Result = keactrl.ResponseSuccess
			statusGetResponse.Arguments = &kea.StatusGetRespArgs{}
		default:
			require.FailNow(t, "unexpected call number to fake agents")
		}
	}, nil)
	fa.MachineState = &agentcomm.State{
		// Legacy agent version.
		AgentVersion: "2.2.0",
		Daemons: []*agentcomm.Daemon{
			{
				Name: daemonname.Bind9,
				AccessPoints: []dbmodel.AccessPoint{
					{
						Type:    dbmodel.AccessPointControl,
						Address: "1.2.3.4",
						Port:    124,
						Key:     "abcd",
					},
					{
						Type:    dbmodel.AccessPointStatistics,
						Address: "1.2.3.4",
						Port:    5678,
					},
				},
			},
			{
				Name: daemonname.PDNS,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "1.2.3.4",
					Port:    134,
					Key:     "abcd",
				}},
			},
			{
				Name: daemonname.CA,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "1.2.3.4",
					Port:    1234,
				}},
			},
		},
	}

	// prepare fake event center
	fec := &storktest.FakeEventCenter{}

	// fake config review dispatcher
	fd := &storktest.FakeDispatcher{}

	// add one machine with one kea daemon
	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// DHCPv4 daemon.
	d := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{{
		Type:    dbmodel.AccessPointControl,
		Address: "1.1.1.1",
		Port:    1234,
		Key:     "",
	}})
	err = d.SetKeaConfigFromJSON([]byte(`{"Dhcp4": { }}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, d)
	require.NoError(t, err)
	require.NotEqual(t, 0, d.ID)

	// CA daemon.
	d = dbmodel.NewDaemon(m, daemonname.CA, true, []*dbmodel.AccessPoint{{
		Type:    dbmodel.AccessPointControl,
		Address: "1.1.1.1",
		Port:    1234,
		Key:     "",
	}})
	err = d.SetKeaConfigFromJSON([]byte(`{"Control-agent": { }}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, d)
	require.NoError(t, err)
	require.NotEqual(t, 0, d.ID)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "state_puller_interval",
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

	// check if daemons have been updated correctly
	daemons, err := dbmodel.GetAllDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 4)

	// Check the detected daemons.
	var daemonNames []daemonname.Name
	for _, daemon := range daemons {
		daemonNames = append(daemonNames, daemon.Name)
	}
	require.Contains(t, daemonNames, daemonname.DHCPv4)
	require.Contains(t, daemonNames, daemonname.CA)
	require.Contains(t, daemonNames, daemonname.Bind9)
	require.Contains(t, daemonNames, daemonname.PDNS)

	for _, daemon := range daemons {
		if daemon.Name.IsKea() {
			require.Len(t, daemon.AccessPoints, 1)
			require.EqualValues(t, daemon.AccessPoints[0].Address, "1.2.3.4")
		}
	}

	// Ensure that the puller initiated configuration review for the Kea daemon.
	require.Len(t, fd.CallLog, 2)
	require.Equal(t, "BeginReview", fd.CallLog[0].CallName)
	require.Equal(t, "BeginReview", fd.CallLog[0].CallName)
}

// Check daemonCompare.
func TestDaemonCompare(t *testing.T) {
	// no access points so equal
	var dbDaemon dbmodel.Daemon
	daemon := &agentcomm.Daemon{}
	require.True(t, daemonCompare(dbDaemon, daemon))

	// access point only in dbDaemon so not equal
	dbDaemon.AccessPoints = []*dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "1.1.1.1",
		Port:     1234,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}}
	require.False(t, daemonCompare(dbDaemon, daemon))

	// the same access points so equal
	daemon.AccessPoints = []dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "1.1.1.1",
		Port:     1234,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}}
	require.True(t, daemonCompare(dbDaemon, daemon))

	// different ports so not equal
	dbDaemon.AccessPoints[0].Port = 4321
	require.False(t, daemonCompare(dbDaemon, daemon))
}

// Test that new configuration review is scheduled when a daemon's
// configuration has changed or when review dispatcher's checkers
// have changed.
func TestConditionallyBeginKeaConfigReviews(t *testing.T) {
	daemon := dbmodel.NewDaemon(&dbmodel.Machine{}, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{})
	err := daemon.SetKeaConfigFromJSON([]byte(`{"Dhcp4": { }}`))
	require.NoError(t, err)
	state := kea.DaemonStateMeta{IsConfigChanged: true}

	dispatcher := &storktest.FakeDispatcher{}

	// New daemon. The review should be initiated.
	conditionallyBeginKeaConfigReviews(daemon, state, dispatcher, false)
	require.Len(t, dispatcher.CallLog, 1)
	require.Equal(t, "BeginReview", dispatcher.CallLog[0].CallName)
	daemon.ConfigReview = &dbmodel.ConfigReview{}

	// IsConfigChanged is still true. The review should be performed again.
	conditionallyBeginKeaConfigReviews(daemon, state, dispatcher, false)
	require.Len(t, dispatcher.CallLog, 2)
	require.Equal(t, "BeginReview", dispatcher.CallLog[1].CallName)

	// Neither daemon's configuration nor dispatcher's signature
	// have changed. The review should not be performed.
	state.IsConfigChanged = false
	conditionallyBeginKeaConfigReviews(daemon, state, dispatcher, false)
	require.Len(t, dispatcher.CallLog, 3)
	require.Equal(t, "GetSignature", dispatcher.CallLog[2].CallName)

	// Modify the dispatcher's signature. It should result in
	// another config review.
	dispatcher.Signature = "new signature"
	conditionallyBeginKeaConfigReviews(daemon, state, dispatcher, false)
	require.Len(t, dispatcher.CallLog, 5)
	require.Equal(t, "GetSignature", dispatcher.CallLog[3].CallName)
	require.Equal(t, "BeginReview", dispatcher.CallLog[4].CallName)

	// Stork Agent configuration changed. The review should be performed again.
	conditionallyBeginKeaConfigReviews(daemon, state, dispatcher, true)
	require.Len(t, dispatcher.CallLog, 7)
	require.Equal(t, "GetSignature", dispatcher.CallLog[5].CallName)
	require.Equal(t, "BeginReview", dispatcher.CallLog[6].CallName)
	require.Len(t, dispatcher.CallLog[6].Triggers, 2)
	require.Equal(t, configreview.StorkAgentConfigModified, dispatcher.CallLog[6].Triggers[0])
	require.Equal(t, configreview.ConfigModified, dispatcher.CallLog[6].Triggers[1])
}
