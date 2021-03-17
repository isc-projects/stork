package restservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storktest "isc.org/stork/server/test"
)

// Generates a success mock response to a command fetching a DHCPv4
// lease by IP address.
func mockLease4Get(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 0,
            "text": "Lease found",
            "arguments": {
                "client-id": "42:42:42:42:42:42:42:42",
                "cltt": 12345678,
                "fqdn-fwd": true,
                "fqdn-rev": true,
                "hostname": "myhost.example.com.",
                "hw-address": "08:08:08:08:08:08",
                "ip-address": "192.0.2.1",
                "state": 0,
                "subnet-id": 44,
                "valid-lft": 3600
            }
        }
    ]`)
	daemons, _ := keactrl.NewDaemons("dhcp4")
	command, _ := keactrl.NewCommand("lease4-get", daemons, nil)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a success mock response to a command fetching DHCPv6 leases
// by DUID.
func mockLeases6Get(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 0,
            "text": "Leases found",
            "arguments": {
                "leases": [
                    {
                        "cltt": 12345678,
                        "duid": "42:42:42:42:42:42:42:42:42:42:42:42:42:42:42",
                        "fqdn-fwd": true,
                        "fqdn-rev": true,
                        "hostname": "myhost.example.com.",
                        "hw-address": "08:08:08:08:08:08",
                        "iaid": 1,
                        "ip-address": "2001:db8:2::1",
                        "preferred-lft": 500,
                        "state": 0,
                        "subnet-id": 44,
                        "type": "IA_NA",
                        "valid-lft": 3600
                    },
                    {
                        "cltt": 12345678,
                        "duid": "42:42:42:42:42:42:42:42:42:42:42:42:42:42:42",
                        "fqdn-fwd": false,
                        "fqdn-rev": false,
                        "hostname": "",
                        "iaid": 1,
                        "ip-address": "2001:db8:0:0:2::",
                        "preferred-lft": 500,
                        "prefix-len": 80,
                        "state": 0,
                        "subnet-id": 44,
                        "type": "IA_PD",
                        "valid-lft": 3600
                    }
                ]
            }
        }
    ]`)
	daemons, _ := keactrl.NewDaemons("dhcp6")
	command, _ := keactrl.NewCommand("lease6-get-by-duid", daemons, nil)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates an error response to lease4-get command.
func mockLease4GetError(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 1,
            "text": "Lease erred"
        }
    ]`)
	daemons, _ := keactrl.NewDaemons("dhcp4")
	command, _ := keactrl.NewCommand("lease4-get", daemons, nil)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// This test verifies that it is possible to search DHCPv4 leases by text
// over the REST API.
func TestFindLeases4(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Add Kea app with a DHCPv4 configuration loading the lease_cmds hooks library.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		Name:         "fxz",
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name: dbmodel.DaemonNameDHCPv4,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: dbmodel.NewKeaConfig(&map[string]interface{}{
						"Dhcp4": map[string]interface{}{
							"hooks-libraries": []interface{}{
								map[string]interface{}{
									"library": "libdhcp_lease_cmds.so",
								},
							},
						},
					}),
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Setup REST API.
	settings := RestAPISettings{}
	agents := agentcommtest.NewFakeAgents(mockLease4Get, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, agents, fec, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// Search by IPv4 address.
	text := "192.0.2.3"
	params := dhcp.GetLeasesParams{
		Text: &text,
	}
	rsp := rapi.GetLeases(ctx, params)
	require.IsType(t, &dhcp.GetLeasesOK{}, rsp)
	okRsp := rsp.(*dhcp.GetLeasesOK)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Total)
	require.Empty(t, okRsp.Payload.ErredApps)

	lease := okRsp.Payload.Items[0]
	require.NotNil(t, lease.AppID)
	require.EqualValues(t, app.ID, *lease.AppID)
	require.NotNil(t, lease.AppName)
	require.Equal(t, app.Name, *lease.AppName)
	require.Equal(t, "42:42:42:42:42:42:42:42", lease.ClientID)
	require.NotNil(t, lease.Cltt)
	require.EqualValues(t, 12345678, *lease.Cltt)
	require.True(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Equal(t, "myhost.example.com.", lease.Hostname)
	require.Equal(t, "08:08:08:08:08:08", lease.HwAddress)
	require.EqualValues(t, "192.0.2.1", *lease.IPAddress)
	require.NotNil(t, lease.State)
	require.EqualValues(t, 0, *lease.State)
	require.NotNil(t, lease.SubnetID)
	require.EqualValues(t, 44, *lease.SubnetID)
	require.NotNil(t, lease.ValidLifetime)
	require.EqualValues(t, 3600, *lease.ValidLifetime)

	// Test the case when the Kea server returns an error.
	agents = agentcommtest.NewFakeAgents(mockLease4GetError, nil)
	rapi, err = NewRestAPI(&settings, dbSettings, db, agents, fec, nil)
	require.NoError(t, err)

	rsp = rapi.GetLeases(ctx, params)
	require.IsType(t, &dhcp.GetLeasesOK{}, rsp)
	okRsp = rsp.(*dhcp.GetLeasesOK)
	require.Empty(t, okRsp.Payload.Items)
	require.Zero(t, okRsp.Payload.Total)

	// Erred apps should contain our app.
	require.Len(t, okRsp.Payload.ErredApps, 1)
	require.NotNil(t, okRsp.Payload.ErredApps[0].ID)
	require.EqualValues(t, app.ID, *okRsp.Payload.ErredApps[0].ID)
	require.NotNil(t, okRsp.Payload.ErredApps[0].Name)
	require.Equal(t, app.Name, *okRsp.Payload.ErredApps[0].Name)
}

// This test verifies that it is possible to search DHCPv6 leases by text
// over the REST API.
func TestFindLeases6(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Add Kea app with a DHCPv6 configuration loading the lease_cmds hooks library.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		Name:         "fyz",
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name: dbmodel.DaemonNameDHCPv6,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: dbmodel.NewKeaConfig(&map[string]interface{}{
						"Dhcp6": map[string]interface{}{
							"hooks-libraries": []interface{}{
								map[string]interface{}{
									"library": "libdhcp_lease_cmds.so",
								},
							},
						},
					}),
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Setup REST API.
	settings := RestAPISettings{}
	agents := agentcommtest.NewFakeAgents(mockLeases6Get, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, agents, fec, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// Search by DUID.
	text := "42:42:42:42:42:42:42:42:42:42:42:42:42:42:42"
	params := dhcp.GetLeasesParams{
		Text: &text,
	}
	rsp := rapi.GetLeases(ctx, params)
	require.IsType(t, &dhcp.GetLeasesOK{}, rsp)
	okRsp := rsp.(*dhcp.GetLeasesOK)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.Empty(t, okRsp.Payload.ErredApps)

	lease := okRsp.Payload.Items[0]
	require.NotNil(t, lease.AppID)
	require.EqualValues(t, app.ID, *lease.AppID)
	require.NotNil(t, lease.AppName)
	require.Equal(t, app.Name, *lease.AppName)
	require.NotNil(t, lease.Cltt)
	require.EqualValues(t, 12345678, *lease.Cltt)
	require.Equal(t, "42:42:42:42:42:42:42:42:42:42:42:42:42:42:42", lease.Duid)
	require.True(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Equal(t, "myhost.example.com.", lease.Hostname)
	require.Equal(t, "08:08:08:08:08:08", lease.HwAddress)
	require.EqualValues(t, 1, lease.Iaid)
	require.NotNil(t, lease.IPAddress)
	require.Equal(t, "2001:db8:2::1", *lease.IPAddress)
	require.EqualValues(t, 500, lease.PreferredLifetime)
	require.NotNil(t, lease.State)
	require.EqualValues(t, 0, *lease.State)
	require.NotNil(t, lease.SubnetID)
	require.EqualValues(t, 44, *lease.SubnetID)
	require.Equal(t, "IA_NA", lease.LeaseType)
	require.NotNil(t, lease.ValidLifetime)
	require.EqualValues(t, 3600, *lease.ValidLifetime)

	lease = okRsp.Payload.Items[1]
	require.NotNil(t, lease.AppID)
	require.EqualValues(t, app.ID, *lease.AppID)
	require.Equal(t, app.Name, *lease.AppName)
	require.Equal(t, app.Name, *lease.AppName)
	require.NotNil(t, lease.Cltt)
	require.EqualValues(t, 12345678, *lease.Cltt)
	require.Equal(t, "42:42:42:42:42:42:42:42:42:42:42:42:42:42:42", lease.Duid)
	require.False(t, lease.FqdnFwd)
	require.False(t, lease.FqdnRev)
	require.Empty(t, lease.Hostname)
	require.Empty(t, lease.HwAddress)
	require.EqualValues(t, 1, lease.Iaid)
	require.NotNil(t, lease.IPAddress)
	require.Equal(t, "2001:db8:0:0:2::", *lease.IPAddress)
	require.EqualValues(t, 500, lease.PreferredLifetime)
	require.EqualValues(t, 80, lease.PrefixLength)
	require.NotNil(t, lease.State)
	require.EqualValues(t, 0, *lease.State)
	require.NotNil(t, lease.SubnetID)
	require.EqualValues(t, 44, *lease.SubnetID)
	require.Equal(t, "IA_PD", lease.LeaseType)
	require.NotNil(t, lease.ValidLifetime)
	require.EqualValues(t, 3600, *lease.ValidLifetime)
}

// Test that when blank search text is specified no leases are returned.
func TestFindLeasesEmptyText(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Add Kea app with a DHCPv4 configuration loading the lease_cmds hooks library.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name: dbmodel.DaemonNameDHCPv4,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: dbmodel.NewKeaConfig(&map[string]interface{}{
						"Dhcp4": map[string]interface{}{
							"hooks-libraries": []interface{}{
								map[string]interface{}{
									"library": "libdhcp_lease_cmds.so",
								},
							},
						},
					}),
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Setup REST API.
	settings := RestAPISettings{}
	agents := agentcommtest.NewFakeAgents(mockLease4Get, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, agents, fec, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// Specify blank search text.
	text := "    "
	params := dhcp.GetLeasesParams{
		Text: &text,
	}
	rsp := rapi.GetLeases(ctx, params)
	require.IsType(t, &dhcp.GetLeasesOK{}, rsp)
	okRsp := rsp.(*dhcp.GetLeasesOK)
	// Expect no leases to be returned.
	require.Empty(t, okRsp.Payload.Items)
	require.Empty(t, okRsp.Payload.ErredApps)
	require.Zero(t, okRsp.Payload.Total)
}
