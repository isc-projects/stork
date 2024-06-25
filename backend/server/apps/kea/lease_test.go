package kea

import (
	"testing"

	require "github.com/stretchr/testify/require"

	keactrl "isc.org/stork/appctrl/kea"
	keadata "isc.org/stork/appdata/kea"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Generates a success mock response to commands fetching a single
// DHCPv4 lease.
func mockLease4Get(callNo int, responses []interface{}) {
	json := []byte(`[
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
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4Get, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a success mock response to commands fetching a single
// DHCPv4 lease with invalid user context data.
func mockLease4GetInvalidJSON(callNo int, responses []interface{}) {
	json := []byte(`[
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
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4Get, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a response to lease4-get command. The first time it is
// called it returns an error response. The second time it returns
// a lease. It is useful to simulate tracking erred communication
// with selected servers.
func mockLease4GetFirstCallError(callNo int, responses []interface{}) {
	var json []byte
	if callNo == 0 {
		json = []byte(`[
            {
                "result": 1,
                "text": "Lease erred",
                "arguments": { }
            }
        ]`)
		command := keactrl.NewCommandBase(keactrl.Lease4Get, keactrl.DHCPv4)
		_ = keactrl.UnmarshalResponseList(command, json, responses[0])
		return
	}

	json = []byte(`[
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
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4Get, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a success mock response to commands fetching a single
// DHCPv6 lease by IPv6 address.
func mockLease6GetByIPAddress(callNo int, responses []interface{}) {
	json := []byte(`[
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
                "user-context": { "ISC": { "client-classes": [ "ALL", "HA_primary", "UNKNOWN" ] }}
            }
        }
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4Get, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a success mock response to commands fetching a single
// DHCPv6 lease by IPv6 address.
func mockLease6GetByPrefix(callNo int, responses []interface{}) {
	json := []byte(`[
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
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease6Get, keactrl.DHCPv6)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a success mock response to commands fetching a single
// DHCPv6 lease with invalid user context data.
func mockLease6GetInvalidJSON(callNo int, responses []interface{}) {
	json := []byte(`[
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
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease6Get, keactrl.DHCPv6)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a success mock response to commands fetching multiple
// DHCPv4 leases.
func mockLeases4Get(callNo int, responses []interface{}) {
	json := []byte(`[
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
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4GetByHWAddress, keactrl.DHCPv4)

	for i := range responses {
		_ = keactrl.UnmarshalResponseList(command, json, responses[i])
	}
}

// Generates a mock response to lease4-get-by-hw-address and lease4-get-by-client-id
// combined in a single gRPC command. The first response is successful, the second
// response indicates an error.
func mockLeases4GetSecondError(callNo int, responses []interface{}) {
	// Response to lease4-get-by-hw-address.
	json := []byte(`[
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
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4GetByHWAddress, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])

	// Response to lease4-get-by-client-id.
	json = []byte(`[
        {
            "result": 1,
            "text": "Leases erred",
            "arguments": { }
        }
    ]`)
	command = keactrl.NewCommandBase(keactrl.Lease4GetByClientID, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[1])
}

// Generates a mock empty response to commands fetching DHCPv4 leases.
func mockLeases4GetEmpty(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 3,
            "text": "No lease found."
        }
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4Get, keactrl.DHCPv4)
	for i := range responses {
		_ = keactrl.UnmarshalResponseList(command, json, responses[i])
	}
}

// Generates a mock empty response to commands fetching DHCPv6 leases.
func mockLeases6GetEmpty(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 3,
            "text": "No lease found."
        }
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease6Get, keactrl.DHCPv6)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a success mock response to commands fetching multiple
// DHCPv6 leases.
func mockLeases6Get(callNo int, responses []interface{}) {
	json := []byte(`[
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
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease6GetByDUID, keactrl.DHCPv6)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates responses to declined leases search. First response comprises
// two DHCPv4 leases, one in the default state and one in the declined state.
// Stork should ignore the lease in the default state. The second response
// contains two declined DHCPv6 leases.
func mockLeasesGetDeclined(callNo int, responses []interface{}) {
	json := []byte(`[
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
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4GetByHWAddress, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])

	json = []byte(`[
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
    ]`)
	command = keactrl.NewCommandBase(keactrl.Lease6GetByDUID, keactrl.DHCPv6)
	_ = keactrl.UnmarshalResponseList(command, json, responses[1])
}

func mockLeasesGetDeclinedErrors(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 1,
            "text": "Leases search erred"
        }
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease4GetByHWAddress, keactrl.DHCPv4)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])

	json = []byte(`[
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
    ]`)
	command = keactrl.NewCommandBase(keactrl.Lease6GetByDUID, keactrl.DHCPv6)
	_ = keactrl.UnmarshalResponseList(command, json, responses[1])
}

// Generate an error mock response to a command fetching lease by an IPv6
// address.
func mockLease6GetError(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 1,
            "text": "Getting an lease erred."
        }
    ]`)
	command := keactrl.NewCommandBase(keactrl.Lease6Get, keactrl.DHCPv6)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Test the success scenario in sending lease4-get command to Kea.
func TestGetLease4ByIPAddress(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLease4Get, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := &dbmodel.App{
		ID:           1,
		AccessPoints: accessPoints,
	}

	lease, err := GetLease4ByIPAddress(agents, app, "192.0.2.3")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, app.ID, lease.AppID)
	require.NotNil(t, lease.App)
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

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := &dbmodel.App{
		ID:           2,
		AccessPoints: accessPoints,
	}

	lease, err := GetLease6ByIPAddress(agents, app, "IA_NA", "2001:db8:2::1")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, app.ID, lease.AppID)
	require.NotNil(t, lease.App)
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

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := &dbmodel.App{
		ID:           3,
		AccessPoints: accessPoints,
	}

	lease, err := GetLease6ByIPAddress(agents, app, "IA_PD", "2001:db8:0:0:2::")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, app.ID, lease.AppID)
	require.NotNil(t, lease.App)
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

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	lease, err := GetLease4ByIPAddress(agents, app, "192.0.2.3")
	require.NoError(t, err)
	require.Nil(t, lease)
}

// Test the scenario in sending lease6-get command to Kea when the lease
// is not found.
func TestGetLease6ByIPAddressEmpty(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	lease, err := GetLease6ByIPAddress(agents, app, "IA_NA", "2001:db8:1::2")
	require.NoError(t, err)
	require.Nil(t, lease)
}

// Test success scenarios in sending lease4-get-by-hw-address, lease4-get-by-client-id
// and lease4-get-by-hostname commands to Kea.
func TestGetLeases4(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4Get, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := &dbmodel.App{
		ID:           4,
		AccessPoints: accessPoints,
	}

	tests := []struct {
		name          string
		function      func(agentcomm.ConnectedAgents, *dbmodel.App, string) ([]dbmodel.Lease, error)
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
			leases, err := testedFunc(agents, app, propertyValue)
			require.NoError(t, err)
			require.Len(t, leases, 1)

			lease := leases[0]
			require.EqualValues(t, app.ID, lease.AppID)
			require.NotNil(t, lease.App)
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

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := &dbmodel.App{
		ID:           5,
		AccessPoints: accessPoints,
	}

	tests := []struct {
		name          string
		function      func(agentcomm.ConnectedAgents, *dbmodel.App, string) ([]dbmodel.Lease, error)
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
			leases, err := testedFunc(agents, app, propertyValue)
			require.NoError(t, err)
			require.Len(t, leases, 2)

			lease := leases[0]
			require.EqualValues(t, app.ID, lease.AppID)
			require.NotNil(t, lease.App)
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
			require.EqualValues(t, app.ID, lease.AppID)
			require.NotNil(t, lease.App)
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

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	leases, err := GetLeases4ByHWAddress(agents, app, "000000000000")
	require.NoError(t, err)
	require.Empty(t, leases)

	// Ensure that MAC address was converted to the format expected by Kea.
	arguments := agents.RecordedCommands[0].(*keactrl.Command).Arguments
	require.NotNil(t, arguments)
	require.Contains(t, arguments.(map[string]interface{}), "hw-address")
	require.Equal(t, "00:00:00:00:00:00", (arguments.(map[string]interface{}))["hw-address"])
}

// Test the scenario in sending lease6-get-by-hostname command to Kea when
// no lease is found.
func TestGetLeases6Empty(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases6GetEmpty, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	leases, err := GetLeases6ByHostname(agents, app, "myhost")
	require.NoError(t, err)
	require.Empty(t, leases)
}

// Test sending multiple combined commands when one of the commands fails. The
// function should still return some leases but it should also return the warns
// flag to indicate that there were issues.
func TestGetLeasesByPropertiesSecondError(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4GetSecondError, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := &dbmodel.App{
		ID:           4,
		AccessPoints: accessPoints,
	}

	leases, warns, err := getLeasesByProperties(agents, app, "42:42:42:42:42:42:42:42", "lease4-get-by-hw-address", "lease4-get-by-client-id")
	require.NoError(t, err)
	require.True(t, warns)
	require.Len(t, leases, 1)

	lease := leases[0]
	require.EqualValues(t, app.ID, lease.AppID)
	require.NotNil(t, lease.App)
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

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := &dbmodel.App{
		ID:           1,
		AccessPoints: accessPoints,
	}

	lease, err := GetLease4ByIPAddress(agents, app, "192.0.2.3")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, app.ID, lease.AppID)
	require.NotNil(t, lease.App)
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

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := &dbmodel.App{
		ID:           2,
		AccessPoints: accessPoints,
	}

	lease, err := GetLease6ByIPAddress(agents, app, "IA_NA", "2001:db8:2::1")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, app.ID, lease.AppID)
	require.NotNil(t, lease.App)
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

	// Add first machine with the Kea app including both DHCPv4 and DHCPv6
	// daemon with the lease_cmds hooks library loaded.
	machine1 := &dbmodel.Machine{
		ID:        0,
		Address:   "machine1",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine1)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app1 := &dbmodel.App{
		MachineID:    machine1.ID,
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
	_, err = dbmodel.AddApp(db, app1)
	require.NoError(t, err)

	// Add second machine with the Kea app with lease_cmds hooks library loaded by
	// the DHCPv4 daemon.
	machine2 := &dbmodel.Machine{
		ID:        0,
		Address:   "machine2",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)

	accessPoints = []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app2 := &dbmodel.App{
		MachineID:    machine2.ID,
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
	_, err = dbmodel.AddApp(db, app2)
	require.NoError(t, err)

	// Add third machine with the Kea app with lease_cmds hooks library loaded by
	// the DHCPv6 daemon.
	machine3 := &dbmodel.Machine{
		ID:        0,
		Address:   "machine3",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, machine3)
	require.NoError(t, err)

	accessPoints = []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app3 := &dbmodel.App{
		MachineID:    machine3.ID,
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
	_, err = dbmodel.AddApp(db, app3)
	require.NoError(t, err)

	agents := agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	//  Find lease by IPv4 address.
	_, erredApps, err := FindLeases(db, agents, "192.0.2.3")
	require.NoError(t, err)
	require.Empty(t, erredApps)

	// It should have sent lease4-get command to first and second Kea.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4Get, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease4Get, agents.RecordedCommands[1].GetCommand())

	agents = agentcommtest.NewFakeAgents(mockLease4GetFirstCallError, nil)

	// Test the case when one of the servers returns an error.
	_, erredApps, err = FindLeases(db, agents, "192.0.2.3")
	require.NoError(t, err)
	require.Len(t, erredApps, 1)
	require.NotNil(t, erredApps[0])
	require.EqualValues(t, app1.ID, erredApps[0].ID)

	// It should have sent lease4-get command to first and second Kea.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4Get, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease4Get, agents.RecordedCommands[1].GetCommand())

	agents = agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	// Find lease by IPv6 address.
	_, erredApps, err = FindLeases(db, agents, "2001:db8:1::")
	require.NoError(t, err)
	require.Empty(t, erredApps)

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
	_, erredApps, err = FindLeases(db, agents, "010203040506")
	require.NoError(t, err)
	require.Empty(t, erredApps)

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
	arguments := agents.RecordedCommands[0].(*keactrl.Command).Arguments
	require.NotNil(t, arguments)
	require.Contains(t, arguments.(map[string]interface{}), "hw-address")
	require.Equal(t, "01:02:03:04:05:06", (arguments.(map[string]interface{}))["hw-address"])

	agents = agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	// Find lease by hostname.
	_, erredApps, err = FindLeases(db, agents, "myhost")
	require.NoError(t, err)
	require.Empty(t, erredApps)

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

	appOld := &dbmodel.App{
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeKea,
		Meta: dbmodel.AppMeta{
			Version: "2.0.2",
		},
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:      dbmodel.AccessPointControl,
				Address:   "localhost",
				Port:      8000,
				MachineID: machine.ID,
			},
		},
		Daemons: []*dbmodel.Daemon{
			{
				Name: dbmodel.DaemonNameDHCPv6,
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
	_, _ = dbmodel.AddApp(db, appOld)

	appModern := &dbmodel.App{
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeKea,
		Meta: dbmodel.AppMeta{
			Version: "2.7.2",
		},
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:      dbmodel.AccessPointControl,
				Address:   "localhost",
				Port:      8001,
				MachineID: machine.ID,
			},
		},
		Daemons: []*dbmodel.Daemon{
			{
				Name: dbmodel.DaemonNameDHCPv6,
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
	_, _ = dbmodel.AddApp(db, appModern)

	agents := agentcommtest.NewFakeAgents(mockLeases6GetEmpty, nil)

	// Act
	_, erredApps, err := FindLeases(db, agents, "0102")

	// Assert
	require.NoError(t, err)
	require.Empty(t, erredApps)

	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6GetByHostname, agents.RecordedCommands[1].GetCommand())
}

// Test declined leases search mechanism. It verifies a positive scenario in which
// the DHCPv4 server returns two leases (one is in a default state and one is in the
// declined state), and the DHCPv6 server returns two declined leases. The DHCPv4 lease
// in the default state should be ignored. The remaining three leases should be
// returned. The test also verifies that the commands sent to Kea are formatted
// correctly, i.e. contain empty MAC address and empty DUID. Finally, the test
// verifies that erred apps are returned if any of the commands returns an error.
func TestFindDeclinedLeases(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine with the Kea app including both DHCPv4 and DHCPv6
	// daemon with the lease_cmds hooks library loaded.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

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

	// Simulate Kea returning 4 leases, one in the default state and three
	// in the declined state.
	agents := agentcommtest.NewFakeAgents(mockLeasesGetDeclined, nil)

	leases, erredApps, err := FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Empty(t, erredApps)

	// One lease should be ignored and three returned.
	require.Len(t, leases, 3)

	// Basic checks if expected leases were returned.
	for i, ipAddress := range []string{"192.0.2.2", "2001:db8:2::1", "2001:db8:2::2"} {
		require.EqualValues(t, app.ID, leases[i].AppID)
		require.NotNil(t, leases[i].App)
		require.Equal(t, ipAddress, leases[i].IPAddress)
		require.EqualValues(t, keadata.LeaseStateDeclined, leases[i].State)
	}

	// Ensure that Stork has sent two commands, one to the DHCPv4 server and one
	// to the DHCPv6 server.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4GetByHWAddress, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[1].GetCommand())

	// Ensure that the hw-address sent in the first command is empty.
	arguments := agents.RecordedCommands[0].(*keactrl.Command).Arguments
	require.NotNil(t, arguments)
	require.Contains(t, arguments.(map[string]interface{}), "hw-address")
	require.Empty(t, (arguments.(map[string]interface{}))["hw-address"])

	// Ensure that the DUID sent in the second command is empty ("00:00:00").
	arguments = agents.RecordedCommands[1].(*keactrl.Command).Arguments
	require.NotNil(t, arguments)
	require.Contains(t, arguments.(map[string]interface{}), "duid")
	require.Equal(t, "00:00:00", (arguments.(map[string]interface{}))["duid"])

	// Simulate an error in the first response. The app returning an error should
	// be recorded, but the DHCPv6 lease should still be returned.
	agents = agentcommtest.NewFakeAgents(mockLeasesGetDeclinedErrors, nil)
	leases, erredApps, err = FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Len(t, erredApps, 1)
	require.Len(t, leases, 1)
}

// Test that Stork sends a single zero DUID to Kea when searching for declined
// leases and Kea version is less than 2.3.8.
func TestFindDeclinedLeasesPriorKea2_3_8(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine with the Kea app including both DHCPv4 and DHCPv6
	// daemon with the lease_cmds hooks library loaded.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := &dbmodel.App{
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Meta: dbmodel.AppMeta{
			Version: "2.3.7",
		},
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

	// Simulate Kea returning 4 leases, one in the default state and three
	// in the declined state.
	agents := agentcommtest.NewFakeAgents(mockLeasesGetDeclined, nil)

	leases, erredApps, err := FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Empty(t, erredApps)

	// One lease should be ignored and three returned.
	require.Len(t, leases, 3)

	// Basic checks if expected leases were returned.
	for i, ipAddress := range []string{"192.0.2.2", "2001:db8:2::1", "2001:db8:2::2"} {
		require.EqualValues(t, app.ID, leases[i].AppID)
		require.NotNil(t, leases[i].App)
		require.Equal(t, ipAddress, leases[i].IPAddress)
		require.EqualValues(t, keadata.LeaseStateDeclined, leases[i].State)
	}

	// Ensure that Stork has sent two commands, one to the DHCPv4 server and one
	// to the DHCPv6 server.
	require.Len(t, agents.RecordedCommands, 2)
	require.Equal(t, keactrl.Lease4GetByHWAddress, agents.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.Lease6GetByDUID, agents.RecordedCommands[1].GetCommand())

	// Ensure that the hw-address sent in the first command is empty.
	arguments := agents.RecordedCommands[0].(*keactrl.Command).Arguments
	require.NotNil(t, arguments)
	require.Contains(t, arguments.(map[string]interface{}), "hw-address")
	require.Empty(t, (arguments.(map[string]interface{}))["hw-address"])

	// Ensure that the DUID sent in the second command is empty ("0").
	arguments = agents.RecordedCommands[1].(*keactrl.Command).Arguments
	require.NotNil(t, arguments)
	require.Contains(t, arguments.(map[string]interface{}), "duid")
	require.Equal(t, "0", (arguments.(map[string]interface{}))["duid"])

	// Simulate an error in the first response. The app returning an error should
	// be recorded, but the DHCPv6 lease should still be returned.
	agents = agentcommtest.NewFakeAgents(mockLeasesGetDeclinedErrors, nil)
	leases, erredApps, err = FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Len(t, erredApps, 1)
	require.Len(t, leases, 1)
}

// Test that a search for declined leases returns empty result when
// none of the servers uses lease_cmds hooks library.
func TestFindDeclinedLeasesNoLeaseCmds(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine with the Kea app including both DHCPv4 and DHCPv6
	// daemon without the lease_cmds hooks library loaded.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "machine",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := &dbmodel.App{
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name: dbmodel.DaemonNameDHCPv4,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: dbmodel.NewKeaConfig(&map[string]interface{}{
						"Dhcp4": map[string]interface{}{},
					}),
				},
			},
			{
				Name: dbmodel.DaemonNameDHCPv6,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: dbmodel.NewKeaConfig(&map[string]interface{}{
						"Dhcp6": map[string]interface{}{},
					}),
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	agents := agentcommtest.NewFakeAgents(mockLeasesGetDeclined, nil)

	leases, erredApps, err := FindDeclinedLeases(db, agents)
	require.NoError(t, err)
	require.Empty(t, erredApps)
	require.Empty(t, leases)
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

	// Create Kea apps with lease_cmds hooks library loaded.
	accessPoints1 := []*dbmodel.AccessPoint{}
	accessPoints1 = dbmodel.AppendAccessPoint(accessPoints1, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app1 := dbmodel.App{
		MachineID:    machine1.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints1,
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
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app1)
	require.NoError(t, err)

	// Create Kea apps with lease_cmds hooks library loaded.
	accessPoints2 := []*dbmodel.AccessPoint{}
	accessPoints2 = dbmodel.AppendAccessPoint(accessPoints2, dbmodel.AccessPointControl, "localhost", "", 8001, false)
	app2 := dbmodel.App{
		MachineID:    machine2.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints2,
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
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app2)
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
				DaemonID:   app1.Daemons[0].ID,
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
				DaemonID:   app2.Daemons[1].ID,
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
	// - lease6-get (by address) to app1 - returning empty response
	// - lease6-get (by prefix) to app1  - returning the lease 2001:db8:0:0:2::"
	// - lease4-get to app2 - returning the lease 192.0.2.1
	// - lease6-get (by address) to app2 - returning empty response
	// - lease6-get (by prefix) to app2 - returning empty response
	agents := agentcommtest.NewKeaFakeAgents(mockLeases6GetEmpty, mockLease6GetByPrefix, mockLease4Get, mockLeases6GetEmpty)

	leases, conflicts, erredApps, err := FindLeasesByHostID(db, agents, host.ID)
	require.NoError(t, err)
	require.Empty(t, conflicts)
	require.Empty(t, erredApps)
	require.Len(t, leases, 2)
	require.EqualValues(t, 1, leases[0].ID)
	require.Equal(t, "2001:db8:0:0:2::", leases[0].IPAddress)
	require.EqualValues(t, 2, leases[1].ID)
	require.Equal(t, "192.0.2.1", leases[1].IPAddress)
	require.Len(t, agents.RecordedCommands, 5)

	// Expecting the following commands and responses:
	// - lease6-get (by address) to app1 - returning the lease 2001:db8:2::1
	// - lease6-get (by prefix) to app1  - returning empty response
	// - lease4-get to app2 - returning empty response
	// - lease6-get (by address) to app2 - returning empty response
	// - lease6-get (by prefix) to app2 - returning empty response
	agents = agentcommtest.NewKeaFakeAgents(mockLease6GetByIPAddress, mockLeases6GetEmpty, mockLeases4GetEmpty, mockLeases6GetEmpty)

	leases, conflicts, erredApps, err = FindLeasesByHostID(db, agents, host.ID)
	require.NoError(t, err)
	require.Empty(t, conflicts)
	require.Empty(t, erredApps)
	require.Len(t, leases, 1)
	require.EqualValues(t, 1, leases[0].ID)
	require.Equal(t, "2001:db8:2::1", leases[0].IPAddress)
	require.Len(t, agents.RecordedCommands, 5)

	// Expecting the following commands and responses:
	// - lease6-get (by address) to app1 - returning an error
	// - lease4-get to app2 - returning the lease 192.0.2.1
	// - lease6-get (by address) to app2 - returning the lease 2001:db8:2::1
	// - lease6-get (by prefix) to app2 - returning the lease 2001:db8:0:0:2::
	agents = agentcommtest.NewKeaFakeAgents(mockLease6GetError, mockLease4Get, mockLease6GetByIPAddress, mockLease6GetByPrefix)

	leases, conflicts, erredApps, err = FindLeasesByHostID(db, agents, host.ID)
	require.NoError(t, err)
	require.Empty(t, conflicts)
	require.Len(t, erredApps, 1)
	require.Len(t, leases, 3)
	require.Len(t, agents.RecordedCommands, 4)

	// Expecting the following commands and responses:
	// - lease6-get (by address) to app1 - returning an error
	// - lease4-get to app2 - returning an error
	// - lease6-get (by address) to app2 - returning an error
	agents = agentcommtest.NewKeaFakeAgents(mockLease6GetError)

	leases, conflicts, erredApps, err = FindLeasesByHostID(db, agents, host.ID)
	require.NoError(t, err)
	require.Empty(t, conflicts)
	require.Len(t, erredApps, 2)
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
