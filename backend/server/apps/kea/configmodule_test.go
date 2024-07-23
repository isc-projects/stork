package kea

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/datamodel"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	appstest "isc.org/stork/server/apps/test"
	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
	storkutil "isc.org/stork/util"
)

// Test config manager. Besides returning database and agents instance
// it also provides additional functions useful in testing.
type testManager struct {
	db     *pg.DB
	agents agentcomm.ConnectedAgents
	lookup keaconfig.DHCPOptionDefinitionLookup

	locks map[int64]bool
}

// Creates new test config manager instance.
func newTestManager(server config.ManagerAccessors) *testManager {
	return &testManager{
		db:     server.GetDB(),
		agents: server.GetConnectedAgents(),
		lookup: server.GetDHCPOptionDefinitionLookup(),
		locks:  make(map[int64]bool),
	}
}

// Returns database instance.
func (tm *testManager) GetDB() *pg.DB {
	return tm.db
}

// Returns an interface to the test agents.
func (tm *testManager) GetConnectedAgents() agentcomm.ConnectedAgents {
	return tm.agents
}

// Returns an interface to the instance providing functions to find
// option definitions.
func (tm *testManager) GetDHCPOptionDefinitionLookup() keaconfig.DHCPOptionDefinitionLookup {
	return tm.lookup
}

// Applies locks on specified daemons.
func (tm *testManager) Lock(ctx context.Context, daemonIDs ...int64) (context.Context, error) {
	for _, id := range daemonIDs {
		tm.locks[id] = true
	}
	return ctx, nil
}

// Removes all locks.
func (tm *testManager) Unlock(ctx context.Context) {
	tm.locks = make(map[int64]bool)
}

// Simulates adding and retrieving a config change from the database. As a result,
// the returned context contains transaction state re-created from the database
// entry. It simulates scheduling config changes.
func (tm *testManager) scheduleAndGetChange(ctx context.Context, t *testing.T) context.Context {
	// User ID is required to schedule a config change.
	userID, ok := config.GetValueAsInt64(ctx, config.UserContextKey)
	require.True(t, ok)

	// The state will be inserted into the database.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)

	// Create the config change entry in the database.
	scc := &dbmodel.ScheduledConfigChange{
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 10),
		UserID:     userID,
	}
	for _, u := range state.Updates {
		update := dbmodel.ConfigUpdate{
			Target:    u.Target,
			Operation: u.Operation,
			DaemonIDs: u.DaemonIDs,
		}
		recipe, err := json.Marshal(u.Recipe)
		require.NoError(t, err)
		update.Recipe = (*json.RawMessage)(&recipe)
		scc.Updates = append(scc.Updates, &update)
	}
	err := dbmodel.AddScheduledConfigChange(tm.db, scc)
	require.NoError(t, err)

	// Get the config change back from the database.
	changes, err := dbmodel.GetDueConfigChanges(tm.db)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	change := changes[0]

	// Override the context state.
	state = config.TransactionState[ConfigRecipe]{
		Scheduled: true,
	}
	for _, u := range change.Updates {
		update := NewConfigUpdateFromDBModel(u)
		state.Updates = append(state.Updates, update)
	}
	ctx = context.WithValue(ctx, config.StateContextKey, state)
	return ctx
}

// Test Kea module commit function.
func TestCommit(t *testing.T) {
	module := NewConfigModule(nil)
	require.NotNil(t, module)

	ctx := context.Background()

	_, err := module.Commit(ctx)
	require.Error(t, err)
}

// Test the first stage of updating global Kea parameters. It checks that the initial
// configuration information is fetched from the database and stored in the context.
// It also checks that appropriate locks are applied.
func TestBeginGlobalParametersUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"valid-lifetime": 3000
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	ctx, err := module.BeginGlobalParametersUpdate(context.Background(), []int64{apps[0].Daemons[0].ID, apps[1].Daemons[0].ID})
	require.NoError(t, err)

	// Make sure that the locks have been applied on the daemons owning
	// the host.
	require.Contains(t, manager.locks, apps[0].Daemons[0].ID)
	require.Contains(t, manager.locks, apps[1].Daemons[0].ID)

	// Make sure that the host information has been stored in the context.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.Len(t, state.Updates, 1)
	require.Equal(t, datamodel.AppTypeKea, state.Updates[0].Target)
	require.Equal(t, "global_parameters_update", state.Updates[0].Operation)

	daemons := state.Updates[0].Recipe.GlobalConfigRecipeParams.KeaDaemonsBeforeConfigUpdate
	require.Len(t, daemons, 2)
	require.EqualValues(t, apps[0].Daemons[0].ID, daemons[0].ID)
	require.EqualValues(t, apps[1].Daemons[0].ID, daemons[1].ID)

	require.Nil(t, state.Updates[0].Recipe.GlobalConfigRecipeParams.KeaDaemonsAfterConfigUpdate)
}

// Test second stage of global parameters update.
func TestApplyGlobalParametersUpdate(t *testing.T) {
	daemonConfig, err := dbmodel.NewKeaConfigFromJSON(`{
		"Dhcp4": {
			"valid-lifetime": 3000
		}
	}`)
	require.NoError(t, err)
	require.NotNil(t, daemonConfig)

	daemons := []dbmodel.Daemon{
		{
			ID:   1,
			Name: dbmodel.DaemonNameDHCPv4,
			KeaDaemon: &dbmodel.KeaDaemon{
				Config: daemonConfig,
			},
			App: &dbmodel.App{
				AccessPoints: []*dbmodel.AccessPoint{
					{
						Type:    dbmodel.AccessPointControl,
						Address: "192.0.2.1",
						Port:    1234,
					},
				},
			},
		},
		{
			ID:   2,
			Name: dbmodel.DaemonNameDHCPv4,
			KeaDaemon: &dbmodel.KeaDaemon{
				Config: daemonConfig,
			},
			App: &dbmodel.App{
				AccessPoints: []*dbmodel.AccessPoint{
					{
						Type:    dbmodel.AccessPointControl,
						Address: "192.0.2.2",
						Port:    2345,
					},
				},
			},
		},
	}

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1, 2}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "global_parameters_update", daemonIDs...)
	recipe := ConfigRecipe{
		GlobalConfigRecipeParams: GlobalConfigRecipeParams{
			KeaDaemonsBeforeConfigUpdate: daemons,
		},
	}
	err = state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Simulate updating configurations.
	config1 := keaconfig.NewSettableDHCPv4Config()
	config2 := keaconfig.NewSettableDHCPv4Config()
	config1.SetValidLifetime(storkutil.Ptr(int64(1111)))
	config2.SetValidLifetime(storkutil.Ptr(int64(1111)))

	ctx, err = module.ApplyGlobalParametersUpdate(ctx, []config.AnnotatedEntity[*keaconfig.SettableConfig]{
		*config.NewAnnotatedEntity(1, config1),
		*config.NewAnnotatedEntity(2, config2),
	})
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	stateReturned, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, stateReturned.Scheduled)

	require.Len(t, stateReturned.Updates, 1)
	update := stateReturned.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "global_parameters_update", update.Operation)
	require.NotNil(t, update.Recipe)
	require.Len(t, update.Recipe.GlobalConfigRecipeParams.KeaDaemonsBeforeConfigUpdate, 2)
	require.Len(t, update.Recipe.GlobalConfigRecipeParams.KeaDaemonsAfterConfigUpdate, 2)

	commands := update.Recipe.Commands
	require.Len(t, commands, 4)

	// Validate the commands to be sent to Kea.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()

		switch {
		case i < 2:
			require.JSONEq(t,
				`{
					"command": "config-set",
					"service": [ "dhcp4" ],
					"arguments": {
						"Dhcp4": {
							"valid-lifetime": 1111
						}
					}
				}`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		}
		// Verify they are associated with appropriate apps.
		require.NotNil(t, commands[i].App)
	}
}

// Test committing global configuration parameters, i.e. actually sending control
// commands to Kea.
func TestCommitGlobalParametersUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app1, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app1, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app2, err := server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app2, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	daemons, err := dbmodel.GetDaemonsByIDs(db, []int64{app1.Daemons[0].GetID(), app2.Daemons[0].GetID()})
	require.NoError(t, err)

	daemonIDs := []int64{daemons[0].GetID(), daemons[1].GetID()}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "global_parameters_update", daemonIDs...)
	recipe := ConfigRecipe{
		GlobalConfigRecipeParams: GlobalConfigRecipeParams{
			KeaDaemonsBeforeConfigUpdate: daemons,
		},
	}
	err = state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Modify the config. The modifications should be applied in
	// the database upon commit.
	modifiedConfig := keaconfig.NewSettableDHCPv4Config()
	modifiedConfig.SetValidLifetime(storkutil.Ptr(int64(1111)))

	ctx, err = module.ApplyGlobalParametersUpdate(ctx, []config.AnnotatedEntity[*keaconfig.SettableConfig]{
		*config.NewAnnotatedEntity(daemons[0].GetID(), modifiedConfig),
		*config.NewAnnotatedEntity(daemons[1].GetID(), modifiedConfig),
	})
	require.NoError(t, err)

	// Committing the changes should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the correct number of commands were sent.
	require.Len(t, agents.RecordedURLs, 4)
	require.Len(t, agents.RecordedCommands, 4)

	// The respective commands should be sent to different servers.
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[1])
	require.NotEqual(t, agents.RecordedURLs[2], agents.RecordedURLs[3])
	require.Equal(t, agents.RecordedURLs[0], agents.RecordedURLs[2])
	require.Equal(t, agents.RecordedURLs[1], agents.RecordedURLs[3])

	// Validate the sent commands and URLS.
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch {
		case i < 2:
			require.JSONEq(t, `{
				"command": "config-set",
				"service": [ "dhcp4" ],
				"arguments": {
					"Dhcp4": {
						"valid-lifetime": 1111
					}
				}
			}`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp4" ]
					}`,
				marshalled)
		}
	}

	// Make sure that the global configurations have been updated in the database.
	updatedDaemons, err := dbmodel.GetDaemonsByIDs(db, []int64{daemons[0].GetID(), daemons[1].GetID()})
	require.NoError(t, err)
	require.Len(t, updatedDaemons, 2)

	// Make sure that the updated configuration has been stored in the database.
	for _, daemon := range updatedDaemons {
		require.NotNil(t, daemon.KeaDaemon)
		config := daemon.KeaDaemon.Config
		require.NotNil(t, config)

		require.NotNil(t, config.GetValidLifetimeParameters().ValidLifetime)
		require.EqualValues(t, 1111, *config.GetValidLifetimeParameters().ValidLifetime)
	}
}

// Test first stage of adding a new host.
func TestBeginHostAdd(t *testing.T) {
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	ctx, err := module.BeginHostAdd(context.Background())
	require.NoError(t, err)

	// There should be no locks on any daemons.
	require.Empty(t, manager.locks)

	// Make sure that the transaction state has been created.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.Len(t, state.Updates, 1)
	require.Equal(t, datamodel.AppTypeKea, state.Updates[0].Target)
	require.Equal(t, "host_add", state.Updates[0].Operation)
}

// Test second stage of adding a new host.
func TestApplyHostAdd(t *testing.T) {
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// Transaction state is required because typically it is created by the
	// BeginHostAdd function.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "host_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Simulate submitting new host entry. The host is associated with
	// two different daemons/apps.
	host := &dbmodel.Host{
		ID: 1,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: 1,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
			},
			{
				DaemonID: 2,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
			},
		},
	}
	ctx, err := module.ApplyHostAdd(ctx, host)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	returnedState, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, returnedState.Scheduled)

	require.Len(t, returnedState.Updates, 1)
	update := state.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "host_add", update.Operation)
	require.NotNil(t, update.Recipe)

	// There should be two commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 2)

	// Validate the first command and associated app.
	command := commands[0].Command
	marshalled := command.Marshal()
	require.JSONEq(t,
		`{
             "command": "reservation-add",
             "service": [ "dhcp4" ],
             "arguments": {
                 "reservation": {
                     "subnet-id": 0,
                     "hw-address": "010203040506",
                     "hostname": "cool.example.org"
                 }
             }
         }`,
		marshalled)

	app := commands[0].App
	require.Equal(t, app, host.LocalHosts[0].Daemon.App)

	// Validate the second command and associated app.
	command = commands[1].Command
	marshalled = command.Marshal()
	require.JSONEq(t,
		`{
             "command": "reservation-add",
             "service": [ "dhcp4" ],
             "arguments": {
                 "reservation": {
                     "subnet-id": 0,
                     "hw-address": "010203040506",
                     "hostname": "cool.example.org"
                 }
             }
         }`,
		marshalled)

	app = commands[1].App
	require.Equal(t, app, host.LocalHosts[1].Daemon.App)
}

// Test committing added host, i.e. actually sending control commands to Kea.
func TestCommitHostAdd(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, apps := storktest.AddTestHosts(t, db)

	// Create the config manager instance "connected to" fake agents.
	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	// Create Kea config module.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// Transaction state is required because typically it is created by the
	// BeginHostAdd function.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "host_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Create new host reservation and store it in the context.
	host := &dbmodel.Host{
		ID: 1001,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: apps[0].Daemons[0].KeaDaemon.DaemonID,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
			},
			{
				DaemonID: apps[1].Daemons[0].KeaDaemon.DaemonID,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
			},
		},
	}
	ctx, err := module.ApplyHostAdd(ctx, host)
	require.NoError(t, err)

	// Committing the host should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the commands were sent to appropriate servers.
	require.Len(t, agents.RecordedURLs, 2)
	require.Equal(t, "http://192.0.2.1:1234/", agents.RecordedURLs[0])
	require.Equal(t, "http://192.0.2.2:2345/", agents.RecordedURLs[1])

	// Validate the sent commands.
	require.Len(t, agents.RecordedCommands, 2)
	for _, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		require.JSONEq(t,
			`{
                 "command": "reservation-add",
                 "service": [ "dhcp4" ],
                 "arguments": {
                     "reservation": {
                         "subnet-id": 0,
                         "hw-address": "010203040506",
                         "hostname": "cool.example.org"
                     }
                 }
             }`,
			marshalled)
	}

	// Make sure that the host has been added to the database too.
	newHost, err := dbmodel.GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, newHost)
	require.Len(t, newHost.LocalHosts, 2)
}

// Test that error is returned when Kea response contains error status code.
func TestCommitHostAddResponseWithErrorStatus(t *testing.T) {
	// Create the config manager instance "connected to" fake agents.
	agents := agentcommtest.NewKeaFakeAgents(func(callNo int, cmdResponses []interface{}) {
		json := []byte(`[
            {
                "result": 1,
                "text": "error is error"
            }
        ]`)
		command := keactrl.NewCommandBase(keactrl.ReservationAdd, keactrl.DHCPv4)
		_ = keactrl.UnmarshalResponseList(command, json, cmdResponses[0])
	})

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	// Create Kea config module.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// Transaction state is required because typically it is created by the
	// BeginHostAdd function.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "host_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Create new host reservation and store it in the context.
	host := &dbmodel.Host{
		ID: 1,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: 1,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
						Name: "kea@192.0.2.1",
					},
				},
			},
			{
				DaemonID: 2,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
						Name: "kea@192.0.2.2",
					},
				},
			},
		},
	}
	ctx, err := module.ApplyHostAdd(ctx, host)
	require.NoError(t, err)

	_, err = module.Commit(ctx)
	require.ErrorContains(t, err, "reservation-add command to kea@192.0.2.1 failed: error status (1) returned by Kea dhcp4 daemon with text: 'error is error'")

	// The second command should not be sent in this case.
	require.Len(t, agents.RecordedCommands, 1)
}

// Test scheduling config changes in the database, retrieving and committing them.
func TestCommitScheduledHostAdd(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, apps := storktest.AddTestHosts(t, db)

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// User is required to associate the config change with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// Transaction state is required because typically it is created by the
	// BeginHostAdd function.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "host_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Set user id in the context.
	ctx = context.WithValue(ctx, config.UserContextKey, int64(user.ID))

	// Create the host and store it in the context.
	host := &dbmodel.Host{
		ID: 1001,
		Subnet: &dbmodel.Subnet{
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					DaemonID:      1,
					LocalSubnetID: 123,
				},
			},
		},
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: apps[0].Daemons[0].KeaDaemon.DaemonID,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
			},
		},
	}
	ctx, err = module.ApplyHostAdd(ctx, host)
	require.NoError(t, err)

	// Simulate scheduling the config change and retrieving it from the database.
	// The context will hold re-created transaction state.
	ctx = manager.scheduleAndGetChange(ctx, t)
	require.NotNil(t, ctx)

	// Try to send the command to Kea server.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure it was sent to appropriate server.
	require.Len(t, agents.RecordedURLs, 1)
	require.Equal(t, "http://192.0.2.1:1234/", agents.RecordedURLs[0])

	// Ensure the command has appropriate structure.
	require.Len(t, agents.RecordedCommands, 1)
	command := agents.RecordedCommands[0]
	marshalled := command.Marshal()
	require.JSONEq(t,
		`{
             "command": "reservation-add",
             "service": [ "dhcp4" ],
             "arguments": {
                 "reservation": {
                     "subnet-id": 123,
                     "hw-address": "010203040506",
                     "hostname": "cool.example.org"
                 }
             }
         }`,
		marshalled)

	// Make sure that the host has been added to the database.
	newHost, err := dbmodel.GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, newHost)
}

// Test the first stage of updating a host. It checks that the host information
// is fetched from the database and stored in the context. It also checks that
// appropriate locks are applied.
func TestBeginHostUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	hosts, apps := storktest.AddTestHosts(t, db)

	ctx, err := module.BeginHostUpdate(context.Background(), hosts[0].ID)
	require.NoError(t, err)

	// Make sure that the locks have been applied on the daemons owning
	// the host.
	require.Contains(t, manager.locks, apps[0].Daemons[0].ID)
	require.Contains(t, manager.locks, apps[1].Daemons[0].ID)

	// Make sure that the host information has been stored in the context.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.Len(t, state.Updates, 1)
	require.Equal(t, datamodel.AppTypeKea, state.Updates[0].Target)
	require.Equal(t, "host_update", state.Updates[0].Operation)
	require.NotNil(t, state.Updates[0].Recipe.HostBeforeUpdate)
}

// Test second stage of a host update.
func TestApplyHostUpdate(t *testing.T) {
	// Create dummy host to be stored in the context. We will later check if
	// it is preserved after applying host update.
	hasher := keaconfig.NewHasher()
	host := &dbmodel.Host{
		ID: 1,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: 1,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
				DHCPOptionSet: dbmodel.NewDHCPOptionSet([]dbmodel.DHCPOption{{
					Code: 1,
				}}, hasher),
			},
			{
				DaemonID: 2,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
				DHCPOptionSet: dbmodel.NewDHCPOptionSet([]dbmodel.DHCPOption{{
					Code: 2,
				}}, hasher),
			},
			{
				DaemonID: 2,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceConfig,
				DHCPOptionSet: dbmodel.NewDHCPOptionSet([]dbmodel.DHCPOption{{
					Code: 3,
				}}, hasher),
			},
		},
	}

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "host_update", daemonIDs...)
	recipe := ConfigRecipe{
		HostConfigRecipeParams: HostConfigRecipeParams{
			HostBeforeUpdate: host,
		},
	}
	err := state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Simulate updating host entry. We change host identifier and hostname.
	host = &dbmodel.Host{
		ID: 1,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{2, 3, 4, 5, 6, 7},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: 1,
				Hostname: "foo.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
				DHCPOptionSet: dbmodel.NewDHCPOptionSet([]dbmodel.DHCPOption{{
					Code: 4,
				}}, hasher),
			},
			{
				DaemonID: 2,
				Hostname: "foo.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
				DHCPOptionSet: dbmodel.NewDHCPOptionSet([]dbmodel.DHCPOption{{
					Code: 4,
				}}, hasher),
			},
		},
	}
	ctx, err = module.ApplyHostUpdate(ctx, host)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	stateReturned, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, stateReturned.Scheduled)

	// Verify the host after update.
	recipeReturned, err := stateReturned.GetRecipeForUpdate(0)
	require.NoError(t, err)
	require.NotNil(t, recipeReturned)
	require.EqualValues(t, 1, recipeReturned.HostAfterUpdate.ID)
	require.Len(t, recipeReturned.HostAfterUpdate.LocalHosts, 3)

	require.EqualValues(t, 1, recipeReturned.HostAfterUpdate.LocalHosts[0].DaemonID)
	require.EqualValues(t, dbmodel.HostDataSourceAPI, recipeReturned.HostAfterUpdate.LocalHosts[0].DataSource)
	require.Len(t, recipeReturned.HostAfterUpdate.LocalHosts[0].DHCPOptionSet.Options, 1)
	require.EqualValues(t, 4, recipeReturned.HostAfterUpdate.LocalHosts[0].DHCPOptionSet.Options[0].Code)

	require.EqualValues(t, 2, recipeReturned.HostAfterUpdate.LocalHosts[1].DaemonID)
	require.EqualValues(t, dbmodel.HostDataSourceAPI, recipeReturned.HostAfterUpdate.LocalHosts[1].DataSource)
	require.Len(t, recipeReturned.HostAfterUpdate.LocalHosts[1].DHCPOptionSet.Options, 1)
	require.EqualValues(t, 4, recipeReturned.HostAfterUpdate.LocalHosts[1].DHCPOptionSet.Options[0].Code)

	require.EqualValues(t, 2, recipeReturned.HostAfterUpdate.LocalHosts[2].DaemonID)
	require.EqualValues(t, dbmodel.HostDataSourceConfig, recipeReturned.HostAfterUpdate.LocalHosts[2].DataSource)
	require.Len(t, recipeReturned.HostAfterUpdate.LocalHosts[2].DHCPOptionSet.Options, 1)
	require.EqualValues(t, 3, recipeReturned.HostAfterUpdate.LocalHosts[2].DHCPOptionSet.Options[0].Code)

	require.Len(t, stateReturned.Updates, 1)
	update := stateReturned.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "host_update", update.Operation)
	require.NotNil(t, update.Recipe)
	require.NotNil(t, update.Recipe.HostBeforeUpdate)

	// There should be four commands ready to send. Two reservation-del and two
	// reservation-add.
	commands := update.Recipe.Commands
	require.Len(t, commands, 4)

	// Validate the commands to be sent to Kea.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()

		// First are the reservation-del commands sent to respective servers.
		// Other are reservation-add commands.
		switch {
		case i < 2:
			require.JSONEq(t,
				`{
                     "command": "reservation-del",
                     "service": [ "dhcp4" ],
                     "arguments": {
                         "subnet-id": 0,
                         "identifier-type": "hw-address",
                         "identifier": "010203040506"
                     }
                 }`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
                     "command": "reservation-add",
                     "service": [ "dhcp4" ],
                     "arguments": {
                         "reservation": {
                             "subnet-id": 0,
                             "hw-address": "020304050607",
                             "hostname": "foo.example.org",
                             "option-data": [{
                                "code": 4,
                                "csv-format": false
                             }]
                         }
                     }
                 }`,
				marshalled)
		}
		// Verify they are associated with appropriate apps.
		app := commands[i].App
		require.Equal(t, app, host.LocalHosts[i%2].Daemon.App)
	}
}

// Test committing updated host, i.e. actually sending control commands to Kea.
func TestCommitHostUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, apps := storktest.AddTestHosts(t, db)

	// Create host reservation.
	host := &dbmodel.Host{
		ID: 1001,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: apps[0].Daemons[0].KeaDaemon.DaemonID,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
			},
			{
				DaemonID: apps[1].Daemons[0].KeaDaemon.DaemonID,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
			},
		},
	}

	require.NoError(t, dbmodel.AddHost(db, host))

	// Create the config manager instance "connected to" fake agents.
	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	// Create Kea config module.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "host_update", daemonIDs...)
	recipe := ConfigRecipe{
		HostConfigRecipeParams: HostConfigRecipeParams{
			HostBeforeUpdate: host,
		},
	}
	err := state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Copy the host and modify it. The modifications should be applied in
	// the database upon commit.
	modifiedHost := *host
	modifiedHost.CreatedAt = time.Time{}
	modifiedHost.LocalHosts[0].NextServer = "192.0.2.22"
	modifiedHost.LocalHosts[0].Hostname = "modified.example.org"
	modifiedHost.LocalHosts[1].NextServer = "192.0.2.22"
	modifiedHost.LocalHosts[1].Hostname = "modified.example.org"

	ctx, err = module.ApplyHostUpdate(ctx, &modifiedHost)
	require.NoError(t, err)

	// Committing the host should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the correct number of commands were sent.
	require.Len(t, agents.RecordedURLs, 4)
	require.Len(t, agents.RecordedCommands, 4)

	// Validate the sent commands and URLS.
	for i, command := range agents.RecordedCommands {
		switch i % 2 {
		case 0:
			require.Equal(t, "http://192.0.2.1:1234/", agents.RecordedURLs[i])
		default:
			require.Equal(t, "http://192.0.2.2:2345/", agents.RecordedURLs[i])
		}
		marshalled := command.Marshal()
		// Every event command is reservation-del. Every odd command is reservation-add.
		switch {
		case i < 2:

			require.JSONEq(t,
				`{
                     "command": "reservation-del",
                     "service": [ "dhcp4" ],
                     "arguments": {
                         "subnet-id": 0,
                         "identifier-type": "hw-address",
                         "identifier": "010203040506"
                     }
                 }`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
                     "command": "reservation-add",
                     "service": [ "dhcp4" ],
                     "arguments": {
                         "reservation": {
                             "subnet-id": 0,
                             "hw-address": "010203040506",
                             "hostname": "modified.example.org",
							 "next-server": "192.0.2.22"
                         }
                     }
                 }`,
				marshalled)
		}
	}

	// Make sure that the host has been updated in the database.
	updatedHost, err := dbmodel.GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedHost)
	require.Len(t, updatedHost.LocalHosts, 2)
	require.Equal(t, "192.0.2.22", updatedHost.LocalHosts[0].NextServer)
	require.Equal(t, "modified.example.org", updatedHost.LocalHosts[0].Hostname)
	require.Equal(t, "192.0.2.22", updatedHost.LocalHosts[0].NextServer)
}

// Test that error is returned when Kea response contains error status code.
func TestCommitHostUpdateResponseWithErrorStatus(t *testing.T) {
	// Create new host reservation.
	host := &dbmodel.Host{
		ID: 1,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: 1,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
						Name: "kea@192.0.2.1",
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
			},
		},
	}
	// Create the config manager instance "connected to" fake agents.
	agents := agentcommtest.NewKeaFakeAgents(func(callNo int, cmdResponses []interface{}) {
		json := []byte(`[
            {
                "result": 1,
                "text": "error is error"
            }
        ]`)
		command := keactrl.NewCommandBase(keactrl.ReservationDel, keactrl.DHCPv4)
		_ = keactrl.UnmarshalResponseList(command, json, cmdResponses[0])
	})

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	// Create Kea config module.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "host_update", daemonIDs...)
	recipe := ConfigRecipe{
		HostConfigRecipeParams: HostConfigRecipeParams{
			HostBeforeUpdate: host,
		},
	}
	err := state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	ctx, err = module.ApplyHostUpdate(ctx, host)
	require.NoError(t, err)

	_, err = module.Commit(ctx)
	require.ErrorContains(t, err, "reservation-del command to kea@192.0.2.1 failed: error status (1) returned by Kea dhcp4 daemon with text: 'error is error'")

	// Other commands should not be sent in this case.
	require.Len(t, agents.RecordedCommands, 1)
}

// Test scheduling config changes in the database, retrieving and committing it.
func TestCommitScheduledHostUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, apps := storktest.AddTestHosts(t, db)

	// Create the host.
	host := &dbmodel.Host{
		ID: 1001,
		Subnet: &dbmodel.Subnet{
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					DaemonID:      1,
					LocalSubnetID: 123,
				},
			},
		},
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: apps[0].Daemons[0].KeaDaemon.DaemonID,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
			},
		},
	}
	require.NoError(t, dbmodel.AddHost(db, host))

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// It is required to associate the config change with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// Prepare the context.
	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)
	ctx = context.WithValue(ctx, config.UserContextKey, int64(user.ID))

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "host_update", daemonIDs...)
	recipe := ConfigRecipe{
		HostConfigRecipeParams: HostConfigRecipeParams{
			HostBeforeUpdate: host,
		},
	}
	err = state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Copy the host and modify it. The modifications should be applied in
	// the database upon commit.
	modifiedHost := *host
	modifiedHost.LocalHosts[0].Hostname = "modified.example.org"

	ctx, err = module.ApplyHostUpdate(ctx, &modifiedHost)
	require.NoError(t, err)

	// Simulate scheduling the config change and retrieving it from the database.
	// The context will hold re-created transaction state.
	ctx = manager.scheduleAndGetChange(ctx, t)
	require.NotNil(t, ctx)

	// Try to send the command to Kea server.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure it was sent to appropriate server.
	require.Len(t, agents.RecordedURLs, 2)
	for _, url := range agents.RecordedURLs {
		require.Equal(t, "http://192.0.2.1:1234/", url)
	}

	// Ensure the commands have appropriate structure.
	require.Len(t, agents.RecordedCommands, 2)
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch {
		case i == 0:
			require.JSONEq(t,
				`{
                 "command": "reservation-del",
                     "service": [ "dhcp4" ],
                     "arguments": {
                         "subnet-id": 123,
                         "identifier-type": "hw-address",
                         "identifier": "010203040506"
                     }
                 }`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
                 "command": "reservation-add",
                     "service": [ "dhcp4" ],
                     "arguments": {
                         "reservation": {
                             "subnet-id": 123,
                             "hw-address": "010203040506",
                             "hostname": "modified.example.org"
                         }
                     }
                 }`,
				marshalled)
		}
	}
	// Make sure that the host has been added to the database too.
	updatedHost, err := dbmodel.GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedHost)
	require.Equal(t, updatedHost.LocalHosts[0].Hostname, "modified.example.org")
}

// Test first stage of deleting a host.
func TestBeginHostDelete(t *testing.T) {
	module := NewConfigModule(nil)
	require.NotNil(t, module)

	ctx1 := context.Background()
	ctx2, err := module.BeginHostDelete(ctx1)
	require.NoError(t, err)
	require.Equal(t, ctx1, ctx2)
}

// Test second stage of deleting a host.
func TestApplyHostDelete(t *testing.T) {
	module := NewConfigModule(nil)
	require.NotNil(t, module)

	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	// Simulate submitting new host entry. The host is associated with
	// two different daemons/apps.
	host := &dbmodel.Host{
		ID: 1,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID: 1,
				Hostname: "cool.example.org",
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
			},
			{
				DaemonID: 2,
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				DataSource: dbmodel.HostDataSourceAPI,
			},
		},
	}
	ctx, err := module.ApplyHostDelete(ctx, host)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, state.Scheduled)

	require.Len(t, state.Updates, 1)
	update := state.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "host_delete", update.Operation)
	require.NotNil(t, update.Recipe)

	// There should be two commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 2)

	// Validate the first command and associated app.
	command := commands[0].Command
	marshalled := command.Marshal()
	require.JSONEq(t,
		`{
             "command": "reservation-del",
             "service": [ "dhcp4" ],
             "arguments": {
                 "subnet-id": 0,
                 "identifier-type": "hw-address",
                 "identifier": "010203040506"
             }
         }`,
		marshalled)

	app := commands[0].App
	require.Equal(t, app, host.LocalHosts[0].Daemon.App)

	// Validate the second command and associated app.
	command = commands[1].Command
	marshalled = command.Marshal()
	require.JSONEq(t,
		`{
             "command": "reservation-del",
             "service": [ "dhcp4" ],
             "arguments": {
                 "subnet-id": 0,
                 "identifier-type": "hw-address",
                 "identifier": "010203040506"
             }
         }`,
		marshalled)

	app = commands[1].App
	require.Equal(t, app, host.LocalHosts[1].Daemon.App)
}

// Test committing added host, i.e. actually sending control commands to Kea.
func TestCommitHostDelete(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	hosts, _ := storktest.AddTestHosts(t, db)

	// Create the config manager instance "connected to" fake agents.
	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	// Create Kea config module.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	// Create new host reservation and store it in the context.
	host, err := dbmodel.GetHost(db, hosts[0].ID)
	require.NoError(t, err)
	ctx, err = module.ApplyHostDelete(ctx, host)
	require.NoError(t, err)

	// Committing the host should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the commands were sent to appropriate servers.
	require.Len(t, agents.RecordedURLs, 2)
	require.Equal(t, "https://localhost:1234/", agents.RecordedURLs[0])
	require.Equal(t, "https://localhost:1235/", agents.RecordedURLs[1])

	// Validate the sent commands.
	require.Len(t, agents.RecordedCommands, 2)
	for _, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		require.JSONEq(t,
			`{
                 "command": "reservation-del",
                 "service": [ "dhcp4" ],
                 "arguments": {
                     "subnet-id": 111,
                     "identifier-type": "hw-address",
                     "identifier": "010203040506"
                  }
             }`,
			marshalled)
	}

	returnedHost, err := dbmodel.GetHost(db, host.ID)
	require.NoError(t, err)
	require.Nil(t, returnedHost)
}

// Test scheduling deleting a host reservation, retrieving the scheduled operation
// from the database and performing it.
func TestCommitScheduledHostDelete(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	hosts, _ := storktest.AddTestHosts(t, db)
	dbmodel.DeleteDaemonFromHosts(db, hosts[0].LocalHosts[1].DaemonID, dbmodel.HostDataSourceUnspecified)

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// User is required to associate the config change with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// Prepare the context.
	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)
	ctx = context.WithValue(ctx, config.UserContextKey, int64(user.ID))

	// Create the host and store it in the context.
	host, err := dbmodel.GetHost(db, hosts[0].ID)
	require.NoError(t, err)
	ctx, err = module.ApplyHostDelete(ctx, host)
	require.NoError(t, err)

	// Simulate scheduling the config change and retrieving it from the database.
	// The context will hold re-created transaction state.
	ctx = manager.scheduleAndGetChange(ctx, t)
	require.NotNil(t, ctx)

	// Try to send the command to Kea server.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure it was sent to appropriate server.
	require.Len(t, agents.RecordedURLs, 1)
	require.Equal(t, "https://localhost:1234/", agents.RecordedURLs[0])

	// Ensure the command has appropriate structure.
	require.Len(t, agents.RecordedCommands, 1)
	command := agents.RecordedCommands[0]
	marshalled := command.Marshal()
	require.JSONEq(t,
		`{
             "command": "reservation-del",
             "service": [ "dhcp4" ],
             "arguments": {
                 "subnet-id": 111,
                 "identifier-type": "hw-address",
                 "identifier": "010203040506"
             }
         }`,
		marshalled)

	returnedHost, err := dbmodel.GetHost(db, host.ID)
	require.NoError(t, err)
	require.Nil(t, returnedHost)
}

// Test first stage of adding a shared network.
func TestBeginSharedNetworkAdd(t *testing.T) {
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	ctx, err := module.BeginSharedNetworkAdd(context.Background())
	require.NoError(t, err)

	// There should be no locks on any daemons.
	require.Empty(t, manager.locks)

	// Make sure that the transaction state has been created.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.Len(t, state.Updates, 1)
	require.Equal(t, datamodel.AppTypeKea, state.Updates[0].Target)
	require.Equal(t, "shared_network_add", state.Updates[0].Operation)
}

// Test second stage of adding a shared network.
func TestApplySharedNetworkAdd(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// Transaction state is required because typically it is created by the
	// BeginSharedNetworkAdd function.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "shared_network_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// New shared network entry.
	sharedNetwork := &dbmodel.SharedNetwork{
		Name:   "bar",
		Family: 4,
		LocalSharedNetworks: []*dbmodel.LocalSharedNetwork{
			{
				DaemonID: 1,
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
			},
			{
				DaemonID: 2,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.5.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
			},
			{
				DaemonID: 4,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.6.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
			},
		},
		Subnets: []dbmodel.Subnet{
			{
				ID:     1,
				Prefix: "192.0.2.0/24",
				LocalSubnets: []*dbmodel.LocalSubnet{
					{
						DaemonID: 1,
						Daemon: &dbmodel.Daemon{
							Name: "dhcp4",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.1",
										Port:    1234,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.100",
								UpperBound: "192.0.2.200",
							},
						},
					},
					{
						DaemonID: 2,
						Daemon: &dbmodel.Daemon{
							Name:    "dhcp4",
							Version: "2.5.0",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.2",
										Port:    2345,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.100",
								UpperBound: "192.0.2.200",
							},
						},
					},
					{
						DaemonID: 4,
						Daemon: &dbmodel.Daemon{
							Name:    "dhcp4",
							Version: "2.6.0",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.2",
										Port:    2345,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.100",
								UpperBound: "192.0.2.200",
							},
						},
					},
				},
			},
		},
	}
	ctx, err := module.ApplySharedNetworkAdd(ctx, sharedNetwork)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	stateReturned, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, stateReturned.Scheduled)

	require.Len(t, stateReturned.Updates, 1)
	update := stateReturned.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "shared_network_add", update.Operation)
	require.NotNil(t, update.Recipe)

	// There should be seven commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 7)

	// Validate the commands to be sent to Kea.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()

		switch i {
		case 0, 1, 2:
			require.JSONEq(t,
				`{
					"command": "network4-add",
					"service": ["dhcp4"],
					"arguments": {
						"shared-networks": [
							{
								"name": "bar",
								"subnet4": [
									{
										"pools": [
											{
												"pool":"192.0.2.100-192.0.2.200"
											}
										],
										"id": 0,
										"subnet": "192.0.2.0/24"
									}
								]
							}
						]
					}
				}`,
				marshalled)
		case 3, 4, 6:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		// The default case is executed for the index of 5. The config-reload
		// is only issued for Kea versions earlier than 2.6.0 that don't
		// recount statistics until reloaded.
		default:
			require.JSONEq(t,
				`{
					"command": "config-reload",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		}
		// Verify they are associated with appropriate apps.
		require.NotNil(t, commands[i].App)
	}
}

// Test committing created shared network, i.e. actually sending control commands to Kea.
func TestCommitSharedNetworkAdd(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	// Transaction state is required because typically it is created by the
	// BeginSharedNetworkAdd function.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "shared_network_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// New shared network entry.
	sharedNetwork := &dbmodel.SharedNetwork{
		Name:   "bar",
		Family: 4,
		LocalSharedNetworks: []*dbmodel.LocalSharedNetwork{
			{
				DaemonID: apps[0].Daemons[0].ID,
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
			},
			{
				DaemonID: apps[1].Daemons[0].ID,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.5.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
			},
		},
		Subnets: []dbmodel.Subnet{
			{
				ID:     1,
				Prefix: "192.0.2.0/24",
				LocalSubnets: []*dbmodel.LocalSubnet{
					{
						DaemonID: apps[0].Daemons[0].ID,
						Daemon: &dbmodel.Daemon{
							Name: "dhcp4",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.1",
										Port:    1234,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.100",
								UpperBound: "192.0.2.200",
							},
						},
					},
					{
						DaemonID: apps[1].Daemons[0].ID,
						Daemon: &dbmodel.Daemon{
							Name:    "dhcp4",
							Version: "2.5.0",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.2",
										Port:    2345,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.100",
								UpperBound: "192.0.2.200",
							},
						},
					},
				},
			},
		},
	}
	ctx, err = module.ApplySharedNetworkAdd(ctx, sharedNetwork)
	require.NoError(t, err)

	// Committing the shared network should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the correct number of commands were sent.
	require.Len(t, agents.RecordedURLs, 5)
	require.Len(t, agents.RecordedCommands, 5)

	// The respective commands should be sent to different servers.
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[1])
	require.NotEqual(t, agents.RecordedURLs[2], agents.RecordedURLs[3])
	require.Equal(t, agents.RecordedURLs[3], agents.RecordedURLs[4])

	// Validate the sent commands and URLs.
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch i {
		case 0, 1:
			require.JSONEq(t,
				`{
					"command": "network4-add",
					"service": ["dhcp4"],
					"arguments": {
						"shared-networks": [
							{
								"name": "bar",
								"subnet4": [
									{
										"pools": [
											{
												"pool":"192.0.2.100-192.0.2.200"
											}
										],
										"id": 0,
										"subnet": "192.0.2.0/24"
									}
								]
							}
						]
					}
				}`,
				marshalled)
		case 2, 3:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		// The default case is executed for the index of 4. The config-reload
		// is only issued for Kea versions earlier than 2.6.0 that don't
		// recount statistics until reloaded.
		default:
			require.JSONEq(t,
				`{
					"command": "config-reload",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		}
	}

	// Make sure that the shared network has been added in the database.
	addedSharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, addedSharedNetworks, 1)
	require.NotNil(t, addedSharedNetworks[0])
	require.Len(t, addedSharedNetworks[0].LocalSharedNetworks, 2)
	require.Nil(t, addedSharedNetworks[0].LocalSharedNetworks[0].KeaParameters)
	require.Nil(t, addedSharedNetworks[0].LocalSharedNetworks[1].KeaParameters)

	recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, 0)
	require.NoError(t, err)
	require.NotNil(t, recipe.SharedNetworkID)
	require.EqualValues(t, addedSharedNetworks[0].ID, *recipe.SharedNetworkID)
}

// Test scheduling shared network config changes in the database, retrieving and committing them.
func TestCommitScheduledSharedNetworkAdd(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// User is required to associate the config change with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	serverConfig := `{
		"Dhcp6": {}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	// Transaction state is required because typically it is created by the
	// BeginSharedNetworkAdd function.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "shared_network_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Set user id in the context.
	ctx = context.WithValue(ctx, config.UserContextKey, int64(user.ID))

	// New shared network entry.
	sharedNetwork := &dbmodel.SharedNetwork{
		Name:   "bar",
		Family: 6,
		LocalSharedNetworks: []*dbmodel.LocalSharedNetwork{
			{
				DaemonID: apps[0].Daemons[0].ID,
				Daemon: &dbmodel.Daemon{
					Name: "dhcp6",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
			},
			{
				DaemonID: apps[1].Daemons[0].ID,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp6",
					Version: "2.5.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
			},
		},
	}
	ctx, err = module.ApplySharedNetworkAdd(ctx, sharedNetwork)
	require.NoError(t, err)

	// Simulate scheduling the config change and retrieving it from the database.
	// The context will hold re-created transaction state.
	ctx = manager.scheduleAndGetChange(ctx, t)
	require.NotNil(t, ctx)

	// Committing the subnet should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the correct number of commands were sent.
	require.Len(t, agents.RecordedURLs, 5)
	require.Len(t, agents.RecordedCommands, 5)

	// The respective commands should be sent to different servers.
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[1])
	require.NotEqual(t, agents.RecordedURLs[2], agents.RecordedURLs[3])
	require.Equal(t, agents.RecordedURLs[3], agents.RecordedURLs[4])

	// Validate the sent commands and URLs.
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch i {
		case 0, 1:
			require.JSONEq(t,
				`{
						"command": "network6-add",
						"service": [ "dhcp6" ],
						"arguments": {
							"shared-networks": [
								{
									"name": "bar"
								}
							]
						}
					}`,
				marshalled)
		case 2, 3:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp6" ]
					}`,
				marshalled)
		// The default case is executed for the index of 4. The config-reload
		// is only issued for Kea versions earlier than 2.6.0 that don't
		// recount statistics until reloaded.
		default:
			require.JSONEq(t,
				`{
						"command": "config-reload",
						"service": [ "dhcp6" ]
					}`,
				marshalled)
		}
	}

	// Make sure that the shared network has been added in the database.
	addedSharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, addedSharedNetworks, 1)
	require.NotNil(t, addedSharedNetworks[0])
	require.Len(t, addedSharedNetworks[0].LocalSharedNetworks, 2)
	require.Nil(t, addedSharedNetworks[0].LocalSharedNetworks[0].KeaParameters)
	require.Nil(t, addedSharedNetworks[0].LocalSharedNetworks[1].KeaParameters)

	recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, 0)
	require.NoError(t, err)
	require.NotNil(t, recipe.SharedNetworkID)
	require.EqualValues(t, addedSharedNetworks[0].ID, *recipe.SharedNetworkID)
}

// Test the first stage of updating a shared network. It checks that the shared
// network information is fetched from the database and stored in the context. It
// also checks that appropriate locks are applied.
func TestBeginSharedNetworkUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds.so"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	ctx, err := module.BeginSharedNetworkUpdate(context.Background(), sharedNetworks[0].ID)
	require.NoError(t, err)

	// Make sure that the locks have been applied on the daemons owning
	// the shared network.
	require.Contains(t, manager.locks, apps[0].Daemons[0].ID)
	require.Contains(t, manager.locks, apps[1].Daemons[0].ID)

	// Make sure that the host information has been stored in the context.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.Len(t, state.Updates, 1)
	require.Equal(t, datamodel.AppTypeKea, state.Updates[0].Target)
	require.Equal(t, "shared_network_update", state.Updates[0].Operation)
	require.NotNil(t, state.Updates[0].Recipe.SharedNetworkBeforeUpdate)
	require.Equal(t, "foo", state.Updates[0].Recipe.SharedNetworkBeforeUpdate.Name)
	require.Len(t, state.Updates[0].Recipe.SharedNetworkBeforeUpdate.LocalSharedNetworks, 2)
}

// Test second stage of a shared network update.
func TestApplySharedNetworkUpdate(t *testing.T) {
	// Create dummy shared network to be stored in the context. We will later check if
	// it is preserved after applying shared network update.
	sharedNetwork := &dbmodel.SharedNetwork{
		Name:   "foo",
		Family: 4,
		LocalSharedNetworks: []*dbmodel.LocalSharedNetwork{
			{
				DaemonID: 1,
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
			},
			{
				DaemonID: 2,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.5.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
			},
			{
				DaemonID: 3,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.6.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
			},
		},
		Subnets: []dbmodel.Subnet{
			{
				ID:     1,
				Prefix: "192.0.2.0/24",
				LocalSubnets: []*dbmodel.LocalSubnet{
					{
						DaemonID: 1,
						Daemon: &dbmodel.Daemon{
							Name: "dhcp4",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.1",
										Port:    1234,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.10",
								UpperBound: "192.0.2.100",
							},
						},
					},
					{
						DaemonID: 2,
						Daemon: &dbmodel.Daemon{
							Name:    "dhcp4",
							Version: "2.5.0",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.2",
										Port:    2345,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.10",
								UpperBound: "192.0.2.100",
							},
						},
					},
					{
						DaemonID: 3,
						Daemon: &dbmodel.Daemon{
							Name:    "dhcp4",
							Version: "2.6.0",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.2",
										Port:    2345,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.10",
								UpperBound: "192.0.2.100",
							},
						},
					},
				},
			},
		},
	}

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1, 2}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "shared_network_update", daemonIDs...)
	recipe := ConfigRecipe{
		SharedNetworkConfigRecipeParams: SharedNetworkConfigRecipeParams{
			SharedNetworkBeforeUpdate: sharedNetwork,
		},
	}
	err := state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Simulate updating subnet entry.
	sharedNetwork = &dbmodel.SharedNetwork{
		Name:   "bar",
		Family: 4,
		LocalSharedNetworks: []*dbmodel.LocalSharedNetwork{
			{
				DaemonID: 1,
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
			},
			{
				DaemonID: 2,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.5.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
			},
			{
				DaemonID: 4,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.6.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
			},
		},
		Subnets: []dbmodel.Subnet{
			{
				ID:     1,
				Prefix: "192.0.2.0/24",
				LocalSubnets: []*dbmodel.LocalSubnet{
					{
						DaemonID: 1,
						Daemon: &dbmodel.Daemon{
							Name: "dhcp4",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.1",
										Port:    1234,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.100",
								UpperBound: "192.0.2.200",
							},
						},
					},
					{
						DaemonID: 2,
						Daemon: &dbmodel.Daemon{
							Name:    "dhcp4",
							Version: "2.5.0",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.2",
										Port:    2345,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.100",
								UpperBound: "192.0.2.200",
							},
						},
					},
					{
						DaemonID: 4,
						Daemon: &dbmodel.Daemon{
							Name:    "dhcp4",
							Version: "2.6.0",
							App: &dbmodel.App{
								AccessPoints: []*dbmodel.AccessPoint{
									{
										Type:    dbmodel.AccessPointControl,
										Address: "192.0.2.2",
										Port:    2345,
									},
								},
							},
						},
						AddressPools: []dbmodel.AddressPool{
							{
								LowerBound: "192.0.2.100",
								UpperBound: "192.0.2.200",
							},
						},
					},
				},
			},
		},
	}
	ctx, err = module.ApplySharedNetworkUpdate(ctx, sharedNetwork)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	stateReturned, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, stateReturned.Scheduled)

	require.Len(t, stateReturned.Updates, 1)
	update := stateReturned.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "shared_network_update", update.Operation)
	require.NotNil(t, update.Recipe)
	require.NotNil(t, update.Recipe.SharedNetworkBeforeUpdate)

	// There should be six commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 12)

	// Validate the commands to be sent to Kea.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()

		switch i {
		case 0, 2, 4, 6:
			require.JSONEq(t,
				`{
					"command": "network4-del",
					"service":["dhcp4"],
					"arguments": {
						"name": "foo",
						"subnets-action":
						"delete"
					}
				}`,
				marshalled)
		case 1, 3, 5:
			require.JSONEq(t,
				`{
					"command": "network4-add",
					"service": ["dhcp4"],
					"arguments": {
						"shared-networks": [
							{
								"name": "bar",
								"subnet4": [
									{
										"pools": [
											{
												"pool":"192.0.2.100-192.0.2.200"
											}
										],
										"id": 0,
										"subnet": "192.0.2.0/24"
									}
								]
							}
						]
					}
				}`,
				marshalled)
		case 7, 8, 10, 11:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		// The default case is executed for the index of 8. The config-reload
		// is only issued for Kea versions earlier than 2.6.0 that don't
		// recount statistics until reloaded.
		default:
			require.JSONEq(t,
				`{
					"command": "config-reload",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		}
		// Verify they are associated with appropriate apps.
		require.NotNil(t, commands[i].App)
	}
}

// Test committing updated shared network, i.e. actually sending control commands to Kea.
func TestCommitSharedNetworkUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	daemonIDs := []int64{apps[0].Daemons[0].ID, apps[1].Daemons[0].ID}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "shared_network_update", daemonIDs...)
	recipe := ConfigRecipe{
		SharedNetworkConfigRecipeParams: SharedNetworkConfigRecipeParams{
			SharedNetworkBeforeUpdate: &sharedNetworks[0],
		},
	}
	err = state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Copy the shared network and modify it. The modifications should be applied in
	// the database upon commit.
	modifiedSharedNetwork := sharedNetworks[0]
	modifiedSharedNetwork.Name = "bar"
	modifiedSharedNetwork.CreatedAt = time.Time{}
	modifiedSharedNetwork.LocalSharedNetworks = sharedNetworks[0].LocalSharedNetworks[0:1]
	modifiedSharedNetwork.LocalSharedNetworks[0].KeaParameters.Allocator = storkutil.Ptr("random")

	ctx, err = module.ApplySharedNetworkUpdate(ctx, &modifiedSharedNetwork)
	require.NoError(t, err)

	// Committing the shared network should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the correct number of commands were sent.
	require.Len(t, agents.RecordedURLs, 5)
	require.Len(t, agents.RecordedCommands, 5)

	// The respective commands should be sent to different servers.
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[2])
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[4])
	require.Equal(t, agents.RecordedURLs[0], agents.RecordedURLs[1])
	require.Equal(t, agents.RecordedURLs[1], agents.RecordedURLs[3])

	// Validate the sent commands and URLS.
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch i {
		case 0, 2:
			require.JSONEq(t,
				`{
					"command": "network4-del",
					"service": ["dhcp4"],
					"arguments": {
						"name":"foo",
						"subnets-action": "delete"
					}
				}`,
				marshalled)
		case 1:
			require.JSONEq(t,
				`{
					"command": "network4-add",
					"service": ["dhcp4"],
					"arguments": {
						"shared-networks": [
							{
								"allocator": "random",
								"name": "bar"
							}
						]
					}
				}`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": ["dhcp4"]
				}`,
				marshalled)
		}
	}

	// Make sure that the subnet has been updated in the database.
	updatedSharedNetwork, err := dbmodel.GetSharedNetwork(db, sharedNetworks[0].ID)
	require.NoError(t, err)
	require.NotNil(t, updatedSharedNetwork)
	require.Equal(t, "bar", updatedSharedNetwork.Name)
	require.Len(t, updatedSharedNetwork.LocalSharedNetworks, 1)
	require.NotNil(t, updatedSharedNetwork.LocalSharedNetworks[0].KeaParameters)
	require.NotNil(t, updatedSharedNetwork.LocalSharedNetworks[0].KeaParameters.Allocator)
	require.Equal(t, "random", *updatedSharedNetwork.LocalSharedNetworks[0].KeaParameters.Allocator)
}

// Test that error is returned when Kea response contains error status code.
func TestCommitSharedNetworkUpdateResponseWithErrorStatus(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents(func(callNo int, cmdResponses []interface{}) {
		json := []byte(`[
			{
				"result": 1,
				"text": "error is error"
			}
		]`)
		command := keactrl.NewCommandBase(keactrl.Network4Del, keactrl.DHCPv4)
		_ = keactrl.UnmarshalResponseList(command, json, cmdResponses[0])
	})

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 1)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	daemonIDs := []int64{apps[0].Daemons[0].ID}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "shared_network_update", daemonIDs...)
	recipe := ConfigRecipe{
		SharedNetworkConfigRecipeParams: SharedNetworkConfigRecipeParams{
			SharedNetworkBeforeUpdate: &sharedNetworks[0],
		},
	}
	err = state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	ctx, err = module.ApplySharedNetworkUpdate(ctx, &sharedNetworks[0])
	require.NoError(t, err)

	_, err = module.Commit(ctx)
	require.ErrorContains(t, err, fmt.Sprintf("network4-del command to %s failed: error status (1) returned by Kea dhcp4 daemon with text: 'error is error'", apps[0].Name))

	// Other commands should not be sent in this case.
	require.Len(t, agents.RecordedURLs, 1)
	require.Len(t, agents.RecordedCommands, 1)
}

// Test scheduling shared network config changes in the database, retrieving and committing them.
func TestCommitScheduledSharedNetworkUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// User is required to associate the config change with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	serverConfig := `{
		"Dhcp6": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet6": [
						{
							"id": 1,
							"subnet": "2001:db8:1::/64"
						}
					]
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	daemonIDs := []int64{apps[0].Daemons[0].ID, apps[1].Daemons[0].ID}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	// Set user id in the context.
	ctx = context.WithValue(ctx, config.UserContextKey, int64(user.ID))

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "shared_network_update", daemonIDs...)
	recipe := ConfigRecipe{
		SharedNetworkConfigRecipeParams: SharedNetworkConfigRecipeParams{
			SharedNetworkBeforeUpdate: &sharedNetworks[0],
		},
	}
	err = state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Copy the shared network and modify it. The modifications should be applied in
	// the database upon commit.
	modifiedSharedNetwork := sharedNetworks[0]
	modifiedSharedNetwork.Name = "bar"
	modifiedSharedNetwork.CreatedAt = time.Time{}
	modifiedSharedNetwork.LocalSharedNetworks[0].KeaParameters.Allocator = storkutil.Ptr("random")
	modifiedSharedNetwork.LocalSharedNetworks[1].KeaParameters.Allocator = storkutil.Ptr("random")

	ctx, err = module.ApplySharedNetworkUpdate(ctx, &modifiedSharedNetwork)
	require.NoError(t, err)

	// Simulate scheduling the config change and retrieving it from the database.
	// The context will hold re-created transaction state.
	ctx = manager.scheduleAndGetChange(ctx, t)
	require.NotNil(t, ctx)

	// Committing the shared network should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the correct number of commands were sent.
	require.Len(t, agents.RecordedURLs, 6)
	require.Len(t, agents.RecordedCommands, 6)

	// The respective commands should be sent to different servers.
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[2])
	require.NotEqual(t, agents.RecordedURLs[1], agents.RecordedURLs[3])
	require.NotEqual(t, agents.RecordedURLs[4], agents.RecordedURLs[5])
	require.Equal(t, agents.RecordedURLs[0], agents.RecordedURLs[1])
	require.Equal(t, agents.RecordedURLs[2], agents.RecordedURLs[3])
	require.Equal(t, agents.RecordedURLs[0], agents.RecordedURLs[4])
	require.Equal(t, agents.RecordedURLs[2], agents.RecordedURLs[5])

	// Validate the sent commands and URLS.
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch i {
		case 0, 2:
			require.JSONEq(t,
				`{
					"command": "network6-del",
					"service": ["dhcp6"],
					"arguments": {
						"name":"foo",
						"subnets-action": "delete"
					}
				}`,
				marshalled)
		case 1, 3:
			require.JSONEq(t,
				`{
					"command": "network6-add",
					"service": ["dhcp6"],
					"arguments": {
						"shared-networks": [
							{
								"allocator": "random",
								"name": "bar"
							}
						]
					}
				}`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": ["dhcp6"]
				}`,
				marshalled)
		}
	}

	// Make sure that the subnet has been updated in the database.
	updatedSharedNetwork, err := dbmodel.GetSharedNetwork(db, sharedNetworks[0].ID)
	require.NoError(t, err)
	require.NotNil(t, updatedSharedNetwork)
	require.Len(t, updatedSharedNetwork.LocalSharedNetworks, 2)
	require.NotNil(t, updatedSharedNetwork.LocalSharedNetworks[0].KeaParameters)
	require.NotNil(t, updatedSharedNetwork.LocalSharedNetworks[0].KeaParameters.Allocator)
	require.Equal(t, "random", *updatedSharedNetwork.LocalSharedNetworks[0].KeaParameters.Allocator)
	require.NotNil(t, updatedSharedNetwork.LocalSharedNetworks[1].KeaParameters)
	require.NotNil(t, updatedSharedNetwork.LocalSharedNetworks[1].KeaParameters.Allocator)
	require.Equal(t, "random", *updatedSharedNetwork.LocalSharedNetworks[1].KeaParameters.Allocator)
}

// Test first stage of deleting a shared network.
func TestBeginSharedNetworkDelete(t *testing.T) {
	module := NewConfigModule(nil)
	require.NotNil(t, module)

	ctx1 := context.Background()
	ctx2, err := module.BeginSharedNetworkDelete(ctx1)
	require.NoError(t, err)
	require.Equal(t, ctx1, ctx2)
}

// Test second stage of deleting an IPv4 shared network.
func TestApplySharedNetwork4Delete(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds.so"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	var daemonIDs []int64
	for _, ls := range sharedNetworks[0].LocalSharedNetworks {
		daemonIDs = append(daemonIDs, ls.DaemonID)
	}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	ctx, err = module.ApplySharedNetworkDelete(ctx, &sharedNetworks[0])
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, state.Scheduled)

	require.Len(t, state.Updates, 1)
	update := state.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "shared_network_delete", update.Operation)
	require.NotNil(t, update.Recipe)

	// There should be four commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 4)

	// Validate the commands.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()
		switch {
		case i < 2:
			require.JSONEq(t,
				fmt.Sprintf(`{
             "command": "network4-del",
             "service": [ "dhcp4" ],
             "arguments": {
                 "name": "%s",
				 "subnets-action": "delete"
             }
         }`, sharedNetworks[0].Name),
				marshalled)
		default:
			require.JSONEq(t,
				`{
					 "command": "config-write",
					 "service": [ "dhcp4" ]
				 }`, marshalled)
		}
		app = commands[i].App
		require.Equal(t, app, sharedNetworks[0].LocalSharedNetworks[i%2].Daemon.App)
	}
}

// Test second stage of deleting an IPv6 shared network.
func TestApplySharedNetwork6Delete(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp6": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet6": [
						{
							"id": 1,
							"subnet": "2001:db8:1::/64"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds.so"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	var daemonIDs []int64
	for _, ls := range sharedNetworks[0].LocalSharedNetworks {
		daemonIDs = append(daemonIDs, ls.DaemonID)
	}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	ctx, err = module.ApplySharedNetworkDelete(ctx, &sharedNetworks[0])
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, state.Scheduled)

	require.Len(t, state.Updates, 1)
	update := state.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "shared_network_delete", update.Operation)
	require.NotNil(t, update.Recipe)

	// There should be two commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 4)

	// Validate the commands.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()
		switch {
		case i < 2:
			require.JSONEq(t,
				fmt.Sprintf(`{
				 "command": "network6-del",
				 "service": [ "dhcp6" ],
				 "arguments": {
					 "name": "%s",
					 "subnets-action": "delete"
				 }
			 }`, sharedNetworks[0].Name),
				marshalled)
		default:
			require.JSONEq(t,
				`{
						 "command": "config-write",
						 "service": [ "dhcp6" ]
					 }`, marshalled)
		}
		app = commands[i].App
		require.Equal(t, app, sharedNetworks[0].LocalSharedNetworks[i%2].Daemon.App)
	}
}

// Test committing shared network deletion, i.e. actually sending control commands to Kea.
func TestCommitSharedNetworkDelete(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds.so"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	var daemonIDs []int64
	for _, ls := range sharedNetworks[0].LocalSharedNetworks {
		daemonIDs = append(daemonIDs, ls.DaemonID)
	}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	ctx, err = module.ApplySharedNetworkDelete(ctx, &sharedNetworks[0])
	require.NoError(t, err)

	// Committing the shared network deletion should result in sending control
	// commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the commands were sent to different servers.
	require.Len(t, agents.RecordedURLs, 4)
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[1])
	require.NotEqual(t, agents.RecordedURLs[2], agents.RecordedURLs[3])

	// Validate the sent commands.
	require.Len(t, agents.RecordedCommands, 4)
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch {
		case i < 2:
			require.JSONEq(t,
				fmt.Sprintf(`{
             "command": "network4-del",
             "service": [ "dhcp4" ],
             "arguments": {
                 "name": "%s",
				 "subnets-action": "delete"
             }
         }`, sharedNetworks[0].Name),
				marshalled)
		default:
			require.JSONEq(t,
				`{
					 "command": "config-write",
					 "service": [ "dhcp4" ]
				 }`,
				marshalled)
		}
	}

	// The shared network should have been deleted from the Stork database.
	returnedSharedNetwork, err := dbmodel.GetSharedNetwork(db, sharedNetworks[0].ID)
	require.NoError(t, err)
	require.Nil(t, returnedSharedNetwork)

	// The subnets should also be deleted.
	returnedSubnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Empty(t, returnedSubnets)
}

// Test first stage of adding a subnet.
func TestBeginSubnetAdd(t *testing.T) {
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	ctx, err := module.BeginSubnetAdd(context.Background())
	require.NoError(t, err)

	// There should be no locks on any daemons.
	require.Empty(t, manager.locks)

	// Make sure that the transaction state has been created.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.Len(t, state.Updates, 1)
	require.Equal(t, datamodel.AppTypeKea, state.Updates[0].Target)
	require.Equal(t, "subnet_add", state.Updates[0].Operation)
}

// Test second stage of subnet creation.
func TestApplySubnetAdd(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// Transaction state is required because typically it is created by the
	// BeginSubnetAdd function.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "subnet_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Simulate creating new subnet entry.
	subnet := &dbmodel.Subnet{
		ID:     1,
		Prefix: "192.0.2.0/24",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				DaemonID: 1,
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				AddressPools: []dbmodel.AddressPool{
					{
						LowerBound: "192.0.2.100",
						UpperBound: "192.0.2.200",
					},
				},
			},
			{
				DaemonID: 2,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.5.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				AddressPools: []dbmodel.AddressPool{
					{
						LowerBound: "192.0.2.100",
						UpperBound: "192.0.2.200",
					},
				},
			},
		},
	}
	ctx, err := module.ApplySubnetAdd(ctx, subnet)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	stateReturned, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, stateReturned.Scheduled)

	require.Len(t, stateReturned.Updates, 1)
	update := stateReturned.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "subnet_add", update.Operation)
	require.NotNil(t, update.Recipe)
	require.Nil(t, update.Recipe.SubnetBeforeUpdate)

	// There should be six commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 5)

	// Validate the commands to be sent to Kea.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()

		switch {
		case i < 2:
			require.JSONEq(t,
				`{
					"command": "subnet4-add",
					"service": [ "dhcp4" ],
					"arguments": {
						"subnet4": [
							{
								"id": 1,
								"subnet": "192.0.2.0/24",
								"pools": [
									{
										"pool": "192.0.2.100-192.0.2.200"
									}
								]
							}
						]
					}
				}`,
				marshalled)
		case i < 4:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
					"command": "config-reload",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		}
		// Verify they are associated with appropriate apps.
		require.NotNil(t, commands[i].App)
	}
}

// Test committing created subnet, i.e. actually sending control commands to Kea.
func TestCommitSubnetAdd(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Empty(t, subnets)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	subnet := dbmodel.Subnet{
		Prefix:        "192.0.2.0/24",
		SharedNetwork: &sharedNetworks[0],
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				DaemonID: apps[0].Daemons[0].ID,
				KeaParameters: &keaconfig.SubnetParameters{
					Allocator: storkutil.Ptr("random"),
				},
			},
			{
				DaemonID: apps[1].Daemons[0].ID,
				KeaParameters: &keaconfig.SubnetParameters{
					Allocator: storkutil.Ptr("random"),
				},
			},
		},
	}
	err = subnet.PopulateDaemons(db)
	require.NoError(t, err)

	// Transaction state is required because typically it is created by the
	// BeginSubnetAdd function.
	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "subnet_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	ctx, err = module.ApplySubnetAdd(ctx, &subnet)
	require.NoError(t, err)

	// Committing the subnet should result in sending control commands to Kea servers.
	ctx, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the correct number of commands were sent.
	require.Len(t, agents.RecordedURLs, 6)
	require.Len(t, agents.RecordedCommands, 6)

	// The respective commands should be sent to different servers.
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[2])
	require.NotEqual(t, agents.RecordedURLs[1], agents.RecordedURLs[3])
	require.NotEqual(t, agents.RecordedURLs[4], agents.RecordedURLs[5])
	require.Equal(t, agents.RecordedURLs[0], agents.RecordedURLs[1])
	require.Equal(t, agents.RecordedURLs[2], agents.RecordedURLs[3])

	// Validate the sent commands and URLS.
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch i {
		case 0, 2:
			require.JSONEq(t,
				`{
					"command": "subnet4-add",
					"service": [ "dhcp4" ],
					"arguments": {
						"subnet4": [
							{
								"id": 1,
								"subnet": "192.0.2.0/24",
								"allocator": "random"
							}
						]
					}
				}`,
				marshalled)
		case 1, 3:
			require.JSONEq(t,
				`{
					"command": "network4-subnet-add",
					"service": [ "dhcp4" ],
					"arguments": {
						"name": "foo",
						"id": 1
					}
				}`,
				marshalled)

		default:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		}
	}

	// Make sure that the subnet has been updated in the database.
	addedSubnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, addedSubnets, 1)
	require.NotNil(t, addedSubnets[0])
	require.Len(t, addedSubnets[0].LocalSubnets, 2)
	require.NotNil(t, addedSubnets[0].LocalSubnets[0].KeaParameters)
	require.NotNil(t, addedSubnets[0].LocalSubnets[0].KeaParameters.Allocator)
	require.Equal(t, "random", *addedSubnets[0].LocalSubnets[0].KeaParameters.Allocator)
	require.NotNil(t, addedSubnets[0].LocalSubnets[1].KeaParameters)
	require.NotNil(t, addedSubnets[0].LocalSubnets[1].KeaParameters.Allocator)
	require.Equal(t, "random", *addedSubnets[0].LocalSubnets[1].KeaParameters.Allocator)

	recipe, err := config.GetRecipeForUpdate[ConfigRecipe](ctx, 0)
	require.NoError(t, err)
	require.NotNil(t, recipe.SubnetID)
	require.EqualValues(t, addedSubnets[0].ID, *recipe.SubnetID)
}

// Test the first stage of updating a subnet. It checks that the subnet information
// is fetched from the database and stored in the context. It also checks that
// appropriate locks are applied.
func TestBeginSubnetUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds.so"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	ctx, err := module.BeginSubnetUpdate(context.Background(), subnets[0].ID)
	require.NoError(t, err)

	// Make sure that the locks have been applied on the daemons owning
	// the host.
	require.Contains(t, manager.locks, apps[0].Daemons[0].ID)
	require.Contains(t, manager.locks, apps[1].Daemons[0].ID)

	// Make sure that the host information has been stored in the context.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.Len(t, state.Updates, 1)
	require.Equal(t, datamodel.AppTypeKea, state.Updates[0].Target)
	require.Equal(t, "subnet_update", state.Updates[0].Operation)
	require.NotNil(t, state.Updates[0].Recipe.SubnetBeforeUpdate)
	require.NotNil(t, state.Updates[0].Recipe.SubnetBeforeUpdate.SharedNetwork)
	require.Equal(t, "foo", state.Updates[0].Recipe.SubnetBeforeUpdate.SharedNetwork.Name)
	require.Equal(t, "192.0.2.0/24", state.Updates[0].Recipe.SubnetBeforeUpdate.Prefix)
	require.Len(t, state.Updates[0].Recipe.SubnetBeforeUpdate.LocalSubnets, 2)
}

// Test second stage of a subnet update.
func TestApplySubnetUpdate(t *testing.T) {
	// Create dummy subnet to be stored in the context. We will later check if
	// it is preserved after applying host update.
	subnet := &dbmodel.Subnet{
		ID:     1,
		Prefix: "192.0.2.0/24",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				DaemonID: 1,
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				AddressPools: []dbmodel.AddressPool{
					{
						LowerBound: "192.0.2.10",
						UpperBound: "192.0.2.100",
					},
				},
			},
			{
				DaemonID: 2,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.5.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				AddressPools: []dbmodel.AddressPool{
					{
						LowerBound: "192.0.2.10",
						UpperBound: "192.0.2.100",
					},
				},
			},
			{
				DaemonID: 3,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.6.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				AddressPools: []dbmodel.AddressPool{
					{
						LowerBound: "192.0.2.10",
						UpperBound: "192.0.2.100",
					},
				},
			},
		},
	}

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1, 2}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "subnet_update", daemonIDs...)
	recipe := ConfigRecipe{
		SubnetConfigRecipeParams: SubnetConfigRecipeParams{
			SubnetBeforeUpdate: subnet,
		},
	}
	err := state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Simulate updating subnet entry.
	subnet = &dbmodel.Subnet{
		ID:     1,
		Prefix: "192.0.2.0/24",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				DaemonID: 1,
				Daemon: &dbmodel.Daemon{
					Name: "dhcp4",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.1",
								Port:    1234,
							},
						},
					},
				},
				AddressPools: []dbmodel.AddressPool{
					{
						LowerBound: "192.0.2.100",
						UpperBound: "192.0.2.200",
					},
				},
			},
			{
				DaemonID: 2,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.5.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				AddressPools: []dbmodel.AddressPool{
					{
						LowerBound: "192.0.2.100",
						UpperBound: "192.0.2.200",
					},
				},
			},
			{
				DaemonID: 4,
				Daemon: &dbmodel.Daemon{
					Name:    "dhcp4",
					Version: "2.6.0",
					App: &dbmodel.App{
						AccessPoints: []*dbmodel.AccessPoint{
							{
								Type:    dbmodel.AccessPointControl,
								Address: "192.0.2.2",
								Port:    2345,
							},
						},
					},
				},
				AddressPools: []dbmodel.AddressPool{
					{
						LowerBound: "192.0.2.100",
						UpperBound: "192.0.2.200",
					},
				},
			},
		},
	}
	ctx, err = module.ApplySubnetUpdate(ctx, subnet)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	stateReturned, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, stateReturned.Scheduled)

	require.Len(t, stateReturned.Updates, 1)
	update := stateReturned.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "subnet_update", update.Operation)
	require.NotNil(t, update.Recipe)
	require.NotNil(t, update.Recipe.SubnetBeforeUpdate)

	// There should be six commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 9)

	// Validate the commands to be sent to Kea.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()

		switch {
		case i < 2:
			require.JSONEq(t,
				`{
					"command": "subnet4-update",
					"service": [ "dhcp4" ],
					"arguments": {
						"subnet4": [
							{
								"id": 0,
								"subnet": "192.0.2.0/24",
								"pools": [
									{
										"pool": "192.0.2.100-192.0.2.200"
									}
								]
							}
						]
					}
				}`,
				marshalled)
		case i == 2:
			require.JSONEq(t,
				`{
					"command": "subnet4-add",
					"service": [ "dhcp4" ],
					"arguments": {
						"subnet4": [
							{
								"id": 0,
								"subnet": "192.0.2.0/24",
								"pools": [
									{
										"pool": "192.0.2.100-192.0.2.200"
									}
								]
							}
						]
					}
				}`,
				marshalled)
		case i == 3:
			require.JSONEq(t,
				`{
					"command": "subnet4-del",
					"service": [ "dhcp4" ],
					"arguments": {
						"id": 0
					}
				}`,
				marshalled)
		case i == 6:
			require.JSONEq(t,
				`{
					"command": "config-reload",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": [ "dhcp4" ]
				}`,
				marshalled)
		}
		// Verify they are associated with appropriate apps.
		require.NotNil(t, commands[i].App)
	}
}

// Test committing updated subnet, i.e. actually sending control commands to Kea.
func TestCommitSubnetUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	daemonIDs := []int64{apps[0].Daemons[0].ID, apps[1].Daemons[0].ID}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "subnet_update", daemonIDs...)
	recipe := ConfigRecipe{
		SubnetConfigRecipeParams: SubnetConfigRecipeParams{
			SubnetBeforeUpdate: &subnets[0],
		},
	}
	err = state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Copy the subnet and modify it. The modifications should be applied in
	// the database upon commit.
	modifiedSubnet := subnets[0]
	modifiedSubnet.CreatedAt = time.Time{}
	modifiedSubnet.ClientClass = "foo"
	modifiedSubnet.LocalSubnets[0].KeaParameters.Allocator = storkutil.Ptr("random")
	modifiedSubnet.LocalSubnets[1].KeaParameters.Allocator = storkutil.Ptr("random")

	ctx, err = module.ApplySubnetUpdate(ctx, &modifiedSubnet)
	require.NoError(t, err)

	// Committing the subnet should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the correct number of commands were sent.
	require.Len(t, agents.RecordedURLs, 4)
	require.Len(t, agents.RecordedCommands, 4)

	// The respective commands should be sent to different servers.
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[1])
	require.NotEqual(t, agents.RecordedURLs[2], agents.RecordedURLs[3])
	require.Equal(t, agents.RecordedURLs[0], agents.RecordedURLs[2])
	require.Equal(t, agents.RecordedURLs[1], agents.RecordedURLs[3])

	// Validate the sent commands and URLS.
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch {
		case i < 2:
			require.JSONEq(t,
				`{
				"command": "subnet4-update",
				"service": [ "dhcp4" ],
				"arguments": {
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24",
							"allocator": "random"
						}
					]
				}
			}`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp4" ]
					}`,
				marshalled)
		}
	}

	// Make sure that the subnet has been updated in the database.
	updatedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, updatedSubnet)
	require.Equal(t, "foo", updatedSubnet.ClientClass)
	require.Len(t, updatedSubnet.LocalSubnets, 2)
	require.NotNil(t, updatedSubnet.LocalSubnets[0].KeaParameters)
	require.NotNil(t, updatedSubnet.LocalSubnets[0].KeaParameters.Allocator)
	require.Equal(t, "random", *updatedSubnet.LocalSubnets[0].KeaParameters.Allocator)
	require.NotNil(t, updatedSubnet.LocalSubnets[1].KeaParameters)
	require.NotNil(t, updatedSubnet.LocalSubnets[1].KeaParameters.Allocator)
	require.Equal(t, "random", *updatedSubnet.LocalSubnets[1].KeaParameters.Allocator)
}

// Test scheduling config changes in the database, retrieving and committing it.
func TestCommitScheduledSubnetUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	// Test scheduling config changes in the database, retrieving and committing it.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// User is required to associate the config change with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	serverConfig := `{
		"Dhcp6": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet6": [
						{
							"id": 1,
							"subnet": "2001:db8:1::/64"
						}
					]
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	daemonIDs := []int64{apps[0].Daemons[0].ID, apps[1].Daemons[0].ID}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	// Set user id in the context.
	ctx = context.WithValue(ctx, config.UserContextKey, int64(user.ID))

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "subnet_update", daemonIDs...)
	recipe := ConfigRecipe{
		SubnetConfigRecipeParams: SubnetConfigRecipeParams{
			SubnetBeforeUpdate: &subnets[0],
		},
	}
	err = state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Copy the subnet and modify it. The modifications should be applied in
	// the database upon commit.
	modifiedSubnet := subnets[0]
	modifiedSubnet.CreatedAt = time.Time{}
	modifiedSubnet.ClientClass = "foo"
	modifiedSubnet.LocalSubnets[0].KeaParameters.Allocator = storkutil.Ptr("random")
	modifiedSubnet.LocalSubnets[1].KeaParameters.Allocator = storkutil.Ptr("random")

	ctx, err = module.ApplySubnetUpdate(ctx, &modifiedSubnet)
	require.NoError(t, err)

	// Simulate scheduling the config change and retrieving it from the database.
	// The context will hold re-created transaction state.
	ctx = manager.scheduleAndGetChange(ctx, t)
	require.NotNil(t, ctx)

	// Committing the subnet should result in sending control commands to Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the correct number of commands were sent.
	require.Len(t, agents.RecordedCommands, 4)

	// Validate the sent commands and URLS.
	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()

		switch {
		case i < 2:
			require.JSONEq(t,
				`{
					"command": "subnet6-update",
					"service": [ "dhcp6" ],
					"arguments": {
						"subnet6": [
							{
								"id": 1,
								"subnet": "2001:db8:1::/64",
								"allocator": "random"
							}
						]
					}
				}`,
				marshalled)
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp6" ]
					}`,
				marshalled)
		}
	}

	// Make sure that the subnet has been updated in the database.
	updatedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, updatedSubnet)
	require.Equal(t, "foo", updatedSubnet.ClientClass)
	require.Len(t, updatedSubnet.LocalSubnets, 2)
	require.NotNil(t, updatedSubnet.LocalSubnets[0].KeaParameters)
	require.NotNil(t, updatedSubnet.LocalSubnets[0].KeaParameters.Allocator)
	require.Equal(t, "random", *updatedSubnet.LocalSubnets[0].KeaParameters.Allocator)
	require.NotNil(t, updatedSubnet.LocalSubnets[1].KeaParameters)
	require.NotNil(t, updatedSubnet.LocalSubnets[1].KeaParameters.Allocator)
	require.Equal(t, "random", *updatedSubnet.LocalSubnets[1].KeaParameters.Allocator)
}

// Test that error is returned when Kea response contains error status code.
func TestCommitSubnetUpdateResponseWithErrorStatus(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents(func(callNo int, cmdResponses []interface{}) {
		json := []byte(`[
            {
                "result": 1,
                "text": "error is error"
            }
        ]`)
		command := keactrl.NewCommandBase(keactrl.Subnet4Update, keactrl.DHCPv4)
		_ = keactrl.UnmarshalResponseList(command, json, cmdResponses[0])
	})

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 1)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	daemonIDs := []int64{apps[0].Daemons[0].ID}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate[ConfigRecipe](datamodel.AppTypeKea, "subnet_update", daemonIDs...)
	recipe := ConfigRecipe{
		SubnetConfigRecipeParams: SubnetConfigRecipeParams{
			SubnetBeforeUpdate: &subnets[0],
		},
	}
	err = state.SetRecipeForUpdate(0, &recipe)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	ctx, err = module.ApplySubnetUpdate(ctx, &subnets[0])
	require.NoError(t, err)

	_, err = module.Commit(ctx)
	require.ErrorContains(t, err, fmt.Sprintf("subnet4-update command to %s failed: error status (1) returned by Kea dhcp4 daemon with text: 'error is error'", apps[0].Name))

	// Other commands should not be sent in this case.
	require.Len(t, agents.RecordedCommands, 1)
}

// Test second stage of deleting an IPv4 subnet.
func TestApplySubnet4Delete(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds.so"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	var daemonIDs []int64
	for _, ls := range subnets[0].LocalSubnets {
		daemonIDs = append(daemonIDs, ls.DaemonID)
	}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	ctx, err = module.ApplySubnetDelete(ctx, &subnets[0])
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, state.Scheduled)

	require.Len(t, state.Updates, 1)
	update := state.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "subnet_delete", update.Operation)
	require.NotNil(t, update.Recipe)

	// There should be six commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 6)
	require.Equal(t, commands[0].App, commands[1].App)
	require.Equal(t, commands[0].App, commands[4].App)
	require.Equal(t, commands[2].App, commands[3].App)
	require.Equal(t, commands[2].App, commands[5].App)
	require.NotEqual(t, commands[0].App, commands[2].App)
	require.NotEqual(t, commands[2].App, commands[4].App)

	// Validate the commands.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()
		switch i {
		case 0, 2:
			require.JSONEq(t, `{
				"command": "network4-subnet-del",
				"service": [ "dhcp4" ],
				"arguments": {
					"id": 1,
					"name": "foo"
				}
			}`, marshalled)
		case 1, 3:
			require.JSONEq(t,
				fmt.Sprintf(`{
				"command": "subnet4-del",
				"service": [ "dhcp4" ],
				"arguments": {
					"id": %d
				}
			}`, subnets[0].ID),
				marshalled)
		default:
			require.JSONEq(t,
				`{
					 "command": "config-write",
					 "service": [ "dhcp4" ]
				 }`, marshalled)
		}
	}
}

// Test second stage of deleting an IPv6 subnet.
func TestApplySubnet6Delete(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp6": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet6": [
						{
							"id": 1,
							"subnet": "2001:db8:1::/64"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds.so"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	var daemonIDs []int64
	for _, ls := range subnets[0].LocalSubnets {
		daemonIDs = append(daemonIDs, ls.DaemonID)
	}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	ctx, err = module.ApplySubnetDelete(ctx, &subnets[0])
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	state, ok := config.GetTransactionState[ConfigRecipe](ctx)
	require.True(t, ok)
	require.False(t, state.Scheduled)

	require.Len(t, state.Updates, 1)
	update := state.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, datamodel.AppTypeKea, update.Target)
	require.Equal(t, "subnet_delete", update.Operation)
	require.NotNil(t, update.Recipe)

	// There should be six commands ready to send.
	commands := update.Recipe.Commands
	require.Len(t, commands, 6)
	require.Equal(t, commands[0].App, commands[1].App)
	require.Equal(t, commands[0].App, commands[4].App)
	require.Equal(t, commands[2].App, commands[3].App)
	require.Equal(t, commands[2].App, commands[5].App)
	require.NotEqual(t, commands[0].App, commands[2].App)
	require.NotEqual(t, commands[2].App, commands[4].App)

	// Validate the commands.
	for i := range commands {
		command := commands[i].Command
		marshalled := command.Marshal()
		switch i {
		case 0, 2:
			require.JSONEq(t, `{
				"command": "network6-subnet-del",
				"service": [ "dhcp6" ],
				"arguments": {
					"id": 1,
					"name": "foo"
				}
			}`, marshalled)
		case 1, 3:
			require.JSONEq(t,
				fmt.Sprintf(`{
				 "command": "subnet6-del",
				 "service": [ "dhcp6" ],
				 "arguments": {
					 "id": %d
				 }
			 }`, subnets[0].ID),
				marshalled)
		default:
			require.JSONEq(t,
				`{
						 "command": "config-write",
						 "service": [ "dhcp6" ]
					 }`, marshalled)
		}
	}
}

// Test committing subnet deletion, i.e. actually sending control commands to Kea.
func TestCommitSubnetDelete(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	module := NewConfigModule(manager)
	require.NotNil(t, module)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds.so"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	apps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	var daemonIDs []int64
	for _, ls := range subnets[0].LocalSubnets {
		daemonIDs = append(daemonIDs, ls.DaemonID)
	}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	ctx, err = module.ApplySubnetDelete(ctx, &subnets[0])
	require.NoError(t, err)

	// Committing the subnet deletion should result in sending control commands to
	// Kea servers.
	_, err = module.Commit(ctx)
	require.NoError(t, err)

	// Make sure that the commands were sent to different servers.
	require.Len(t, agents.RecordedURLs, 6)
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[2])
	require.NotEqual(t, agents.RecordedURLs[1], agents.RecordedURLs[3])
	require.NotEqual(t, agents.RecordedURLs[0], agents.RecordedURLs[5])
	require.NotEqual(t, agents.RecordedURLs[2], agents.RecordedURLs[4])

	// Validate the sent commands.
	require.Len(t, agents.RecordedCommands, 6)

	for i, command := range agents.RecordedCommands {
		marshalled := command.Marshal()
		switch i {
		case 0, 2:
			require.JSONEq(t, `{
			"command": "network4-subnet-del",
			"service": [ "dhcp4" ],
			"arguments": {
				"id": 1,
				"name": "foo"
			}
		}`, marshalled)
		case 1, 3:
			require.JSONEq(t,
				fmt.Sprintf(`{
             "command": "subnet4-del",
             "service": [ "dhcp4" ],
             "arguments": {
                 "id": %d
             }
         }`, subnets[0].ID),
				marshalled)
		default:
			require.JSONEq(t,
				`{
					 "command": "config-write",
					 "service": [ "dhcp4" ]
				 }`,
				marshalled)
		}
	}

	// The subnet should have been deleted from the Stork database.
	returnedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.Nil(t, returnedSubnet)
}
