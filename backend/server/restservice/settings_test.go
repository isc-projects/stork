package restservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/settings"
	storktest "isc.org/stork/server/test"
)

// Check getting and setting global settings via rest api functions.
func TestSettings(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare rest api
	rSettings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&rSettings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// initialize global settings
	err = dbmodel.InitializeSettings(db)
	require.NoError(t, err)

	// get all settings
	paramsGS := settings.GetSettingsParams{}
	rsp := rapi.GetSettings(ctx, paramsGS)
	require.IsType(t, &settings.GetSettingsOK{}, rsp)
	okRsp := rsp.(*settings.GetSettingsOK)
	require.EqualValues(t, 60, okRsp.Payload.Bind9StatsPullerInterval)
	require.EqualValues(t, "", okRsp.Payload.GrafanaURL)

	// update settings
	paramsUS := settings.UpdateSettingsParams{
		Settings: &models.Settings{
			Bind9StatsPullerInterval: 10,
			GrafanaURL:               "http://localhost:3000",
		},
	}
	rsp = rapi.UpdateSettings(ctx, paramsUS)
	require.IsType(t, &settings.UpdateSettingsOK{}, rsp)

	// get all settings and check updates
	paramsGS = settings.GetSettingsParams{}
	rsp = rapi.GetSettings(ctx, paramsGS)
	require.IsType(t, &settings.GetSettingsOK{}, rsp)
	okRsp = rsp.(*settings.GetSettingsOK)
	require.EqualValues(t, 10, okRsp.Payload.Bind9StatsPullerInterval)
	require.EqualValues(t, "http://localhost:3000", okRsp.Payload.GrafanaURL)
}
