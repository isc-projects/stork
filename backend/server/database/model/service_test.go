package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database/test"
)

func TestAddService(t *testing.T) {
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

	// add service but without machine, error should be raised
	s := &Service{
		Id: 0,
		Type: "kea",
	}
	err = AddService(db, s)
	require.NotNil(t, err)

	// add service but without type, error should be raised
	s = &Service{
		Id: 0,
		MachineID: m.Id,
	}
	err = AddService(db, s)
	require.NotNil(t, err)

	// add service, no error expected
	s = &Service{
		Id: 0,
		MachineID: m.Id,
		Type: "kea",
		CtrlPort: 1234,
		Active: true,
	}
	err = AddService(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.Id)

	// add service for the same machine and ctrl port - error should be raised
	s = &Service{
		Id: 0,
		MachineID: m.Id,
		Type: "bind",
		CtrlPort: 1234,
		Active: true,
	}
	err = AddService(db, s)
	require.Contains(t, err.Error(), "duplicate")
}

func TestDeleteService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// delete non-existing service
	s0 := &Service{
		Id: 123,
	}
	err := DeleteService(db, s0)
	require.Contains(t, err.Error(), "no rows in result")

	// add first machine, should be no error
	m := &Machine{
		Id: 0,
		Address: "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.Id)

	// add service, no error expected
	s := &Service{
		Id: 0,
		MachineID: m.Id,
		Type: "kea",
		CtrlPort: 1234,
		Active: true,
	}
	err = AddService(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.Id)

	// delete added service
	err = DeleteService(db, s)
	require.NoError(t, err)
}

func TestGetServicesByMachine(t *testing.T) {
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

	// there should be no services yet
	services, err := GetServicesByMachine(db, m.Id)
	require.Len(t, services, 0)
	require.NoError(t, err)

	// add service, no error expected
	s := &Service{
		Id: 0,
		MachineID: m.Id,
		Type: "kea",
		CtrlPort: 1234,
		Active: true,
	}
	err = AddService(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.Id)

	// get services of given machine
	services, err = GetServicesByMachine(db, m.Id)
	require.Len(t, services, 1)
	require.NoError(t, err)
}

func TestGetServiceById(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing service
	service, err := GetServiceById(db, 321)
	require.NoError(t, err)
	require.Nil(t, service)

	// add first machine, should be no error
	m := &Machine{
		Id: 0,
		Address: "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.Id)

	// add service, no error expected
	s := &Service{
		Id: 0,
		MachineID: m.Id,
		Type: "kea",
		CtrlPort: 1234,
		Active: true,
	}
	err = AddService(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.Id)

	// get service by id
	service, err = GetServiceById(db, s.Id)
	require.NoError(t, err)
	require.NotNil(t, service)
	require.Equal(t, s.Id, service.Id)
}


func TestGetServicesByPage(t *testing.T) {
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

	// add kea service, no error expected
	sKea := &Service{
		Id: 0,
		MachineID: m.Id,
		Type: "kea",
		CtrlPort: 1234,
		Active: true,
	}
	err = AddService(db, sKea)
	require.NoError(t, err)
	require.NotEqual(t, 0, sKea.Id)

	// add bind service, no error expected
	sBind := &Service{
		Id: 0,
		MachineID: m.Id,
		Type: "bind",
		CtrlPort: 4321,
		Active: true,
	}
	err = AddService(db, sBind)
	require.NoError(t, err)
	require.NotEqual(t, 0, sBind.Id)

	// get all services
	services, total, err := GetServicesByPage(db, 0, 10, "", "")
	require.NoError(t, err)
	require.Len(t, services, 2)
	require.Equal(t, int64(2), total)

	// get kea services
	services, total, err = GetServicesByPage(db, 0, 10, "", "kea")
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, "kea", services[0].Type)

	// get bind services
	services, total, err = GetServicesByPage(db, 0, 10, "", "bind")
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, "bind", services[0].Type)
}
