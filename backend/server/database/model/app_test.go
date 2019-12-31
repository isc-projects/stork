package dbmodel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	//log "github.com/sirupsen/logrus"

	dbtest "isc.org/stork/server/database/test"
)

func TestAddApp(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// add app but without machine, error should be raised
	s := &App{
		ID:   0,
		Type: KeaAppType,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)

	// add app but without type, error should be raised
	s = &App{
		ID:        0,
		MachineID: m.ID,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)

	// add app, no error expected
	s = &App{
		ID:          0,
		MachineID:   m.ID,
		Type:        KeaAppType,
		CtrlAddress: "cool.example.org",
		CtrlPort:    1234,
		Active:      true,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.ID)

	// add app for the same machine and ctrl port - error should be raised
	s = &App{
		ID:        0,
		MachineID: m.ID,
		Type:      Bind9AppType,
		CtrlPort:  1234,
		Active:    true,
	}
	err = AddApp(db, s)
	require.Contains(t, err.Error(), "duplicate")

	// add app with empty control address - error should be raised
	s = &App{
		ID:          0,
		MachineID:   m.ID,
		Type:        Bind9AppType,
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
		ID: 123,
	}
	err := DeleteApp(db, s0)
	require.Contains(t, err.Error(), "no rows in result")

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// add app, no error expected
	s := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      KeaAppType,
		CtrlPort:  1234,
		Active:    true,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.ID)

	// delete added app
	err = DeleteApp(db, s)
	require.NoError(t, err)
}

func TestGetAppsByMachine(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// there should be no apps yet
	apps, err := GetAppsByMachine(db, m.ID)
	require.Len(t, apps, 0)
	require.NoError(t, err)

	// add app, no error expected
	s := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      KeaAppType,
		CtrlPort:  1234,
		Active:    true,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.ID)

	// get apps of given machine
	apps, err = GetAppsByMachine(db, m.ID)
	require.Len(t, apps, 1)
	require.NoError(t, err)
}

func TestGetAppByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing app
	app, err := GetAppByID(db, 321)
	require.NoError(t, err)
	require.Nil(t, app)

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// add app, no error expected
	s := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      KeaAppType,
		CtrlPort:  1234,
		Active:    true,
	}

	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.ID)

	// get app by id
	app, err = GetAppByID(db, s.ID)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, s.ID, app.ID)

	// The control address is a special case. If it is not specified
	// it should be localhost.
	require.Equal(t, "localhost", app.CtrlAddress)
}

func TestGetAppsByPage(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// add kea app, no error expected
	sKea := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      KeaAppType,
		CtrlPort:  1234,
		Active:    true,
	}
	err = AddApp(db, sKea)
	require.NoError(t, err)
	require.NotEqual(t, 0, sKea.ID)

	// add bind app, no error expected
	sBind := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      Bind9AppType,
		CtrlPort:  4321,
		Active:    true,
	}
	err = AddApp(db, sBind)
	require.NoError(t, err)
	require.NotEqual(t, 0, sBind.ID)

	// get all apps
	apps, total, err := GetAppsByPage(db, 0, 10, "", "")
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.Equal(t, int64(2), total)

	// get kea apps
	apps, total, err = GetAppsByPage(db, 0, 10, "", KeaAppType)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, KeaAppType, apps[0].Type)

	// get bind apps
	apps, total, err = GetAppsByPage(db, 0, 10, "", Bind9AppType)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, Bind9AppType, apps[0].Type)
}

// Test that two names of the active DHCP daemons are returned.
func TestGetActiveDHCPMultiple(t *testing.T) {
	a := &App{
		Type: KeaAppType,
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
		Type: KeaAppType,
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
		Type:    KeaAppType,
		Details: AppBind9{},
	}
	daemons := a.GetActiveDHCPDeamonNames()
	require.Empty(t, daemons)
}

func TestGetAllApps(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// add kea app, no error expected
	aKea := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      KeaAppType,
		CtrlPort:  1234,
		Active:    true,
	}
	err = AddApp(db, aKea)
	require.NoError(t, err)
	require.NotEqual(t, 0, aKea.ID)

	// add bind app, no error expected
	aBind := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      Bind9AppType,
		CtrlPort:  4321,
		Active:    true,
	}
	err = AddApp(db, aBind)
	require.NoError(t, err)
	require.NotEqual(t, 0, aBind.ID)

	// get all apps
	apps, err := GetAllApps(db)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.True(t, apps[0].Type == KeaAppType || apps[1].Type == KeaAppType)
	require.True(t, apps[0].Type == Bind9AppType || apps[1].Type == Bind9AppType)
}

func TestAfterScanKea(t *testing.T) {
	ctx := context.Background()

	// for now details are nil
	aKea := &App{
		ID:        0,
		MachineID: 0,
		Type:      KeaAppType,
		CtrlPort:  1234,
		Active:    true,
	}
	err := aKea.AfterScan(ctx)
	require.Nil(t, err)
	require.Nil(t, aKea.Details)

	// add some details
	aKea.Details = map[string]interface{}{
		"ExtendedVersion": "1.2.3",
		"Daemons": []map[string]interface{}{
			{
				"Pid":  123,
				"Name": "dhcp4",
			},
		},
	}
	err = aKea.AfterScan(ctx)
	require.Nil(t, err)
	require.NotNil(t, aKea.Details)
	require.Equal(t, "1.2.3", aKea.Details.(AppKea).ExtendedVersion)
	require.Equal(t, "dhcp4", aKea.Details.(AppKea).Daemons[0].Name)
}

func TestAfterScanBind(t *testing.T) {
	ctx := context.Background()

	// for now details are nil
	aBind := &App{
		ID:        0,
		MachineID: 0,
		Type:      Bind9AppType,
		CtrlPort:  1234,
		Active:    true,
	}
	err := aBind.AfterScan(ctx)
	require.Nil(t, err)
	require.Nil(t, aBind.Details)

	// add some details
	aBind.Details = map[string]interface{}{
		"Daemon": map[string]interface{}{
			"Pid":  123,
			"Name": "named",
		},
	}
	err = aBind.AfterScan(ctx)
	require.Nil(t, err)
	require.NotNil(t, aBind.Details)
	require.Equal(t, "named", aBind.Details.(AppBind9).Daemon.Name)
	require.Equal(t, int32(123), aBind.Details.(AppBind9).Daemon.Pid)
}
