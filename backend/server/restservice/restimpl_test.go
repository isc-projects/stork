package restservice

import (
	//"log"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps/kea"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/general"
	"isc.org/stork/server/gen/restapi/operations/services"
	storktest "isc.org/stork/server/test"
)

func TestGetVersion(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
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
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
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
	require.Equal(t, int64(8080), okRsp.Payload.AgentPort)
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
		list2 := cmdResponses[1].(*[]kea.CAConfigGetResponse)
		*list2 = []kea.CAConfigGetResponse{
			{
				KeaResponseHeader: agentcomm.KeaResponseHeader{
					Result: 0,
					Daemon: "ca",
				},
				Arguments: &kea.CAConfigGetRespArgs{
					ControlAgent: &kea.ControlAgentData{
						ControlSockets: &kea.ControlSocketsData{
							Dhcp4: &kea.SocketData{
								SocketName: "aaaa",
								SocketType: "unix",
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
	fa := storktest.NewFakeAgents(mockGetAppsState)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
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
	keaApp := &dbmodel.App{
		MachineID:   m.ID,
		Machine:     m,
		Type:        "kea",
		CtrlAddress: "1.2.3.4",
		CtrlPort:    123,
		CtrlKey:     "",
	}
	err = dbmodel.AddApp(db, keaApp)
	require.NoError(t, err)
	m.Apps = append(m.Apps, keaApp)

	// add BIND 9 app
	bind9App := &dbmodel.App{
		MachineID:   m.ID,
		Machine:     m,
		Type:        "bind9",
		CtrlAddress: "1.2.3.4",
		CtrlPort:    124,
		CtrlKey:     "abcd",
	}
	err = dbmodel.AddApp(db, bind9App)
	require.NoError(t, err)
	m.Apps = append(m.Apps, bind9App)

	fa.MachineState = &agentcomm.State{
		Apps: []*agentcomm.App{
			{
				Type:        "kea",
				CtrlAddress: "1.2.3.4",
				CtrlPort:    123,
				CtrlKey:     "",
			},
			{
				Type:        "bind9",
				CtrlAddress: "1.2.3.4",
				CtrlPort:    124,
				CtrlKey:     "abcd",
			},
		},
	}

	fa.Bind9State = &agentcomm.Bind9State{
		Version: "1.2.3",
	}

	// get added machine
	params := services.GetMachineStateParams{
		ID: m.ID,
	}
	rsp := rapi.GetMachineState(ctx, params)
	require.IsType(t, &services.GetMachineStateOK{}, rsp)
	okRsp := rsp.(*services.GetMachineStateOK)
	require.Len(t, okRsp.Payload.Apps, 2)
	require.Equal(t, "kea", okRsp.Payload.Apps[0].Type)
}

func TestCreateMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// bad host
	addr := "//a/"
	params := services.CreateMachineParams{
		Machine: &models.Machine{
			Address:   &addr,
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
			Address:   &addr,
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
			Address:   &addr,
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

	settings := RestAPISettings{}
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
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

	settings := RestAPISettings{}
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
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
	s := &dbmodel.App{
		ID:        0,
		MachineID: m2.ID,
		Type:      dbmodel.KeaAppType,
		CtrlPort:  1234,
		CtrlKey:   "",
		Active:    true,
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{},
		},
	}
	err = dbmodel.AddApp(db, s)
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
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// update non-existing machine
	addr := "localhost"
	params := services.UpdateMachineParams{
		ID: 123,
		Machine: &models.Machine{
			Address:   &addr,
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
	require.Equal(t, 400, getStatusCode(*defaultRsp))
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
	require.Equal(t, 400, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot parse address", *defaultRsp.Payload.Message)
}

func TestDeleteMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
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
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp = rsp.(*services.GetMachineOK)
	require.Equal(t, m.ID, okRsp.Payload.ID)
}

func TestGetApp(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing app
	params := services.GetAppParams{
		ID: 123,
	}
	rsp := rapi.GetApp(ctx, params)
	require.IsType(t, &services.GetAppDefault{}, rsp)
	defaultRsp := rsp.(*services.GetAppDefault)
	require.Equal(t, 404, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find app with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app to machine
	s := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.KeaAppType,
		CtrlPort:  1234,
		CtrlKey:   "",
		Active:    true,
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{},
		},
	}
	err = dbmodel.AddApp(db, s)
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
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing app
	params := services.GetAppParams{
		ID: 123,
	}
	rsp := rapi.GetApp(ctx, params)
	require.IsType(t, &services.GetAppDefault{}, rsp)
	defaultRsp := rsp.(*services.GetAppDefault)
	require.Equal(t, 404, getStatusCode(*defaultRsp))
	require.Equal(t, "cannot find app with id 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add kea app to machine
	keaApp := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.KeaAppType,
		CtrlPort:  1234,
		CtrlKey:   "",
		Active:    true,
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{},
		},
	}
	err = dbmodel.AddApp(db, keaApp)
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
	bind9App := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.Bind9AppType,
		CtrlPort:  953,
		CtrlKey:   "abcd",
		Active:    true,
		Details:   dbmodel.AppBind9{},
	}
	err = dbmodel.AddApp(db, bind9App)
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
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// get empty list of app
	params := services.GetAppsParams{}
	rsp := rapi.GetApps(ctx, params)
	require.IsType(t, &services.GetAppsOK{}, rsp)
	okRsp := rsp.(*services.GetAppsOK)
	require.Equal(t, int64(0), okRsp.Payload.Total)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app kea to machine
	s1 := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.KeaAppType,
		CtrlPort:  1234,
		CtrlKey:   "",
		Active:    true,
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{},
		},
	}
	err = dbmodel.AddApp(db, s1)
	require.NoError(t, err)

	// add app BIND 9 to machine
	s2 := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.Bind9AppType,
		CtrlPort:  4321,
		CtrlKey:   "abcd",
		Active:    true,
		Details:   dbmodel.AppBind9{},
	}
	err = dbmodel.AddApp(db, s2)
	require.NoError(t, err)

	// get added apps
	params = services.GetAppsParams{}
	rsp = rapi.GetApps(ctx, params)
	require.IsType(t, &services.GetAppsOK{}, rsp)
	okRsp = rsp.(*services.GetAppsOK)
	require.Equal(t, int64(2), okRsp.Payload.Total)
}

// Generates a response to the status-get command including two status
// structures, one for DHCPv4 and one for DHCPv6. Both contain HA
// status information.
func mockGetStatusWithHA(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4", "dhcp6")
	command, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)
	json := `[
        {
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "ha-servers":
                    {
                        "local": {
                            "role": "primary",
                            "scopes": [ "server1" ],
                            "state": "load-balancing"
                        },
                        "remote": {
                            "age": 10,
                            "in-touch": true,
                            "role": "secondary",
                            "last-scopes": [ "server2" ],
                            "last-state": "load-balancing"
                        }
                    }
              }
         },
         {
             "result": 0,
             "text": "Everything is fine",
             "arguments": {
                 "pid": 2345,
                 "uptime": 3333,
                 "reload": 2222,
                 "ha-servers":
                     {
                         "local": {
                             "role": "primary",
                             "scopes": [ "server1" ],
                             "state": "hot-standby"
                         },
                         "remote": {
                             "age": 3,
                             "in-touch": true,
                             "role": "standby",
                             "last-scopes": [ ],
                             "last-state": "waiting"
                         }
                     }
               }
          }
    ]`
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Test that status of two HA services for a Kea application is parsed
// correctly.
func TestRestGetAppServicesStatus(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	// Configure the fake control agents to mimic returning a status of
	// two HA services for Kea.
	fa := storktest.NewFakeAgents(mockGetStatusWithHA)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
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
	keaApp := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.KeaAppType,
		CtrlPort:  1234,
		CtrlKey:   "",
		Active:    true,
	}
	err = dbmodel.AddApp(db, keaApp)
	require.NoError(t, err)

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
	require.Equal(t, 2, len(okRsp.Payload.Items))

	statusList := okRsp.Payload.Items

	// Validate the status of the DHCPv4 pair.
	status := statusList[0].Status.KeaStatus
	require.Equal(t, int64(1234), status.Pid)
	require.Equal(t, int64(1111), status.Reload)
	require.Equal(t, int64(3024), status.Uptime)
	require.NotNil(t, status.HaServers)

	haStatus := status.HaServers
	require.NotNil(t, haStatus.LocalServer)
	require.NotNil(t, haStatus.RemoteServer)

	require.Equal(t, "primary", haStatus.LocalServer.Role)
	require.Equal(t, 1, len(haStatus.LocalServer.Scopes))
	require.Contains(t, haStatus.LocalServer.Scopes, "server1")
	require.Equal(t, "load-balancing", haStatus.LocalServer.State)

	require.Equal(t, "secondary", haStatus.RemoteServer.Role)
	require.Equal(t, 1, len(haStatus.RemoteServer.Scopes))
	require.Contains(t, haStatus.RemoteServer.Scopes, "server2")
	require.Equal(t, "load-balancing", haStatus.RemoteServer.State)
	require.Equal(t, int64(10), haStatus.RemoteServer.Age)
	require.True(t, haStatus.RemoteServer.InTouch)

	// Validate the status of the DHCPv6 pair.
	status = statusList[1].Status.KeaStatus
	require.Equal(t, int64(2345), status.Pid)
	require.Equal(t, int64(2222), status.Reload)
	require.Equal(t, int64(3333), status.Uptime)
	require.NotNil(t, status.HaServers)

	haStatus = status.HaServers
	require.NotNil(t, haStatus.LocalServer)
	require.NotNil(t, haStatus.RemoteServer)

	require.Equal(t, "primary", haStatus.LocalServer.Role)
	require.Equal(t, 1, len(haStatus.LocalServer.Scopes))
	require.Contains(t, haStatus.LocalServer.Scopes, "server1")
	require.Equal(t, "hot-standby", haStatus.LocalServer.State)

	require.Equal(t, "standby", haStatus.RemoteServer.Role)
	require.Empty(t, haStatus.RemoteServer.Scopes)
	require.Equal(t, "waiting", haStatus.RemoteServer.State)
	require.Equal(t, int64(3), haStatus.RemoteServer.Age)
	require.True(t, haStatus.RemoteServer.InTouch)
}

func TestRestGetAppsStats(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := storktest.NewFakeAgents(nil)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// get empty list of app
	params := services.GetAppsStatsParams{}
	rsp := rapi.GetAppsStats(ctx, params)
	require.IsType(t, &services.GetAppsStatsOK{}, rsp)
	okRsp := rsp.(*services.GetAppsStatsOK)
	require.Equal(t, int64(0), okRsp.Payload.KeaAppsTotal)
	require.Equal(t, int64(0), okRsp.Payload.KeaAppsNotOk)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add app kea to machine
	s1 := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.KeaAppType,
		CtrlPort:  1234,
		CtrlKey:   "",
		Active:    true,
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{},
		},
	}
	err = dbmodel.AddApp(db, s1)
	require.NoError(t, err)

	// add app bind9 to machine
	s2 := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.Bind9AppType,
		CtrlPort:  4321,
		CtrlKey:   "abcd",
		Active:    false,
		Details:   dbmodel.AppBind9{},
	}
	err = dbmodel.AddApp(db, s2)
	require.NoError(t, err)

	// get added app
	params = services.GetAppsStatsParams{}
	rsp = rapi.GetAppsStats(ctx, params)
	require.IsType(t, &services.GetAppsStatsOK{}, rsp)
	okRsp = rsp.(*services.GetAppsStatsOK)
	require.Equal(t, int64(1), okRsp.Payload.KeaAppsTotal)
	require.Equal(t, int64(0), okRsp.Payload.KeaAppsNotOk)
	require.Equal(t, int64(1), okRsp.Payload.Bind9AppsTotal)
	require.Equal(t, int64(1), okRsp.Payload.Bind9AppsNotOk)
}
