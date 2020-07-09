package restservice

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	"isc.org/stork/server/apps/kea"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	"isc.org/stork/server/gen/restapi/operations/general"
	"isc.org/stork/server/gen/restapi/operations/services"
	storktest "isc.org/stork/server/test"
	storkutil "isc.org/stork/util"
)

// makeAccessPoint is an utility to make an array of one access point.
func makeAccessPoint(tp, address, key string, port int64) (ap []agentcomm.AccessPoint) {
	return append(ap, agentcomm.AccessPoint{
		Type:    tp,
		Address: address,
		Port:    port,
		Key:     key,
	})
}

func TestGetVersion(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	params := general.GetVersionParams{}

	rsp := rapi.GetVersion(ctx, params)
	p := rsp.(*general.GetVersionOK).Payload
	require.Regexp(t, `^\d+.\d+.\d+$`, *p.Version)
	require.Equal(t, "unset", *p.Date)
}

func TestGetMachineStateOnly(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// get state of non-exisiting machine
	params := services.GetMachineStateParams{
		ID: 123,
	}
	rsp := rapi.GetMachineState(ctx, params)
	require.IsType(t, &services.GetMachineStateDefault{}, rsp)
	defaultRsp := rsp.(*services.GetMachineStateDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find machine with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// get added machine
	params = services.GetMachineStateParams{
		ID: m.ID,
	}
	rsp = rapi.GetMachineState(ctx, params)
	require.IsType(t, &services.GetMachineStateOK{}, rsp)
	okRsp := rsp.(*services.GetMachineStateOK)
	require.Equal(t, "localhost", *okRsp.Payload.Address)
	require.EqualValues(t, 8080, okRsp.Payload.AgentPort)
	require.Less(t, int64(0), okRsp.Payload.Memory)
	require.Less(t, int64(0), okRsp.Payload.Cpus)
	require.LessOrEqual(t, int64(0), okRsp.Payload.Uptime)
}

func mockGetAppsState(callNo int, cmdResponses []interface{}) {
	switch callNo {
	case 0:
		list1 := cmdResponses[0].(*[]kea.VersionGetResponse)
		*list1 = []kea.VersionGetResponse{
			{
				KeaResponseHeader: agentcomm.KeaResponseHeader{
					Result: 0,
					Daemon: "ca",
				},
				Arguments: &kea.VersionGetRespArgs{
					Extended: "Extended version",
				},
			},
		}
		list2 := cmdResponses[1].(*[]agentcomm.KeaResponse)
		*list2 = []agentcomm.KeaResponse{
			{
				KeaResponseHeader: agentcomm.KeaResponseHeader{
					Result: 0,
					Daemon: "ca",
				},
				Arguments: &map[string]interface{}{
					"Control-agent": map[string]interface{}{
						"control-sockets": map[string]interface{}{
							"dhcp4": map[string]interface{}{
								"socket-name": "aaa",
								"socket-type": "unix",
							},
						},
					},
				},
			},
		}
	case 1:
		// version-get response
		list1 := cmdResponses[0].(*[]kea.VersionGetResponse)
		*list1 = []kea.VersionGetResponse{
			{
				KeaResponseHeader: agentcomm.KeaResponseHeader{
					Result: 0,
					Daemon: "dhcp4",
				},
				Arguments: &kea.VersionGetRespArgs{
					Extended: "Extended version",
				},
			},
		}
		// status-get response
		list2 := cmdResponses[1].(*[]kea.StatusGetResponse)
		*list2 = []kea.StatusGetResponse{
			{
				KeaResponseHeader: agentcomm.KeaResponseHeader{
					Result: 0,
					Daemon: "dhcp4",
				},
				Arguments: &kea.StatusGetRespArgs{
					Pid: 123,
				},
			},
		}
		// config-get response
		list3 := cmdResponses[2].(*[]agentcomm.KeaResponse)
		*list3 = []agentcomm.KeaResponse{
			{
				KeaResponseHeader: agentcomm.KeaResponseHeader{
					Result: 0,
					Daemon: "dhcp4",
				},
				Arguments: &map[string]interface{}{
					"Dhcp4": map[string]interface{}{
						"hooks-libraries": []interface{}{
							map[string]interface{}{
								"library": "hook_abc.so",
							},
							map[string]interface{}{
								"library": "hook_def.so",
							},
						},
					},
				},
			},
		}
	}
}

func TestGetMachineAndAppsState(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(mockGetAppsState, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add kea app
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "1.2.3.4", "", 123)
	keaApp := &dbmodel.App{
		MachineID:    m.ID,
		Machine:      m,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: keaPoints,
	}
	_, err = dbmodel.AddApp(db, keaApp)
	require.NoError(t, err)
	m.Apps = append(m.Apps, keaApp)

	// add BIND 9 app
	var bind9Points []*dbmodel.AccessPoint
	bind9Points = dbmodel.AppendAccessPoint(bind9Points, dbmodel.AccessPointControl, "1.2.3.4", "abcd", 124)
	bind9App := &dbmodel.App{
		MachineID:    m.ID,
		Machine:      m,
		Type:         dbmodel.AppTypeBind9,
		AccessPoints: bind9Points,
	}
	_, err = dbmodel.AddApp(db, bind9App)
	require.NoError(t, err)
	m.Apps = append(m.Apps, bind9App)

	fa.MachineState = &agentcomm.State{
		Apps: []*agentcomm.App{
			{
				Type:         dbmodel.AppTypeKea,
				AccessPoints: makeAccessPoint(dbmodel.AccessPointControl, "1.2.3.4", "", 123),
			},
			{
				Type:         dbmodel.AppTypeBind9,
				AccessPoints: makeAccessPoint(dbmodel.AccessPointControl, "1.2.3.4", "abcd", 124),
			},
		},
	}

	// get added machine
	params := services.GetMachineStateParams{
		ID: m.ID,
	}
	rsp := rapi.GetMachineState(ctx, params)
	require.IsType(t, &services.GetMachineStateOK{}, rsp)
	okRsp := rsp.(*services.GetMachineStateOK)
	require.Len(t, okRsp.Payload.Apps, 2)
	require.Equal(t, dbmodel.AppTypeKea, okRsp.Payload.Apps[0].Type)
	require.Equal(t, dbmodel.AppTypeBind9, okRsp.Payload.Apps[1].Type)
}

func TestCreateMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// empty request, variant 1 - should raise an error
	params := services.CreateMachineParams{}
	rsp := rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp := rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "missing parameters", *defaultRsp.Payload.Message)

	// empty request, variant 2 - should raise an error
	params = services.CreateMachineParams{
		Machine: &models.Machine{},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "missing parameters", *defaultRsp.Payload.Message)

	// bad host
	addr := "//a/"
	params = services.CreateMachineParams{
		Machine: &models.Machine{
			Address:   &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot parse address", *defaultRsp.Payload.Message)

	// bad port
	addr = "1.2.3.4"
	params = services.CreateMachineParams{
		Machine: &models.Machine{
			Address:   &addr,
			AgentPort: 0,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "bad port", *defaultRsp.Payload.Message)

	// all ok
	addr = "1.2.3.4"
	params = services.CreateMachineParams{
		Machine: &models.Machine{
			Address:   &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineOK{}, rsp)
	okRsp := rsp.(*services.CreateMachineOK)
	require.Equal(t, addr, *okRsp.Payload.Address)
	require.EqualValues(t, 8080, okRsp.Payload.AgentPort)
	require.Less(t, int64(0), okRsp.Payload.Memory)
	require.Less(t, int64(0), okRsp.Payload.Cpus)
	require.LessOrEqual(t, int64(0), okRsp.Payload.Uptime)
}

func TestGetMachines(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	var start, limit int64 = 0, 10
	params := services.GetMachinesParams{
		Start: &start,
		Limit: &limit,
	}

	rsp := rapi.GetMachines(ctx, params)
	ms := rsp.(*services.GetMachinesOK).Payload
	require.EqualValues(t, ms.Total, 0)
	//require.Greater(t, ms.Items, )
}

func TestGetMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing machine
	params := services.GetMachineParams{
		ID: 123,
	}
	rsp := rapi.GetMachine(ctx, params)
	require.IsType(t, &services.GetMachineDefault{}, rsp)
	defaultRsp := rsp.(*services.GetMachineDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find machine with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// get added machine
	params = services.GetMachineParams{
		ID: m.ID,
	}
	rsp = rapi.GetMachine(ctx, params)
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp := rsp.(*services.GetMachineOK)
	require.Equal(t, m.ID, okRsp.Payload.ID)

	// add machine 2
	m2 := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8082,
	}
	err = dbmodel.AddMachine(db, m2)
	require.NoError(t, err)

	// add app to machine 2
	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)
	s := &dbmodel.App{
		ID:           0,
		MachineID:    m2.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons:      []*dbmodel.Daemon{},
	}
	_, err = dbmodel.AddApp(db, s)
	require.NoError(t, err)
	require.NotEqual(t, 0, s.ID)

	// get added machine 2 with kea app
	params = services.GetMachineParams{
		ID: m2.ID,
	}
	rsp = rapi.GetMachine(ctx, params)
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp = rsp.(*services.GetMachineOK)
	require.Equal(t, m2.ID, okRsp.Payload.ID)
	require.Len(t, okRsp.Payload.Apps, 1)
	require.Equal(t, s.ID, okRsp.Payload.Apps[0].ID)
}

func TestUpdateMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// empty request, variant 1 - should raise an error
	params := services.UpdateMachineParams{}
	rsp := rapi.UpdateMachine(ctx, params)
	defaultRsp := rsp.(*services.UpdateMachineDefault)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "missing parameters", *defaultRsp.Payload.Message)

	// empty request, variant 2 - should raise an error
	params = services.UpdateMachineParams{
		Machine: &models.Machine{},
	}
	rsp = rapi.UpdateMachine(ctx, params)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.UpdateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "missing parameters", *defaultRsp.Payload.Message)

	// update non-existing machine
	addr := "localhost"
	params = services.UpdateMachineParams{
		ID: 123,
		Machine: &models.Machine{
			Address:   &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.UpdateMachine(ctx, params)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.UpdateMachineDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find machine with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 1010,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// update added machine - all ok
	params = services.UpdateMachineParams{
		ID: m.ID,
		Machine: &models.Machine{
			Address:   &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.UpdateMachine(ctx, params)
	okRsp := rsp.(*services.UpdateMachineOK)
	require.Equal(t, m.ID, okRsp.Payload.ID)
	require.Equal(t, addr, *okRsp.Payload.Address)

	// add another machine
	m2 := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 2020,
	}
	err = dbmodel.AddMachine(db, m2)
	require.NoError(t, err)

	// update second machine to have the same address - should raise error due to duplicatation
	params = services.UpdateMachineParams{
		ID: m2.ID,
		Machine: &models.Machine{
			Address:   &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.UpdateMachine(ctx, params)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.UpdateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "machine with address localhost:8080 already exists", *defaultRsp.Payload.Message)

	// update second machine with bad address
	addr = "aaa:"
	params = services.UpdateMachineParams{
		ID: m2.ID,
		Machine: &models.Machine{
			Address:   &addr,
			AgentPort: 8080,
		},
	}
	rsp = rapi.UpdateMachine(ctx, params)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.UpdateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot parse address", *defaultRsp.Payload.Message)
}

func TestDeleteMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
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
		Address:   "localhost:1010",
		AgentPort: 1010,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// get added machine
	params2 := services.GetMachineParams{
		ID: m.ID,
	}
	rsp = rapi.GetMachine(ctx, params2)
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp := rsp.(*services.GetMachineOK)
	require.Equal(t, m.ID, okRsp.Payload.ID)

	// delete added machine
	params = services.DeleteMachineParams{
		ID: m.ID,
	}
	rsp = rapi.DeleteMachine(ctx, params)
	require.IsType(t, &services.DeleteMachineOK{}, rsp)

	// get deleted machine - should return not found
	params2 = services.GetMachineParams{
		ID: m.ID,
	}
	rsp = rapi.GetMachine(ctx, params2)
	require.IsType(t, &services.GetMachineDefault{}, rsp)
	defaultRsp := rsp.(*services.GetMachineDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
}

func TestGetApp(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing app
	params := services.GetAppParams{
		ID: 123,
	}
	rsp := rapi.GetApp(ctx, params)
	require.IsType(t, &services.GetAppDefault{}, rsp)
	defaultRsp := rsp.(*services.GetAppDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find app with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app to machine
	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)
	s := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons:      []*dbmodel.Daemon{},
	}
	_, err = dbmodel.AddApp(db, s)
	require.NoError(t, err)

	// get added app
	params = services.GetAppParams{
		ID: s.ID,
	}
	rsp = rapi.GetApp(ctx, params)
	require.IsType(t, &services.GetAppOK{}, rsp)
	okRsp := rsp.(*services.GetAppOK)
	require.Equal(t, s.ID, okRsp.Payload.ID)
}

func TestRestGetApp(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing app
	params := services.GetAppParams{
		ID: 123,
	}
	rsp := rapi.GetApp(ctx, params)
	require.IsType(t, &services.GetAppDefault{}, rsp)
	defaultRsp := rsp.(*services.GetAppDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find app with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add kea app to machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "", "", 1234)
	keaApp := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons:      []*dbmodel.Daemon{},
	}
	_, err = dbmodel.AddApp(db, keaApp)
	require.NoError(t, err)

	// get added kea app
	params = services.GetAppParams{
		ID: keaApp.ID,
	}
	rsp = rapi.GetApp(ctx, params)
	require.IsType(t, &services.GetAppOK{}, rsp)
	okRsp := rsp.(*services.GetAppOK)
	require.Equal(t, keaApp.ID, okRsp.Payload.ID)

	// add BIND 9 app to machine
	var bind9Points []*dbmodel.AccessPoint
	bind9Points = dbmodel.AppendAccessPoint(bind9Points, dbmodel.AccessPointControl, "", "abcd", 953)
	bind9App := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeBind9,
		Active:       true,
		AccessPoints: bind9Points,
		Daemons: []*dbmodel.Daemon{
			{
				Bind9Daemon: &dbmodel.Bind9Daemon{},
			},
		},
	}
	_, err = dbmodel.AddApp(db, bind9App)
	require.NoError(t, err)

	// get added BIND 9 app
	params = services.GetAppParams{
		ID: bind9App.ID,
	}
	rsp = rapi.GetApp(ctx, params)
	require.IsType(t, &services.GetAppOK{}, rsp)
	okRsp = rsp.(*services.GetAppOK)
	require.Equal(t, bind9App.ID, okRsp.Payload.ID)
}

func TestRestGetApps(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// get empty list of app
	params := services.GetAppsParams{}
	rsp := rapi.GetApps(ctx, params)
	require.IsType(t, &services.GetAppsOK{}, rsp)
	okRsp := rsp.(*services.GetAppsOK)
	require.EqualValues(t, 0, okRsp.Payload.Total)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app kea to machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "", "", 1234)
	s1 := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:      "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{},
				LogTargets: []*dbmodel.LogTarget{
					{
						Name:     "kea-dhcp4",
						Severity: "DEBUG",
						Output:   "/tmp/log",
					},
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, s1)
	require.NoError(t, err)

	// add app BIND 9 to machine
	var bind9Points []*dbmodel.AccessPoint
	bind9Points = dbmodel.AppendAccessPoint(bind9Points, dbmodel.AccessPointControl, "", "abcd", 4321)
	s2 := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeBind9,
		Active:       true,
		AccessPoints: bind9Points,
		Daemons: []*dbmodel.Daemon{
			{
				Bind9Daemon: &dbmodel.Bind9Daemon{},
			},
		},
	}
	_, err = dbmodel.AddApp(db, s2)
	require.NoError(t, err)

	// get added apps
	params = services.GetAppsParams{}
	rsp = rapi.GetApps(ctx, params)
	require.IsType(t, &services.GetAppsOK{}, rsp)
	okRsp = rsp.(*services.GetAppsOK)
	require.EqualValues(t, 2, okRsp.Payload.Total)

	// Verify that the communication error counters are returned. See fake_agents.go
	// to see where those counters are set.
	require.Len(t, okRsp.Payload.Items, 2)
	for _, app := range okRsp.Payload.Items {
		if app.Type == dbmodel.AppTypeKea {
			appKea := app.Details.AppKea
			require.Len(t, appKea.Daemons, 1)
			daemon := appKea.Daemons[0]
			require.EqualValues(t, 1, daemon.AgentCommErrors)
			require.EqualValues(t, 2, daemon.CaCommErrors)
			require.EqualValues(t, 5, daemon.DaemonCommErrors)
			require.Len(t, daemon.LogTargets, 1)
			require.Equal(t, "kea-dhcp4", daemon.LogTargets[0].Name)
			require.Equal(t, "debug", daemon.LogTargets[0].Severity)
			require.Equal(t, "/tmp/log", daemon.LogTargets[0].Output)
		} else if app.Type == dbmodel.AppTypeBind9 {
			appBind9 := app.Details.AppBind9
			daemon := appBind9.Daemon
			require.EqualValues(t, 1, daemon.AgentCommErrors)
			require.EqualValues(t, 2, daemon.RndcCommErrors)
			require.EqualValues(t, 3, daemon.StatsCommErrors)
		}
	}
}

// Test that status of two HA services for a Kea application is parsed
// correctly.
func TestRestGetAppServicesStatus(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	// Configure the fake control agents to mimic returning a status of
	// two HA services for Kea.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea application to the machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "127.0.0.1", "", 1234)
	keaApp := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	_, err = dbmodel.AddApp(db, keaApp)
	require.NoError(t, err)
	require.NotZero(t, keaApp.ID)

	exampleTime := storkutil.UTCNow().Add(-5 * time.Second)
	commInterrupted := []bool{true, true}
	keaServices := []dbmodel.Service{
		{
			BaseService: dbmodel.BaseService{
				ServiceType: "ha_dhcp",
			},
			HAService: &dbmodel.BaseHAService{
				HAType:                      "dhcp4",
				HAMode:                      "load-balancing",
				PrimaryID:                   keaApp.ID,
				PrimaryStatusCollectedAt:    exampleTime,
				SecondaryStatusCollectedAt:  exampleTime,
				PrimaryLastState:            "load-balancing",
				SecondaryLastState:          "load-balancing",
				PrimaryLastScopes:           []string{"server1"},
				SecondaryLastScopes:         []string{"server2"},
				PrimaryLastFailoverAt:       exampleTime,
				SecondaryLastFailoverAt:     exampleTime,
				PrimaryCommInterrupted:      &commInterrupted[0],
				SecondaryCommInterrupted:    &commInterrupted[1],
				PrimaryConnectingClients:    7,
				SecondaryConnectingClients:  9,
				PrimaryUnackedClients:       3,
				SecondaryUnackedClients:     4,
				PrimaryUnackedClientsLeft:   8,
				SecondaryUnackedClientsLeft: 9,
				PrimaryAnalyzedPackets:      10,
				SecondaryAnalyzedPackets:    11,
			},
		},
		{
			BaseService: dbmodel.BaseService{
				ServiceType: "ha_dhcp",
			},
			HAService: &dbmodel.BaseHAService{
				HAType:                      "dhcp6",
				HAMode:                      "hot-standby",
				PrimaryID:                   keaApp.ID,
				PrimaryStatusCollectedAt:    exampleTime,
				SecondaryStatusCollectedAt:  exampleTime,
				PrimaryLastState:            "hot-standby",
				SecondaryLastState:          "waiting",
				PrimaryLastScopes:           []string{"server1"},
				SecondaryLastScopes:         []string{},
				PrimaryLastFailoverAt:       exampleTime,
				SecondaryLastFailoverAt:     exampleTime,
				PrimaryCommInterrupted:      &commInterrupted[0],
				SecondaryCommInterrupted:    nil,
				PrimaryConnectingClients:    5,
				SecondaryConnectingClients:  0,
				PrimaryUnackedClients:       2,
				SecondaryUnackedClients:     0,
				PrimaryUnackedClientsLeft:   9,
				SecondaryUnackedClientsLeft: 0,
				PrimaryAnalyzedPackets:      123,
				SecondaryAnalyzedPackets:    0,
			},
		},
	}

	// Add the services and associate them with the app.
	for i := range keaServices {
		err = dbmodel.AddService(db, &keaServices[i])
		require.NoError(t, err)
		err = dbmodel.AddDaemonToService(db, keaServices[i].ID, keaApp.Daemons[0])
		require.NoError(t, err)
	}

	params := services.GetAppServicesStatusParams{
		ID: keaApp.ID,
	}
	rsp := rapi.GetAppServicesStatus(ctx, params)

	// Make sure that the response is ok.
	require.IsType(t, &services.GetAppServicesStatusOK{}, rsp)
	okRsp := rsp.(*services.GetAppServicesStatusOK)
	require.NotNil(t, okRsp.Payload.Items)

	// There should be two structures returned, one with a status of
	// the DHCPv4 server and one with the status of the DHCPv6 server.
	require.Len(t, okRsp.Payload.Items, 2)

	statusList := okRsp.Payload.Items

	// Validate the status of the DHCPv4 pair.
	status := statusList[0].Status.KeaStatus
	require.NotNil(t, status.HaServers)

	haStatus := status.HaServers
	require.NotNil(t, haStatus.PrimaryServer)
	require.NotNil(t, haStatus.SecondaryServer)

	require.EqualValues(t, keaApp.Daemons[0].ID, haStatus.PrimaryServer.ID)
	require.Equal(t, "primary", haStatus.PrimaryServer.Role)
	require.Len(t, haStatus.PrimaryServer.Scopes, 1)
	require.Contains(t, haStatus.PrimaryServer.Scopes, "server1")
	require.Equal(t, "load-balancing", haStatus.PrimaryServer.State)
	require.GreaterOrEqual(t, haStatus.PrimaryServer.Age, int64(5))
	require.Equal(t, "127.0.0.1", haStatus.PrimaryServer.ControlAddress)
	require.EqualValues(t, keaApp.ID, haStatus.PrimaryServer.AppID)
	require.NotEmpty(t, haStatus.PrimaryServer.StatusTime.String())
	require.EqualValues(t, 1, haStatus.PrimaryServer.CommInterrupted)
	require.EqualValues(t, 7, haStatus.PrimaryServer.ConnectingClients)
	require.EqualValues(t, 3, haStatus.PrimaryServer.UnackedClients)
	require.EqualValues(t, 8, haStatus.PrimaryServer.UnackedClientsLeft)
	require.EqualValues(t, 10, haStatus.PrimaryServer.AnalyzedPackets)

	require.Equal(t, "secondary", haStatus.SecondaryServer.Role)
	require.Len(t, haStatus.SecondaryServer.Scopes, 1)
	require.Contains(t, haStatus.SecondaryServer.Scopes, "server2")
	require.Equal(t, "load-balancing", haStatus.SecondaryServer.State)
	require.GreaterOrEqual(t, haStatus.SecondaryServer.Age, int64(5))
	require.False(t, haStatus.SecondaryServer.InTouch)
	require.Empty(t, haStatus.SecondaryServer.ControlAddress)
	require.NotEmpty(t, haStatus.SecondaryServer.StatusTime.String())
	require.EqualValues(t, 1, haStatus.SecondaryServer.CommInterrupted)
	require.EqualValues(t, 9, haStatus.SecondaryServer.ConnectingClients)
	require.EqualValues(t, 4, haStatus.SecondaryServer.UnackedClients)
	require.EqualValues(t, 9, haStatus.SecondaryServer.UnackedClientsLeft)
	require.EqualValues(t, 11, haStatus.SecondaryServer.AnalyzedPackets)

	// Validate the status of the DHCPv6 pair.
	status = statusList[1].Status.KeaStatus
	require.NotNil(t, status.HaServers)

	haStatus = status.HaServers
	require.NotNil(t, haStatus.PrimaryServer)
	require.NotNil(t, haStatus.SecondaryServer)

	require.EqualValues(t, keaApp.Daemons[0].ID, haStatus.PrimaryServer.ID)
	require.Equal(t, "primary", haStatus.PrimaryServer.Role)
	require.Len(t, haStatus.PrimaryServer.Scopes, 1)
	require.Contains(t, haStatus.PrimaryServer.Scopes, "server1")
	require.Equal(t, "hot-standby", haStatus.PrimaryServer.State)
	require.GreaterOrEqual(t, haStatus.PrimaryServer.Age, int64(5))
	require.Equal(t, "127.0.0.1", haStatus.PrimaryServer.ControlAddress)
	require.EqualValues(t, 1, haStatus.PrimaryServer.CommInterrupted)
	require.EqualValues(t, 5, haStatus.PrimaryServer.ConnectingClients)
	require.EqualValues(t, 2, haStatus.PrimaryServer.UnackedClients)
	require.EqualValues(t, 9, haStatus.PrimaryServer.UnackedClientsLeft)
	require.EqualValues(t, 123, haStatus.PrimaryServer.AnalyzedPackets)

	require.Equal(t, "standby", haStatus.SecondaryServer.Role)
	require.Empty(t, haStatus.SecondaryServer.Scopes)
	require.Equal(t, "waiting", haStatus.SecondaryServer.State)
	require.GreaterOrEqual(t, haStatus.SecondaryServer.Age, int64(5))
	require.False(t, haStatus.SecondaryServer.InTouch)
	require.Empty(t, haStatus.SecondaryServer.ControlAddress)
	require.EqualValues(t, -1, haStatus.SecondaryServer.CommInterrupted)
	require.Zero(t, haStatus.SecondaryServer.ConnectingClients)
	require.Zero(t, haStatus.SecondaryServer.UnackedClients)
	require.Zero(t, haStatus.SecondaryServer.UnackedClientsLeft)
	require.Zero(t, haStatus.SecondaryServer.AnalyzedPackets)
}

// Test that status of a HA service providing passive-backup mode is
// parsed correctly.
func TestRestGetAppServicesStatusPassiveBackup(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	// Configure the fake control agents to mimic returning a status of
	// two HA services for Kea.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea application to the machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "127.0.0.1", "", 1234)
	keaApp := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	_, err = dbmodel.AddApp(db, keaApp)
	require.NoError(t, err)
	require.NotZero(t, keaApp.ID)

	exampleTime := storkutil.UTCNow().Add(-5 * time.Second)
	keaServices := []dbmodel.Service{
		{
			BaseService: dbmodel.BaseService{
				ServiceType: "ha_dhcp",
			},
			HAService: &dbmodel.BaseHAService{
				HAType:                   "dhcp4",
				HAMode:                   "passive-backup",
				PrimaryID:                keaApp.ID,
				PrimaryStatusCollectedAt: exampleTime,
				PrimaryLastState:         "passive-backup",
				PrimaryLastScopes:        []string{"server1"},
			},
		},
	}

	// Add the services and associate them with the app.
	for i := range keaServices {
		err = dbmodel.AddService(db, &keaServices[i])
		require.NoError(t, err)
		err = dbmodel.AddDaemonToService(db, keaServices[i].ID, keaApp.Daemons[0])
		require.NoError(t, err)
	}

	params := services.GetAppServicesStatusParams{
		ID: keaApp.ID,
	}
	rsp := rapi.GetAppServicesStatus(ctx, params)

	// Make sure that the response is ok.
	require.IsType(t, &services.GetAppServicesStatusOK{}, rsp)
	okRsp := rsp.(*services.GetAppServicesStatusOK)
	require.NotNil(t, okRsp.Payload.Items)

	// There should be one structure returned with a status of the
	// DHCPv4 server.
	require.Len(t, okRsp.Payload.Items, 1)

	statusList := okRsp.Payload.Items

	// Validate the status of the DHCPv4 pair.
	status := statusList[0].Status.KeaStatus
	require.NotNil(t, status.HaServers)

	haStatus := status.HaServers
	require.NotNil(t, haStatus.PrimaryServer)
	require.Nil(t, haStatus.SecondaryServer)

	require.EqualValues(t, keaApp.Daemons[0].ID, haStatus.PrimaryServer.ID)
	require.Equal(t, "primary", haStatus.PrimaryServer.Role)
	require.Len(t, haStatus.PrimaryServer.Scopes, 1)
	require.Contains(t, haStatus.PrimaryServer.Scopes, "server1")
	require.Equal(t, "passive-backup", haStatus.PrimaryServer.State)
	require.GreaterOrEqual(t, haStatus.PrimaryServer.Age, int64(5))
	require.Equal(t, "127.0.0.1", haStatus.PrimaryServer.ControlAddress)
	require.EqualValues(t, keaApp.ID, haStatus.PrimaryServer.AppID)
	require.NotEmpty(t, haStatus.PrimaryServer.StatusTime.String())
	require.EqualValues(t, -1, haStatus.PrimaryServer.CommInterrupted)
	require.Zero(t, haStatus.PrimaryServer.ConnectingClients)
	require.Zero(t, haStatus.PrimaryServer.UnackedClients)
	require.Zero(t, haStatus.PrimaryServer.UnackedClientsLeft)
	require.Zero(t, haStatus.PrimaryServer.AnalyzedPackets)
}

func TestRestGetAppsStats(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// get empty list of app
	params := services.GetAppsStatsParams{}
	rsp := rapi.GetAppsStats(ctx, params)
	require.IsType(t, &services.GetAppsStatsOK{}, rsp)
	okRsp := rsp.(*services.GetAppsStatsOK)
	require.EqualValues(t, 0, okRsp.Payload.KeaAppsTotal)
	require.EqualValues(t, 0, okRsp.Payload.KeaAppsNotOk)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app kea to machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "", "", 1234)
	s1 := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons:      []*dbmodel.Daemon{},
	}
	_, err = dbmodel.AddApp(db, s1)
	require.NoError(t, err)

	// add app bind9 to machine
	var bind9Points []*dbmodel.AccessPoint
	bind9Points = dbmodel.AppendAccessPoint(bind9Points, dbmodel.AccessPointControl, "", "abcd", 4321)
	s2 := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeBind9,
		Active:       false,
		AccessPoints: bind9Points,
		Daemons:      []*dbmodel.Daemon{},
	}
	_, err = dbmodel.AddApp(db, s2)
	require.NoError(t, err)

	// get added app
	params = services.GetAppsStatsParams{}
	rsp = rapi.GetAppsStats(ctx, params)
	require.IsType(t, &services.GetAppsStatsOK{}, rsp)
	okRsp = rsp.(*services.GetAppsStatsOK)
	require.EqualValues(t, 1, okRsp.Payload.KeaAppsTotal)
	require.EqualValues(t, 0, okRsp.Payload.KeaAppsNotOk)
	require.EqualValues(t, 1, okRsp.Payload.Bind9AppsTotal)
	require.EqualValues(t, 1, okRsp.Payload.Bind9AppsNotOk)
}

func TestGetDhcpOverview(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app kea to machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "localhost", "", 1234)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// get overview, generally it should be empty
	params := dhcp.GetDhcpOverviewParams{}
	rsp := rapi.GetDhcpOverview(ctx, params)
	require.IsType(t, &dhcp.GetDhcpOverviewOK{}, rsp)
	okRsp := rsp.(*dhcp.GetDhcpOverviewOK)
	require.Len(t, okRsp.Payload.Subnets4.Items, 0)
	require.Len(t, okRsp.Payload.Subnets6.Items, 0)
	require.Len(t, okRsp.Payload.SharedNetworks4.Items, 0)
	require.Len(t, okRsp.Payload.SharedNetworks6.Items, 0)
	require.Len(t, okRsp.Payload.DhcpDaemons, 1)

	require.EqualValues(t, 1, okRsp.Payload.DhcpDaemons[0].AgentCommErrors)
	require.EqualValues(t, 2, okRsp.Payload.DhcpDaemons[0].CaCommErrors)
	require.EqualValues(t, 5, okRsp.Payload.DhcpDaemons[0].DaemonCommErrors)
}

// Test updating daemon.
func TestUpdateDaemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	// Configure the fake control agents to mimic returning a status of
	// two HA services for Kea.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea application to the machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "127.0.0.1", "", 1234)
	keaApp := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	_, err = dbmodel.AddApp(db, keaApp)
	require.NoError(t, err)
	require.NotZero(t, keaApp.ID)
	require.NotZero(t, keaApp.Daemons[0].ID)

	// get added app
	getAppParams := services.GetAppParams{
		ID: keaApp.ID,
	}
	rsp := rapi.GetApp(ctx, getAppParams)
	require.IsType(t, &services.GetAppOK{}, rsp)
	okRsp := rsp.(*services.GetAppOK)
	require.Equal(t, keaApp.ID, okRsp.Payload.ID)
	require.True(t, okRsp.Payload.Details.AppKea.Daemons[0].Monitored) // now it is true

	// update daemon: change monitored to false
	params := services.UpdateDaemonParams{
		ID: keaApp.Daemons[0].ID,
		Daemon: services.UpdateDaemonBody{
			Monitored: false,
		},
	}
	rsp = rapi.UpdateDaemon(ctx, params)
	require.IsType(t, &services.UpdateDaemonOK{}, rsp)

	// get app with modified daemon
	rsp = rapi.GetApp(ctx, getAppParams)
	require.IsType(t, &services.GetAppOK{}, rsp)
	okRsp = rsp.(*services.GetAppOK)
	require.Equal(t, keaApp.ID, okRsp.Payload.ID)
	require.False(t, okRsp.Payload.Details.AppKea.Daemons[0].Monitored) // now it is false
}
