package daemons

import (
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

//go:generate mockgen -package=daemons -destination=dispatchermock_test.go isc.org/stork/server/configreview Dispatcher

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

// Creates Kea mock returning an error response for old access point and a
// successful response for new access point.
// It is aware of the order of calls.
func createKeaMockForAccessPointChange(t *testing.T) func(int, agentcomm.ControlledDaemon, []any) {
	return func(i int, daemon agentcomm.ControlledDaemon, responses []any) {
		accessPoint, err := daemon.GetAccessPoint(dbmodel.AccessPointControl)
		require.NoError(t, err)

		switch {
		case daemon.GetName() == daemonname.DHCPv4 && accessPoint.Address == "203.0.113.111":
			// The DHCPv4 daemon with an old access point is offline, so the
			// puller should not be able to retrieve its state.
			versionResponse := responses[0].(*kea.VersionGetResponse)
			versionResponse.Result = keactrl.ResponseError
			versionResponse.Text = "server is likely to be offline"

			response := responses[1].(*keactrl.Response)
			response.Result = keactrl.ResponseError
			response.Text = "server is likely to be offline"

			statusResponse := responses[2].(*kea.StatusGetResponse)
			statusResponse.Result = keactrl.ResponseError
			statusResponse.Text = "server is likely to be offline"
		case daemon.GetName() == daemonname.DHCPv4 && accessPoint.Address == "203.0.113.123":
			// The DHCPv4 daemon with a new access point is online, so the
			// puller should be able to retrieve its state.
			r := responses[1].(*keactrl.Response)
			r.Arguments = []byte(`{ "Dhcp4": {} }`)
		case daemon.GetName() == daemonname.CA:
			// The CA daemon is online, so the puller should be able to
			// retrieve its state.
			r := responses[1].(*keactrl.Response)
			r.Arguments = []byte(`{ "Control-agent": {} }`)
		default:
			require.FailNow(t, "unexpected call to fake agents")
		}
	}
}

// Check if puller correctly pulls data.
func TestStatePullerPullData(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	fa := agentcommtest.NewFakeAgents(createKeaMockForAccessPointChange(t), nil)
	fa.MachineState = &agentcomm.State{
		AgentVersion: "2.4.0",
		Daemons: []*agentcomm.Daemon{
			{
				Name: daemonname.DHCPv4,
				// access point is changing from 203.0.113.111 to 203.0.113.123
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "203.0.113.123",
					Port:    1234,
				}},
			},
			{
				Name: daemonname.Bind9,
				AccessPoints: []dbmodel.AccessPoint{
					{
						Type:    dbmodel.AccessPointControl,
						Address: "203.0.113.123",
						Port:    124,
						Key:     "abcd",
					},
					{
						Type:    dbmodel.AccessPointStatistics,
						Address: "203.0.113.123",
						Port:    5678,
					},
				},
			},
			{
				Name: daemonname.PDNS,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "203.0.113.123",
					Port:    134,
					Key:     "abcd",
				}},
			},
			{
				Name: daemonname.CA,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "203.0.113.111",
					Port:    5678,
				}},
			},
		},
	}

	// prepare fake event center
	fec := &storktest.FakeEventCenter{}

	// Ensure that the puller initiated configuration review for the Kea daemons.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	fd := NewMockDispatcher(ctrl)

	fd.EXPECT().BeginReview(gomock.Cond(func(daemon *dbmodel.Daemon) bool {
		return daemon.Name == daemonname.CA
	}), gomock.Any(), gomock.Any())
	fd.EXPECT().BeginReview(gomock.Cond(func(daemon *dbmodel.Daemon) bool {
		return daemon.Name == daemonname.DHCPv4 && daemon.AccessPoints[0].Address == "203.0.113.111"
	}), gomock.Any(), gomock.Any())
	fd.EXPECT().BeginReview(gomock.Cond(func(daemon *dbmodel.Daemon) bool {
		return daemon.Name == daemonname.DHCPv4 && daemon.AccessPoints[0].Address == "203.0.113.123"
	}), gomock.Any(), gomock.Any())

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
		Address: "203.0.113.111",
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
	require.Len(t, daemons, 5)

	var keaDaemons []*dbmodel.Daemon
	for _, daemon := range daemons {
		if daemon.Name == daemonname.DHCPv4 {
			keaDaemons = append(keaDaemons, &daemon)
		}
	}
	sort.Slice(keaDaemons, func(i, j int) bool {
		return keaDaemons[i].ID < keaDaemons[j].ID
	})

	require.Len(t, keaDaemons, 2)
	// The daemon with access point before change. It's no longer active but
	// should still be in the database.
	require.Len(t, keaDaemons[0].AccessPoints, 1)
	require.EqualValues(t, keaDaemons[0].AccessPoints[0].Address, "203.0.113.111")
	require.False(t, keaDaemons[0].Active)
	// The daemon with updated access point.
	require.Len(t, keaDaemons[1].AccessPoints, 1)
	require.EqualValues(t, keaDaemons[1].AccessPoints[0].Address, "203.0.113.123")
	require.True(t, keaDaemons[1].Active)
}

// Check if puller correctly pulls data from an agent that can communicate only
// with the Kea CA and cannot connect directly to the daemons.
func TestStatePullerPullDataFromLegacyAgent(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	fa := agentcommtest.NewFakeAgents(func(callNo int, daemon agentcomm.ControlledDaemon, response []any) {
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
		case 1:
			// Call to old Kea DHCP to retrieve its state.
			versionResponse := response[0].(*kea.VersionGetResponse)
			versionResponse.Result = keactrl.ResponseError
			versionResponse.Text = "server is likely to be offline"
			configGetResponse := response[1].(*keactrl.Response)
			configGetResponse.Result = keactrl.ResponseError
			configGetResponse.Text = "server is likely to be offline"
		case 2:
			// Call to old Kea CA to retrieve its state.
			versionResponse := response[0].(*kea.VersionGetResponse)
			versionResponse.Result = keactrl.ResponseError
			versionResponse.Text = "server is likely to be offline"
			configGetResponse := response[1].(*keactrl.Response)
			configGetResponse.Result = keactrl.ResponseError
			configGetResponse.Text = "server is likely to be offline"
		case 3:
			// Call to Kea CA to retrieve its state.
			versionResponse := response[0].(*kea.VersionGetResponse)
			versionResponse.Result = keactrl.ResponseSuccess
			versionResponse.Text = "2.4.0"
			configGetResponse := response[1].(*keactrl.Response)
			configGetResponse.Result = keactrl.ResponseSuccess
			configGetResponse.Arguments = []byte(`{ "Control-agent": {} }`)
		case 4:
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
						Address: "203.0.113.123",
						Port:    124,
						Key:     "abcd",
					},
					{
						Type:    dbmodel.AccessPointStatistics,
						Address: "203.0.113.123",
						Port:    5678,
					},
				},
			},
			{
				Name: daemonname.PDNS,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "203.0.113.123",
					Port:    134,
					Key:     "abcd",
				}},
			},
			{
				Name: daemonname.CA,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "203.0.113.123",
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
		Address: "203.0.113.111",
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
		Address: "203.0.113.111",
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
	require.Len(t, daemons, 6)
	sort.Slice(daemons, func(i, j int) bool {
		return daemons[i].ID < daemons[j].ID
	})

	// Check the detected daemons.
	require.Equal(t, daemons[0].Name, daemonname.DHCPv4)
	require.Equal(t, daemons[0].AccessPoints[0].Address, "203.0.113.111")
	require.Equal(t, daemons[1].Name, daemonname.CA)
	require.Equal(t, daemons[1].AccessPoints[0].Address, "203.0.113.111")
	require.Equal(t, daemons[2].Name, daemonname.CA)
	require.Equal(t, daemons[2].AccessPoints[0].Address, "203.0.113.123")
	require.Equal(t, daemons[3].Name, daemonname.DHCPv4)
	require.Equal(t, daemons[3].AccessPoints[0].Address, "203.0.113.123")
	require.Equal(t, daemons[4].Name, daemonname.Bind9)
	require.Equal(t, daemons[5].Name, daemonname.PDNS)
}

// Check if puller correctly recognizes a modification of access points in
// an existing daemon and updates them without creating duplicates.
func TestStatePullerPullDataModifyAccessPoints(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fa.MachineState = &agentcomm.State{
		AgentVersion: "2.4.0",
		Daemons: []*agentcomm.Daemon{
			{
				Name: daemonname.DHCPv4,
				// new control access point is added
				AccessPoints: []dbmodel.AccessPoint{
					{
						Type:     dbmodel.AccessPointControl,
						Address:  "/var/run/kea/kea4-ctrl-socket",
						Protocol: protocoltype.Socket,
					},
					{
						Type:     dbmodel.AccessPointControl,
						Address:  "203.0.113.111",
						Port:     1234,
						Protocol: protocoltype.HTTP,
					},
				},
			},
			{
				Name: daemonname.Bind9,
				// the control endpoint is disabled
				AccessPoints: []dbmodel.AccessPoint{
					{
						Type:    dbmodel.AccessPointStatistics,
						Address: "203.0.113.123",
						Port:    356,
					},
				},
			},
			{
				// the key is set
				Name: daemonname.PDNS,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "203.0.113.123",
					Port:    134,
					Key:     "abcd",
				}},
			},
			{
				// the protocol is promoted to HTTPS
				Name: daemonname.CA,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:     dbmodel.AccessPointControl,
					Address:  "203.0.113.111",
					Port:     5678,
					Protocol: protocoltype.HTTPS,
				}},
			},
		},
	}

	// prepare fake event center
	fec := &storktest.FakeEventCenter{}

	// Ensure that the puller initiated configuration review for the Kea daemons.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	fd := NewMockDispatcher(ctrl)

	fd.EXPECT().BeginReview(gomock.Cond(func(daemon *dbmodel.Daemon) bool {
		return daemon.Name == daemonname.CA
	}), gomock.Any(), gomock.Any())
	fd.EXPECT().BeginReview(gomock.Cond(func(daemon *dbmodel.Daemon) bool {
		return daemon.Name == daemonname.DHCPv4
	}), gomock.Any(), gomock.Any())

	// add one machine with all daemons
	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	daemonDHCPv4 := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "/var/run/kea/kea4-ctrl-socket",
		Protocol: protocoltype.Socket,
	}})
	err = daemonDHCPv4.SetKeaConfigFromJSON([]byte(`{"Dhcp4": { }}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemonDHCPv4)
	require.NoError(t, err)
	require.NotEqual(t, 0, daemonDHCPv4.ID)

	daemonCA := dbmodel.NewDaemon(m, daemonname.CA, true, []*dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "203.0.113.111",
		Port:     5678,
		Protocol: protocoltype.HTTP,
	}})
	err = daemonCA.SetKeaConfigFromJSON([]byte(`{"Control-agent": { }}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemonCA)
	require.NoError(t, err)
	require.NotEqual(t, 0, daemonCA.ID)

	daemonNamed := dbmodel.NewDaemon(m, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "203.0.113.123",
			Port:    124,
			Key:     "abcd",
		},
		{
			Type:    dbmodel.AccessPointStatistics,
			Address: "203.0.113.123",
			Port:    356,
		},
	})
	err = dbmodel.AddDaemon(db, daemonNamed)
	require.NoError(t, err)
	require.NotEqual(t, 0, daemonNamed.ID)

	daemonPDNS := dbmodel.NewDaemon(m, daemonname.PDNS, true, []*dbmodel.AccessPoint{{
		Type:    dbmodel.AccessPointControl,
		Address: "203.0.113.123",
		Port:    134,
	}})
	err = dbmodel.AddDaemon(db, daemonPDNS)
	require.NoError(t, err)
	require.NotEqual(t, 0, daemonPDNS.ID)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "state_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "0",
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
	daemonIndex := make(map[daemonname.Name]dbmodel.Daemon)
	for _, d := range daemons {
		daemonIndex[d.Name] = d
	}

	// Check the detected daemons.
	daemonDHCPv4New := daemonIndex[daemonname.DHCPv4]
	require.Equal(t, daemonDHCPv4.ID, daemonDHCPv4New.ID)
	require.Len(t, daemonDHCPv4New.AccessPoints, 2)
	require.Equal(t, "/var/run/kea/kea4-ctrl-socket", daemonDHCPv4New.AccessPoints[0].Address)
	require.Equal(t, protocoltype.Socket, daemonDHCPv4New.AccessPoints[0].Protocol)
	// Old access point is preserved.
	require.Equal(t, daemonDHCPv4.AccessPoints[0].ID, daemonDHCPv4New.AccessPoints[0].ID)
	// New access point has beend added.
	require.Equal(t, "203.0.113.111", daemonDHCPv4New.AccessPoints[1].Address)
	require.Equal(t, protocoltype.HTTP, daemonDHCPv4New.AccessPoints[1].Protocol)

	daemonCANew := daemonIndex[daemonname.CA]
	require.Equal(t, daemonCA.ID, daemonCANew.ID)
	require.Len(t, daemonCANew.AccessPoints, 1)
	require.Equal(t, "203.0.113.111", daemonCANew.AccessPoints[0].Address)
	// The protocol has been promoted to HTTPS but the access point has the same ID.
	require.Equal(t, protocoltype.HTTPS, daemonCANew.AccessPoints[0].Protocol)
	require.Equal(t, daemonCA.AccessPoints[0].ID, daemonCANew.AccessPoints[0].ID)

	daemonNamedNew := daemonIndex[daemonname.Bind9]
	require.Equal(t, daemonNamed.ID, daemonNamedNew.ID)
	// One access point has been removed.
	require.Len(t, daemonNamedNew.AccessPoints, 1)
	require.Equal(t, "203.0.113.123", daemonNamedNew.AccessPoints[0].Address)
	require.Equal(t, dbmodel.AccessPointStatistics, daemonNamedNew.AccessPoints[0].Type)

	daemonPDNSNew := daemonIndex[daemonname.PDNS]
	require.Equal(t, daemonPDNS.ID, daemonPDNSNew.ID)
	require.Len(t, daemonPDNSNew.AccessPoints, 1)
	// The key has been set but the access point hhas the same ID.
	require.Equal(t, "abcd", daemonPDNSNew.AccessPoints[0].Key)
	require.Equal(t, daemonPDNS.AccessPoints[0].ID, daemonPDNSNew.AccessPoints[0].ID)
}

// Check daemonCompare.
func TestDaemonCompare(t *testing.T) {
	// no access points so equal
	dbDaemon := &dbmodel.Daemon{}
	grpcDaemon := &agentcomm.Daemon{}
	require.True(t, daemonCompare(dbDaemon, grpcDaemon))

	// access point only in dbDaemon so not equal
	dbDaemon.AccessPoints = []*dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "203.0.113.111",
		Port:     1234,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}}
	require.False(t, daemonCompare(dbDaemon, grpcDaemon))

	// the same access points so equal
	grpcDaemon.AccessPoints = []dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "203.0.113.111",
		Port:     1234,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}}
	require.True(t, daemonCompare(dbDaemon, grpcDaemon))

	// different ports so not equal
	dbDaemon.AccessPoints[0].Port = 4321
	require.False(t, daemonCompare(dbDaemon, grpcDaemon))

	// same second access point added to both daemons so equal
	dbDaemon.AccessPoints = append(dbDaemon.AccessPoints, &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "203.0.113.111",
		Port:     5678,
		Protocol: protocoltype.HTTP,
	})
	grpcDaemon.AccessPoints = append(grpcDaemon.AccessPoints, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "203.0.113.111",
		Port:     5678,
		Protocol: protocoltype.HTTP,
	})
	require.True(t, daemonCompare(dbDaemon, grpcDaemon))

	// the protocol has been promoted to HTTPS but the daemons are still
	// considered equal
	grpcDaemon.AccessPoints[1].Protocol = protocoltype.HTTPS
	require.True(t, daemonCompare(dbDaemon, grpcDaemon))

	// new access point added to grpcDaemon but not to dbDaemon but they still
	// have a common access point so they are considered equal
	grpcDaemon.AccessPoints = append(grpcDaemon.AccessPoints, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "203.0.113.125",
		Port:     6789,
		Protocol: protocoltype.HTTP,
	})
	require.True(t, daemonCompare(dbDaemon, grpcDaemon))
}

// Test that the puller updates access points of daemons.
func TestUpdateAccessPoints(t *testing.T) {
	// No access points.
	dbDaemon := &dbmodel.Daemon{}
	grpcDaemon := &agentcomm.Daemon{}
	updateAccessPoints(dbDaemon, grpcDaemon)
	require.Empty(t, dbDaemon.AccessPoints)
	require.Empty(t, grpcDaemon.AccessPoints)

	// Access point only in dbDaemon. It should be removed.
	dbDaemon.AccessPoints = []*dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "203.0.113.111",
		Port:     1234,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}}
	updateAccessPoints(dbDaemon, grpcDaemon)
	require.Empty(t, dbDaemon.AccessPoints)
	require.Empty(t, grpcDaemon.AccessPoints)

	// Access point only in grpcDaemon. It should be added to dbDaemon.
	grpcDaemon.AccessPoints = []dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "203.0.113.111",
		Port:     1234,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}}
	updateAccessPoints(dbDaemon, grpcDaemon)
	require.Len(t, dbDaemon.AccessPoints, 1)
	require.Equal(t, dbDaemon.AccessPoints[0].Type, dbmodel.AccessPointControl)
	require.Equal(t, dbDaemon.AccessPoints[0].Address, "203.0.113.111")
	require.EqualValues(t, dbDaemon.AccessPoints[0].Port, 1234)
	require.Equal(t, dbDaemon.AccessPoints[0].Key, "abcd")
	require.Equal(t, dbDaemon.AccessPoints[0].Protocol, protocoltype.HTTPS)
	require.Len(t, grpcDaemon.AccessPoints, 1)

	// Access point in both daemons but with different parameters. It should be
	// updated in dbDaemon.
	dbDaemon.AccessPoints[0].ID = 42
	grpcDaemon.AccessPoints[0].Protocol = protocoltype.HTTP
	grpcDaemon.AccessPoints[0].Key = "foo"
	updateAccessPoints(dbDaemon, grpcDaemon)
	require.Len(t, dbDaemon.AccessPoints, 1)
	require.EqualValues(t, 42, dbDaemon.AccessPoints[0].ID)
	require.EqualValues(t, dbDaemon.AccessPoints[0].Port, 1234)
	require.Equal(t, dbDaemon.AccessPoints[0].Key, "foo")
	require.Equal(t, dbDaemon.AccessPoints[0].Protocol, protocoltype.HTTP)

	// Access point in both daemons but with different ports. The port should
	// recreated.
	grpcDaemon.AccessPoints[0].Port = 4321
	updateAccessPoints(dbDaemon, grpcDaemon)
	require.Len(t, dbDaemon.AccessPoints, 1)
	require.Zero(t, dbDaemon.AccessPoints[0].ID)
	require.EqualValues(t, dbDaemon.AccessPoints[0].Port, 4321)

	// Two access points in grpcDaemon. The second one should be added to dbDaemon.
	dbDaemon.AccessPoints[0].ID = 42
	grpcDaemon.AccessPoints = append(grpcDaemon.AccessPoints, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "203.0.113.124",
		Port:     5678,
		Protocol: protocoltype.HTTPS,
	})
	updateAccessPoints(dbDaemon, grpcDaemon)
	require.Len(t, dbDaemon.AccessPoints, 2)
	require.Equal(t, dbDaemon.AccessPoints[0].Type, dbmodel.AccessPointControl)
	require.Equal(t, dbDaemon.AccessPoints[0].Address, "203.0.113.111")
	require.EqualValues(t, dbDaemon.AccessPoints[0].Port, 4321)
	require.Equal(t, dbDaemon.AccessPoints[0].Protocol, protocoltype.HTTP)
	require.EqualValues(t, 42, dbDaemon.AccessPoints[0].ID)
	require.Equal(t, dbDaemon.AccessPoints[1].Type, dbmodel.AccessPointControl)
	require.Equal(t, dbDaemon.AccessPoints[1].Address, "203.0.113.124")
	require.EqualValues(t, dbDaemon.AccessPoints[1].Port, 5678)
	require.Equal(t, dbDaemon.AccessPoints[1].Protocol, protocoltype.HTTPS)
	require.Zero(t, dbDaemon.AccessPoints[1].ID)
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

// Test that concurrent pulls should not cause data duplication.
func TestStatePullerConcurrentPulls(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	fa := agentcommtest.NewFakeAgents(createKeaMockForAccessPointChange(t), nil)

	fa.MachineState = &agentcomm.State{
		AgentVersion: "2.4.0",
		Daemons: []*agentcomm.Daemon{
			{
				Name: daemonname.DHCPv4,
				// access point is changing from 203.0.113.111 to 203.0.113.123
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "203.0.113.123",
					Port:    1234,
				}},
			},
			{
				Name: daemonname.Bind9,
				AccessPoints: []dbmodel.AccessPoint{
					{
						Type:    dbmodel.AccessPointControl,
						Address: "203.0.113.123",
						Port:    124,
						Key:     "abcd",
					},
					{
						Type:    dbmodel.AccessPointStatistics,
						Address: "203.0.113.123",
						Port:    5678,
					},
				},
			},
			{
				Name: daemonname.PDNS,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "203.0.113.123",
					Port:    134,
					Key:     "abcd",
				}},
			},
			{
				Name: daemonname.CA,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:    dbmodel.AccessPointControl,
					Address: "203.0.113.111",
					Port:    5678,
				}},
			},
		},
	}

	// prepare fake event center
	fec := &storktest.FakeEventCenter{}

	// Ensure that the puller initiated configuration review for the Kea daemons.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	fd := NewMockDispatcher(ctrl)

	fd.EXPECT().BeginReview(gomock.Cond(func(daemon *dbmodel.Daemon) bool {
		return daemon.Name == daemonname.CA
	}), gomock.Any(), gomock.Any()).MinTimes(1)
	fd.EXPECT().BeginReview(gomock.Cond(func(daemon *dbmodel.Daemon) bool {
		return daemon.Name == daemonname.DHCPv4 && daemon.AccessPoints[0].Address == "203.0.113.111"
	}), gomock.Any(), gomock.Any()).MinTimes(1)
	fd.EXPECT().BeginReview(gomock.Cond(func(daemon *dbmodel.Daemon) bool {
		return daemon.Name == daemonname.DHCPv4 && daemon.AccessPoints[0].Address == "203.0.113.123"
	}), gomock.Any(), gomock.Any()).MinTimes(1)

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
		Address: "203.0.113.111",
		Port:    1234,
		Key:     "",
	}})
	err = d.SetKeaConfigFromJSON([]byte(`{"Dhcp4": { }}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, d)
	require.NoError(t, err)
	require.NotEqual(t, 0, d.ID)

	// Re-fetch the machine to get all references set up correctly.
	m, err = dbmodel.GetMachineByID(db, m.ID)
	require.NoError(t, err)

	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	// prepare state puller
	sp, err := NewStatePuller(db, fa, fec, fd, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)
	// shutdown state puller at the end
	defer sp.Shutdown()

	// invoke pulling state concurrently
	// run as a puller iteration
	var wg sync.WaitGroup
	wg.Go(func() {
		err = sp.pullData()
		require.NoError(t, err)
	})
	for range 10 {
		wg.Go(func() {
			machine, err := sp.UpdateMachineAndDaemonsState(t.Context(), m.ID)
			require.NoError(t, err)
			require.NotNil(t, machine)
			require.Len(t, machine.Daemons, 5)
		})
	}
	wg.Wait()

	// check if daemons have been updated correctly
	daemons, err := dbmodel.GetAllDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 5)

	var keaDaemons []*dbmodel.Daemon
	for _, daemon := range daemons {
		if daemon.Name == daemonname.DHCPv4 {
			keaDaemons = append(keaDaemons, &daemon)
		}
	}
	sort.Slice(keaDaemons, func(i, j int) bool {
		return keaDaemons[i].ID < keaDaemons[j].ID
	})

	require.Len(t, keaDaemons, 2)
	// The daemon with access point before change. It's no longer active but
	// should still be in the database.
	require.Len(t, keaDaemons[0].AccessPoints, 1)
	require.EqualValues(t, keaDaemons[0].AccessPoints[0].Address, "203.0.113.111")
	require.False(t, keaDaemons[0].Active)
	// The daemon with updated access point.
	require.Len(t, keaDaemons[1].AccessPoints, 1)
	require.EqualValues(t, keaDaemons[1].AccessPoints[0].Address, "203.0.113.123")
	require.True(t, keaDaemons[1].Active)
}
