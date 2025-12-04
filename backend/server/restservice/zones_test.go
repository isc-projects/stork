package restservice

import (
	context "context"
	_ "embed"
	"encoding/json"
	"fmt"
	iter "iter"
	http "net/http"
	"slices"
	"strings"
	"testing"
	"time"

	dnslib "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	dnsconfig "isc.org/stork/daemoncfg/dnsconfig"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/dnsop"
	"isc.org/stork/server/gen/models"
	dns "isc.org/stork/server/gen/restapi/operations/dns"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

//go:embed testdata/valid-zone.json
var validZone []byte

// Error used in the unit tests.
type testError struct{}

// Error returned as a string.
func (err *testError) Error() string {
	return "test error"
}

//go:generate mockgen -package=restservice -destination=dnsopmanagermock_test.go -source=../dnsop/manager.go isc.org/stork/server/dnsop Manager

// Test getting zones from the database over the REST API.
func TestGetZones(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db)
	require.NoError(t, err)

	// Store zones in the database and associate them with our app.
	randomZones := testutil.GenerateRandomZones(25)
	randomZones = testutil.GenerateMoreZonesWithClass(randomZones, 25, "CH")
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 25, "secondary")
	randomZones = testutil.GenerateMoreZonesWithSerial(randomZones, 25, 123456)
	randomZones = testutil.GenerateMoreZonesWithRPZ(randomZones, 25, true)

	var (
		daemons []*dbmodel.Daemon
		zones   []*dbmodel.Zone
	)
	for i, randomZone := range randomZones {
		machine := &dbmodel.Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err = dbmodel.AddMachine(db, machine)
		require.NoError(t, err)

		accessPoint := &dbmodel.AccessPoint{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8080,
			Key:      "",
			Protocol: protocoltype.RNDC,
		}

		daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{accessPoint})
		err = dbmodel.AddDaemon(db, daemon)
		require.NoError(t, err)
		daemons = append(daemons, daemon)
		zones = append(zones, &dbmodel.Zone{
			Name: fmt.Sprintf("zone.%s", randomZones[i].Name),
			LocalZones: []*dbmodel.LocalZone{
				{
					DaemonID: daemon.ID,
					View:     fmt.Sprintf("view-%d", i),
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					RPZ:      randomZone.RPZ,
					LoadedAt: time.Now().UTC(),
				},
			},
		})
	}
	err = dbmodel.AddZones(db, zones...)
	require.NoError(t, err)

	t.Run("default page", func(t *testing.T) {
		ctx := context.Background()
		params := dns.GetZonesParams{}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 10)
		require.EqualValues(t, 125, rspOK.Payload.Total)
	})

	t.Run("page and offset", func(t *testing.T) {
		ctx := context.Background()
		start := int64(15)
		limit := int64(20)
		params := dns.GetZonesParams{
			Start: &start,
			Limit: &limit,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 20)
		require.EqualValues(t, 125, rspOK.Payload.Total)
	})

	t.Run("filter by serial", func(t *testing.T) {
		ctx := context.Background()
		serial := "3456"
		params := dns.GetZonesParams{
			Serial: &serial,
			Limit:  storkutil.Ptr(int64(1000)),
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 25)
		require.EqualValues(t, 25, rspOK.Payload.Total)
		for _, zone := range rspOK.Payload.Items {
			require.EqualValues(t, 123456, zone.LocalZones[0].Serial)
		}
	})

	t.Run("filter by non-matching serial", func(t *testing.T) {
		ctx := context.Background()
		serial := "890123"
		params := dns.GetZonesParams{
			Serial: &serial,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Empty(t, rspOK.Payload.Items)
		require.Zero(t, rspOK.Payload.Total)
	})

	t.Run("filter by class", func(t *testing.T) {
		ctx := context.Background()
		class := "CH"
		params := dns.GetZonesParams{
			Class: &class,
			Limit: storkutil.Ptr(int64(1000)),
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 25)
		require.EqualValues(t, 25, rspOK.Payload.Total)
		for _, zone := range rspOK.Payload.Items {
			require.Equal(t, "CH", zone.LocalZones[0].Class)
		}
	})

	t.Run("filter by non-matching class", func(t *testing.T) {
		ctx := context.Background()
		class := "HS"
		params := dns.GetZonesParams{
			Class: &class,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Empty(t, rspOK.Payload.Items)
		require.Zero(t, rspOK.Payload.Total)
	})

	t.Run("filter by zone type", func(t *testing.T) {
		ctx := context.Background()
		params := dns.GetZonesParams{
			Limit:    storkutil.Ptr(int64(1000)),
			ZoneType: []string{"secondary"},
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 25)
		require.EqualValues(t, 25, rspOK.Payload.Total)
		for _, zone := range rspOK.Payload.Items {
			require.Equal(t, "secondary", zone.LocalZones[0].ZoneType)
		}
	})

	t.Run("filter by several zone types", func(t *testing.T) {
		ctx := context.Background()
		params := dns.GetZonesParams{
			Limit:    storkutil.Ptr(int64(1000)),
			ZoneType: []string{"primary", "secondary"},
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 125)
		require.EqualValues(t, 125, rspOK.Payload.Total)

		// Check unique zone types.
		collectedZoneTypes := make(map[string]struct{})
		for _, zone := range rspOK.Payload.Items {
			collectedZoneTypes[zone.LocalZones[0].ZoneType] = struct{}{}
		}
		require.Equal(t, 2, len(collectedZoneTypes))
		require.Contains(t, collectedZoneTypes, "primary")
		require.Contains(t, collectedZoneTypes, "secondary")
	})

	t.Run("filter by non-existent zone type", func(t *testing.T) {
		ctx := context.Background()
		params := dns.GetZonesParams{
			ZoneType: []string{"foo"},
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Empty(t, rspOK.Payload.Items)
		require.Zero(t, rspOK.Payload.Total)
	})

	t.Run("filter by app ID", func(t *testing.T) {
		ctx := context.Background()
		virtualApp := daemons[0].GetVirtualApp()
		appID := virtualApp.ID
		params := dns.GetZonesParams{
			AppID: &appID,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.NotEmpty(t, rspOK.Payload.Items)
		require.Equal(t, 1, len(rspOK.Payload.Items))
		require.EqualValues(t, virtualApp.ID, rspOK.Payload.Items[0].LocalZones[0].AppID)
		require.EqualValues(t, 1, rspOK.Payload.Total)
	})

	t.Run("filter by non-existent app ID", func(t *testing.T) {
		ctx := context.Background()
		appID := daemons[99].GetVirtualApp().ID + 100
		params := dns.GetZonesParams{
			AppID: &appID,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Empty(t, rspOK.Payload.Items)
		require.Zero(t, rspOK.Payload.Total)
	})

	t.Run("filter by DNS app type", func(t *testing.T) {
		ctx := context.Background()
		appType := "bind9"
		params := dns.GetZonesParams{
			AppType: &appType,
			Limit:   storkutil.Ptr(int64(1000)),
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 125)
		require.EqualValues(t, 125, rspOK.Payload.Total)
	})

	t.Run("filter by non-DNS app type", func(t *testing.T) {
		ctx := context.Background()
		appType := "kea"
		params := dns.GetZonesParams{
			AppType: &appType,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Empty(t, rspOK.Payload.Items)
		require.Zero(t, rspOK.Payload.Total)
	})

	t.Run("filter by zone name using text", func(t *testing.T) {
		ctx := context.Background()
		// Use the first zone's name as search text
		searchText := zones[0].Name
		params := dns.GetZonesParams{
			Text: &searchText,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		// We expect that typically there is only one item returned. However,
		// the zone names are autogenerated and may sometimes contain the search
		// text. To avoid sporadic test failures, let's just make sure that the
		// searched zone is present in the returned list.
		require.GreaterOrEqual(t, len(rspOK.Payload.Items), 1)
		index := slices.IndexFunc(rspOK.Payload.Items, func(zone *models.Zone) bool {
			return zone.Name == searchText
		})
		require.GreaterOrEqual(t, index, 0)
	})

	t.Run("filter by view name using text", func(t *testing.T) {
		ctx := context.Background()
		view := zones[0].LocalZones[0].View
		params := dns.GetZonesParams{
			Text: &view,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		// We expect that typically there is only one item returned. However,
		// the zone names are autogenerated and may sometimes contain the view
		// name. To avoid sporadic test failures, let's just make sure that the
		// searched zone is present in the returned list.
		require.GreaterOrEqual(t, len(rspOK.Payload.Items), 1)
		index := slices.IndexFunc(rspOK.Payload.Items, func(zone *models.Zone) bool {
			return zone.LocalZones[0].View == view
		})
		require.GreaterOrEqual(t, index, 0)
	})

	t.Run("filter with excluding RPZ", func(t *testing.T) {
		ctx := context.Background()
		params := dns.GetZonesParams{
			Limit: storkutil.Ptr(int64(1000)),
			Rpz:   storkutil.Ptr(false),
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 100)
		require.EqualValues(t, 100, rspOK.Payload.Total)
		for _, zone := range rspOK.Payload.Items {
			// Must not return RPZ zones.
			require.False(t, zone.LocalZones[0].Rpz)
		}
	})

	t.Run("filter with only RPZ", func(t *testing.T) {
		ctx := context.Background()
		params := dns.GetZonesParams{
			Limit: storkutil.Ptr(int64(1000)),
			Rpz:   storkutil.Ptr(true),
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 25)
		require.EqualValues(t, 25, rspOK.Payload.Total)
		for _, zone := range rspOK.Payload.Items {
			// All zones must be RPZ.
			require.True(t, zone.LocalZones[0].Rpz)
		}
	})

	t.Run("zones with custom sorting", func(t *testing.T) {
		ctx := context.Background()
		start := int64(0)
		limit := int64(150)
		sortField := "type"
		sortDir := string(dbmodel.SortDirDesc)
		params := dns.GetZonesParams{
			Start:     &start,
			Limit:     &limit,
			SortField: &sortField,
			SortDir:   &sortDir,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.Len(t, rspOK.Payload.Items, 125)
		require.EqualValues(t, 125, rspOK.Payload.Total)
		zones := rspOK.Payload.Items
		for i := range zones {
			if i > 0 {
				require.LessOrEqual(t, zones[i].LocalZones[0].ZoneType, zones[i-1].LocalZones[0].ZoneType)
			}
		}
	})
}

// Test getting single zone from the database over the REST API.
func TestGetZone(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db)
	require.NoError(t, err)

	// Store zones in the database and associate them with our app.
	randomZones := testutil.GenerateRandomZones(5)
	randomZones = testutil.GenerateMoreZonesWithClass(randomZones, 5, "CH")
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 5, "secondary")
	randomZones = testutil.GenerateMoreZonesWithSerial(randomZones, 5, 123456)
	randomZones = testutil.GenerateMoreZonesWithRPZ(randomZones, 5, true)
	var zones []*dbmodel.Zone
	for i, randomZone := range randomZones {
		machine := &dbmodel.Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err = dbmodel.AddMachine(db, machine)
		require.NoError(t, err)

		daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{})
		err := dbmodel.AddDaemon(db, daemon)
		require.NoError(t, err)
		zones = append(zones, &dbmodel.Zone{
			Name: randomZones[i].Name,
			LocalZones: []*dbmodel.LocalZone{
				{
					DaemonID: daemon.ID,
					View:     fmt.Sprintf("view-%d", i),
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					RPZ:      randomZone.RPZ,
					LoadedAt: time.Now().UTC(),
				},
			},
		})
	}
	err = dbmodel.AddZones(db, zones...)
	require.NoError(t, err)

	ctx := context.Background()
	// Pass case test.
	params := dns.GetZoneParams{
		ZoneID: zones[0].ID,
	}
	rsp := rapi.GetZone(ctx, params)
	require.IsType(t, &dns.GetZoneOK{}, rsp)
	rspOK := (rsp).(*dns.GetZoneOK)
	require.EqualValues(t, zones[0].Name, rspOK.Payload.Name)
	require.EqualValues(t, zones[0].LocalZones[0].Serial, rspOK.Payload.LocalZones[0].Serial)

	// Non-existing ID. GetZone should return a default response.
	params = dns.GetZoneParams{
		ZoneID: 456123,
	}
	rsp = rapi.GetZone(ctx, params)
	require.IsType(t, &dns.GetZoneDefault{}, rsp)
	defaultRsp := rsp.(*dns.GetZoneDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "Cannot find DNS zone with ID 456123", *defaultRsp.Payload.Message)
}

// Test that the HTTP InternalServerError status is returned when the
// database query fails.
func TestGetZonesError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	// Teardown the database connection to cause an error.
	teardown()

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db)
	require.NoError(t, err)

	// Make a request to the REST API when the connection to the database
	// is unavailable.
	ctx := context.Background()
	params := dns.GetZonesParams{}
	rsp := rapi.GetZones(ctx, params)

	// An error should be returned.
	require.IsType(t, &dns.GetZonesDefault{}, rsp)
	defaultRsp := rsp.(*dns.GetZonesDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to get zones from the database", *defaultRsp.Payload.Message)
}

// Test getting zone inventory states from the database over the REST API.
func TestGetZonesFetch(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create several states.
	details := []*dbmodel.ZoneInventoryStateDetails{
		{
			Status:            dbmodel.ZoneInventoryStatusBusy,
			Error:             storkutil.Ptr("busy error"),
			ZoneCount:         storkutil.Ptr(int64(123)),
			DistinctZoneCount: storkutil.Ptr(int64(200)),
			BuiltinZoneCount:  storkutil.Ptr(int64(23)),
		},
		{
			Status:            dbmodel.ZoneInventoryStatusErred,
			Error:             storkutil.Ptr("other error"),
			ZoneCount:         storkutil.Ptr(int64(234)),
			DistinctZoneCount: storkutil.Ptr(int64(300)),
			BuiltinZoneCount:  storkutil.Ptr(int64(233)),
		},
		{
			Status:            dbmodel.ZoneInventoryStatusUninitialized,
			Error:             storkutil.Ptr("uninitialized error"),
			ZoneCount:         storkutil.Ptr(int64(345)),
			DistinctZoneCount: storkutil.Ptr(int64(400)),
			BuiltinZoneCount:  storkutil.Ptr(int64(234)),
		},
	}
	// Add the machines and daemons and associate them with the states.
	for i := range details {
		machine := &dbmodel.Machine{
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := dbmodel.AddMachine(db, machine)
		require.NoError(t, err)

		accessPoint := &dbmodel.AccessPoint{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8080,
			Key:      "",
			Protocol: protocoltype.RNDC,
		}

		daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{accessPoint})
		err = dbmodel.AddDaemon(db, daemon)
		require.NoError(t, err)

		state := dbmodel.NewZoneInventoryState(daemon.ID, details[i])
		err = dbmodel.AddZoneInventoryState(db, state)
		require.NoError(t, err)
	}

	// Mock returns false, so the actual zone inventory states will be
	// fetched from the database.
	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetFetchZonesProgress().Return(false, 10, 10)

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)

	// Get the states from the database.
	ctx := context.Background()
	params := dns.GetZonesFetchParams{}
	rsp := rapi.GetZonesFetch(ctx, params)
	require.NotNil(t, rsp)
	require.IsType(t, &dns.GetZonesFetchOK{}, rsp)
	rspOK := (rsp).(*dns.GetZonesFetchOK)
	require.Len(t, rspOK.Payload.Items, 3)
	require.EqualValues(t, 3, rspOK.Payload.Total)

	// Compare the returned states with the ones inserted to the database.
	for _, d := range details {
		index := slices.IndexFunc(rspOK.Payload.Items, func(state *models.ZoneInventoryState) bool {
			return d.Status == dbmodel.ZoneInventoryStatus(state.Status)
		})
		require.GreaterOrEqual(t, index, 0)
		require.Equal(t, d.Error, rspOK.Payload.Items[index].Error)
		require.Equal(t, d.ZoneCount, rspOK.Payload.Items[index].ZoneConfigsCount)
		require.Equal(t, d.DistinctZoneCount, rspOK.Payload.Items[index].DistinctZonesCount)
		require.Equal(t, d.BuiltinZoneCount, rspOK.Payload.Items[index].BuiltinZonesCount)
		require.Positive(t, rspOK.Payload.Items[index].DaemonID)
		require.Positive(t, rspOK.Payload.Items[index].AppID)
		require.NotZero(t, rspOK.Payload.Items[index].CreatedAt)
	}
}

// Test that HTTP Accepted status is returned when zones fetch hasn't
// finished and another is attempted.
func TestGetZonesFetchAlreadyFetching(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Mock returns false, so the actual zone inventory states will be
	// fetched from the database.
	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetFetchZonesProgress().Return(true, 10, 5)

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, db, dbSettings, mockManager)
	require.NoError(t, err)

	ctx := context.Background()
	params := dns.GetZonesFetchParams{}
	rsp := rapi.GetZonesFetch(ctx, params)
	require.NotNil(t, rsp)
	require.IsType(t, &dns.GetZonesFetchAccepted{}, rsp)
	rspAccepted := (rsp).(*dns.GetZonesFetchAccepted)
	require.EqualValues(t, 10, rspAccepted.Payload.AppsCount)
	require.EqualValues(t, 5, rspAccepted.Payload.CompletedAppsCount)
}

// Test getting the zone inventory states when no state is available.
func TestGetZonesFetchNoContent(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetFetchZonesProgress().Return(false, 10, 10)

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, db, dbSettings, mockManager)
	require.NoError(t, err)

	ctx := context.Background()
	params := dns.GetZonesFetchParams{}
	rsp := rapi.GetZonesFetch(ctx, params)
	require.NotNil(t, rsp)
	require.IsType(t, &dns.GetZonesFetchNoContent{}, rsp)
}

// Tests triggering the zones fetch in background and that the
// HTTP Accepted status is returned.
func TestPutZonesFetch(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().FetchZones(gomock.Any(), gomock.Any(), false).Return(nil, nil)

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)
	ctx := context.Background()

	params := dns.PutZonesFetchParams{}
	rsp := rapi.PutZonesFetch(ctx, params)
	require.IsType(t, &dns.PutZonesFetchAccepted{}, rsp)
}

// Test that the HTTP Accepted status is returned upon an attempt to
// fetch the zones while the fetch is already in progress.
func TestPutZonesFetchAlreadyFetching(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().FetchZones(gomock.Any(), gomock.Any(), false).Return(nil, &dnsop.ManagerAlreadyFetchingError{})

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)
	ctx := context.Background()

	params := dns.PutZonesFetchParams{}
	rsp := rapi.PutZonesFetch(ctx, params)
	require.IsType(t, &dns.PutZonesFetchAccepted{}, rsp)
}

// Test that HTTP InternalServerError status is returned when fetching the
// zones fails to start.
func TestPutZonesFetchError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().FetchZones(gomock.Any(), gomock.Any(), false).Return(nil, &testError{})

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)
	ctx := context.Background()

	params := dns.PutZonesFetchParams{}
	rsp := rapi.PutZonesFetch(ctx, params)
	require.IsType(t, &dns.PutZonesFetchDefault{}, rsp)
	defaultRsp := rsp.(*dns.PutZonesFetchDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to start fetching the zones", *defaultRsp.Payload.Message)
}

// Test successfully receiving the zone RRs from the manager.
func TestGetZoneRRs(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var rrs []string
	err := json.Unmarshal(validZone, &rrs)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetZoneRRs(int64(1), int64(2), "trusted", gomock.Any(), dnsop.GetZoneRRsOptionExcludeTrailingSOA).DoAndReturn(func(zoneID int64, daemonID int64, viewName string, filter *dbmodel.GetZoneRRsFilter, options ...dnsop.GetZoneRRsOption) iter.Seq[*dnsop.RRResponse] {
		// Ensure that the filter is applied.
		require.NotNil(t, filter)
		require.Equal(t, 1, filter.GetOffset())
		require.Equal(t, 10, filter.GetLimit())
		require.ElementsMatch(t, []string{"A", "AAAA", "SOA"}, filter.GetTypes())
		require.Equal(t, "example.com", filter.GetText())
		return func(yield func(*dnsop.RRResponse) bool) {
			for _, rr := range rrs {
				rr, err := dnsconfig.NewRR(rr)
				require.NoError(t, err)
				if !yield(dnsop.NewZoneTransferRRResponse([]*dnsconfig.RR{rr})) {
					return
				}
			}
		}
	})

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)
	ctx := context.Background()

	params := dns.GetZoneRRsParams{
		ZoneID:   1,
		DaemonID: 2,
		ViewName: "trusted",
		Start:    storkutil.Ptr(int64(1)),
		Limit:    storkutil.Ptr(int64(10)),
		RrType:   []string{"A", "AAAA", "SOA"},
		Text:     storkutil.Ptr("example.com"),
	}
	rsp := rapi.GetZoneRRs(ctx, params)
	require.IsType(t, &dns.GetZoneRRsOK{}, rsp)
	rspOK := (rsp).(*dns.GetZoneRRsOK)
	require.Equal(t, len(rrs), len(rspOK.Payload.Items))

	for i, rr := range rrs {
		parsedRR, err := dnslib.NewRR(rr)
		require.NoError(t, err)
		require.Equal(t, parsedRR.Header().Name, rspOK.Payload.Items[i].Name)
		require.EqualValues(t, parsedRR.Header().Ttl, rspOK.Payload.Items[i].TTL)
		require.Equal(t, dnslib.ClassToString[parsedRR.Header().Class], rspOK.Payload.Items[i].RrClass)
		require.Equal(t, dnslib.TypeToString[parsedRR.Header().Rrtype], rspOK.Payload.Items[i].RrType)
		parsedFields := strings.Fields(rr)
		require.Greater(t, len(parsedFields), 4)
		fields := strings.Fields(rspOK.Payload.Items[i].Data)
		for _, field := range fields {
			require.Contains(t, parsedFields[4:], field)
		}
	}
}

// Test successfully refreshing the zone RRs cache.
func TestPutZoneRRsCache(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var rrs []string
	err := json.Unmarshal(validZone, &rrs)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetZoneRRs(int64(1), int64(2), "trusted", gomock.Any(), dnsop.GetZoneRRsOptionForceZoneTransfer, dnsop.GetZoneRRsOptionExcludeTrailingSOA).AnyTimes().DoAndReturn(func(zoneID int64, daemonID int64, viewName string, filter *dbmodel.GetZoneRRsFilter, options ...dnsop.GetZoneRRsOption) iter.Seq[*dnsop.RRResponse] {
		// Ensure that the filter is applied.
		require.NotNil(t, filter)
		require.Equal(t, 1, filter.GetOffset())
		require.Equal(t, 10, filter.GetLimit())
		require.ElementsMatch(t, []string{"A", "AAAA", "SOA"}, filter.GetTypes())
		require.Equal(t, "example.com", filter.GetText())
		return func(yield func(*dnsop.RRResponse) bool) {
			for _, rr := range rrs {
				rr, err := dnsconfig.NewRR(rr)
				require.NoError(t, err)
				if !yield(dnsop.NewZoneTransferRRResponse([]*dnsconfig.RR{rr})) {
					return
				}
			}
		}
	})

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)
	ctx := context.Background()

	params := dns.PutZoneRRsCacheParams{
		ZoneID:   1,
		DaemonID: 2,
		ViewName: "trusted",
		Start:    storkutil.Ptr(int64(1)),
		Limit:    storkutil.Ptr(int64(10)),
		RrType:   []string{"A", "AAAA", "SOA"},
		Text:     storkutil.Ptr("example.com"),
	}
	rsp := rapi.PutZoneRRsCache(ctx, params)
	require.IsType(t, &dns.PutZoneRRsCacheOK{}, rsp)
	rspOK := (rsp).(*dns.PutZoneRRsCacheOK)
	require.Equal(t, len(rrs), len(rspOK.Payload.Items))

	zoneTransferAt := time.Time(rspOK.Payload.ZoneTransferAt)
	require.NotZero(t, zoneTransferAt)
	require.WithinDuration(t, time.Now().UTC(), zoneTransferAt, 5*time.Second)
	require.False(t, rspOK.Payload.Cached)

	for i, rr := range rrs {
		parsedRR, err := dnslib.NewRR(rr)
		require.NoError(t, err)
		require.Equal(t, parsedRR.Header().Name, rspOK.Payload.Items[i].Name)
		require.EqualValues(t, parsedRR.Header().Ttl, rspOK.Payload.Items[i].TTL)
		require.Equal(t, dnslib.ClassToString[parsedRR.Header().Class], rspOK.Payload.Items[i].RrClass)
		require.Equal(t, dnslib.TypeToString[parsedRR.Header().Rrtype], rspOK.Payload.Items[i].RrType)
		parsedFields := strings.Fields(rr)
		require.Greater(t, len(parsedFields), 4)
		fields := strings.Fields(rspOK.Payload.Items[i].Data)
		for _, field := range fields {
			require.Contains(t, parsedFields[4:], field)
		}
	}

	// Run it again. It should refresh RRs.
	rsp = rapi.PutZoneRRsCache(ctx, params)
	require.IsType(t, &dns.PutZoneRRsCacheOK{}, rsp)
	rspOK = (rsp).(*dns.PutZoneRRsCacheOK)
	require.Equal(t, len(rrs), len(rspOK.Payload.Items))

	zoneTransferAt2 := time.Time(rspOK.Payload.ZoneTransferAt)
	require.NotZero(t, zoneTransferAt2)
	require.WithinDuration(t, time.Now().UTC(), zoneTransferAt2, 5*time.Second)
	require.False(t, rspOK.Payload.Cached)

	// Check that the transfer times are not the same.
	require.NotEqual(t, zoneTransferAt, zoneTransferAt2)
}

// Test that HTTP Conflict status is returned when the zone transfer for the
// same zone and view is already in progress.
func TestGetZoneRRsAnotherRequestInProgress(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var rrs []string
	err := json.Unmarshal(validZone, &rrs)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetZoneRRs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(zoneID int64, daemonID int64, viewName string, filter *dbmodel.GetZoneRRsFilter, options ...dnsop.GetZoneRRsOption) iter.Seq[*dnsop.RRResponse] {
		return func(yield func(*dnsop.RRResponse) bool) {
			yield(dnsop.NewErrorRRResponse(dnsop.NewManagerRRsAlreadyRequestedError("trusted", "example.com")))
		}
	})

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("get zone RRs", func(t *testing.T) {
		params := dns.GetZoneRRsParams{}
		rsp := rapi.GetZoneRRs(ctx, params)
		require.IsType(t, &dns.GetZoneRRsDefault{}, rsp)
		defaultRsp := rsp.(*dns.GetZoneRRsDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
		require.Contains(t, *defaultRsp.Payload.Message, "zone transfer for view trusted, zone example.com has been already requested by another user")
	})

	t.Run("put zone RRs cache", func(t *testing.T) {
		params := dns.PutZoneRRsCacheParams{
			ZoneID:   1,
			DaemonID: 2,
			ViewName: "trusted",
		}
		rsp := rapi.PutZoneRRsCache(ctx, params)
		require.IsType(t, &dns.PutZoneRRsCacheDefault{}, rsp)
		defaultRsp := rsp.(*dns.PutZoneRRsCacheDefault)
		require.Equal(t, http.StatusAccepted, getStatusCode(*defaultRsp))
		require.Contains(t, *defaultRsp.Payload.Message, "zone transfer for view trusted, zone example.com has been already requested by another user")
	})
}

// Test that HTTP Conflict status is returned when the zone inventory is busy.
func TestGetZoneRRsBusy(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetZoneRRs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(zoneID int64, daemonID int64, viewName string, filter *dbmodel.GetZoneRRsFilter, options ...dnsop.GetZoneRRsOption) iter.Seq[*dnsop.RRResponse] {
		return func(yield func(*dnsop.RRResponse) bool) {
			yield(dnsop.NewErrorRRResponse(agentcomm.NewZoneInventoryBusyError("localhost:8080")))
		}
	})

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("get zone RRs", func(t *testing.T) {
		params := dns.GetZoneRRsParams{
			ZoneID:   1,
			DaemonID: 2,
			ViewName: "trusted",
		}
		rsp := rapi.GetZoneRRs(ctx, params)
		require.IsType(t, &dns.GetZoneRRsDefault{}, rsp)
		defaultRsp := rsp.(*dns.GetZoneRRsDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
		require.Contains(t, *defaultRsp.Payload.Message, "Zone inventory is temporarily busy on the agent localhost:8080")
	})

	t.Run("put zone RRs cache", func(t *testing.T) {
		params := dns.PutZoneRRsCacheParams{
			ZoneID:   1,
			DaemonID: 2,
			ViewName: "trusted",
		}
		rsp := rapi.PutZoneRRsCache(ctx, params)
		require.IsType(t, &dns.PutZoneRRsCacheDefault{}, rsp)
		defaultRsp := rsp.(*dns.PutZoneRRsCacheDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
		require.Contains(t, *defaultRsp.Payload.Message, "Zone inventory is temporarily busy on the agent localhost:8080")
	})
}

// Test that HTTP ServiceUnavailable status is returned when the zone inventory
// is not initialized.
func TestGetZoneRRsNotInited(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetZoneRRs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(zoneID int64, daemonID int64, viewName string, filter *dbmodel.GetZoneRRsFilter, options ...dnsop.GetZoneRRsOption) iter.Seq[*dnsop.RRResponse] {
		return func(yield func(*dnsop.RRResponse) bool) {
			yield(dnsop.NewErrorRRResponse(agentcomm.NewZoneInventoryNotInitedError("localhost:8080")))
		}
	})

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("get zone RRs", func(t *testing.T) {
		params := dns.GetZoneRRsParams{
			ZoneID:   1,
			DaemonID: 2,
			ViewName: "trusted",
		}
		rsp := rapi.GetZoneRRs(ctx, params)
		require.IsType(t, &dns.GetZoneRRsDefault{}, rsp)
		defaultRsp := rsp.(*dns.GetZoneRRsDefault)
		require.Equal(t, http.StatusServiceUnavailable, getStatusCode(*defaultRsp))
		require.Contains(t, *defaultRsp.Payload.Message, "DNS zones have not been loaded on the agent localhost:8080")
	})

	t.Run("put zone RRs cache", func(t *testing.T) {
		params := dns.PutZoneRRsCacheParams{
			ZoneID:   1,
			DaemonID: 2,
			ViewName: "trusted",
		}
		rsp := rapi.PutZoneRRsCache(ctx, params)
		require.IsType(t, &dns.PutZoneRRsCacheDefault{}, rsp)
		defaultRsp := rsp.(*dns.PutZoneRRsCacheDefault)
		require.Equal(t, http.StatusServiceUnavailable, getStatusCode(*defaultRsp))
		require.Contains(t, *defaultRsp.Payload.Message, "DNS zones have not been loaded on the agent localhost:8080")
	})
}

// Test that HTTP InternalServerError status is returned when an unknown error
// occurs while getting the zone RRs.
func TestGetZoneRRsUnknownError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetZoneRRs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(zoneID int64, daemonID int64, viewName string, filter *dbmodel.GetZoneRRsFilter, options ...dnsop.GetZoneRRsOption) iter.Seq[*dnsop.RRResponse] {
		return func(yield func(*dnsop.RRResponse) bool) {
			yield(dnsop.NewErrorRRResponse(&testError{}))
		}
	})

	settings := RestAPISettings{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mockManager)
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("get zone RRs", func(t *testing.T) {
		params := dns.GetZoneRRsParams{
			ZoneID:   1,
			DaemonID: 2,
			ViewName: "trusted",
		}
		rsp := rapi.GetZoneRRs(ctx, params)
		require.IsType(t, &dns.GetZoneRRsDefault{}, rsp)
		defaultRsp := rsp.(*dns.GetZoneRRsDefault)
		require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
		require.Contains(t, *defaultRsp.Payload.Message, "Failed to get zone contents: test error")
	})

	t.Run("put zone RRs cache", func(t *testing.T) {
		params := dns.PutZoneRRsCacheParams{
			ZoneID:   1,
			DaemonID: 2,
			ViewName: "trusted",
		}
		rsp := rapi.PutZoneRRsCache(ctx, params)
		require.IsType(t, &dns.PutZoneRRsCacheDefault{}, rsp)
		defaultRsp := rsp.(*dns.PutZoneRRsCacheDefault)
		require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
		require.Contains(t, *defaultRsp.Payload.Message, "Failed to refresh zone contents using zone transfer")
	})
}
