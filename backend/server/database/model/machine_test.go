package dbmodel

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database/test"
)

func TestAddMachine(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		Id: 0,
		Address: "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.Id)

	// add another one but with the same address - an error should be raised
	m2 := &Machine{
		Address: "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m2)
	require.Contains(t, err.Error(), "duplicate")
}

func TestGetMachineByAddress(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing machine
	m, err := GetMachineByAddressAndAgentPort(db, "localhost", 8080, false)
	require.Nil(t, err)
	require.Nil(t, m)

	// add machine
	m2 := &Machine{
		Address: "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m2)
	require.NoError(t, err)

	// get added machine
	m, err = GetMachineByAddressAndAgentPort(db, "localhost", 8080, false)
	require.Nil(t, err)
	require.Equal(t, m2.Address, m.Address)

	// delete machine
	err = DeleteMachine(db, m)
	require.Nil(t, err)

	// get deleted machine while do not include deleted machines
	m, err = GetMachineByAddressAndAgentPort(db, "localhost", 8080, false)
	require.Nil(t, err)
	require.Nil(t, m)

	// get deleted machine but this time include deleted machines
	m, err = GetMachineByAddressAndAgentPort(db, "localhost", 8080, true)
	require.Nil(t, err)
	require.Equal(t, m2.Address, m.Address)
}

func TestGetMachineById(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing machine
	m, err := GetMachineById(db, 123)
	require.Nil(t, err)
	require.Nil(t, m)

	// add machine
	m2 := &Machine{
		Address: "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m2)
	require.NoError(t, err)

	// get added machine
	m, err = GetMachineById(db, int64(m2.Id))
	require.Nil(t, err)
	require.Equal(t, m2.Address, m.Address)

	// delete machine
	err = DeleteMachine(db, m)
	require.Nil(t, err)

	// even if machine was delete it should be gettable by id
	m, err = GetMachineById(db, int64(m2.Id))
	require.Nil(t, err)
	require.Equal(t, m2.Address, m.Address)
}

func TestGetMachinesByPageBasic(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// no machines yet but try to get some
	ms, total, err := GetMachinesByPage(db, 0, 10, "")
	require.Nil(t, err)
	require.Equal(t, int64(0), total)
	require.Len(t, ms, 0)

	// add 10 machines
	for i := 1; i <= 10; i++ {
		m := &Machine{
			Address: fmt.Sprintf("host-%d", i),
			AgentPort: int64(i),
		}
		err = AddMachine(db, m)
		require.NoError(t, err)
	}

	// get 10 machines from 0
	ms, total, err = GetMachinesByPage(db, 0, 10, "")
	require.Nil(t, err)
	require.Equal(t, int64(10), total)
	require.Len(t, ms, 10)

	// get 2 machines out of 10, from 0
	ms, total, err = GetMachinesByPage(db, 0, 2, "")
	require.Nil(t, err)
	require.Equal(t, int64(10), total)
	require.Len(t, ms, 2)

	// get 3 machines out of 10, from 2
	ms, total, err = GetMachinesByPage(db, 2, 3, "")
	require.Nil(t, err)
	require.Equal(t, int64(10), total)
	require.Len(t, ms, 3)

	// get 10 machines out of 10, from 0, but with '1' in contents; should return 2: 1 and 10
	ms, total, err = GetMachinesByPage(db, 0, 10, "1")
	require.Nil(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, ms, 2)
}

func TestGetMachinesByPageWithFiltering(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address: "localhost",
		AgentPort: 8080,
		State: MachineState{
			Hostname: "my-host",
			PlatformFamily: "redhat",
		},
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// filter machines by json fields: redhat
	ms, total, err := GetMachinesByPage(db, 0, 10, "redhat")
	require.Nil(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, ms, 1)

	// filter machines by json fields: my
	ms, total, err = GetMachinesByPage(db, 0, 10, "my")
	require.Nil(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, ms, 1)
}

func TestDeleteMachine(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address: "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// delete machine
	err = DeleteMachine(db, m)
	require.Nil(t, err)

	// delete non-existing machine
	m2 := &Machine{
		Id: 123,
		Address: "localhost",
		AgentPort: 123,
	}
	err = DeleteMachine(db, m2)
	require.Contains(t, err.Error(), "no rows in result")
}

func TestRefreshMachineFromDb(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address: "localhost",
		AgentPort: 8080,
		Error: "some error",
		State: MachineState{
			Hostname: "aaaa",
			Cpus: 4,
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
	require.Equal(t, int64(4), m.State.Cpus)
	require.Equal(t, "some error", m.Error)
}
