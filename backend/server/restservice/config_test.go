package restservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/restapi/operations/services"
	storktest "isc.org/stork/server/test"
)

// Test that GetDaemonConfig works for Kea daemon with assigned configuration.
func TestGetDaemonConfigForKeaDaemonWithAssignedConfiguration(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// setup a user session, it is required to check user role
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app kea to machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234, true)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Name:         "test-app",
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
			dbmodel.NewKeaDaemon("dhcp6", true),
		},
	}
	// Daemon has assigned configuration
	configDhcp4, err := dbmodel.NewKeaConfigFromJSON(`{
		"Dhcp4": { }
    }`)
	require.NoError(t, err)

	app.Daemons[0].KeaDaemon.Config = configDhcp4

	configDhcp6, err := dbmodel.NewKeaConfigFromJSON(`{
		"Dhcp6": { }
    }`)
	require.NoError(t, err)

	app.Daemons[1].KeaDaemon.Config = configDhcp6

	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Check Dhcp4 daemon
	params := services.GetDaemonConfigParams{
		ID: app.Daemons[0].ID,
	}

	rsp := rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonConfigOK)
	require.NotEmpty(t, okRsp.Payload)
	require.Equal(t, configDhcp4, okRsp.Payload)

	params = services.GetDaemonConfigParams{
		ID: app.Daemons[1].ID,
	}

	// Check Dhcp6 daemon
	rsp = rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonConfigOK)
	require.NotEmpty(t, okRsp.Payload)
	require.Equal(t, configDhcp6, okRsp.Payload)
}

// Test that GetDaemonConfig returns the secrets for super admin.
func TestGetDaemonConfigWithSecretsForSuperAdmin(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// setup a user session, it is required to check user role
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	require.True(t, user.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}))

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app kea to machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234, false)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Name:         "test-app",
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	// Daemon has assigned configuration
	configDhcp4, err := dbmodel.NewKeaConfigFromJSON(`{
		"Dhcp4": {
			"primitive": {
				"password": "PASSWORD",
				"secret": "SECRET"
			},
			"complex": {
				"password": {
					"key": "value"
				},
				"secret": [
					"a", "b", "c"
				]
			},
			"fake": {
				"password-fake": "FAKE",
				"fake-secret": "FAKE"
			}
		}
    }`)
	require.NoError(t, err)

	app.Daemons[0].KeaDaemon.Config = configDhcp4

	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Check Dhcp4 daemon
	params := services.GetDaemonConfigParams{
		ID: app.Daemons[0].ID,
	}

	rsp := rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonConfigOK)
	require.NotEmpty(t, okRsp.Payload)
	require.Equal(t, configDhcp4, okRsp.Payload)
}

// Test that GetDaemonConfig hides the secrets for standard users.
func TestGetDaemonConfigWithoutSecretsForAdmin(t *testing.T) {
	// Test database initialization
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// REST API mock
	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// Create "standard" user (without any special group)
	user := &dbmodel.SystemUser{
		Email:    "john@example.org",
		Lastname: "Smith",
		Name:     "John",
		Password: "pass",
	}

	conflict, err := dbmodel.CreateUser(rapi.DB, user)
	require.False(t, conflict)
	require.NoError(t, err)

	// Log-in the user
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Check if user isn't a super admin
	require.False(t, user.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}))

	// Fill the database
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app kea to machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234, true)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Name:         "test-app",
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	// Daemon has assigned configuration with secrets
	configDhcp4, err := dbmodel.NewKeaConfigFromJSON(`{
		"Dhcp4": {
			"primitive": {
				"password": "PASSWORD",
				"secret": "SECRET"
			},
			"complex": {
				"password": {
					"key": "value"
				},
				"secret": [
					"a", "b", "c"
				]
			},
			"fake": {
				"password-fake": "FAKE",
				"fake-secret": "FAKE"
			}
		}
    }`)
	require.NoError(t, err)

	app.Daemons[0].KeaDaemon.Config = configDhcp4

	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Check Dhcp4 daemon
	params := services.GetDaemonConfigParams{
		ID: app.Daemons[0].ID,
	}

	rsp := rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonConfigOK)
	require.NotEmpty(t, okRsp.Payload)

	// Expected daemon config (without secrets)
	expected, err := dbmodel.NewKeaConfigFromJSON(`{
		"Dhcp4": {
			"primitive": {
				"password": null,
				"secret": null
			},
			"complex": {
				"password": null,
				"secret": null
			},
			"fake": {
				"password-fake": "FAKE",
				"fake-secret": "FAKE"
			}
		}
    }`)

	require.NoError(t, err)
	require.NotEmpty(t, expected)
	require.Equal(t, expected, okRsp.Payload)
}

// Test that GetDaemonConfig returns HTTP Not Found status for Kea daemon
// without assigned configuration.
func TestGetDaemonConfigForKeaDaemonWithoutAssignedConfiguration(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// setup a user session, it is required to check user role
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app kea to machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234, false)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Name:         "test-app",
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
			dbmodel.NewKeaDaemon("dhcp6", true),
		},
	}

	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	params := services.GetDaemonConfigParams{
		ID: app.Daemons[0].ID,
	}

	rsp := rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigDefault{}, rsp)
	defaultRsp := rsp.(*services.GetDaemonConfigDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	msg := fmt.Sprintf("config not assigned for daemon with id %d", params.ID)
	require.Equal(t, msg, *defaultRsp.Payload.Message)
}

// Test that GetDaemonConfig returns HTTP Bad Request status for not-Kea daemon.
func TestGetDaemonConfigForBind9Daemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// setup a user session, it is required to check user role
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add BIND 9 app
	var bind9Points []*dbmodel.AccessPoint
	bind9Points = dbmodel.AppendAccessPoint(bind9Points, dbmodel.AccessPointControl, "1.2.3.4", "abcd", 124, true)
	app := &dbmodel.App{
		MachineID:    m.ID,
		Machine:      m,
		Type:         dbmodel.AppTypeBind9,
		AccessPoints: bind9Points,
		Daemons: []*dbmodel.Daemon{
			{
				Bind9Daemon: &dbmodel.Bind9Daemon{},
			},
		},
	}

	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	params := services.GetDaemonConfigParams{
		ID: app.Daemons[0].ID,
	}

	rsp := rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigDefault{}, rsp)
	defaultRsp := rsp.(*services.GetDaemonConfigDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	msg := fmt.Sprintf("daemon with id %d isn't Kea daemon", params.ID)
	require.Equal(t, msg, *defaultRsp.Payload.Message)
}

// Test that GetDaemonConfig returns HTTP Bad Request for not exist daemon.
func TestGetDaemonConfigForNonExistsDaemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// setup a user session, it is required to check user role
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add an app
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234, false)
	app := &dbmodel.App{
		MachineID:    m.ID,
		Machine:      m,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: keaPoints,
	}

	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	params := services.GetDaemonConfigParams{
		ID: 42,
	}

	rsp := rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigDefault{}, rsp)
	defaultRsp := rsp.(*services.GetDaemonConfigDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	msg := fmt.Sprintf("cannot find daemon with id %d", params.ID)
	require.Equal(t, msg, *defaultRsp.Payload.Message)
}

// Test that GetDaemonConfig returns HTTP Internal Server Error status for failed database connection.
func TestGetDaemonConfigForDatabaseError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// setup a user session, it is required to check user role
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add an app
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234, true)
	app := &dbmodel.App{
		MachineID:    m.ID,
		Machine:      m,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: keaPoints,
	}

	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	params := services.GetDaemonConfigParams{
		ID: 42,
	}

	// Disconnect database for fail connection
	teardown()

	rsp := rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigDefault{}, rsp)
	defaultRsp := rsp.(*services.GetDaemonConfigDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	msg := fmt.Sprintf("cannot get daemon with id %d from db", params.ID)
	require.Equal(t, msg, *defaultRsp.Payload.Message)
}

// Test that config review reports are successfully retrieved for a daemon.
func TestGetDaemonConfigReports(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add an app.
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234, false)
	app := &dbmodel.App{
		MachineID:    m.ID,
		Machine:      m,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
			dbmodel.NewKeaDaemon("dhcp6", true),
		},
	}

	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// Create several config reports - two for first daemon and one for the
	// second daemon.
	configReports := []dbmodel.ConfigReport{
		{
			CheckerName: "name 1",
			Content:     "funny review contents for {daemon} and {daemon}",
			DaemonID:    app.Daemons[0].ID,
			RefDaemons: []*dbmodel.Daemon{
				{
					ID: app.Daemons[0].ID,
				},
				{
					ID: app.Daemons[1].ID,
				},
			},
		},
		{
			CheckerName: "name 2",
			Content:     "another funny review contents for {daemon}",
			DaemonID:    app.Daemons[0].ID,
			RefDaemons: []*dbmodel.Daemon{
				{
					ID: app.Daemons[1].ID,
				},
			},
		},
		{
			CheckerName: "name 3",
			Content:     "review contents for another daemon",
			DaemonID:    app.Daemons[1].ID,
			RefDaemons: []*dbmodel.Daemon{
				{
					ID: app.Daemons[1].ID,
				},
			},
		},
	}

	// Add the config reports to the database.
	for i := range configReports {
		err = dbmodel.AddConfigReport(db, &configReports[i])
		require.NoError(t, err)
	}

	// Add related config review entries.
	configReviews := []dbmodel.ConfigReview{
		{
			DaemonID:   app.Daemons[0].ID,
			ConfigHash: "1234",
			Signature:  "2345",
		},
		{
			DaemonID:   app.Daemons[1].ID,
			ConfigHash: "2345",
			Signature:  "3456",
		},
	}
	for i := range configReviews {
		err = dbmodel.AddConfigReview(db, &configReviews[i])
		require.NoError(t, err)
	}

	// Try to fetch config reports for the first daemon.
	params := services.GetDaemonConfigReportsParams{
		ID: app.Daemons[0].ID,
	}

	rsp := rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonConfigReportsOK)

	// Make sure that both have been returned.
	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, "name 1", okRsp.Payload.Items[0].Checker)
	require.Equal(t, "funny review contents for <daemon id=\"1\" name=\"dhcp4\" appId=\"1\" appType=\"kea\"> and <daemon id=\"2\" name=\"dhcp6\" appId=\"1\" appType=\"kea\">",
		okRsp.Payload.Items[0].Content)

	require.EqualValues(t, "name 2", okRsp.Payload.Items[1].Checker)
	require.Equal(t, "another funny review contents for <daemon id=\"2\" name=\"dhcp6\" appId=\"1\" appType=\"kea\">", okRsp.Payload.Items[1].Content)

	// Test getting the paged result.
	params.Start = new(int64)
	params.Limit = new(int64)
	*params.Start = 0
	*params.Limit = 1
	rsp = rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonConfigReportsOK)

	// The total number is two but only one report has been returned.
	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, "name 1", okRsp.Payload.Items[0].Checker)
	require.Equal(t, "funny review contents for <daemon id=\"1\" name=\"dhcp4\" appId=\"1\" appType=\"kea\"> and <daemon id=\"2\" name=\"dhcp6\" appId=\"1\" appType=\"kea\">",
		okRsp.Payload.Items[0].Content)
	require.NotNil(t, okRsp.Payload.Review)
	require.NotZero(t, okRsp.Payload.Review.ID)

	// Start at offset 1.
	*params.Start = 1
	*params.Limit = 2
	rsp = rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonConfigReportsOK)

	// The total number is two but only one report has been returned.
	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, "name 2", okRsp.Payload.Items[0].Checker)
	require.Equal(t, "another funny review contents for <daemon id=\"2\" name=\"dhcp6\" appId=\"1\" appType=\"kea\">", okRsp.Payload.Items[0].Content)

	// Try to fetch the config report for the second daemon.
	params = services.GetDaemonConfigReportsParams{
		ID: app.Daemons[1].ID,
	}
	rsp = rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonConfigReportsOK)

	require.EqualValues(t, 1, okRsp.Payload.Total)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, "name 3", okRsp.Payload.Items[0].Checker)
	require.Equal(t, "review contents for another daemon", okRsp.Payload.Items[0].Content)

	// If the config review is in progress it should return HTTP Accepted.
	fd.InProgress = true
	rsp = rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsAccepted{}, rsp)

	// Fetching non-existing reports should return HTTP No Content.
	fd.InProgress = false
	params.ID = 1111
	rsp = rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsNoContent{}, rsp)
}

// Test that HTTP internal server error is returned when the database
// connection fails while fetching the config reports.
func TestGetDaemonConfigReportsDatabaseError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	// Close the database connection to cause the failure while
	// fetching the config reports.
	teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	params := services.GetDaemonConfigReportsParams{
		ID: 1,
	}
	rsp := rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsDefault{}, rsp)
	defaultRsp := rsp.(*services.GetDaemonConfigReportsDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot get configuration review for daemon with id 1 from db",
		*defaultRsp.Payload.Message)
}

// Test triggering new configuration review for a daemon.
func TestPutDaemonConfigReview(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Create DHCPv4 config.
	configDhcp4, err := dbmodel.NewKeaConfigFromJSON(`{
		"Dhcp4": { }
    }`)
	require.NoError(t, err)

	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234, false)
	app := &dbmodel.App{
		MachineID:    machine.ID,
		Machine:      machine,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	app.Daemons[0].KeaDaemon.Config = configDhcp4

	daemons, err := dbmodel.AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 1)
	require.NotZero(t, daemons[0].ID)

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// Use a valid daemon ID to create new config review.
	params := services.PutDaemonConfigReviewParams{
		ID: daemons[0].ID,
	}
	rsp := rapi.PutDaemonConfigReview(ctx, params)
	require.IsType(t, &services.PutDaemonConfigReviewAccepted{}, rsp)
	acceptedRsp := rsp.(*services.PutDaemonConfigReviewAccepted)
	require.NotNil(t, acceptedRsp)

	// Ensure that the review has been started.
	require.Len(t, fd.CallLog, 1)
	require.Equal(t, "BeginReview", fd.CallLog[0])

	// Try to create a new review for a non-existing daemon.
	params.ID++
	rsp = rapi.PutDaemonConfigReview(ctx, params)
	require.IsType(t, &services.PutDaemonConfigReviewDefault{}, rsp)
	defaultRsp := rsp.(*services.PutDaemonConfigReviewDefault)
	require.NotNil(t, defaultRsp)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Contains(t, *defaultRsp.Payload.Message, "cannot find daemon with id")
}

// Test that HTTP internal server error is returned when the database
// connection fails while creating new config review.
func TestPutDaemonConfigReviewDatabaseError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	// Close the database connection to cause the failure while
	// communicating with the database
	teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	params := services.PutDaemonConfigReviewParams{
		ID: 1,
	}
	rsp := rapi.PutDaemonConfigReview(ctx, params)
	require.IsType(t, &services.PutDaemonConfigReviewDefault{}, rsp)
	defaultRsp := rsp.(*services.PutDaemonConfigReviewDefault)
	require.NotNil(t, defaultRsp)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot get daemon with id 1 from db", *defaultRsp.Payload.Message)
}

// Test that HTTP Bad Request status is returned as a result of requesting
// a configuration review for a non-Kea daemon.
func TestPutDaemonConfigReviewNotKeaDaemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Create BIND9 app instance.
	var bind9Points []*dbmodel.AccessPoint
	bind9Points = dbmodel.AppendAccessPoint(bind9Points, dbmodel.AccessPointControl, "1.2.3.4", "abcd", 124, true)
	app := &dbmodel.App{
		MachineID:    machine.ID,
		Machine:      machine,
		Type:         dbmodel.AppTypeBind9,
		AccessPoints: bind9Points,
		Daemons: []*dbmodel.Daemon{
			{
				Bind9Daemon: &dbmodel.Bind9Daemon{},
			},
		},
	}
	daemons, err := dbmodel.AddApp(db, app)
	require.NoError(t, err)

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	params := services.PutDaemonConfigReviewParams{
		ID: daemons[0].ID,
	}
	rsp := rapi.PutDaemonConfigReview(ctx, params)
	require.IsType(t, &services.PutDaemonConfigReviewDefault{}, rsp)
	defaultRsp := rsp.(*services.PutDaemonConfigReviewDefault)
	require.NotNil(t, defaultRsp)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, fmt.Sprintf("daemon with id %d is not a Kea daemon", daemons[0].ID),
		*defaultRsp.Payload.Message)
}

// Test that HTTP Bad Request status is returned as a result of requesting
// a Kea daemon configuration review when the configuration is not found in
// the database.
func TestPutDaemonConfigReviewNoConfig(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Create Kea app instance with a DHCPv4 daemon with no configuration
	// assigned.
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234, false)
	app := &dbmodel.App{
		MachineID:    machine.ID,
		Machine:      machine,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	daemons, err := dbmodel.AddApp(db, app)
	require.NoError(t, err)

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	params := services.PutDaemonConfigReviewParams{
		ID: daemons[0].ID,
	}
	rsp := rapi.PutDaemonConfigReview(ctx, params)
	require.IsType(t, &services.PutDaemonConfigReviewDefault{}, rsp)
	defaultRsp := rsp.(*services.PutDaemonConfigReviewDefault)
	require.NotNil(t, defaultRsp)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, fmt.Sprintf("configuration not found for daemon with id %d", daemons[0].ID),
		*defaultRsp.Payload.Message)
}
