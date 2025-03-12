package dnsop

import (
	context "context"
	iter "iter"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	bind9stats "isc.org/stork/appdata/bind9stats"
	agentcomm "isc.org/stork/server/agentcomm"
	appstest "isc.org/stork/server/apps/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
)

//go:generate mockgen -package=dnsop -destination=connectedagentsmock_test.go -source=../agentcomm/agentcomm.go ConnectedAgents

// Test error used in the unit tests.
type testError struct{}

// Returns error as a string.
func (err *testError) Error() string {
	return "test error"
}

// Test an error indicating that the manager is already fetching zones from the agents.
func TestManagerAlreadyFetchingError(t *testing.T) {
	err := ManagerAlreadyFetchingError{}
	require.Equal(t, "DNS manager is already fetching zones from the agents", err.Error())
}

// Test instantiating new DNS manager.
func TestNewManager(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NotNil(t, manager)
}

// Test that an error is returned when trying to fetch the zones but there
// is a database issue.
func TestFetchZonesDatabaseError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	// Close the database connection to cause an error.
	teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NotNil(t, manager)

	_, err := manager.FetchZones(10, 1000, true)
	require.Error(t, err)
	require.ErrorContains(t, err, "problem getting")
}

// This test verifies that "busy" status is set in the database when zone
// inventory is busy while fetching zones.
func TestFetchZonesInventoryBusyError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Add a machine and app.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	app := &dbmodel.App{
		ID:        0,
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeBind9,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewBind9Daemon(true),
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Return "busy" error on first iteration.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), nil).DoAndReturn(func(context.Context, *dbmodel.App, *bind9stats.ZoneFilter) iter.Seq2[*bind9stats.ExtendedZone, error] {
		return func(yield func(*bind9stats.ExtendedZone, error) bool) {
			_ = !yield(nil, agentcomm.NewZoneInventoryBusyError("foo"))
		}
	})
	require.NoError(t, err)

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NotNil(t, manager)

	notifyChannel, err := manager.FetchZones(10, 100, true)
	require.NoError(t, err)
	notification := <-notifyChannel

	// Make sure the "busy" error was reported.
	require.Len(t, notification.results, 1)
	require.NotNil(t, notification.results[app.Daemons[0].ID].Error)
	require.Contains(t, *notification.results[app.Daemons[0].ID].Error, "Zone inventory is temporarily busy on the agent foo")

	// The database should also hold the fetch result.
	state, err := dbmodel.GetZoneInventoryState(db, app.Daemons[0].ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, app.Daemons[0].ID, state.DaemonID)
	require.NotZero(t, state.CreatedAt)
	require.NotNil(t, state.State.Error)
	require.Contains(t, *state.State.Error, "Zone inventory is temporarily busy on the agent foo")
	require.Equal(t, dbmodel.ZoneInventoryStatusBusy, state.State.Status)
	require.Nil(t, state.State.ZoneCount)

	// Make sure that no zones have been added to the database.
	zones, total, err := dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, zones)
}

// This test verifies that "uninitialized" status is set in the database when zone
// inventory is busy while fetching zones.
func TestFetchZonesInventoryNotInitedError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Add a machine and app.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	app := &dbmodel.App{
		ID:        0,
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeBind9,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewBind9Daemon(true),
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Return "uninitialized" error on first iteration.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), nil).DoAndReturn(func(context.Context, *dbmodel.App, *bind9stats.ZoneFilter) iter.Seq2[*bind9stats.ExtendedZone, error] {
		return func(yield func(*bind9stats.ExtendedZone, error) bool) {
			_ = !yield(nil, agentcomm.NewZoneInventoryNotInitedError("foo"))
		}
	})
	require.NoError(t, err)

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NotNil(t, manager)

	notifyChannel, err := manager.FetchZones(10, 100, true)
	require.NoError(t, err)
	notification := <-notifyChannel

	// Make sure the "busy" error was reported.
	require.Len(t, notification.results, 1)
	require.NotNil(t, notification.results[app.Daemons[0].ID].Error)
	require.Contains(t, *notification.results[app.Daemons[0].ID].Error, "DNS zones have not been loaded on the agent foo")

	// The database should also hold the fetch result.
	state, err := dbmodel.GetZoneInventoryState(db, app.Daemons[0].ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, app.Daemons[0].ID, state.DaemonID)
	require.NotZero(t, state.CreatedAt)
	require.NotNil(t, state.State.Error)
	require.Contains(t, *state.State.Error, "DNS zones have not been loaded on the agent foo")
	require.Equal(t, dbmodel.ZoneInventoryStatusUninitialized, state.State.Status)
	require.Nil(t, state.State.ZoneCount)

	// Make sure that no zones have been added to the database.
	zones, total, err := dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, zones)
}

// This test verifies that "erred" status is set in the database when zone
// inventory is busy while fetching zones.
func TestFetchZonesInventoryOtherError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Add a machine and app.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	app := &dbmodel.App{
		ID:        0,
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeBind9,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewBind9Daemon(true),
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Return "uninitialized" error on first iteration.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), nil).DoAndReturn(func(context.Context, *dbmodel.App, *bind9stats.ZoneFilter) iter.Seq2[*bind9stats.ExtendedZone, error] {
		return func(yield func(*bind9stats.ExtendedZone, error) bool) {
			_ = !yield(nil, &testError{})
		}
	})
	require.NoError(t, err)

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NotNil(t, manager)

	notifyChannel, err := manager.FetchZones(10, 100, true)
	require.NoError(t, err)
	notification := <-notifyChannel

	// Make sure other error was reported.
	require.Len(t, notification.results, 1)
	require.NotNil(t, notification.results[app.Daemons[0].ID].Error)
	require.Contains(t, *notification.results[app.Daemons[0].ID].Error, "test error")

	// The database should also hold the fetch result.
	state, err := dbmodel.GetZoneInventoryState(db, app.Daemons[0].ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, app.Daemons[0].ID, state.DaemonID)
	require.NotZero(t, state.CreatedAt)
	require.NotNil(t, state.State.Error)
	require.Contains(t, *state.State.Error, "test error")
	require.Equal(t, dbmodel.ZoneInventoryStatusErred, state.State.Status)
	require.Nil(t, state.State.ZoneCount)

	// Make sure that no zones have been added to the database.
	zones, total, err := dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, zones)
}

// This test verifies that the manager can fetch zones from many servers
// simultaneously.
func TestFetchZones(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	randomZones := testutil.GenerateRandomZones(1000)

	// Generate many machines and apps.
	for i := 0; i < 100; i++ {
		machine := &dbmodel.Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := dbmodel.AddMachine(db, machine)
		require.NoError(t, err)

		app := &dbmodel.App{
			ID:        0,
			MachineID: machine.ID,
			Type:      dbmodel.AppTypeBind9,
			Daemons: []*dbmodel.Daemon{
				dbmodel.NewBind9Daemon(true),
			},
		}
		_, err = dbmodel.AddApp(db, app)
		require.NoError(t, err)

		// We're going to test a corner case all of the servers have exactly the
		// same set of zones. This is unrealistic scenario but it well stresses
		// the code generating many conflicts in the database. We want to make
		// sure that no zone is lost due to conflicts.
		mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(a any) bool {
			return a.(*dbmodel.App).ID == app.ID
		}), nil).DoAndReturn(func(context.Context, *dbmodel.App, *bind9stats.ZoneFilter) iter.Seq2[*bind9stats.ExtendedZone, error] {
			return func(yield func(*bind9stats.ExtendedZone, error) bool) {
				for _, zone := range randomZones {
					zone := &bind9stats.ExtendedZone{
						Zone: bind9stats.Zone{
							ZoneName: zone.Name,
							Class:    zone.Class,
							Serial:   zone.Serial,
							Type:     zone.Type,
							Loaded:   time.Now().UTC(),
						},
						ViewName:       "foo",
						TotalZoneCount: int64(len(randomZones)),
					}
					if !yield(zone, nil) {
						return
					}
				}
			}
		})
		require.NoError(t, err)
	}

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NotNil(t, manager)

	// Start zones fetch. Use up to 10 goroutines and set the batch
	// size to 100 zones.
	notifyChannel, err := manager.FetchZones(10, 100, true)
	require.Nil(t, err)
	notification := <-notifyChannel

	// Make sure there are no errors.
	for _, result := range notification.results {
		if result.Error != nil {
			require.Empty(t, *result.Error)
		}
	}

	// Get all the zones from the database to make sure that all zones
	// have been inserted.
	zones, total, err := dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Equal(t, 1000, total)
	require.Len(t, zones, 1000)

	for _, zone := range zones {
		// Each zone is shared by 100 servers.
		require.Len(t, zone.LocalZones, 100)
		for _, localZone := range zone.LocalZones {
			require.Equal(t, "foo", localZone.View)
			require.Equal(t, "IN", localZone.Class)
			require.Equal(t, "primary", localZone.Type)
		}
	}

	// Notification should contain the results for all servers.
	require.Len(t, notification.results, 100)
	for daemonID, returnedState := range notification.results {
		require.NotNil(t, returnedState)
		require.Equal(t, dbmodel.ZoneInventoryStatusOK, returnedState.Status)
		require.Nil(t, returnedState.Error)

		// The database should also hold the fetch results for all servers.
		state, err := dbmodel.GetZoneInventoryState(db, daemonID)
		require.NoError(t, err)
		require.NotNil(t, state)
		require.Equal(t, daemonID, state.DaemonID)
		require.NotZero(t, state.CreatedAt)
		require.Nil(t, state.State.Error)
		require.Equal(t, dbmodel.ZoneInventoryStatusOK, state.State.Status)
		require.NotNil(t, state.State.ZoneCount)
		require.EqualValues(t, len(randomZones), *state.State.ZoneCount)
	}
}

// Test triggering the zone fetch multiple times. The second attempt
// to fetch the zones should return an error.
func TestFetchZonesMultipleTimes(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	app := &dbmodel.App{
		ID:        0,
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeBind9,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewBind9Daemon(true),
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Return an empty iterator. Getting actual zones is not in scope for this test.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Any(), nil).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.App, *bind9stats.ZoneFilter) iter.Seq2[*bind9stats.ExtendedZone, error] {
		return func(yield func(*bind9stats.ExtendedZone, error) bool) {}
	})

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NotNil(t, manager)

	// Begin first fetch but do not receive the result from the notifyChannel.
	// This should keep the fetch active.
	notifyChannel, err := manager.FetchZones(1, 1000, true)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, time.Second, time.Millisecond)

	// Begin the second fetch. It should return an error.
	_, err = manager.FetchZones(10, 1000, true)
	var alreadyFetching *ManagerAlreadyFetchingError
	require.ErrorAs(t, err, &alreadyFetching)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, time.Second, time.Millisecond)

	// Complete the fetch.
	<-notifyChannel
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return !isFetching && appsNum == 1 && completedAppsNum == 1
	}, time.Second, time.Millisecond)

	// This time the new attempt should succeed.
	notifyChannel, err = manager.FetchZones(10, 1000, true)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, time.Second, time.Millisecond)

	// Complete the fetch.
	<-notifyChannel
}

// This test verifies that the manager can fetch the same zones from
// multiple views in a single batch.
func TestFetchRepeatedZones(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	randomZones := testutil.GenerateRandomZones(10)

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	app := &dbmodel.App{
		ID:        0,
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeBind9,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewBind9Daemon(true),
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), nil).DoAndReturn(func(context.Context, *dbmodel.App, *bind9stats.ZoneFilter) iter.Seq2[*bind9stats.ExtendedZone, error] {
		return func(yield func(*bind9stats.ExtendedZone, error) bool) {
			// Return the same zones from two different views.
			for _, view := range []string{"foo", "bar"} {
				for _, zone := range randomZones {
					zone := &bind9stats.ExtendedZone{
						Zone: bind9stats.Zone{
							ZoneName: zone.Name,
							Class:    zone.Class,
							Serial:   zone.Serial,
							Type:     zone.Type,
							Loaded:   time.Now().UTC(),
						},
						ViewName:       view,
						TotalZoneCount: int64(len(randomZones)),
					}
					if !yield(zone, nil) {
						return
					}
				}
			}
		}
	})

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NotNil(t, manager)

	// Set the batch size that will include the ones from both views.
	notifyChannel, err := manager.FetchZones(1, 100, true)
	require.Nil(t, err)
	notification := <-notifyChannel

	// Make sure there are no errors.
	for _, result := range notification.results {
		if result.Error != nil {
			require.Empty(t, *result.Error)
		}
	}

	// Get all the zones from the database to make sure that all zones
	// have been inserted.
	zones, total, err := dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Equal(t, 10, total)
	require.Len(t, zones, 10)

	// Make sure that all zones have two associations.
	for _, zone := range zones {
		require.Len(t, zone.LocalZones, 2)
	}
}
