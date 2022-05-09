package kea

import (
	"context"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Test config manager. Besides returning database and agents instance
// it also provides additional functions useful in testing.
type testManager struct {
	db     *pg.DB
	agents agentcomm.ConnectedAgents
}

// Creates new test config manager instance.
func newTestManager(db *pg.DB, agents *agentcommtest.FakeAgents) *testManager {
	return &testManager{
		db:     db,
		agents: agents,
	}
}

// Returns database instance.
func (tm testManager) GetDB() *pg.DB {
	return tm.db
}

// Returns an interface to the test agents.
func (tm testManager) GetConnectedAgents() agentcomm.ConnectedAgents {
	return tm.agents
}

// Simulates adding and retrieving a config change from the database. As a result,
// the returned context contains transaction state re-created from the database
// entry. It simulates scheduling config changes.
func (tm testManager) scheduleAndGetChange(ctx context.Context, t *testing.T) context.Context {
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
	module := NewConfigModule(nil)
	require.NotNil(t, module)

	ctx1 := context.Background()
	ctx2, err := module.BeginHostAdd(ctx1)
	require.NoError(t, err)
	require.Equal(t, ctx1, ctx2)
}

// Test second stage of adding a new host.
func TestApplyHostAdd(t *testing.T) {
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
	ctx, err := module.ApplyHostAdd(ctx, host)
	require.NoError(t, err)

	// Make sure that the transaction state exists and comprises expected data.
	state, ok := config.GetTransactionState(ctx)
	require.True(t, ok)
	require.False(t, state.Scheduled)

	require.Len(t, state.Updates, 1)
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
	manager := newTestManager(nil, agents)

	// Create Kea config module.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

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
	_, err = module.commitHostAdd(ctx)
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

	manager := newTestManager(nil, agents)

	// Create Kea config module.
	module := NewConfigModule(manager)
	require.NotNil(t, module)

	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)

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

	_, err = module.commitHostAdd(ctx)
	require.ErrorContains(t, err, "reservation-add command to kea@192.0.2.1 failed: error status (1) returned by Kea dhcp4 daemon with text: 'error is error'")

	// The second command should not be sent in this case.
	require.Len(t, agents.RecordedCommands, 1)
}

// Test scheduling config changes in the database, retrieving and committing it.
func TestCommitScheduledHostAdd(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := newTestManager(db, agents)

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

	// Prepare the context.
	daemonIDs := []int64{1}
	ctx := context.WithValue(context.Background(), config.DaemonsContextKey, daemonIDs)
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
	_, err = module.commitHostAdd(ctx)
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
