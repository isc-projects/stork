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
                "valid-lft": 3600
            }
        }
    ]`)
	daemons, _ := keactrl.NewDaemons("dhcp4")
	command, _ := keactrl.NewCommand("lease4-get", daemons, nil)
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
                "valid-lft": 3600
            }
        }
    ]`)
	daemons, _ := keactrl.NewDaemons("dhcp4")
	command, _ := keactrl.NewCommand("lease4-get", daemons, nil)
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
                "valid-lft": 3600
            }
        }
    ]`)
	daemons, _ := keactrl.NewDaemons("dhcp6")
	command, _ := keactrl.NewCommand("lease6-get", daemons, nil)
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
	daemons, _ := keactrl.NewDaemons("dhcp4")
	command, _ := keactrl.NewCommand("lease4-get-by-hw-address", daemons, nil)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a mock empty response to commands fetching DHCPv4 leases.
func mockLeases4GetEmpty(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 3,
            "text": "No lease found."
        }
    ]`)
	daemons, _ := keactrl.NewDaemons("dhcp4")
	command, _ := keactrl.NewCommand("lease4-get", daemons, nil)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Generates a mock empty response to commands fetching DHCPv6 leases.
func mockLeases6GetEmpty(callNo int, responses []interface{}) {
	json := []byte(`[
        {
            "result": 3,
            "text": "No lease found."
        }
    ]`)
	daemons, _ := keactrl.NewDaemons("dhcp6")
	command, _ := keactrl.NewCommand("lease6-get", daemons, nil)
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
	daemons, _ := keactrl.NewDaemons("dhcp6")
	command, _ := keactrl.NewCommand("lease6-get-by-duid", daemons, nil)
	_ = keactrl.UnmarshalResponseList(command, json, responses[0])
}

// Test the success scenario in sending lease4-get command to Kea.
func TestGetLease4ByIPAddress(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLease4Get, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	lease, err := GetLease4ByIPAddress(agents, app, "192.0.2.3")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.Equal(t, "42:42:42:42:42:42:42:42", lease.ClientID)
	require.EqualValues(t, 12345678, lease.Cltt)
	require.False(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Equal(t, "myhost.example.com.", lease.Hostname)
	require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
	require.Equal(t, "192.0.2.1", lease.IPAddress)
	require.EqualValues(t, 0, lease.State)
	require.EqualValues(t, 44, lease.SubnetID)
	require.EqualValues(t, 3600, lease.ValidLifetime)
}

// Test the success scenario in sending lease6-get command to Kea to get
// a lease by IPv6 address.
func TestGetLease6ByIPAddress(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLease6GetByIPAddress, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	lease, err := GetLease6ByIPAddress(agents, app, "IA_NA", "2001:db8:2::1")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, 12345678, lease.Cltt)
	require.Equal(t, "42:42:42:42:42:42:42:42", lease.DUID)
	require.False(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Equal(t, "myhost.example.com.", lease.Hostname)
	require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
	require.EqualValues(t, 1, lease.IAID)
	require.Equal(t, "2001:db8:2::1", lease.IPAddress)
	require.EqualValues(t, 500, lease.PreferredLifetime)
	require.EqualValues(t, 0, lease.State)
	require.EqualValues(t, 44, lease.SubnetID)
	require.Equal(t, "IA_NA", lease.Type)
	require.EqualValues(t, 3600, lease.ValidLifetime)
}

// Test the success scenario in sending lease6-get command to Kea to get
// a lease by IPv6 prefix.
func TestGetLease6ByPrefix(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLease6GetByPrefix, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	lease, err := GetLease6ByIPAddress(agents, app, "IA_PD", "2001:db8:0:0:2::")
	require.NoError(t, err)
	require.NotNil(t, lease)

	require.EqualValues(t, 12345678, lease.Cltt)
	require.Equal(t, "42:42:42:42:42:42:42:42", lease.DUID)
	require.False(t, lease.FqdnFwd)
	require.True(t, lease.FqdnRev)
	require.Empty(t, lease.Hostname)
	require.Empty(t, lease.HWAddress)
	require.EqualValues(t, 1, lease.IAID)
	require.Equal(t, "2001:db8:0:0:2::", lease.IPAddress)
	require.EqualValues(t, 500, lease.PreferredLifetime)
	require.EqualValues(t, 80, lease.PrefixLength)
	require.EqualValues(t, 0, lease.State)
	require.EqualValues(t, 44, lease.SubnetID)
	require.Equal(t, "IA_PD", lease.Type)
	require.EqualValues(t, 3600, lease.ValidLifetime)
}

// Test the scenario in sending lease4-get command to Kea when the lease
// is not found.
func TestGetLease4ByIPAddressEmpty(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
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
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
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
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	tests := []struct {
		name          string
		function      func(agentcomm.ConnectedAgents, *dbmodel.App, string) ([]keadata.Lease4, error)
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
			require.Equal(t, "42:42:42:42:42:42:42:42", lease.ClientID)
			require.EqualValues(t, 12345678, lease.Cltt)
			require.False(t, lease.FqdnFwd)
			require.True(t, lease.FqdnRev)
			require.Equal(t, "myhost.example.com.", lease.Hostname)
			require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
			require.Equal(t, "192.0.2.1", lease.IPAddress)
			require.EqualValues(t, 0, lease.State)
			require.EqualValues(t, 44, lease.SubnetID)
			require.EqualValues(t, 3600, lease.ValidLifetime)
		})
	}
}

// Test success scenarios in sending lease6-get-by-duid, lease6-get-by-hostname
// commands to Kea.
func TestGetLeases6(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases6Get, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	tests := []struct {
		name          string
		function      func(agentcomm.ConnectedAgents, *dbmodel.App, string) ([]keadata.Lease6, error)
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
			require.EqualValues(t, 12345678, lease.Cltt)
			require.Equal(t, "42:42:42:42:42:42:42:42", lease.DUID)
			require.False(t, lease.FqdnFwd)
			require.True(t, lease.FqdnRev)
			require.Equal(t, "myhost.example.com.", lease.Hostname)
			require.Equal(t, "08:08:08:08:08:08", lease.HWAddress)
			require.EqualValues(t, 1, lease.IAID)
			require.Equal(t, "2001:db8:2::1", lease.IPAddress)
			require.EqualValues(t, 500, lease.PreferredLifetime)
			require.EqualValues(t, 0, lease.State)
			require.EqualValues(t, 44, lease.SubnetID)
			require.Equal(t, "IA_NA", lease.Type)
			require.EqualValues(t, 3600, lease.ValidLifetime)

			lease = leases[1]
			require.EqualValues(t, 12345678, lease.Cltt)
			require.Equal(t, "42:42:42:42:42:42:42:42", lease.DUID)
			require.False(t, lease.FqdnFwd)
			require.True(t, lease.FqdnRev)
			require.Empty(t, lease.Hostname)
			require.Empty(t, lease.HWAddress)
			require.EqualValues(t, 1, lease.IAID)
			require.Equal(t, "2001:db8:0:0:2::", lease.IPAddress)
			require.EqualValues(t, 500, lease.PreferredLifetime)
			require.EqualValues(t, 80, lease.PrefixLength)
			require.EqualValues(t, 0, lease.State)
			require.EqualValues(t, 44, lease.SubnetID)
			require.Equal(t, "IA_PD", lease.Type)
			require.EqualValues(t, 3600, lease.ValidLifetime)
		})
	}
}

// Test the scenario in sending lease4-get-by-hw-address command to Kea when
// no lease is found.
func TestGetLeases4Empty(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	leases, err := GetLeases4ByHWAddress(agents, app, "00:00:00:00:00:00")
	require.NoError(t, err)
	require.Empty(t, leases)
}

// Test the scenario in sending lease6-get-by-hostname command to Kea when
// no lease is found.
func TestGetLeases6Empty(t *testing.T) {
	agents := agentcommtest.NewFakeAgents(mockLeases6GetEmpty, nil)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		AccessPoints: accessPoints,
	}

	leases, err := GetLeases6ByHostname(agents, app, "myhost")
	require.NoError(t, err)
	require.Empty(t, leases)
}

// Test validation of the Kea servers' iunvalid responses or indicating errors.
func TestValidateGetLeasesResponse(t *testing.T) {
	validArgs := &struct{}{}
	invalidArgs := (*struct{})(nil)
	require.Error(t, validateGetLeasesResponse("command", keactrl.ResponseError, validArgs))
	require.Error(t, validateGetLeasesResponse("command", keactrl.ResponseCommandUnsupported, validArgs))
	require.Error(t, validateGetLeasesResponse("command", keactrl.ResponseSuccess, invalidArgs))
}

func TestFindLeases(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := &dbmodel.App{
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	agents := agentcommtest.NewFakeAgents(mockLeases4GetEmpty, nil)

	leases4, leases6, err := FindLeases(agents, "01:02:03:04:05:06")
	require.NoError(t, err)
	require.Len(t, leases4, 0)
	require.Len(t, leases6, 0)
}
