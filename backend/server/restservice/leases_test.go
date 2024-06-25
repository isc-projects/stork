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
                "valid-lft": 3600,
                "user-context": { "ISC": { "client-classes": [ "ALL", "HA_primary", "UNKNOWN" ] }}
            }
        }
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4Get, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a success mock response to a command fetching a DHCPv4
// lease by IP address.
func mockLease6Get(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 0,
            "text": "Lease found",
            "arguments": {
                "cltt": 12345678,
                "duid": "42:42:42:42:42:42:42:42:42:42:42:42:42:42:42",
                "iaid": 1,
                "ip-address": "2001:db8:2::1",
                "preferred-lft": 500,
                "state": 0,
                "subnet-id": 44,
                "type": "IA_NA",
                "valid-lft": 3600
            }
        }
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease6Get, keactrl.DHCPv6)
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
                        "valid-lft": 3600,
                        "user-context": { "ISC": { "client-classes": [ "ALL", "HA_primary", "UNKNOWN" ] }}
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
                        "valid-lft": 3600,
                        "user-context": { "ISC": { "client-classes": [ "ALL", "HA_primary", "UNKNOWN" ] }}
                    }
                ]
            }
        }
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease6GetByDUID, keactrl.DHCPv6)
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
	command := keactrl.NewCommandBase(keactrl.Lease4Get, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates response to declined leases searching on the DHCPv4 and DHCPv6 server.
func mockLeasesGetDeclined(callNo int, responses []interface{}) {
	json4 := []byte(`[
        {
            "result": 0,
            "text": "Lease found.",
            "arguments": {
                "leases": [
                    {
                        "cltt": 12345678,
                        "ip-address": "192.0.2.1",
                        "state": 1,
                        "subnet-id": 44,
                        "valid-lft": 3600
                    }
                ]
            }
        }
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4GetByHWAddress, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json4, responses[0])

	json6 := []byte(`[
        {
            "result": 0,
            "text": "Lease found.",
            "arguments": {
                "leases": [
                    {
                        "cltt": 12345678,
                        "duid": "00:00:00",
                        "iaid": 1,
                        "ip-address": "2001:db8:2::1",
                        "preferred-lft": 500,
                        "state": 1,
                        "subnet-id": 44,
                        "type": "IA_NA",
                        "valid-lft": 3600
                    }
                ]
            }
        }
    ]`)

	command = keactrl.NewCommandBase(keactrl.Lease6GetByDUID, keactrl.DHCPv6)
	_ = keactrl.UnmarshalResponseList(command, json6, responses[1])
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
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
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
	agents := agentcommtest.NewFakeAgents(mockLease4Get, nil)
	rapi, err := NewRestAPI(dbSettings, db, agents)
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
	require.Zero(t, *lease.State)
	require.NotNil(t, lease.SubnetID)
	require.EqualValues(t, 44, *lease.SubnetID)
	require.NotNil(t, lease.ValidLifetime)
	require.EqualValues(t, 3600, *lease.ValidLifetime)
	require.NotNil(t, lease.UserContext)
	require.Len(t, lease.UserContext, 1)
	userContext := lease.UserContext.(map[string]interface{})
	require.NotNil(t, userContext["ISC"])
	require.Len(t, userContext["ISC"], 1)
	context := userContext["ISC"].(map[string]interface{})
	require.NotNil(t, context["client-classes"])
	require.Len(t, context["client-classes"], 3)
	require.Equal(t, "ALL", context["client-classes"].([]any)[0])
	require.Equal(t, "HA_primary", context["client-classes"].([]any)[1])
	require.Equal(t, "UNKNOWN", context["client-classes"].([]any)[2])

	// Test the case when the Kea server returns an error.
	agents = agentcommtest.NewFakeAgents(mockLease4GetError, nil)
	rapi, err = NewRestAPI(dbSettings, db, agents)
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
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
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
	agents := agentcommtest.NewFakeAgents(mockLeases6Get, nil)
	rapi, err := NewRestAPI(dbSettings, db, agents)
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
	require.Zero(t, *lease.State)
	require.NotNil(t, lease.SubnetID)
	require.EqualValues(t, 44, *lease.SubnetID)
	require.Equal(t, "IA_NA", lease.LeaseType)
	require.NotNil(t, lease.ValidLifetime)
	require.EqualValues(t, 3600, *lease.ValidLifetime)
	require.NotNil(t, lease.UserContext)
	require.Len(t, lease.UserContext, 1)
	userContext := lease.UserContext.(map[string]interface{})
	require.NotNil(t, userContext["ISC"])
	require.Len(t, userContext["ISC"], 1)
	context := userContext["ISC"].(map[string]interface{})
	require.NotNil(t, context["client-classes"])
	require.Len(t, context["client-classes"], 3)
	require.Equal(t, "ALL", context["client-classes"].([]any)[0])
	require.Equal(t, "HA_primary", context["client-classes"].([]any)[1])
	require.Equal(t, "UNKNOWN", context["client-classes"].([]any)[2])

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
	require.Zero(t, *lease.State)
	require.NotNil(t, lease.SubnetID)
	require.EqualValues(t, 44, *lease.SubnetID)
	require.Equal(t, "IA_PD", lease.LeaseType)
	require.NotNil(t, lease.ValidLifetime)
	require.EqualValues(t, 3600, *lease.ValidLifetime)
	require.NotNil(t, lease.UserContext)
	require.Len(t, lease.UserContext, 1)
	userContext = lease.UserContext.(map[string]interface{})
	require.NotNil(t, userContext["ISC"])
	require.Len(t, userContext["ISC"], 1)
	context = userContext["ISC"].(map[string]interface{})
	require.NotNil(t, context["client-classes"])
	require.Len(t, context["client-classes"], 3)
	require.Equal(t, "ALL", context["client-classes"].([]any)[0])
	require.Equal(t, "HA_primary", context["client-classes"].([]any)[1])
	require.Equal(t, "UNKNOWN", context["client-classes"].([]any)[2])
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
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
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
	agents := agentcommtest.NewFakeAgents(mockLease4Get, nil)
	rapi, err := NewRestAPI(dbSettings, db, agents)
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

// Test that declined leases are searched when state:declined search text
// is specified.
func TestFindDeclinedLeases(t *testing.T) {
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

	// Add Kea app with a DHCPv4 and DHCPv6 configuration loading the
	// lease_cmds hooks library.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
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
	agents := agentcommtest.NewFakeAgents(mockLeasesGetDeclined, nil)
	rapi, err := NewRestAPI(dbSettings, db, agents)
	require.NoError(t, err)
	ctx := context.Background()

	text := "state:declined"
	params := dhcp.GetLeasesParams{
		Text: &text,
	}
	rsp := rapi.GetLeases(ctx, params)
	require.IsType(t, &dhcp.GetLeasesOK{}, rsp)
	okRsp := rsp.(*dhcp.GetLeasesOK)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.Empty(t, okRsp.Payload.ErredApps)

	// Verify that the lease contents were parsed correctly. Specifically, we should
	// ensure that HW address, client-id and DUID (in v4 case) are empty.
	leases := okRsp.Payload.Items
	require.NotNil(t, leases[0].IPAddress)
	require.Equal(t, "192.0.2.1", *leases[0].IPAddress)
	require.NotNil(t, leases[0].State)
	require.Empty(t, leases[0].HwAddress)
	require.Empty(t, leases[0].ClientID)
	require.Empty(t, leases[0].Duid)
	require.EqualValues(t, 1, *leases[0].State)
	require.NotNil(t, leases[1].IPAddress)
	require.Equal(t, "2001:db8:2::1", *leases[1].IPAddress)
	require.EqualValues(t, 1, *leases[1].State)
	require.NotNil(t, leases[1].State)
	require.Empty(t, leases[1].HwAddress)
	require.Empty(t, leases[1].ClientID)
	require.Equal(t, "00:00:00", leases[1].Duid)

	// Ensure that appropriate commands were sent to Kea.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4GetByHWAddress, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[1].GetCommand())

	// Whitespace should be allowed between state: and declined.
	text = "state:   declined"
	rsp = rapi.GetLeases(ctx, params)
	require.IsType(t, &dhcp.GetLeasesOK{}, rsp)

	require.Len(t, agents.RecordedCommands, 4)
	require.Equal(t, keactrl.Lease4GetByHWAddress, agents.RecordedCommands[2].GetCommand())
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[3].GetCommand())

	// Ensure that the invalid search text is treated as a hostname rather
	// than a search string to find declined leases.
	text = "ABCstate:declinedDEF"
	rsp = rapi.GetLeases(ctx, params)
	require.IsType(t, &dhcp.GetLeasesOK{}, rsp)

	require.Len(t, agents.RecordedCommands, 6)
	require.Equal(t, keactrl.Lease4GetByHostname, agents.RecordedCommands[4].GetCommand())
	require.Equal(t, keactrl.Lease6GetByHostname, agents.RecordedCommands[5].GetCommand())

	// Whitespace after state is not allowed.
	text = "state :declined"
	rsp = rapi.GetLeases(ctx, params)
	require.IsType(t, &dhcp.GetLeasesOK{}, rsp)

	require.Len(t, agents.RecordedCommands, 8)
	require.Equal(t, keactrl.Lease4GetByHostname, agents.RecordedCommands[6].GetCommand())
	require.Equal(t, keactrl.Lease6GetByHostname, agents.RecordedCommands[7].GetCommand())
}

// Test searching leases and conflicting leases by host ID.
func TestFindLeasesByHostID(t *testing.T) {
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
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
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

	// Add a host.
	host := dbmodel.Host{
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{8, 8, 8, 8, 8, 8},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID:   app.Daemons[0].ID,
				DataSource: dbmodel.HostDataSourceConfig,
				IPReservations: []dbmodel.IPReservation{
					{
						Address: "192.0.2.1",
					},
					{
						Address: "2001:db8:2::1",
					},
				},
			},
		},
	}
	err = dbmodel.AddHost(db, &host)
	require.NoError(t, err)

	// Setup REST API.
	agents := agentcommtest.NewKeaFakeAgents(mockLease4Get, mockLease6Get)
	rapi, err := NewRestAPI(dbSettings, db, agents)
	require.NoError(t, err)
	ctx := context.Background()

	hostID := host.ID
	params := dhcp.GetLeasesParams{
		HostID: &hostID,
	}
	// Get leases by host ID.
	rsp := rapi.GetLeases(ctx, params)
	require.IsType(t, &dhcp.GetLeasesOK{}, rsp)
	okRsp := rsp.(*dhcp.GetLeasesOK)

	// Two leases should be returned. One DHCPv4 and one DHCPv6 lease.
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Total)

	// The DHCPv6 is in conflict with host reservation, because DUID
	// is not in the host reservation.
	require.Len(t, okRsp.Payload.Conflicts, 1)
	require.Empty(t, okRsp.Payload.ErredApps)

	// Verify the leases.
	require.NotNil(t, okRsp.Payload.Items[0].IPAddress)
	require.Equal(t, "192.0.2.1", *okRsp.Payload.Items[0].IPAddress)
	require.NotNil(t, okRsp.Payload.Items[1].IPAddress)
	require.Equal(t, "2001:db8:2::1", *okRsp.Payload.Items[1].IPAddress)
	require.Len(t, okRsp.Payload.Conflicts, 1)
	require.EqualValues(t, *okRsp.Payload.Items[1].ID, okRsp.Payload.Conflicts[0])
}
