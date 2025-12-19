package restservice

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	keaconfig "isc.org/stork/daemoncfg/kea"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/daemondata/bind9stats"
	pdnsdata "isc.org/stork/daemondata/pdns"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/pki"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	"isc.org/stork/server/certs"
	"isc.org/stork/server/daemons/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	"isc.org/stork/server/gen/restapi/operations/general"
	"isc.org/stork/server/gen/restapi/operations/services"
	storktest "isc.org/stork/server/test/dbmodel"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=restservice -destination=connectedagentsmock_test.go isc.org/stork/server/agentcomm ConnectedAgents

// Type of the function returning agent communication stats wrapped.
type wrapperFunc = func(string, int64) *agentcomm.CommStatsWrapper

// Convenience function wrapping statistics in CommStatsWrapper and
// returning in tests mocks.
func wrap(stats *agentcomm.CommStats) wrapperFunc {
	return func(string, int64) *agentcomm.CommStatsWrapper {
		return agentcomm.NewCommStatsWrapper(stats)
	}
}

func TestGetVersion(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
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
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// get state of non-existing machine
	params := services.GetMachineStateParams{
		ID: 123,
	}
	rsp := rapi.GetMachineState(ctx, params)
	require.IsType(t, &services.GetMachineStateDefault{}, rsp)
	defaultRsp := rsp.(*services.GetMachineStateDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "Cannot find machine with ID 123", *defaultRsp.Payload.Message)

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
	require.NotNil(t, okRsp.Payload.Daemons)
	require.Empty(t, okRsp.Payload.Daemons)
}

func mockGetAppsState(callNo int, cmdResponses []interface{}) {
	switch callNo {
	case 0:
		versionResponse := cmdResponses[0].(*kea.VersionGetResponse)
		*versionResponse = kea.VersionGetResponse{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Text:   "2.3.2",
			},
			Arguments: &kea.VersionGetRespArgs{
				Extended: "Extended version",
			},
		}
		response := cmdResponses[1].(*keactrl.Response)
		*response = keactrl.Response{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
			},
			Arguments: []byte(`{
				"Control-agent": {
					"control-sockets": {
						"dhcp4": {
							"socket-name": "aaa",
							"socket-type": "unix"
						}
					}
				}
			}`),
		}
	case 1:
		// version-get response
		versionResponse := cmdResponses[0].(*kea.VersionGetResponse)
		*versionResponse = kea.VersionGetResponse{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
				Text:   "2.3.0",
			},
			Arguments: &kea.VersionGetRespArgs{
				Extended: "Extended version",
			},
		}

		// config-get response
		response := cmdResponses[1].(*keactrl.Response)
		*response = keactrl.Response{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
			},
			Arguments: []byte(`{
				"Dhcp4": {
					"hooks-libraries": [
						{
							"library": "hook_abc.so"
						},
						{
							"library": "hook_def.so"
						}
					]
				}
			}`),
		}

		// status-get response
		statusResponse := cmdResponses[2].(*kea.StatusGetResponse)
		*statusResponse = kea.StatusGetResponse{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
			},
			Arguments: &kea.StatusGetRespArgs{
				Pid: 123,
			},
		}
	default:
		panic("unexpected call number")
	}
}

func TestGetMachineAndDaemonsState(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(mockGetAppsState, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add kea daemon
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "1.2.3.4",
		Port:     123,
		Key:      "",
		Protocol: protocoltype.HTTP,
	}
	keaDaemon := dbmodel.NewDaemon(m, daemonname.CA, true, []*dbmodel.AccessPoint{keaAccessPoint})
	err = dbmodel.AddDaemon(db, keaDaemon)
	require.NoError(t, err)

	// add BIND 9 daemon
	bind9AccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "1.2.3.4",
		Port:     124,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}
	bind9Daemon := dbmodel.NewDaemon(m, daemonname.Bind9, true, []*dbmodel.AccessPoint{bind9AccessPoint})
	err = dbmodel.AddDaemon(db, bind9Daemon)
	require.NoError(t, err)

	fa.MachineState = &agentcomm.State{
		AgentVersion: "2.3.2",
		Daemons: []*agentcomm.Daemon{
			{
				Name: daemonname.CA,
				AccessPoints: []dbmodel.AccessPoint{{
					Type:     dbmodel.AccessPointControl,
					Address:  "1.2.3.4",
					Port:     123,
					Key:      "",
					Protocol: protocoltype.HTTP,
				}},
				Machine: m,
			},
			{
				Name: daemonname.Bind9,
				AccessPoints: []dbmodel.AccessPoint{
					{
						Type:     dbmodel.AccessPointControl,
						Address:  "1.2.3.4",
						Port:     124,
						Key:      "abcd",
						Protocol: protocoltype.RNDC,
					},
					{
						Type:     dbmodel.AccessPointStatistics,
						Address:  "1.2.3.4",
						Port:     125,
						Protocol: protocoltype.HTTP,
					},
				},
				Machine: m,
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
	require.Len(t, okRsp.Payload.Daemons, 2)
	require.EqualValues(t, daemonname.CA, okRsp.Payload.Daemons[0].Name)
	require.EqualValues(t, daemonname.Bind9, okRsp.Payload.Daemons[1].Name)
	require.Nil(t, okRsp.Payload.LastVisitedAt)
}

// Test that machine state includes PowerDNS daemon information and that this
// information is correctly populated.
func TestGetMachineAndPowerDNSState(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add machine to db.
	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Add PowerDNS daemon to db.
	accessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "1.2.3.4",
		Port:     124,
		Protocol: protocoltype.HTTP,
	}
	pdnsDaemon := dbmodel.NewDaemon(machine, daemonname.PDNS, true, []*dbmodel.AccessPoint{accessPoint})
	pdnsDaemon.Pid = 123
	pdnsDaemon.Active = true
	pdnsDaemon.Monitored = true
	pdnsDaemon.Version = "4.7.3"
	pdnsDaemon.ExtendedVersion = "4.7.3"
	pdnsDaemon.Uptime = 1000
	pdnsDaemon.PDNSDaemon = &dbmodel.PDNSDaemon{
		Details: dbmodel.PDNSDaemonDetails{
			URL:              "http://1.2.3.4:123",
			ConfigURL:        "http://1.2.3.4:123/config",
			ZonesURL:         "http://1.2.3.4:123/zones",
			AutoprimariesURL: "http://1.2.3.4:123/autoprimaries",
		},
	}
	err = dbmodel.AddDaemon(db, pdnsDaemon)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	mockAgents := NewMockConnectedAgents(controller)

	mockAgents.EXPECT().GetState(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, machineTag dbmodel.MachineTag) (*agentcomm.State, error) {
		require.EqualValues(t, machine.ID, machineTag.GetID())
		return &agentcomm.State{
			AgentVersion: "2.3.2",
			Daemons: []*agentcomm.Daemon{
				{
					Name: daemonname.PDNS,
					AccessPoints: []dbmodel.AccessPoint{{
						Type:     dbmodel.AccessPointControl,
						Address:  "1.2.3.4",
						Port:     124,
						Key:      "",
						Protocol: protocoltype.HTTP,
					}},
				},
			},
		}, nil
	})

	mockAgents.EXPECT().GetPowerDNSServerInfo(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, daemon agentcomm.ControlledDaemon) (*pdnsdata.ServerInfo, error) {
		require.EqualValues(t, pdnsDaemon.ID, daemon.GetMachineTag().GetID())
		return &pdnsdata.ServerInfo{
			Type:             "PowerDNS",
			ID:               "127.0.0.1",
			DaemonType:       "pdns",
			Version:          "4.7.0",
			URL:              "http://127.0.0.1:8081",
			ConfigURL:        "http://127.0.0.1:8081/config",
			ZonesURL:         "http://127.0.0.1:8081/zones",
			AutoprimariesURL: "http://127.0.0.1:8081/autoprimaries",
			Uptime:           1234,
		}, nil
	})

	mockAgents.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	rapi, err := NewRestAPI(&RestAPISettings{}, dbSettings, db, mockAgents, &storktest.FakeEventCenter{}, &storktest.FakeDispatcher{})
	require.NoError(t, err)

	rsp := rapi.GetMachineState(context.Background(), services.GetMachineStateParams{
		ID: 1,
	})
	require.IsType(t, &services.GetMachineStateOK{}, rsp)
	okRsp := rsp.(*services.GetMachineStateOK)
	require.Len(t, okRsp.Payload.Daemons, 1)

	returnedDaemon := okRsp.Payload.Daemons[0]

	require.EqualValues(t, pdnsDaemon.ID, returnedDaemon.ID)
	require.EqualValues(t, pdnsDaemon.Name, returnedDaemon.Name)
	require.Equal(t, "1.2.3.4", returnedDaemon.AccessPoints[0].Address)
	require.Len(t, returnedDaemon.AccessPoints, 1)
	require.EqualValues(t, 124, returnedDaemon.AccessPoints[0].Port)

	require.EqualValues(t, pdnsDaemon.Name, returnedDaemon.Name)
	require.Equal(t, "4.7.0", returnedDaemon.Version)
	require.EqualValues(t, 1234, returnedDaemon.Uptime)
	require.Equal(t, "http://127.0.0.1:8081", returnedDaemon.URL)
	require.Equal(t, "http://127.0.0.1:8081/config", returnedDaemon.ConfigURL)
	require.Equal(t, "http://127.0.0.1:8081/zones", returnedDaemon.ZonesURL)
	require.Equal(t, "http://127.0.0.1:8081/autoprimaries", returnedDaemon.AutoprimariesURL)

	updatedDaemon, err := dbmodel.GetDaemonByID(db, pdnsDaemon.ID)
	require.NoError(t, err)
	require.EqualValues(t, returnedDaemon.Name, updatedDaemon.Name)
	require.Equal(t, "4.7.0", updatedDaemon.Version)
	require.Equal(t, "4.7.0", updatedDaemon.ExtendedVersion)
	require.EqualValues(t, 1234, updatedDaemon.Uptime)
	require.Equal(t, "http://127.0.0.1:8081", updatedDaemon.PDNSDaemon.Details.URL)
	require.Equal(t, "http://127.0.0.1:8081/config", updatedDaemon.PDNSDaemon.Details.ConfigURL)
	require.Equal(t, "http://127.0.0.1:8081/zones", updatedDaemon.PDNSDaemon.Details.ZonesURL)
	require.Equal(t, "http://127.0.0.1:8081/autoprimaries", updatedDaemon.PDNSDaemon.Details.AutoprimariesURL)
}

func TestCreateMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	ec := NewEndpointControl()
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd, ec)
	require.NoError(t, err)
	ctx := context.Background()

	// empty request, variant 1 - should raise an error
	params := services.CreateMachineParams{}
	rsp := rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp := rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Missing parameters", *defaultRsp.Payload.Message)

	// empty request, variant 2 - should raise an error
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Missing parameters", *defaultRsp.Payload.Message)

	// prepare request arguments
	addr := "1.2.3.4"
	port := int64(8080)
	serverToken := "serverToken" // it will be corrected later when server cert is generated
	agentToken := "agentToken"
	privKeyPEM, err := pki.GenKey()
	require.NoError(t, err)
	csrPEM, _, err := pki.GenCSRUsingKey("agent", []string{"name"}, []net.IP{net.ParseIP("192.0.2.1")}, privKeyPEM)
	require.NoError(t, err)
	agentCSR := string(csrPEM)

	// bad host
	badAddr := "//a/"
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     &badAddr,
			AgentPort:   port,
			AgentCSR:    &agentCSR,
			ServerToken: serverToken,
			AgentToken:  &agentToken,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Cannot parse address", *defaultRsp.Payload.Message)

	// bad port
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     &addr,
			AgentPort:   0,
			AgentCSR:    &agentCSR,
			ServerToken: serverToken,
			AgentToken:  &agentToken,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Bad port", *defaultRsp.Payload.Message)

	// missing server cert in db so there will be server internal error
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     &addr,
			AgentPort:   port,
			AgentCSR:    &agentCSR,
			ServerToken: serverToken,
			AgentToken:  &agentToken,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "Problem calculating fingerprint of server cert", *defaultRsp.Payload.Message)

	// add server cert to db
	caCertPEM1, serverCertPEM1, serverKeyPEM1, err := certs.SetupServerCerts(db)
	require.NoError(t, err)

	// check if rerun of setup server certs works as well
	caCertPEM2, serverCertPEM2, serverKeyPEM2, err := certs.SetupServerCerts(db)
	require.NoError(t, err)
	require.Equal(t, caCertPEM1, caCertPEM2)
	require.Equal(t, serverCertPEM1, serverCertPEM2)
	require.Equal(t, serverKeyPEM1, serverKeyPEM2)

	// get server token to use it in requests below
	dbServerToken, err := dbmodel.GetSecret(db, dbmodel.SecretServerToken)
	require.NoError(t, err)
	serverToken = string(dbServerToken) // set correct value for server token

	// bad server token
	badServerToken := "abc"
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     &addr,
			AgentPort:   port,
			AgentCSR:    &agentCSR,
			ServerToken: badServerToken,
			AgentToken:  &agentToken,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Provided server token is wrong", *defaultRsp.Payload.Message)

	// bad agent CSR
	badAgentCSR := "abc"
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     &addr,
			AgentPort:   port,
			AgentCSR:    &badAgentCSR,
			ServerToken: serverToken,
			AgentToken:  &agentToken,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Problem with agent CSR", *defaultRsp.Payload.Message)

	// bad (empty) agent token
	emptyAgentToken := ""
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     &addr,
			AgentPort:   8080,
			AgentCSR:    &agentCSR,
			ServerToken: serverToken,
			AgentToken:  &emptyAgentToken,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Agent token cannot be empty", *defaultRsp.Payload.Message)

	// all ok
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     &addr,
			AgentPort:   8080,
			AgentCSR:    &agentCSR,
			ServerToken: serverToken,
			AgentToken:  &agentToken,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineOK{}, rsp)
	okRsp := rsp.(*services.CreateMachineOK)
	require.NotEmpty(t, okRsp.Payload.ID)
	require.NotEmpty(t, okRsp.Payload.ServerCACert)
	require.EqualValues(t, caCertPEM1, okRsp.Payload.ServerCACert)
	require.NotEmpty(t, okRsp.Payload.ServerCertFingerprint)
	expectedServerCertFingerprint, _ := pki.CalculateFingerprintFromPEM(serverCertPEM1)
	expectedServerCertFingerprintHex := storkutil.BytesToHex(expectedServerCertFingerprint[:])
	require.Equal(t, expectedServerCertFingerprintHex, okRsp.Payload.ServerCertFingerprint)

	require.NotEmpty(t, okRsp.Payload.AgentCert)
	machines, err := dbmodel.GetAllMachines(db, nil)
	require.NoError(t, err)
	require.Len(t, machines, 1)
	m1 := machines[0]
	require.True(t, m1.Authorized)
	certFingerprint1 := m1.CertFingerprint

	serverCACertFingerprint, err := pki.CalculateFingerprintFromPEM([]byte(okRsp.Payload.ServerCACert))
	require.NoError(t, err)

	// ok, now lets ping the machine if it is alive
	pingParams := services.PingMachineParams{
		ID: m1.ID,
		Ping: services.PingMachineBody{
			ServerToken: serverToken,
			AgentToken:  agentToken,
		},
	}
	require.False(t, fa.GetStateCalled)
	pingRsp := rapi.PingMachine(ctx, pingParams)
	require.IsType(t, &services.PingMachineOK{}, pingRsp)
	_, ok := pingRsp.(*services.PingMachineOK)
	require.True(t, ok)
	// check if GetMachineAndAppsState was called
	require.True(t, fa.GetStateCalled)

	// Make sure that the event informing about the new machine registration
	// has been emitted.
	require.Len(t, fec.Events, 1)
	require.Contains(t, fec.Events[0].Text, "added")
	require.Contains(t, fec.Events[0].SSEStreams, dbmodel.SSERegistration)

	// re-register (the same) machine (it should be break)
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:           &addr,
			AgentPort:         8080,
			AgentCSR:          &agentCSR,
			ServerToken:       serverToken,
			AgentToken:        &agentToken,
			CaCertFingerprint: storkutil.BytesToHex(serverCACertFingerprint[:]),
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineConflict{}, rsp)
	conflictRsp := rsp.(*services.CreateMachineConflict)
	require.NotEmpty(t, conflictRsp.Location)
	expectedLocation := fmt.Sprintf("/machines/%d", okRsp.Payload.ID)
	require.Equal(t, expectedLocation, conflictRsp.Location)
	require.Equal(t, expectedServerCertFingerprintHex, conflictRsp.Payload.ServerCertFingerprint)
	require.Equal(t, okRsp.Payload.ID, conflictRsp.Payload.ID)

	machines, err = dbmodel.GetAllMachines(db, nil)
	require.NoError(t, err)
	require.Len(t, machines, 1)
	m1 = machines[0]
	require.True(t, m1.Authorized)
	// agent cert isn't re-signed so fingerprint should be the same
	require.Equal(t, certFingerprint1, m1.CertFingerprint)
	require.True(t, m1.LastVisitedAt.IsZero())

	// There should be no new events generated.
	require.Len(t, fec.Events, 1)

	// Re-register (the same) machine but with different CA cert fingerprint.
	// Server should generate new agent cert. The authorization status should
	// be preserved.
	nonMatchingFingerprint := [32]byte{42}
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:   &addr,
			AgentPort: 8080,
			AgentCSR:  &agentCSR,
			// Missing server token.
			AgentToken:        &agentToken,
			CaCertFingerprint: storkutil.BytesToHex(nonMatchingFingerprint[:]),
		},
	}

	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineOK{}, rsp)
	okRsp2 := rsp.(*services.CreateMachineOK)
	require.Equal(t, okRsp.Payload.ID, okRsp2.Payload.ID)
	require.EqualValues(t, caCertPEM1, okRsp2.Payload.ServerCACert)
	require.Equal(t, expectedServerCertFingerprintHex, okRsp2.Payload.ServerCertFingerprint)
	require.NotEqual(t, okRsp.Payload.AgentCert, okRsp2.Payload.AgentCert)

	// The new event should not be sent in the registration stream.
	require.Len(t, fec.Events, 2)
	require.Contains(t, fec.Events[1].Text, "re-registered")
	require.NotContains(t, fec.Events[1].SSEStreams, dbmodel.SSERegistration)

	machines, err = dbmodel.GetAllMachines(db, nil)
	require.NoError(t, err)
	require.Len(t, machines, 1)
	m1 = machines[0]
	require.True(t, m1.Authorized)
	// agent cert is re-signed so fingerprint shouldn't be the same
	require.NotEqual(t, certFingerprint1, m1.CertFingerprint)
	require.True(t, m1.LastVisitedAt.IsZero())

	// De-authorize and repeat. The machine cannot be authorized.
	m1.Authorized = false
	err = dbmodel.UpdateMachine(db, &m1)
	require.NoError(t, err)

	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:   &addr,
			AgentPort: 8080,
			AgentCSR:  &agentCSR,
			// Missing server token.
			AgentToken:        &agentToken,
			CaCertFingerprint: storkutil.BytesToHex(nonMatchingFingerprint[:]),
		},
	}

	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineOK{}, rsp)
	okRsp3 := rsp.(*services.CreateMachineOK)
	require.Equal(t, okRsp.Payload.ID, okRsp3.Payload.ID)
	require.EqualValues(t, caCertPEM1, okRsp3.Payload.ServerCACert)
	require.Equal(t, expectedServerCertFingerprintHex, okRsp3.Payload.ServerCertFingerprint)
	require.NotEqual(t, okRsp.Payload.AgentCert, okRsp3.Payload.AgentCert)

	machines, err = dbmodel.GetAllMachines(db, nil)
	require.NoError(t, err)
	require.Len(t, machines, 1)
	m1 = machines[0]
	require.False(t, m1.Authorized)

	// add another machine but with no server token (agent token is used for authorization)
	addr = "5.6.7.8"
	serverToken = ""
	params = services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     &addr,
			AgentPort:   8080,
			AgentCSR:    &agentCSR,
			ServerToken: serverToken,
			AgentToken:  &agentToken,
		},
	}
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineOK{}, rsp)
	okRsp = rsp.(*services.CreateMachineOK)
	require.NotEmpty(t, okRsp.Payload.ID)
	require.NotEmpty(t, okRsp.Payload.ServerCACert)
	require.NotEmpty(t, okRsp.Payload.AgentCert)
	machines, err = dbmodel.GetAllMachines(db, nil)
	require.NoError(t, err)
	require.Len(t, machines, 2)
	m2 := machines[0]
	if m2.ID == m1.ID {
		m2 = machines[1]
	}
	require.False(t, m2.Authorized)
}

// Test that HTTP Forbidden status code is returned when machine registration
// endpoint is disabled.
func TestCreateMachineForbidden(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	ec := NewEndpointControl()
	ec.SetEnabled(EndpointOpCreateNewMachine, false)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd, ec)
	require.NoError(t, err)
	ctx := context.Background()

	// Create certs.
	_, _, _, err = certs.SetupServerCerts(db)
	require.NoError(t, err)

	privKeyPEM, err := pki.GenKey()
	require.NoError(t, err)
	csrPEM, _, err := pki.GenCSRUsingKey("agent", []string{"name"}, []net.IP{net.ParseIP("192.0.2.1")}, privKeyPEM)
	require.NoError(t, err)
	agentCSR := string(csrPEM)

	dbServerToken, err := dbmodel.GetSecret(db, dbmodel.SecretServerToken)
	require.NoError(t, err)

	// Send a request to register new machine while the registration is disabled.
	params := services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     storkutil.Ptr("1.2.3.4"),
			AgentPort:   8080,
			AgentCSR:    &agentCSR,
			ServerToken: string(dbServerToken),
			AgentToken:  storkutil.Ptr("agentToken"),
		},
	}
	rsp := rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineDefault{}, rsp)
	defaultRsp := rsp.(*services.CreateMachineDefault)
	require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
	require.Equal(t, "Machine registration is administratively disabled", *defaultRsp.Payload.Message)
}

// Test that machine can be re-registered when the registration of the
// new machines is disabled.
func TestReregisterCreatedMachineNotForbidden(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	ec := NewEndpointControl()
	ec.SetEnabled(EndpointOpCreateNewMachine, true)
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd, ec)
	require.NoError(t, err)
	ctx := context.Background()

	// Create certs.
	_, _, _, err = certs.SetupServerCerts(db)
	require.NoError(t, err)

	privKeyPEM, err := pki.GenKey()
	require.NoError(t, err)
	csrPEM, _, err := pki.GenCSRUsingKey("agent", []string{"name"}, []net.IP{net.ParseIP("192.0.2.1")}, privKeyPEM)
	require.NoError(t, err)
	agentCSR := string(csrPEM)

	dbServerToken, err := dbmodel.GetSecret(db, dbmodel.SecretServerToken)
	require.NoError(t, err)

	// Send a request to register new machine while the registration is enabled.
	params := services.CreateMachineParams{
		Machine: &models.NewMachineReq{
			Address:     storkutil.Ptr("1.2.3.4"),
			AgentPort:   8080,
			AgentCSR:    &agentCSR,
			ServerToken: string(dbServerToken),
			AgentToken:  storkutil.Ptr("agentToken"),
		},
	}
	rsp := rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineOK{}, rsp)

	// Disable registration of the new machines.
	rapi.EndpointControl.SetEnabled(EndpointOpCreateNewMachine, false)

	// Ensure that re-registration of the already registered machine is successful.
	rsp = rapi.CreateMachine(ctx, params)
	require.IsType(t, &services.CreateMachineOK{}, rsp)
}

func TestGetMachines(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine1 := &dbmodel.Machine{
		Address:   "machine1.example.org",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine1)
	require.NoError(t, err)

	machine2 := &dbmodel.Machine{
		Address:   "machine2.example.org",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	var start, limit int64 = 0, 10
	params := services.GetMachinesParams{
		Start: &start,
		Limit: &limit,
	}

	rsp := rapi.GetMachines(ctx, params)
	ms := rsp.(*services.GetMachinesOK).Payload
	require.EqualValues(t, ms.Total, 2)

	// Check if results were sorted by default by ID in ascending order.
	require.Greater(t, ms.Items[1].ID, ms.Items[0].ID)

	// Test GetMachines again but with explicit sorting applied.
	params = services.GetMachinesParams{
		Start:     &start,
		Limit:     &limit,
		SortField: storkutil.Ptr("address"),
		SortDir:   storkutil.Ptr(string(dbmodel.SortDirDesc)),
	}

	rsp = rapi.GetMachines(ctx, params)
	ms = rsp.(*services.GetMachinesOK).Payload
	require.EqualValues(t, ms.Total, 2)

	// Check if results were sorted as expected.
	require.EqualValues(t, "machine2.example.org", *ms.Items[0].Address)
	require.EqualValues(t, "machine1.example.org", *ms.Items[1].Address)
}

// Test that the empty list is returned when the database contains no machines.
func TestGetMachinesEmptyList(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
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
	require.NotNil(t, ms.Items)
	require.Len(t, ms.Items, 0)
}

// Test that a list of machines' ids and addresses/names is returned
// via the API.
func TestGetMachinesDirectory(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add machines.
	machine1 := &dbmodel.Machine{
		Address:    "machine1.example.org",
		AgentPort:  8080,
		Authorized: true,
	}
	err := dbmodel.AddMachine(db, machine1)
	require.NoError(t, err)

	machine2 := &dbmodel.Machine{
		Address:    "machine2.example.org",
		AgentPort:  8080,
		Authorized: true,
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	params := services.GetMachinesDirectoryParams{}

	rsp := rapi.GetMachinesDirectory(ctx, params)
	machines := rsp.(*services.GetMachinesDirectoryOK).Payload
	require.EqualValues(t, machines.Total, 2)

	// Ensure that the returned machines are in a coherent order.
	sort.Slice(machines.Items, func(i, j int) bool {
		return machines.Items[i].ID < machines.Items[j].ID
	})

	// Validate the returned data.
	require.Equal(t, machine1.ID, machines.Items[0].ID)
	require.NotNil(t, machines.Items[0].Address)
	require.Equal(t, machine1.Address, *machines.Items[0].Address)
	require.Equal(t, machine2.ID, machines.Items[1].ID)
	require.NotNil(t, machines.Items[1].Address)
	require.Equal(t, machine2.Address, *machines.Items[1].Address)
}

// Test getting the number of the unauthorized machines.
func TestGetUnauthorizedMachinesCount(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	for i := 0; i < 10; i++ {
		machine := &dbmodel.Machine{
			Address:   fmt.Sprintf("machine%d", i),
			AgentPort: 8080,
		}
		// Only some machines are unauthorized.
		if i > 8 {
			machine.Authorized = true
		}
		err := dbmodel.AddMachine(db, machine)
		require.NoError(t, err)
	}

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	params := services.GetUnauthorizedMachinesCountParams{}
	rsp := rapi.GetUnauthorizedMachinesCount(ctx, params)

	require.IsType(t, &services.GetUnauthorizedMachinesCountOK{}, rsp)
	okRsp := rsp.(*services.GetUnauthorizedMachinesCountOK)
	require.EqualValues(t, 9, okRsp.Payload)
}

// Test an error case when getting the number of the unauthorized machines.
func TestGetUnauthorizedMachinesCountError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	// Close the database before we make a query. It should result
	// in an error.
	teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	params := services.GetUnauthorizedMachinesCountParams{}
	rsp := rapi.GetUnauthorizedMachinesCount(ctx, params)

	require.IsType(t, &services.GetUnauthorizedMachinesCountDefault{}, rsp)
	defaultRsp := rsp.(*services.GetUnauthorizedMachinesCountDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "Cannot get a number of the unauthorized machines from the database", *defaultRsp.Payload.Message)
}

func TestGetMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
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
	require.Equal(t, "Cannot find machine with ID 123", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:       "localhost",
		AgentPort:     8080,
		LastVisitedAt: time.Now(),
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
	require.NotNil(t, okRsp.Payload.LastVisitedAt)

	// add machine 2
	m2 := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8082,
	}
	err = dbmodel.AddMachine(db, m2)
	require.NoError(t, err)

	// add daemon to machine 2
	accessPoint := &dbmodel.AccessPoint{
		Type:    dbmodel.AccessPointControl,
		Address: "",
		Port:    1234,
	}
	daemon := dbmodel.NewDaemon(m2, daemonname.CA, true, []*dbmodel.AccessPoint{accessPoint})
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)
	require.NotEqual(t, 0, daemon.ID)

	// get added machine 2 with kea app
	params = services.GetMachineParams{
		ID: m2.ID,
	}
	rsp = rapi.GetMachine(ctx, params)
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp = rsp.(*services.GetMachineOK)
	require.Equal(t, m2.ID, okRsp.Payload.ID)
	require.Len(t, okRsp.Payload.Daemons, 1)
	require.Equal(t, daemon.ID, okRsp.Payload.Daemons[0].ID)
	require.Len(t, okRsp.Payload.Daemons[0].AccessPoints, 1)
	require.Nil(t, okRsp.Payload.LastVisitedAt)
}

func TestUpdateMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// empty request, variant 1 - should raise an error
	params := services.UpdateMachineParams{}
	rsp := rapi.UpdateMachine(ctx, params)
	defaultRsp := rsp.(*services.UpdateMachineDefault)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Missing parameters", *defaultRsp.Payload.Message)

	// empty request, variant 2 - should raise an error
	params = services.UpdateMachineParams{
		Machine: &models.Machine{},
	}
	rsp = rapi.UpdateMachine(ctx, params)
	require.IsType(t, &services.UpdateMachineDefault{}, rsp)
	defaultRsp = rsp.(*services.UpdateMachineDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Missing parameters", *defaultRsp.Payload.Message)

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
	require.Equal(t, "Cannot find machine with ID 123", *defaultRsp.Payload.Message)

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
	require.False(t, okRsp.Payload.Authorized) // machine is not yet authorized
	require.Nil(t, okRsp.Payload.LastVisitedAt)

	// setup a user session, it is required to check user role in UpdateMachine
	// in case of authorization change
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx2, err := rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx2, user)
	require.NoError(t, err)

	// authorize the machine
	require.False(t, fa.GetStateCalled)
	params = services.UpdateMachineParams{
		ID: m.ID,
		Machine: &models.Machine{
			Address:    &addr,
			AgentPort:  8080,
			Authorized: true,
		},
	}
	rsp = rapi.UpdateMachine(ctx2, params)
	okRsp = rsp.(*services.UpdateMachineOK)
	require.Equal(t, m.ID, okRsp.Payload.ID)
	require.Equal(t, addr, *okRsp.Payload.Address)
	require.True(t, okRsp.Payload.Authorized) // machine is authorized now
	// check if GetMachineAndAppsState was called because it was just authorized
	require.True(t, fa.GetStateCalled)

	// add another machine
	m2 := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 2020,
	}
	err = dbmodel.AddMachine(db, m2)
	require.NoError(t, err)

	// update second machine to have the same address - should raise error due to duplication
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
	require.Equal(t, "Machine with address localhost:8080 already exists", *defaultRsp.Payload.Message)

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
	require.Equal(t, "Cannot parse address", *defaultRsp.Payload.Message)
}

// Test deleting a machine and its associations.
func TestDeleteMachine(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// Setup a user session, it is required to check user role in DeleteMachine.
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx2, err := rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx2, user)
	require.NoError(t, err)

	// Delete a non-existing machine.
	params := services.DeleteMachineParams{
		ID: 123,
	}
	rsp := rapi.DeleteMachine(ctx2, params)
	require.IsType(t, &services.DeleteMachineOK{}, rsp)

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost:1010",
		AgentPort: 1010,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add a daemon.
	accessPoint := &dbmodel.AccessPoint{
		Type:    dbmodel.AccessPointControl,
		Address: "localhost",
		Port:    1234,
	}
	daemon := dbmodel.NewDaemon(m, daemonname.Bind9, true, []*dbmodel.AccessPoint{accessPoint})
	daemon.Bind9Daemon = &dbmodel.Bind9Daemon{}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// Add a zone associated with the app. We will later check if its deleted
	// when the machine is deleted.
	zone := &dbmodel.Zone{
		Name: "example.org",
		LocalZones: []*dbmodel.LocalZone{
			{
				DaemonID: daemon.ID,
				View:     "default",
				Class:    "IN",
				Serial:   1,
				Type:     "master",
				LoadedAt: time.Now(),
			},
		},
	}
	err = dbmodel.AddZones(db, zone)
	require.NoError(t, err)

	// Get the machine.
	params2 := services.GetMachineParams{
		ID: m.ID,
	}
	rsp = rapi.GetMachine(ctx2, params2)
	require.IsType(t, &services.GetMachineOK{}, rsp)
	okRsp := rsp.(*services.GetMachineOK)
	require.Equal(t, m.ID, okRsp.Payload.ID)

	// Delete the machine.
	params = services.DeleteMachineParams{
		ID: m.ID,
	}
	rsp = rapi.DeleteMachine(ctx2, params)
	require.IsType(t, &services.DeleteMachineOK{}, rsp)

	// Get the deleted machine - should return not found.
	params2 = services.GetMachineParams{
		ID: m.ID,
	}
	rsp = rapi.GetMachine(ctx2, params2)
	require.IsType(t, &services.GetMachineDefault{}, rsp)
	defaultRsp := rsp.(*services.GetMachineDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))

	// Make sure that the zone associations are deleted.
	zones, _, err := dbmodel.GetZones(db, nil, "", dbmodel.SortDirAny)
	require.NoError(t, err)
	require.Empty(t, zones)
}

func TestGetDaemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing app
	params := services.GetDaemonParams{
		ID: 123,
	}
	rsp := rapi.GetDaemon(ctx, params)
	require.IsType(t, &services.GetDaemonDefault{}, rsp)
	defaultRsp := rsp.(*services.GetDaemonDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "Daemon with ID 123 not found", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add daemon to machine
	accessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1234,
		Protocol: protocoltype.HTTPS,
	}
	keaDaemon := dbmodel.NewDaemon(m, daemonname.CA, true, []*dbmodel.AccessPoint{accessPoint})
	err = dbmodel.AddDaemon(db, keaDaemon)
	require.NoError(t, err)

	// get added daemon
	params = services.GetDaemonParams{
		ID: keaDaemon.ID,
	}
	rsp = rapi.GetDaemon(ctx, params)
	require.IsType(t, &services.GetDaemonOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonOK)
	require.Equal(t, keaDaemon.ID, okRsp.Payload.ID)
}

func TestRestGetDaemon(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// get non-existing daemon
	params := services.GetDaemonParams{
		ID: 123,
	}
	rsp := rapi.GetDaemon(ctx, params)
	require.IsType(t, &services.GetDaemonDefault{}, rsp)
	defaultRsp := rsp.(*services.GetDaemonDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "Daemon with ID 123 not found", *defaultRsp.Payload.Message)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add kea daemon to machine
	accessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	keaDaemon := dbmodel.NewDaemon(m, daemonname.CA, true, []*dbmodel.AccessPoint{accessPoint})
	err = dbmodel.AddDaemon(db, keaDaemon)
	require.NoError(t, err)

	// get added kea daemon
	params = services.GetDaemonParams{
		ID: keaDaemon.ID,
	}
	rsp = rapi.GetDaemon(ctx, params)
	require.IsType(t, &services.GetDaemonOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonOK)
	require.Equal(t, keaDaemon.ID, okRsp.Payload.ID)
	require.EqualValues(t, keaDaemon.Name, okRsp.Payload.Name)

	// add BIND 9 daemon to machine
	bind9AccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     953,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}
	bind9Daemon := dbmodel.NewDaemon(m, daemonname.Bind9, true, []*dbmodel.AccessPoint{bind9AccessPoint})
	bind9Daemon.Bind9Daemon = &dbmodel.Bind9Daemon{}
	err = dbmodel.AddDaemon(db, bind9Daemon)
	require.NoError(t, err)

	// get added BIND 9 daemon
	params = services.GetDaemonParams{
		ID: bind9Daemon.ID,
	}
	rsp = rapi.GetDaemon(ctx, params)
	require.IsType(t, &services.GetDaemonOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonOK)
	require.Equal(t, bind9Daemon.ID, okRsp.Payload.ID)
	require.EqualValues(t, bind9Daemon.Name, okRsp.Payload.Name)
}

// Test getting PowerDNS app by ID.
func TestGetPowerDNSApp(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// Add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add PowerDNS daemon to machine
	accessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	pdnsDaemon := dbmodel.NewDaemon(m, daemonname.PDNS, true, []*dbmodel.AccessPoint{accessPoint})
	pdnsDaemon.PDNSDaemon = &dbmodel.PDNSDaemon{
		Details: dbmodel.PDNSDaemonDetails{
			URL:              "https://pdns.example.com",
			ConfigURL:        "https://pdns.example.com/config",
			ZonesURL:         "https://pdns.example.com/zones",
			AutoprimariesURL: "https://pdns.example.com/autoprimaries",
		},
	}
	err = dbmodel.AddDaemon(db, pdnsDaemon)
	require.NoError(t, err)

	// get added kea daemon
	params := services.GetDaemonParams{
		ID: pdnsDaemon.ID,
	}
	rsp := rapi.GetDaemon(ctx, params)
	require.IsType(t, &services.GetDaemonOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonOK)
	require.Equal(t, pdnsDaemon.ID, okRsp.Payload.ID)
	require.EqualValues(t, pdnsDaemon.Name, okRsp.Payload.Name)
	require.Equal(t, "https://pdns.example.com", okRsp.Payload.PdnsDaemonDetails.URL)
	require.Equal(t, "https://pdns.example.com/config", okRsp.Payload.PdnsDaemonDetails.ConfigURL)
	require.Equal(t, "https://pdns.example.com/zones", okRsp.Payload.PdnsDaemonDetails.ZonesURL)
	require.Equal(t, "https://pdns.example.com/autoprimaries", okRsp.Payload.PdnsDaemonDetails.AutoprimariesURL)
}

func TestRestGetDaemons(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	settings := RestAPISettings{}
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mock, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// get empty list of daemon
	params := services.GetDaemonsParams{}
	rsp := rapi.GetDaemons(ctx, params)
	require.IsType(t, &services.GetDaemonsOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonsOK)
	require.Zero(t, okRsp.Payload.Total)

	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add kea daemon to machine
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	s1 := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{keaAccessPoint})
	s1.Version = "2.6.0"
	s1.KeaDaemon = &dbmodel.KeaDaemon{}
	s1.LogTargets = []*dbmodel.LogTarget{
		{
			Name:     "kea-dhcp4",
			Severity: "DEBUG",
			Output:   "/tmp/log",
		},
	}
	err = dbmodel.AddDaemon(db, s1)
	require.NoError(t, err)

	// add BIND 9 daemon to machine
	bind9AccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     4321,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}
	s2 := dbmodel.NewDaemon(m, daemonname.Bind9, true, []*dbmodel.AccessPoint{bind9AccessPoint})
	s2.Bind9Daemon = &dbmodel.Bind9Daemon{}
	err = dbmodel.AddDaemon(db, s2)
	require.NoError(t, err)

	// add PowerDNS daemon to machine
	pdnsAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     5300,
		Key:      "defg",
		Protocol: protocoltype.HTTP,
	}
	pdnsDaemon := dbmodel.NewDaemon(m, daemonname.PDNS, true, []*dbmodel.AccessPoint{pdnsAccessPoint})
	pdnsDaemon.PDNSDaemon = &dbmodel.PDNSDaemon{
		Details: dbmodel.PDNSDaemonDetails{
			URL:              "https://pdns.example.com",
			ConfigURL:        "https://pdns.example.com/config",
			ZonesURL:         "https://pdns.example.com/zones",
			AutoprimariesURL: "https://pdns.example.com/autoprimaries",
		},
	}
	err = dbmodel.AddDaemon(db, pdnsDaemon)
	require.NoError(t, err)

	stats := agentcomm.NewAgentStats()
	stats.IncreaseAgentErrorCount("foo")

	keaStats := stats.GetKeaStats()
	keaStats.IncreaseErrorCountBy(daemonname.CA, 2)
	keaStats.IncreaseErrorCountBy(daemonname.DHCPv4, 5)

	bind9Stats := stats.GetBind9Stats()
	bind9Stats.IncreaseErrorCountBy(dbmodel.AccessPointControl, 2)
	bind9Stats.IncreaseErrorCountBy(dbmodel.AccessPointStatistics, 3)

	mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), gomock.Any()).DoAndReturn(wrap(stats)).AnyTimes()

	t.Run("get all daemons", func(t *testing.T) {
		// get added daemons
		params = services.GetDaemonsParams{}
		rsp = rapi.GetDaemons(ctx, params)
		require.IsType(t, &services.GetDaemonsOK{}, rsp)
		okRsp = rsp.(*services.GetDaemonsOK)
		require.EqualValues(t, 3, okRsp.Payload.Total)

		// Verify that the communication error counters are returned.
		require.Len(t, okRsp.Payload.Items, 3)
		for _, daemon := range okRsp.Payload.Items {
			daemonName, _ := daemonname.Parse(daemon.Name)
			switch daemonName {
			case daemonname.DHCPv4:
				require.EqualValues(t, 1, daemon.AgentCommErrors)
				require.EqualValues(t, 2, daemon.CaCommErrors)
				require.EqualValues(t, 5, daemon.DaemonCommErrors)
				require.Len(t, daemon.LogTargets, 1)
				require.Equal(t, "kea-dhcp4", daemon.LogTargets[0].Name)
				require.Equal(t, "debug", daemon.LogTargets[0].Severity)
				require.Equal(t, "/tmp/log", daemon.LogTargets[0].Output)
			case daemonname.Bind9:
				require.EqualValues(t, 1, daemon.AgentCommErrors)
				require.EqualValues(t, 2, daemon.DaemonCommErrors)
				require.EqualValues(t, 3, daemon.StatsCommErrors)
			case daemonname.PDNS:
				require.Equal(t, "https://pdns.example.com", daemon.PdnsDaemonDetails.URL)
				require.Equal(t, "https://pdns.example.com/config", daemon.PdnsDaemonDetails.ConfigURL)
				require.Equal(t, "https://pdns.example.com/zones", daemon.PdnsDaemonDetails.ZonesURL)
				require.Equal(t, "https://pdns.example.com/autoprimaries", daemon.PdnsDaemonDetails.AutoprimariesURL)
			}
		}
	})

	t.Run("get kea daemons", func(t *testing.T) {
		params = services.GetDaemonsParams{
			Daemons: []string{string(daemonname.DHCPv4), string(daemonname.DHCPv6), string(daemonname.CA)},
		}
		rsp = rapi.GetDaemons(ctx, params)
		require.IsType(t, &services.GetDaemonsOK{}, rsp)
		okRsp = rsp.(*services.GetDaemonsOK)
		require.EqualValues(t, 1, okRsp.Payload.Total)
		require.EqualValues(t, string(daemonname.DHCPv4), okRsp.Payload.Items[0].Name)
	})

	t.Run("get bind9 daemons", func(t *testing.T) {
		params = services.GetDaemonsParams{
			Daemons: []string{string(daemonname.Bind9)},
		}
		rsp = rapi.GetDaemons(ctx, params)
		require.IsType(t, &services.GetDaemonsOK{}, rsp)
		okRsp = rsp.(*services.GetDaemonsOK)
		require.EqualValues(t, 1, okRsp.Payload.Total)
		require.EqualValues(t, string(daemonname.Bind9), okRsp.Payload.Items[0].Name)
	})

	t.Run("get pdns daemons", func(t *testing.T) {
		params = services.GetDaemonsParams{
			Daemons: []string{string(daemonname.PDNS)},
		}
		rsp = rapi.GetDaemons(ctx, params)
		require.IsType(t, &services.GetDaemonsOK{}, rsp)
		okRsp = rsp.(*services.GetDaemonsOK)
		require.EqualValues(t, 1, okRsp.Payload.Total)
		require.EqualValues(t, string(daemonname.PDNS), okRsp.Payload.Items[0].Name)
	})

	t.Run("get daemons with multiple names", func(t *testing.T) {
		params = services.GetDaemonsParams{
			Daemons: []string{string(daemonname.DHCPv4), string(daemonname.Bind9)},
		}
		rsp = rapi.GetDaemons(ctx, params)
		require.IsType(t, &services.GetDaemonsOK{}, rsp)
		okRsp = rsp.(*services.GetDaemonsOK)
		require.EqualValues(t, 2, okRsp.Payload.Total)
		for _, daemon := range okRsp.Payload.Items {
			switch daemon.Name {
			case string(daemonname.DHCPv4):
				require.Equal(t, s1.ID, daemon.ID)
			case string(daemonname.Bind9):
				require.Equal(t, s2.ID, daemon.ID)
			}
		}
	})
}

// Test converting BIND9 daemon to REST API format with DNS query stats.
func TestRestGetBind9DaemonWithQueryStats(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	settings := RestAPISettings{}
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mock, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// get empty list of daemons
	params := services.GetDaemonsParams{}
	rsp := rapi.GetDaemons(ctx, params)
	require.IsType(t, &services.GetDaemonsOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonsOK)
	require.Zero(t, okRsp.Payload.Total)

	// Add machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add BIND 9 daemon to the machine.
	bind9AccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     4321,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}
	daemon := dbmodel.NewDaemon(m, daemonname.Bind9, true, []*dbmodel.AccessPoint{bind9AccessPoint})
	daemon.Bind9Daemon = &dbmodel.Bind9Daemon{
		Stats: dbmodel.Bind9DaemonStats{
			ZoneCount:          int64(100),
			AutomaticZoneCount: int64(50),
			NamedStats: bind9stats.Bind9NamedStats{
				Views: map[string]*bind9stats.Bind9StatsView{
					"trusted": {
						Resolver: &bind9stats.Bind9StatsResolver{
							CacheStats: map[string]int64{"QueryHits": 150, "QueryMisses": 50},
						},
					},
					"guest": {
						Resolver: &bind9stats.Bind9StatsResolver{
							CacheStats: map[string]int64{"QueryHits": 75, "QueryMisses": 50},
						},
					},
				},
			},
		},
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	stats := agentcomm.NewAgentStats()
	mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), gomock.Any()).DoAndReturn(wrap(stats)).AnyTimes()

	// Get daemons.
	params = services.GetDaemonsParams{}
	rsp = rapi.GetDaemons(ctx, params)
	require.IsType(t, &services.GetDaemonsOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonsOK)
	require.EqualValues(t, 1, okRsp.Payload.Total)

	// Verify BIND9 views
	require.Len(t, okRsp.Payload.Items, 1)
	bind9Daemon := okRsp.Payload.Items[0]
	require.NotNil(t, bind9Daemon)
	require.NotNil(t, bind9Daemon.Daemon)
	require.NotNil(t, bind9Daemon.Bind9DaemonDetails.Views)

	// Zone counts.
	require.EqualValues(t, 100, bind9Daemon.Bind9DaemonDetails.ZoneCount)
	require.EqualValues(t, 50, bind9Daemon.Bind9DaemonDetails.AutoZoneCount)

	// Test "trusted" view. It is at index 1 because the views are sorted by name.
	trustedView := bind9Daemon.Bind9DaemonDetails.Views[1]
	require.NotNil(t, trustedView)
	require.EqualValues(t, 150, trustedView.QueryHits)
	require.EqualValues(t, 50, trustedView.QueryMisses)
	require.EqualValues(t, 0.75, trustedView.QueryHitRatio)

	// Test "guest" view. It is at index 0 because the views are sorted by name.
	guestView := bind9Daemon.Bind9DaemonDetails.Views[0]
	require.NotNil(t, guestView)
	require.EqualValues(t, 75, guestView.QueryHits)
	require.EqualValues(t, 50, guestView.QueryMisses)
	require.EqualValues(t, 0.6, guestView.QueryHitRatio)
}

// Test that a list of daemons' ids and names is returned via the API.
func TestGetDaemonsDirectory(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// Add a machine,
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea daemon to the machine.
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	keaDaemon := dbmodel.NewDaemon(m, daemonname.CA, true, []*dbmodel.AccessPoint{keaAccessPoint})
	err = dbmodel.AddDaemon(db, keaDaemon)
	require.NoError(t, err)

	// Add BIND 9 daemon to the machine.
	bind9AccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     4321,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}
	bind9Daemon := dbmodel.NewDaemon(m, daemonname.Bind9, true, []*dbmodel.AccessPoint{bind9AccessPoint})
	bind9Daemon.Bind9Daemon = &dbmodel.Bind9Daemon{}
	err = dbmodel.AddDaemon(db, bind9Daemon)
	require.NoError(t, err)

	params := services.GetDaemonsDirectoryParams{}
	rsp := rapi.GetDaemonsDirectory(ctx, params)
	require.IsType(t, &services.GetDaemonsDirectoryOK{}, rsp)
	apps := rsp.(*services.GetDaemonsDirectoryOK).Payload
	require.EqualValues(t, 2, apps.Total)

	// Ensure that the returned apps are in a coherent order.
	sort.Slice(apps.Items, func(i, j int) bool {
		return apps.Items[i].ID < apps.Items[j].ID
	})

	// Validate the returned data.
	require.Equal(t, keaDaemon.ID, apps.Items[0].ID)
	require.NotNil(t, apps.Items[0].Name)
	require.EqualValues(t, keaDaemon.Name, apps.Items[0].Name)
	require.Equal(t, bind9Daemon.ID, apps.Items[1].ID)
	require.NotNil(t, apps.Items[1].Name)
	require.EqualValues(t, bind9Daemon.Name, apps.Items[1].Name)
}

// Test that a list of daemons with communication issues is returned.
func TestGetDaemonsCommunicationIssues(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	ctx := context.Background()

	// Add a first machine.
	m1 := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m1)
	require.NoError(t, err)

	// Add a second machine.
	m2 := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8081,
	}
	err = dbmodel.AddMachine(db, m2)
	require.NoError(t, err)

	// Add a third machine.
	m3 := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8082,
	}
	err = dbmodel.AddMachine(db, m3)
	require.NoError(t, err)

	// Add Kea daemon to the first machine.
	keaAccessPoint1 := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	keaDaemon1 := dbmodel.NewDaemon(m1, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{keaAccessPoint1})
	keaDaemon1.Version = "2.6.0"
	keaDaemon1.Monitored = true
	keaDaemon1.KeaDaemon = &dbmodel.KeaDaemon{}
	err = dbmodel.AddDaemon(db, keaDaemon1)
	require.NoError(t, err)

	// Add Kea daemon to the second machine.
	keaAccessPoint2 := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     2345,
		Protocol: protocoltype.HTTP,
	}
	keaDaemon2 := dbmodel.NewDaemon(m2, daemonname.CA, true, []*dbmodel.AccessPoint{keaAccessPoint2})
	keaDaemon2.Version = "2.6.0"
	keaDaemon2.Monitored = true
	keaDaemon2.KeaDaemon = &dbmodel.KeaDaemon{}
	err = dbmodel.AddDaemon(db, keaDaemon2)
	require.NoError(t, err)

	// Add BIND 9 daemon to the third machine.
	bind9AccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     3456,
		Protocol: protocoltype.HTTP,
	}
	bind9Daemon := dbmodel.NewDaemon(m3, daemonname.Bind9, true, []*dbmodel.AccessPoint{bind9AccessPoint})
	bind9Daemon.Monitored = true
	bind9Daemon.Bind9Daemon = &dbmodel.Bind9Daemon{}
	err = dbmodel.AddDaemon(db, bind9Daemon)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)

	rapi, err := NewRestAPI(&settings, dbSettings, db, mock, fec, fd)
	require.NoError(t, err)

	t.Run("current errors", func(t *testing.T) {
		stats1 := agentcomm.NewAgentStats()
		stats1.IncreaseAgentErrorCount("foo")
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8080)).DoAndReturn(wrap(stats1))

		stats2 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8081)).DoAndReturn(wrap(stats2))

		stats3 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8082)).DoAndReturn(wrap(stats3))

		params := services.GetDaemonsWithCommunicationIssuesParams{}
		rsp := rapi.GetDaemonsWithCommunicationIssues(ctx, params)
		require.IsType(t, &services.GetDaemonsWithCommunicationIssuesOK{}, rsp)
		daemons := rsp.(*services.GetDaemonsWithCommunicationIssuesOK).Payload
		require.EqualValues(t, 1, daemons.Total)
		require.Equal(t, string(daemonname.DHCPv4), daemons.Items[0].Name)
	})

	t.Run("ca errors", func(t *testing.T) {
		stats1 := agentcomm.NewAgentStats()
		keaStats := stats1.GetKeaStats()
		keaStats.IncreaseErrorCountBy(daemonname.CA, 10)
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8080)).DoAndReturn(wrap(stats1))

		stats2 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8081)).DoAndReturn(wrap(stats2))

		stats3 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8082)).DoAndReturn(wrap(stats3))

		params := services.GetDaemonsWithCommunicationIssuesParams{}
		rsp := rapi.GetDaemonsWithCommunicationIssues(ctx, params)
		require.IsType(t, &services.GetDaemonsWithCommunicationIssuesOK{}, rsp)
		daemons := rsp.(*services.GetDaemonsWithCommunicationIssuesOK).Payload
		require.EqualValues(t, 1, daemons.Total)
		require.Equal(t, string(daemonname.DHCPv4), daemons.Items[0].Name)
	})

	t.Run("kea daemon errors", func(t *testing.T) {
		stats1 := agentcomm.NewAgentStats()
		keaStats := stats1.GetKeaStats()
		keaStats.IncreaseErrorCount(daemonname.DHCPv4)
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8080)).DoAndReturn(wrap(stats1))

		stats2 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8081)).DoAndReturn(wrap(stats2))

		stats3 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8082)).DoAndReturn(wrap(stats3))

		params := services.GetDaemonsWithCommunicationIssuesParams{}
		rsp := rapi.GetDaemonsWithCommunicationIssues(ctx, params)
		require.IsType(t, &services.GetDaemonsWithCommunicationIssuesOK{}, rsp)
		daemons := rsp.(*services.GetDaemonsWithCommunicationIssuesOK).Payload
		require.EqualValues(t, 1, daemons.Total)
		require.Equal(t, string(daemonname.DHCPv4), daemons.Items[0].Name)
	})

	t.Run("bind9 current errors", func(t *testing.T) {
		stats1 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8080)).DoAndReturn(wrap(stats1))

		stats2 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8081)).DoAndReturn(wrap(stats2))

		stats3 := agentcomm.NewAgentStats()
		stats3.IncreaseAgentErrorCount("foo")
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8082)).DoAndReturn(wrap(stats3))

		params := services.GetDaemonsWithCommunicationIssuesParams{}
		rsp := rapi.GetDaemonsWithCommunicationIssues(ctx, params)
		require.IsType(t, &services.GetDaemonsWithCommunicationIssuesOK{}, rsp)
		apps := rsp.(*services.GetDaemonsWithCommunicationIssuesOK).Payload
		require.EqualValues(t, 1, apps.Total)
		require.Equal(t, string(daemonname.Bind9), apps.Items[0].Name)
	})

	t.Run("rndc errors", func(t *testing.T) {
		stats1 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8080)).DoAndReturn(wrap(stats1))

		stats2 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8081)).DoAndReturn(wrap(stats2))

		stats3 := agentcomm.NewAgentStats()
		bind9Stats := stats3.GetBind9Stats()
		bind9Stats.IncreaseErrorCountBy(dbmodel.AccessPointControl, 10)
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8082)).DoAndReturn(wrap(stats3))

		params := services.GetDaemonsWithCommunicationIssuesParams{}
		rsp := rapi.GetDaemonsWithCommunicationIssues(ctx, params)
		require.IsType(t, &services.GetDaemonsWithCommunicationIssuesOK{}, rsp)
		daemons := rsp.(*services.GetDaemonsWithCommunicationIssuesOK).Payload
		require.EqualValues(t, 1, daemons.Total)
		require.Equal(t, string(daemonname.Bind9), daemons.Items[0].Name)
	})

	t.Run("stats errors", func(t *testing.T) {
		stats1 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8080)).DoAndReturn(wrap(stats1))

		stats2 := agentcomm.NewAgentStats()
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8081)).DoAndReturn(wrap(stats2))

		stats3 := agentcomm.NewAgentStats()
		bind9Stats := stats3.GetBind9Stats()
		bind9Stats.IncreaseErrorCountBy(dbmodel.AccessPointStatistics, 10)
		mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8082)).DoAndReturn(wrap(stats3))

		params := services.GetDaemonsWithCommunicationIssuesParams{}
		rsp := rapi.GetDaemonsWithCommunicationIssues(ctx, params)
		require.IsType(t, &services.GetDaemonsWithCommunicationIssuesOK{}, rsp)
		apps := rsp.(*services.GetDaemonsWithCommunicationIssuesOK).Payload
		require.EqualValues(t, 1, apps.Total)
		require.Equal(t, string(daemonname.Bind9), apps.Items[0].Name)
	})
}

// Test that non-monitored daemons are not returned even when they
// report communication issues.
func TestGetDaemonsCommunicationIssuesNotMonitored(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	ctx := context.Background()

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea daemon to the machine.
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	keaDaemon := dbmodel.NewDaemon(m, daemonname.CA, true, []*dbmodel.AccessPoint{keaAccessPoint})
	keaDaemon.Monitored = false
	err = dbmodel.AddDaemon(db, keaDaemon)
	require.NoError(t, err)

	rapi, err := NewRestAPI(&settings, dbSettings, db, fec, fd)
	require.NoError(t, err)

	params := services.GetDaemonsWithCommunicationIssuesParams{}
	rsp := rapi.GetDaemonsWithCommunicationIssues(ctx, params)
	require.IsType(t, &services.GetDaemonsWithCommunicationIssuesOK{}, rsp)
	apps := rsp.(*services.GetDaemonsWithCommunicationIssuesOK).Payload
	require.EqualValues(t, 0, apps.Total)
}

// Test that status of three HA services for a Kea daemon is parsed
// correctly.
func TestRestGetDaemonServicesStatus(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	// Configure the fake control agents to mimic returning a status of
	// two HA services for Kea.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea daemon to the machine
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	keaDaemon := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{keaAccessPoint})
	err = dbmodel.AddDaemon(db, keaDaemon)
	require.NoError(t, err)
	require.NotZero(t, keaDaemon.ID)

	exampleTime := storkutil.UTCNow().Add(-5 * time.Second)
	commInterrupted := []bool{true, true}
	keaServices := []dbmodel.Service{
		{
			BaseService: dbmodel.BaseService{},
			HAService: &dbmodel.BaseHAService{
				HAType:                      "dhcp4",
				HAMode:                      "load-balancing",
				Relationship:                "server1",
				PrimaryID:                   keaDaemon.ID,
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
			BaseService: dbmodel.BaseService{},
			HAService: &dbmodel.BaseHAService{
				HAType:                      "dhcp4",
				HAMode:                      "load-balancing",
				Relationship:                "server3",
				PrimaryID:                   keaDaemon.ID,
				PrimaryStatusCollectedAt:    exampleTime,
				SecondaryStatusCollectedAt:  exampleTime,
				PrimaryLastState:            "load-balancing",
				SecondaryLastState:          "load-balancing",
				PrimaryLastScopes:           []string{"server3"},
				SecondaryLastScopes:         []string{"server4"},
				PrimaryLastFailoverAt:       exampleTime,
				SecondaryLastFailoverAt:     exampleTime,
				PrimaryCommInterrupted:      &commInterrupted[0],
				SecondaryCommInterrupted:    &commInterrupted[1],
				PrimaryConnectingClients:    1,
				SecondaryConnectingClients:  2,
				PrimaryUnackedClients:       3,
				SecondaryUnackedClients:     4,
				PrimaryUnackedClientsLeft:   5,
				SecondaryUnackedClientsLeft: 6,
				PrimaryAnalyzedPackets:      7,
				SecondaryAnalyzedPackets:    8,
			},
		},
		{
			BaseService: dbmodel.BaseService{},
			HAService: &dbmodel.BaseHAService{
				HAType:                      "dhcp6",
				HAMode:                      "hot-standby",
				Relationship:                "server1",
				PrimaryID:                   keaDaemon.ID,
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
		err = dbmodel.AddDaemonToService(db, keaServices[i].ID, keaDaemon)
		require.NoError(t, err)
	}

	params := services.GetDaemonServicesStatusParams{
		ID: keaDaemon.ID,
	}
	rsp := rapi.GetDaemonServicesStatus(ctx, params)

	// Make sure that the response is ok.
	require.IsType(t, &services.GetDaemonServicesStatusOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonServicesStatusOK)
	require.NotNil(t, okRsp.Payload.Items)

	// There should be three structures returned, two with the status of
	// the DHCPv4 server relationships and one with the status of the DHCPv6
	// server relationship.
	require.Len(t, okRsp.Payload.Items, 3)

	statusList := okRsp.Payload.Items

	// Validate the status of the first relationship.
	status := statusList[0].Status.KeaStatus
	require.NotNil(t, status.HaServers)

	haStatus := status.HaServers
	require.NotNil(t, haStatus.PrimaryServer)
	require.NotNil(t, haStatus.SecondaryServer)

	require.Equal(t, "server1", haStatus.Relationship)

	require.EqualValues(t, keaDaemon.ID, haStatus.PrimaryServer.ID)
	require.Equal(t, "primary", haStatus.PrimaryServer.Role)
	require.Len(t, haStatus.PrimaryServer.Scopes, 1)
	require.Contains(t, haStatus.PrimaryServer.Scopes, "server1")
	require.Equal(t, "load-balancing", haStatus.PrimaryServer.State)
	require.GreaterOrEqual(t, haStatus.PrimaryServer.Age, int64(5))
	require.Equal(t, "127.0.0.1", haStatus.PrimaryServer.ControlAddress)
	require.EqualValues(t, keaDaemon.ID, haStatus.PrimaryServer.ID)
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

	// Validate the status of the second relationship.
	status = statusList[1].Status.KeaStatus
	require.NotNil(t, status.HaServers)

	haStatus = status.HaServers
	require.NotNil(t, haStatus.PrimaryServer)
	require.NotNil(t, haStatus.SecondaryServer)

	require.Equal(t, "server3", haStatus.Relationship)

	require.EqualValues(t, keaDaemon.ID, haStatus.PrimaryServer.ID)
	require.Equal(t, "primary", haStatus.PrimaryServer.Role)
	require.Len(t, haStatus.PrimaryServer.Scopes, 1)
	require.Contains(t, haStatus.PrimaryServer.Scopes, "server3")
	require.Equal(t, "load-balancing", haStatus.PrimaryServer.State)
	require.GreaterOrEqual(t, haStatus.PrimaryServer.Age, int64(5))
	require.Equal(t, "127.0.0.1", haStatus.PrimaryServer.ControlAddress)
	require.EqualValues(t, keaDaemon.ID, haStatus.PrimaryServer.ID)
	require.NotEmpty(t, haStatus.PrimaryServer.StatusTime.String())
	require.EqualValues(t, 1, haStatus.PrimaryServer.CommInterrupted)
	require.EqualValues(t, 1, haStatus.PrimaryServer.ConnectingClients)
	require.EqualValues(t, 3, haStatus.PrimaryServer.UnackedClients)
	require.EqualValues(t, 5, haStatus.PrimaryServer.UnackedClientsLeft)
	require.EqualValues(t, 7, haStatus.PrimaryServer.AnalyzedPackets)

	require.Equal(t, "secondary", haStatus.SecondaryServer.Role)
	require.Len(t, haStatus.SecondaryServer.Scopes, 1)
	require.Contains(t, haStatus.SecondaryServer.Scopes, "server4")
	require.Equal(t, "load-balancing", haStatus.SecondaryServer.State)
	require.GreaterOrEqual(t, haStatus.SecondaryServer.Age, int64(5))
	require.False(t, haStatus.SecondaryServer.InTouch)
	require.Empty(t, haStatus.SecondaryServer.ControlAddress)
	require.NotEmpty(t, haStatus.SecondaryServer.StatusTime.String())
	require.EqualValues(t, 1, haStatus.SecondaryServer.CommInterrupted)
	require.EqualValues(t, 2, haStatus.SecondaryServer.ConnectingClients)
	require.EqualValues(t, 4, haStatus.SecondaryServer.UnackedClients)
	require.EqualValues(t, 6, haStatus.SecondaryServer.UnackedClientsLeft)
	require.EqualValues(t, 8, haStatus.SecondaryServer.AnalyzedPackets)

	// Validate the status of the DHCPv6 pair.
	status = statusList[2].Status.KeaStatus
	require.NotNil(t, status.HaServers)

	haStatus = status.HaServers
	require.NotNil(t, haStatus.PrimaryServer)
	require.NotNil(t, haStatus.SecondaryServer)

	require.Equal(t, "server1", haStatus.Relationship)

	require.EqualValues(t, keaDaemon.ID, haStatus.PrimaryServer.ID)
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
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea daemon to the machine
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     1234,
		Protocol: protocoltype.HTTPS,
	}
	keaDaemon := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{keaAccessPoint})
	err = dbmodel.AddDaemon(db, keaDaemon)
	require.NoError(t, err)
	require.NotZero(t, keaDaemon.ID)

	exampleTime := storkutil.UTCNow().Add(-5 * time.Second)
	keaServices := []dbmodel.Service{
		{
			BaseService: dbmodel.BaseService{},
			HAService: &dbmodel.BaseHAService{
				HAType:                   "dhcp4",
				HAMode:                   "passive-backup",
				Relationship:             "server1",
				PrimaryID:                keaDaemon.ID,
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
		err = dbmodel.AddDaemonToService(db, keaServices[i].ID, keaDaemon)
		require.NoError(t, err)
	}

	params := services.GetDaemonServicesStatusParams{
		ID: keaDaemon.ID,
	}
	rsp := rapi.GetDaemonServicesStatus(ctx, params)

	// Make sure that the response is ok.
	require.IsType(t, &services.GetDaemonServicesStatusOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonServicesStatusOK)
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

	require.EqualValues(t, keaDaemon.ID, haStatus.PrimaryServer.ID)
	require.Equal(t, "primary", haStatus.PrimaryServer.Role)
	require.Len(t, haStatus.PrimaryServer.Scopes, 1)
	require.Contains(t, haStatus.PrimaryServer.Scopes, "server1")
	require.Equal(t, "passive-backup", haStatus.PrimaryServer.State)
	require.GreaterOrEqual(t, haStatus.PrimaryServer.Age, int64(5))
	require.Equal(t, "127.0.0.1", haStatus.PrimaryServer.ControlAddress)
	require.EqualValues(t, keaDaemon.ID, haStatus.PrimaryServer.ID)
	require.NotEmpty(t, haStatus.PrimaryServer.StatusTime.String())
	require.EqualValues(t, -1, haStatus.PrimaryServer.CommInterrupted)
	require.Zero(t, haStatus.PrimaryServer.ConnectingClients)
	require.Zero(t, haStatus.PrimaryServer.UnackedClients)
	require.Zero(t, haStatus.PrimaryServer.UnackedClientsLeft)
	require.Zero(t, haStatus.PrimaryServer.AnalyzedPackets)
}

func TestRestGetDaemonsStats(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// get empty list of daemons
	params := services.GetDaemonsStatsParams{}
	rsp := rapi.GetDaemonsStats(ctx, params)
	require.IsType(t, &services.GetDaemonsStatsOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonsStatsOK)
	require.Zero(t, okRsp.Payload.DhcpDaemonsTotal)
	require.Zero(t, okRsp.Payload.DhcpDaemonsNotOk)
	// add machine
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add kea daemon to machine
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	s1 := dbmodel.NewDaemon(m, daemonname.CA, true, []*dbmodel.AccessPoint{keaAccessPoint})
	s1.KeaDaemon = &dbmodel.KeaDaemon{}
	err = dbmodel.AddDaemon(db, s1)
	require.NoError(t, err)

	// add bind9 daemon to machine
	bind9AccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     4321,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}
	s2 := dbmodel.NewDaemon(m, daemonname.Bind9, false, []*dbmodel.AccessPoint{bind9AccessPoint})
	s2.Bind9Daemon = &dbmodel.Bind9Daemon{}
	err = dbmodel.AddDaemon(db, s2)
	require.NoError(t, err)

	// Add PowerDNS daemon to the machine.
	pdnsAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     5432,
		Key:      "abcd",
		Protocol: protocoltype.HTTPS,
	}
	s3 := dbmodel.NewDaemon(m, daemonname.PDNS, false, []*dbmodel.AccessPoint{pdnsAccessPoint})
	s3.PDNSDaemon = &dbmodel.PDNSDaemon{}
	err = dbmodel.AddDaemon(db, s3)
	require.NoError(t, err)

	// get added app
	params = services.GetDaemonsStatsParams{}
	rsp = rapi.GetDaemonsStats(ctx, params)
	require.IsType(t, &services.GetDaemonsStatsOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonsStatsOK)
	require.EqualValues(t, 1, okRsp.Payload.DhcpDaemonsTotal)
	require.Zero(t, okRsp.Payload.DhcpDaemonsNotOk)
	require.EqualValues(t, 2, okRsp.Payload.DNSDaemonsTotal)
	require.EqualValues(t, 2, okRsp.Payload.DNSDaemonsNotOk)
}

func TestGetDhcpOverview(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// add daemon kea to machine
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "localhost",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	dhcp4Daemon := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{keaAccessPoint})
	dhcp4Daemon.Version = "2.6.9"
	dhcp4Daemon.Monitored = true
	err = dbmodel.AddDaemon(db, dhcp4Daemon)
	require.NoError(t, err)

	dhcp6Daemon := dbmodel.NewDaemon(m, daemonname.DHCPv6, true, []*dbmodel.AccessPoint{keaAccessPoint})
	dhcp6Daemon.Version = "2.6.9"
	dhcp6Daemon.Monitored = false
	err = dbmodel.AddDaemon(db, dhcp6Daemon)
	require.NoError(t, err)

	err = dbmodel.AddSubnet(db, &dbmodel.Subnet{
		Prefix: "3001:fed8::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: nil,
			},
			{
				Stats: dbmodel.Stats{
					"total-addresses": 0,
				},
			},
		},
	})
	require.NoError(t, err)

	controller := gomock.NewController(t)
	mock := NewMockConnectedAgents(controller)
	stats := agentcomm.NewAgentStats()
	stats.IncreaseAgentErrorCount("foo")
	keaStats := stats.GetKeaStats()
	keaStats.IncreaseErrorCountBy(daemonname.CA, 2)
	keaStats.IncreaseErrorCountBy(daemonname.DHCPv4, 5)
	mock.EXPECT().GetConnectedAgentStatsWrapper(gomock.Any(), int64(8080)).DoAndReturn(wrap(stats))

	settings := RestAPISettings{}
	//	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, mock, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// get overview, generally it should be empty
	params := dhcp.GetDhcpOverviewParams{}
	rsp := rapi.GetDhcpOverview(ctx, params)
	require.IsType(t, &dhcp.GetDhcpOverviewOK{}, rsp)
	okRsp := rsp.(*dhcp.GetDhcpOverviewOK)
	require.Len(t, okRsp.Payload.Subnets4.Items, 0)
	require.Len(t, okRsp.Payload.Subnets6.Items, 1)
	require.Nil(t, okRsp.Payload.Subnets6.Items[0].LocalSubnets)
	require.Len(t, okRsp.Payload.SharedNetworks4.Items, 0)
	require.Len(t, okRsp.Payload.SharedNetworks6.Items, 0)
	require.Len(t, okRsp.Payload.DhcpDaemons, 1)

	// only dhcp4 is present
	require.EqualValues(t, "dhcp4", okRsp.Payload.DhcpDaemons[0].Name)
	require.EqualValues(t, 1, okRsp.Payload.DhcpDaemons[0].AgentCommErrors)
	require.EqualValues(t, 2, okRsp.Payload.DhcpDaemons[0].CaCommErrors)
	require.EqualValues(t, 5, okRsp.Payload.DhcpDaemons[0].DaemonCommErrors)

	// HA is not enabled.
	require.False(t, okRsp.Payload.DhcpDaemons[0].HaEnabled)
	require.Empty(t, okRsp.Payload.DhcpDaemons[0].HaOverview)
}

// This test verifies that the overview response includes HA state.
func TestHAInDhcpOverview(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	// Configure the fake control agents to mimic returning a status of
	// two HA services for Kea.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea daemon to the machine
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     1234,
		Protocol: protocoltype.HTTPS,
	}
	keaDaemon := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{keaAccessPoint})
	err = dbmodel.AddDaemon(db, keaDaemon)
	require.NoError(t, err)
	require.NotZero(t, keaDaemon.ID)

	// Create an HA service.
	exampleTime := storkutil.UTCNow().Add(-5 * time.Second)
	keaService := dbmodel.Service{
		BaseService: dbmodel.BaseService{},
		HAService: &dbmodel.BaseHAService{
			HAType:                   "dhcp4",
			HAMode:                   "load-balancing",
			Relationship:             "server1",
			PrimaryID:                keaDaemon.ID,
			PrimaryStatusCollectedAt: exampleTime,
			PrimaryLastState:         "load-balancing",
			SecondaryLastFailoverAt:  exampleTime,
		},
	}

	// Add the service and associate the daemon with that service.
	err = dbmodel.AddService(db, &keaService)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToService(db, keaService.ID, keaDaemon)
	require.NoError(t, err)

	// Get the overview.
	params := dhcp.GetDhcpOverviewParams{}
	rsp := rapi.GetDhcpOverview(ctx, params)
	require.IsType(t, &dhcp.GetDhcpOverviewOK{}, rsp)
	okRsp := rsp.(*dhcp.GetDhcpOverviewOK)
	require.Len(t, okRsp.Payload.DhcpDaemons, 1)

	// Test that the HA specific information was returned.
	require.True(t, okRsp.Payload.DhcpDaemons[0].HaEnabled)
	require.Len(t, okRsp.Payload.DhcpDaemons[0].HaOverview, 1)
	require.Equal(t, "load-balancing", okRsp.Payload.DhcpDaemons[0].HaOverview[0].HaState)
	require.NotEmpty(t, okRsp.Payload.DhcpDaemons[0].HaOverview[0].HaFailureAt.String())
}

// Test that the DHCP Overview is properly generated when the statistic
// table contains the NULL values.
func TestGetDhcpOverviewWithNullStatistics(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeStats(db)

	rapi, _ := NewRestAPI(db, dbSettings)
	ctx := context.Background()
	params := dhcp.GetDhcpOverviewParams{}

	// Act
	err := dbmodel.SetStats(db, dbmodel.Stats{
		dbmodel.StatNameTotalAddresses: nil,
	})
	rsp := rapi.GetDhcpOverview(ctx, params)

	// Assert
	require.NoError(t, err)
	require.IsType(t, &dhcp.GetDhcpOverviewOK{}, rsp)
	okRsp := rsp.(*dhcp.GetDhcpOverviewOK)
	require.EqualValues(t, "<nil>", okRsp.Payload.Dhcp4Stats.TotalAddresses)
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
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea daemon to the machine
	keaAccessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     1234,
		Protocol: protocoltype.HTTP,
	}
	keaDaemon := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{keaAccessPoint})
	keaDaemon.Monitored = true
	err = dbmodel.AddDaemon(db, keaDaemon)
	require.NoError(t, err)
	require.NotZero(t, keaDaemon.ID)

	// get added daemon
	getDaemonParams := services.GetDaemonParams{
		ID: keaDaemon.ID,
	}
	rsp := rapi.GetDaemon(ctx, getDaemonParams)
	require.IsType(t, &services.GetDaemonOK{}, rsp)
	okRsp := rsp.(*services.GetDaemonOK)
	require.Equal(t, keaDaemon.ID, okRsp.Payload.ID)
	require.True(t, okRsp.Payload.Daemon.Monitored) // now it is true

	// setup a user session (UpdateDaemon needs user db object)
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx2, err := rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx2, user)
	require.NoError(t, err)

	// update daemon: change monitored to false
	params := services.UpdateDaemonParams{
		ID: keaDaemon.ID,
		Daemon: services.UpdateDaemonBody{
			Monitored: false,
		},
	}
	rsp = rapi.UpdateDaemon(ctx2, params)
	require.IsType(t, &services.UpdateDaemonOK{}, rsp)

	// get daemon with modified daemon
	rsp = rapi.GetDaemon(ctx2, getDaemonParams)
	require.IsType(t, &services.GetDaemonOK{}, rsp)
	okRsp = rsp.(*services.GetDaemonOK)
	require.Equal(t, keaDaemon.ID, okRsp.Payload.ID)
	require.False(t, okRsp.Payload.Daemon.Monitored) // now it is false
}

// Check if generating and getting server token works.
func TestServerToken(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	settings := RestAPISettings{}

	// Configure the fake control and prepare rest API.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	// setup a user session, it is required to check user role in GetMachinesServerToken
	user, err := dbmodel.GetUserByID(rapi.DB, 1)
	require.NoError(t, err)
	ctx2, err := rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)
	err = rapi.SessionManager.LoginHandler(ctx2, user)
	require.NoError(t, err)

	// get server token but it was not created yet, so error is expected
	params := services.GetMachinesServerTokenParams{}
	rsp := rapi.GetMachinesServerToken(ctx2, params)
	require.IsType(t, &services.GetMachinesServerTokenDefault{}, rsp)
	errRsp := rsp.(*services.GetMachinesServerTokenDefault)
	require.EqualValues(t, "Server internal problem - server token is empty", *errRsp.Payload.Message)

	// generate server token
	serverToken, err := certs.GenerateServerToken(db)
	require.NoError(t, err)
	require.NotEmpty(t, serverToken)

	// get server token
	rsp = rapi.GetMachinesServerToken(ctx2, params)
	require.IsType(t, &services.GetMachinesServerTokenOK{}, rsp)
	okRsp := rsp.(*services.GetMachinesServerTokenOK)
	require.EqualValues(t, serverToken, okRsp.Payload.Token)

	// regenerate server token
	regenParams := services.RegenerateMachinesServerTokenParams{}
	rsp = rapi.RegenerateMachinesServerToken(ctx2, regenParams)
	require.IsType(t, &services.RegenerateMachinesServerTokenOK{}, rsp)
	okRegenRsp := rsp.(*services.RegenerateMachinesServerTokenOK)
	require.NotEmpty(t, okRegenRsp.Payload.Token)
}

// This test verifies that database backend configurations are parsed correctly
// from the Kea configuration and formatted such that they can be returned over
// the REST API.
func TestGetKeaStorages(t *testing.T) {
	configString := `{
        "Dhcp4": {
            "lease-database": {
                "type": "memfile",
                "name": "/tmp/leases4.csv"
            },
            "hosts-database": {
                "type": "mysql",
                "name": "kea-hosts-mysql",
                "host": "mysql.example.org"
            },
            "config-control": {
                "config-databases": [
                    {
                        "type": "mysql",
                        "name": "kea-config-mysql"
                    }
                ]
            },
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_legal_log.so",
                    "parameters": {
                         "path": "/tmp/legal_log.log"
                    }
                }
            ]
        }
    }`
	keaConfig, err := keaconfig.NewConfig([]byte(configString))
	require.NoError(t, err)
	require.NotNil(t, keaConfig)

	files, databases := getKeaStorages(keaConfig)
	require.Len(t, files, 2)
	require.Len(t, databases, 2)

	// Ensure that the lease file is present.
	require.Equal(t, "/tmp/leases4.csv", files[0].Filename)
	require.Equal(t, "Lease file", files[0].Filetype)
	require.True(t, files[0].Persist)

	// Ensure that the forensic logging file is present.
	require.Equal(t, "/tmp/legal_log.log", files[1].Filename)
	require.Equal(t, "Forensic Logging", files[1].Filetype)
	require.True(t, files[1].Persist)

	// Test database configurations.
	for _, d := range databases {
		if d.Database == "kea-hosts-mysql" {
			require.Equal(t, "mysql", d.BackendType)
			require.Equal(t, "mysql.example.org", d.Host)
		} else {
			require.Equal(t, "mysql", d.BackendType)
			require.Equal(t, "kea-config-mysql", d.Database)
			require.Equal(t, "localhost", d.Host)
		}
	}
}

// This test verifies that file backend configurations are parsed correctly
// from the Kea configuration if they use non-persistent storage.
func TestGetKeaStoragesNonPersist(t *testing.T) {
	configString := `{
        "Dhcp4": {
            "lease-database": {
                "type": "memfile",
                "persist": false
            },
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_legal_log.so",
                    "parameters": {
                         "type": "syslog"
                    }
                }
            ]
        }
    }`
	keaConfig, err := keaconfig.NewConfig([]byte(configString))
	require.NoError(t, err)
	require.NotNil(t, keaConfig)

	files, databases := getKeaStorages(keaConfig)
	require.Len(t, files, 2)
	require.Empty(t, databases)

	// Ensure that the lease file is present.
	require.Empty(t, files[0].Filename)
	require.Equal(t, "Lease file", files[0].Filetype)
	require.False(t, files[0].Persist)

	// Ensure that the forensic logging file is present.
	require.Empty(t, files[1].Filename)
	require.Equal(t, "Forensic Logging", files[1].Filetype)
	require.False(t, files[1].Persist)
}

// Test that converting machine with a daemon with nil Kea config doesn't cause panic.
func TestMachineToRestAPIForNilKeaConfig(t *testing.T) {
	// Arrange
	machine := &dbmodel.Machine{ID: 1}
	daemon := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{})
	machine.Daemons = []*dbmodel.Daemon{daemon}

	fa := agentcommtest.NewFakeAgents(nil, nil)
	rapi, err := NewRestAPI(&dbops.DatabaseSettings{}, fa)
	require.NoError(t, err)

	// Act
	restMachine := rapi.machineToRestAPI(*machine)

	// Assert
	require.NotNil(t, restMachine)
	require.Len(t, restMachine.Daemons, 1)
}

// Test conversion of a KeaDaemon to REST API format.
func TestKeaDaemonToRestAPI(t *testing.T) {
	fa := agentcommtest.NewFakeAgents(nil, nil)
	rapi, err := NewRestAPI(&dbops.DatabaseSettings{}, fa)
	require.NoError(t, err)

	machine := &dbmodel.Machine{ID: 2}
	daemon := &dbmodel.Daemon{
		ID:              1,
		Pid:             1234,
		Name:            "dhcp4",
		Active:          true,
		Monitored:       true,
		Version:         "2.1",
		ExtendedVersion: "2.1.x",
		Uptime:          1000,
		ReloadedAt:      time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
		Machine:         machine,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: &dbmodel.KeaConfig{
				Config: func() *keaconfig.Config {
					cfg, _ := keaconfig.NewConfig([]byte(`{
						"Dhcp4": {
							"hooks-libraries": [
								{
									"library": "hook_abc.so"
								},
								{
									"library": "hook_def.so"
								}
							]
						}
					}`))
					return cfg
				}(),
			},
		},
	}
	converted := rapi.daemonToRestAPI(daemon)
	require.NotNil(t, converted)
	require.EqualValues(t, daemon.ID, converted.ID)
	require.EqualValues(t, daemon.Pid, converted.Pid)
	require.Equal(t, daemon.Active, converted.Active)
	require.Equal(t, daemon.Monitored, converted.Monitored)
	require.Equal(t, daemon.Version, converted.Version)
	require.Equal(t, daemon.ExtendedVersion, converted.ExtendedVersion)
	require.EqualValues(t, daemon.Uptime, converted.Uptime)
	require.Equal(t, "2009-11-10T23:00:00.000Z", converted.ReloadedAt.String())
	require.Len(t, converted.Hooks, 2)
	require.Contains(t, converted.Hooks, "hook_abc.so")
	require.Contains(t, converted.Hooks, "hook_def.so")
}

// This test verifies that the lease database configuration storing the
// leases in an SQL database is correctly recognized.
func TestGetKeaStoragesLeaseDatabase(t *testing.T) {
	configString := `{
        "Dhcp4": {
            "lease-database": {
                "type": "postgresql",
                "name": "kea"
            }
        }
    }`
	keaConfig, err := keaconfig.NewConfig([]byte(configString))
	require.NoError(t, err)
	require.NotNil(t, keaConfig)

	files, databases := getKeaStorages(keaConfig)
	require.Empty(t, files, 0)
	require.Len(t, databases, 1)

	require.Equal(t, "postgresql", databases[0].BackendType)
	require.Equal(t, "kea", databases[0].Database)
	require.Equal(t, "localhost", databases[0].Host)
}

// This test verifies that the forensic logging database configuration is
// correctly recognized.
func TestGetKeaStoragesForensicDatabase(t *testing.T) {
	configString := `{
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_legal_log.so",
                    "parameters": {
                         "type": "mysql",
                         "name": "kea"
                    }
                }
            ]
        }
    }`
	keaConfig, err := keaconfig.NewConfig([]byte(configString))
	require.NoError(t, err)
	require.NotNil(t, keaConfig)

	files, databases := getKeaStorages(keaConfig)
	require.Empty(t, files, 0)
	require.Len(t, databases, 1)

	require.Equal(t, "mysql", databases[0].BackendType)
	require.Equal(t, "kea", databases[0].Database)
	require.Equal(t, "localhost", databases[0].Host)
}

// Test that the GetMachineDump returns OK status when the machine exists.
func TestGetMachineDumpOK(t *testing.T) {
	// Arrange
	// Database init
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, m)
	// REST init
	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	ctx := context.Background()
	// User init
	user, _ := dbmodel.GetUserByID(rapi.DB, 1)
	ctx, _ = rapi.SessionManager.Load(ctx, "")
	_ = rapi.SessionManager.LoginHandler(ctx, user)
	// Request init
	params := services.GetMachineDumpParams{
		ID: m.ID,
	}

	// Act
	rsp := rapi.GetMachineDump(ctx, params)

	// Assert
	require.IsType(t, &services.GetMachineDumpOK{}, rsp)
}

// Test that the GetMachineDump returns a tarball file.
func TestGetMachineDumpReturnsTarball(t *testing.T) {
	// Arrange
	// Database init
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, m)
	// REST init
	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	ctx := context.Background()
	// User init
	user, _ := dbmodel.GetUserByID(rapi.DB, 1)
	ctx, _ = rapi.SessionManager.Load(ctx, "")
	_ = rapi.SessionManager.LoginHandler(ctx, user)
	// Request init
	params := services.GetMachineDumpParams{
		ID: m.ID,
	}

	// Act
	rsp := rapi.GetMachineDump(ctx, params).(*services.GetMachineDumpOK)
	filenames, err := storkutil.ListFilesInTarball(rsp.Payload)

	// Assert
	require.NoError(t, err)
	require.NotZero(t, len(filenames))
}

// Test that the GetMachineDump returns HTTP 404 Not Found when the machine is missing.
func TestGetMachineDumpNotExists(t *testing.T) {
	// Arrange
	// Database init
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	// REST init
	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	ctx := context.Background()
	// User init
	user, _ := dbmodel.GetUserByID(rapi.DB, 1)
	ctx, _ = rapi.SessionManager.Load(ctx, "")
	_ = rapi.SessionManager.LoginHandler(ctx, user)
	// Request init
	params := services.GetMachineDumpParams{
		ID: 42,
	}

	// Act
	rsp := rapi.GetMachineDump(ctx, params)

	// Assert
	defaultRsp, ok := rsp.(*services.GetMachineDumpDefault)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
}

// Test that the GetMachineDump returns a file with expected filename.
func TestGetMachineDumpReturnsExpectedFilename(t *testing.T) {
	// Arrange
	// Database init
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	m := &dbmodel.Machine{
		ID:        42,
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, m)
	// REST init
	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	ctx := context.Background()
	// User init
	user, _ := dbmodel.GetUserByID(rapi.DB, 1)
	ctx, _ = rapi.SessionManager.Load(ctx, "")
	_ = rapi.SessionManager.LoginHandler(ctx, user)
	// Request init
	params := services.GetMachineDumpParams{
		ID: m.ID,
	}

	headerPattern := regexp.MustCompile("attachment; filename=\"(.*)\"")

	// Act
	rsp := rapi.GetMachineDump(ctx, params).(*services.GetMachineDumpOK)
	headerValue := rsp.ContentDisposition
	submatches := headerPattern.FindStringSubmatch(headerValue)

	// Assert
	require.Len(t, submatches, 2)
	filename := submatches[1]
	rest, timestamp, extension, err := testutil.ParseTimestampFilename(filename)
	require.NoError(t, err)
	require.Contains(t, rest, "machine-42")
	require.EqualValues(t, extension, ".tar.gz")
	require.LessOrEqual(t, time.Now().UTC().Sub(timestamp).Seconds(), float64(10))
}

// Test that only super-administrators can fetch the authentication key of the
// access point.
func TestGetAccessPointKeyIsRestrictedToSuperAdmins(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := &RestAPISettings{}
	rapi, _ := NewRestAPI(settings, dbSettings, db)

	// Create an admin user.
	ctx, _ := rapi.SessionManager.Load(context.Background(), "")
	user := &dbmodel.SystemUser{
		Login:    "foo",
		Name:     "baz",
		Lastname: "boz",
		Groups:   []*dbmodel.SystemGroup{{ID: dbmodel.AdminGroupID}},
	}
	_, _ = dbmodel.CreateUser(rapi.DB, user)
	_ = rapi.SessionManager.LoginHandler(ctx, user)

	// Act
	rsp := rapi.GetAccessPointKey(ctx, services.GetAccessPointKeyParams{
		DaemonID: 42,
		Type:     string(dbmodel.AccessPointControl),
	})

	// Assert
	defaultRsp, ok := rsp.(*services.GetAccessPointKeyDefault)
	require.True(t, ok)
	require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
}

// Test that the HTTP 500 Internal Server Error status is returned if the
// database is not available.
func TestGetAccessPointKeyForInvalidDatabase(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)

	settings := &RestAPISettings{}
	rapi, _ := NewRestAPI(settings, dbSettings, db)

	ctx, _ := rapi.SessionManager.Load(context.Background(), "")
	user, _ := dbmodel.GetUserByID(rapi.DB, 1)
	_ = rapi.SessionManager.LoginHandler(ctx, user)

	// Act
	teardown()
	rsp := rapi.GetAccessPointKey(ctx, services.GetAccessPointKeyParams{
		DaemonID: 42,
		Type:     string(dbmodel.AccessPointControl),
	})

	// Assert
	defaultRsp, ok := rsp.(*services.GetAccessPointKeyDefault)
	require.True(t, ok)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
}

// Test that the HTTP 404 Not Found status is returned if the access point
// doesn't exist.
func TestGetAccessPointKeyForMissingEntry(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := &RestAPISettings{}
	rapi, _ := NewRestAPI(settings, dbSettings, db)

	ctx, _ := rapi.SessionManager.Load(context.Background(), "")
	user, _ := dbmodel.GetUserByID(rapi.DB, 1)
	_ = rapi.SessionManager.LoginHandler(ctx, user)

	// Act
	rsp := rapi.GetAccessPointKey(ctx, services.GetAccessPointKeyParams{
		DaemonID: 42,
		Type:     string(dbmodel.AccessPointControl),
	})

	// Assert
	defaultRsp, ok := rsp.(*services.GetAccessPointKeyDefault)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
}

// Test that the authentication key of a given access point is fetched properly.
func TestGetAccessPointKey(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := &RestAPISettings{}
	rapi, _ := NewRestAPI(settings, dbSettings, db)

	ctx, _ := rapi.SessionManager.Load(context.Background(), "")
	user, _ := dbmodel.GetUserByID(rapi.DB, 1)
	_ = rapi.SessionManager.LoginHandler(ctx, user)

	machine := &dbmodel.Machine{Address: "localhost", AgentPort: 8080}
	_ = dbmodel.AddMachine(db, machine)

	accessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     8080,
		Key:      "secret",
		Protocol: protocoltype.HTTPS,
	}
	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{accessPoint})
	daemon.Bind9Daemon = &dbmodel.Bind9Daemon{}
	_ = dbmodel.AddDaemon(db, daemon)

	// Act
	rsp := rapi.GetAccessPointKey(ctx, services.GetAccessPointKeyParams{
		DaemonID: daemon.ID,
		Type:     string(dbmodel.AccessPointControl),
	})

	// Assert
	okRsp, ok := rsp.(*services.GetAccessPointKeyOK)
	require.True(t, ok)
	require.EqualValues(t, "secret", okRsp.Payload)
}

// Helper function to store and defer restore
// original path of versions.json file.
func RememberVersionsJSONPath() func() {
	originalPath := VersionsJSONPath

	return func() {
		VersionsJSONPath = originalPath
	}
}

// Test that an error is returned by getOfflineVersionsJSON
// if the versions.json file doesn't exist.
func TestGetOfflineVersionsJSONErrorNoSuchFile(t *testing.T) {
	// Arrange
	restoreJSONPath := RememberVersionsJSONPath()
	defer restoreJSONPath()
	sb := testutil.NewSandbox()
	defer sb.Close()
	VersionsJSONPath = path.Join(sb.BasePath, "not-exists.json")

	// Act
	appsVersions, err := getOfflineVersionsJSON()

	// Assert
	require.Nil(t, appsVersions)
	require.Error(t, err)
	require.ErrorContains(t, err, "problem opening the JSON file")
	require.ErrorContains(t, err, "no such file")
}

// Test that getOnlineVersionsJSON sends appropriate HTTP GET request
// and it returns the data received from the server.
func TestGetOnlineVersionsJSON(t *testing.T) {
	// Arrange
	restoreJSONPath := RememberVersionsJSONPath()
	defer restoreJSONPath()

	testBody := []byte(`{
		"date": "2024-10-03",
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-31",
					"eolDate": "2026-07-01"
				}
			],
			"latestDev": {
			"version": "2.7.3",
			"releaseDate": "2024-09-25"
			}
		},
		"stork": {
			"latestDev": {
			"version": "1.19.0",
			"releaseDate": "2024-10-02"
			},
			"latestSecure": [
			]
		},
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.30",
					"releaseDate": "2024-09-18",
					"eolDate": "2026-07-01",
					"ESV": "true"
				}
			],
			"latestDev": {
			"version": "9.21.1",
			"releaseDate": "2024-09-18"
			}
		}
	}`)

	// Prepare test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hAccept := r.Header.Get("Accept")
		require.NotEmpty(t, hAccept)
		require.Contains(t, hAccept, "application/json")
		hUserAgent := r.Header.Get("User-Agent")
		require.NotEmpty(t, hUserAgent)
		require.Regexp(t, `^ISC Stork / \d+\.\d+\.\d+ built on .+$`, hUserAgent)
		w.WriteHeader(http.StatusOK)
		w.Write(testBody)
	}))
	defer ts.Close()
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := &RestAPISettings{VersionsURL: ts.URL}
	rapi, _ := NewRestAPI(settings, dbSettings, db)

	// Act
	appsVersions, err := rapi.getOnlineVersionsJSON()

	// Assert
	require.NotNil(t, appsVersions)
	require.NoError(t, err)
	require.Equal(t, "9.21.1", *appsVersions.Bind9.LatestDev.Version)
	require.Equal(t, "2.7.3", *appsVersions.Kea.LatestDev.Version)
	require.Equal(t, "1.19.0", *appsVersions.Stork.LatestDev.Version)
}

// Test that getOnlineVersionsJSON returns an error when the online versions.json URL is incorrect.
func TestGetOnlineVersionsJSONSendingError(t *testing.T) {
	// Arrange
	restoreJSONPath := RememberVersionsJSONPath()
	defer restoreJSONPath()
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := &RestAPISettings{VersionsURL: "foobar"}
	rapi, _ := NewRestAPI(settings, dbSettings, db)

	// Act
	appsVersions, err := rapi.getOnlineVersionsJSON()

	// Assert
	require.Nil(t, appsVersions)
	require.Error(t, err)
	require.ErrorContains(t, err, "problem sending HTTP GET request to")
}

// Test that an error is returned by unmarshalVersionsJSONData
// if the versions.json file content is truncated.
func TestUnmarshalVersionsJSONDataTruncatedVersionsJSONError(t *testing.T) {
	// Arrange
	bytes := []byte(`"date": "2024-12-08"`)

	// Act
	appsVersions, err := unmarshalVersionsJSONData(&bytes, models.VersionsDataSourceOffline)

	// Assert
	require.Nil(t, appsVersions)
	require.Error(t, err)
	require.ErrorContains(t, err, "problem unmarshalling contents of the offline JSON file")
}

// Test that an error is returned by unmarshalVersionsJSONData
// if the versions.json BIND 9 metadata has wrong format.
func TestUnmarshalVersionsJSONDataBindMetadataError(t *testing.T) {
	// Arrange
	bytes := []byte(`{
		"date": "2024-10-03",
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-31",
					"eolDate": "2026-07-01"
				}
			],
			"latestDev": {
			"version": "2.7.3",
			"releaseDate": "2024-09-25"
			}
		},
		"stork": {
			"latestDev": {
			"version": "1.19.0",
			"releaseDate": "2024-10-02"
			},
			"latestSecure": [
			]
		},
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.30",
					"releaseDate": "202a-09-18",
					"eolDate": "2026-07-01",
					"ESV": "true"
				}
			],
			"latestDev": {
			"version": "9.21.1",
			"releaseDate": "2024-09-18"
			}
		}
	}`)

	// Act
	appsVersions, err := unmarshalVersionsJSONData(&bytes, models.VersionsDataSourceOffline)

	// Assert
	require.Nil(t, appsVersions)
	require.Error(t, err)
	require.ErrorContains(t, err, "problem converting BIND 9 data")
}

// Test that an error is returned by unmarshalVersionsJSONData
// if the versions.json Kea metadata has wrong format.
func TestUnmarshalVersionsJSONDataKeaMetadataError(t *testing.T) {
	// Arrange
	bytes := []byte(`{
		"date": "2024-10-03",
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-31",
					"eolDate": "026-07-01"
				}
			],
			"latestDev": {
			"version": "2.7.3",
			"releaseDate": "2024-09-25"
			}
		},
		"stork": {
			"latestDev": {
			"version": "1.19.0",
			"releaseDate": "2024-10-02"
			},
			"latestSecure": [
			]
		},
		"bind9": {
			"currentStable": [
				{
					"version": "9.20.2",
					"releaseDate": "2024-09-18",
					"eolDate": "2028-07-01"
				}
			],
			"latestDev": {
			"version": "9.21.1",
			"releaseDate": "2024-09-18"
			}
		}
	}`)

	// Act
	appsVersions, err := unmarshalVersionsJSONData(&bytes, models.VersionsDataSourceOffline)

	// Assert
	require.Nil(t, appsVersions)
	require.Error(t, err)
	require.ErrorContains(t, err, "problem converting Kea data")
}

// Test that an error is returned by unmarshalVersionsJSONData
// if the versions.json Stork metadata has wrong format.
func TestUnmarshalVersionsJSONDataStorkMetadataError(t *testing.T) {
	// Arrange
	bytes := []byte(`{
		"date": "2024-10-03",
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-31",
					"eolDate": "2026-07-01"
				}
			],
			"latestDev": {
			"version": "2.7.3",
			"releaseDate": "2024-09-25"
			}
		},
		"stork": {
			"latestDev": {
			"version": "1.19.0",
			"releaseDate": "z024-10-02"
			},
			"latestSecure": [
			]
		},
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.30",
					"releaseDate": "2024-09-18",
					"eolDate": "2026-07-01",
					"ESV": "true"
				}
			],
			"latestDev": {
			"version": "9.21.1",
			"releaseDate": "2024-09-18"
			}
		}
	}`)

	// Act
	appsVersions, err := unmarshalVersionsJSONData(&bytes, models.VersionsDataSourceOffline)

	// Assert
	require.Nil(t, appsVersions)
	require.Error(t, err)
	require.ErrorContains(t, err, "problem converting Stork data")
}

// Test that an error is returned by unmarshalVersionsJSONData
// if the versions.json date field has wrong format.
func TestUnmarshalVersionsJSONDataDateError(t *testing.T) {
	// Arrange
	bytes := []byte(`{
		"date": "024-10-03",
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-31",
					"eolDate": "2026-07-01"
				}
			],
			"latestDev": {
			"version": "2.7.3",
			"releaseDate": "2024-09-25"
			}
		},
		"stork": {
			"latestDev": {
			"version": "1.19.0",
			"releaseDate": "2024-10-02"
			},
			"latestSecure": [
			]
		},
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.30",
					"releaseDate": "2024-09-18",
					"eolDate": "2026-07-01",
					"ESV": "true"
				}
			],
			"latestDev": {
			"version": "9.21.1",
			"releaseDate": "2024-09-18"
			}
		}
	}`)

	// Act
	appsVersions, err := unmarshalVersionsJSONData(&bytes, models.VersionsDataSourceOffline)

	// Assert
	require.Nil(t, appsVersions)
	require.Error(t, err)
	require.ErrorContains(t, err, "problem parsing date")
}

// Test that information about current ISC software versions is returned
// via the API. Getting online versions.json fails, so there should be
// fallback to offline mode.
func TestGetSoftwareVersionsOffline(t *testing.T) {
	// Arrange
	restoreJSONPath := RememberVersionsJSONPath()
	defer restoreJSONPath()
	sb := testutil.NewSandbox()
	defer sb.Close()
	content := `{
		"date": "2024-10-03",
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-31",
					"eolDate": "2026-07-01"
				},
				{
					"version": "2.4.1",
					"releaseDate": "2023-11-29",
					"eolDate": "2025-07-01"
				}
			],
			"latestDev": {
			"version": "2.7.3",
			"releaseDate": "2024-09-25"
			}
		},
		"stork": {
			"latestDev": {
			"version": "1.19.0",
			"releaseDate": "2024-10-02"
			},
			"latestSecure": [
				{
					"version": "1.15.1",
					"releaseDate": "2024-03-27"
				}
			]
		},
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.30",
					"releaseDate": "2024-09-18",
					"eolDate": "2026-07-01",
					"ESV": "true"
				},
				{
					"version": "9.20.2",
					"releaseDate": "2024-09-18",
					"eolDate": "2028-07-01"
				}
			],
			"latestDev": {
			"version": "9.21.1",
			"releaseDate": "2024-09-18"
			}
		}
	}`
	VersionsJSONPath, _ = sb.Write("versions.json", content)

	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbmodel.InitializeSettings(db, 0)

	settings := &RestAPISettings{VersionsURL: "foobar"}
	rapi, _ := NewRestAPI(settings, dbSettings, db)
	ctx, _ := rapi.SessionManager.Load(context.Background(), "")

	// Act
	rsp := rapi.GetSoftwareVersions(ctx, general.GetSoftwareVersionsParams{})

	// Assert
	okRsp, ok := rsp.(*general.GetSoftwareVersionsOK)
	require.True(t, ok)
	require.Equal(t, "2024-10-03", okRsp.Payload.Date.String())
	require.NotNil(t, okRsp.Payload.DataSource)
	require.EqualValues(t, "offline", okRsp.Payload.DataSource)
	require.NotNil(t, okRsp.Payload.Bind9)
	require.NotNil(t, okRsp.Payload.Kea)
	require.NotNil(t, okRsp.Payload.Stork)
	require.Equal(t, "2.7.3", *okRsp.Payload.Kea.LatestDev.Version)
	require.Equal(t, int64(2), okRsp.Payload.Kea.LatestDev.Major)
	require.Equal(t, int64(7), okRsp.Payload.Kea.LatestDev.Minor)
	require.Equal(t, "2.4.1", okRsp.Payload.Kea.SortedStableVersions[0])
	require.Equal(t, "2.6.1", okRsp.Payload.Kea.SortedStableVersions[1])
	require.Equal(t, "1.19.0", *okRsp.Payload.Stork.LatestDev.Version)
	require.Equal(t, "9.21.1", *okRsp.Payload.Bind9.LatestDev.Version)
}

// Test that information about current ISC software versions is returned
// via the API using online versions.json file data.
func TestGetSoftwareVersionsOnline(t *testing.T) {
	// Arrange
	restoreJSONPath := RememberVersionsJSONPath()
	defer restoreJSONPath()
	sb := testutil.NewSandbox()
	defer sb.Close()
	offlineContent := `{
		"date": "2024-10-03",
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-31",
					"eolDate": "2026-07-01"
				},
				{
					"version": "2.4.1",
					"releaseDate": "2023-11-29",
					"eolDate": "2025-07-01"
				}
			],
			"latestDev": {
			"version": "2.7.3",
			"releaseDate": "2024-09-25"
			}
		},
		"stork": {
			"latestDev": {
			"version": "1.19.0",
			"releaseDate": "2024-10-02"
			},
			"latestSecure": [
				{
					"version": "1.15.1",
					"releaseDate": "2024-03-27"
				}
			]
		},
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.30",
					"releaseDate": "2024-09-18",
					"eolDate": "2026-07-01",
					"ESV": "true"
				},
				{
					"version": "9.20.2",
					"releaseDate": "2024-09-18",
					"eolDate": "2028-07-01"
				}
			],
			"latestDev": {
			"version": "9.21.1",
			"releaseDate": "2024-09-18"
			}
		}
	}`
	onlineContent := `{
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.31",
					"releaseDate": "2024-10-01",
					"eolDate": "2026-07-01",
					"ESV": "true"
				},
				{
					"version": "9.20.3",
					"releaseDate": "2024-10-01",
					"eolDate": "2028-07-01"
				}
			],
			"latestDev": {
				"version": "9.21.2",
				"releaseDate": "2024-10-01"
			},
			"latestSecure": []
		},
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-01",
					"eolDate": "2026-07-01"
				},
				{
					"version": "2.4.1",
					"releaseDate": "2023-11-01",
					"eolDate": "2025-07-01"
				}
			],
			"latestDev": {
				"version": "2.7.4",
				"releaseDate": "2024-10-01"
			},
			"latestSecure": []
		},
		"stork": {
			"currentStable": [
				{
					"version": "2.0.0",
					"releaseDate": "2024-11-01",
					"eolDate": "2025-07-01"
				}
			],
			"latestDev": {
				"version": "1.19.0",
				"releaseDate": "2024-10-01"
			},
			"latestSecure": []
		},
		"date": "2024-12-05"
	}`
	// Prepare test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(onlineContent))
	}))
	defer ts.Close()
	VersionsJSONPath, _ = sb.Write("versions.json", offlineContent)

	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbmodel.InitializeSettings(db, 0)

	settings := &RestAPISettings{VersionsURL: ts.URL}
	rapi, _ := NewRestAPI(settings, dbSettings, db)
	ctx, _ := rapi.SessionManager.Load(context.Background(), "")

	// Act
	rsp := rapi.GetSoftwareVersions(ctx, general.GetSoftwareVersionsParams{})

	// Assert
	okRsp, ok := rsp.(*general.GetSoftwareVersionsOK)
	require.True(t, ok)
	require.Equal(t, "2024-12-05", okRsp.Payload.Date.String())
	require.NotNil(t, okRsp.Payload.DataSource)
	require.EqualValues(t, "online", okRsp.Payload.DataSource)
	require.NotNil(t, okRsp.Payload.Bind9)
	require.NotNil(t, okRsp.Payload.Kea)
	require.NotNil(t, okRsp.Payload.Stork)
	require.Equal(t, "2.7.4", *okRsp.Payload.Kea.LatestDev.Version)
	require.Equal(t, int64(2), okRsp.Payload.Kea.LatestDev.Major)
	require.Equal(t, int64(7), okRsp.Payload.Kea.LatestDev.Minor)
	require.Equal(t, "2.4.1", okRsp.Payload.Kea.SortedStableVersions[0])
	require.Equal(t, "2.6.1", okRsp.Payload.Kea.SortedStableVersions[1])
	require.Equal(t, "1.19.0", *okRsp.Payload.Stork.LatestDev.Version)
	require.Equal(t, "9.21.2", *okRsp.Payload.Bind9.LatestDev.Version)
	require.NotNil(t, okRsp.Payload.Stork.CurrentStable)
	require.Len(t, okRsp.Payload.Stork.CurrentStable, 1)
	require.Equal(t, "2.0.0", *okRsp.Payload.Stork.CurrentStable[0].Version)
}

// Test that information about current ISC software versions is returned
// via the API using offline versions.json file data, because online mode is disabled in settings.
func TestGetSoftwareVersionsOnlineDisabled(t *testing.T) {
	// Arrange
	restoreJSONPath := RememberVersionsJSONPath()
	defer restoreJSONPath()
	sb := testutil.NewSandbox()
	defer sb.Close()
	offlineContent := `{
		"date": "2024-10-03",
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-31",
					"eolDate": "2026-07-01"
				},
				{
					"version": "2.4.1",
					"releaseDate": "2023-11-29",
					"eolDate": "2025-07-01"
				}
			],
			"latestDev": {
			"version": "2.7.3",
			"releaseDate": "2024-09-25"
			}
		},
		"stork": {
			"latestDev": {
			"version": "1.19.0",
			"releaseDate": "2024-10-02"
			},
			"latestSecure": [
				{
					"version": "1.15.1",
					"releaseDate": "2024-03-27"
				}
			]
		},
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.30",
					"releaseDate": "2024-09-18",
					"eolDate": "2026-07-01",
					"ESV": "true"
				},
				{
					"version": "9.20.2",
					"releaseDate": "2024-09-18",
					"eolDate": "2028-07-01"
				}
			],
			"latestDev": {
			"version": "9.21.1",
			"releaseDate": "2024-09-18"
			}
		}
	}`
	onlineContent := `{
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.31",
					"releaseDate": "2024-10-01",
					"eolDate": "2026-07-01",
					"ESV": "true"
				},
				{
					"version": "9.20.3",
					"releaseDate": "2024-10-01",
					"eolDate": "2028-07-01"
				}
			],
			"latestDev": {
				"version": "9.21.2",
				"releaseDate": "2024-10-01"
			},
			"latestSecure": []
		},
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-01",
					"eolDate": "2026-07-01"
				},
				{
					"version": "2.4.1",
					"releaseDate": "2023-11-01",
					"eolDate": "2025-07-01"
				}
			],
			"latestDev": {
				"version": "2.7.4",
				"releaseDate": "2024-10-01"
			},
			"latestSecure": []
		},
		"stork": {
			"currentStable": [
				{
					"version": "2.0.0",
					"releaseDate": "2024-11-01",
					"eolDate": "2025-07-01"
				}
			],
			"latestDev": {
				"version": "1.19.0",
				"releaseDate": "2024-10-01"
			},
			"latestSecure": []
		},
		"date": "2024-12-05"
	}`
	// Prepare test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(onlineContent))
	}))
	defer ts.Close()
	VersionsJSONPath, _ = sb.Write("versions.json", offlineContent)

	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbmodel.InitializeSettings(db, 0)
	err := dbmodel.SetSettingBool(db, "enable_online_software_versions", false)
	require.NoError(t, err)

	settings := &RestAPISettings{VersionsURL: ts.URL}
	rapi, _ := NewRestAPI(settings, dbSettings, db)
	ctx, _ := rapi.SessionManager.Load(context.Background(), "")

	// Act
	rsp := rapi.GetSoftwareVersions(ctx, general.GetSoftwareVersionsParams{})

	// Assert
	okRsp, ok := rsp.(*general.GetSoftwareVersionsOK)
	require.True(t, ok)
	require.Equal(t, "2024-10-03", okRsp.Payload.Date.String())
	require.NotNil(t, okRsp.Payload.DataSource)
	require.EqualValues(t, "offline", okRsp.Payload.DataSource)
	require.NotNil(t, okRsp.Payload.Bind9)
	require.NotNil(t, okRsp.Payload.Kea)
	require.NotNil(t, okRsp.Payload.Stork)
	require.Equal(t, "2.7.3", *okRsp.Payload.Kea.LatestDev.Version)
	require.Equal(t, int64(2), okRsp.Payload.Kea.LatestDev.Major)
	require.Equal(t, int64(7), okRsp.Payload.Kea.LatestDev.Minor)
	require.Equal(t, "2.4.1", okRsp.Payload.Kea.SortedStableVersions[0])
	require.Equal(t, "2.6.1", okRsp.Payload.Kea.SortedStableVersions[1])
	require.Equal(t, "1.19.0", *okRsp.Payload.Stork.LatestDev.Version)
	require.Equal(t, "9.21.1", *okRsp.Payload.Bind9.LatestDev.Version)
}

// Test that information about current ISC software versions is returned
// via the API when some of the JSON values are empty.
func TestGetSoftwareVersionsSomeValuesEmpty(t *testing.T) {
	// Arrange
	restoreJSONPath := RememberVersionsJSONPath()
	defer restoreJSONPath()
	sb := testutil.NewSandbox()
	defer sb.Close()
	content := `{
		"date": "2024-10-03",
		"kea": {
			"currentStable": [
				{
					"version": "2.6.1",
					"releaseDate": "2024-07-31",
					"eolDate": "2026-07-01"
				},
				{
					"version": "2.4.1",
					"releaseDate": "2023-11-29",
					"eolDate": "2025-07-01"
				}
			],
			"latestDev": {},
			"latestSecure" : []
		},
		"stork": {
			"latestDev": {
			"version": "1.19.0",
			"releaseDate": "2024-10-02"
			},
			"latestSecure": [
				{
					"version": "1.15.1",
					"releaseDate": "2024-03-27"
				}
			],
			"currentStable": []
		},
		"bind9": {
			"currentStable": [
				{
					"version": "9.18.30",
					"releaseDate": "2024-09-18",
					"eolDate": "2026-07-01",
					"ESV": "true"
				},
				{
					"version": "9.20.2",
					"releaseDate": "2024-09-18",
					"eolDate": "2028-07-01"
				}
			],
			"latestDev": {
			"version": "9.21.1",
			"releaseDate": "2024-09-18"
			}
		}
	}`
	VersionsJSONPath, _ = sb.Write("versions.json", content)

	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbmodel.InitializeSettings(db, 0)
	err := dbmodel.SetSettingBool(db, "enable_online_software_versions", false)
	require.NoError(t, err)

	settings := &RestAPISettings{}
	rapi, _ := NewRestAPI(settings, dbSettings, db)
	ctx, _ := rapi.SessionManager.Load(context.Background(), "")

	// Act
	rsp := rapi.GetSoftwareVersions(ctx, general.GetSoftwareVersionsParams{})

	// Assert
	okRsp, ok := rsp.(*general.GetSoftwareVersionsOK)
	require.True(t, ok)
	require.Equal(t, "2024-10-03", okRsp.Payload.Date.String())
	require.NotNil(t, okRsp.Payload.DataSource)
	require.EqualValues(t, "offline", okRsp.Payload.DataSource)
	require.NotNil(t, okRsp.Payload.Bind9)
	require.NotNil(t, okRsp.Payload.Kea)
	require.NotNil(t, okRsp.Payload.Stork)
	require.Nil(t, okRsp.Payload.Kea.LatestDev)
	require.NotNil(t, okRsp.Payload.Kea.LatestSecure)
	require.Len(t, okRsp.Payload.Kea.LatestSecure, 0)
	require.Equal(t, "2.4.1", okRsp.Payload.Kea.SortedStableVersions[0])
	require.Equal(t, "2.6.1", okRsp.Payload.Kea.SortedStableVersions[1])
	require.Equal(t, "1.19.0", *okRsp.Payload.Stork.LatestDev.Version)
	require.Equal(t, "9.21.1", *okRsp.Payload.Bind9.LatestDev.Version)
	require.Len(t, okRsp.Payload.Stork.CurrentStable, 0)
}

// Test that a list of all authorized machines' ids and apps versions is returned
// via the API.
func TestGetMachinesAppsVersions(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add machines.
	machine1 := &dbmodel.Machine{
		Address:    "machine1.example.org",
		AgentPort:  8080,
		Authorized: true,
		State: dbmodel.MachineState{
			AgentVersion: "9.8.7",
		},
	}
	err := dbmodel.AddMachine(db, machine1)
	require.NoError(t, err)

	// Add Kea daemons with software versions.
	dhcp4Daemon1 := dbmodel.NewDaemon(machine1, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{})
	dhcp4Daemon1.Version = "3.2.2"
	dhcp4Daemon1.KeaDaemon = &dbmodel.KeaDaemon{}
	err = dbmodel.AddDaemon(db, dhcp4Daemon1)
	require.NoError(t, err)

	dhcp6Daemon1 := dbmodel.NewDaemon(machine1, daemonname.DHCPv6, true, []*dbmodel.AccessPoint{})
	dhcp6Daemon1.Version = "3.2.1"
	dhcp6Daemon1.KeaDaemon = &dbmodel.KeaDaemon{}
	err = dbmodel.AddDaemon(db, dhcp6Daemon1)
	require.NoError(t, err)

	machine2 := &dbmodel.Machine{
		Address:    "machine2.example.org",
		AgentPort:  8080,
		Authorized: true,
		State: dbmodel.MachineState{
			AgentVersion: "8.7.6",
		},
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)

	// Add Kea daemons with software versions.
	dhcp4Daemon2 := dbmodel.NewDaemon(machine2, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{})
	dhcp4Daemon2.Version = "1.2.1"
	dhcp4Daemon2.KeaDaemon = &dbmodel.KeaDaemon{}
	err = dbmodel.AddDaemon(db, dhcp4Daemon2)
	require.NoError(t, err)

	dhcp6Daemon2 := dbmodel.NewDaemon(machine2, daemonname.DHCPv6, true, []*dbmodel.AccessPoint{})
	dhcp6Daemon2.Version = "1.2.1"
	dhcp6Daemon2.KeaDaemon = &dbmodel.KeaDaemon{}
	err = dbmodel.AddDaemon(db, dhcp6Daemon2)
	require.NoError(t, err)

	machine3 := &dbmodel.Machine{
		Address:    "machine3.example.org",
		AgentPort:  8080,
		Authorized: false,
	}
	err = dbmodel.AddMachine(db, machine3)
	require.NoError(t, err)

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, fd)
	require.NoError(t, err)
	ctx := context.Background()

	params := services.GetMachinesVersionsParams{}

	// Act
	rsp := rapi.GetMachinesVersions(ctx, params)
	machines := rsp.(*services.GetMachinesVersionsOK).Payload
	require.EqualValues(t, machines.Total, 2)

	// Ensure that the returned machines are in a coherent order.
	sort.Slice(machines.Items, func(i, j int) bool {
		return machines.Items[i].ID < machines.Items[j].ID
	})

	// Assert
	// Validate the returned data.
	require.Equal(t, int64(2), machines.Total)
	require.Equal(t, machine1.ID, machines.Items[0].ID)
	require.NotNil(t, machines.Items[0].Address)
	require.Equal(t, machine1.Address, machines.Items[0].Address)
	require.Equal(t, machine2.ID, machines.Items[1].ID)
	require.NotNil(t, machines.Items[1].Address)
	require.Equal(t, machine2.Address, machines.Items[1].Address)
	require.NotNil(t, machines.Items[0].Daemons)
	require.NotNil(t, machines.Items[1].Daemons)
	require.Equal(t, "3.2.2", machines.Items[0].Daemons[0].Version)
	require.Equal(t, "9.8.7", machines.Items[0].AgentVersion)
	require.Equal(t, "3.2.2", machines.Items[0].Daemons[0].Version)
	require.Equal(t, "3.2.1", machines.Items[0].Daemons[1].Version)
	require.True(t, machines.Items[0].Daemons[0].Active)
	require.Equal(t, "1.2.1", machines.Items[1].Daemons[0].Version)
	require.Equal(t, "8.7.6", machines.Items[1].AgentVersion)
	require.Equal(t, "1.2.1", machines.Items[1].Daemons[0].Version)
	require.Equal(t, "1.2.1", machines.Items[1].Daemons[1].Version)
	require.True(t, machines.Items[1].Daemons[0].Active)
}
