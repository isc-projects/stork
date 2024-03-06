package dbmodel

import (
	"fmt"
	"testing"
	"time"

	require "github.com/stretchr/testify/require"
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
	a1 := &App{
		ID:   0,
		Type: AppTypeKea,
	}
	addedDaemons, err := AddApp(db, a1)
	require.NotNil(t, err)
	require.Len(t, addedDaemons, 0)

	// add app but without type, error should be raised
	a2 := &App{
		ID:        0,
		MachineID: m.ID,
	}
	addedDaemons, err = AddApp(db, a2)
	require.NotNil(t, err)
	require.Len(t, addedDaemons, 0)

	// add app, no error expected
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "cool.example.org", "", 1234, false)

	a3 := &App{
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
	addedDaemons, err = AddApp(db, a3)
	require.NoError(t, err)
	require.NotZero(t, a3.ID)
	require.Len(t, addedDaemons, 2)
	require.Len(t, a3.AccessPoints, 1)
	require.False(t, a3.AccessPoints[0].UseSecureProtocol)

	// add the same app but with no daemon this time
	a3.Daemons = []*Daemon{
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
	}
	addedDaemons, err = AddApp(db, a3)
	require.NotNil(t, err)
	require.Len(t, addedDaemons, 0)

	// add app for the same machine and ctrl port - error should be raised
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234, true)
	a4 := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	addedDaemons, err = AddApp(db, a4)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "duplicate")
	require.Len(t, addedDaemons, 0)
	require.Len(t, a4.AccessPoints, 1)
	require.True(t, a4.AccessPoints[0].UseSecureProtocol)

	// add app with empty control address, no error expected.
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "abcd", 4321, false)
	a5 := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	addedDaemons, err = AddApp(db, a5)
	require.Nil(t, err)
	require.Len(t, addedDaemons, 0)

	// add app with two control points - error should be raised.
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "dns1.example.org", "", 5555, true)
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "dns2.example.org", "", 5656, false)
	a6 := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	addedDaemons, err = AddApp(db, a6)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "duplicate")
	require.Len(t, addedDaemons, 0)
	require.Len(t, a6.AccessPoints, 2)
	require.True(t, a6.AccessPoints[0].UseSecureProtocol)
	require.False(t, a6.AccessPoints[1].UseSecureProtocol)

	// add app with explicit access point, bad type - error should be raised.
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, "foobar", "dns1.example.org", "", 6666, true)
	a7 := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	addedDaemons, err = AddApp(db, a7)
	require.NotNil(t, err)
	require.Len(t, addedDaemons, 0)
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
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "cool.example.org", "", 1234, false)

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
						Stats: KeaDHCPDaemonStats{
							RPS1: 1024,
							RPS2: 2048,
						},
					},
				},
				LogTargets: []*LogTarget{
					{
						Name:     "frog",
						Severity: "ERROR",
						Output:   "stdout",
					},
					{
						Name:     "lion",
						Severity: "FATAL",
						Output:   "/tmp/filename.log",
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
				LogTargets: []*LogTarget{
					{
						Name:     "foo",
						Severity: "INFO",
						Output:   "stdout",
					},
					{
						Name:     "bar",
						Severity: "DEBUG",
						Output:   "/tmp/filename.log",
					},
				},
			},
		},
	}
	_, err = AddApp(db, a)
	require.NoError(t, err)
	require.NotZero(t, a.ID)

	// Remember the creation time so it can be compared after the update.
	createdAt := a.CreatedAt
	require.NotZero(t, createdAt)

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
	require.NotNil(t, returned.Daemons[0].KeaDaemon.Config.DHCPv4Config)

	require.NotNil(t, returned.Daemons[0].KeaDaemon.KeaDHCPDaemon)
	require.NotZero(t, returned.Daemons[0].KeaDaemon.KeaDHCPDaemon.ID)
	require.EqualValues(t, 1024, returned.Daemons[0].KeaDaemon.KeaDHCPDaemon.Stats.RPS1)
	require.EqualValues(t, 2048, returned.Daemons[0].KeaDaemon.KeaDHCPDaemon.Stats.RPS2)

	// Make sure that the logging targets were stored.
	require.Len(t, returned.Daemons[0].LogTargets, 2)
	require.NotEqual(t, returned.Daemons[0].LogTargets[0].Name, returned.Daemons[0].LogTargets[1].Name)
	for _, target := range returned.Daemons[0].LogTargets {
		require.NotZero(t, target.ID)
		require.NotZero(t, target.CreatedAt)
		require.True(t, target.Name == "frog" || target.Name == "lion")
		if target.Name == "frog" {
			require.Equal(t, "error", target.Severity)
			require.Equal(t, "stdout", target.Output)
		} else {
			require.Equal(t, "fatal", target.Severity)
			require.Equal(t, "/tmp/filename.log", target.Output)
		}
	}

	// Save the log target creation time so we can later ensure it hasn't been modified.
	logTargetCreatedAt := returned.Daemons[1].CreatedAt
	require.NotZero(t, logTargetCreatedAt)

	require.NotZero(t, returned.Daemons[1].ID)
	require.Equal(t, "kea-ctrl-agent", returned.Daemons[1].Name)
	require.Equal(t, "1.7.4", returned.Daemons[1].Version)
	require.False(t, returned.Daemons[1].Active)
	require.NotNil(t, returned.Daemons[1].KeaDaemon)
	require.NotZero(t, returned.Daemons[1].ID)
	require.NotNil(t, returned.Daemons[1].KeaDaemon.Config)
	require.Nil(t, returned.Daemons[1].KeaDaemon.KeaDHCPDaemon)

	require.Len(t, returned.Daemons[1].LogTargets, 2)
	require.NotEqual(t, returned.Daemons[1].LogTargets[0].Name, returned.Daemons[1].LogTargets[1].Name)
	for _, target := range returned.Daemons[1].LogTargets {
		require.NotZero(t, target.ID)
		require.NotZero(t, target.CreatedAt)
		require.True(t, target.Name == "foo" || target.Name == "bar")
		if target.Name == "foo" {
			require.Equal(t, "info", target.Severity)
			require.Equal(t, "stdout", target.Output)
		} else {
			require.Equal(t, "debug", target.Severity)
			require.Equal(t, "/tmp/filename.log", target.Output)
		}
	}

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
				Stats: KeaDHCPDaemonStats{
					RPS1: 2048,
					RPS2: 4096,
				},
			},
		},
	}

	a.Daemons[1].Version = "1.7.5"
	a.Daemons[1].Active = false
	a.Daemons[1].LogTargets = a.Daemons[1].LogTargets[:1]

	// Reset creation time, to ensure that the creation time is not modified
	// during update.
	a.CreatedAt = time.Time{}
	a.Daemons[1].LogTargets[0].CreatedAt = time.Time{}

	addedDaemons, deletedDaemons, err := UpdateApp(db, a)
	require.NoError(t, err)
	require.False(t, a.Active)
	require.Len(t, addedDaemons, 1)
	require.Len(t, deletedDaemons, 1)

	// Validate the updated date.
	updated, err := GetAppByID(db, a.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.EqualValues(t, a.ID, updated.ID)
	require.False(t, updated.Active)
	require.Equal(t, createdAt, updated.CreatedAt)

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
			require.EqualValues(t, 2048, d.KeaDaemon.KeaDHCPDaemon.Stats.RPS1)
			require.EqualValues(t, 4096, d.KeaDaemon.KeaDHCPDaemon.Stats.RPS2)
			require.Empty(t, d.LogTargets)
		case "kea-ctrl-agent":
			// The ID of the daemon should be preserved to keep data integrity if
			// something is referencing the updated daemon.
			require.EqualValues(t, returned.Daemons[1].ID, d.ID)
			require.Equal(t, "1.7.5", d.Version)
			require.False(t, d.Active)
			require.NotNil(t, d.KeaDaemon)
			require.NotNil(t, d.KeaDaemon.Config)
			require.Nil(t, d.KeaDaemon.KeaDHCPDaemon)
			require.Len(t, d.LogTargets, 1)
			require.Equal(t, logTargetCreatedAt, d.LogTargets[0].CreatedAt)
		}
	}

	// change access point
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "warm.example.org", "abcd", 2345, true)
	a.AccessPoints = accessPoints
	addedDaemons, deletedDaemons, err = UpdateApp(db, a)
	require.NoError(t, err)
	require.Len(t, a.AccessPoints, 1)
	pt := a.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "warm.example.org", pt.Address)
	require.EqualValues(t, 2345, pt.Port)
	require.Equal(t, "abcd", pt.Key)
	require.True(t, pt.UseSecureProtocol)
	require.Len(t, addedDaemons, 0)
	require.Len(t, deletedDaemons, 0)

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
	require.True(t, pt.UseSecureProtocol)

	// add access point
	accessPoints = AppendAccessPoint(accessPoints, AccessPointStatistics, "cold.example.org", "", 1234, false)
	a.AccessPoints = accessPoints
	addedDaemons, deletedDaemons, err = UpdateApp(db, a)
	require.NoError(t, err)
	require.Len(t, a.AccessPoints, 2)
	pt = a.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "warm.example.org", pt.Address)
	require.EqualValues(t, 2345, pt.Port)
	require.Equal(t, "abcd", pt.Key)
	require.True(t, pt.UseSecureProtocol)
	pt = a.AccessPoints[1]
	require.Equal(t, AccessPointStatistics, pt.Type)
	require.Equal(t, "cold.example.org", pt.Address)
	require.EqualValues(t, 1234, pt.Port)
	require.Empty(t, pt.Key)
	require.False(t, pt.UseSecureProtocol)
	require.Len(t, addedDaemons, 0)
	require.Len(t, deletedDaemons, 0)

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
	require.True(t, pt.UseSecureProtocol)
	pt = updated.AccessPoints[1]
	require.Equal(t, AccessPointStatistics, pt.Type)
	require.Equal(t, "cold.example.org", pt.Address)
	require.EqualValues(t, 1234, pt.Port)
	require.Empty(t, pt.Key)
	require.False(t, pt.UseSecureProtocol)

	// delete access point
	accessPoints = accessPoints[0:1]
	a.AccessPoints = accessPoints
	addedDaemons, deletedDaemons, err = UpdateApp(db, a)
	require.NoError(t, err)
	require.Len(t, a.AccessPoints, 1)
	pt = a.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "warm.example.org", pt.Address)
	require.EqualValues(t, 2345, pt.Port)
	require.Equal(t, "abcd", pt.Key)
	require.Len(t, addedDaemons, 0)
	require.Len(t, deletedDaemons, 0)

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

// Test that an app can be renamed without affecting other app
// specific information.
func TestRenameApp(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add first app. We won't be renaming it but we want to
	// have more than one machine and more than one machine
	// in the database because it is a real life case.
	machine := &Machine{
		ID:        0,
		Address:   "dns.example.org",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	app := &App{
		Type:      AppTypeBind9,
		Name:      "dns-server1",
		MachineID: machine.ID,
	}
	_, err = AddApp(db, app)
	require.NoError(t, err, "found error %+v", err)

	machine = &Machine{
		ID:        0,
		Address:   "dhcp.example.org",
		AgentPort: 8080,
	}
	err = AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	app = &App{
		Type:      AppTypeKea,
		Name:      "dhcp-server1",
		MachineID: machine.ID,
	}
	_, err = AddApp(db, app)
	require.NoError(t, err, "found error %+v", err)

	oldApp, err := RenameApp(db, app.ID, "dhcp-server2")
	require.NoError(t, err)
	require.NotNil(t, oldApp)
	require.Equal(t, "dhcp-server1", oldApp.Name)
	require.Equal(t, AppTypeKea, oldApp.Type)
	require.Equal(t, machine.ID, oldApp.MachineID)

	// Make sure the app has been renamed in the database.
	appReturned, err := GetAppByID(db, app.ID)
	require.NoError(t, err)
	require.NotNil(t, appReturned)
	require.Equal(t, "dhcp-server2", appReturned.Name)
	require.Equal(t, AppTypeKea, appReturned.Type)

	// Trying to set invalid name should cause an error.
	oldApp, err = RenameApp(db, app.ID, "dhcp-server2@machine3")
	require.Error(t, err)
	require.Nil(t, oldApp)
}

func TestDeleteApp(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// delete non-existing app
	s0 := &App{
		ID: 123,
	}
	err := DeleteApp(db, s0)
	require.Contains(t, err.Error(), "database entry not found")

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
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "10.0.0.1", "", 4321, false)

	s := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
	}
	_, err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

	// delete added app
	err = DeleteApp(db, s)
	require.NoError(t, err)
}

// This test verifies that apps' names are set to the default values and that
// they are modified when the machine's address changes.
func TestAutoAppName(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// An app is always associated with a machine.
	machine := &Machine{
		ID:        0,
		Address:   "machine-floor1",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	// Add a first app for this machine.
	app1 := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Active:    true,
	}
	_, err = AddApp(db, app1)
	require.NoError(t, err)
	require.NotZero(t, app1.ID)

	// Make sure the name was auto generated.
	require.Equal(t, "kea@machine-floor1", app1.Name)

	// Add a second app of the same type.
	app2 := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Active:    true,
	}
	_, err = AddApp(db, app2)
	require.NoError(t, err)
	require.NotZero(t, app2.ID)

	// The name should be auto generated and the id should be appended to the
	// name to ensure that the apps' names are unique.
	require.Equal(t, fmt.Sprintf("kea@machine-floor1%%%d", app2.ID), app2.Name)

	// Add the third app of different type.
	app3 := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeBind9,
		Active:    true,
	}
	_, err = AddApp(db, app3)
	require.NoError(t, err)
	require.NotZero(t, app3.ID)

	// Its name should have no postfix, because there is only one BIND9 app
	// on this machine.
	require.Equal(t, "bind9@machine-floor1", app3.Name)

	// Modify the machine address. We expect that this will affect the apps'
	// names.
	machine.Address = "machine-floor2"
	_, err = db.Model(machine).WherePK().Update()
	require.NoError(t, err)

	// The first apps' name should be changed to reflect that it now runs
	// on the machine-floor2.
	app1, err = GetAppByID(db, app1.ID)
	require.NoError(t, err)
	require.NotNil(t, app1)
	require.Equal(t, "kea@machine-floor2", app1.Name)

	// The name of the second app should be modified too.
	app2, err = GetAppByID(db, app2.ID)
	require.NoError(t, err)
	require.NotNil(t, app2)
	require.Equal(t, fmt.Sprintf("kea@machine-floor2%%%d", app2.ID), app2.Name)

	// Finally, let's verify the same for the third app.
	app3, err = GetAppByID(db, app3.ID)
	require.NoError(t, err)
	require.NotNil(t, app3)
	require.Equal(t, "bind9@machine-floor2", app3.Name)
}

// Test that app name can be modified and that it is not checked against the
// machine's address if it doesn't follow the [app-type]@[machine-address]
// pattern.
func TestEditAppName(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// An app is always associated with a machine.
	machine := &Machine{
		ID:        0,
		Address:   "machine-floor1",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	// Add a first app for this machine.
	app1 := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Active:    true,
	}
	_, err = AddApp(db, app1)
	require.NoError(t, err)
	require.NotZero(t, app1.ID)

	// Make sure the name was auto generated.
	require.Equal(t, "kea@machine-floor1", app1.Name)

	// Add a second app of the same type.
	app2 := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Active:    true,
	}
	_, err = AddApp(db, app2)
	require.NoError(t, err)
	require.NotZero(t, app2.ID)

	// The name should be auto generated and the id should be appended to the
	// name to ensure that the apps' names are unique.
	require.Equal(t, fmt.Sprintf("kea@machine-floor1%%%d", app2.ID), app2.Name)

	// Let's try to modify first app's name. It should fail, because the
	// machine-floor2 does not exist.
	app1.Name = "fish@machine-floor2"
	_, _, err = UpdateApp(db, app1)
	require.Error(t, err)

	// Try to append app id. This should fail too.
	app1.Name = fmt.Sprintf("fish@machine-floor2%%%d", app1.ID)
	_, _, err = UpdateApp(db, app1)
	require.Error(t, err)

	// But, if we specify a name with a different pattern, it should succeed
	// because we don't specify the machine's name.
	app1.Name = "fish.on.the.floor"
	_, _, err = UpdateApp(db, app1)
	require.NoError(t, err)

	// Update machine address. It should only affect the second app.
	machine.Address = "machine-floor2"
	_, err = db.Model(machine).WherePK().Update()
	require.NoError(t, err)

	// The first app's name should remain the same.
	app1, err = GetAppByID(db, app1.ID)
	require.NoError(t, err)
	require.NotNil(t, app1)
	require.Equal(t, "fish.on.the.floor", app1.Name)

	// The second app's name should change.
	app2, err = GetAppByID(db, app2.ID)
	require.NoError(t, err)
	require.NotNil(t, app2)
	require.Equal(t, fmt.Sprintf("kea@machine-floor2%%%d", app2.ID), app2.Name)

	// When using double at character, the machine name check should not be
	// triggered.
	app1.Name = "kea@@machine-floor3"
	_, _, err = UpdateApp(db, app1)
	require.NoError(t, err)
}

// Test that the app name is auto generated when empty name was specified during
// the app update.
func TestSetEmptyAppName(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// An app is always associated with a machine.
	machine := &Machine{
		ID:        0,
		Address:   "machine-floor1",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	// Add an app for this machine.
	app1 := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Active:    true,
		Name:      "fish",
	}
	_, err = AddApp(db, app1)
	require.NoError(t, err)
	require.NotZero(t, app1.ID)

	// Set empty app name. It should result in auto generating the name.
	app1.Name = " "
	_, _, err = UpdateApp(db, app1)
	require.NoError(t, err)

	// Make sure the name was auto generated.
	app1, err = GetAppByID(db, app1.ID)
	require.NoError(t, err)
	require.Equal(t, "kea@machine-floor1", app1.Name)
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
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234, true)

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
	_, err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

	cr := &ConfigReview{
		ConfigHash: "1234",
		Signature:  "2345",
		DaemonID:   s.Daemons[0].ID,
	}
	err = AddConfigReview(db, cr)
	require.NoError(t, err)

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

	// Make sure the config review was returned.
	require.NotNil(t, apps[0].Daemons[0].ConfigReview)
	require.Equal(t, "1234", apps[0].Daemons[0].ConfigReview.ConfigHash)
	require.Equal(t, "2345", apps[0].Daemons[0].ConfigReview.Signature)
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
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234, false)
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
	_, err = AddApp(db, aKea)
	require.NoError(t, err)
	require.NotZero(t, aKea.ID)

	// add bind9 app
	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "", 2234, false)
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
	_, err = AddApp(db, aBind9)
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
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 4444, false)
	accessPoints = AppendAccessPoint(accessPoints, AccessPointStatistics, "10.0.0.2", "abcd", 5555, true)

	s := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}

	_, err = AddApp(db, s)
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
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234, false)

	sKea := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Name:         "unique-kea",
		Active:       true,
		AccessPoints: keaPoints,
		Meta: AppMeta{
			Version: "1.2.3",
		},
		Daemons: []*Daemon{
			{
				KeaDaemon: &KeaDaemon{
					KeaDHCPDaemon: &KeaDHCPDaemon{},
				},
			},
		},
	}
	_, err = AddApp(db, sKea)
	require.NoError(t, err)
	require.NotZero(t, sKea.ID)

	// add bind app, no error expected
	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "abcd", 4321, true)

	sBind := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Name:         "unique-bind9",
		Active:       true,
		AccessPoints: bind9Points,
		Meta: AppMeta{
			Version: "1.2.4",
		},
		Daemons: []*Daemon{
			{
				Bind9Daemon: &Bind9Daemon{},
			},
		},
	}
	_, err = AddApp(db, sBind)
	require.NoError(t, err)
	require.NotZero(t, sBind.ID)

	// get all apps
	apps, total, err := GetAppsByPage(db, 0, 10, nil, "", "", SortDirAny)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, 2, total)

	// get kea apps
	apps, total, err = GetAppsByPage(db, 0, 10, nil, AppTypeKea, "", SortDirAny)
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
	apps, total, err = GetAppsByPage(db, 0, 10, nil, AppTypeBind9, "", SortDirAny)
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

	// get apps sorted by id descending
	apps, total, err = GetAppsByPage(db, 0, 10, nil, "", "", SortDirDesc)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, 2, total)
	require.Equal(t, AppTypeBind9, apps[0].Type)
	require.Equal(t, AppTypeKea, apps[1].Type)

	// get apps sorted by id ascending
	apps, total, err = GetAppsByPage(db, 0, 10, nil, "", "", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, 2, total)
	require.Equal(t, AppTypeKea, apps[0].Type)
	require.Equal(t, AppTypeBind9, apps[1].Type)

	// get apps sorted by type descending
	apps, total, err = GetAppsByPage(db, 0, 10, nil, "", "type", SortDirDesc)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, 2, total)
	require.Equal(t, AppTypeKea, apps[0].Type)
	require.Equal(t, AppTypeBind9, apps[1].Type)

	// get apps sorted by type ascending
	apps, total, err = GetAppsByPage(db, 0, 10, nil, "", "type", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, 2, total)
	require.Equal(t, AppTypeBind9, apps[0].Type)
	require.Equal(t, AppTypeKea, apps[1].Type)

	// get apps by filter text, case 1
	text := "1.2.3"
	apps, total, err = GetAppsByPage(db, 0, 10, &text, "", "", SortDirAny)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, 1, total)
	require.Equal(t, AppTypeKea, apps[0].Type)

	// get apps by filter text, case 2
	text = "1.2.4"
	apps, total, err = GetAppsByPage(db, 0, 10, &text, "", "", SortDirAny)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, 1, total)
	require.Equal(t, AppTypeBind9, apps[0].Type)

	// get apps by filter text, case 3
	text = "unique"
	apps, total, err = GetAppsByPage(db, 0, 10, &text, "", "", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, 2, total)
	require.Equal(t, AppTypeKea, apps[0].Type)
	require.Equal(t, AppTypeBind9, apps[1].Type)

	// get apps by filter text, case 4
	text = "unique-k"
	apps, total, err = GetAppsByPage(db, 0, 10, &text, "", "", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, 1, total)
	require.Equal(t, AppTypeKea, apps[0].Type)

	// get apps by filter text, case 5
	text = "unique-b"
	apps, total, err = GetAppsByPage(db, 0, 10, &text, "", "", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, 1, total)
	require.Equal(t, AppTypeBind9, apps[0].Type)
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

// Test that a single name of the active DHCP daemon is returned.
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

// Test that empty list of daemons is returned if the daemon type
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
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234, false)

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
	_, err = AddApp(db, aKea)
	require.NoError(t, err)
	require.NotZero(t, aKea.ID)

	// add bind app, no error expected
	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "abcd", 4321, true)

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
	_, err = AddApp(db, aBind)
	require.NoError(t, err)
	require.NotZero(t, aBind.ID)

	// get all apps
	apps, err := GetAllApps(db, true)
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

// Test that the relations can be selected when querying for all apps.
func TestGetAllAppsWithRelations(t *testing.T) {
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

	var keaPoints []*AccessPoint
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234, false)

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
				LogTargets: []*LogTarget{
					{
						Name:     "frog",
						Severity: "ERROR",
						Output:   "stdout",
					},
				},
			},
		},
	}
	_, err = AddApp(db, aKea)
	require.NoError(t, err)
	require.NotZero(t, aKea.ID)

	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "abcd", 4321, true)

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
	_, err = AddApp(db, aBind)
	require.NoError(t, err)
	require.NotZero(t, aBind.ID)

	t.Run("machine", func(t *testing.T) {
		apps, err := GetAllAppsWithRelations(db, AppRelationMachine)
		require.NoError(t, err)
		require.Len(t, apps, 2)
		require.NotEqual(t, apps[0].Type, apps[1].Type)
		for _, app := range apps {
			require.NotNil(t, app.Machine)
			require.Empty(t, app.Daemons)
			require.Empty(t, app.AccessPoints)
		}
	})

	t.Run("access points", func(t *testing.T) {
		apps, err := GetAllAppsWithRelations(db, AppRelationMachine, AppRelationAccessPoints)
		require.NoError(t, err)
		require.Len(t, apps, 2)
		require.NotEqual(t, apps[0].Type, apps[1].Type)
		for _, app := range apps {
			require.NotNil(t, app.Machine)
			require.Empty(t, app.Daemons)
			require.Len(t, app.AccessPoints, 1)
		}
	})

	t.Run("daemons", func(t *testing.T) {
		apps, err := GetAllAppsWithRelations(db, AppRelationMachine, AppRelationDaemons)
		require.NoError(t, err)
		require.Len(t, apps, 2)
		require.NotEqual(t, apps[0].Type, apps[1].Type)
		for _, app := range apps {
			require.NotNil(t, app.Machine)
			require.Len(t, app.Daemons, 1)
			require.Nil(t, app.Daemons[0].KeaDaemon)
			require.Empty(t, app.AccessPoints)
		}
	})

	t.Run("log targets", func(t *testing.T) {
		apps, err := GetAllAppsWithRelations(db, AppRelationMachine, AppRelationDaemonsLogTargets)
		require.NoError(t, err)
		require.Len(t, apps, 2)
		require.NotEqual(t, apps[0].Type, apps[1].Type)
		for _, app := range apps {
			require.NotNil(t, app.Machine)
			require.Len(t, app.Daemons, 1)
			require.Nil(t, app.Daemons[0].KeaDaemon)
			require.Empty(t, app.AccessPoints)

			switch app.Type {
			case AppTypeKea:
				require.Len(t, app.Daemons[0].LogTargets, 1)
			default:
				require.Empty(t, app.Daemons[0].LogTargets)
			}
		}
	})

	t.Run("kea daemons", func(t *testing.T) {
		apps, err := GetAllAppsWithRelations(db, AppRelationMachine, AppRelationDaemons, AppRelationKeaDaemons)
		require.NoError(t, err)
		require.Len(t, apps, 2)
		require.NotEqual(t, apps[0].Type, apps[1].Type)
		for _, app := range apps {
			require.NotNil(t, app.Machine)
			require.Len(t, app.Daemons, 1)
			require.Empty(t, app.AccessPoints)

			switch app.Type {
			case AppTypeKea:
				require.NotNil(t, app.Daemons[0].KeaDaemon)
				require.Nil(t, app.Daemons[0].KeaDaemon.KeaDHCPDaemon)
			default:
				require.Nil(t, app.Daemons[0].KeaDaemon)
			}
		}
	})

	t.Run("kea dhcp daemons", func(t *testing.T) {
		apps, err := GetAllAppsWithRelations(db, AppRelationMachine, AppRelationDaemons, AppRelationKeaDaemons, AppRelationKeaDHCPDaemons)
		require.NoError(t, err)
		require.Len(t, apps, 2)
		require.NotEqual(t, apps[0].Type, apps[1].Type)
		for _, app := range apps {
			require.NotNil(t, app.Machine)
			require.Len(t, app.Daemons, 1)
			require.Empty(t, app.AccessPoints)

			switch app.Type {
			case AppTypeKea:
				require.NotNil(t, app.Daemons[0].KeaDaemon)
				require.NotNil(t, app.Daemons[0].KeaDaemon.KeaDHCPDaemon)
			default:
				require.Nil(t, app.Daemons[0].KeaDaemon)
			}
		}
	})

	t.Run("bind9 daemons", func(t *testing.T) {
		apps, err := GetAllAppsWithRelations(db, AppRelationMachine, AppRelationDaemons, AppRelationKeaDaemons, AppRelationBind9Daemons)
		require.NoError(t, err)
		require.Len(t, apps, 2)
		require.NotEqual(t, apps[0].Type, apps[1].Type)
		for _, app := range apps {
			require.NotNil(t, app.Machine)
			require.Len(t, app.Daemons, 1)
			require.Empty(t, app.AccessPoints)

			switch app.Type {
			case AppTypeKea:
				require.NotNil(t, app.Daemons[0].KeaDaemon)
				require.Nil(t, app.Daemons[0].KeaDaemon.KeaDHCPDaemon)
				require.Nil(t, app.Daemons[0].Bind9Daemon)
			default:
				require.Nil(t, app.Daemons[0].KeaDaemon)
				require.NotNil(t, app.Daemons[0].Bind9Daemon)
			}
		}
	})
}

// Tests that daemon can be found by name for an app.
func TestGetDaemonByName(t *testing.T) {
	app := &App{
		Daemons: []*Daemon{
			{
				Name: "kea-dhcp4",
			},
			{
				Name: "kea-dhcp6",
			},
		},
	}
	daemon := app.GetDaemonByName("kea-dhcp4")
	require.NotNil(t, daemon)
	require.Same(t, daemon, app.Daemons[0])

	daemon = app.GetDaemonByName("kea-dhcp6")
	require.NotNil(t, daemon)
	require.Same(t, daemon, app.Daemons[1])

	require.Nil(t, app.GetDaemonByName("kea-ca"))
}

// Test AppTag interface implementation.
func TestAppTag(t *testing.T) {
	app := App{
		ID:   11,
		Name: "kea@xyz",
		Type: AppTypeKea,
		Meta: AppMeta{
			Version: "2.1.1",
		},
		MachineID: 42,
	}
	require.EqualValues(t, 11, app.GetID())
	require.Equal(t, "kea@xyz", app.GetName())
	require.Equal(t, AppTypeKea, app.GetType())
	require.Equal(t, "2.1.1", app.GetVersion())
	require.EqualValues(t, 42, app.GetMachineID())
}

// Test getting control access point.
func TestGetControlAccessPoint(t *testing.T) {
	app := &App{}

	// An error should be returned when there is no control access point.
	address, port, key, secure, err := app.GetControlAccessPoint()
	require.Error(t, err)
	require.Empty(t, address)
	require.Zero(t, port)
	require.Empty(t, key)
	require.False(t, secure)

	// Specify control access point and check it is returned.
	app.AccessPoints = AppendAccessPoint(app.AccessPoints, AccessPointControl, "cool.example.org", "key", 1234, true)
	address, port, key, secure, err = app.GetControlAccessPoint()
	require.NoError(t, err)
	require.Equal(t, "cool.example.org", address)
	require.Equal(t, "key", key)
	require.EqualValues(t, 1234, port)
	require.True(t, secure)
}

// Test getting MachineTag interface from an app.
func TestGetMachineTag(t *testing.T) {
	app := App{
		Machine: &Machine{
			ID: 10,
		},
	}
	machine := app.GetMachineTag()
	require.NotNil(t, machine)
	require.EqualValues(t, 10, machine.GetID())
}

// Test getting DaemonTag interfaces from an app that has app
// type at the app level.
func TestGetDaemonTagsAppType(t *testing.T) {
	app := App{
		Type: AppTypeBind9,
		Daemons: []*Daemon{
			{
				ID: 10,
			},
			{
				ID: 11,
			},
		},
	}
	daemons := app.GetDaemonTags()
	require.Len(t, daemons, 2)
	require.EqualValues(t, 10, daemons[0].GetID())
	require.Equal(t, AppTypeBind9, daemons[0].GetAppType())
	require.EqualValues(t, 11, daemons[1].GetID())
	require.Equal(t, AppTypeBind9, daemons[1].GetAppType())
}

// Test getting DaemonTag interfaces from an app.
func TestGetDaemonTagsKea(t *testing.T) {
	app := App{
		Type: AppTypeBind9,
		Daemons: []*Daemon{
			{
				ID: 10,
			},
			{
				ID: 11,
			},
		},
	}
	daemons := app.GetDaemonTags()
	require.Len(t, daemons, 2)
	require.EqualValues(t, 10, daemons[0].GetID())
	require.Equal(t, AppTypeBind9, daemons[0].GetAppType())
	require.EqualValues(t, 11, daemons[1].GetID())
	require.Equal(t, AppTypeBind9, daemons[1].GetAppType())
}

// Test getting a selected daemon tag.
func TestGetDaemonTagKea(t *testing.T) {
	app := App{
		Type: AppTypeKea,
		Daemons: []*Daemon{
			{
				ID:   10,
				Name: DaemonNameDHCPv4,
			},
			{
				ID:   11,
				Name: DaemonNameDHCPv6,
			},
		},
	}
	daemon := app.GetDaemonTag(DaemonNameDHCPv4)
	require.NotNil(t, daemon)
	require.EqualValues(t, 10, daemon.GetID())

	daemon = app.GetDaemonTag(DaemonNameDHCPv6)
	require.NotNil(t, daemon)
	require.EqualValues(t, 11, daemon.GetID())

	daemon = app.GetDaemonTag(DaemonNameD2)
	require.Nil(t, daemon)
}
