package restservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	apps "isc.org/stork/server/apps"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
)

func mockStatusError(commandName string, cmdResponses []interface{}) {
	command := keactrl.NewCommand(commandName, []string{"dhcp4"}, nil)
	json := `[
        {
            "result": 1,
            "text": "unable to communicate with the daemon"
        }
    ]`
	_ = keactrl.UnmarshalResponseList(command, []byte(json), cmdResponses[0])
}

// This function creates multiple hosts used in tests which fetch and
// filter hosts.
func addTestHosts(t *testing.T, db *pg.DB) (hosts []dbmodel.Host, apps []dbmodel.App) {
	// Add two apps.
	for i := 0; i < 2; i++ {
		m := &dbmodel.Machine{
			ID:        0,
			Address:   "cool.example.org",
			AgentPort: int64(8080 + i),
		}
		err := dbmodel.AddMachine(db, m)
		require.NoError(t, err)

		accessPoints := []*dbmodel.AccessPoint{}
		accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", int64(1234+i), true)

		a := dbmodel.App{
			ID:           0,
			MachineID:    m.ID,
			Type:         dbmodel.AppTypeKea,
			Name:         fmt.Sprintf("dhcp-server%d", i),
			Active:       true,
			AccessPoints: accessPoints,
			Daemons: []*dbmodel.Daemon{
				dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true),
				dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv6, true),
			},
		}
		err = a.Daemons[0].SetConfigFromJSON(`{
            "Dhcp4": {
                "subnet4": [
                    {
                        "id": 111,
                        "subnet": "192.0.2.0/24"
                    }
                ]
            }
        }`)
		require.NoError(t, err)

		err = a.Daemons[1].SetConfigFromJSON(`{
            "Dhcp6": {
                "subnet6": [
                    {
                        "id": 222,
                        "subnet": "2001:db8:1::/64"
                    }
                ]
            }
        }`)
		require.NoError(t, err)
		apps = append(apps, a)
	}

	subnets := []dbmodel.Subnet{
		{
			ID:     1,
			Prefix: "192.0.2.0/24",
		},
		{
			ID:     2,
			Prefix: "2001:db8:1::/64",
		},
	}
	for i, s := range subnets {
		subnet := s
		err := dbmodel.AddSubnet(db, &subnet)
		require.NoError(t, err)
		require.NotZero(t, subnet.ID)
		subnets[i] = subnet
	}

	hosts = []dbmodel.Host{
		{
			SubnetID: 1,
			Hostname: "first.example.org",
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
				{
					Type:  "circuit-id",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "192.0.2.4",
				},
				{
					Address: "192.0.2.5",
				},
			},
		},
		{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{2, 3, 4, 5, 6, 7},
				},
				{
					Type:  "circuit-id",
					Value: []byte{2, 3, 4, 5},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "192.0.2.6",
				},
				{
					Address: "192.0.2.7",
				},
			},
		},
		{
			SubnetID: 2,
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "2001:db8:1::1",
				},
			},
		},
		{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "duid",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "2001:db8:1::2",
				},
			},
		},
		{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "duid",
					Value: []byte{2, 2, 2, 2},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "3000::/48",
				},
			},
		},
	}

	// Add apps to the database.
	for i, a := range apps {
		app := a
		_, err := dbmodel.AddApp(db, &app)
		require.NoError(t, err)
		require.NotZero(t, app.ID)
		// Associate the daemons with the subnets.
		for j := range apps[i].Daemons {
			err = dbmodel.AddDaemonToSubnet(db, &subnets[j], apps[i].Daemons[j])
			require.NoError(t, err)
		}
		apps[i] = app
	}

	// Add hosts to the database.
	for i, h := range hosts {
		host := h
		err := dbmodel.AddHost(db, &host)
		require.NoError(t, err)
		require.NotZero(t, host.ID)
		hosts[i] = host
	}
	return hosts, apps
}

// Test that all hosts can be fetched without filtering.
func TestGetHostsNoFiltering(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts, apps := addTestHosts(t, db)

	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, "config")
	require.NoError(t, err)
	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[1].Daemons[0].ID, "config")
	require.NoError(t, err)

	params := dhcp.GetHostsParams{}
	rsp := rapi.GetHosts(ctx, params)
	require.IsType(t, &dhcp.GetHostsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostsOK)
	require.Len(t, okRsp.Payload.Items, 5)
	require.EqualValues(t, 5, okRsp.Payload.Total)

	items := okRsp.Payload.Items
	require.NotNil(t, items)

	// There should be a total of 5 hosts, 4 of them including IP address
	// reservations and 1 with a prefix reservation.
	require.Len(t, items, 5)
	for i := range items {
		require.NotNil(t, items[i])
		require.NotZero(t, items[i].ID)
		require.EqualValues(t, hosts[i].SubnetID, items[i].SubnetID)

		// Check that the host identifiers types match.
		require.EqualValues(t, len(items[i].HostIdentifiers), len(hosts[i].HostIdentifiers))
		for j := range items[i].HostIdentifiers {
			require.NotNil(t, items[i].HostIdentifiers[j])
			require.EqualValues(t, hosts[i].HostIdentifiers[j].Type, items[i].HostIdentifiers[j].IDType)
		}

		// The total number of reservations, which includes both address and
		// prefix reservations should be equal to the number of reservations for
		// a given host.
		require.EqualValues(t, len(hosts[i].IPReservations),
			len(items[i].AddressReservations)+len(items[i].PrefixReservations))

		// Walk over the address and prefix reservations for a host.
		for _, ips := range [][]*models.IPReservation{items[i].AddressReservations, items[i].PrefixReservations} {
			for j, resrv := range ips {
				require.NotNil(t, resrv)
				require.EqualValues(t, hosts[i].IPReservations[j].Address, resrv.Address)
			}
		}
	}

	// The identifiers should have been converted to hex values.
	require.EqualValues(t, "01:02:03:04:05:06", items[0].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "01:02:03:04", items[0].HostIdentifiers[1].IDHexValue)
	require.EqualValues(t, "02:03:04:05:06:07", items[1].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "02:03:04:05", items[1].HostIdentifiers[1].IDHexValue)
	require.EqualValues(t, "01:02:03:04:05:06", items[2].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "01:02:03:04", items[3].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "02:02:02:02", items[4].HostIdentifiers[0].IDHexValue)

	require.Equal(t, "192.0.2.0/24", items[0].SubnetPrefix)
	require.Empty(t, items[1].SubnetPrefix)
	require.NotNil(t, "2001:db8:1::/64", items[2].SubnetPrefix)
	require.Empty(t, items[3].SubnetPrefix)
	require.Empty(t, items[4].SubnetPrefix)

	// Hosts
	require.Equal(t, "first.example.org", items[0].Hostname)

	// The first host should be associated with two apps.
	require.Len(t, items[0].LocalHosts, 2)
	require.NotNil(t, items[0].LocalHosts[0])
	require.EqualValues(t, apps[0].ID, items[0].LocalHosts[0].AppID)
	require.Equal(t, "config", items[0].LocalHosts[0].DataSource)
	require.Equal(t, "dhcp-server0", items[0].LocalHosts[0].AppName)
	require.NotNil(t, items[0].LocalHosts[1])
	require.EqualValues(t, apps[1].ID, items[0].LocalHosts[1].AppID)
	require.Equal(t, "config", items[0].LocalHosts[1].DataSource)
	require.Equal(t, "dhcp-server1", items[0].LocalHosts[1].AppName)
}

// Test that hosts can be filtered by subnet ID.
func TestGetHostsBySubnetID(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	_, _ = addTestHosts(t, db)

	subnetID := int64(2)
	params := dhcp.GetHostsParams{
		SubnetID: &subnetID,
	}
	rsp := rapi.GetHosts(ctx, params)
	require.IsType(t, &dhcp.GetHostsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostsOK)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Total)
}

// Test that hosts can be filtered by text.
func TestGetHostsWithFiltering(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	_, _ = addTestHosts(t, db)

	filteringText := "2001:db"
	params := dhcp.GetHostsParams{
		Text: &filteringText,
	}
	rsp := rapi.GetHosts(ctx, params)
	require.IsType(t, &dhcp.GetHostsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostsOK)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Total)
}

// Test that host can be fetched by ID over the REST API.
func TestGetHost(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts, _ := addTestHosts(t, db)

	params := dhcp.GetHostParams{
		ID: hosts[0].ID,
	}
	rsp := rapi.GetHost(ctx, params)
	require.IsType(t, &dhcp.GetHostOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostOK)
	returnedHost := okRsp.Payload
	require.EqualValues(t, hosts[0].ID, returnedHost.ID)
	require.EqualValues(t, hosts[0].SubnetID, returnedHost.SubnetID)
	require.Equal(t, hosts[0].Hostname, returnedHost.Hostname)

	// Get host for non-existing ID should return a default response.
	params = dhcp.GetHostParams{
		ID: 100000000,
	}
	rsp = rapi.GetHost(ctx, params)
	require.IsType(t, &dhcp.GetHostDefault{}, rsp)
}

// Test that the call creating a new host reservation transaction.
func TestCreateHostBeginSubmit(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving reservation-add commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	// Create the config manager using these agents.
	cm := apps.NewManager(db, fa)
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	_, apps := addTestHosts(t, db)

	// Begin transaction.
	params := dhcp.CreateHostBeginParams{}
	rsp := rapi.CreateHostBegin(ctx, params)
	require.IsType(t, &dhcp.CreateHostBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateHostBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, apps and subnets.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.Len(t, contents.Apps, 2)
	require.Len(t, contents.Subnets, 2)

	// Submit transaction.
	params2 := dhcp.CreateHostSubmitParams{
		ID: transactionID,
		Host: &models.Host{
			SubnetID: 1,
			Hostname: "example.org",
			LocalHosts: []*models.LocalHost{
				{
					DaemonID: apps[0].Daemons[0].ID,
				},
				{
					DaemonID: apps[1].Daemons[0].ID,
				},
			},
		},
	}
	rsp2 := rapi.CreateHostSubmit(ctx, params2)
	require.IsType(t, &dhcp.CreateHostSubmitOK{}, rsp2)

	// It should result in sending commands to two Kea servers.
	require.Len(t, fa.RecordedCommands, 2)

	for _, c := range fa.RecordedCommands {
		require.JSONEq(t, `{
            "command": "reservation-add",
            "service": ["dhcp4"],
            "arguments": {
                "reservation": {
                    "subnet-id": 111,
                    "hostname": "example.org"
                }
            }
        }`, c.Marshal())
	}
}

// Test error case when a user attempts to begin new transaction when the
// user has no session.
func TestCreateHostBeginNoSession(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving reservation-add commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	// Create the config manager using these agents.
	cm := apps.NewManager(db, fa)
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm)
	require.NoError(t, err)

	// Create session manager but do not login the user.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	_, _ = addTestHosts(t, db)

	// Begin transaction.
	params := dhcp.CreateHostBeginParams{}
	rsp := rapi.CreateHostBegin(ctx, params)
	require.IsType(t, &dhcp.CreateHostBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.CreateHostBeginDefault)
	require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
}

// Test error cases for submitting new host reservation.
func TestCreateHostSubmitError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Setup fake agents that return an error in response to reservation-add
	// command.
	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("reservation-add", cmdResponses)
	}, nil)
	require.NotNil(t, fa)

	// Create config manager.
	cm := apps.NewManager(db, fa)
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm)
	require.NoError(t, err)

	// Start session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create a user and simulate logging in.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	_, apps := addTestHosts(t, db)

	// Begin transaction. It will be needed for the actual part of the
	// test that relies on the existence of the transaction.
	params := dhcp.CreateHostBeginParams{}
	rsp := rapi.CreateHostBegin(ctx, params)
	require.IsType(t, &dhcp.CreateHostBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateHostBeginOK)
	contents := okRsp.Payload

	// Capture transaction ID.
	transactionID := contents.ID
	require.NotZero(t, transactionID)

	// Submit transaction without the host information.
	t.Run("no host", func(t *testing.T) {
		params := dhcp.CreateHostSubmitParams{
			ID:   transactionID,
			Host: nil,
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	})

	// Submit transaction with non-matching transaction ID.
	t.Run("wrong transaction id", func(t *testing.T) {
		params := dhcp.CreateHostSubmitParams{
			ID: transactionID + 1,
			Host: &models.Host{
				LocalHosts: []*models.LocalHost{
					{
						DaemonID: apps[0].Daemons[0].ID,
					},
					{
						DaemonID: apps[1].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	})

	// Submit transaction with a host that is not associated with any daemons.
	// It simulates a failure in "apply" step which typically is caused by
	// some internal server problem rather than malformed request.
	t.Run("no daemons in host", func(t *testing.T) {
		params := dhcp.CreateHostSubmitParams{
			ID: transactionID,
			Host: &models.Host{
				LocalHosts: []*models.LocalHost{},
			},
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	})

	// Submit transaction with valid ID and host but expect the agent to
	// return an error code. This is considered a conflict with the state
	// of the Kea servers.
	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.CreateHostSubmitParams{
			ID: transactionID,
			Host: &models.Host{
				LocalHosts: []*models.LocalHost{
					{
						DaemonID: apps[0].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
	})

	// Submit transaction with valid ID and host but the user has no
	// session.
	t.Run("no user session", func(t *testing.T) {
		err = rapi.SessionManager.LogoutHandler(ctx)
		require.NoError(t, err)

		params := dhcp.CreateHostSubmitParams{
			ID: transactionID,
			Host: &models.Host{
				LocalHosts: []*models.LocalHost{
					{
						DaemonID: apps[0].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
	})
}
