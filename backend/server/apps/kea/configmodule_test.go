package kea

import (
	"context"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	appstest "isc.org/stork/server/apps/test"
	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	testutil "isc.org/stork/testutil"
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
	state, ok := config.GetTransactionState(ctx)
	require.True(t, ok)

	// Create the config change entry in the database.
	scc := &dbmodel.ScheduledConfigChange{
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 10),
		UserID:     userID,
		Updates:    state.Updates,
	}
	err := dbmodel.AddScheduledConfigChange(tm.db, scc)
	require.NoError(t, err)

	// Get the config change back from the database.
	changes, err := dbmodel.GetDueConfigChanges(tm.db)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	change := changes[0]

	// Override the context state.
	state = config.TransactionState{
		Scheduled: true,
		Updates:   change.Updates,
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
	state, ok := config.GetTransactionState(ctx)
	require.True(t, ok)
	require.Len(t, state.Updates, 1)
	require.Equal(t, "kea", state.Updates[0].Target)
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
	state := config.NewTransactionStateWithUpdate("kea", "host_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Simulate submitting new host entry. The host is associated with
	// two different daemons/apps.
	host := &dbmodel.Host{
		ID:       1,
		Hostname: "cool.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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
	returnedState, ok := config.GetTransactionState(ctx)
	require.True(t, ok)
	require.False(t, returnedState.Scheduled)

	require.Len(t, returnedState.Updates, 1)
	update := state.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, "kea", update.Target)
	require.Equal(t, "host_add", update.Operation)
	require.NotNil(t, update.Recipe)
	require.Contains(t, update.Recipe, "commands")

	// There should be two commands ready to send.
	commands := update.Recipe["commands"].([]interface{})
	require.Len(t, commands, 2)

	// Validate the first command and associated app.
	command, ok := commands[0].(map[string]interface{})["command"].(*keactrl.Command)
	require.True(t, ok)
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

	app, ok := commands[0].(map[string]interface{})["app"].(*dbmodel.App)
	require.True(t, ok)
	require.Equal(t, app, host.LocalHosts[0].Daemon.App)

	// Validate the second command and associated app.
	command, ok = commands[1].(map[string]interface{})["command"].(*keactrl.Command)
	require.True(t, ok)
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

	app, ok = commands[1].(map[string]interface{})["app"].(*dbmodel.App)
	require.True(t, ok)
	require.Equal(t, app, host.LocalHosts[1].Daemon.App)
}

// Test committing added host, i.e. actually sending control commands to Kea.
func TestCommitHostAdd(t *testing.T) {
	// Create the config manager instance "connected to" fake agents.
	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	// Create Kea config module.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	// Transaction state is required because typically it is created by the
	// BeginHostAdd function.
	state := config.NewTransactionStateWithUpdate("kea", "host_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Create new host reservation and store it in the context.
	host := &dbmodel.Host{
		ID:       1,
		Hostname: "cool.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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
		command := keactrl.NewCommand("reservation-add", []string{"dhcp4"}, nil)
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
	state := config.NewTransactionStateWithUpdate("kea", "host_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Create new host reservation and store it in the context.
	host := &dbmodel.Host{
		ID:       1,
		Hostname: "cool.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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
						Name: "kea@192.0.2.1",
					},
				},
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

// Test scheduling config changes in the database, retrieving and committing it.
func TestCommitScheduledHostAdd(t *testing.T) {
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
		Password: "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// Transaction state is required because typically it is created by the
	// BeginHostAdd function.
	state := config.NewTransactionStateWithUpdate("kea", "host_add")
	ctx := context.WithValue(context.Background(), config.StateContextKey, *state)

	// Set user id in the context.
	ctx = context.WithValue(ctx, config.UserContextKey, int64(user.ID))

	// Create the host and store it in the context.
	host := &dbmodel.Host{
		ID: 1,
		Subnet: &dbmodel.Subnet{
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					DaemonID:      1,
					LocalSubnetID: 123,
				},
			},
		},
		Hostname: "cool.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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

	hosts, apps := testutil.AddTestHosts(t, db)
	err := dbmodel.AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[1].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)

	ctx, err := module.BeginHostUpdate(context.Background(), hosts[0].ID)
	require.NoError(t, err)

	// Make sure that the locks have been applied on the daemons owning
	// the host.
	require.Contains(t, manager.locks, apps[0].Daemons[0].ID)
	require.Contains(t, manager.locks, apps[1].Daemons[0].ID)

	// Make sure that the host information has been stored in the context.
	state, ok := config.GetTransactionState(ctx)
	require.True(t, ok)
	require.Len(t, state.Updates, 1)
	require.Equal(t, "kea", state.Updates[0].Target)
	require.Equal(t, "host_update", state.Updates[0].Operation)
	require.Contains(t, state.Updates[0].Recipe, "host_before_update")
}

// Test second stage of a host update.
func TestApplyHostUpdate(t *testing.T) {
	// Create dummy host to be stored in the context. We will later check if
	// it is preserved after applying host update.
	host := &dbmodel.Host{
		ID:       1,
		Hostname: "cool.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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

	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate("kea", "host_update", daemonIDs...)
	err := state.SetValueForUpdate(0, "host_before_update", *host)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	// Simulate updating host entry. We change host identifier and hostname.
	host = &dbmodel.Host{
		ID:       1,
		Hostname: "foo.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{2, 3, 4, 5, 6, 7},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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
	ctx, err = module.ApplyHostUpdate(ctx, host)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	stateReturned, ok := config.GetTransactionState(ctx)
	require.True(t, ok)
	require.False(t, stateReturned.Scheduled)

	require.Len(t, stateReturned.Updates, 1)
	update := stateReturned.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, "kea", update.Target)
	require.Equal(t, "host_update", update.Operation)
	require.NotNil(t, update.Recipe)
	require.Contains(t, update.Recipe, "commands")
	require.Contains(t, update.Recipe, "host_before_update")

	// There should be four commands ready to send. Two reservation-del and two
	// reservation-add.
	commands := update.Recipe["commands"].([]interface{})
	require.Len(t, commands, 4)

	// Validate the commands to be sent to Kea.
	for i := range commands {
		command, ok := commands[i].(map[string]interface{})["command"].(*keactrl.Command)
		require.True(t, ok)
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
                             "hostname": "foo.example.org"
                         }
                     }
                 }`,
				marshalled)
		}
		// Verify they are associated with appropriate apps.
		app, ok := commands[i].(map[string]interface{})["app"].(*dbmodel.App)
		require.True(t, ok)
		require.Equal(t, app, host.LocalHosts[i%2].Daemon.App)
	}
}

// Test committing updated host, i.e. actually sending control commands to Kea.
func TestCommitHostUpdate(t *testing.T) {
	// Create host reservation.
	host := &dbmodel.Host{
		ID:       1,
		Hostname: "cool.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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

	// Create the config manager instance "connected to" fake agents.
	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(&appstest.ManagerAccessorsWrapper{
		Agents:    agents,
		DefLookup: dbmodel.NewDHCPOptionDefinitionLookup(),
	})

	// Create Kea config module.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

	state := config.NewTransactionStateWithUpdate("kea", "host_update", daemonIDs...)
	err := state.SetValueForUpdate(0, "host_before_update", *host)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	ctx, err = module.ApplyHostUpdate(ctx, host)
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
                             "hostname": "cool.example.org"
                         }
                     }
                 }`,
				marshalled)
		}
	}
}

// Test that error is returned when Kea response contains error status code.
func TestCommitHostUpdateResponseWithErrorStatus(t *testing.T) {
	// Create new host reservation.
	host := &dbmodel.Host{
		ID:       1,
		Hostname: "cool.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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
						Name: "kea@192.0.2.1",
					},
				},
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
		command := keactrl.NewCommand("reservation-del", []string{"dhcp4"}, nil)
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

	state := config.NewTransactionStateWithUpdate("kea", "host_update", daemonIDs...)
	err := state.SetValueForUpdate(0, "host_before_update", *host)
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
	// Create the host.
	host := &dbmodel.Host{
		ID: 1,
		Subnet: &dbmodel.Subnet{
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					DaemonID:      1,
					LocalSubnetID: 123,
				},
			},
		},
		Hostname: "cool.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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
		},
	}

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

	// It is required to associate the config change with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
		Password: "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// Prepare the context.
	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)
	ctx = context.WithValue(ctx, config.UserContextKey, int64(user.ID))

	state := config.NewTransactionStateWithUpdate("kea", "host_update", daemonIDs...)
	err = state.SetValueForUpdate(0, "host_before_update", *host)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, config.StateContextKey, *state)

	ctx, err = module.ApplyHostUpdate(ctx, host)
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
                             "hostname": "cool.example.org"
                         }
                     }
                 }`,
				marshalled)
		}
	}
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
		ID:       1,
		Hostname: "cool.example.org",
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
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
	ctx, err := module.ApplyHostDelete(ctx, host)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	state, ok := config.GetTransactionState(ctx)
	require.True(t, ok)
	require.False(t, state.Scheduled)

	require.Len(t, state.Updates, 1)
	update := state.Updates[0]

	// Basic validation of the retrieved state.
	require.Equal(t, "kea", update.Target)
	require.Equal(t, "host_delete", update.Operation)
	require.NotNil(t, update.Recipe)
	require.Contains(t, update.Recipe, "commands")

	// There should be two commands ready to send.
	commands := update.Recipe["commands"].([]interface{})
	require.Len(t, commands, 2)

	// Validate the first command and associated app.
	command, ok := commands[0].(map[string]interface{})["command"].(*keactrl.Command)
	require.True(t, ok)
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

	app, ok := commands[0].(map[string]interface{})["app"].(*dbmodel.App)
	require.True(t, ok)
	require.Equal(t, app, host.LocalHosts[0].Daemon.App)

	// Validate the second command and associated app.
	command, ok = commands[1].(map[string]interface{})["command"].(*keactrl.Command)
	require.True(t, ok)
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

	app, ok = commands[1].(map[string]interface{})["app"].(*dbmodel.App)
	require.True(t, ok)
	require.Equal(t, app, host.LocalHosts[1].Daemon.App)
}

// Test committing added host, i.e. actually sending control commands to Kea.
func TestCommitHostDelete(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	hosts, apps := testutil.AddTestHosts(t, db)
	err := dbmodel.AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[1].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)

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

	hosts, apps := testutil.AddTestHosts(t, db)
	err := dbmodel.AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)

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
		Password: "test",
	}
	_, err = dbmodel.CreateUser(db, user)
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
