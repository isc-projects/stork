package dbmodel

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	dbtest "isc.org/stork/server/database/test"
)

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
	err = AddApp(db, a)
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
	}
	err = AddApp(db, a)
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

	// delete machine
	err = DeleteMachine(db, m)
	require.Nil(t, err)

	m, err = GetMachineByID(db, m2.ID)
	require.NoError(t, err)
	require.Nil(t, m)
}

func TestGetMachinesByPageBasic(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// no machines yet but try to get some
	ms, total, err := GetMachinesByPage(db, 0, 10, "")
	require.Nil(t, err)
	require.EqualValues(t, 0, total)
	require.Len(t, ms, 0)

	// add 10 machines
	for i := 1; i <= 10; i++ {
		m := &Machine{
			Address:   fmt.Sprintf("host-%d", i),
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
		err = AddApp(db, a)
		require.NoError(t, err)
	}

	// get 10 machines from 0
	ms, total, err = GetMachinesByPage(db, 0, 10, "")
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)

	// get 2 machines out of 10, from 0
	ms, total, err = GetMachinesByPage(db, 0, 2, "")
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 2)

	// get 3 machines out of 10, from 2
	ms, total, err = GetMachinesByPage(db, 2, 3, "")
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 3)

	// get 10 machines out of 10, from 0, but with '1' in contents; should return 2: 1 and 10
	ms, total, err = GetMachinesByPage(db, 0, 10, "1")
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
	require.EqualValues(t, 8010, ms[1].Apps[0].AccessPoints[0].Port)
	require.Empty(t, ms[1].Apps[0].AccessPoints[0].Key)
}

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
	ms, total, err := GetMachinesByPage(db, 0, 10, "redhat")
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)

	// filter machines by json fields: my
	ms, total, err = GetMachinesByPage(db, 0, 10, "my")
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)
}

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
	err = AddApp(db, a)
	require.NoError(t, err)
	appID := a.ID
	require.NotEqual(t, 0, appID)

	// reload machine from db to get apps relation loaded
	err = RefreshMachineFromDb(db, m)
	require.Nil(t, err)

	// delete machine
	err = DeleteMachine(db, m)
	require.NoError(t, err)

	// check if app is also deleted
	a, err = GetAppByID(db, appID)
	require.NoError(t, err)
	require.Nil(t, a)
}

func TestRefreshMachineFromDb(t *testing.T) {
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

	err = RefreshMachineFromDb(db, m)
	require.Nil(t, err)
	require.Equal(t, "aaaa", m.State.Hostname)
	require.EqualValues(t, 4, m.State.Cpus)
	require.Equal(t, "some error", m.Error)
}
