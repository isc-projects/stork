package kea

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Kea servers' response to config-get command from CA. The argument indicates if
// it is a response from a single server or two servers.
func mockGetConfigFromCAResponse(daemons int, cmdResponses []interface{}) {
	list1 := cmdResponses[0].(*[]VersionGetResponse)
	*list1 = []VersionGetResponse{
		{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Daemon: "ca",
			},
			Arguments: &VersionGetRespArgs{
				Extended: "Extended version",
			},
		},
	}
	list2 := cmdResponses[1].(*[]keactrl.HashedResponse)
	*list2 = []keactrl.HashedResponse{
		{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Daemon: "ca",
			},
		},
	}
	if daemons > 1 {
		(*list2)[0].Arguments = &map[string]interface{}{
			"Control-agent": map[string]interface{}{
				"control-sockets": map[string]interface{}{
					"dhcp4": map[string]interface{}{
						"socket-name": "aaa",
						"socket-type": "unix",
					},
					"dhcp6": map[string]interface{}{
						"socket-name": "bbbb",
						"socket-type": "unix",
					},
				},
			},
		}
		(*list2)[0].ArgumentsHash = "hash1"
	} else {
		(*list2)[0].Arguments = &map[string]interface{}{
			"Control-agent": map[string]interface{}{
				"control-sockets": map[string]interface{}{
					"dhcp4": map[string]interface{}{
						"socket-name": "aaa",
						"socket-type": "unix",
					},
				},
				"loggers": []interface{}{
					map[string]interface{}{
						"name":     "kea-ca",
						"severity": "DEBUG",
						"output_options": []interface{}{
							map[string]interface{}{
								"output": "stdout",
							},
						},
					},
					map[string]interface{}{
						"name":     "kea-ca.sockets",
						"severity": "DEBUG",
						"output_options": []interface{}{
							map[string]interface{}{
								"output": "/tmp/kea-ca-sockets.log",
							},
						},
					},
				},
			},
		}
		(*list2)[0].ArgumentsHash = "hash2"
	}
}

// Kea servers' response to config-get command from other Kea daemons. The argument indicates if
// it is a response from a single server or two servers.
func mockGetConfigFromOtherDaemonsResponse(daemons int, cmdResponses []interface{}) {
	// version-get response
	list1 := cmdResponses[0].(*[]VersionGetResponse)
	*list1 = []VersionGetResponse{
		{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Daemon: "dhcp4",
			},
			Arguments: &VersionGetRespArgs{
				Extended: "Extended version",
			},
		},
	}
	if daemons > 1 {
		*list1 = append(*list1, VersionGetResponse{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Daemon: "dhcp6",
			},
			Arguments: &VersionGetRespArgs{
				Extended: "Extended version",
			},
		})
	}
	// status-get response
	list2 := cmdResponses[1].(*[]StatusGetResponse)
	*list2 = []StatusGetResponse{
		{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Daemon: "dhcp4",
			},
			Arguments: &StatusGetRespArgs{
				Pid: 123,
			},
		},
	}
	if daemons > 1 {
		*list2 = append(*list2, StatusGetResponse{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Daemon: "dhcp6",
			},
			Arguments: &StatusGetRespArgs{
				Pid: 123,
			},
		})
	}
	// config-get response
	list3 := cmdResponses[2].(*[]keactrl.HashedResponse)
	*list3 = []keactrl.HashedResponse{
		{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Daemon: "dhcp4",
			},
			Arguments: &map[string]interface{}{
				"Dhcp4": map[string]interface{}{
					"hooks-libraries": []interface{}{
						map[string]interface{}{
							"library": "hook_abc.so",
						},
						map[string]interface{}{
							"library": "hook_def.so",
						},
					},
				},
			},
		},
	}
	(*list3)[0].ArgumentsHash = "hash1"
	if daemons > 1 {
		*list3 = append(*list3, keactrl.HashedResponse{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Daemon: "dhcp6",
			},
			Arguments: &map[string]interface{}{
				"Dhcp6": map[string]interface{}{
					"hooks-libraries": []interface{}{
						map[string]interface{}{
							"library": "hook_abc.so",
						},
						map[string]interface{}{
							"library": "hook_def.so",
						},
					},
				},
			},
		})
		(*list3)[1].ArgumentsHash = "hash2"
	}
}

// Check if GetAppState returns response to the forwarded command.
func TestGetAppStateWith1Daemon(t *testing.T) {
	ctx := context.Background()

	// check getting config of 1 daemon
	keaMock := func(callNo int, cmdResponses []interface{}) {
		if callNo == 0 {
			mockGetConfigFromCAResponse(1, cmdResponses)
		} else if callNo == 1 {
			mockGetConfigFromOtherDaemonsResponse(1, cmdResponses)
		}
	}
	fa := agentcommtest.NewFakeAgents(keaMock, nil)
	fec := &storktest.FakeEventCenter{}

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "192.0.2.0", "", 1234, true)

	dbApp := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	GetAppState(ctx, fa, &dbApp, fec)

	require.Contains(t, fa.RecordedURLs, "https://192.0.2.0:1234/")
	require.Equal(t, keactrl.VersionGet, fa.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.ConfigGet, fa.RecordedCommands[1].GetCommand())
}

func TestGetAppStateWith2Daemons(t *testing.T) {
	ctx := context.Background()

	// check getting configs of 2 daemons
	keaMock := func(callNo int, cmdResponses []interface{}) {
		if callNo == 0 {
			mockGetConfigFromCAResponse(2, cmdResponses)
		} else if callNo == 1 {
			mockGetConfigFromOtherDaemonsResponse(2, cmdResponses)
		}
	}
	fa := agentcommtest.NewFakeAgents(keaMock, nil)
	fec := &storktest.FakeEventCenter{}

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "192.0.2.0", "", 1234, false)

	dbApp := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	GetAppState(ctx, fa, &dbApp, fec)

	require.Contains(t, fa.RecordedURLs, "http://192.0.2.0:1234/")
	require.Equal(t, keactrl.VersionGet, fa.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.ConfigGet, fa.RecordedCommands[1].GetCommand())
}

// Check GetAppState when app already exists.
func TestGetAppStateForExistingApp(t *testing.T) {
	ctx := context.Background()

	// check getting config of 1 daemon
	keaMock := func(callNo int, cmdResponses []interface{}) {
		if callNo%2 == 0 {
			mockGetConfigFromCAResponse(1, cmdResponses)
		} else if callNo%2 == 1 {
			mockGetConfigFromOtherDaemonsResponse(1, cmdResponses)
		}
	}
	fa := agentcommtest.NewFakeAgents(keaMock, nil)
	fec := &storktest.FakeEventCenter{}

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "192.0.2.0", "", 1234, true)

	dbApp := dbmodel.App{
		ID:           1,
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
		Daemons: []*dbmodel.Daemon{
			{
				Name:      "dhcp4",
				Active:    false,
				KeaDaemon: &dbmodel.KeaDaemon{},
			},
			{
				Name:      "ca",
				Active:    false,
				KeaDaemon: &dbmodel.KeaDaemon{},
			},
		},
	}

	err := dbApp.Daemons[0].SetConfigFromJSON(`{"Dhcp4": {}}`)
	require.NoError(t, err)
	err = dbApp.Daemons[1].SetConfigFromJSON(`{
        "Control-agent": {
            "loggers": [
                {
                    "name": "kea-ca",
                    "severity": "debug",
                    "output_options": [
                        {
                            "output": "stdout"
                        }
                    ]
                }
            ]
        }
    }`)
	require.NoError(t, err)
	dbApp.Daemons[1].LogTargets[0].ID = 1

	// Remember current config hash for daemons.
	dhcp4Hash := dbApp.Daemons[0].KeaDaemon.ConfigHash
	caHash := dbApp.Daemons[1].KeaDaemon.ConfigHash

	state := GetAppState(ctx, fa, &dbApp, fec)
	require.NotNil(t, state)
	require.Empty(t, state.SameConfigDaemons)

	require.Contains(t, fa.RecordedURLs, "https://192.0.2.0:1234/")
	require.Equal(t, keactrl.VersionGet, fa.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.ConfigGet, fa.RecordedCommands[1].GetCommand())

	require.Len(t, dbApp.Daemons, 2)

	var (
		dhcp4Daemon *dbmodel.Daemon
		caDaemon    *dbmodel.Daemon
	)
	for i := range dbApp.Daemons {
		// We successfully communicated with the daemons so they should
		// be in active state.
		require.True(t, dbApp.Daemons[i].Active)
		if dbApp.Daemons[i].Name == "ca" {
			caDaemon = dbApp.Daemons[i]
		} else if dbApp.Daemons[i].Name == "dhcp4" {
			dhcp4Daemon = dbApp.Daemons[i]
		}
	}

	// Make sure that logging information is populated correctly.
	require.NotNil(t, caDaemon)
	require.Len(t, caDaemon.LogTargets, 2)
	for _, target := range caDaemon.LogTargets {
		if target.Name == "kea-ca" {
			// The CA daemon should have an updated log target information, but the
			// ID should not change as a result of the update.
			require.EqualValues(t, 1, target.ID)
			require.Equal(t, "debug", target.Severity)
			require.Equal(t, "stdout", target.Output)
		} else {
			// This is new logging target. Its ID should be 0.
			require.Zero(t, target.ID)
			require.Equal(t, "debug", target.Severity)
			require.Equal(t, "/tmp/kea-ca-sockets.log", target.Output)
		}
	}

	// Make sure that the config hashes have changed.
	require.NotEmpty(t, dhcp4Daemon.KeaDaemon.ConfigHash)
	require.NotEqual(t, dhcp4Daemon.KeaDaemon.ConfigHash, dhcp4Hash)
	require.NotEmpty(t, caDaemon.KeaDaemon.ConfigHash)
	require.NotEqual(t, caDaemon.KeaDaemon.ConfigHash, caHash)

	dhcp4Config := dhcp4Daemon.KeaDaemon.Config
	caConfig := caDaemon.KeaDaemon.Config

	state = GetAppState(ctx, fa, &dbApp, fec)
	require.NotNil(t, state)
	require.Contains(t, state.SameConfigDaemons, "ca")
	require.Contains(t, state.SameConfigDaemons, "dhcp4")

	require.NotNil(t, dhcp4Daemon.KeaDaemon.Config)
	require.Same(t, dhcp4Config, dhcp4Daemon.KeaDaemon.Config)
	require.NotNil(t, caDaemon.KeaDaemon.Config)
	require.Same(t, caConfig, caDaemon.KeaDaemon.Config)
}

// Check if GetDaemonHooks returns hooks for given daemon.
func TestGetDaemonHooksFrom1Daemon(t *testing.T) {
	dbDaemon := &dbmodel.Daemon{
		Name: "dhcp4",
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: dbmodel.NewKeaConfig(&map[string]interface{}{
				"Dhcp4": map[string]interface{}{
					"hooks-libraries": []interface{}{
						map[string]interface{}{
							"library": "hook_abc.so",
						},
						map[string]interface{}{
							"library": "hook_def.so",
						},
					},
				},
			}),
		},
	}

	hooks := GetDaemonHooks(dbDaemon)
	require.Len(t, hooks, 2)
	require.Equal(t, "hook_abc.so", hooks[0])
	require.Equal(t, "hook_def.so", hooks[1])
}

// Tests that Kea can be added and then updated in the database.
func TestCommitAppIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	// add app with particular access point
	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234, false)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    machine.ID,
		Machine:      machine,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
	}

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err = CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	// now change access point (different port) and trigger updating app in database
	accessPoints = []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 2345, true)
	app.AccessPoints = accessPoints
	err = CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	returned, err := dbmodel.GetAppByID(db, app.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Len(t, returned.AccessPoints, 1)
	require.EqualValues(t, 2345, returned.AccessPoints[0].Port)
	require.True(t, returned.AccessPoints[0].UseSecureProtocol)
}
