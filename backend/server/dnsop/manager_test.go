package dnsop

import (
	context "context"
	_ "embed"
	"encoding/json"
	iter "iter"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	dnsconfig "isc.org/stork/appcfg/dnsconfig"
	bind9stats "isc.org/stork/appdata/bind9stats"
	agentcomm "isc.org/stork/server/agentcomm"
	appstest "isc.org/stork/server/apps/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
)

//go:generate mockgen -package=dnsop -destination=connectedagentsmock_test.go -source=../agentcomm/agentcomm.go ConnectedAgents

//go:embed testdata/valid-zone.json
var validZoneData []byte

// Test error used in the unit tests.
type testError struct{}

// Returns error as a string.
func (err *testError) Error() string {
	return "test error"
}

// Test instantiating new zone transfer RRResponse.
func TestNewZoneTransferRRResponse(t *testing.T) {
	rr, err := dnsconfig.NewRR("example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 123456 7200 3600 1209600 3600")
	require.NoError(t, err)
	rrs := []*dnsconfig.RR{
		rr,
	}
	rrResponse := NewZoneTransferRRResponse(rrs)
	require.False(t, rrResponse.Cached)
	require.InDelta(t, time.Now().UTC().Unix(), rrResponse.ZoneTransferAt.Unix(), 5)
	require.NoError(t, rrResponse.Err)
	require.Equal(t, rrs, rrResponse.RRs)
}

// Test instantiating new cached RRResponse.
func TestNewCacheRRRResponse(t *testing.T) {
	rr, err := dnsconfig.NewRR("example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 123456 7200 3600 1209600 3600")
	require.NoError(t, err)
	rrs := []*dnsconfig.RR{
		rr,
	}
	timestamp := time.Now().UTC()
	rrResponse := NewCacheRRResponse(rrs, timestamp)
	require.True(t, rrResponse.Cached)
	require.Equal(t, rrs, rrResponse.RRs)
	require.Equal(t, timestamp, rrResponse.ZoneTransferAt)
	require.NoError(t, rrResponse.Err)
}

// Test instantiating new error RRResponse.
func TestNewErrorRRResponse(t *testing.T) {
	err := &testError{}
	rrResponse := NewErrorRRResponse(err)
	require.Error(t, rrResponse.Err)
	require.Equal(t, err, rrResponse.Err)
	require.False(t, rrResponse.Cached)
	require.Nil(t, rrResponse.RRs)
	require.Zero(t, rrResponse.ZoneTransferAt)
}

// Test an error indicating that the manager is already fetching zones from the agents.
func TestManagerAlreadyFetchingError(t *testing.T) {
	err := ManagerAlreadyFetchingError{}
	require.Equal(t, "DNS manager is already fetching zones from the agents", err.Error())
}

// Test an error indicating that the manager is already requesting RRs for the same zone.
func TestManagerRRsAlreadyRequestedError(t *testing.T) {
	err := ManagerRRsAlreadyRequestedError{
		viewName: "foo",
		zoneName: "bar",
	}
	require.Equal(t, "zone transfer for view foo, zone bar has been already requested by another user", err.Error())
}

// Test instantiating new DNS manager.
func TestNewManager(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	manager.Shutdown()
}

// Test that an error is returned when trying to fetch the zones but there
// is a database issue.
func TestFetchZonesDatabaseError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	// Close the database connection to cause an error.
	teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	_, err = manager.FetchZones(10, 1000, true)
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

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()
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

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

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

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

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

// This test verifies that an error is returned indicating the database error
// while deleting local zones before inserting new ones.
func TestFetchZonesInventoryDeleteLocalZonesError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	randomZones := testutil.GenerateRandomZones(1)

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
			// We are on the fist iteration. Let's close the database connection
			// to cause an error.
			teardown()
			// Return the zone.
			zone := &bind9stats.ExtendedZone{
				Zone: bind9stats.Zone{
					ZoneName: randomZones[0].Name,
					Class:    randomZones[0].Class,
					Serial:   randomZones[0].Serial,
					Type:     randomZones[0].Type,
					Loaded:   time.Now().UTC(),
				},
				ViewName:       "foo",
				TotalZoneCount: int64(len(randomZones)),
			}
			_ = yield(zone, nil)
		}
	})
	require.NoError(t, err)

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	notifyChannel, err := manager.FetchZones(10, 100, true)
	require.NoError(t, err)
	notification := <-notifyChannel

	// Make sure other error was reported.
	require.Len(t, notification.results, 1)
	require.NotNil(t, notification.results[app.Daemons[0].ID].Error)
	require.Contains(t, *notification.results[app.Daemons[0].ID].Error, "database is closed")
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
						RPZ:            true,
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

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

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
			require.True(t, localZone.RPZ)
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
		require.NotNil(t, state.State.DistinctZoneCount)
		require.NotNil(t, state.State.BuiltinZoneCount)
		require.EqualValues(t, len(randomZones), *state.State.DistinctZoneCount)
		require.Zero(t, *state.State.BuiltinZoneCount)
	}
}

// Test triggering the zone fetch multiple times. The second attempt
// to fetch the zones should return an error.
func TestFetchZonesMultipleTimes(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	randomZones := testutil.GenerateRandomZones(1000)

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

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	// Begin first fetch but do not receive the result from the notifyChannel.
	// This should keep the fetch active.
	notifyChannel, err := manager.FetchZones(1, 1000, true)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)

	// All zones should be in the database.
	zones, _, err := dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zones, 1000)

	// Reduce the number of returned zones to 100. Remaining zones
	// should be removed from the database.
	randomZones = randomZones[:100]

	// Begin the second fetch. It should return an error.
	_, err = manager.FetchZones(10, 1000, true)
	var alreadyFetching *ManagerAlreadyFetchingError
	require.ErrorAs(t, err, &alreadyFetching)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)

	// The zones should remain untouched.
	zones, _, err = dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zones, 1000)

	// Complete the fetch.
	<-notifyChannel
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return !isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)

	// All zones should be in the database.
	zones, _, err = dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zones, 1000)

	// This time the new attempt should succeed.
	notifyChannel, err = manager.FetchZones(10, 1000, true)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)

	// Complete the fetch.
	<-notifyChannel

	// This time we should have only 100 zones and all other zones should be removed.
	zones, _, err = dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zones, 100)
}

// This test verifies that the manager can fetch the same zones from
// multiple views in a single batch.
func TestFetchRepeatedZones(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	randomZones := testutil.GenerateRandomZones(10)
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 10, string(dbmodel.ZoneTypeBuiltin))

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

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	// Set the batch size that will include the ones from both views.
	notifyChannel, err := manager.FetchZones(1, 100, true)
	require.Nil(t, err)
	notification := <-notifyChannel

	// Make sure there are no errors.
	for _, result := range notification.results {
		if result.Error != nil {
			require.Empty(t, *result.Error)
		}
		require.NotNil(t, result.DistinctZoneCount)
		require.NotNil(t, result.BuiltinZoneCount)
		require.EqualValues(t, 20, *result.DistinctZoneCount)
		require.EqualValues(t, 10, *result.BuiltinZoneCount)
	}

	// Get all the zones from the database to make sure that all zones
	// have been inserted.
	zones, total, err := dbmodel.GetZones(db, nil, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.EqualValues(t, 20, total)
	require.Len(t, zones, 20)

	// Make sure that all zones have two associations.
	for _, zone := range zones {
		require.Len(t, zone.LocalZones, 2)
	}
}

// Test successfully receiving the zone RRs from the agents.
func TestGetZoneRRs(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create a machine and an app. The manager will determine the app
	// to contact based on the daemon ID.
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

	// Add a zone. We will use zone ID to fetch the RRs.
	zone := &dbmodel.Zone{
		ID:    1,
		Name:  "example.com",
		Rname: "com.example",
		LocalZones: []*dbmodel.LocalZone{
			{
				ID:       1,
				View:     "_default",
				DaemonID: app.Daemons[0].ID,
				Class:    "IN",
				Type:     "primary",
				Serial:   1,
				LoadedAt: time.Now().UTC(),
			},
		},
	}
	err = dbmodel.AddZones(db, []*dbmodel.Zone{zone}...)
	require.NoError(t, err)

	// Read the RRs to the returned by the agent from the file.
	var rrs []string
	err = json.Unmarshal(validZoneData, &rrs)
	require.NoError(t, err)

	// Return the RRs using the mock.
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.App, string, string) iter.Seq2[[]*dnsconfig.RR, error] {
		return func(yield func([]*dnsconfig.RR, error) bool) {
			for _, rr := range rrs {
				rr, err := dnsconfig.NewRR(rr)
				require.NoError(t, err)
				if !yield([]*dnsconfig.RR{rr}, nil) {
					return
				}
			}
		}
	})

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	// Collect received RRs.
	collectedRRs := make([]*dnsconfig.RR, 0, len(rrs))
	rrResponses := manager.GetZoneRRs(zone.ID, app.Daemons[0].ID, "_default")
	for rrResponse := range rrResponses {
		require.False(t, rrResponse.Cached)
		require.InDelta(t, time.Now().UTC().Unix(), rrResponse.ZoneTransferAt.Unix(), 5)
		require.NoError(t, rrResponse.Err)
		collectedRRs = append(collectedRRs, rrResponse.RRs...)
	}
	// Validate the returned RRs against the original ones.
	require.Equal(t, len(rrs), len(collectedRRs))
	for i, rr := range collectedRRs {
		// Replace tabs with spaces in the original RR.
		original := strings.Join(strings.Fields(rrs[i]), " ")
		require.Equal(t, original, rr.GetString())
	}
	// Make sure that the timestamp was set.
	zone, err = dbmodel.GetZoneByID(db, zone.ID)
	require.NoError(t, err)
	require.Len(t, zone.LocalZones, 1)
	require.NotNil(t, zone.LocalZones[0].ZoneTransferAt)
	lastRRsTransferAt := *zone.LocalZones[0].ZoneTransferAt

	// Get RRs again. It should return cached RRs.
	collectedRRs = make([]*dnsconfig.RR, 0, len(rrs))
	rrResponses = manager.GetZoneRRs(zone.ID, app.Daemons[0].ID, "_default")
	for rrResponse := range rrResponses {
		require.True(t, rrResponse.Cached)
		require.Equal(t, *zone.LocalZones[0].ZoneTransferAt, rrResponse.ZoneTransferAt)
		require.NoError(t, rrResponse.Err)
		collectedRRs = append(collectedRRs, rrResponse.RRs...)
	}
	// Validate the returned RRs against the original ones.
	require.Equal(t, len(rrs), len(collectedRRs))
	for i, rr := range collectedRRs {
		// Replace tabs with spaces in the original RR.
		original := strings.Join(strings.Fields(rrs[i]), " ")
		require.Equal(t, original, rr.GetString())
	}
	// Make sure that the timestamp was not changed.
	zone, err = dbmodel.GetZoneByID(db, zone.ID)
	require.NoError(t, err)
	require.Len(t, zone.LocalZones, 1)
	require.Equal(t, lastRRsTransferAt, *zone.LocalZones[0].ZoneTransferAt)

	// Finally, get RRs again with forcing zone transfer.
	collectedRRs = make([]*dnsconfig.RR, 0, len(rrs))
	rrResponses = manager.GetZoneRRs(zone.ID, app.Daemons[0].ID, "_default", GetZoneRRsOptionForceZoneTransfer)
	for rrResponse := range rrResponses {
		require.False(t, rrResponse.Cached)
		require.InDelta(t, time.Now().UTC().Unix(), rrResponse.ZoneTransferAt.Unix(), 5)
		require.NoError(t, rrResponse.Err)
		collectedRRs = append(collectedRRs, rrResponse.RRs...)
	}
	// Validate the returned RRs against the original ones.
	require.Equal(t, len(rrs), len(collectedRRs))
	for i, rr := range collectedRRs {
		// Replace tabs with spaces in the original RR.
		original := strings.Join(strings.Fields(rrs[i]), " ")
		require.Equal(t, original, rr.GetString())
	}
	// Make sure that the timestamp was updated.
	zone, err = dbmodel.GetZoneByID(db, zone.ID)
	require.NoError(t, err)
	require.Len(t, zone.LocalZones, 1)
	require.NotNil(t, zone.LocalZones[0].ZoneTransferAt)
	require.Greater(t, *zone.LocalZones[0].ZoneTransferAt, lastRRsTransferAt)
}

// Test that an error is returned if the daemon with the given ID
// is not found.
func TestGetZoneRRsNoDaemon(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Make sure that the agent is not contacted by the DNS manager.
	// The manager should return an error after trying to get the daemon
	// by ID.
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(0)

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	var errors []error
	responses := manager.GetZoneRRs(12, 13, "_default")
	for response := range responses {
		errors = append(errors, response.Err)
	}
	// There should be exactly one error returned.
	require.Len(t, errors, 1)
	require.Contains(t, errors[0].Error(), "daemon with the ID of 13 not found")
}

// Test that an error is returned if the zone with the given ID
// is not found.
func TestGetZoneRRsNoZone(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create the app to ensure that the manager will not fail on
	// trying to get the daemon from the database.
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

	// Make sure that the agent is not contacted by the DNS manager.
	// The manager should return an error after trying to get the zone
	// from the database.
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(0)

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	var errors []error
	responses := manager.GetZoneRRs(12, app.Daemons[0].ID, "_default")
	for response := range responses {
		errors = append(errors, response.Err)
	}
	// There should be exactly one error returned.
	require.Len(t, errors, 1)
	require.Contains(t, errors[0].Error(), "zone with the ID of 12 not found")
}

// Test that an error is returned if another request for the same
// zone is in progress.
func TestGetZoneRRsAnotherRequestInProgress(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create the app.
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

	// Create the zone and associated with the app/daemon.
	zone := &dbmodel.Zone{
		ID:    1,
		Name:  "example.com",
		Rname: "com.example",
		LocalZones: []*dbmodel.LocalZone{
			{
				ID:       1,
				View:     "_default",
				DaemonID: app.Daemons[0].ID,
				Class:    "IN",
				Type:     "primary",
				Serial:   1,
				LoadedAt: time.Now().UTC(),
			},
		},
	}
	err = dbmodel.AddZones(db, []*dbmodel.Zone{zone}...)
	require.NoError(t, err)

	// We need to run first request and ensure it stops, so we can
	// run another request before it completes. The first synchronization
	// group will be used to wait for the mock to start. The other one will
	// pause it while we perform the second request.
	wg1 := sync.WaitGroup{}
	wg2 := sync.WaitGroup{}
	wg1.Add(1)
	wg2.Add(1)
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.App, string, string) iter.Seq2[[]*dnsconfig.RR, error] {
		return func(yield func([]*dnsconfig.RR, error) bool) {
			// Signalling here that the second request can start.
			wg1.Done()
			// Wait for the test to unpause the mock.
			wg2.Wait()
			// Return an empty result. It doesn't really matter what is returned.
			yield([]*dnsconfig.RR{}, nil)
		}
	})

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	// This wait group is used to ensure that the background goroutine is
	// finished before we complete the test.
	var wg3 sync.WaitGroup
	wg3.Add(1)
	go func() {
		defer wg3.Done()
		responses := manager.GetZoneRRs(zone.ID, app.Daemons[0].ID, "_default")
		for response := range responses {
			require.NoError(t, response.Err)
		}
	}()

	// Wait for the first request to start.
	wg1.Wait()

	// Run the second request while the first one is in progress.
	// Since we use the same IDs and view name, the manager should
	// refuse it.
	var errors []error
	responses := manager.GetZoneRRs(zone.ID, app.Daemons[0].ID, "_default")
	for response := range responses {
		errors = append(errors, response.Err)
	}
	require.Len(t, errors, 1)
	require.Contains(t, errors[0].Error(), "has been already requested")

	// Unblock the mock.
	wg2.Done()

	// Wait for the first request to complete.
	wg3.Wait()
}

// Test that an error is not returned when there is an ongoing
// request but another request contains different zone ID, daemon ID,
// or view name.
func TestGetZoneRRsAnotherRequestInProgressDifferentZone(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create the app.
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
			dbmodel.NewBind9Daemon(true),
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Create two zones associated with the daemons.
	zones := []*dbmodel.Zone{
		{
			Name:  "example.com",
			Rname: "com.example",
			LocalZones: []*dbmodel.LocalZone{
				{
					View:     "_default",
					DaemonID: app.Daemons[0].ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
				{
					View:     "trusted",
					DaemonID: app.Daemons[0].ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
				{
					View:     "_default",
					DaemonID: app.Daemons[1].ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
				{
					View:     "trusted",
					DaemonID: app.Daemons[1].ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
			},
		},
		{
			Name:  "example.org",
			Rname: "org.example",
			LocalZones: []*dbmodel.LocalZone{
				{
					View:     "_default",
					DaemonID: app.Daemons[0].ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
				{
					View:     "_default",
					DaemonID: app.Daemons[1].ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
			},
		},
	}
	err = dbmodel.AddZones(db, zones...)
	require.NoError(t, err)

	// We need to run first request and ensure it stops, so we can
	// run another request before it completes. The first synchronization
	// group will be used to wait for the mock to start. The other one will
	// pause it while we perform the second request.
	wg1 := sync.WaitGroup{}
	wg2 := sync.WaitGroup{}
	wg1.Add(1)
	wg2.Add(1)
	var mocks []any
	mocks = append(mocks, mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *dbmodel.App, string, string) iter.Seq2[[]*dnsconfig.RR, error] {
		return func(yield func([]*dnsconfig.RR, error) bool) {
			// Signalling here that the second request can start.
			wg1.Done()
			// Wait for the test to unpause the mock.
			wg2.Wait()
			// Return an empty result. It doesn't really matter what is returned.
			yield([]*dnsconfig.RR{}, nil)
		}
	}))
	mocks = append(mocks, mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.App, string, string) iter.Seq2[[]*dnsconfig.RR, error] {
		return func(yield func([]*dnsconfig.RR, error) bool) {
			yield([]*dnsconfig.RR{}, nil)
		}
	}))
	gomock.InOrder(mocks...)

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)

	// This wait group is used to ensure that the background goroutine is
	// finished before we complete the test.
	var wg3 sync.WaitGroup
	wg3.Add(1)
	go func() {
		defer wg3.Done()
		responses := manager.GetZoneRRs(zones[0].ID, app.Daemons[0].ID, "_default")
		for response := range responses {
			// Since these responses will be read in the cleanup phase, it is expected
			// that some of them will indicate failures while communicating with the
			// database. Therefore, we don't validate them. We merely read them and
			// ensure they are not nil.
			require.NotNil(t, response)
		}
	}()

	// Wait for the first request to start.
	wg1.Wait()

	// Ensure we cleanup after the tests.
	t.Cleanup(func() {
		wg2.Done()
		wg3.Wait()
	})

	t.Run("different zone ID", func(t *testing.T) {
		var errors []error
		responses := manager.GetZoneRRs(zones[1].ID, app.Daemons[0].ID, "_default")
		for response := range responses {
			errors = append(errors, response.Err)
		}
		require.Len(t, errors, 1)
		require.NoError(t, errors[0])
	})

	t.Run("different daemon ID", func(t *testing.T) {
		var errors []error
		responses := manager.GetZoneRRs(zones[0].ID, app.Daemons[1].ID, "_default")
		for response := range responses {
			errors = append(errors, response.Err)
		}
		require.Len(t, errors, 1)
		require.NoError(t, errors[0])
	})

	t.Run("different view name", func(t *testing.T) {
		var errors []error
		responses := manager.GetZoneRRs(zones[0].ID, app.Daemons[0].ID, "trusted")
		for response := range responses {
			errors = append(errors, response.Err)
		}
		require.Len(t, errors, 1)
		require.NoError(t, errors[0])
	})
}

// Test that RRs are not cached when a caller stops reading them before
// all of them are received.
func TestZoneRRsCacheWithEarlyReturn(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create a machine and an app. The manager will determine the app
	// to contact based on the daemon ID.
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

	// Add a zone. We will use zone ID to fetch the RRs.
	zone := &dbmodel.Zone{
		ID:    1,
		Name:  "example.com",
		Rname: "com.example",
		LocalZones: []*dbmodel.LocalZone{
			{
				ID:       1,
				View:     "_default",
				DaemonID: app.Daemons[0].ID,
				Class:    "IN",
				Type:     "primary",
				Serial:   1,
				LoadedAt: time.Now().UTC(),
			},
		},
	}
	err = dbmodel.AddZones(db, []*dbmodel.Zone{zone}...)
	require.NoError(t, err)

	// Read the RRs to the returned by the agent from the file.
	var rrs []string
	err = json.Unmarshal(validZoneData, &rrs)
	require.NoError(t, err)

	// Return the RRs using the mock.
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.App, string, string) iter.Seq2[[]*dnsconfig.RR, error] {
		return func(yield func([]*dnsconfig.RR, error) bool) {
			for _, rr := range rrs {
				rr, err := dnsconfig.NewRR(rr)
				require.NoError(t, err)
				if !yield([]*dnsconfig.RR{rr}, nil) {
					return
				}
			}
		}
	})

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	// Start reading the RRs.
	rrResponses := manager.GetZoneRRs(zone.ID, app.Daemons[0].ID, "_default")

	// Read only the first RR and stop the iterator.
	next, stop := iter.Pull(rrResponses)
	rrResponse, exists := next()
	stop()

	// Make sure the RR was read properly.
	require.True(t, exists)
	require.NotNil(t, rrResponse)
	require.False(t, rrResponse.Cached)
	require.NoError(t, rrResponse.Err)

	// Make sure that there are no RRs in the database.
	localZoneRRs, err := dbmodel.GetDNSConfigRRs(db, zone.LocalZones[0].ID)
	require.NoError(t, err)
	require.Empty(t, localZoneRRs)
}

// Test that a database error is propagated while iterating over the RRs.
func TestZoneRRsCacheDatabaseError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create a machine and an app. The manager will determine the app
	// to contact based on the daemon ID.
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

	// Add a zone. We will use zone ID to fetch the RRs.
	zone := &dbmodel.Zone{
		ID:    1,
		Name:  "example.com",
		Rname: "com.example",
		LocalZones: []*dbmodel.LocalZone{
			{
				ID:       1,
				View:     "_default",
				DaemonID: app.Daemons[0].ID,
				Class:    "IN",
				Type:     "primary",
				Serial:   1,
				LoadedAt: time.Now().UTC(),
			},
		},
	}
	err = dbmodel.AddZones(db, []*dbmodel.Zone{zone}...)
	require.NoError(t, err)

	// Read the RRs to the returned by the agent from the file.
	var rrs []string
	err = json.Unmarshal(validZoneData, &rrs)
	require.NoError(t, err)

	// Return the RRs using the mock.
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(a any) bool {
		return a.(*dbmodel.App).ID == app.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.App, string, string) iter.Seq2[[]*dnsconfig.RR, error] {
		return func(yield func([]*dnsconfig.RR, error) bool) {
			for _, rr := range rrs {
				rr, err := dnsconfig.NewRR(rr)
				require.NoError(t, err)
				if !yield([]*dnsconfig.RR{rr}, nil) {
					return
				}
			}
		}
	})

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	// Start reading the RRs.
	rrResponses := manager.GetZoneRRs(zone.ID, app.Daemons[0].ID, "_default")

	next, stop := iter.Pull(rrResponses)
	defer stop()

	// Get the first RR successfully.
	rrResponse, exists := next()
	require.True(t, exists)
	require.NotNil(t, rrResponse)
	require.NoError(t, rrResponse.Err)

	// Shutdown the database connection to cause an error for subsequent RRs.
	teardown()

	// Iterate over the RRs and expect an error to be returned.
	var capturedErr error
	for {
		rrResponse, exists := next()
		if !exists {
			break
		}
		// The error may not be returned immediately because we're inserting the
		// records using a batch.
		capturedErr = rrResponse.Err
	}
	require.Error(t, capturedErr)
	require.ErrorContains(t, capturedErr, "failed to flush the batch of RRs")
}
