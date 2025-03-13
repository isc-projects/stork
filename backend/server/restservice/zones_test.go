package restservice

import (
	context "context"
	"fmt"
	http "net/http"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/dnsop"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/dns"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

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

	var (
		apps  []*dbmodel.App
		zones []*dbmodel.Zone
	)
	for i, randomZone := range randomZones {
		machine := &dbmodel.Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err = dbmodel.AddMachine(db, machine)
		require.NoError(t, err)

		app := &dbmodel.App{
			ID:        0,
			MachineID: machine.ID,
			Type:      dbmodel.AppTypeBind9,
			Name:      fmt.Sprintf("app-%d", i),
			Daemons: []*dbmodel.Daemon{
				dbmodel.NewBind9Daemon(true),
			},
		}
		addedDaemons, err := dbmodel.AddApp(db, app)
		require.NoError(t, err)
		require.Len(t, addedDaemons, 1)
		apps = append(apps, app)
		zones = append(zones, &dbmodel.Zone{
			Name: randomZones[i].Name,
			LocalZones: []*dbmodel.LocalZone{
				{
					DaemonID: addedDaemons[0].ID,
					View:     fmt.Sprintf("view-%d", i),
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
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
		require.EqualValues(t, 100, rspOK.Payload.Total)
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
		require.EqualValues(t, 100, rspOK.Payload.Total)
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
		require.Len(t, rspOK.Payload.Items, 100)
		require.EqualValues(t, 100, rspOK.Payload.Total)

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
		appID := apps[0].ID
		params := dns.GetZonesParams{
			AppID: &appID,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		require.NotEmpty(t, rspOK.Payload.Items)
		require.Equal(t, 1, len(rspOK.Payload.Items))
		require.EqualValues(t, apps[0].ID, rspOK.Payload.Items[0].LocalZones[0].AppID)
		require.EqualValues(t, 1, rspOK.Payload.Total)
	})

	t.Run("filter by non-existent app ID", func(t *testing.T) {
		ctx := context.Background()
		appID := apps[99].ID + 100
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
		require.Len(t, rspOK.Payload.Items, 100)
		require.EqualValues(t, 100, rspOK.Payload.Total)
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

	t.Run("filter by app name using text", func(t *testing.T) {
		ctx := context.Background()
		// Use the first zone's name as search text
		searchText := apps[0].Name
		params := dns.GetZonesParams{
			Text: &searchText,
		}
		rsp := rapi.GetZones(ctx, params)
		require.IsType(t, &dns.GetZonesOK{}, rsp)
		rspOK := (rsp).(*dns.GetZonesOK)
		// We expect that typically there is only one item returned. However,
		// the zone names are autogenerated and may sometimes contain the app
		// name. To avoid sporadic test failures, let's just make sure that the
		// searched zone is present in the returned list.
		require.GreaterOrEqual(t, len(rspOK.Payload.Items), 1)
		index := slices.IndexFunc(rspOK.Payload.Items, func(zone *models.Zone) bool {
			return zone.LocalZones[0].AppName == searchText
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
			Status:    dbmodel.ZoneInventoryStatusBusy,
			Error:     storkutil.Ptr("busy error"),
			ZoneCount: storkutil.Ptr(int64(123)),
		},
		{
			Status:    dbmodel.ZoneInventoryStatusErred,
			Error:     storkutil.Ptr("other error"),
			ZoneCount: storkutil.Ptr(int64(234)),
		},
		{
			Status:    dbmodel.ZoneInventoryStatusUninitialized,
			Error:     storkutil.Ptr("uninitialized error"),
			ZoneCount: storkutil.Ptr(int64(345)),
		},
	}
	// Add the machines and apps and associate them with the states.
	for i := range details {
		machine := &dbmodel.Machine{
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := dbmodel.AddMachine(db, machine)
		require.NoError(t, err)

		app := &dbmodel.App{
			MachineID: machine.ID,
			Type:      dbmodel.AppTypeBind9,
			Daemons: []*dbmodel.Daemon{
				dbmodel.NewBind9Daemon(true),
			},
		}
		addedDaemons, err := dbmodel.AddApp(db, app)
		require.NoError(t, err)
		require.Len(t, addedDaemons, 1)

		state := dbmodel.NewZoneInventoryState(addedDaemons[0].ID, details[i])
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
		require.Equal(t, d.ZoneCount, rspOK.Payload.Items[index].ZoneCount)
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
