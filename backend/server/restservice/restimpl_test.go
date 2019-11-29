package restservice

import (
	//"log"
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/general"
	"isc.org/stork/server/gen/restapi/operations/services"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/database/model"
)

type FakeAgents struct {
}

func (fa *FakeAgents) Shutdown() {}
func (fa *FakeAgents) GetConnectedAgent(address string) (*agentcomm.Agent, error) {
	return nil, nil
}
func (fa *FakeAgents) GetState(ctx context.Context, address string, agentPort int64) (*agentcomm.State, error) {
	state := agentcomm.State{
		Cpus: 1,
		Memory: 4,
	}
	return &state, nil
}


func TestGetVersion(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	params := general.GetVersionParams{}

	rsp := rapi.GetVersion(ctx, params)
	p := rsp.(*general.GetVersionOK).Payload
	require.Equal(t, "unstable", *p.Type)
	require.Regexp(t, `^\d+.\d+.\d+$`, *p.Version)
}

func getStatusCode(rsp interface{}) int {
	code := int(reflect.ValueOf(rsp).FieldByName("_statusCode").Int())
	return code
}

func TestGetMachineState(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	// get state of non-exisiting machine
	params := services.GetMachineStateParams{
		ID: 123,
	}
	rsp := rapi.GetMachineState(ctx, params)
	require.IsType(t, &services.GetMachineStateDefault{}, rsp)
	defaultRsp := rsp.(*services.GetMachineStateDefault)
	require.Equal(t, 404, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find machine with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address: "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// get added machine
	params = services.GetMachineStateParams{
		ID: m.Id,
	}
	rsp = rapi.GetMachineState(ctx, params)
	require.IsType(t, &services.GetMachineStateOK{}, rsp)
	okRsp := rsp.(*services.GetMachineStateOK)
	require.Equal(t, "localhost", *okRsp.Payload.Address)
	require.Equal(t, int64(8080), okRsp.Payload.AgentPort)
	require.Less(t, int64(0), okRsp.Payload.Memory)
	require.Less(t, int64(0), okRsp.Payload.Cpus)
	require.LessOrEqual(t, int64(0), okRsp.Payload.Uptime)
}

func TestCreateMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	// bad host
	addr := "//a/"
	params := services.CreateMachineParams{
		Machine: &models.Machine{
			Address: &addr,
			AgentPort: 8080,
		},
	}
	rsp := rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp := rsp.(*services.CreateMachineDefault)
	require.Equal(t, 400, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot parse address", *defaultRsp.Payload.Message)

	// bad port
	addr = "1.2.3.4"
	params = services.CreateMachineParams{
		Machine: &models.Machine{
			Address: &addr,
			AgentPort: 0,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, 400, getStatusCode(*defaultRsp))
	require.Equal(t, "bad port", *defaultRsp.Payload.Message)

	// all ok
	addr = "1.2.3.4"
	params = services.CreateMachineParams{
		Machine: &models.Machine{
			Address: &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineOK{}, rsp)
	okRsp := rsp.(*services.CreateMachineOK)
	require.Equal(t, addr, *okRsp.Payload.Address)
	require.Equal(t, int64(8080), okRsp.Payload.AgentPort)
	require.Less(t, int64(0), okRsp.Payload.Memory)
	require.Less(t, int64(0), okRsp.Payload.Cpus)
	require.LessOrEqual(t, int64(0), okRsp.Payload.Uptime)
}

func TestGetMachines(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	var start, limit int64 = 0, 10
	params := services.GetMachinesParams{
		Start: &start,
		Limit: &limit,
	}

	rsp := rapi.GetMachines(ctx, params)
	ms := rsp.(*services.GetMachinesOK).Payload
	require.Equal(t, ms.Total, int64(0))
	//require.Greater(t, ms.Items, )
}

func TestGetMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing machine
	params := services.GetMachineParams{
		ID: 123,
	}
	rsp := rapi.GetMachine(ctx, params)
	require.IsType(t, &services.GetMachineDefault{}, rsp)
	defaultRsp := rsp.(*services.GetMachineDefault)
	require.Equal(t, 404, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find machine with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address: "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// get added machine
	params = services.GetMachineParams{
		ID: m.Id,
	}
	rsp = rapi.GetMachine(ctx, params)
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp := rsp.(*services.GetMachineOK)
	require.Equal(t, m.Id, okRsp.Payload.ID)

	// add machine 2
	m2 := &dbmodel.Machine{
		Address: "localhost",
		AgentPort: 8082,
	}
	err = dbmodel.AddMachine(db, m2)
	require.NoError(t, err)

	// add service to machine 2
	s := &dbmodel.Service{
		Id: 0,
		MachineID: m2.Id,
		Type: "kea",
		CtrlPort: 1234,
		Active: true,
	}
	err = dbmodel.AddService(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.Id)

	// get added machine 2 with kea service
	params = services.GetMachineParams{
		ID: m2.Id,
	}
	rsp = rapi.GetMachine(ctx, params)
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp = rsp.(*services.GetMachineOK)
	require.Equal(t, m2.Id, okRsp.Payload.ID)
	require.Len(t, okRsp.Payload.Services, 1)
	require.Equal(t, s.Id, okRsp.Payload.Services[0].ID)

}

func TestUpdateMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	// update non-existing machine
	addr := "localhost"
	params := services.UpdateMachineParams{
		ID: 123,
		Machine: &models.Machine{
			Address: &addr,
			AgentPort: 8080,
		},
	}
	rsp := rapi.UpdateMachine(ctx, params)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	defaultRsp := rsp.(*services.UpdateMachineDefault)
	require.Equal(t, 404, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find machine with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address: "localhost",
		AgentPort: 1010,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// update added machine - all ok
	params = services.UpdateMachineParams{
		ID: m.Id,
		Machine: &models.Machine{
			Address: &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.UpdateMachine(ctx, params)
	okRsp := rsp.(*services.UpdateMachineOK)
	require.Equal(t, m.Id, okRsp.Payload.ID)
	require.Equal(t, addr, *okRsp.Payload.Address)

	// add another machine
	m2 := &dbmodel.Machine{
		Address: "localhost",
		AgentPort: 2020,
	}
	err = dbmodel.AddMachine(db, m2)
	require.NoError(t, err)

	// update second machine to have the same address - should raise error due to duplicatation
	params = services.UpdateMachineParams{
		ID: m2.Id,
		Machine: &models.Machine{
			Address: &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.UpdateMachine(ctx, params)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.UpdateMachineDefault)
	require.Equal(t, 400, getStatusCode(*defaultRsp))
	require.Equal(t, "machine with address localhost:8080 already exists", *defaultRsp.Payload.Message)

	// update second machine with bad address
	addr = "aaa:"
	params = services.UpdateMachineParams{
		ID: m2.Id,
		Machine: &models.Machine{
			Address: &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.UpdateMachine(ctx, params)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.UpdateMachineDefault)
	require.Equal(t, 400, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot parse address", *defaultRsp.Payload.Message)
}

func TestDeleteMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	// delete non-existing machine
	params := services.DeleteMachineParams{
		ID: 123,
	}
	rsp := rapi.DeleteMachine(ctx, params)
	require.IsType(t, &services.DeleteMachineOK{}, rsp)

	// add machine
	m := &dbmodel.Machine{
		Address: "localhost:1010",
		AgentPort: 1010,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// get added machine
	params2 := services.GetMachineParams{
		ID: m.Id,
	}
	rsp = rapi.GetMachine(ctx, params2)
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp := rsp.(*services.GetMachineOK)
	require.Equal(t, m.Id, okRsp.Payload.ID)

	// delete added machine
	params = services.DeleteMachineParams{
		ID: m.Id,
	}
	rsp = rapi.DeleteMachine(ctx, params)
	require.IsType(t, &services.DeleteMachineOK{}, rsp)

	// get deleted machine - should return not found
	params2 = services.GetMachineParams{
		ID: m.Id,
	}
	rsp = rapi.GetMachine(ctx, params2)
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp = rsp.(*services.GetMachineOK)
	require.Equal(t, m.Id, okRsp.Payload.ID)
}

func TestGetService(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing service
	params := services.GetServiceParams{
		ID: 123,
	}
	rsp := rapi.GetService(ctx, params)
	require.IsType(t, &services.GetServiceDefault{}, rsp)
	defaultRsp := rsp.(*services.GetServiceDefault)
	require.Equal(t, 404, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find service with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address: "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add service to machine
	s := &dbmodel.Service{
		Id: 0,
		MachineID: m.Id,
		Type: "kea",
		CtrlPort: 1234,
		Active: true,
	}
	err = dbmodel.AddService(db, s)
	require.NoError(t, err)

	// get added service
	params = services.GetServiceParams{
		ID: s.Id,
	}
	rsp = rapi.GetService(ctx, params)
	require.IsType(t, &services.GetServiceOK{}, rsp)
	okRsp := rsp.(*services.GetServiceOK)
	require.Equal(t, s.Id, okRsp.Payload.ID)
}

func TestRestGetService(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing service
	params := services.GetServiceParams{
		ID: 123,
	}
	rsp := rapi.GetService(ctx, params)
	require.IsType(t, &services.GetServiceDefault{}, rsp)
	defaultRsp := rsp.(*services.GetServiceDefault)
	require.Equal(t, 404, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find service with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address: "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add service to machine
	s := &dbmodel.Service{
		Id: 0,
		MachineID: m.Id,
		Type: "kea",
		CtrlPort: 1234,
		Active: true,
	}
	err = dbmodel.AddService(db, s)
	require.NoError(t, err)

	// get added service
	params = services.GetServiceParams{
		ID: s.Id,
	}
	rsp = rapi.GetService(ctx, params)
	require.IsType(t, &services.GetServiceOK{}, rsp)
	okRsp := rsp.(*services.GetServiceOK)
	require.Equal(t, s.Id, okRsp.Payload.ID)
}

func TestRestGetServices(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestApiSettings{}
	fa := FakeAgents{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, &fa)
	require.NoError(t, err)
	ctx := context.Background()

	// get empty list of service
	params := services.GetServicesParams{}
	rsp := rapi.GetServices(ctx, params)
	require.IsType(t, &services.GetServicesOK{}, rsp)
	okRsp := rsp.(*services.GetServicesOK)
	require.Equal(t, int64(0), okRsp.Payload.Total)

	// add machine
	m := &dbmodel.Machine{
		Address: "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add service kea to machine
	s1 := &dbmodel.Service{
		Id: 0,
		MachineID: m.Id,
		Type: "kea",
		CtrlPort: 1234,
		Active: true,
	}
	err = dbmodel.AddService(db, s1)
	require.NoError(t, err)

	// add service bind to machine
	s2 := &dbmodel.Service{
		Id: 0,
		MachineID: m.Id,
		Type: "bind",
		CtrlPort: 4321,
		Active: true,
	}
	err = dbmodel.AddService(db, s2)
	require.NoError(t, err)

	// get added service
	params = services.GetServicesParams{
	}
	rsp = rapi.GetServices(ctx, params)
	require.IsType(t, &services.GetServicesOK{}, rsp)
	okRsp = rsp.(*services.GetServicesOK)
	require.Equal(t, int64(2), okRsp.Payload.Total)
}
