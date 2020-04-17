package dbmodel

import (
	"context"
	"testing"

	require "github.com/stretchr/testify/require"
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
	require.NotZero(t, m.ID)

	// add app but without machine, error should be raised
	s := &App{
		ID:   0,
		Type: AppTypeKea,
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
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "cool.example.org", "", 1234)

	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons: []*Daemon{
			{
				Name:    "kea-dhcp4",
				Version: "1.7.5",
				Active:  true,
			},
			{
				Name:    "kea-ctrl-agent",
				Version: "1.7.5",
				Active:  true,
			},
		},
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

	// add app for the same machine and ctrl port - error should be raised
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234)
	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "duplicate")

	// add app with empty control address, no error expected.
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "abcd", 4321)
	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.Nil(t, err)

	// add app with two control points - error should be raised.
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "dns1.example.org", "", 5555)
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "dns2.example.org", "", 5656)
	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "duplicate")

	// add app with explicit access point, bad type - error should be raised.
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, "foobar", "dns1.example.org", "", 6666)
	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)
}

// Test that the app can be updated in the database.
func TestUpdateApp(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	accessPoints := []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "cool.example.org", "", 1234)

	dhcp4Config, err := NewKeaConfigFromJSON(`{
        "Dhcp4": {
            "valid-lifetime": 4000
        }
    }`)
	require.NoError(t, err)
	require.NotNil(t, dhcp4Config)

	caConfig, err := NewKeaConfigFromJSON(`{
        "Control-agent": {
            "http-host": "10.20.30.40"
        }
    }`)
	require.NoError(t, err)
	require.NotNil(t, caConfig)

	a := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons: []*Daemon{
			{
				Name:    "kea-dhcp4",
				Version: "1.7.5",
				Active:  true,
				KeaDaemon: &KeaDaemon{
					Config: dhcp4Config,
					KeaDHCPDaemon: &KeaDHCPDaemon{
						LPS15min: 1024,
					},
				},
			},
			{
				Name:    "kea-ctrl-agent",
				Version: "1.7.4",
				Active:  false,
				KeaDaemon: &KeaDaemon{
					Config: caConfig,
				},
			},
		},
	}
	err = AddApp(db, a)
	require.NoError(t, err)
	require.NotZero(t, a.ID)

	// Make sure that the app along with the dependent information has been
	// added to the database.
	returned, err := GetAppByID(db, a.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Len(t, returned.Daemons, 2)
	require.NotZero(t, returned.Daemons[0].ID)
	require.Equal(t, "kea-dhcp4", returned.Daemons[0].Name)
	require.Equal(t, "1.7.5", returned.Daemons[0].Version)
	require.True(t, returned.Daemons[0].Active)
	require.NotNil(t, returned.Daemons[0].KeaDaemon)
	require.NotZero(t, returned.Daemons[0].KeaDaemon.ID)

	// Make sure that the configuration specified in the JSON format was
	// read and parsed correctly.
	require.NotNil(t, returned.Daemons[0].KeaDaemon.Config)
	rootName, ok := returned.Daemons[0].KeaDaemon.Config.GetRootName()
	require.True(t, ok)
	require.Equal(t, "Dhcp4", rootName)

	require.NotNil(t, returned.Daemons[0].KeaDaemon.KeaDHCPDaemon)
	require.NotZero(t, returned.Daemons[0].KeaDaemon.KeaDHCPDaemon.ID)
	require.EqualValues(t, 1024, returned.Daemons[0].KeaDaemon.KeaDHCPDaemon.LPS15min)

	require.NotZero(t, returned.Daemons[1].ID)
	require.Equal(t, "kea-ctrl-agent", returned.Daemons[1].Name)
	require.Equal(t, "1.7.4", returned.Daemons[1].Version)
	require.False(t, returned.Daemons[1].Active)
	require.NotNil(t, returned.Daemons[1].KeaDaemon)
	require.NotZero(t, returned.Daemons[1].ID)
	require.NotNil(t, returned.Daemons[1].KeaDaemon.Config)
	require.Nil(t, returned.Daemons[1].KeaDaemon.KeaDHCPDaemon)

	// Modify the app information.
	a.Active = false

	// Modify the daemons to make sure they also get updated. Some daemons
	// are now gone, some are modified, some added.
	a.Daemons[0] = &Daemon{
		Name:    "kea-dhcp6",
		Version: "1.7.6",
		Active:  true,
		KeaDaemon: &KeaDaemon{
			KeaDHCPDaemon: &KeaDHCPDaemon{
				LPS15min: 2048,
			},
		},
	}

	a.Daemons[1].Version = "1.7.5"
	a.Daemons[1].Active = false

	err = UpdateApp(db, a)
	require.NoError(t, err)
	require.False(t, a.Active)

	// Validate the updated date.
	updated, err := GetAppByID(db, a.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.EqualValues(t, a.ID, updated.ID)
	require.False(t, updated.Active)
	// Check daemons.
	require.Len(t, updated.Daemons, 2)
	// Make sure there are two distinct daemons.
	require.NotEqual(t, updated.Daemons[0].Name, updated.Daemons[1].Name)
	// Make sure that the daemon names are as expected.
	for _, d := range updated.Daemons {
		require.Contains(t, []string{"kea-dhcp6", "kea-ctrl-agent"}, d.Name,
			"daemons haven't been updated together with the application")
	}
	// For each daemon name, check that the rest of values is fine.
	for _, d := range updated.Daemons {
		switch d.Name {
		case "kea-dhcp6":
			require.Equal(t, "1.7.6", d.Version)
			require.True(t, d.Active)
			require.NotNil(t, d.KeaDaemon)
			require.Nil(t, d.KeaDaemon.Config)
			require.NotNil(t, d.KeaDaemon.KeaDHCPDaemon)
			require.EqualValues(t, 2048, d.KeaDaemon.KeaDHCPDaemon.LPS15min)
		case "kea-ctrl-agent":
			// The ID of the daemon should be preserved to keep data integrity if
			// something is referencing the updated daemon.
			require.EqualValues(t, returned.Daemons[1].ID, d.ID)
			require.Equal(t, "1.7.5", d.Version)
			require.False(t, d.Active)
			require.NotNil(t, d.KeaDaemon)
			require.NotNil(t, d.KeaDaemon.Config)
			require.Nil(t, d.KeaDaemon.KeaDHCPDaemon)
		}
	}

	// change access point
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "warm.example.org", "abcd", 2345)
	a.AccessPoints = accessPoints
	err = UpdateApp(db, a)
	require.NoError(t, err)
	require.Len(t, a.AccessPoints, 1)
	pt := a.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "warm.example.org", pt.Address)
	require.EqualValues(t, 2345, pt.Port)
	require.Equal(t, "abcd", pt.Key)

	updated, err = GetAppByID(db, a.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.EqualValues(t, a.ID, updated.ID)
	require.Len(t, updated.AccessPoints, 1)
	pt = updated.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "warm.example.org", pt.Address)
	require.EqualValues(t, 2345, pt.Port)
	require.Equal(t, "abcd", pt.Key)

	// add access point
	accessPoints = AppendAccessPoint(accessPoints, AccessPointStatistics, "cold.example.org", "", 1234)
	a.AccessPoints = accessPoints
	err = UpdateApp(db, a)
	require.NoError(t, err)
	require.Len(t, a.AccessPoints, 2)
	pt = a.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "warm.example.org", pt.Address)
	require.EqualValues(t, 2345, pt.Port)
	require.Equal(t, "abcd", pt.Key)
	pt = a.AccessPoints[1]
	require.Equal(t, AccessPointStatistics, pt.Type)
	require.Equal(t, "cold.example.org", pt.Address)
	require.EqualValues(t, 1234, pt.Port)
	require.Empty(t, pt.Key)

	updated, err = GetAppByID(db, a.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.EqualValues(t, a.ID, updated.ID)
	require.Len(t, updated.AccessPoints, 2)
	pt = updated.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "warm.example.org", pt.Address)
	require.EqualValues(t, 2345, pt.Port)
	require.Equal(t, "abcd", pt.Key)
	pt = updated.AccessPoints[1]
	require.Equal(t, AccessPointStatistics, pt.Type)
	require.Equal(t, "cold.example.org", pt.Address)
	require.EqualValues(t, 1234, pt.Port)
	require.Empty(t, pt.Key)

	// delete access point
	accessPoints = accessPoints[0:1]
	a.AccessPoints = accessPoints
	err = UpdateApp(db, a)
	require.NoError(t, err)
	require.Len(t, a.AccessPoints, 1)
	pt = a.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "warm.example.org", pt.Address)
	require.EqualValues(t, 2345, pt.Port)
	require.Equal(t, "abcd", pt.Key)

	updated, err = GetAppByID(db, a.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.EqualValues(t, a.ID, updated.ID)
	require.Len(t, updated.AccessPoints, 1)
	pt = updated.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "warm.example.org", pt.Address)
	require.EqualValues(t, 2345, pt.Port)
	require.Equal(t, "abcd", pt.Key)
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
	require.NotZero(t, m.ID)

	// add app, no error expected
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "10.0.0.1", "", 4321)

	s := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

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
	require.NotZero(t, m.ID)

	// there should be no apps yet
	apps, err := GetAppsByMachine(db, m.ID)
	require.Len(t, apps, 0)
	require.NoError(t, err)

	// add app, no error expected
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234)

	s := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons: []*Daemon{
			{
				Name:        "named",
				Bind9Daemon: &Bind9Daemon{},
			},
		},
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

	// get apps of given machine
	apps, err = GetAppsByMachine(db, m.ID)
	require.Len(t, apps, 1)
	require.NoError(t, err)
	app := apps[0]
	require.Equal(t, m.ID, app.MachineID)
	require.Equal(t, AppTypeBind9, app.Type)
	// check access point
	require.Len(t, app.AccessPoints, 1)
	pt := app.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.EqualValues(t, 1234, pt.Port)
	require.Empty(t, pt.Key)

	// Make sure that the daemon is returned.
	require.Len(t, apps[0].Daemons, 1)
	require.Equal(t, "named", apps[0].Daemons[0].Name)
	require.NotNil(t, apps[0].Daemons[0].Bind9Daemon)

	// test GetAccessPoint
	pt, err = app.GetAccessPoint(AccessPointControl)
	require.NotNil(t, pt)
	require.NoError(t, err)
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.EqualValues(t, 1234, pt.Port)
	require.Empty(t, pt.Key)
	// bad access point type
	pt, err = app.GetAccessPoint("foobar")
	require.Nil(t, pt)
	require.Error(t, err)
}

// Check getting apps by type only.
func TestGetAppsByType(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add a machine
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// add kea app
	var keaPoints []*AccessPoint
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234)
	aKea := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*Daemon{
			{
				Name: "kea-dhcp4",
				KeaDaemon: &KeaDaemon{
					KeaDHCPDaemon: &KeaDHCPDaemon{},
				},
			},
		},
	}
	err = AddApp(db, aKea)
	require.NoError(t, err)
	require.NotZero(t, aKea.ID)

	// add bind9 app
	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "", 2234)
	aBind9 := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: bind9Points,
		Daemons: []*Daemon{
			{
				Name:        "named",
				Bind9Daemon: &Bind9Daemon{},
			},
		},
	}
	err = AddApp(db, aBind9)
	require.NoError(t, err)
	require.NotZero(t, aBind9.ID)

	// check getting kea apps
	apps, err := GetAppsByType(db, AppTypeKea)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, aKea.ID, apps[0].ID)
	require.NotNil(t, apps[0].Machine)
	require.Len(t, apps[0].Daemons, 1)
	require.NotNil(t, apps[0].Daemons[0].KeaDaemon)
	require.NotNil(t, apps[0].Daemons[0].KeaDaemon.KeaDHCPDaemon)

	// check getting bind9 apps
	apps, err = GetAppsByType(db, AppTypeBind9)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, aBind9.ID, apps[0].ID)
	require.NotNil(t, apps[0].Machine)
	require.Len(t, apps[0].Daemons, 1)
	require.NotNil(t, apps[0].Daemons[0].Bind9Daemon)
}

// Check getting app by its ID.
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
	require.NotZero(t, m.ID)

	// add app, no error expected
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 4444)
	accessPoints = AppendAccessPoint(accessPoints, AccessPointStatistics, "10.0.0.2", "abcd", 5555)

	s := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}

	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

	// Get app by ID.
	app, err = GetAppByID(db, s.ID)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, s.ID, app.ID)
	// Machine is set.
	require.Equal(t, m.ID, app.Machine.ID)
	require.Equal(t, "localhost", app.Machine.Address)
	require.EqualValues(t, 8080, app.Machine.AgentPort)
	// Check access points.
	require.Len(t, app.AccessPoints, 2)

	pt := app.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	// The control address is a special case.
	// If it is not specified it should be localhost.
	require.Equal(t, "localhost", pt.Address)
	require.EqualValues(t, 4444, pt.Port)
	require.Empty(t, pt.Key)

	pt = app.AccessPoints[1]
	require.Equal(t, AccessPointStatistics, pt.Type)
	require.Equal(t, "10.0.0.2", pt.Address)
	require.EqualValues(t, 5555, pt.Port)
	require.Equal(t, "abcd", pt.Key)

	// test GetAccessPoint
	pt, err = app.GetAccessPoint(AccessPointControl)
	require.NotNil(t, pt)
	require.NoError(t, err)
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.EqualValues(t, 4444, pt.Port)
	require.Empty(t, pt.Key)

	pt, err = app.GetAccessPoint(AccessPointStatistics)
	require.NotNil(t, pt)
	require.NoError(t, err)
	require.Equal(t, AccessPointStatistics, pt.Type)
	require.Equal(t, "10.0.0.2", pt.Address)
	require.EqualValues(t, 5555, pt.Port)
	require.Equal(t, "abcd", pt.Key)
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
	require.NotZero(t, m.ID)

	// add kea app, no error expected
	var keaPoints []*AccessPoint
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234)

	sKea := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*Daemon{
			{
				KeaDaemon: &KeaDaemon{
					KeaDHCPDaemon: &KeaDHCPDaemon{},
				},
			},
		},
	}
	err = AddApp(db, sKea)
	require.NoError(t, err)
	require.NotZero(t, sKea.ID)

	// add bind app, no error expected
	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "abcd", 4321)

	sBind := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: bind9Points,
		Daemons: []*Daemon{
			{
				Bind9Daemon: &Bind9Daemon{},
			},
		},
	}
	err = AddApp(db, sBind)
	require.NoError(t, err)
	require.NotZero(t, sBind.ID)

	// get all apps
	apps, total, err := GetAppsByPage(db, 0, 10, "", "")
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, 2, total)

	// get kea apps
	apps, total, err = GetAppsByPage(db, 0, 10, "", AppTypeKea)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, 1, total)
	require.Equal(t, AppTypeKea, apps[0].Type)
	require.Len(t, apps[0].Daemons, 1)
	require.NotNil(t, apps[0].Daemons[0].KeaDaemon)
	require.NotNil(t, apps[0].Daemons[0].KeaDaemon.KeaDHCPDaemon)
	require.Len(t, apps[0].AccessPoints, 1)
	pt := apps[0].AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.EqualValues(t, 1234, pt.Port)
	require.Empty(t, pt.Key)

	// get bind apps
	apps, total, err = GetAppsByPage(db, 0, 10, "", AppTypeBind9)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, 1, total)
	require.Equal(t, AppTypeBind9, apps[0].Type)
	require.Len(t, apps[0].Daemons, 1)
	require.NotNil(t, apps[0].Daemons[0].Bind9Daemon)
	require.Len(t, apps[0].AccessPoints, 1)
	pt = apps[0].AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.EqualValues(t, 4321, pt.Port)
	require.Equal(t, "abcd", pt.Key)
}

// Test that two names of the active DHCP daemons are returned.
func TestGetActiveDHCPMultiple(t *testing.T) {
	a := &App{
		Type: AppTypeKea,
		Daemons: []*Daemon{
			{
				Active:    true,
				Name:      "dhcp4",
				KeaDaemon: &KeaDaemon{},
			},
			{
				Active:    true,
				Name:      "dhcp6",
				KeaDaemon: &KeaDaemon{},
			},
		},
	}

	daemons := a.GetActiveDHCPDaemonNames()
	require.Len(t, daemons, 2)
	require.Contains(t, daemons, "dhcp4")
	require.Contains(t, daemons, "dhcp6")
}

// Test that a single name of the active DHCP deamon is returned.
func TestGetActiveDHCPSingle(t *testing.T) {
	a := &App{
		Type: AppTypeKea,
		Daemons: []*Daemon{
			{
				Active:    false,
				Name:      "dhcp4",
				KeaDaemon: &KeaDaemon{},
			},
			{
				Active:    true,
				Name:      "dhcp6",
				KeaDaemon: &KeaDaemon{},
			},
		},
	}
	daemons := a.GetActiveDHCPDaemonNames()
	require.Len(t, daemons, 1)
	require.NotContains(t, daemons, "dhcp4")
	require.Contains(t, daemons, "dhcp6")
}

// Test that empty list of deamons is returned if the daemon type
// is not Kea.
func TestGetActiveDHCPAppMismatch(t *testing.T) {
	a := &App{
		Type: AppTypeKea,
		Daemons: []*Daemon{
			{
				Bind9Daemon: &Bind9Daemon{},
			},
		},
	}
	daemons := a.GetActiveDHCPDaemonNames()
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
	require.NotZero(t, m.ID)

	// add kea app, no error expected
	var keaPoints []*AccessPoint
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234)

	aKea := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*Daemon{
			{
				KeaDaemon: &KeaDaemon{
					KeaDHCPDaemon: &KeaDHCPDaemon{},
				},
			},
		},
	}
	err = AddApp(db, aKea)
	require.NoError(t, err)
	require.NotZero(t, aKea.ID)

	// add bind app, no error expected
	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "abcd", 4321)

	aBind := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: bind9Points,
		Daemons: []*Daemon{
			{
				Bind9Daemon: &Bind9Daemon{},
			},
		},
	}
	err = AddApp(db, aBind)
	require.NoError(t, err)
	require.NotZero(t, aBind.ID)

	// get all apps
	apps, err := GetAllApps(db)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.True(t, apps[0].Type == AppTypeKea || apps[1].Type == AppTypeKea)
	require.True(t, apps[0].Type == AppTypeBind9 || apps[1].Type == AppTypeBind9)

	// Make sure that app specific fields are set.
	for _, a := range apps {
		require.Len(t, a.Daemons, 1)
		switch a.Type {
		case AppTypeKea:
			require.NotNil(t, a.Daemons[0].KeaDaemon)
			require.NotNil(t, a.Daemons[0].KeaDaemon.KeaDHCPDaemon)
		case AppTypeBind9:
			require.NotNil(t, a.Daemons[0].Bind9Daemon)
		}
	}
}

// Test that local subnet id of the Kea subnet can be extracted.
func TestGetLocalSubnetID(t *testing.T) {
	ctx := context.Background()

	accessPoints := []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234)
	aKea := &App{
		ID:           0,
		MachineID:    0,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons: []*Daemon{
			{
				KeaDaemon: &KeaDaemon{
					Config: NewKeaConfig(&map[string]interface{}{
						"Dhcp4": map[string]interface{}{
							"subnet4": []map[string]interface{}{
								{
									"id":     1,
									"subnet": "192.0.2.0/24",
								},
							},
						},
					}),
				},
			},
		},
	}

	err := aKea.Daemons[0].KeaDaemon.AfterScan(ctx)
	require.Nil(t, err)
	require.NotNil(t, aKea.Daemons[0].KeaDaemon.Config)

	// Try to find a non-existing subnet.
	require.Zero(t, aKea.GetLocalSubnetID("192.0.3.0/24"))
	// Next, try to find the existing subnet.
	require.EqualValues(t, 1, aKea.GetLocalSubnetID("192.0.2.0/24"))
}
