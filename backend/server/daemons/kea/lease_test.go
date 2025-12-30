package kea

import (
	"encoding/json"
	"testing"

	require "github.com/stretchr/testify/require"

	keactrl "isc.org/stork/daemonctrl/kea"
	keadata "isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Generates a success mock response to commands fetching a single
// DHCPv4 lease.
func mockLease4Get(callNo int, responses []any) {
	bytes := []byte(`
        {
            "result": 0,
            "text": "Lease found",
            "arguments": {
                "client-id": "42:42:42:42:42:42:42:42",
                "cltt": 12345678,
                "fqdn-fwd": false,
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
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Generates a success mock response to commands fetching a single
// DHCPv4 lease with invalid user context data.
func mockLease4GetInvalidJSON(callNo int, responses []any) {
	bytes := []byte(`
        {
            "result": 0,
            "text": "Lease found",
            "arguments": {
                "client-id": "42:42:42:42:42:42:42:42",
                "cltt": 12345678,
                "fqdn-fwd": false,
                "fqdn-rev": true,
                "hostname": "myhost.example.com.",
                "hw-address": "08:08:08:08:08:08",
                "ip-address": "192.0.2.1",
                "state": 0,
                "subnet-id": 44,
                "valid-lft": 3600,
                "user-context": "invalid"
            }
        }
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Generates a response to lease4-get command. The first time it is
// called it returns an error response. The second time it returns
// a lease. It is useful to simulate tracking erred communication
// with selected servers.
func mockLease4GetFirstCallError(callNo int, responses []any) {
	var bytes []byte
	if callNo == 0 {
		bytes = []byte(`
            {
                "result": 1,
                "text": "Lease erred",
                "arguments": { }
            }
        `)
		_ = json.Unmarshal(bytes, responses[0])
		return
	}

	bytes = []byte(`
        {
            "result": 0,
            "text": "Lease found",
            "arguments": {
                "client-id": "42:42:42:42:42:42:42:42",
                "cltt": 12345678,
                "fqdn-fwd": false,
                "fqdn-rev": true,
                "hostname": "myhost.example.com.",
                "hw-address": "08:08:08:08:08:08",
                "ip-address": "192.0.2.1",
                "state": 0,
                "subnet-id": 44,
                "valid-lft": 3600
            }
        }
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Generates a success mock response to commands fetching a single
// DHCPv6 lease by IPv6 address.
func mockLease6GetByIPAddress(callNo int, responses []any) {
	bytes := []byte(`{
        "result": 0,
        "text": "Lease found",
        "arguments": {
            "cltt": 12345678,
                "duid": "42:42:42:42:42:42:42:42",
                "fqdn-fwd": false,
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
            }
        }
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Generates a success mock response to commands fetching a single
// DHCPv6 lease by IPv6 address.
func mockLease6GetByPrefix(callNo int, responses []any) {
	bytes := []byte(`
        {
            "result": 0,
            "text": "Lease found",
            "arguments": {
                "cltt": 12345678,
                "duid": "42:42:42:42:42:42:42:42",
                "fqdn-fwd": false,
                "fqdn-rev": true,
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
        }
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Generates a success mock response to commands fetching a single
// DHCPv6 lease with invalid user context data.
func mockLease6GetInvalidJSON(callNo int, responses []any) {
	bytes := []byte(`
        {
            "result": 0,
            "text": "Lease found",
            "arguments": {
                "cltt": 12345678,
                "duid": "42:42:42:42:42:42:42:42",
                "fqdn-fwd": false,
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
                "user-context": "invalid"
            }
        }
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Generates a mock response to lease4-get-by-hw-address and lease4-get-by-client-id
// combined in a single gRPC command. The first response is successful, the second
// response indicates an error.
func mockLeases4GetSecondError(callNo int, responses []any) {
	// Response to lease4-get-by-hw-address.
	bytes := []byte(`
        {
            "result": 0,
            "text": "Leases found",
            "arguments": {
                "leases": [
                    {
                        "client-id": "42:42:42:42:42:42:42:42",
                        "cltt": 12345678,
                        "fqdn-fwd": false,
                        "fqdn-rev": true,
                        "hostname": "myhost.example.com.",
                        "hw-address": "08:08:08:08:08:08",
                        "ip-address": "192.0.2.1",
                        "state": 0,
                        "subnet-id": 44,
                        "valid-lft": 3600
                    }
                ]
            }
        }
    `)
	_ = json.Unmarshal(bytes, responses[0])

	// Response to lease4-get-by-client-id.
	bytes = []byte(`
        {
            "result": 1,
            "text": "Leases erred",
            "arguments": { }
        }
    `)
	_ = json.Unmarshal(bytes, responses[1])
}

// Generates a mock empty response to commands fetching DHCPv4 leases.
func mockLeases4GetEmpty(callNo int, responses []any) {
	bytes := []byte(`
        {
            "result": 3,
            "text": "No lease found."
        }
    `)

	for _, response := range responses {
		_ = json.Unmarshal(bytes, response)
	}
}

// Generates a mock empty response to commands fetching DHCPv6 leases.
func mockLeases6GetEmpty(callNo int, responses []any) {
	bytes := []byte(`
        {
            "result": 3,
            "text": "No lease found."
        }
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Generates a success mock response to commands fetching multiple
// DHCPv4 leases.
func mockLeases4Get(callNo int, responses []any) {
	bytes := []byte(`
        {
            "result": 0,
            "text": "Leases found",
            "arguments": {
                "leases": [
                    {
                        "client-id": "42:42:42:42:42:42:42:42",
                        "cltt": 12345678,
                        "fqdn-fwd": false,
                        "fqdn-rev": true,
                        "hostname": "myhost.example.com.",
                        "hw-address": "08:08:08:08:08:08",
                        "ip-address": "192.0.2.1",
                        "state": 0,
                        "subnet-id": 44,
                        "valid-lft": 3600
                    }
                ]
            }
        }
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Generates a success mock response to commands fetching multiple
// DHCPv6 leases.
func mockLeases6Get(callNo int, responses []interface{}) {
	bytes := []byte(`
        {
            "result": 0,
            "text": "Leases found",
            "arguments": {
                "leases": [
                    {
                        "cltt": 12345678,
                        "duid": "42:42:42:42:42:42:42:42",
                        "fqdn-fwd": false,
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
                        "duid": "42:42:42:42:42:42:42:42",
                        "fqdn-fwd": false,
                        "fqdn-rev": true,
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
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Generates responses to declined leases search. First response comprises
// two DHCPv4 leases, one in the default state and one in the declined state.
// Stork should ignore the lease in the default state. The second response
// contains two declined DHCPv6 leases.
func mockLeasesGetDeclined(callNo int, responses []any) {
	switch callNo {
	case 0:
		bytes := []byte(`
			{
				"result": 0,
				"text": "Leases found",
				"arguments": {
					"leases": [
						{
							"cltt": 12345678,
							"hw-address": "",
							"ip-address": "192.0.2.1",
							"state": 0,
							"subnet-id": 44,
							"valid-lft": 3600
						},
						{
							"cltt": 12345678,
							"hw-address": "",
							"ip-address": "192.0.2.2",
							"state": 1,
							"subnet-id": 44,
							"valid-lft": 3600
						}
					]
				}
			}
		`)
		_ = json.Unmarshal(bytes, responses[0])
	case 1:
		bytes := []byte(`
        {
            "result": 0,
            "text": "Leases found",
            "arguments": {
                "leases": [
                    {
                        "cltt": 12345678,
                        "duid": "",
                        "hw-address": "",
                        "iaid": 1,
                        "ip-address": "2001:db8:2::1",
                        "preferred-lft": 500,
                        "state": 1,
                        "subnet-id": 44,
                        "type": "IA_NA",
                        "valid-lft": 3600
                    },
                    {
                        "cltt": 12345678,
                        "duid": "",
                        "hw-address": "08:08:08:08:08:08",
                        "iaid": 1,
                        "ip-address": "2001:db8:2::2",
                        "preferred-lft": 500,
                        "state": 1,
                        "subnet-id": 44,
                        "type": "IA_NA",
                        "valid-lft": 3600
                    }
                ]
            }
        }
    `)
		_ = json.Unmarshal(bytes, responses[0])
	}
}

// Generates responses to declined leases search. First response comprises
// one in the declined state. The second response contains two declined DHCPv6
// leases.
func mockLeasesGetByStatusDeclined(callNo int, responses []any) {
	switch callNo {
	case 0:
		bytes := []byte(`
			{
				"result": 0,
				"text": "Leases found",
				"arguments": {
					"leases": [
						{
							"cltt": 12345678,
							"hw-address": "",
							"ip-address": "192.0.2.2",
							"state": 1,
							"subnet-id": 44,
							"valid-lft": 3600
						}
					]
				}
			}
		`)
		_ = json.Unmarshal(bytes, responses[0])
	case 1:
		bytes := []byte(`
        {
            "result": 0,
            "text": "Leases found",
            "arguments": {
                "leases": [
                    {
                        "cltt": 12345678,
                        "duid": "",
                        "hw-address": "",
                        "iaid": 1,
                        "ip-address": "2001:db8:2::1",
                        "preferred-lft": 500,
                        "state": 1,
                        "subnet-id": 44,
                        "type": "IA_NA",
                        "valid-lft": 3600
                    },
                    {
                        "cltt": 12345678,
                        "duid": "",
                        "hw-address": "08:08:08:08:08:08",
                        "iaid": 1,
                        "ip-address": "2001:db8:2::2",
                        "preferred-lft": 500,
                        "state": 1,
                        "subnet-id": 44,
                        "type": "IA_NA",
                        "valid-lft": 3600
                    }
                ]
            }
        }
    `)
		_ = json.Unmarshal(bytes, responses[0])
	}
}

func mockLeasesGetDeclinedErrors(callNo int, responses []any) {
	switch callNo {
	case 0:
		bytes := []byte(`
			{
				"result": 1,
				"text": "Leases search erred"
			}
		`)
		_ = json.Unmarshal(bytes, responses[0])
	case 1:
		bytes := []byte(`
        {
            "result": 0,
            "text": "Leases found",
            "arguments": {
                "leases": [
                    {
                        "cltt": 12345678,
                        "duid": "",
                        "hw-address": "08:08:08:08:08:08",
                        "iaid": 1,
                        "ip-address": "2001:db8:2::2",
                        "preferred-lft": 500,
                        "state": 1,
                        "subnet-id": 44,
                        "type": "IA_NA",
                        "valid-lft": 3600
                    }
                ]
            }
        }
    `)
		_ = json.Unmarshal(bytes, responses[0])
	}
}

// Generate an error mock response to a command fetching lease by an IPv6
// address.
func mockLease6GetError(callNo int, responses []any) {
	bytes := []byte(`
        {
            "result": 1,
            "text": "Getting an lease erred."
        }
    `)
	_ = json.Unmarshal(bytes, responses[0])
}

// Test the success scenario in sending lease4-get command to Kea.
func TestGetLease4ByIPAddress(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLease4Get, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}
	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv4, true, accessPoints)
	daemon.ID = 1

	lease, err := GetLease4ByIPAddress(agents, daemon, "192.0.2.3")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, daemon.ID, lease.DaemonID)
	require.NotNil(t, lease.Daemon)
	require.Equal(t, "42:42:42:42:42:42:42:42", lease.ClientID)
	require.EqualValues(t, 12345678, lease.CLTT)
	require.False(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Equal(t, "myhost.example.com.", lease.Hostname)
	require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
	require.Equal(t, "192.0.2.1", lease.IPAddress)
	require.Zero(t, lease.State)
	require.EqualValues(t, 44, lease.SubnetID)
	require.EqualValues(t, 3600, lease.ValidLifetime)
	require.NotNil(t, lease.UserContext)
	require.Len(t, lease.UserContext, 1)
	require.NotNil(t, lease.UserContext["ISC"])
	require.Len(t, lease.UserContext["ISC"], 1)
	context := lease.UserContext["ISC"].(map[string]any)
	require.NotNil(t, context["client-classes"])
	require.Len(t, context["client-classes"], 3)
	require.Equal(t, "ALL", context["client-classes"].([]any)[0])
	require.Equal(t, "HA_primary", context["client-classes"].([]any)[1])
	require.Equal(t, "UNKNOWN", context["client-classes"].([]any)[2])
}

// Test the success scenario in sending lease6-get command to Kea to get
// a lease by IPv6 address.
func TestGetLease6ByIPAddress(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLease6GetByIPAddress, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTPS,
		},
	}
	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv6, true, accessPoints)
	daemon.ID = 2

	lease, err := GetLease6ByIPAddress(agents, daemon, "IA_NA", "2001:db8:2::1")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, daemon.ID, lease.DaemonID)
	require.NotNil(t, lease.Daemon)
	require.EqualValues(t, 12345678, lease.CLTT)
	require.Equal(t, "42:42:42:42:42:42:42:42", lease.DUID)
	require.False(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Equal(t, "myhost.example.com.", lease.Hostname)
	require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
	require.EqualValues(t, 1, lease.IAID)
	require.Equal(t, "2001:db8:2::1", lease.IPAddress)
	require.EqualValues(t, 500, lease.PreferredLifetime)
	require.Zero(t, lease.State)
	require.EqualValues(t, 44, lease.SubnetID)
	require.Equal(t, "IA_NA", lease.Type)
	require.EqualValues(t, 3600, lease.ValidLifetime)
	require.NotNil(t, lease.UserContext)
	require.Len(t, lease.UserContext, 1)
	require.NotNil(t, lease.UserContext["ISC"])
	require.Len(t, lease.UserContext["ISC"], 1)
	context := lease.UserContext["ISC"].(map[string]any)
	require.NotNil(t, context["client-classes"])
	require.Len(t, context["client-classes"], 3)
	require.Equal(t, "ALL", context["client-classes"].([]any)[0])
	require.Equal(t, "HA_primary", context["client-classes"].([]any)[1])
	require.Equal(t, "UNKNOWN", context["client-classes"].([]any)[2])
}

// Test the success scenario in sending lease6-get command to Kea to get
// a lease by IPv6 prefix.
func TestGetLease6ByPrefix(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLease6GetByPrefix, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTPS,
		},
	}
	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv6, true, accessPoints)
	daemon.ID = 3

	lease, err := GetLease6ByIPAddress(agents, daemon, "IA_PD", "2001:db8:0:0:2::")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, daemon.ID, lease.DaemonID)
	require.NotNil(t, lease.Daemon)
	require.EqualValues(t, 12345678, lease.CLTT)
	require.Equal(t, "42:42:42:42:42:42:42:42", lease.DUID)
	require.False(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Empty(t, lease.Hostname)
	require.Empty(t, lease.HWAddress)
	require.EqualValues(t, 1, lease.IAID)
	require.Equal(t, "2001:db8:0:0:2::", lease.IPAddress)
	require.EqualValues(t, 500, lease.PreferredLifetime)
	require.EqualValues(t, 80, lease.PrefixLength)
	require.Zero(t, lease.State)
	require.EqualValues(t, 44, lease.SubnetID)
	require.Equal(t, "IA_PD", lease.Type)
	require.EqualValues(t, 3600, lease.ValidLifetime)
	require.NotNil(t, lease.UserContext)
	require.Len(t, lease.UserContext, 1)
	require.NotNil(t, lease.UserContext["ISC"])
	require.Len(t, lease.UserContext["ISC"], 1)
	context := lease.UserContext["ISC"].(map[string]any)
	require.NotNil(t, context["client-classes"])
	require.Len(t, context["client-classes"], 3)
	require.Equal(t, "ALL", context["client-classes"].([]any)[0])
	require.Equal(t, "HA_primary", context["client-classes"].([]any)[1])
	require.Equal(t, "UNKNOWN", context["client-classes"].([]any)[2])
}

// Test the scenario in sending lease4-get command to Kea when the lease
// is not found.
func TestGetLease4ByIPAddressEmpty(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}
	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv4, true, accessPoints)

	lease, err := GetLease4ByIPAddress(agents, daemon, "192.0.2.3")
	require.NoError(t, err)
	require.Nil(t, lease)
}

// Test the scenario in sending lease6-get command to Kea when the lease
// is not found.
func TestGetLease6ByIPAddressEmpty(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}
	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv6, true, accessPoints)

	lease, err := GetLease6ByIPAddress(agents, daemon, "IA_NA", "2001:db8:1::2")
	require.NoError(t, err)
	require.Nil(t, lease)
}

// Test success scenarios in sending lease4-get-by-hw-address, lease4-get-by-client-id
// and lease4-get-by-hostname commands to Kea.
func TestGetLeases4(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4Get, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv4, true, accessPoints)

	tests := []struct {
		name          string
		function      func(agentcomm.ConnectedAgents, *dbmodel.Daemon, string) ([]dbmodel.Lease, error)
		propertyValue string
	}{
		{
			name:          "hw-address",
			function:      GetLeases4ByHWAddress,
			propertyValue: "08:08:08:08:08:08",
		},
		{
			name:          "client-id",
			function:      GetLeases4ByClientID,
			propertyValue: "42:42:42:42:42:42:42:42",
		},
		{
			name:          "hostname",
			function:      GetLeases4ByHostname,
			propertyValue: "myhost.example.com.",
		},
	}

	for i := range tests {
		testedFunc := tests[i].function
		propertyValue := tests[i].propertyValue
		t.Run(tests[i].name, func(t *testing.T) {
			leases, err := testedFunc(agents, daemon, propertyValue)
			require.NoError(t, err)
			require.Len(t, leases, 1)

			lease := leases[0]
			require.EqualValues(t, daemon.ID, lease.DaemonID)
			require.NotNil(t, lease.Daemon)
			require.Equal(t, "42:42:42:42:42:42:42:42", lease.ClientID)
			require.EqualValues(t, 12345678, lease.CLTT)
			require.False(t, lease.FqdnFwd)
			require.True(t, lease.FqdnRev)
			require.Equal(t, "myhost.example.com.", lease.Hostname)
			require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
			require.Equal(t, "192.0.2.1", lease.IPAddress)
			require.Zero(t, lease.State)
			require.EqualValues(t, 44, lease.SubnetID)
			require.EqualValues(t, 3600, lease.ValidLifetime)
			require.Nil(t, lease.UserContext)
		})
	}
}

// Test success scenarios in sending lease6-get-by-duid, lease6-get-by-hostname
// commands to Kea.
func TestGetLeases6(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases6Get, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv6, true, accessPoints)

	tests := []struct {
		name          string
		function      func(agentcomm.ConnectedAgents, *dbmodel.Daemon, string) ([]dbmodel.Lease, error)
		propertyValue string
	}{
		{
			name:          "duid",
			function:      GetLeases6ByDUID,
			propertyValue: "08:08:08:08:08:08",
		},
		{
			name:          "hostname",
			function:      GetLeases6ByHostname,
			propertyValue: "myhost.example.com.",
		},
	}

	for i := range tests {
		testedFunc := tests[i].function
		propertyValue := tests[i].propertyValue
		t.Run(tests[i].name, func(t *testing.T) {
			leases, err := testedFunc(agents, daemon, propertyValue)
			require.NoError(t, err)
			require.Len(t, leases, 2)

			lease := leases[0]
			require.EqualValues(t, daemon.ID, lease.DaemonID)
			require.NotNil(t, lease.Daemon)
			require.EqualValues(t, 12345678, lease.CLTT)
			require.Equal(t, "42:42:42:42:42:42:42:42", lease.DUID)
			require.False(t, lease.FqdnFwd)
			require.True(t, lease.FqdnRev)
			require.Equal(t, "myhost.example.com.", lease.Hostname)
			require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
			require.EqualValues(t, 1, lease.IAID)
			require.Equal(t, "2001:db8:2::1", lease.IPAddress)
			require.EqualValues(t, 500, lease.PreferredLifetime)
			require.Zero(t, lease.State)
			require.EqualValues(t, 44, lease.SubnetID)
			require.Equal(t, "IA_NA", lease.Type)
			require.EqualValues(t, 3600, lease.ValidLifetime)
			require.Nil(t, lease.UserContext)

			lease = leases[1]
			require.EqualValues(t, daemon.ID, lease.DaemonID)
			require.NotNil(t, lease.Daemon)
			require.EqualValues(t, 12345678, lease.CLTT)
			require.Equal(t, "42:42:42:42:42:42:42:42", lease.DUID)
			require.False(t, lease.FqdnFwd)
			require.True(t, lease.FqdnRev)
			require.Empty(t, lease.Hostname)
			require.Empty(t, lease.HWAddress)
			require.EqualValues(t, 1, lease.IAID)
			require.Equal(t, "2001:db8:0:0:2::", lease.IPAddress)
			require.EqualValues(t, 500, lease.PreferredLifetime)
			require.EqualValues(t, 80, lease.PrefixLength)
			require.Zero(t, lease.State)
			require.EqualValues(t, 44, lease.SubnetID)
			require.Equal(t, "IA_PD", lease.Type)
			require.EqualValues(t, 3600, lease.ValidLifetime)
			require.Nil(t, lease.UserContext)
		})
	}
}

// Test the scenario in sending lease4-get-by-hw-address command to Kea when
// no lease is found.
func TestGetLeases4Empty(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv4, true, accessPoints)

	leases, err := GetLeases4ByHWAddress(agents, daemon, "000000000000")
	require.NoError(t, err)
	require.Empty(t, leases)

	// Ensure that MAC address was converted to the format expected by Kea.
	arguments := agents.GetCommandArguments(0)
	require.NotNil(t, arguments)
	require.Contains(t, arguments, "hw-address")
	require.Equal(t, "00:00:00:00:00:00", arguments["hw-address"])
}

// Test the scenario in sending lease6-get-by-hostname command to Kea when
// no lease is found.
func TestGetLeases6Empty(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases6GetEmpty, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv6, true, accessPoints)

	leases, err := GetLeases6ByHostname(agents, daemon, "myhost")
	require.NoError(t, err)
	require.Empty(t, leases)
}

// Test sending multiple combined commands when one of the commands fails. The
// function should still return some leases but it should also return the warns
// flag to indicate that there were issues.
func TestGetLeasesByPropertiesSecondError(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4GetSecondError, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}
	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv4, true, accessPoints)
	daemon.ID = 4

	leases, warns, err := getLeasesByProperties(agents, daemon, "42:42:42:42:42:42:42:42", "lease4-get-by-hw-address", "lease4-get-by-client-id")
	require.NoError(t, err)
	require.True(t, warns)
	require.Len(t, leases, 1)

	lease := leases[0]
	require.EqualValues(t, daemon.ID, lease.DaemonID)
	require.NotNil(t, lease.Daemon)
	require.Equal(t, "42:42:42:42:42:42:42:42", lease.ClientID)
	require.EqualValues(t, 12345678, lease.CLTT)
	require.False(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Equal(t, "myhost.example.com.", lease.Hostname)
	require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
	require.Equal(t, "192.0.2.1", lease.IPAddress)
	require.Zero(t, lease.State)
	require.EqualValues(t, 44, lease.SubnetID)
	require.EqualValues(t, 3600, lease.ValidLifetime)
	require.Nil(t, lease.UserContext)
}

// Test the scenario in sending lease4-get command to Kea when
// user context is not a map.
func TestGetLeases4InvalidJSON(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLease4GetInvalidJSON, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}
	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv4, true, accessPoints)
	daemon.ID = 1

	lease, err := GetLease4ByIPAddress(agents, daemon, "192.0.2.3")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, daemon.ID, lease.DaemonID)
	require.NotNil(t, lease.Daemon)
	require.Equal(t, "42:42:42:42:42:42:42:42", lease.ClientID)
	require.EqualValues(t, 12345678, lease.CLTT)
	require.False(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Equal(t, "myhost.example.com.", lease.Hostname)
	require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
	require.Equal(t, "192.0.2.1", lease.IPAddress)
	require.Zero(t, lease.State)
	require.EqualValues(t, 44, lease.SubnetID)
	require.EqualValues(t, 3600, lease.ValidLifetime)
	require.Nil(t, lease.UserContext)
}

// Test the scenario in sending lease6-get command to Kea when
// user context is not a map.
func TestGetLease6InvalidJSON(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLease6GetInvalidJSON, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTPS,
		},
	}
	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv6, true, accessPoints)
	daemon.ID = 2

	lease, err := GetLease6ByIPAddress(agents, daemon, "IA_NA", "2001:db8:2::1")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, daemon.ID, lease.DaemonID)
	require.NotNil(t, lease.Daemon)
	require.EqualValues(t, 12345678, lease.CLTT)
	require.Equal(t, "42:42:42:42:42:42:42:42", lease.DUID)
	require.False(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Equal(t, "myhost.example.com.", lease.Hostname)
	require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
	require.EqualValues(t, 1, lease.IAID)
	require.Equal(t, "2001:db8:2::1", lease.IPAddress)
	require.EqualValues(t, 500, lease.PreferredLifetime)
	require.Zero(t, lease.State)
	require.EqualValues(t, 44, lease.SubnetID)
	require.Equal(t, "IA_NA", lease.Type)
	require.EqualValues(t, 3600, lease.ValidLifetime)
	require.Nil(t, lease.UserContext)
}

// Test validation of the Kea servers' invalid responses or indicating errors.
func TestValidateGetLeasesResponse(t *testing.T) {
	validArgs := &struct{}{}
	invalidArgs := (*struct{})(nil)
	require.Error(t, validateGetLeasesResponse("command", keactrl.ResponseError, validArgs))
	require.Error(t, validateGetLeasesResponse("command", keactrl.ResponseCommandUnsupported, validArgs))
	require.Error(t, validateGetLeasesResponse("command", keactrl.ResponseSuccess, invalidArgs))
}

// Test various positive scenarios to search a lease on multiple Kea servers.
// It checks that FindLeases function sends expected commands to the servers.
// The responses generated by the fake agents are always empty, so it does
// not validate parsing the responses. However, responses parsing is already
// tested in other unit tests.
func TestFindLeases(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add first machine with the Kea daemons including both DHCPv4 and DHCPv6
	// daemon with the lease_cmds hooks library loaded.
	machine1 := &dbmodel.Machine{
		ID:        0,
		Address:   "machine1",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine1)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTPS,
		},
	}

	// Create DHCPv4 daemon
	daemon1v4 := dbmodel.NewDaemon(machine1, daemonname.DHCPv4, true, accessPoints)
	err = daemon1v4.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon1v4)
	require.NoError(t, err)

	// Create DHCPv6 daemon
	daemon1v6 := dbmodel.NewDaemon(machine1, daemonname.DHCPv6, true, accessPoints)
	err = daemon1v6.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon1v6)
	require.NoError(t, err)

	// Add second machine with the Kea daemon with lease_cmds hooks library loaded by
	// the DHCPv4 daemon.
	machine2 := &dbmodel.Machine{
		ID:        0,
		Address:   "machine2",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)

	accessPoints2 := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}

	// Create DHCPv4 daemon for machine2
	daemon2v4 := dbmodel.NewDaemon(machine2, daemonname.DHCPv4, true, accessPoints2)
	err = daemon2v4.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon2v4)
	require.NoError(t, err)

	// Add third machine with the Kea daemon with lease_cmds hooks library loaded by
	// the DHCPv6 daemon.
	machine3 := &dbmodel.Machine{
		ID:        0,
		Address:   "machine3",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, machine3)
	require.NoError(t, err)

	accessPoints3 := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}

	// Create DHCPv6 daemon for machine3
	daemon3v6 := dbmodel.NewDaemon(machine3, daemonname.DHCPv6, true, accessPoints3)
	err = daemon3v6.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon3v6)
	require.NoError(t, err)

	agents := agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	//  Find lease by IPv4 address.
	_, erredDaemons, err := FindLeases(db, agents, "192.0.2.3")
	require.NoError(t, err)
	require.Empty(t, erredDaemons)

	// It should have sent lease4-get command to first and second Kea.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4Get, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease4Get, agents.RecordedCommands[1].GetCommand())

	agents = agentcommtest.NewFakeAgents(mockLease4GetFirstCallError, nil)

	// Test the case when one of the servers returns an error.
	_, erredDaemons, err = FindLeases(db, agents, "192.0.2.3")
	require.NoError(t, err)
	require.Len(t, erredDaemons, 1)
	require.NotNil(t, erredDaemons[0])
	require.EqualValues(t, daemon1v4.ID, erredDaemons[0].ID)

	// It should have sent lease4-get command to first and second Kea.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4Get, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease4Get, agents.RecordedCommands[1].GetCommand())

	agents = agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	// Find lease by IPv6 address.
	_, erredDaemons, err = FindLeases(db, agents, "2001:db8:1::")
	require.NoError(t, err)
	require.Empty(t, erredDaemons)

	// It should have sent lease6-get command to first and third Kea.
	// The commands are duplicated because we need to send one for
	// an address and one for prefix.
	require.Len(t, agents.RecordedCommands, 4)
	require.Equal(t, keactrl.Lease6Get, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6Get, agents.RecordedCommands[1].GetCommand())
	require.Equal(t, keactrl.Lease6Get, agents.RecordedCommands[2].GetCommand())
	require.Equal(t, keactrl.Lease6Get, agents.RecordedCommands[3].GetCommand())

	agents = agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	// Find lease by identifier.
	_, erredDaemons, err = FindLeases(db, agents, "010203040506")
	require.NoError(t, err)
	require.Empty(t, erredDaemons)

	// It should have sent commands to fetch a lease by HW address or client
	// id to first two servers, and a command to fetch a lease by DUID to two
	// DHCPv6 servers.
	require.Len(t, agents.RecordedCommands, 6)
	require.Equal(t, keactrl.Lease4GetByHWAddress, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease4GetByClientID, agents.RecordedCommands[1].GetCommand())
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[2].GetCommand())
	require.Equal(t, keactrl.Lease4GetByHWAddress, agents.RecordedCommands[3].GetCommand())
	require.Equal(t, keactrl.Lease4GetByClientID, agents.RecordedCommands[4].GetCommand())
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[5].GetCommand())

	// In addition, ensure that the HW address was converted to the format
	// expected by Kea.
	arguments := agents.GetCommandArguments(0)
	require.NotNil(t, arguments)
	require.Contains(t, arguments, "hw-address")
	require.Equal(t, "01:02:03:04:05:06", arguments["hw-address"])

	agents = agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	// Find lease by hostname.
	_, erredDaemons, err = FindLeases(db, agents, "myhost")
	require.NoError(t, err)
	require.Empty(t, erredDaemons)

	// It should have sent a command to fetch a lease by hostname to both DHCPv4
	// and DHCPv6 servers.
	require.Len(t, agents.RecordedCommands, 4)
	require.Equal(t, keactrl.Lease4GetByHostname, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6GetByHostname, agents.RecordedCommands[1].GetCommand())
	require.Equal(t, keactrl.Lease4GetByHostname, agents.RecordedCommands[2].GetCommand())
	require.Equal(t, keactrl.Lease6GetByHostname, agents.RecordedCommands[3].GetCommand())
}

// Test that the Kea servers prior to version 2.3.8 receive lease6-get-by-duid
// command and the Kea servers version 2.3.8 and later receive
// lease6-get-by-hostname command when the DUID has less than 3 bytes.
func TestFindLeasesTooShortDUID(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine2",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, machine)

	accessPointsOld := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: "",
		},
	}

	// Create old version daemon (2.0.2)
	daemonOld := dbmodel.NewDaemon(machine, daemonname.DHCPv6, true, accessPointsOld)
	daemonOld.Version = "2.0.2"
	err := daemonOld.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemonOld)
	require.NoError(t, err)

	accessPointsModern := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8001,
			Protocol: "",
		},
	}

	// Create modern version daemon (2.7.2)
	daemonModern := dbmodel.NewDaemon(machine, daemonname.DHCPv6, true, accessPointsModern)
	daemonModern.Version = "2.7.2"
	err = daemonModern.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemonModern)
	require.NoError(t, err)

	agents := agentcommtest.NewFakeAgents(mockLeases6GetEmpty, nil)

	// Act
	_, erredDaemons, err := FindLeases(db, agents, "0102")

	// Assert
	require.NoError(t, err)
	require.Empty(t, erredDaemons)

	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6GetByHostname, agents.RecordedCommands[1].GetCommand())
}

// Test declined leases search mechanism for Kea versions 3.1.0 and before. It
// verifies a positive scenario in which the DHCPv4 server returns two leases
// (one is in a default state and one is in the declined state), and the DHCPv6
// server returns two declined leases. The DHCPv4 lease in the default state
// should be ignored. The remaining three leases should be returned. The test
// also verifies that the commands sent to Kea are formatted correctly, i.e.
// contain empty MAC address and empty DUID. Finally, the test verifies that
// erred daemons are returned if any of the commands returns an error.
func TestFindDeclinedLeasesUsingStatusLegacy(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine with the Kea daemons including both DHCPv4 and DHCPv6
	// daemon with the lease_cmds hooks library loaded.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTPS,
		},
	}

	// Create DHCPv4 daemon
	daemon4 := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)
	daemon4.Version = "3.1.0"
	err = daemon4.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon4)
	require.NoError(t, err)

	// Create DHCPv6 daemon
	daemon6 := dbmodel.NewDaemon(machine, daemonname.DHCPv6, true, accessPoints)
	daemon6.Version = "3.1.0"
	err = daemon6.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon6)
	require.NoError(t, err)

	// Simulate Kea returning 4 leases, one in the default state and three
	// in the declined state.
	agents := agentcommtest.NewFakeAgents(mockLeasesGetDeclined, nil)

	leases, erredDaemons, err := FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Empty(t, erredDaemons)

	// One lease should be ignored and three returned.
	require.Len(t, leases, 3)

	// Basic checks if expected leases were returned.
	for i, ipAddress := range []string{"192.0.2.2", "2001:db8:2::1", "2001:db8:2::2"} {
		expectedDaemon := daemon4
		if i > 0 {
			expectedDaemon = daemon6
		}

		require.EqualValues(t, expectedDaemon.ID, leases[i].DaemonID)
		require.NotNil(t, leases[i].Daemon)
		require.Equal(t, ipAddress, leases[i].IPAddress)
		require.EqualValues(t, keadata.LeaseStateDeclined, leases[i].State)
	}

	// Ensure that Stork has sent two commands, one to the DHCPv4 server and one
	// to the DHCPv6 server.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4GetByHWAddress, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[1].GetCommand())

	// Ensure that the hw-address sent in the first command is empty.
	arguments := agents.GetCommandArguments(0)
	require.NotNil(t, arguments)
	require.Contains(t, arguments, "hw-address")
	require.Empty(t, arguments["hw-address"])

	// Ensure that the DUID sent in the second command is empty ("00:00:00").
	arguments = agents.GetCommandArguments(1)
	require.NotNil(t, arguments)
	require.Contains(t, arguments, "duid")
	require.Equal(t, "00:00:00", arguments["duid"])

	// Simulate an error in the first response. The daemon returning an error should
	// be recorded, but the DHCPv6 lease should still be returned.
	agents = agentcommtest.NewFakeAgents(mockLeasesGetDeclinedErrors, nil)
	leases, erredDaemons, err = FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Len(t, erredDaemons, 1)
	require.Len(t, leases, 1)
}

// Test that Stork sends a single zero DUID to Kea when searching for declined
// leases and Kea version is less than 2.3.8.
func TestFindDeclinedLeasesPriorKea2_3_8(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine with the Kea daemons including both DHCPv4 and DHCPv6
	// daemon with the lease_cmds hooks library loaded.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTPS,
		},
	}

	// Create DHCPv4 daemon with version 2.3.7
	daemon4 := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)
	daemon4.Version = "2.3.7"
	err = daemon4.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon4)
	require.NoError(t, err)

	// Create DHCPv6 daemon with version 2.3.7
	daemon6 := dbmodel.NewDaemon(machine, daemonname.DHCPv6, true, accessPoints)
	daemon6.Version = "2.3.7"
	err = daemon6.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon6)
	require.NoError(t, err)

	// Simulate Kea returning 4 leases, one in the default state and three
	// in the declined state.
	agents := agentcommtest.NewFakeAgents(mockLeasesGetDeclined, nil)

	leases, erredDaemons, err := FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Empty(t, erredDaemons)

	// One lease should be ignored and three returned.
	require.Len(t, leases, 3)

	// Basic checks if expected leases were returned.
	for i, ipAddress := range []string{"192.0.2.2", "2001:db8:2::1", "2001:db8:2::2"} {
		expectedDaemon := daemon4
		if i > 0 {
			expectedDaemon = daemon6
		}

		require.EqualValues(t, expectedDaemon.ID, leases[i].DaemonID)
		require.NotNil(t, leases[i].Daemon)
		require.Equal(t, ipAddress, leases[i].IPAddress)
		require.EqualValues(t, keadata.LeaseStateDeclined, leases[i].State)
	}

	// Ensure that Stork has sent two commands, one to the DHCPv4 server and one
	// to the DHCPv6 server.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4GetByHWAddress, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[1].GetCommand())

	// Ensure that the hw-address sent in the first command is empty.
	arguments := agents.GetCommandArguments(0)
	require.NotNil(t, arguments)
	require.Contains(t, arguments, "hw-address")
	require.Empty(t, arguments["hw-address"])

	// Ensure that the DUID sent in the second command is empty ("0").
	arguments = agents.GetCommandArguments(1)
	require.NotNil(t, arguments)
	require.Contains(t, arguments, "duid")
	require.Equal(t, "0", arguments["duid"])

	// Simulate an error in the first response. The daemon returning an error should
	// be recorded, but the DHCPv6 lease should still be returned.
	agents = agentcommtest.NewFakeAgents(mockLeasesGetDeclinedErrors, nil)
	leases, erredDaemons, err = FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Len(t, erredDaemons, 1)
	require.Len(t, leases, 1)
}

// Test that a search for declined leases returns empty result when
// none of the servers uses lease_cmds hooks library.
func TestFindDeclinedLeasesNoLeaseCmds(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine with the Kea daemons including both DHCPv4 and DHCPv6
	// daemon without the lease_cmds hooks library loaded.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTP,
		},
	}

	// Create DHCPv4 daemon without lease_cmds hooks
	daemon4 := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)
	daemon4.Version = "3.1.5"
	err = daemon4.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon4)
	require.NoError(t, err)

	// Create DHCPv6 daemon without lease_cmds hooks
	daemon6 := dbmodel.NewDaemon(machine, daemonname.DHCPv6, true, accessPoints)
	daemon4.Version = "3.1.5"
	err = daemon6.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon6)
	require.NoError(t, err)

	agents := agentcommtest.NewFakeAgents(mockLeasesGetDeclined, nil)

	leases, erredDaemons, err := FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Empty(t, erredDaemons)
	require.Empty(t, leases)
}

// Ensure the FindDeclinedLeases call reports an error for Kea 3.1.1 through 3.1.4, where the "send a zero/emptystr hardware address/DUID" method was identified as a bug and fixed, and before the get-by-status API was added.
func TestFindDeclinedLeasesWithBrokenVersions(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine with the Kea daemons including both DHCPv4 and DHCPv6
	// daemon with the lease_cmds hooks library loaded.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTPS,
		},
	}

	// Create DHCPv4 daemon representing Kea 3.1.3 (broken zero-hwaddr method)
	daemon4 := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)
	daemon4.Version = "3.1.1"
	err = daemon4.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon4)
	require.NoError(t, err)

	// Create DHCPv6 daemon
	daemon6 := dbmodel.NewDaemon(machine, daemonname.DHCPv6, true, accessPoints)
	daemon6.Version = "3.1.4"
	err = daemon6.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon6)
	require.NoError(t, err)

	// Simulate Kea returning 4 leases, one in the default state and three
	// in the declined state.
	agents := agentcommtest.NewFakeAgents(mockLeasesGetByStatusDeclined, nil)

	leases, erredDaemons, err := FindDeclinedLeases(db, agents)
	require.NoError(t, err)

	// Ensure that Stork has sent no commands (because they won't work anyway).
	require.Len(t, agents.RecordedCommands, 0)

	// Ensure that both daemons are reported as errored.
	require.Len(t, erredDaemons, 2)

	// Ensure that Stork did not find any leases from daemons that can't answer.
	require.Len(t, leases, 0)
}

// Ensure the FindDeclinedLeases call works with Kea 3.1.5 and above, when the lease[46]-get-by-status API was added.
func TestFindDeclinedLeasesWithGetByStatusAPI(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine with the Kea daemons including both DHCPv4 and DHCPv6
	// daemon with the lease_cmds hooks library loaded.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTPS,
		},
	}

	// Create DHCPv4 daemon representing Kea 3.1.5
	daemon4 := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)
	daemon4.Version = "3.1.5"
	err = daemon4.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon4)
	require.NoError(t, err)

	// Create DHCPv6 daemon
	daemon6 := dbmodel.NewDaemon(machine, daemonname.DHCPv6, true, accessPoints)
	daemon6.Version = "3.1.5"
	err = daemon6.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon6)
	require.NoError(t, err)

	// Simulate Kea returning 4 leases, one in the default state and three
	// in the declined state.
	agents := agentcommtest.NewFakeAgents(mockLeasesGetByStatusDeclined, nil)

	leases, erredDaemons, err := FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Empty(t, erredDaemons)

	// One lease should be ignored and three returned.
	require.Len(t, leases, 3)

	// Basic checks if expected leases were returned.
	for i, ipAddress := range []string{"192.0.2.2", "2001:db8:2::1", "2001:db8:2::2"} {
		expectedDaemon := daemon4
		if i > 0 {
			expectedDaemon = daemon6
		}

		require.EqualValues(t, expectedDaemon.ID, leases[i].DaemonID)
		require.NotNil(t, leases[i].Daemon)
		require.Equal(t, ipAddress, leases[i].IPAddress)
		require.EqualValues(t, keadata.LeaseStateDeclined, leases[i].State)
	}

	// Ensure that Stork has sent two commands, one to the DHCPv4 server and one
	// to the DHCPv6 server.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4GetByState, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6GetByState, agents.RecordedCommands[1].GetCommand())

	// Ensure that the state in the command is "declined" (1).
	arguments := agents.GetCommandArguments(0)
	require.NotNil(t, arguments)
	require.Contains(t, arguments, "state")
	require.Equal(t, 1, arguments["state"])

	// Ensure that the state in the second command is "declined" (1).
	arguments = agents.GetCommandArguments(1)
	require.NotNil(t, arguments)
	require.Contains(t, arguments, "state")
	require.Equal(t, 1, arguments["state"])
}

// Test searching leases associated with a host reservation.
func TestFindLeasesByHostID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine1 := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine1)
	require.NoError(t, err)

	machine2 := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8081,
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)

	// Create DHCPv6 daemon on machine1 with lease_cmds hooks library loaded.
	accessPoints1 := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8000,
			Protocol: protocoltype.HTTPS,
		},
	}
	daemon1 := dbmodel.NewDaemon(machine1, daemonname.DHCPv6, true, accessPoints1)
	err = daemon1.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon1)
	require.NoError(t, err)

	// Create DHCPv4 and DHCPv6 daemons on machine2 with lease_cmds hooks library loaded.
	accessPoints2 := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     8001,
			Protocol: protocoltype.HTTP,
		},
	}

	daemon2v4 := dbmodel.NewDaemon(machine2, daemonname.DHCPv4, true, accessPoints2)
	err = daemon2v4.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon2v4)
	require.NoError(t, err)

	daemon2v6 := dbmodel.NewDaemon(machine2, daemonname.DHCPv6, true, accessPoints2)
	err = daemon2v6.SetKeaConfigFromJSON([]byte(`{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "libdhcp_lease_cmds.so"
				}
			]
		}
	}`))
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, daemon2v6)
	require.NoError(t, err)

	host := dbmodel.Host{
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{8, 8, 8, 8, 8, 8},
			},
			{
				Type:  "duid",
				Value: []byte{0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID:   daemon1.ID,
				DataSource: dbmodel.HostDataSourceConfig,
				IPReservations: []dbmodel.IPReservation{
					{
						Address: "192.0.2.1",
					},
					{
						Address: "2001:db8:2::1",
					},
					{
						Address: "2001:db8:0:0:2::/80",
					},
				},
			},
			{
				DaemonID:   daemon2v6.ID,
				DataSource: dbmodel.HostDataSourceConfig,
				IPReservations: []dbmodel.IPReservation{
					{
						Address: "192.0.2.1",
					},
					{
						Address: "2001:db8:2::1",
					},
					{
						Address: "2001:db8:0:0:2::/80",
					},
				},
			},
		},
	}
	err = dbmodel.AddHost(db, &host)
	require.NoError(t, err)

	// Expecting the following commands and responses:
	// - lease6-get (by address) to daemon1 - returning empty response
	// - lease6-get (by prefix) to daemon1  - returning the lease 2001:db8:0:0:2::"
	// - lease4-get to daemon2v4 - returning the lease 192.0.2.1
	// - lease6-get (by address) to daemon2v6 - returning empty response
	// - lease6-get (by prefix) to daemon2v6 - returning empty response
	agents := agentcommtest.NewKeaFakeAgents(mockLeases6GetEmpty, mockLease6GetByPrefix, mockLease4Get, mockLeases6GetEmpty)

	leases, conflicts, erredDaemons, err := FindLeasesByHostID(db, agents, host.ID)
	require.NoError(t, err)
	require.Empty(t, conflicts)
	require.Empty(t, erredDaemons)
	require.Len(t, leases, 2)
	require.EqualValues(t, 1, leases[0].ID)
	require.Equal(t, "2001:db8:0:0:2::", leases[0].IPAddress)
	require.EqualValues(t, 2, leases[1].ID)
	require.Equal(t, "192.0.2.1", leases[1].IPAddress)
	require.Len(t, agents.RecordedCommands, 5)

	// Expecting the following commands and responses:
	// - lease6-get (by address) to daemon1 - returning the lease 2001:db8:2::1
	// - lease6-get (by prefix) to daemon1  - returning empty response
	// - lease4-get to daemon2v4 - returning empty response
	// - lease6-get (by address) to daemon2v6 - returning empty response
	// - lease6-get (by prefix) to daemon2v6 - returning empty response
	agents = agentcommtest.NewKeaFakeAgents(mockLease6GetByIPAddress, mockLeases6GetEmpty, mockLeases4GetEmpty, mockLeases6GetEmpty)

	leases, conflicts, erredDaemons, err = FindLeasesByHostID(db, agents, host.ID)
	require.NoError(t, err)
	require.Empty(t, conflicts)
	require.Empty(t, erredDaemons)
	require.Len(t, leases, 1)
	require.EqualValues(t, 1, leases[0].ID)
	require.Equal(t, "2001:db8:2::1", leases[0].IPAddress)
	require.Len(t, agents.RecordedCommands, 5)

	// Expecting the following commands and responses:
	// - lease6-get (by address) to daemon1 - returning an error
	// - lease4-get to daemon2v4 - returning the lease 192.0.2.1
	// - lease6-get (by address) to daemon2v6 - returning the lease 2001:db8:2::1
	// - lease6-get (by prefix) to daemon2v6 - returning the lease 2001:db8:0:0:2::
	agents = agentcommtest.NewKeaFakeAgents(mockLease6GetError, mockLease4Get, mockLease6GetByIPAddress, mockLease6GetByPrefix)

	leases, conflicts, erredDaemons, err = FindLeasesByHostID(db, agents, host.ID)
	require.NoError(t, err)
	require.Empty(t, conflicts)
	require.Len(t, erredDaemons, 1)
	require.Len(t, leases, 3)
	require.Len(t, agents.RecordedCommands, 4)

	// Expecting the following commands and responses:
	// - lease6-get (by address) to daemon1 - returning an error
	// - lease4-get to daemon2v4 - returning an error
	// - lease6-get (by address) to daemon2v6 - returning an error
	agents = agentcommtest.NewKeaFakeAgents(mockLease6GetError)

	leases, conflicts, erredDaemons, err = FindLeasesByHostID(db, agents, host.ID)
	require.NoError(t, err)
	require.Empty(t, conflicts)
	require.Len(t, erredDaemons, 3)
	require.Empty(t, leases)
	require.Len(t, agents.RecordedCommands, 3)
}

// Test that lease conflicts with host reservations are detected for reservations
// using hw-address, client-id and DUID.
func TestFindHostLeaseConflicts(t *testing.T) {
	host := &dbmodel.Host{
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{8, 8, 8, 8, 8, 8},
			},
			{
				Type:  "client-id",
				Value: []byte{1, 1, 1, 1},
			},
			{
				Type:  "duid",
				Value: []byte{0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42},
			},
		},
	}

	// None of the leases matches the host reservation.
	leases := []dbmodel.Lease{
		{
			Lease: keadata.Lease{
				HWAddress: "010203040506",
			},
		},
		{
			Lease: keadata.Lease{
				ClientID: "02020202",
			},
		},
		{
			Lease: keadata.Lease{
				DUID: "4343434343434343",
			},
		},
	}
	conflicts := findHostLeaseConflicts(host, leases)
	require.Len(t, conflicts, 3)
	require.Contains(t, conflicts, leases[0].ID)
	require.Contains(t, conflicts, leases[1].ID)
	require.Contains(t, conflicts, leases[2].ID)

	// First lease matches the host reservation.
	leases = []dbmodel.Lease{
		{
			Lease: keadata.Lease{
				HWAddress: "08:08:08:08:08:08",
			},
		},
		{
			Lease: keadata.Lease{
				ClientID: "02020202",
			},
		},
		{
			Lease: keadata.Lease{
				DUID: "4343434343434343",
			},
		},
	}
	conflicts = findHostLeaseConflicts(host, leases)
	require.Len(t, conflicts, 2)
	require.Contains(t, conflicts, leases[1].ID)
	require.Contains(t, conflicts, leases[2].ID)

	// Second lease matches the host reservation.
	leases = []dbmodel.Lease{
		{
			Lease: keadata.Lease{
				HWAddress: "09:09:09:09:09:09",
			},
		},
		{
			Lease: keadata.Lease{
				ClientID: "01010101",
			},
		},
		{
			Lease: keadata.Lease{
				DUID: "4343434343434343",
			},
		},
	}
	conflicts = findHostLeaseConflicts(host, leases)
	require.Len(t, conflicts, 2)
	require.Contains(t, conflicts, leases[0].ID)
	require.Contains(t, conflicts, leases[2].ID)

	// Third lease matches the host reservation.
	leases = []dbmodel.Lease{
		{
			Lease: keadata.Lease{
				HWAddress: "09:09:09:09:09:09",
			},
		},
		{
			Lease: keadata.Lease{
				ClientID: "02020202",
			},
		},
		{
			Lease: keadata.Lease{
				DUID: "4242424242424242",
			},
		},
	}
	conflicts = findHostLeaseConflicts(host, leases)
	require.Len(t, conflicts, 2)
	require.Contains(t, conflicts, leases[0].ID)
	require.Contains(t, conflicts, leases[1].ID)

	// All leases contain one of the identifiers that matches the host reservation.
	leases = []dbmodel.Lease{
		{
			Lease: keadata.Lease{
				HWAddress: "09:09:09:09:09:09",
				ClientID:  "02020202",
				DUID:      "4242424242424242",
			},
		},
		{
			Lease: keadata.Lease{
				HWAddress: "09:09:09:09:09:09",
				ClientID:  "01010101",
				DUID:      "4343434343434343",
			},
		},
		{
			Lease: keadata.Lease{
				HWAddress: "09:09:09:09:09:09",
				ClientID:  "02020202",
				DUID:      "4242424242424242",
			},
		},
	}
	conflicts = findHostLeaseConflicts(host, leases)
	require.Empty(t, conflicts)
}

// Test that conflict detection between leases and and host reservation
// is not triggered when host reservation contains flex-id and/or circuit-id.
func TestFindHostLeaseConflictsSkipDetection(t *testing.T) {
	tests := []string{"flex-id", "circuit-id"}
	for i := range tests {
		idName := tests[i]
		t.Run(idName, func(t *testing.T) {
			host := &dbmodel.Host{
				HostIdentifiers: []dbmodel.HostIdentifier{
					{
						Type:  "hw-address",
						Value: []byte{8, 8, 8, 8, 8, 8},
					},
					{
						Type:  idName,
						Value: []byte{8, 8, 8, 8, 8, 8},
					},
				},
			}

			leases := []dbmodel.Lease{
				{
					Lease: keadata.Lease{
						HWAddress: "010203040506",
					},
				},
			}
			conflicts := findHostLeaseConflicts(host, leases)
			require.Empty(t, conflicts)
		})
	}
}
