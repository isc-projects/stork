package dbmodel

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Check if adding machine to database works.
func TestAddMachine(t *testing.T) {
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

	// add another one but with the same address - an error should be raised
	m2 := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m2)
	require.Contains(t, err.Error(), "duplicate")
}

// Check if updating machine in database works.
func TestUpdateMachine(t *testing.T) {
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

	// change authorization
	m1, err := GetMachineByID(db, m.ID)
	require.NoError(t, err)
	require.False(t, m1.Authorized)

	m.Authorized = true
	err = UpdateMachine(db, m)
	require.NoError(t, err)

	m2, err := GetMachineByID(db, m.ID)
	require.NoError(t, err)
	require.True(t, m2.Authorized)
}

// Check if getting machine by address.
func TestGetMachineByAddress(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing machine
	m, err := GetMachineByAddressAndAgentPort(db, "localhost", 8080)
	require.Nil(t, err)
	require.Nil(t, m)

	// add machine
	m2 := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m2)
	require.NoError(t, err)

	// add app
	a := &App{
		ID:        0,
		MachineID: m2.ID,
		Type:      "kea",
		AccessPoints: []*AccessPoint{
			{
				MachineID: m2.ID,
				Type:      "control",
				Address:   "localhost",
				Port:      1234,
				Key:       "",
			},
		},
	}
	_, err = AddApp(db, a)
	require.NoError(t, err)

	// get added machine
	m, err = GetMachineByAddressAndAgentPort(db, "localhost", 8080)
	require.Nil(t, err)
	require.Equal(t, m2.Address, m.Address)
	require.Len(t, m.Apps, 1)
	require.Len(t, m.Apps[0].AccessPoints, 1)
	require.Equal(t, "control", m.Apps[0].AccessPoints[0].Type)
	require.Equal(t, "localhost", m.Apps[0].AccessPoints[0].Address)
	require.EqualValues(t, 1234, m.Apps[0].AccessPoints[0].Port)
	require.Empty(t, m.Apps[0].AccessPoints[0].Key)

	// delete machine
	err = DeleteMachine(db, m)
	require.Nil(t, err)

	// get deleted machine while do not include deleted machines
	m, err = GetMachineByAddressAndAgentPort(db, "localhost", 8080)
	require.Nil(t, err)
	require.Nil(t, m)
}

// Check if getting machine by its ID.
func TestGetMachineByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing machine
	m, err := GetMachineByID(db, 123)
	require.Nil(t, err)
	require.Nil(t, m)

	// add machine
	m2 := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m2)
	require.NoError(t, err)

	// add app
	a := &App{
		ID:        0,
		MachineID: m2.ID,
		Type:      "bind9",
		AccessPoints: []*AccessPoint{
			{
				MachineID: m2.ID,
				Type:      "control",
				Address:   "dns.example.",
				Port:      953,
				Key:       "abcd",
			},
		},
		Daemons: []*Daemon{
			{
				Name:    "kea-dhcp4",
				Version: "1.7.5",
				Active:  true,
			},
		},
	}
	_, err = AddApp(db, a)
	require.NoError(t, err)

	// get added machine
	m, err = GetMachineByID(db, m2.ID)
	require.Nil(t, err)
	require.Equal(t, m2.Address, m.Address)
	require.Len(t, m.Apps, 1)
	require.Len(t, m.Apps[0].AccessPoints, 1)
	require.Equal(t, "control", m.Apps[0].AccessPoints[0].Type)
	require.Equal(t, "dns.example.", m.Apps[0].AccessPoints[0].Address)
	require.EqualValues(t, 953, m.Apps[0].AccessPoints[0].Port)
	require.Equal(t, "abcd", m.Apps[0].AccessPoints[0].Key)
	require.Len(t, m.Apps[0].Daemons, 1)
	require.Equal(t, "kea-dhcp4", m.Apps[0].Daemons[0].Name)

	// delete machine
	err = DeleteMachine(db, m)
	require.Nil(t, err)

	m, err = GetMachineByID(db, m2.ID)
	require.NoError(t, err)
	require.Nil(t, m)
}

// Check if getting machine by its ID with relations.
func TestGetMachineByIDWithRelations(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:         42,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
		AgentToken: "secret",
	}
	_ = AddMachine(db, m)

	a := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      "bind9",
		AccessPoints: []*AccessPoint{
			{
				MachineID: m.ID,
				Type:      "control",
				Address:   "dns.example.",
				Port:      953,
				Key:       "abcd",
			},
		},
		Daemons: []*Daemon{
			NewKeaDaemon(DaemonNameDHCPv4, true),
			{
				Name:    DaemonNameBind9,
				Version: "1.0.0",
				Active:  true,
				LogTargets: []*LogTarget{
					{
						Output: "stdout",
					},
					{
						Output: "/tmp/filename.log",
					},
				},
				Bind9Daemon: &Bind9Daemon{},
			},
		},
	}
	ds, _ := AddApp(db, a)

	d := ds[0]
	_ = d.SetConfigFromJSON(`{
        "Dhcp4": {
            "valid-lifetime": 1234,
			"secret": "hidden"
        }
    }`)
	_ = UpdateDaemon(db, d)

	// Act
	machine, machineErr := GetMachineByIDWithRelations(db, 42)
	machineApps, machineAppsErr := GetMachineByIDWithRelations(db, 42, MachineRelationApps)
	machineDaemons, machineDaemonsErr := GetMachineByIDWithRelations(db, 42, MachineRelationDaemons)
	machineKeaDaemons, machineKeaDaemonsErr := GetMachineByIDWithRelations(db, 42, MachineRelationKeaDaemons)
	machineBind9Daemons, machineBind9DaemonsErr := GetMachineByIDWithRelations(db, 42, MachineRelationBind9Daemons)
	machineDaemonLogTargets, machineDaemonLogTargetsErr := GetMachineByIDWithRelations(db, 42, MachineRelationDaemonLogTargets)
	machineAppAccessPoints, machineAppAccessPointsErr := GetMachineByIDWithRelations(db, 42, MachineRelationAppAccessPoints)
	machineKeaDHCPConfigs, machineKeaDHCPConfigsErr := GetMachineByIDWithRelations(db, 42, MachineRelationKeaDHCPConfigs)
	machineAppAccessPointsKeaDHCPConfigs, machineAppAccessPointsKeaDHCPConfigsErr := GetMachineByIDWithRelations(db, 42, MachineRelationAppAccessPoints, MachineRelationKeaDHCPConfigs)

	// Assert
	require.NoError(t, machineErr)
	require.NoError(t, machineAppsErr)
	require.NoError(t, machineDaemonsErr)
	require.NoError(t, machineKeaDaemonsErr)
	require.NoError(t, machineBind9DaemonsErr)
	require.NoError(t, machineDaemonLogTargetsErr)
	require.NoError(t, machineAppAccessPointsErr)
	require.NoError(t, machineKeaDHCPConfigsErr)
	require.NoError(t, machineAppAccessPointsKeaDHCPConfigsErr)

	// Just machine
	require.NotNil(t, machine.State)
	require.Len(t, machine.Apps, 0)
	// Machine with apps
	require.Nil(t, machineApps.Apps[0].AccessPoints)
	require.Nil(t, machineApps.Apps[0].Daemons)
	// Machine with daemons
	require.Nil(t, machineDaemons.Apps[0].AccessPoints)
	require.Len(t, machineDaemons.Apps[0].Daemons, 2)
	require.Nil(t, machineDaemons.Apps[0].Daemons[1].LogTargets)
	require.Nil(t, machineDaemons.Apps[0].Daemons[1].KeaDaemon)
	require.Nil(t, machineDaemons.Apps[0].Daemons[0].Bind9Daemon)
	// Machine with kea daemons
	require.Nil(t, machineKeaDaemons.Apps[0].Daemons[0].Bind9Daemon)
	require.Nil(t, machineKeaDaemons.Apps[0].Daemons[1].KeaDaemon.KeaDHCPDaemon)
	require.Nil(t, machineKeaDaemons.Apps[0].Daemons[1].LogTargets)
	// Machine with Bind9 daemons
	require.NotNil(t, machineBind9Daemons.Apps[0].Daemons[0].Bind9Daemon)
	require.Nil(t, machineBind9Daemons.Apps[0].Daemons[1].KeaDaemon)
	require.Nil(t, machineBind9Daemons.Apps[0].Daemons[1].LogTargets)
	// Machine with daemon log targets
	require.Len(t, machineDaemonLogTargets.Apps[0].Daemons[0].LogTargets, 2)
	require.Nil(t, machineDaemonLogTargets.Apps[0].Daemons[1].KeaDaemon)
	require.Nil(t, machineDaemonLogTargets.Apps[0].Daemons[0].Bind9Daemon)
	require.Nil(t, machineDaemonLogTargets.Apps[0].AccessPoints)
	// Machine with the access points
	require.NotNil(t, machineAppAccessPoints.Apps[0].AccessPoints)
	require.Nil(t, machineAppAccessPoints.Apps[0].Daemons)
	// Machine with Kea DHCP configurations
	require.NotNil(t, machineKeaDHCPConfigs.Apps[0].Daemons[1].KeaDaemon.KeaDHCPDaemon)
	// Machine with the access points and Kea DHCP configurations
	require.NotNil(t, machineAppAccessPointsKeaDHCPConfigs.Apps[0].Daemons[1].KeaDaemon.KeaDHCPDaemon)
	require.Len(t, machineAppAccessPointsKeaDHCPConfigs.Apps[0].AccessPoints, 1)
}

// Basic check if getting machines by pages works.
func TestGetMachinesByPageBasic(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// no machines yet but try to get some
	ms, total, err := GetMachinesByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.Zero(t, total)
	require.Len(t, ms, 0)

	// add 10 machines
	for i := 1; i <= 10; i++ {
		m := &Machine{
			Address:   fmt.Sprintf("host-%d", 21-i),
			AgentPort: int64(i),
		}
		err = AddMachine(db, m)
		require.NoError(t, err)

		// add app
		a := &App{
			ID:        0,
			MachineID: m.ID,
			Type:      "bind9",
			AccessPoints: []*AccessPoint{
				{
					MachineID: m.ID,
					Type:      "control",
					Address:   "localhost",
					Port:      int64(8000 + i),
					Key:       "",
				},
			},
		}
		_, err = AddApp(db, a)
		require.NoError(t, err)
	}

	// get 10 machines from 0
	ms, total, err = GetMachinesByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)

	// get 2 machines out of 10, from 0
	ms, total, err = GetMachinesByPage(db, 0, 2, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 2)

	// get 3 machines out of 10, from 2
	ms, total, err = GetMachinesByPage(db, 2, 3, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 3)

	// get 10 machines out of 10, from 0, but with '2' in contents; should return 1: 20 and 12
	text := "2"
	ms, total, err = GetMachinesByPage(db, 0, 10, &text, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, ms, 2)

	// check machine details
	require.Len(t, ms[0].Apps, 1)
	require.Len(t, ms[0].Apps[0].AccessPoints, 1)
	require.Equal(t, "control", ms[0].Apps[0].AccessPoints[0].Type)
	require.Equal(t, "localhost", ms[0].Apps[0].AccessPoints[0].Address)
	require.EqualValues(t, 8001, ms[0].Apps[0].AccessPoints[0].Port)
	require.Empty(t, ms[0].Apps[0].AccessPoints[0].Key)

	require.Len(t, ms[1].Apps, 1)
	require.Len(t, ms[1].Apps[0].AccessPoints, 1)
	require.Equal(t, "control", ms[1].Apps[0].AccessPoints[0].Type)
	require.Equal(t, "localhost", ms[1].Apps[0].AccessPoints[0].Address)
	require.EqualValues(t, 8009, ms[1].Apps[0].AccessPoints[0].Port)
	require.Empty(t, ms[1].Apps[0].AccessPoints[0].Key)

	// check sorting by id asc
	ms, total, err = GetMachinesByPage(db, 0, 100, nil, nil, "", SortDirAsc)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)
	require.EqualValues(t, 1, ms[0].ID)
	require.EqualValues(t, 6, ms[5].ID)

	// check sorting by id desc
	ms, total, err = GetMachinesByPage(db, 0, 100, nil, nil, "", SortDirDesc)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)
	require.EqualValues(t, 10, ms[0].ID)
	require.EqualValues(t, 5, ms[5].ID)

	// check sorting by address asc
	ms, total, err = GetMachinesByPage(db, 0, 100, nil, nil, "address", SortDirAsc)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)
	require.EqualValues(t, 10, ms[0].ID)
	require.EqualValues(t, 5, ms[5].ID)

	// check sorting by address desc
	ms, total, err = GetMachinesByPage(db, 0, 100, nil, nil, "address", SortDirDesc)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)
	require.EqualValues(t, 1, ms[0].ID)
	require.EqualValues(t, 6, ms[5].ID)
}

// Check if getting machines with filtering by pages works.
func TestGetMachinesByPageWithFiltering(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
		State: MachineState{
			Hostname:       "my-host",
			PlatformFamily: "redhat",
		},
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// filter machines by json fields: redhat
	text := "redhat"
	ms, total, err := GetMachinesByPage(db, 0, 10, &text, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)

	// filter machines by json fields: my
	text = "my"
	ms, total, err = GetMachinesByPage(db, 0, 10, &text, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)
}

// Check if getting authorized machines with filtering by pages works.
func TestGetMachinesByPageFilteredByAuthorized(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine2
	m := &Machine{
		Address:   "unauthorized",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	m = &Machine{
		Address:    "authorized",
		AgentPort:  8080,
		Authorized: true,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)

	// get unauthorized machines
	authorized := false
	ms, total, err := GetMachinesByPage(db, 0, 10, nil, &authorized, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)
	require.EqualValues(t, "unauthorized", ms[0].Address)
	require.False(t, ms[0].Authorized)

	// get authorized machines
	authorized = true
	ms, total, err = GetMachinesByPage(db, 0, 10, nil, &authorized, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)
	require.EqualValues(t, "authorized", ms[0].Address)
	require.True(t, ms[0].Authorized)

	// get all machines
	ms, total, err = GetMachinesByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, ms, 2)
}

// Check if deleting only machine works.
func TestDeleteMachineOnly(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// delete machine
	err = DeleteMachine(db, m)
	require.Nil(t, err)

	// delete non-existing machine
	m2 := &Machine{
		ID:        123,
		Address:   "localhost",
		AgentPort: 123,
	}
	err = DeleteMachine(db, m2)
	require.Contains(t, err.Error(), "database entry not found")
}

// Check if deleting machine and its apps works.
func TestDeleteMachineWithApps(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// add app
	a := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      AppTypeKea,
	}
	_, err = AddApp(db, a)
	require.NoError(t, err)
	appID := a.ID
	require.NotEqual(t, 0, appID)

	// reload machine from db to get apps relation loaded
	err = RefreshMachineFromDB(db, m)
	require.Nil(t, err)

	// delete machine
	err = DeleteMachine(db, m)
	require.NoError(t, err)

	// check if app is also deleted
	a, err = GetAppByID(db, appID)
	require.NoError(t, err)
	require.Nil(t, a)
}

// Check if refreshing machine works.
func TestRefreshMachineFromDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
		Error:     "some error",
		State: MachineState{
			Hostname: "aaaa",
			Cpus:     4,
		},
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	m.State.Hostname = "bbbb"
	m.State.Cpus = 2
	m.Error = ""

	err = RefreshMachineFromDB(db, m)
	require.Nil(t, err)
	require.Equal(t, "aaaa", m.State.Hostname)
	require.EqualValues(t, 4, m.State.Cpus)
	require.Equal(t, "some error", m.Error)
}

// Check if getting all machines works.
func TestGetAllMachines(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add 20 machines
	for i := 1; i <= 20; i++ {
		m := &Machine{
			Address:   "localhost",
			AgentPort: 8080 + int64(i),
			Error:     "some error",
			State: MachineState{
				Hostname: "aaaa",
				Cpus:     4,
			},
			Authorized: i%2 == 0,
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		a := &App{
			MachineID: m.ID,
			Type:      AppTypeKea,
			AccessPoints: []*AccessPoint{
				{
					MachineID: m.ID,
					Type:      "control",
					Address:   "localhost",
					Port:      1234,
					Key:       "",
				},
			},
			Daemons: []*Daemon{
				{
					Name:   "dhcp4",
					Active: true,
				},
			},
		}
		_, err = AddApp(db, a)
		require.NoError(t, err)

		cr := &ConfigReview{
			ConfigHash: "1234",
			Signature:  "2345",
			DaemonID:   a.Daemons[0].ID,
		}
		err = AddConfigReview(db, cr)
		require.NoError(t, err)
	}

	// get all machines should return 20 machines
	machines, err := GetAllMachines(db, nil)
	require.NoError(t, err)
	require.Len(t, machines, 20)
	require.EqualValues(t, "localhost", machines[0].Address)
	require.EqualValues(t, "localhost", machines[19].Address)
	require.EqualValues(t, "some error", machines[0].Error)
	require.EqualValues(t, "some error", machines[19].Error)
	require.EqualValues(t, 4, machines[0].State.Cpus)
	require.EqualValues(t, 4, machines[19].State.Cpus)
	require.NotEqual(t, machines[0].AgentPort, machines[19].AgentPort)

	// Ensure that we fetched apps, daemons and config reviews too.
	require.Len(t, machines[0].Apps, 1)
	require.Len(t, machines[0].Apps[0].Daemons, 1)
	require.NotNil(t, machines[0].Apps[0].Daemons[0].ConfigReview)
	require.Equal(t, "1234", machines[0].Apps[0].Daemons[0].ConfigReview.ConfigHash)
	require.Equal(t, "2345", machines[0].Apps[0].Daemons[0].ConfigReview.Signature)

	// get only unauthorized machines
	authorized := false
	machines, err = GetAllMachines(db, &authorized)
	require.NoError(t, err)
	require.Len(t, machines, 10)

	// and now only authorized machines
	authorized = true
	machines, err = GetAllMachines(db, &authorized)
	require.NoError(t, err)
	require.Len(t, machines, 10)

	// paged get should return indicated limit, not all
	machines, total, err := GetMachinesByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.Len(t, machines, 10)
	require.EqualValues(t, 20, total)
}

// Test MachineTag interface implementation.
func TestMachineTag(t *testing.T) {
	machine := Machine{
		ID:        10,
		Address:   "192.0.2.2",
		AgentPort: 1234,
		State: MachineState{
			Hostname: "cool.example.org",
		},
	}
	require.EqualValues(t, 10, machine.GetID())
	require.Equal(t, "192.0.2.2", machine.GetAddress())
	require.EqualValues(t, 1234, machine.GetAgentPort())
	require.Equal(t, "cool.example.org", machine.GetHostname())
}
