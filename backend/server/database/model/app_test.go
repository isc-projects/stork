package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database/test"
)

func TestAddApp(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		Id:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.Id)

	// add app but without machine, error should be raised
	s := &App{
		Id:   0,
		Type: "kea",
	}
	err = AddApp(db, s)
	require.NotNil(t, err)

	// add app but without type, error should be raised
	s = &App{
		Id:        0,
		MachineID: m.Id,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)

	// add app, no error expected
	s = &App{
		Id:          0,
		MachineID:   m.Id,
		Type:        "kea",
		CtrlAddress: "cool.example.org",
		CtrlPort:    1234,
		Active:      true,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.Id)

	// add app for the same machine and ctrl port - error should be raised
	s = &App{
		Id:        0,
		MachineID: m.Id,
		Type:      "bind9",
		CtrlPort:  1234,
		Active:    true,
	}
	err = AddApp(db, s)
	require.Contains(t, err.Error(), "duplicate")

	// add app with empty control address - error should be raised
	s = &App{
		Id:          0,
		MachineID:   m.Id,
		Type:        "bind",
		CtrlAddress: "",
		CtrlPort:    1234,
		Active:      true,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)
}

func TestDeleteApp(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// delete non-existing app
	s0 := &App{
		Id: 123,
	}
	err := DeleteApp(db, s0)
	require.Contains(t, err.Error(), "no rows in result")

	// add first machine, should be no error
	m := &Machine{
		Id:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.Id)

	// add app, no error expected
	s := &App{
		Id:        0,
		MachineID: m.Id,
		Type:      "kea",
		CtrlPort:  1234,
		Active:    true,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.Id)

	// delete added app
	err = DeleteApp(db, s)
	require.NoError(t, err)
}

func TestGetAppsByMachine(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		Id:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.Id)

	// there should be no apps yet
	apps, err := GetAppsByMachine(db, m.Id)
	require.Len(t, apps, 0)
	require.NoError(t, err)

	// add app, no error expected
	s := &App{
		Id:        0,
		MachineID: m.Id,
		Type:      "kea",
		CtrlPort:  1234,
		Active:    true,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.Id)

	// get apps of given machine
	apps, err = GetAppsByMachine(db, m.Id)
	require.Len(t, apps, 1)
	require.NoError(t, err)
}

func TestGetAppById(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing app
	app, err := GetAppById(db, 321)
	require.NoError(t, err)
	require.Nil(t, app)

	// add first machine, should be no error
	m := &Machine{
		Id:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.Id)

	// add app, no error expected
	s := &App{
		Id:        0,
		MachineID: m.Id,
		Type:      "kea",
		CtrlPort:  1234,
		Active:    true,
	}

	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.Id)

	// get app by id
	app, err = GetAppById(db, s.Id)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, s.Id, app.Id)

	// The control address is a special case. If it is not specified
	// it should be localhost.
	require.Equal(t, "localhost", app.CtrlAddress)
}

func TestGetAppsByPage(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		Id:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.Id)

	// add kea app, no error expected
	sKea := &App{
		Id:        0,
		MachineID: m.Id,
		Type:      "kea",
		CtrlPort:  1234,
		Active:    true,
	}
	err = AddApp(db, sKea)
	require.NoError(t, err)
	require.NotEqual(t, 0, sKea.Id)

	// add bind app, no error expected
	sBind := &App{
		Id:        0,
		MachineID: m.Id,
		Type:      "bind9",
		CtrlPort:  4321,
		Active:    true,
	}
	err = AddApp(db, sBind)
	require.NoError(t, err)
	require.NotEqual(t, 0, sBind.Id)

	// get all apps
	apps, total, err := GetAppsByPage(db, 0, 10, "", "")
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.Equal(t, int64(2), total)

	// get kea apps
	apps, total, err = GetAppsByPage(db, 0, 10, "", "kea")
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, "kea", apps[0].Type)

	// get bind apps
	apps, total, err = GetAppsByPage(db, 0, 10, "", "bind9")
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, "bind9", apps[0].Type)
}

// Test that two names of the active DHCP daemons are returned.
func TestGetActiveDHCPMultiple(t *testing.T) {
	a := &App{
		Type: "kea",
		Details: AppKea{
			Daemons: []KeaDaemon{
				{
					Active: true,
					Name:   "dhcp4",
				},
				{
					Active: true,
					Name:   "dhcp6",
				},
			},
		},
	}

	daemons := a.GetActiveDHCPDeamonNames()
	require.Equal(t, 2, len(daemons))
	require.Contains(t, daemons, "dhcp4")
	require.Contains(t, daemons, "dhcp6")
}

// Test that a single name of the active DHCP deamon is returned.
func TestGetActiveDHCPSingle(t *testing.T) {
	a := &App{
		Type: "kea",
		Details: AppKea{
			Daemons: []KeaDaemon{
				{
					Active: false,
					Name:   "dhcp4",
				},
				{
					Active: true,
					Name:   "dhcp6",
				},
			},
		},
	}
	daemons := a.GetActiveDHCPDeamonNames()
	require.Equal(t, 1, len(daemons))
	require.NotContains(t, daemons, "dhcp4")
	require.Contains(t, daemons, "dhcp6")
}

// Test that empty list of deamons is returned if the application type
// is not Kea.
func TestGetActiveDHCPAppMismatch(t *testing.T) {
	a := &App{
		Type:    "kea",
		Details: AppBind9{},
	}
	daemons := a.GetActiveDHCPDeamonNames()
	require.Empty(t, daemons)
}
