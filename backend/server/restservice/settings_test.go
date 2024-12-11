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
	storktest "isc.org/stork/server/test/dbmodel"
)

// Check getting and setting global settings via rest api functions.
func TestSettings(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Prepare rest API.
	rSettings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	ec := NewEndpointControl()
	rapi, err := NewRestAPI(&rSettings, dbSettings, db, fa, fec, fd, ec)
	require.NoError(t, err)
	ctx := context.Background()

	// Initialize global settings.
	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	// Get all settings.
	paramsGS := settings.GetSettingsParams{}
	rsp := rapi.GetSettings(ctx, paramsGS)
	require.IsType(t, &settings.GetSettingsOK{}, rsp)
	okRsp := rsp.(*settings.GetSettingsOK)
	require.EqualValues(t, 60, okRsp.Payload.Bind9StatsPullerInterval)
	require.Empty(t, okRsp.Payload.GrafanaURL)

	// Update settings.
	paramsUS := settings.UpdateSettingsParams{
		Settings: &models.Settings{
			Bind9StatsPullerInterval:     1,
			AppsStatePullerInterval:      2,
			KeaHostsPullerInterval:       3,
			KeaStatsPullerInterval:       4,
			KeaStatusPullerInterval:      5,
			GrafanaURL:                   "http://foo:3000",
			EnableMachineRegistration:    false,
			EnableOnlineSoftwareVersions: false,
		},
	}
	rsp = rapi.UpdateSettings(ctx, paramsUS)
	require.IsType(t, &settings.UpdateSettingsOK{}, rsp)

	// Get all settings and check updates.
	paramsGS = settings.GetSettingsParams{}
	rsp = rapi.GetSettings(ctx, paramsGS)
	require.IsType(t, &settings.GetSettingsOK{}, rsp)
	okRsp = rsp.(*settings.GetSettingsOK)

	require.EqualValues(t, 1, okRsp.Payload.Bind9StatsPullerInterval)
	require.EqualValues(t, 2, okRsp.Payload.AppsStatePullerInterval)
	require.EqualValues(t, 3, okRsp.Payload.KeaHostsPullerInterval)
	require.EqualValues(t, 4, okRsp.Payload.KeaStatsPullerInterval)
	require.EqualValues(t, 5, okRsp.Payload.KeaStatusPullerInterval)

	require.EqualValues(t, "http://foo:3000", okRsp.Payload.GrafanaURL)

	require.False(t, okRsp.Payload.EnableMachineRegistration)
	require.False(t, okRsp.Payload.EnableOnlineSoftwareVersions)
}
