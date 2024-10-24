package restservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	apps "isc.org/stork/server/apps"
	"isc.org/stork/server/apps/kea"
	appstest "isc.org/stork/server/apps/test"
	"isc.org/stork/server/configreview"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	"isc.org/stork/server/gen/restapi/operations/services"
	storktest "isc.org/stork/server/test/dbmodel"
	storkutil "isc.org/stork/util"
)

// Test that GetDaemonConfig works for Kea daemon with assigned configuration.
func TestGetDaemonConfigForKeaDaemonWithAssignedConfiguration(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	require.Equal(t, configDhcp4, okRsp.Payload.Config)
	require.NotZero(t, okRsp.Payload.AppID)
	require.Equal(t, "test-app", okRsp.Payload.AppName)
	require.Equal(t, "dhcp4", okRsp.Payload.DaemonName)
	require.Equal(t, "kea", okRsp.Payload.AppType)
	require.True(t, okRsp.Payload.Editable)

	params = services.GetDaemonConfigParams{
		ID: app.Daemons[1].ID,
	}

	// Check Dhcp6 daemon
	rsp = rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonConfigOK)
	require.NotEmpty(t, okRsp.Payload)
	require.Equal(t, configDhcp6, okRsp.Payload.Config)
	require.NotZero(t, okRsp.Payload.AppID)
	require.Equal(t, "test-app", okRsp.Payload.AppName)
	require.Equal(t, "dhcp6", okRsp.Payload.DaemonName)
	require.Equal(t, "kea", okRsp.Payload.AppType)
	require.True(t, okRsp.Payload.Editable)
}

// Test that GetDaemonConfig returns the secrets for super admin.
func TestGetDaemonConfigWithSecretsForSuperAdmin(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	require.Equal(t, configDhcp4, okRsp.Payload.Config)
}

// Test that GetDaemonConfig hides the secrets for standard users.
func TestGetDaemonConfigWithoutSecretsForAdmin(t *testing.T) {
	// Test database initialization
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// REST API mock
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// Create "standard" user (without any special group)
	user := &dbmodel.SystemUser{
		Email:    "john@example.org",
		Lastname: "Smith",
		Name:     "John",
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
	require.Equal(t, expected, okRsp.Payload.Config)
}

// Test that GetDaemonConfig returns correct editable flag value when
// daemon is not active and not monitored.
func TestGetDaemonConfigForNonActiveKeaDaemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	app.Daemons[0].Active = false

	configDhcp6, err := dbmodel.NewKeaConfigFromJSON(`{
		"Dhcp6": { }
    }`)
	require.NoError(t, err)

	app.Daemons[1].KeaDaemon.Config = configDhcp6
	app.Daemons[1].Monitored = false

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
	require.Equal(t, configDhcp4, okRsp.Payload.Config)
	require.NotZero(t, okRsp.Payload.AppID)
	require.Equal(t, "test-app", okRsp.Payload.AppName)
	require.Equal(t, "dhcp4", okRsp.Payload.DaemonName)
	require.Equal(t, "kea", okRsp.Payload.AppType)
	require.False(t, okRsp.Payload.Editable)

	params = services.GetDaemonConfigParams{
		ID: app.Daemons[1].ID,
	}

	// Check Dhcp6 daemon
	rsp = rapi.GetDaemonConfig(ctx, params)
	require.IsType(t, &services.GetDaemonConfigOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonConfigOK)
	require.NotEmpty(t, okRsp.Payload)
	require.Equal(t, configDhcp6, okRsp.Payload.Config)
	require.NotZero(t, okRsp.Payload.AppID)
	require.Equal(t, "test-app", okRsp.Payload.AppName)
	require.Equal(t, "dhcp6", okRsp.Payload.DaemonName)
	require.Equal(t, "kea", okRsp.Payload.AppType)
	require.False(t, okRsp.Payload.Editable)
}

// Test that GetDaemonConfig returns HTTP Not Found status for Kea daemon
// without assigned configuration.
func TestGetDaemonConfigForKeaDaemonWithoutAssignedConfiguration(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	msg := fmt.Sprintf("Config not assigned for daemon with ID %d", params.ID)
	require.Equal(t, msg, *defaultRsp.Payload.Message)
}

// Test that GetDaemonConfig returns HTTP Bad Request status for not-Kea daemon.
func TestGetDaemonConfigForBind9Daemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	msg := fmt.Sprintf("Daemon with ID %d is not a Kea daemon", params.ID)
	require.Equal(t, msg, *defaultRsp.Payload.Message)
}

// Test that GetDaemonConfig returns HTTP Bad Request for not exist daemon.
func TestGetDaemonConfigForNonExistsDaemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	msg := fmt.Sprintf("Cannot find daemon with ID %d", params.ID)
	require.Equal(t, msg, *defaultRsp.Payload.Message)
}

// Test that GetDaemonConfig returns HTTP Internal Server Error status for failed database connection.
func TestGetDaemonConfigForDatabaseError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	msg := fmt.Sprintf("Cannot get daemon with ID %d from db", params.ID)
	require.Equal(t, msg, *defaultRsp.Payload.Message)
}

// Test that the converted DHCP options are included in the response.
func TestGetDaemonConfigWithDHCPOptions(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	rapi, _ := NewRestAPI(dbSettings, db, fa, fd, lookup)
	ctx := context.Background()

	user, _ := dbmodel.GetUserByID(rapi.DB, 1)
	ctx, _ = rapi.SessionManager.Load(ctx, "")
	_ = rapi.SessionManager.LoginHandler(ctx, user)

	serverConfig := `{
		"Dhcp4": {
			"option-data": [{
				"always-send": true,
				"code": 3,
				"csv-format": true,
				"data": "192.0.3.1",
				"name": "routers",
				"space": "dhcp4"
			}]
		}
	}`

	server, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server.Configure(serverConfig)
	require.NoError(t, err)
	kea, _ := server.GetKea()
	require.NotNil(t, kea)
	require.NotEmpty(t, kea.Daemons)

	params := services.GetDaemonConfigParams{
		ID: kea.Daemons[0].ID,
	}

	// Act
	rsp := rapi.GetDaemonConfig(ctx, params)

	// Assert
	require.IsType(t, &services.GetDaemonConfigOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonConfigOK)
	require.NotEmpty(t, okRsp.Payload)
	require.Len(t, okRsp.Payload.Options.Options, 1)
	option := okRsp.Payload.Options.Options[0]
	require.True(t, option.AlwaysSend)
	require.EqualValues(t, 3, option.Code)
	require.Empty(t, option.Encapsulate)
	require.EqualValues(t, 4, option.Universe)
	require.Len(t, option.Fields, 1)
	field := option.Fields[0]
	require.Len(t, field.Values, 1)
	require.EqualValues(t, "192.0.3.1", field.Values[0])
	require.EqualValues(t, "ipv4-address", field.FieldType)
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

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// Create several config reports - two for first daemon and two for the
	// second daemon (including one empty).
	content1 := "funny review contents for {daemon} and {daemon}"
	content2 := "another funny review contents for {daemon}"
	content3 := "review contents for another daemon"
	configReports := []dbmodel.ConfigReport{
		{
			CheckerName: "name 1",
			Content:     &content1,
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
			Content:     &content2,
			DaemonID:    app.Daemons[0].ID,
			RefDaemons: []*dbmodel.Daemon{
				{
					ID: app.Daemons[1].ID,
				},
			},
		},
		{
			CheckerName: "name 3",
			Content:     &content3,
			DaemonID:    app.Daemons[1].ID,
			RefDaemons: []*dbmodel.Daemon{
				{
					ID: app.Daemons[1].ID,
				},
			},
		},
		{
			CheckerName: "empty 4",
			Content:     nil,
			DaemonID:    app.Daemons[1].ID,
			RefDaemons:  []*dbmodel.Daemon{},
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
	require.EqualValues(t, 2, okRsp.Payload.TotalIssues)
	require.EqualValues(t, 2, okRsp.Payload.TotalReports)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, "name 1", okRsp.Payload.Items[0].Checker)
	require.Equal(t, "funny review contents for <daemon id=\"1\" name=\"dhcp4\" appId=\"1\" appType=\"kea\"> and <daemon id=\"2\" name=\"dhcp6\" appId=\"1\" appType=\"kea\">",
		*okRsp.Payload.Items[0].Content)

	require.EqualValues(t, "name 2", okRsp.Payload.Items[1].Checker)
	require.Equal(t, "another funny review contents for <daemon id=\"2\" name=\"dhcp6\" appId=\"1\" appType=\"kea\">", *okRsp.Payload.Items[1].Content)

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
	require.EqualValues(t, 2, okRsp.Payload.TotalIssues)
	require.EqualValues(t, 2, okRsp.Payload.TotalReports)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, "name 1", okRsp.Payload.Items[0].Checker)
	require.Equal(t, "funny review contents for <daemon id=\"1\" name=\"dhcp4\" appId=\"1\" appType=\"kea\"> and <daemon id=\"2\" name=\"dhcp6\" appId=\"1\" appType=\"kea\">",
		*okRsp.Payload.Items[0].Content)
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
	require.EqualValues(t, 2, okRsp.Payload.TotalIssues)
	require.EqualValues(t, 2, okRsp.Payload.TotalReports)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, "name 2", okRsp.Payload.Items[0].Checker)
	require.Equal(t, "another funny review contents for <daemon id=\"2\" name=\"dhcp6\" appId=\"1\" appType=\"kea\">", *okRsp.Payload.Items[0].Content)

	// Try to fetch the config reports for the second daemon.
	params = services.GetDaemonConfigReportsParams{
		ID: app.Daemons[1].ID,
	}
	rsp = rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonConfigReportsOK)

	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.EqualValues(t, 1, okRsp.Payload.TotalIssues)
	require.EqualValues(t, 2, okRsp.Payload.TotalReports)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, "name 3", okRsp.Payload.Items[0].Checker)
	require.Equal(t, "review contents for another daemon", *okRsp.Payload.Items[0].Content)

	require.EqualValues(t, "empty 4", okRsp.Payload.Items[1].Checker)
	require.Nil(t, okRsp.Payload.Items[1].Content)

	// If the only issues flag is provided, it should return only one report.
	issuesOnly := true
	params.IssuesOnly = &issuesOnly
	rsp = rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonConfigReportsOK)

	require.EqualValues(t, 1, okRsp.Payload.Total)
	require.EqualValues(t, 1, okRsp.Payload.TotalIssues)
	require.EqualValues(t, 2, okRsp.Payload.TotalReports)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, "name 3", okRsp.Payload.Items[0].Checker)
	require.Equal(t, "review contents for another daemon", *okRsp.Payload.Items[0].Content)

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

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
	require.NoError(t, err)
	ctx := context.Background()

	params := services.GetDaemonConfigReportsParams{
		ID: 1,
	}
	rsp := rapi.GetDaemonConfigReports(ctx, params)
	require.IsType(t, &services.GetDaemonConfigReportsDefault{}, rsp)
	defaultRsp := rsp.(*services.GetDaemonConfigReportsDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "Cannot get configuration review for daemon with ID 1 from db",
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

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	require.Equal(t, "BeginReview", fd.CallLog[0].CallName)

	// Try to create a new review for a non-existing daemon.
	params.ID++
	rsp = rapi.PutDaemonConfigReview(ctx, params)
	require.IsType(t, &services.PutDaemonConfigReviewDefault{}, rsp)
	defaultRsp := rsp.(*services.PutDaemonConfigReviewDefault)
	require.NotNil(t, defaultRsp)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Contains(t, *defaultRsp.Payload.Message, "Cannot find daemon with ID")
}

// Test that HTTP internal server error is returned when the database
// connection fails while creating new config review.
func TestPutDaemonConfigReviewDatabaseError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	// Close the database connection to cause the failure while
	// communicating with the database
	teardown()

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	require.Equal(t, "Cannot get daemon with ID 1 from db", *defaultRsp.Payload.Message)
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

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	require.Equal(t, fmt.Sprintf("Daemon with ID %d is not a Kea daemon", daemons[0].ID),
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

	fa := agentcommtest.NewFakeAgents(nil, nil)
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(dbSettings, db, fa, fd)
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
	require.Equal(t, fmt.Sprintf("Configuration not found for daemon with ID %d", daemons[0].ID),
		*defaultRsp.Payload.Message)
}

// Test that the config checker metadata is converted properly to API structure.
func TestConvertConfigCheckerMetadataToRestAPI(t *testing.T) {
	// Arrange
	metadata := configreview.CheckerMetadata{
		Name:            "foo",
		Triggers:        configreview.Triggers{configreview.ConfigModified, configreview.ManualRun},
		Selectors:       configreview.DispatchGroupSelectors{configreview.Bind9Daemon, configreview.KeaDHCPDaemon},
		GloballyEnabled: true,
		State:           configreview.CheckerStateEnabled,
	}

	// Act
	payload := convertConfigCheckerMetadataToRestAPI([]*configreview.CheckerMetadata{&metadata})

	// Assert
	require.Len(t, payload.Items, 1)
	require.EqualValues(t, 1, payload.Total)
	apiMetadata := payload.Items[0]
	require.EqualValues(t, "foo", *apiMetadata.Name)
	require.Contains(t, apiMetadata.Triggers, "manual")
	require.Contains(t, apiMetadata.Triggers, "config change")
	require.Contains(t, apiMetadata.Selectors, "bind9-daemon")
	require.Contains(t, apiMetadata.Selectors, "kea-dhcp-daemon")
	require.EqualValues(t, "enabled", apiMetadata.State)
	require.True(t, *apiMetadata.GloballyEnabled)
}

// Test that the config checker state is properly converted from the REST API enum.
func TestConvertConfigCheckerStateFromRestAPI(t *testing.T) {
	// Act
	disabled, disabledOk := convertConfigCheckerStateFromRestAPI(models.ConfigCheckerStateDisabled)
	enabled, enabledOk := convertConfigCheckerStateFromRestAPI(models.ConfigCheckerStateEnabled)
	inherit, inheritOk := convertConfigCheckerStateFromRestAPI(models.ConfigCheckerStateInherit)
	unknown, unknownOk := convertConfigCheckerStateFromRestAPI(models.ConfigCheckerState("unknown"))

	// Assert
	require.EqualValues(t, configreview.CheckerStateDisabled, disabled)
	require.True(t, disabledOk)
	require.EqualValues(t, configreview.CheckerStateEnabled, enabled)
	require.True(t, enabledOk)
	require.EqualValues(t, configreview.CheckerStateInherit, inherit)
	require.True(t, inheritOk)
	require.EqualValues(t, configreview.CheckerStateEnabled, unknown)
	require.False(t, unknownOk)
}

// Test that the global configuration checkers are returned properly.
func TestGetGlobalConfigCheckers(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	fd := &storktest.FakeDispatcher{}
	fd.SetCheckerState(nil, "foo", configreview.CheckerStateDisabled)

	rapi, _ := NewRestAPI(dbSettings, db, fd)

	// Act
	ctx := context.Background()
	params := services.GetGlobalConfigCheckersParams{}
	rsp := rapi.GetGlobalConfigCheckers(ctx, params)

	// Assert
	require.IsType(t, &services.GetGlobalConfigCheckersOK{}, rsp)
	okRsp := rsp.(*services.GetGlobalConfigCheckersOK)
	require.NotNil(t, okRsp)
	require.EqualValues(t, 1, okRsp.Payload.Total)
	require.NotEmpty(t, okRsp.Payload.Items)
}

// Test that the configuration checkers for a given daemon are returned properly.
func TestGetDaemonConfigCheckers(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, m)
	app := &dbmodel.App{
		Type: dbmodel.AppTypeKea,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true),
		},
		MachineID: m.ID,
	}
	daemons, _ := dbmodel.AddApp(db, app)
	daemon := daemons[0]

	fd := &storktest.FakeDispatcher{}
	fd.SetCheckerState(daemon, "foo", configreview.CheckerStateDisabled)
	fd.SetCheckerState(daemon, "bar", configreview.CheckerStateEnabled)

	rapi, _ := NewRestAPI(dbSettings, db, fd)

	// Act
	ctx := context.Background()
	params := services.GetDaemonConfigCheckersParams{
		ID: daemon.ID,
	}
	rsp := rapi.GetDaemonConfigCheckers(ctx, params)

	// Assert
	require.IsType(t, &services.GetDaemonConfigCheckersOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonConfigCheckersOK)
	require.NotNil(t, okRsp)
	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.NotEmpty(t, okRsp.Payload.Items)
}

// Test that the configuration checkers for a non-existing daemon causes no panic.
func TestGetDaemonConfigCheckersForMissingDaemon(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(dbSettings, db, fd)

	// Act
	ctx := context.Background()
	params := services.GetDaemonConfigCheckersParams{
		ID: 1,
	}
	rsp := rapi.GetDaemonConfigCheckers(ctx, params)

	// Assert
	require.IsType(t, &services.GetDaemonConfigCheckersDefault{}, rsp)
	defaultRsp := rsp.(*services.GetDaemonConfigCheckersDefault)
	require.NotNil(t, defaultRsp)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
}

// Test that the global config checkers are inserted properly.
func TestPutNewGlobalConfigCheckerPreferences(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(dbSettings, db, fd)

	// Act
	ctx := context.Background()
	params := services.PutGlobalConfigCheckerPreferencesParams{
		Changes: &models.ConfigCheckerPreferences{
			Total: 3,
			Items: []*models.ConfigCheckerPreference{
				{
					Name:  "foo",
					State: "enabled",
				},
				{
					Name:  "bar",
					State: "disabled",
				},
				{
					Name:  "baz",
					State: "inherit",
				},
			},
		},
	}
	rsp := rapi.PutGlobalConfigCheckerPreferences(ctx, params)

	// Assert
	require.IsType(t, &services.GetDaemonConfigCheckersOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonConfigCheckersOK)
	require.NotNil(t, okRsp)
	require.EqualValues(t, 1, okRsp.Payload.Total)
	require.EqualValues(t, "bar", *okRsp.Payload.Items[0].Name)
	preferences, _ := dbmodel.GetCheckerPreferences(db, 0)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.False(t, preferences[0].Enabled)
}

// Test that the global config checkers are updated properly.
func TestPutUpdateGlobalConfigCheckerPreferences(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(dbSettings, db, fd)

	// Act
	rsp1 := rapi.PutGlobalConfigCheckerPreferences(context.Background(), services.PutGlobalConfigCheckerPreferencesParams{
		Changes: &models.ConfigCheckerPreferences{
			Total: 2,
			Items: []*models.ConfigCheckerPreference{
				{
					Name:  "foo",
					State: "disabled",
				},
				{
					Name:  "bar",
					State: "disabled",
				},
			},
		},
	})

	rsp2 := rapi.PutGlobalConfigCheckerPreferences(context.Background(), services.PutGlobalConfigCheckerPreferencesParams{
		Changes: &models.ConfigCheckerPreferences{
			Total: 2,
			Items: []*models.ConfigCheckerPreference{
				{
					Name:  "foo",
					State: "inherit",
				},
				{
					Name:  "bar",
					State: "enabled",
				},
			},
		},
	})

	// Assert
	require.IsType(t, &services.GetDaemonConfigCheckersOK{}, rsp1)
	require.IsType(t, &services.GetDaemonConfigCheckersOK{}, rsp2)
	okRsp := rsp2.(*services.GetDaemonConfigCheckersOK)
	require.NotNil(t, okRsp)
	require.EqualValues(t, 0, okRsp.Payload.Total)
	require.Empty(t, okRsp.Payload.Items)
	preferences, _ := dbmodel.GetCheckerPreferences(db, 0)
	require.Empty(t, preferences)
}

// Test that inserting the daemon config checkers produces a proper API response.
func TestPutDaemonConfigCheckerPreferencesAPIResponse(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, m)
	app := &dbmodel.App{
		Type: dbmodel.AppTypeKea,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true),
		},
		MachineID: m.ID,
	}
	daemons, _ := dbmodel.AddApp(db, app)
	daemon := daemons[0]

	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(dbSettings, db, fd)

	// Act
	ctx := context.Background()
	params := services.PutDaemonConfigCheckerPreferencesParams{
		ID: daemon.ID,
		Changes: &models.ConfigCheckerPreferences{
			Total: 1,
			Items: []*models.ConfigCheckerPreference{
				{
					Name: "foo", State: "enabled",
				},
			},
		},
	}
	rsp := rapi.PutDaemonConfigCheckerPreferences(ctx, params)

	// Assert
	require.IsType(t, &services.PutDaemonConfigCheckerPreferencesOK{}, rsp)
	okRsp := rsp.(*services.PutDaemonConfigCheckerPreferencesOK)
	require.NotNil(t, okRsp)
	require.EqualValues(t, 1, okRsp.Payload.Total)
	require.EqualValues(t, "foo", *okRsp.Payload.Items[0].Name)
	require.EqualValues(t, "enabled", okRsp.Payload.Items[0].State)
}

// Test that new daemon config checker preferences are inserted properly.
func TestPutNewDaemonConfigCheckers(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, m)
	app := &dbmodel.App{
		Type: dbmodel.AppTypeKea,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true),
		},
		MachineID: m.ID,
	}
	daemons, _ := dbmodel.AddApp(db, app)
	daemon := daemons[0]

	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(dbSettings, db, fd)

	// Act
	ctx := context.Background()
	params := services.PutDaemonConfigCheckerPreferencesParams{
		ID: daemon.ID,
		Changes: &models.ConfigCheckerPreferences{
			Total: 1,
			Items: []*models.ConfigCheckerPreference{
				{
					Name: "foo", State: "enabled",
				},
			},
		},
	}
	rsp := rapi.PutDaemonConfigCheckerPreferences(ctx, params)

	// Assert
	require.IsType(t, &services.PutDaemonConfigCheckerPreferencesOK{}, rsp)
	preferences, err := dbmodel.GetCheckerPreferences(db, daemon.ID)
	require.NoError(t, err)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "foo", preferences[0].CheckerName)
	require.True(t, preferences[0].Enabled)
}

// Test that the config checker preferences are updated properly.
func TestPutDaemonConfigCheckerPreferencesUpdate(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, m)
	app := &dbmodel.App{
		Type: dbmodel.AppTypeKea,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true),
		},
		MachineID: m.ID,
	}
	daemons, _ := dbmodel.AddApp(db, app)
	daemon := daemons[0]

	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(dbSettings, db, fd)

	// Act
	// Initialize the config checker preferences.
	rsp1 := rapi.PutDaemonConfigCheckerPreferences(
		context.Background(),
		services.PutDaemonConfigCheckerPreferencesParams{
			ID: daemon.ID,
			Changes: &models.ConfigCheckerPreferences{
				Total: 2,
				Items: []*models.ConfigCheckerPreference{
					{
						Name:  "foo",
						State: "enabled",
					},
					{
						Name:  "bar",
						State: "disabled",
					},
				},
			},
		},
	)
	// Modify the config checker preferences.
	rsp2 := rapi.PutDaemonConfigCheckerPreferences(
		context.Background(),
		services.PutDaemonConfigCheckerPreferencesParams{
			ID: daemon.ID,
			Changes: &models.ConfigCheckerPreferences{
				Total: 3,
				Items: []*models.ConfigCheckerPreference{
					// Update the existing preference.
					{
						Name:  "foo",
						State: "disabled",
					},
					// Delete the existing preference.
					{
						Name:  "bar",
						State: "inherit",
					},
					// Add new preference.
					{
						Name:  "baz",
						State: "enabled",
					},
				},
			},
		},
	)

	// Assert
	require.IsType(t, &services.PutDaemonConfigCheckerPreferencesOK{}, rsp1)
	require.IsType(t, &services.PutDaemonConfigCheckerPreferencesOK{}, rsp2)
	okRsp := rsp2.(*services.PutDaemonConfigCheckerPreferencesOK)
	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.EqualValues(t, "baz", *okRsp.Payload.Items[0].Name)
	require.EqualValues(t, "enabled", okRsp.Payload.Items[0].State)
	require.EqualValues(t, "foo", *okRsp.Payload.Items[1].Name)
	require.EqualValues(t, "disabled", okRsp.Payload.Items[1].State)
	preferences, _ := dbmodel.GetCheckerPreferences(db, daemon.ID)
	require.Len(t, preferences, 2)
	require.EqualValues(t, "baz", preferences[0].CheckerName)
	require.True(t, preferences[0].Enabled)
	require.EqualValues(t, "foo", preferences[1].CheckerName)
	require.False(t, preferences[1].Enabled)
}

// Test that updating the daemon config checkers for non-existing daemon causes
// no panic.
func TestPutDaemonConfigCheckerPreferencesForMissingDaemon(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(dbSettings, db, fd)

	// Act
	ctx := context.Background()
	params := services.PutDaemonConfigCheckerPreferencesParams{
		ID: 1,
		Changes: &models.ConfigCheckerPreferences{
			Total: 0,
			Items: []*models.ConfigCheckerPreference{},
		},
	}
	rsp := rapi.PutDaemonConfigCheckerPreferences(ctx, params)

	// Assert
	require.IsType(t, &services.PutDaemonConfigCheckerPreferencesDefault{}, rsp)
	defaultRsp := rsp.(*services.PutDaemonConfigCheckerPreferencesDefault)
	require.NotNil(t, defaultRsp)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	daemonID := int64(1)
	preferences, _ := dbmodel.GetCheckerPreferences(db, daemonID)
	require.Empty(t, preferences)
}

// Test resetting Kea daemons' config hashes.
func TestDeleteKeaConfigHashes(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(dbSettings, db, fd)

	kea, err := dbmodeltest.NewKea(db)
	require.NoError(t, err)
	require.NotNil(t, kea)

	dhcp4, err := kea.NewKeaDHCPv4Server()
	require.NoError(t, err)
	require.NotNil(t, dhcp4)
	dhcp6, err := kea.NewKeaDHCPv6Server()
	require.NoError(t, err)
	require.NotNil(t, dhcp6)

	require.NoError(t, dhcp4.Configure(`{ "Dhcp4": { } }`))
	require.NoError(t, dhcp6.Configure(`{ "Dhcp4": { } }`))

	daemons, err := dbmodel.GetKeaDHCPDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 2)
	require.NotEmpty(t, daemons[0].KeaDaemon.ConfigHash)
	require.NotEmpty(t, daemons[1].KeaDaemon.ConfigHash)

	ctx := context.Background()
	params := services.DeleteKeaDaemonConfigHashesParams{}
	rsp := rapi.DeleteKeaDaemonConfigHashes(ctx, params)
	require.IsType(t, &services.DeleteKeaDaemonConfigHashesOK{}, rsp)

	daemons, err = dbmodel.GetKeaDHCPDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 2)
	require.Empty(t, daemons[0].KeaDaemon.ConfigHash)
	require.Empty(t, daemons[1].KeaDaemon.ConfigHash)
}

// Test that internal server error is returned when the database goes
// away during resetting the Kea config hashes.
func TestDeleteKeaConfigHashesError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	// Close the database. It should result in an error trying to reset the hashes.
	teardown()

	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(dbSettings, db, fd)

	ctx := context.Background()
	params := services.DeleteKeaDaemonConfigHashesParams{}
	rsp := rapi.DeleteKeaDaemonConfigHashes(ctx, params)
	require.IsType(t, &services.DeleteKeaDaemonConfigHashesDefault{}, rsp)
	defaultRsp := rsp.(*services.DeleteKeaDaemonConfigHashesDefault)
	require.NotNil(t, defaultRsp)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
}

// Test the calls for creating new transaction and updating global Kea DHCPv4 parameters.
func TestUpdateGlobalParameters4BeginSubmit(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"valid-lifetime": 2222
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)
	err = server1.SetVersion("3.0.0")
	require.NoError(t, err)

	app1, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app1, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)
	err = server2.SetVersion("3.0.0")
	require.NoError(t, err)

	app2, err := server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app2, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	daemonIDs := []int64{app1.Daemons[0].GetID(), app2.Daemons[0].GetID()}

	daemons, err := dbmodel.GetDaemonsByIDs(db, daemonIDs)
	require.NoError(t, err)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.UpdateKeaGlobalParametersBeginParams{
		Request: &models.UpdateKeaDaemonsGlobalParametersBeginRequest{
			DaemonIds: daemonIDs,
		},
	}
	rsp := rapi.UpdateKeaGlobalParametersBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateKeaGlobalParametersBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID and configurations.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Configs)
	require.Len(t, contents.Configs, 2)
	require.EqualValues(t, daemons[0].GetID(), contents.Configs[0].DaemonID)
	require.Equal(t, dbmodel.DaemonNameDHCPv4, contents.Configs[0].DaemonName)
	require.Equal(t, "3.0.0", contents.Configs[0].DaemonVersion)
	require.EqualValues(t, daemons[1].GetID(), contents.Configs[1].DaemonID)
	require.Equal(t, dbmodel.DaemonNameDHCPv4, contents.Configs[1].DaemonName)
	require.Equal(t, "3.0.0", contents.Configs[1].DaemonVersion)

	// Submit transaction.

	partialConfig := &models.KeaConfigurableGlobalParameters{
		KeaConfigAssortedGlobalParameters: models.KeaConfigAssortedGlobalParameters{
			Allocator:                     storkutil.Ptr("flq"),
			Authoritative:                 storkutil.Ptr(true),
			EarlyGlobalReservationsLookup: storkutil.Ptr(false),
			EchoClientID:                  storkutil.Ptr(true),
			HostReservationIdentifiers:    []string{"hw-address", "client-id"},
		},
		KeaConfigCacheParameters: models.KeaConfigCacheParameters{
			CacheThreshold: storkutil.Ptr(float32(0.2)),
		},
		KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
			DdnsGeneratedPrefix:        storkutil.Ptr("myhost.example.org"),
			DdnsOverrideClientUpdate:   storkutil.Ptr(true),
			DdnsOverrideNoUpdate:       storkutil.Ptr(false),
			DdnsQualifyingSuffix:       storkutil.Ptr("example.org"),
			DdnsReplaceClientName:      storkutil.Ptr("never"),
			DdnsSendUpdates:            storkutil.Ptr(false),
			DdnsUpdateOnRenew:          storkutil.Ptr(true),
			DdnsUseConflictResolution:  storkutil.Ptr(true),
			DdnsConflictResolutionMode: storkutil.Ptr("check-with-dhcid"),
		},
		KeaConfigDhcpDdnsParameters: models.KeaConfigDhcpDdnsParameters{
			DhcpDdnsEnableUpdates: storkutil.Ptr(true),
			DhcpDdnsMaxQueueSize:  storkutil.Ptr(int64(100)),
			DhcpDdnsNcrFormat:     storkutil.Ptr("JSON"),
			DhcpDdnsNcrProtocol:   storkutil.Ptr("UDP"),
			DhcpDdnsSenderIP:      storkutil.Ptr("192.0.2.1"),
			DhcpDdnsSenderPort:    storkutil.Ptr(int64(8080)),
			DhcpDdnsServerIP:      storkutil.Ptr("192.0.2.2"),
			DhcpDdnsServerPort:    storkutil.Ptr(int64(8081)),
		},
		KeaConfigExpiredLeasesProcessingParameters: models.KeaConfigExpiredLeasesProcessingParameters{
			ExpiredFlushReclaimedTimerWaitTime: storkutil.Ptr(int64(12)),
			ExpiredHoldReclaimedTime:           storkutil.Ptr(int64(13)),
			ExpiredMaxReclaimLeases:            storkutil.Ptr(int64(14)),
			ExpiredMaxReclaimTime:              storkutil.Ptr(int64(15)),
			ExpiredReclaimTimerWaitTime:        storkutil.Ptr(int64(16)),
			ExpiredUnwarnedReclaimCycles:       storkutil.Ptr(int64(17)),
		},
		KeaConfigReservationParameters: models.KeaConfigReservationParameters{
			ReservationsGlobal:    storkutil.Ptr(true),
			ReservationsInSubnet:  storkutil.Ptr(false),
			ReservationsOutOfPool: storkutil.Ptr(true),
		},
		KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
			ValidLifetime: storkutil.Ptr(int64(1111)),
		},
		DHCPOptions: models.DHCPOptions{
			Options: []*models.DHCPOption{
				{
					Code:       42,
					AlwaysSend: true,
					Fields: []*models.DHCPOptionField{
						{
							FieldType: "int32",
							Values:    []string{"4242"},
						},
					},
					Universe: 4,
				},
			},
		},
	}
	params2 := dhcp.UpdateKeaGlobalParametersSubmitParams{
		ID: transactionID,
		Request: &models.UpdateKeaDaemonsGlobalParametersSubmitRequest{
			Configs: []*models.KeaDaemonConfigurableGlobalParameters{
				{
					DaemonID:      daemons[0].GetID(),
					DaemonName:    dbmodel.DaemonNameDHCPv4,
					PartialConfig: partialConfig,
				},
				{
					DaemonID:      daemons[1].GetID(),
					DaemonName:    dbmodel.DaemonNameDHCPv4,
					PartialConfig: partialConfig,
				},
			},
		},
	}

	rsp2 := rapi.UpdateKeaGlobalParametersSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersSubmitOK{}, rsp2)

	// It should result in sending commands to two Kea servers. Each server
	// receives the config-set and config-write commands.
	require.Len(t, fa.RecordedCommands, 4)

	for i, c := range fa.RecordedCommands {
		switch {
		case i < 2:
			require.JSONEq(t,
				`{
					"command": "config-set",
					"service": [ "dhcp4" ],
					"arguments": {
						"Dhcp4": {
							"allocator": "flq",
							"authoritative": true,
							"early-global-reservations-lookup": false,
							"echo-client-id": true,
							"host-reservation-identifiers": [ "hw-address", "client-id" ],
							"cache-threshold": 0.2,
							"ddns-generated-prefix": "myhost.example.org",
							"ddns-override-client-update": true,
							"ddns-override-no-update": false,
							"ddns-qualifying-suffix": "example.org",
							"ddns-replace-client-name": "never",
							"ddns-send-updates": false,
							"ddns-update-on-renew": true,
							"ddns-use-conflict-resolution": true,
							"ddns-conflict-resolution-mode": "check-with-dhcid",
							"dhcp-ddns": {
								"enable-updates": true,
								"max-queue-size": 100,
								"ncr-format": "JSON",
								"ncr-protocol": "UDP",
								"sender-ip": "192.0.2.1",
								"sender-port": 8080,
								"server-ip": "192.0.2.2",
								"server-port": 8081
							},
							"expired-leases-processing": {
								"flush-reclaimed-timer-wait-time": 12,
								"hold-reclaimed-time": 13,
								"max-reclaim-leases": 14,
								"max-reclaim-time": 15,
								"reclaim-timer-wait-time": 16,
								"unwarned-reclaim-cycles": 17
							},
							"option-data": [
								{
									"code": 42,
									"always-send": true,
									"csv-format": true,
									"data": "4242",
									"space": "dhcp4"
								}
							],
							"reservations-global": true,
							"reservations-in-subnet": false,
							"reservations-out-of-pool": true,
							"valid-lifetime": 1111
						}
					}
				}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": [ "dhcp4" ]
				}`,
				c.Marshal())
		}
	}

	// Make sure that the transaction is done.
	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.

	if cctx != nil {
		cm.Done(cctx)
	}

	require.Nil(t, cctx)

	// Make sure that the daemon configurations have been updated in the database.
	updatedDaemons, err := dbmodel.GetDaemonsByIDs(db, daemonIDs)
	require.NoError(t, err)
	require.Len(t, updatedDaemons, 2)
	for _, daemon := range updatedDaemons {
		require.NotNil(t, daemon.KeaDaemon)
		config := daemon.KeaDaemon.Config
		require.NotNil(t, config)

		require.NotNil(t, config.GetValidLifetimeParameters().ValidLifetime)
		require.EqualValues(t, 1111, *config.GetValidLifetimeParameters().ValidLifetime)
	}
}

// Test the calls for creating new transaction and updating global Kea DHCPv6 sparameters.
func TestUpdateGlobalParameters6BeginSubmit(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp6": {
			"valid-lifetime": 2222
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)
	err = server1.SetVersion("3.0.0")
	require.NoError(t, err)

	app1, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app1, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)
	err = server2.SetVersion("3.0.0")
	require.NoError(t, err)

	app2, err := server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app2, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	daemonIDs := []int64{app1.Daemons[0].GetID(), app2.Daemons[0].GetID()}

	daemons, err := dbmodel.GetDaemonsByIDs(db, daemonIDs)
	require.NoError(t, err)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.UpdateKeaGlobalParametersBeginParams{
		Request: &models.UpdateKeaDaemonsGlobalParametersBeginRequest{
			DaemonIds: daemonIDs,
		},
	}
	rsp := rapi.UpdateKeaGlobalParametersBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateKeaGlobalParametersBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID and configurations.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Configs)
	require.Len(t, contents.Configs, 2)
	require.EqualValues(t, daemons[0].GetID(), contents.Configs[0].DaemonID)
	require.Equal(t, dbmodel.DaemonNameDHCPv6, contents.Configs[0].DaemonName)
	require.Equal(t, "3.0.0", contents.Configs[0].DaemonVersion)
	require.EqualValues(t, daemons[1].GetID(), contents.Configs[1].DaemonID)
	require.Equal(t, dbmodel.DaemonNameDHCPv6, contents.Configs[1].DaemonName)
	require.Equal(t, "3.0.0", contents.Configs[0].DaemonVersion)

	// Submit transaction.

	partialConfig := &models.KeaConfigurableGlobalParameters{
		KeaConfigAssortedGlobalParameters: models.KeaConfigAssortedGlobalParameters{
			Allocator:                     storkutil.Ptr("flq"),
			EarlyGlobalReservationsLookup: storkutil.Ptr(false),
			HostReservationIdentifiers:    []string{"hw-address", "client-id"},
			PdAllocator:                   storkutil.Ptr("random"),
		},
		KeaConfigCacheParameters: models.KeaConfigCacheParameters{
			CacheThreshold: storkutil.Ptr(float32(0.2)),
		},
		KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
			DdnsGeneratedPrefix:        storkutil.Ptr("myhost.example.org"),
			DdnsOverrideClientUpdate:   storkutil.Ptr(true),
			DdnsOverrideNoUpdate:       storkutil.Ptr(false),
			DdnsQualifyingSuffix:       storkutil.Ptr("example.org"),
			DdnsReplaceClientName:      storkutil.Ptr("never"),
			DdnsSendUpdates:            storkutil.Ptr(false),
			DdnsUpdateOnRenew:          storkutil.Ptr(true),
			DdnsUseConflictResolution:  storkutil.Ptr(true),
			DdnsConflictResolutionMode: storkutil.Ptr("check-with-dhcid"),
		},
		KeaConfigDhcpDdnsParameters: models.KeaConfigDhcpDdnsParameters{
			DhcpDdnsEnableUpdates: storkutil.Ptr(true),
			DhcpDdnsMaxQueueSize:  storkutil.Ptr(int64(100)),
			DhcpDdnsNcrFormat:     storkutil.Ptr("JSON"),
			DhcpDdnsNcrProtocol:   storkutil.Ptr("UDP"),
			DhcpDdnsSenderIP:      storkutil.Ptr("2001:db8:1::1"),
			DhcpDdnsSenderPort:    storkutil.Ptr(int64(8080)),
			DhcpDdnsServerIP:      storkutil.Ptr("2001:db8:1::2"),
			DhcpDdnsServerPort:    storkutil.Ptr(int64(8081)),
		},
		KeaConfigExpiredLeasesProcessingParameters: models.KeaConfigExpiredLeasesProcessingParameters{
			ExpiredFlushReclaimedTimerWaitTime: storkutil.Ptr(int64(12)),
			ExpiredHoldReclaimedTime:           storkutil.Ptr(int64(13)),
			ExpiredMaxReclaimLeases:            storkutil.Ptr(int64(14)),
			ExpiredMaxReclaimTime:              storkutil.Ptr(int64(15)),
			ExpiredReclaimTimerWaitTime:        storkutil.Ptr(int64(16)),
			ExpiredUnwarnedReclaimCycles:       storkutil.Ptr(int64(17)),
		},
		KeaConfigReservationParameters: models.KeaConfigReservationParameters{
			ReservationsGlobal:    storkutil.Ptr(true),
			ReservationsInSubnet:  storkutil.Ptr(false),
			ReservationsOutOfPool: storkutil.Ptr(true),
		},
		KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
			ValidLifetime: storkutil.Ptr(int64(1111)),
		},
		DHCPOptions: models.DHCPOptions{
			Options: []*models.DHCPOption{
				{
					Code:       42,
					AlwaysSend: true,
					Fields: []*models.DHCPOptionField{
						{
							FieldType: "int32",
							Values:    []string{"4242"},
						},
					},
					Universe: 6,
				},
			},
		},
	}
	params2 := dhcp.UpdateKeaGlobalParametersSubmitParams{
		ID: transactionID,
		Request: &models.UpdateKeaDaemonsGlobalParametersSubmitRequest{
			Configs: []*models.KeaDaemonConfigurableGlobalParameters{
				{
					DaemonID:      daemons[0].GetID(),
					DaemonName:    dbmodel.DaemonNameDHCPv6,
					PartialConfig: partialConfig,
				},
				{
					DaemonID:      daemons[1].GetID(),
					DaemonName:    dbmodel.DaemonNameDHCPv6,
					PartialConfig: partialConfig,
				},
			},
		},
	}

	rsp2 := rapi.UpdateKeaGlobalParametersSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersSubmitOK{}, rsp2)

	// It should result in sending commands to two Kea servers. Each server
	// receives the config-set and config-write commands.
	require.Len(t, fa.RecordedCommands, 4)

	for i, c := range fa.RecordedCommands {
		switch {
		case i < 2:
			require.JSONEq(t,
				`{
					"command": "config-set",
					"service": [ "dhcp6" ],
					"arguments": {
						"Dhcp6": {
							"allocator": "flq",
							"early-global-reservations-lookup": false,
							"host-reservation-identifiers": [ "hw-address", "client-id" ],
							"cache-threshold": 0.2,
							"ddns-generated-prefix": "myhost.example.org",
							"ddns-override-client-update": true,
							"ddns-override-no-update": false,
							"ddns-qualifying-suffix": "example.org",
							"ddns-replace-client-name": "never",
							"ddns-send-updates": false,
							"ddns-update-on-renew": true,
							"ddns-use-conflict-resolution": true,
							"ddns-conflict-resolution-mode": "check-with-dhcid",
							"dhcp-ddns": {
								"enable-updates": true,
								"max-queue-size": 100,
								"ncr-format": "JSON",
								"ncr-protocol": "UDP",
								"sender-ip": "2001:db8:1::1",
								"sender-port": 8080,
								"server-ip": "2001:db8:1::2",
								"server-port": 8081
							},
							"expired-leases-processing": {
								"flush-reclaimed-timer-wait-time": 12,
								"hold-reclaimed-time": 13,
								"max-reclaim-leases": 14,
								"max-reclaim-time": 15,
								"reclaim-timer-wait-time": 16,
								"unwarned-reclaim-cycles": 17
							},
							"option-data": [
								{
									"code": 42,
									"always-send": true,
									"csv-format": true,
									"data": "4242",
									"space": "dhcp6"
								}
							],
							"pd-allocator": "random",
							"reservations-global": true,
							"reservations-in-subnet": false,
							"reservations-out-of-pool": true,
							"valid-lifetime": 1111
						}
					}
				}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
					"command": "config-write",
					"service": [ "dhcp6" ]
			}`,
				c.Marshal())
		}
	}

	// Make sure that the transaction is done.
	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.

	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)

	// Make sure that the daemon configurations have been updated in the database.
	updatedDaemons, err := dbmodel.GetDaemonsByIDs(db, daemonIDs)
	require.NoError(t, err)
	require.Len(t, updatedDaemons, 2)
	for _, daemon := range updatedDaemons {
		require.NotNil(t, daemon.KeaDaemon)
		config := daemon.KeaDaemon.Config
		require.NotNil(t, config)

		require.NotNil(t, config.GetValidLifetimeParameters().ValidLifetime)
		require.EqualValues(t, 1111, *config.GetValidLifetimeParameters().ValidLifetime)
	}
}

// Test that an error is returned when it is attempted to begin new
// transaction for updating non-existing daemons' configurations.
func TestUpdateGlobalParametersBeginNoDaemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.UpdateKeaGlobalParametersBeginParams{
		Request: &models.UpdateKeaDaemonsGlobalParametersBeginRequest{
			DaemonIds: []int64{1},
		},
	}
	rsp := rapi.UpdateKeaGlobalParametersBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.UpdateKeaGlobalParametersBeginDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
}

// Test error cases for submitting global parameters update.
func TestUpdateGlobalParametersSubmitError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"valid-lifetime": 2222
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	daemonIDs := []int64{app.Daemons[0].GetID()}

	daemon, err := dbmodel.GetDaemonByID(db, app.Daemons[0].GetID())
	require.NoError(t, err)
	require.NotNil(t, daemon)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("config-set", cmdResponses)
	}, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction. It will be needed for the actual part of the
	// test that relies on the existence of the transaction.
	params := dhcp.UpdateKeaGlobalParametersBeginParams{
		Request: &models.UpdateKeaDaemonsGlobalParametersBeginRequest{
			DaemonIds: daemonIDs,
		},
	}
	rsp := rapi.UpdateKeaGlobalParametersBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateKeaGlobalParametersBeginOK)
	contents := okRsp.Payload

	// Capture transaction ID.
	transactionID := contents.ID
	require.NotZero(t, transactionID)

	// Submit transaction without the subnet information.
	t.Run("no configurations", func(t *testing.T) {
		params := dhcp.UpdateKeaGlobalParametersSubmitParams{
			ID: transactionID,
			Request: &models.UpdateKeaDaemonsGlobalParametersSubmitRequest{
				Configs: nil,
			},
		}
		rsp := rapi.UpdateKeaGlobalParametersSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateKeaGlobalParametersSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateKeaGlobalParametersSubmitDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
		require.Equal(t, "No configs for update have been specified", *defaultRsp.Payload.Message)
	})

	// Submit transaction with non-matching transaction ID.
	t.Run("wrong transaction id", func(t *testing.T) {
		params := dhcp.UpdateKeaGlobalParametersSubmitParams{
			ID: transactionID + 1,
			Request: &models.UpdateKeaDaemonsGlobalParametersSubmitRequest{
				Configs: []*models.KeaDaemonConfigurableGlobalParameters{
					{
						DaemonID:   daemon.GetID(),
						DaemonName: dbmodel.DaemonNameDHCPv4,
						PartialConfig: &models.KeaConfigurableGlobalParameters{
							KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
								ValidLifetime: storkutil.Ptr(int64(1111)),
							},
						},
					},
				},
			},
		}
		rsp := rapi.UpdateKeaGlobalParametersSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateKeaGlobalParametersSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateKeaGlobalParametersSubmitDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
		require.Equal(t, "Transaction expired for the Kea configs update", *defaultRsp.Payload.Message)
	})

	// Submit transaction with no configurations. It simulates a failure in
	// "apply" step which typically is caused by some internal server problem
	// rather than malformed request.
	t.Run("no daemons in request", func(t *testing.T) {
		params := dhcp.UpdateKeaGlobalParametersSubmitParams{
			ID: transactionID,
			Request: &models.UpdateKeaDaemonsGlobalParametersSubmitRequest{
				Configs: []*models.KeaDaemonConfigurableGlobalParameters{},
			},
		}
		rsp := rapi.UpdateKeaGlobalParametersSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateKeaGlobalParametersSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateKeaGlobalParametersSubmitDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
		require.Equal(t, "No configs for update have been specified", *defaultRsp.Payload.Message)
	})

	t.Run("invalid DHCP option", func(t *testing.T) {
		params := dhcp.UpdateKeaGlobalParametersSubmitParams{
			ID: transactionID,
			Request: &models.UpdateKeaDaemonsGlobalParametersSubmitRequest{
				Configs: []*models.KeaDaemonConfigurableGlobalParameters{
					{
						DaemonID:   daemon.GetID(),
						DaemonName: dbmodel.DaemonNameDHCPv6,
						PartialConfig: &models.KeaConfigurableGlobalParameters{
							DHCPOptions: models.DHCPOptions{
								Options: []*models.DHCPOption{
									{
										Code:     42,
										Universe: 4,
										Fields: []*models.DHCPOptionField{
											{
												FieldType: "uint32",
												// The field value is missing.
												Values: []string{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		rsp := rapi.UpdateKeaGlobalParametersSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateKeaGlobalParametersSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateKeaGlobalParametersSubmitDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
		require.Equal(t, "Problem with flattening DHCP options: no values in the option field",
			*defaultRsp.Payload.Message)
	})

	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.UpdateKeaGlobalParametersSubmitParams{
			ID: transactionID,
			Request: &models.UpdateKeaDaemonsGlobalParametersSubmitRequest{
				Configs: []*models.KeaDaemonConfigurableGlobalParameters{
					{
						DaemonID:      daemon.GetID(),
						DaemonName:    dbmodel.DaemonNameDHCPv4,
						PartialConfig: &models.KeaConfigurableGlobalParameters{},
					},
				},
			},
		}
		rsp := rapi.UpdateKeaGlobalParametersSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateKeaGlobalParametersSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateKeaGlobalParametersSubmitDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
		require.Equal(t, fmt.Sprintf("Problem with committing Kea config: config-set command to %s failed: error status (1) returned by Kea dhcp4 daemon with text: 'unable to communicate with the daemon'", app.GetName()),
			*defaultRsp.Payload.Message)
	})
}

// Test that the transaction to update global Kea parameters can be canceled,
// resulting in the removal of this transaction from the config manager and
// allowing another user to apply config updates.
func TestUpdateGlobalParametersBeginCancel(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp6": {
			"valid-lifetime": 2222
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app1, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app1, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app2, err := server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app2, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	daemonIDs := []int64{app1.Daemons[0].GetID(), app2.Daemons[0].GetID()}

	daemons, err := dbmodel.GetDaemonsByIDs(db, daemonIDs)
	require.NoError(t, err)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.UpdateKeaGlobalParametersBeginParams{
		Request: &models.UpdateKeaDaemonsGlobalParametersBeginRequest{
			DaemonIds: daemonIDs,
		},
	}
	rsp := rapi.UpdateKeaGlobalParametersBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateKeaGlobalParametersBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID and configurations.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Configs)
	require.Len(t, contents.Configs, 2)
	require.EqualValues(t, daemons[0].GetID(), contents.Configs[0].DaemonID)
	require.Equal(t, dbmodel.DaemonNameDHCPv6, daemons[0].GetName(), contents.Configs[0].DaemonName)
	require.EqualValues(t, daemons[1].GetID(), contents.Configs[1].DaemonID)
	require.Equal(t, dbmodel.DaemonNameDHCPv6, daemons[1].GetName(), contents.Configs[1].DaemonName)

	// Try to start another session by another user.
	ctx2, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user = &dbmodel.SystemUser{
		ID: 2345,
	}
	err = rapi.SessionManager.LoginHandler(ctx2, user)
	require.NoError(t, err)

	// It should fail because the first session locked the daemons for
	// update.
	rsp = rapi.UpdateKeaGlobalParametersBegin(ctx2, params)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.UpdateKeaGlobalParametersBeginDefault)
	require.Equal(t, http.StatusLocked, getStatusCode(*defaultRsp))

	// Cancel the transaction.
	params2 := dhcp.UpdateKeaGlobalParametersDeleteParams{
		ID: transactionID,
	}
	rsp2 := rapi.UpdateKeaGlobalParametersDelete(ctx, params2)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersDeleteOK{}, rsp2)

	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)

	// After we released the lock, another user should be able to apply
	// his changes.
	rsp = rapi.UpdateKeaGlobalParametersBegin(ctx2, params)
	require.IsType(t, &dhcp.UpdateKeaGlobalParametersBeginOK{}, rsp)
}
