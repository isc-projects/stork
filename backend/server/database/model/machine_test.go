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

// Basic check if getting machines by pages works.
func TestGetMachinesByPageBasic(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// no machines yet but try to get some
	ms, total, err := GetMachinesByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 0, total)
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
	require.EqualValues(t, false, ms[0].Authorized)

	// get authorized machines
	authorized = true
	ms, total, err = GetMachinesByPage(db, 0, 10, nil, &authorized, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)
	require.EqualValues(t, "authorized", ms[0].Address)
	require.EqualValues(t, true, ms[0].Authorized)

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
	require.Contains(t, err.Error(), "no rows in result")
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
