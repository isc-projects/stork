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
	agentapi "isc.org/stork/api"
	bind9config "isc.org/stork/daemoncfg/bind9"
	"isc.org/stork/datamodel/daemonname"
	dnsmodel "isc.org/stork/datamodel/dns"
	agentcomm "isc.org/stork/server/agentcomm"
	appstest "isc.org/stork/server/daemons/test"
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
	rr, err := dnsmodel.NewRR("example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 123456 7200 3600 1209600 3600")
	require.NoError(t, err)
	rrs := []*dnsmodel.RR{
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
	rr, err := dnsmodel.NewRR("example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 123456 7200 3600 1209600 3600")
	require.NoError(t, err)
	rrs := []*dnsmodel.RR{
		rr,
	}
	timestamp := time.Now().UTC()
	rrResponse := NewCacheRRResponse(rrs, 2, timestamp)
	require.True(t, rrResponse.Cached)
	require.Equal(t, rrs, rrResponse.RRs)
	require.Equal(t, 2, rrResponse.Total)
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

// Test that filtering the RRs works properly.
func TestRRResponseApplyFilter(t *testing.T) {
	rrStrings := []string{
		"example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 123456 7200 3600 1209600 3600",
		"example.com. 3600 IN A 192.0.2.1",
		"ipv6.example.com. 3600 IN AAAA 2001:db8::1",
		"ipv4.example.com. 3600 IN A 192.0.2.2",
		"any.example.org. 3600 IN AAAA 2001:db8::2",
		"any.example.org. 3600 IN A 192.0.2.3",
	}
	rrs := []*dnsmodel.RR{}
	for _, rrString := range rrStrings {
		rr, err := dnsmodel.NewRR(rrString)
		require.NoError(t, err)
		rrs = append(rrs, rr)
	}
	rrResponse := NewZoneTransferRRResponse(rrs)
	t.Run("no filter", func(t *testing.T) {
		filteredResponse, pos := rrResponse.applyFilter(nil, 0)
		require.Equal(t, 6, filteredResponse.Total)
		require.Len(t, filteredResponse.RRs, 6)
		require.Equal(t, 6, pos)
	})

	t.Run("filter by type", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.EnableType("A")
		filteredResponse, pos := rrResponse.applyFilter(filter, 0)
		require.Equal(t, 3, filteredResponse.Total)
		require.Len(t, filteredResponse.RRs, 3)
		require.Equal(t, 3, pos)
	})

	t.Run("filter by name text", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.SetText("example.com")
		filteredResponse, pos := rrResponse.applyFilter(filter, 0)
		require.Equal(t, 4, filteredResponse.Total)
		require.Len(t, filteredResponse.RRs, 4)
		require.Equal(t, 4, pos)
	})

	t.Run("filter by rdata text", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.SetText("2001:db8:")
		filteredResponse, pos := rrResponse.applyFilter(filter, 0)
		require.Equal(t, 2, filteredResponse.Total)
		require.Len(t, filteredResponse.RRs, 2)
		require.Equal(t, 2, pos)
	})

	t.Run("filter by offset no limit", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.SetOffset(2)
		filteredResponse, pos := rrResponse.applyFilter(filter, 0)
		require.Equal(t, 6, filteredResponse.Total)
		require.Len(t, filteredResponse.RRs, 4)
		require.Equal(t, 6, pos)
	})

	t.Run("filter limit no offset", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.SetLimit(2)
		filteredResponse, pos := rrResponse.applyFilter(filter, 0)
		require.Equal(t, 6, filteredResponse.Total)
		require.Len(t, filteredResponse.RRs, 2)
		require.Equal(t, 6, pos)
	})

	t.Run("filter offset before pos", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.SetOffset(2)
		filteredResponse, pos := rrResponse.applyFilter(filter, 4)
		require.Equal(t, 10, filteredResponse.Total)
		require.Len(t, filteredResponse.RRs, 6)
		require.Equal(t, 10, pos)
	})

	t.Run("filter offset before pos limit after new pos", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.SetOffset(2)
		filter.SetLimit(10)
		filteredResponse, pos := rrResponse.applyFilter(filter, 4)
		require.Equal(t, 10, filteredResponse.Total)
		require.Len(t, filteredResponse.RRs, 6)
		require.Equal(t, 10, pos)
	})

	t.Run("filter offset and limit after new pos", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.SetOffset(8)
		filter.SetLimit(10)
		filteredResponse, pos := rrResponse.applyFilter(filter, 0)
		require.Equal(t, 6, filteredResponse.Total)
		require.Empty(t, filteredResponse.RRs)
		require.Equal(t, 6, pos)
	})

	t.Run("filter offset and limit in range", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.SetOffset(2)
		filter.SetLimit(2)
		filteredResponse, pos := rrResponse.applyFilter(filter, 2)
		require.Equal(t, 8, filteredResponse.Total)
		require.Len(t, filteredResponse.RRs, 2)
		require.Equal(t, 8, pos)
	})

	t.Run("filter offset and limit empty range", func(t *testing.T) {
		filter := dbmodel.NewGetZoneRRsFilter()
		filter.SetText("non-existent.com")
		filter.SetOffset(2)
		filter.SetLimit(2)
		filteredResponse, pos := rrResponse.applyFilter(filter, 4)
		require.Equal(t, 4, filteredResponse.Total)
		require.Empty(t, filteredResponse.RRs)
		require.Equal(t, 4, pos)
	})
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

// Test an error indicating that the manager is already requesting BIND 9 configuration
// for the same daemon.
func TestManagerBind9FormattedConfigAlreadyRequestedError(t *testing.T) {
	err := NewManagerBind9FormattedConfigAlreadyRequestedError()
	require.ErrorContains(t, err, "BIND 9 configuration for the specified daemon has been already requested by another user")
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

	_, err = manager.FetchZones(10, 1000, FetchZonesOptionBlock)
	require.Error(t, err)
	require.ErrorContains(t, err, "problem selecting daemons with names: [named pdns]")
}

// This test verifies that "busy" status is set in the database when zone
// inventory is busy while fetching zones.
func TestFetchZonesInventoryBusyError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Add a machine and daemon.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// Return "busy" error on first iteration.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), nil, false).DoAndReturn(func(context.Context, *dbmodel.Daemon, *dnsmodel.ZoneFilter, bool) iter.Seq2[*dnsmodel.ExtendedZone, error] {
		return func(yield func(*dnsmodel.ExtendedZone, error) bool) {
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
	notifyChannel, err := manager.FetchZones(10, 100, FetchZonesOptionBlock)
	require.NoError(t, err)
	notification := <-notifyChannel

	// Make sure the "busy" error was reported.
	require.Len(t, notification.results, 1)
	require.NotNil(t, notification.results[daemon.ID].Error)
	require.Contains(t, *notification.results[daemon.ID].Error, "Zone inventory is temporarily busy on the agent foo")

	// The database should also hold the fetch result.
	state, err := dbmodel.GetZoneInventoryState(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, daemon.ID, state.DaemonID)
	require.NotZero(t, state.CreatedAt)
	require.NotNil(t, state.State.Error)
	require.Contains(t, *state.State.Error, "Zone inventory is temporarily busy on the agent foo")
	require.Equal(t, dbmodel.ZoneInventoryStatusBusy, state.State.Status)
	require.Nil(t, state.State.ZoneCount)

	// Make sure that no zones have been added to the database.
	zones, total, err := dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
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

	// Add a machine and daemon.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// Return "uninitialized" error on first iteration.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), nil, false).DoAndReturn(func(context.Context, *dbmodel.Daemon, *dnsmodel.ZoneFilter, bool) iter.Seq2[*dnsmodel.ExtendedZone, error] {
		return func(yield func(*dnsmodel.ExtendedZone, error) bool) {
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

	notifyChannel, err := manager.FetchZones(10, 100, FetchZonesOptionBlock)
	require.NoError(t, err)
	notification := <-notifyChannel

	// Make sure the "busy" error was reported.
	require.Len(t, notification.results, 1)
	require.NotNil(t, notification.results[daemon.ID].Error)
	require.Contains(t, *notification.results[daemon.ID].Error, "DNS zones have not been loaded on the agent foo")

	// The database should also hold the fetch result.
	state, err := dbmodel.GetZoneInventoryState(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, daemon.ID, state.DaemonID)
	require.NotZero(t, state.CreatedAt)
	require.NotNil(t, state.State.Error)
	require.Contains(t, *state.State.Error, "DNS zones have not been loaded on the agent foo")
	require.Equal(t, dbmodel.ZoneInventoryStatusUninitialized, state.State.Status)
	require.Nil(t, state.State.ZoneCount)

	// Make sure that no zones have been added to the database.
	zones, total, err := dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
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

	// Add a machine and daemon.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// Return "uninitialized" error on first iteration.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), nil, false).DoAndReturn(func(context.Context, *dbmodel.Daemon, *dnsmodel.ZoneFilter, bool) iter.Seq2[*dnsmodel.ExtendedZone, error] {
		return func(yield func(*dnsmodel.ExtendedZone, error) bool) {
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

	notifyChannel, err := manager.FetchZones(10, 100, FetchZonesOptionBlock)
	require.NoError(t, err)
	notification := <-notifyChannel

	// Make sure other error was reported.
	require.Len(t, notification.results, 1)
	require.NotNil(t, notification.results[daemon.ID].Error)
	require.Contains(t, *notification.results[daemon.ID].Error, "test error")

	// The database should also hold the fetch result.
	state, err := dbmodel.GetZoneInventoryState(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, daemon.ID, state.DaemonID)
	require.NotZero(t, state.CreatedAt)
	require.NotNil(t, state.State.Error)
	require.Contains(t, *state.State.Error, "test error")
	require.Equal(t, dbmodel.ZoneInventoryStatusErred, state.State.Status)
	require.Nil(t, state.State.ZoneCount)

	// Make sure that no zones have been added to the database.
	zones, total, err := dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
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

	// Add a machine and daemon.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// Return "uninitialized" error on first iteration.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), nil, false).DoAndReturn(func(context.Context, *dbmodel.Daemon, *dnsmodel.ZoneFilter, bool) iter.Seq2[*dnsmodel.ExtendedZone, error] {
		return func(yield func(*dnsmodel.ExtendedZone, error) bool) {
			// We are on the fist iteration. Let's close the database connection
			// to cause an error.
			teardown()
			// Return the zone.
			zone := &dnsmodel.ExtendedZone{
				Zone: dnsmodel.Zone{
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

	notifyChannel, err := manager.FetchZones(10, 100, FetchZonesOptionBlock)
	require.NoError(t, err)
	notification := <-notifyChannel

	// Make sure other error was reported.
	require.Len(t, notification.results, 1)
	require.NotNil(t, notification.results[daemon.ID].Error)
	require.Contains(t, *notification.results[daemon.ID].Error, "database is closed")
}

// This test verifies that the manager can fetch zones from many servers
// simultaneously.
func TestFetchZones(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	randomZones := testutil.GenerateRandomZones(1000)

	// Generate many machines and daemons.
	for i := 0; i < 100; i++ {
		machine := &dbmodel.Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := dbmodel.AddMachine(db, machine)
		require.NoError(t, err)

		daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    953,
			},
		})
		err = dbmodel.AddDaemon(db, daemon)
		require.NoError(t, err)

		// We're going to test a corner case all of the servers have exactly the
		// same set of zones. This is unrealistic scenario but it well stresses
		// the code generating many conflicts in the database. We want to make
		// sure that no zone is lost due to conflicts.
		mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(d any) bool {
			return d.(*dbmodel.Daemon).ID == daemon.ID
		}), nil, false).DoAndReturn(func(context.Context, *dbmodel.Daemon, *dnsmodel.ZoneFilter, bool) iter.Seq2[*dnsmodel.ExtendedZone, error] {
			return func(yield func(*dnsmodel.ExtendedZone, error) bool) {
				for _, zone := range randomZones {
					zone := &dnsmodel.ExtendedZone{
						Zone: dnsmodel.Zone{
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
	notifyChannel, err := manager.FetchZones(10, 100, FetchZonesOptionBlock)
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
	zones, total, err := dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
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

// This test verifies that the manager can request the zone inventory to be populated
// from the DNS server before fetching the zones.
func TestFetchZonesForcePopulate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	randomZones := testutil.GenerateRandomZones(1000)

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// Expact that the forcePopulate boolean flag is set to true.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), nil, true).DoAndReturn(func(context.Context, *dbmodel.Daemon, *dnsmodel.ZoneFilter, bool) iter.Seq2[*dnsmodel.ExtendedZone, error] {
		return func(yield func(*dnsmodel.ExtendedZone, error) bool) {
			for _, zone := range randomZones {
				zone := &dnsmodel.ExtendedZone{
					Zone: dnsmodel.Zone{
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

	// Begin the fetch.
	notifyChannel, err := manager.FetchZones(1, 1000, FetchZonesOptionBlock, FetchZonesOptionForcePopulate)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)

	// All zones should be in the database.
	zones, _, err := dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zones, 1000)

	// Complete the fetch.
	<-notifyChannel
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return !isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)
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

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Return an empty iterator. Getting actual zones is not in scope for this test.
	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Any(), nil, false).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.Daemon, *dnsmodel.ZoneFilter, bool) iter.Seq2[*dnsmodel.ExtendedZone, error] {
		return func(yield func(*dnsmodel.ExtendedZone, error) bool) {
			for _, zone := range randomZones {
				zone := &dnsmodel.ExtendedZone{
					Zone: dnsmodel.Zone{
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
	notifyChannel, err := manager.FetchZones(1, 1000, FetchZonesOptionBlock)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)

	// All zones should be in the database.
	zones, _, err := dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zones, 1000)

	// Reduce the number of returned zones to 100. Remaining zones
	// should be removed from the database.
	randomZones = randomZones[:100]

	// Begin the second fetch. It should return an error.
	_, err = manager.FetchZones(10, 1000, FetchZonesOptionBlock)
	var alreadyFetching *ManagerAlreadyFetchingError
	require.ErrorAs(t, err, &alreadyFetching)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)

	// The zones should remain untouched.
	zones, _, err = dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zones, 1000)

	// Complete the fetch.
	<-notifyChannel
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return !isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)

	// All zones should be in the database.
	zones, _, err = dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zones, 1000)

	// This time the new attempt should succeed.
	notifyChannel, err = manager.FetchZones(10, 1000, FetchZonesOptionBlock)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		isFetching, appsNum, completedAppsNum := manager.GetFetchZonesProgress()
		return isFetching && appsNum == 1 && completedAppsNum == 1
	}, 5*time.Second, 10*time.Millisecond)

	// Complete the fetch.
	<-notifyChannel

	// This time we should have only 100 zones and all other zones should be removed.
	zones, _, err = dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
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

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	mock.EXPECT().ReceiveZones(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), nil, false).DoAndReturn(func(context.Context, *dbmodel.Daemon, *dnsmodel.ZoneFilter, bool) iter.Seq2[*dnsmodel.ExtendedZone, error] {
		return func(yield func(*dnsmodel.ExtendedZone, error) bool) {
			// Return the same zones from two different views.
			for _, view := range []string{"foo", "bar"} {
				for _, zone := range randomZones {
					zone := &dnsmodel.ExtendedZone{
						Zone: dnsmodel.Zone{
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
	notifyChannel, err := manager.FetchZones(1, 100, FetchZonesOptionBlock)
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
	zones, total, err := dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.EqualValues(t, 20, total)
	require.Len(t, zones, 20)

	// Make sure that all zones have two associations.
	for _, zone := range zones {
		require.Len(t, zone.LocalZones, 2)
	}
}

// Test successfully receiving the zone RRs from the agents with filtering.
func TestGetZoneRRsWithFiltering(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create a machine and a daemon. The manager will determine the daemon
	// to contact based on the daemon ID.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
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
				DaemonID: daemon.ID,
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

	var filteredRRs []string
	for _, rr := range rrs {
		parsedRR, err := dnsmodel.NewRR(rr)
		require.NoError(t, err)
		if parsedRR.Type == "A" {
			filteredRRs = append(filteredRRs, rr)
		}
	}

	// Return the RRs using the mock.
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.Daemon, string, string) iter.Seq2[[]*dnsmodel.RR, error] {
		return func(yield func([]*dnsmodel.RR, error) bool) {
			for _, rr := range rrs {
				rr, err := dnsmodel.NewRR(rr)
				require.NoError(t, err)
				if !yield([]*dnsmodel.RR{rr}, nil) {
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
	collectedRRs := []*dnsmodel.RR{}
	filter := dbmodel.NewGetZoneRRsFilter()
	filter.EnableType("A")
	filter.SetOffset(1)
	filter.SetLimit(4)
	rrResponses := manager.GetZoneRRs(zone.ID, daemon.ID, "_default", filter)
	for rrResponse := range rrResponses {
		require.False(t, rrResponse.Cached)
		require.InDelta(t, time.Now().UTC().Unix(), rrResponse.ZoneTransferAt.Unix(), 4)
		require.NoError(t, rrResponse.Err)
		collectedRRs = append(collectedRRs, rrResponse.RRs...)
	}
	// Validate the returned RRs against the original ones.
	require.Equal(t, len(filteredRRs[1:]), len(collectedRRs))
	for i, rr := range collectedRRs {
		// Replace tabs with spaces in the original RR.
		original := strings.Join(strings.Fields(filteredRRs[i+1]), " ")
		require.Equal(t, original, rr.GetString())
	}
	// Make sure that the timestamp was set.
	zone, err = dbmodel.GetZoneByID(db, zone.ID, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zone.LocalZones, 1)
	require.NotNil(t, zone.LocalZones[0].ZoneTransferAt)

	// Get RRs again without filtering. It should return all RRs.
	collectedRRs = make([]*dnsmodel.RR, 0, len(rrs))
	rrResponses = manager.GetZoneRRs(zone.ID, daemon.ID, "_default", nil)
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
	// Finally, get RRs again with filtering from the database.
	collectedRRs = make([]*dnsmodel.RR, 0, len(filteredRRs))
	rrResponses = manager.GetZoneRRs(zone.ID, daemon.ID, "_default", filter)
	for rrResponse := range rrResponses {
		require.True(t, rrResponse.Cached)
		require.InDelta(t, time.Now().UTC().Unix(), rrResponse.ZoneTransferAt.Unix(), 5)
		require.NoError(t, rrResponse.Err)
		collectedRRs = append(collectedRRs, rrResponse.RRs...)
	}
	// Validate the returned RRs against the original ones.
	require.Equal(t, len(filteredRRs[1:]), len(collectedRRs))
	for i, rr := range collectedRRs {
		// Replace tabs with spaces in the original RR.
		original := strings.Join(strings.Fields(filteredRRs[i+1]), " ")
		require.Equal(t, original, rr.GetString())
	}
}

// Test successfully receiving the zone RRs from the agents.
func TestGetZoneRRs(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create a machine and a daemon. The manager will determine the daemon
	// to contact based on the daemon ID.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
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
				DaemonID: daemon.ID,
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
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.Daemon, string, string) iter.Seq2[[]*dnsmodel.RR, error] {
		return func(yield func([]*dnsmodel.RR, error) bool) {
			for _, rr := range rrs {
				rr, err := dnsmodel.NewRR(rr)
				require.NoError(t, err)
				if !yield([]*dnsmodel.RR{rr}, nil) {
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
	collectedRRs := make([]*dnsmodel.RR, 0, len(rrs))
	rrResponses := manager.GetZoneRRs(zone.ID, daemon.ID, "_default", nil)
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
	zone, err = dbmodel.GetZoneByID(db, zone.ID, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zone.LocalZones, 1)
	require.NotNil(t, zone.LocalZones[0].ZoneTransferAt)
	lastRRsTransferAt := *zone.LocalZones[0].ZoneTransferAt

	// Get RRs again. It should return cached RRs.
	collectedRRs = make([]*dnsmodel.RR, 0, len(rrs))
	rrResponses = manager.GetZoneRRs(zone.ID, daemon.ID, "_default", nil)
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
	zone, err = dbmodel.GetZoneByID(db, zone.ID, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zone.LocalZones, 1)
	require.Equal(t, lastRRsTransferAt, *zone.LocalZones[0].ZoneTransferAt)

	// Finally, get RRs again with forcing zone transfer.
	collectedRRs = make([]*dnsmodel.RR, 0, len(rrs))
	rrResponses = manager.GetZoneRRs(zone.ID, daemon.ID, "_default", nil, GetZoneRRsOptionForceZoneTransfer)
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
	zone, err = dbmodel.GetZoneByID(db, zone.ID, dbmodel.ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, zone.LocalZones, 1)
	require.NotNil(t, zone.LocalZones[0].ZoneTransferAt)
	require.Greater(t, *zone.LocalZones[0].ZoneTransferAt, lastRRsTransferAt)
}

// Test successfully receiving the zone RRs from the agents
// and excluding the trailing SOA RR.
func TestGetZoneRRsExcludeTrailingSOA(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create a machine and a daemon. The manager will determine the daemon
	// to contact based on the daemon ID.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
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
				DaemonID: daemon.ID,
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
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.Daemon, string, string) iter.Seq2[[]*dnsmodel.RR, error] {
		return func(yield func([]*dnsmodel.RR, error) bool) {
			for _, rr := range rrs {
				rr, err := dnsmodel.NewRR(rr)
				require.NoError(t, err)
				if !yield([]*dnsmodel.RR{rr}, nil) {
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
	collectedRRs := make([]*dnsmodel.RR, 0, len(rrs))
	rrResponses := manager.GetZoneRRs(zone.ID, daemon.ID, "_default", nil, GetZoneRRsOptionExcludeTrailingSOA)
	for rrResponse := range rrResponses {
		require.False(t, rrResponse.Cached)
		//		require.InDelta(t, time.Now().UTC().Unix(), rrResponse.ZoneTransferAt.Unix(), 5)
		require.NoError(t, rrResponse.Err)
		collectedRRs = append(collectedRRs, rrResponse.RRs...)
	}
	// Validate the returned RRs against the original ones.
	require.Equal(t, len(rrs)-1, len(collectedRRs))
	for i, rr := range collectedRRs {
		// Replace tabs with spaces in the original RR.
		original := strings.Join(strings.Fields(rrs[i]), " ")
		require.Equal(t, original, rr.GetString())
	}
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
	responses := manager.GetZoneRRs(12, 13, "_default", nil)
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

	// Create the daemon to ensure that the manager will not fail on
	// trying to get the daemon from the database.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
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
	responses := manager.GetZoneRRs(12, daemon.ID, "_default", nil)
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

	// Create the daemon.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// Create the zone and associated with the daemon.
	zone := &dbmodel.Zone{
		ID:    1,
		Name:  "example.com",
		Rname: "com.example",
		LocalZones: []*dbmodel.LocalZone{
			{
				ID:       1,
				View:     "_default",
				DaemonID: daemon.ID,
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
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.Daemon, string, string) iter.Seq2[[]*dnsmodel.RR, error] {
		return func(yield func([]*dnsmodel.RR, error) bool) {
			// Signalling here that the second request can start.
			wg1.Done()
			// Wait for the test to unpause the mock.
			wg2.Wait()
			// Return an empty result. It doesn't really matter what is returned.
			yield([]*dnsmodel.RR{}, nil)
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
		responses := manager.GetZoneRRs(zone.ID, daemon.ID, "_default", nil)
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
	responses := manager.GetZoneRRs(zone.ID, daemon.ID, "_default", nil)
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

	// Create the machine and daemons.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon1 := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon1)
	require.NoError(t, err)

	daemon2 := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    954,
		},
	})
	err = dbmodel.AddDaemon(db, daemon2)
	require.NoError(t, err)

	// Create two zones associated with the daemons.
	zones := []*dbmodel.Zone{
		{
			Name:  "example.com",
			Rname: "com.example",
			LocalZones: []*dbmodel.LocalZone{
				{
					View:     "_default",
					DaemonID: daemon1.ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
				{
					View:     "trusted",
					DaemonID: daemon1.ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
				{
					View:     "_default",
					DaemonID: daemon2.ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
				{
					View:     "trusted",
					DaemonID: daemon2.ID,
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
					DaemonID: daemon1.ID,
					Class:    "IN",
					Type:     "primary",
					Serial:   1,
					LoadedAt: time.Now().UTC(),
				},
				{
					View:     "_default",
					DaemonID: daemon2.ID,
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
	mocks = append(mocks, mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon1.ID
	}), gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *dbmodel.Daemon, string, string) iter.Seq2[[]*dnsmodel.RR, error] {
		return func(yield func([]*dnsmodel.RR, error) bool) {
			// Signalling here that the second request can start.
			wg1.Done()
			// Wait for the test to unpause the mock.
			wg2.Wait()
			// Return an empty result. It doesn't really matter what is returned.
			yield([]*dnsmodel.RR{}, nil)
		}
	}))
	mocks = append(mocks, mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon1.ID || d.(*dbmodel.Daemon).ID == daemon2.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.Daemon, string, string) iter.Seq2[[]*dnsmodel.RR, error] {
		return func(yield func([]*dnsmodel.RR, error) bool) {
			yield([]*dnsmodel.RR{}, nil)
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
		responses := manager.GetZoneRRs(zones[0].ID, daemon1.ID, "_default", nil)
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
		responses := manager.GetZoneRRs(zones[1].ID, daemon1.ID, "_default", nil)
		for response := range responses {
			errors = append(errors, response.Err)
		}
		require.Len(t, errors, 1)
		require.NoError(t, errors[0])
	})

	t.Run("different daemon ID", func(t *testing.T) {
		var errors []error
		responses := manager.GetZoneRRs(zones[0].ID, daemon2.ID, "_default", nil)
		for response := range responses {
			errors = append(errors, response.Err)
		}
		require.Len(t, errors, 1)
		require.NoError(t, errors[0])
	})

	t.Run("different view name", func(t *testing.T) {
		var errors []error
		responses := manager.GetZoneRRs(zones[0].ID, daemon1.ID, "trusted", nil)
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

	// Create a machine and a daemon. The manager will determine the daemon
	// to contact based on the daemon ID.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
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
				DaemonID: daemon.ID,
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
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.Daemon, string, string) iter.Seq2[[]*dnsmodel.RR, error] {
		return func(yield func([]*dnsmodel.RR, error) bool) {
			for _, rr := range rrs {
				rr, err := dnsmodel.NewRR(rr)
				require.NoError(t, err)
				if !yield([]*dnsmodel.RR{rr}, nil) {
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
	rrResponses := manager.GetZoneRRs(zone.ID, daemon.ID, "_default", nil)

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
	localZoneRRs, total, err := dbmodel.GetDNSConfigRRs(db, zone.LocalZones[0].ID, nil)
	require.NoError(t, err)
	require.Empty(t, localZoneRRs)
	require.Zero(t, total)
}

// Test that a database error is propagated while iterating over the RRs.
func TestZoneRRsCacheDatabaseError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create a machine and a daemon. The manager will determine the daemon
	// to contact based on the daemon ID.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    953,
		},
	})
	err = dbmodel.AddDaemon(db, daemon)
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
				DaemonID: daemon.ID,
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
	mock.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(context.Context, *dbmodel.Daemon, string, string) iter.Seq2[[]*dnsmodel.RR, error] {
		return func(yield func([]*dnsmodel.RR, error) bool) {
			for _, rr := range rrs {
				rr, err := dnsmodel.NewRR(rr)
				require.NoError(t, err)
				if !yield([]*dnsmodel.RR{rr}, nil) {
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
	rrResponses := manager.GetZoneRRs(zone.ID, daemon.ID, "_default", nil)

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

// Test that one BIND 9 configuration file is returned from the agent.
func TestGetBind9FormattedConfig(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, false, []*dbmodel.AccessPoint{})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	mock.EXPECT().ReceiveBind9FormattedConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(func(ctx context.Context, daemon *dbmodel.Daemon, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) iter.Seq2[*agentapi.ReceiveBind9ConfigRsp, error] {
			require.NotNil(t, fileSelector)
			require.NotNil(t, filter)
			require.True(t, filter.IsEnabled(bind9config.FilterTypeConfig))
			require.True(t, filter.IsEnabled(bind9config.FilterTypeView))
			require.False(t, filter.IsEnabled(bind9config.FilterTypeZone))
			require.True(t, fileSelector.IsEnabled(bind9config.FileTypeConfig))
			require.False(t, fileSelector.IsEnabled(bind9config.FileTypeRndcKey))
			return func(yield func(*agentapi.ReceiveBind9ConfigRsp, error) bool) {
				responses := []*agentapi.ReceiveBind9ConfigRsp{
					{
						Response: &agentapi.ReceiveBind9ConfigRsp_File{
							File: &agentapi.ReceiveBind9ConfigFile{
								FileType: agentapi.Bind9ConfigFileType_CONFIG,
							},
						},
					},
					{
						Response: &agentapi.ReceiveBind9ConfigRsp_Line{
							Line: "config;",
						},
					},
					{
						Response: &agentapi.ReceiveBind9ConfigRsp_Line{
							Line: "view;",
						},
					},
				}
				for _, response := range responses {
					if !yield(response, nil) {
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

	// Filter config only.
	next, cancel := iter.Pull(
		manager.GetBind9FormattedConfig(
			context.Background(),
			daemon.ID,
			bind9config.NewFileTypeSelector(bind9config.FileTypeConfig),
			bind9config.NewFilter(bind9config.FilterTypeConfig, bind9config.FilterTypeView),
		),
	)
	defer cancel()
	rsp, ok := next()
	require.True(t, ok)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.File)
	require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, rsp.File.FileType)

	rsp, ok = next()
	require.True(t, ok)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Contents)
	require.Equal(t, `config;`, *rsp.Contents)

	rsp, ok = next()
	require.True(t, ok)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Contents)
	require.Equal(t, `view;`, *rsp.Contents)

	_, ok = next()
	require.False(t, ok)
}

// Test that multiple files are returned when getting BIND 9 configuration
// from the agent.
func TestGetBind9FormattedConfigMultipleFiles(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// Depending on the filter the mock returns different configurations.
	// We can use the output to determine that the filter is applied correctly.
	mock.EXPECT().ReceiveBind9FormattedConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(
			func(ctx context.Context, daemon *dbmodel.Daemon, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) iter.Seq2[*agentapi.ReceiveBind9ConfigRsp, error] {
				return func(yield func(*agentapi.ReceiveBind9ConfigRsp, error) bool) {
					responses := []*agentapi.ReceiveBind9ConfigRsp{
						{
							Response: &agentapi.ReceiveBind9ConfigRsp_File{
								File: &agentapi.ReceiveBind9ConfigFile{
									FileType:   agentapi.Bind9ConfigFileType_CONFIG,
									SourcePath: "config.conf",
								},
							},
						},
						{
							Response: &agentapi.ReceiveBind9ConfigRsp_Line{
								Line: "config;",
							},
						},
						{
							Response: &agentapi.ReceiveBind9ConfigRsp_File{
								File: &agentapi.ReceiveBind9ConfigFile{
									FileType:   agentapi.Bind9ConfigFileType_RNDC_KEY,
									SourcePath: "rndc.key",
								},
							},
						},
						{
							Response: &agentapi.ReceiveBind9ConfigRsp_Line{
								Line: "rndc-key;",
							},
						},
					}
					for _, response := range responses {
						if !yield(response, nil) {
							return
						}
					}
				}
			},
		)

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)

	next, cancel := iter.Pull(
		manager.GetBind9FormattedConfig(
			context.Background(),
			daemon.ID,
			bind9config.NewFileTypeSelector(bind9config.FileTypeConfig, bind9config.FileTypeRndcKey),
			nil),
	)
	defer cancel()
	rsp, ok := next()
	require.True(t, ok)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.File)
	require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, rsp.File.FileType)

	rsp, ok = next()
	require.True(t, ok)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Contents)
	require.Equal(t, `config;`, *rsp.Contents)

	rsp, ok = next()
	require.True(t, ok)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.File)
	require.Equal(t, agentapi.Bind9ConfigFileType_RNDC_KEY, rsp.File.FileType)

	rsp, ok = next()
	require.True(t, ok)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Contents)
	require.Equal(t, `rndc-key;`, *rsp.Contents)

	_, ok = next()
	require.False(t, ok)
}

// Test that an error is returned when getting BIND 9 configuration from a
// non-existing daemon.
func TestGetBind9FormattedConfigNoDaemon(t *testing.T) {
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

	next, cancel := iter.Pull(manager.GetBind9FormattedConfig(context.Background(), int64(1), nil, nil))
	defer cancel()
	rsp, ok := next()
	require.True(t, ok)
	require.ErrorContains(t, rsp.Err, "unable to get BIND 9 configuration from non-existent daemon with the ID 1")
}

// Test that an error is returned when getting BIND 9 daemon from the
// database fails.
func TestGetBind9FormattedConfigError(t *testing.T) {
	// Setup the database and tear it down immediately to cause an error.
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)

	next, cancel := iter.Pull(manager.GetBind9FormattedConfig(context.Background(), int64(1), nil, nil))
	defer cancel()
	rsp, ok := next()
	require.True(t, ok)
	require.ErrorContains(t, rsp.Err, "problem getting daemon 1")
}

// Test that an error is returned when getting BIND 9 configuration from the
// agent returns a nil response.
func TestGetBind9FormattedConfigNilResponse(t *testing.T) {
	// Setup the database and tear it down immediately to cause an error.
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	mock.EXPECT().ReceiveBind9FormattedConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(
			func(ctx context.Context, daemon *dbmodel.Daemon, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) iter.Seq2[*agentapi.ReceiveBind9ConfigRsp, error] {
				return func(yield func(*agentapi.ReceiveBind9ConfigRsp, error) bool) {
					yield(nil, nil)
				}
			})

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)

	next, cancel := iter.Pull(manager.GetBind9FormattedConfig(context.Background(), int64(1), nil, nil))
	defer cancel()
	rsp, ok := next()
	require.True(t, ok)
	require.ErrorContains(t, rsp.Err, "unexpected nil response while getting BIND 9 configuration from the agent")
}

// Test that an error is returned if another request for getting
// BIND 9 configuration from the same daemon is in progress.
func TestGetBind9FormattedConfigAnotherRequestInProgress(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create the daemon.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// We need to run first request and ensure it stops, so we can
	// run another request before it completes. The first synchronization
	// group will be used to wait for the mock to start. The other one will
	// pause it while we perform the second request.
	wg1 := sync.WaitGroup{}
	wg2 := sync.WaitGroup{}
	wg1.Add(1)
	wg2.Add(1)
	mock.EXPECT().ReceiveBind9FormattedConfig(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(
			func(context.Context, *dbmodel.Daemon, *bind9config.FileTypeSelector, *bind9config.Filter) iter.Seq2[*agentapi.ReceiveBind9ConfigRsp, error] {
				return func(yield func(*agentapi.ReceiveBind9ConfigRsp, error) bool) {
					// Signalling here that the second request can start.
					wg1.Done()
					// Wait for the test to unpause the mock.
					wg2.Wait()
					// Return an empty result. It doesn't really matter what is returned.
					yield(&agentapi.ReceiveBind9ConfigRsp{
						Response: &agentapi.ReceiveBind9ConfigRsp_File{
							File: &agentapi.ReceiveBind9ConfigFile{
								FileType: agentapi.Bind9ConfigFileType_CONFIG,
							},
						},
					}, nil)
				}
			},
		)

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
		responses := manager.GetBind9FormattedConfig(context.Background(), daemon.ID, nil, nil)
		for response := range responses {
			require.NoError(t, response.Err)
		}
	}()

	// Wait for the first request to start.
	wg1.Wait()

	// Run the second request while the first one is in progress.
	// Since we use the same daemon ID, the manager should refuse it.
	var errors []error
	responses := manager.GetBind9FormattedConfig(context.Background(), daemon.ID, nil, nil)
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
// request but another request contains different daemon ID.
func TestGetBind9FormattedConfigAnotherRequestInProgressDifferentDaemon(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	// Create the daemon.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon1 := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{})
	err = dbmodel.AddDaemon(db, daemon1)
	require.NoError(t, err)

	daemon2 := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{})
	err = dbmodel.AddDaemon(db, daemon2)
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
	mocks = append(mocks, mock.EXPECT().ReceiveBind9FormattedConfig(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon1.ID
	}), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(context.Context, *dbmodel.Daemon, *bind9config.FileTypeSelector, *bind9config.Filter) iter.Seq2[*agentapi.ReceiveBind9ConfigRsp, error] {
				return func(yield func(*agentapi.ReceiveBind9ConfigRsp, error) bool) {
					// Signalling here that the second request can start.
					wg1.Done()
					// Wait for the test to unpause the mock.
					wg2.Wait()
					// Return an empty result. It doesn't really matter what is returned.
					yield(&agentapi.ReceiveBind9ConfigRsp{
						Response: &agentapi.ReceiveBind9ConfigRsp_File{
							File: &agentapi.ReceiveBind9ConfigFile{
								FileType: agentapi.Bind9ConfigFileType_CONFIG,
							},
						},
					}, nil)
				}
			},
		),
	)
	mocks = append(mocks, mock.EXPECT().ReceiveBind9FormattedConfig(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon2.ID
	}), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(
			func(context.Context, *dbmodel.Daemon, *bind9config.FileTypeSelector, *bind9config.Filter) iter.Seq2[*agentapi.ReceiveBind9ConfigRsp, error] {
				return func(yield func(*agentapi.ReceiveBind9ConfigRsp, error) bool) {
					yield(&agentapi.ReceiveBind9ConfigRsp{
						Response: &agentapi.ReceiveBind9ConfigRsp_File{
							File: &agentapi.ReceiveBind9ConfigFile{
								FileType: agentapi.Bind9ConfigFileType_CONFIG,
							},
						},
					}, nil)
				}
			},
		),
	)
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
		responses := manager.GetBind9FormattedConfig(context.Background(), daemon1.ID, nil, nil)
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

	var errors []error
	responses := manager.GetBind9FormattedConfig(context.Background(), daemon2.ID, nil, nil)
	for response := range responses {
		errors = append(errors, response.Err)
	}
	require.Len(t, errors, 1)
	require.NoError(t, errors[0])
}

// Test cancelling a request while getting BIND 9 configuration.
func TestGetBind9FormattedConfigCancelRequest(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	mock.EXPECT().ReceiveBind9FormattedConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(
			func(ctx context.Context, daemon *dbmodel.Daemon, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) iter.Seq2[*agentapi.ReceiveBind9ConfigRsp, error] {
				return func(yield func(*agentapi.ReceiveBind9ConfigRsp, error) bool) {
					responses := []*agentapi.ReceiveBind9ConfigRsp{
						{
							Response: &agentapi.ReceiveBind9ConfigRsp_File{
								File: &agentapi.ReceiveBind9ConfigFile{
									FileType: agentapi.Bind9ConfigFileType_CONFIG,
								},
							},
						},
						{
							Response: &agentapi.ReceiveBind9ConfigRsp_File{
								File: &agentapi.ReceiveBind9ConfigFile{
									FileType: agentapi.Bind9ConfigFileType_RNDC_KEY,
								},
							},
						},
					}
					for _, response := range responses {
						if !yield(response, nil) {
							return
						}
					}
				}
			},
		)

	manager, err := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: mock,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)

	ctx, ctxCancel := context.WithCancel(context.Background())
	next, cancel := iter.Pull(
		manager.GetBind9FormattedConfig(
			ctx,
			daemon.ID,
			bind9config.NewFileTypeSelector(bind9config.FileTypeConfig),
			bind9config.NewFilter(bind9config.FilterTypeConfig, bind9config.FilterTypeView),
		),
	)
	defer cancel()

	// We should receive the first chunk before the context is cancelled.
	rsp, ok := next()
	require.True(t, ok)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.File)
	require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, rsp.File.FileType)

	// Cancel the context.
	ctxCancel()

	// The second chunk should not be received.
	_, ok = next()
	require.False(t, ok)
}
