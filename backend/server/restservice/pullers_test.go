package restservice

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	apps "isc.org/stork/server/apps"
	"isc.org/stork/server/apps/bind9"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/restapi/operations/settings"
)

// Test that the puller status list is returned properly.
func TestGetPullers(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)

	rapiSettings := RestAPISettings{}

	statePuller, _ := apps.NewStatePuller(db, nil, nil, nil, nil)
	bind9Puller, _ := bind9.NewStatsPuller(db, nil, nil)
	pullers := &apps.Pullers{
		AppsStatePuller:  statePuller,
		Bind9StatsPuller: bind9Puller,
	}
	rapi, _ := NewRestAPI(&rapiSettings, dbSettings, db, pullers)

	ctx := context.Background()
	params := settings.GetPullersParams{}

	// Act
	rsp := rapi.GetPullers(ctx, params)

	// Assert
	require.IsType(t, &settings.GetPullersOK{}, rsp)
	rspOk := rsp.(*settings.GetPullersOK)
	require.Len(t, rspOk.Payload.Items, 2)
	require.EqualValues(t, len(rspOk.Payload.Items), rspOk.Payload.Total)
}

// Test that the puller status is returned properly.
func TestGetPuller(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)

	rapiSettings := RestAPISettings{}

	statePuller, _ := apps.NewStatePuller(db, nil, nil, nil, nil)
	bind9Puller, _ := bind9.NewStatsPuller(db, nil, nil)
	pullers := &apps.Pullers{
		AppsStatePuller:  statePuller,
		Bind9StatsPuller: bind9Puller,
	}
	rapi, _ := NewRestAPI(&rapiSettings, dbSettings, db, pullers)

	ctx := context.Background()
	params := settings.GetPullerParams{
		ID: "bind9_stats_puller_interval",
	}

	// Act
	rsp := rapi.GetPuller(ctx, params)

	// Assert
	require.IsType(t, &settings.GetPullerOK{}, rsp)
	rspOk := rsp.(*settings.GetPullerOK)
	require.EqualValues(t, "bind9_stats_puller_interval", rspOk.Payload.ID)
}

// Test that the HTTP 404 Not Found status is returned if a puller with the
// given name doesn't exist.
func TestGetNonExistPuller(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)

	rapiSettings := RestAPISettings{}

	pullers := &apps.Pullers{}
	rapi, _ := NewRestAPI(&rapiSettings, dbSettings, db, pullers)

	ctx := context.Background()
	params := settings.GetPullerParams{
		ID: "not_exists",
	}

	// Act
	rsp := rapi.GetPuller(ctx, params)

	// Assert
	require.IsType(t, &settings.GetPullerDefault{}, rsp)
	rspDefault := rsp.(*settings.GetPullerDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*rspDefault))
}
