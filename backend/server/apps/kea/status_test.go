package kea

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storktest "isc.org/stork/server/test"
)

// Generate test response to status-get command including status of the
// HA pair doing load balancing.
func mockGetStatusLoadBalancing(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
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
            }
    ]`
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Generates test response to status-get command lacking a status of the
// HA pair.
func mockGetStatusNoHA(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
	command, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)
	json := `[
        {
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111
            }
        }
    ]`
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Generates test response to status-get command indicating an error and
// lacking argument.s
func mockGetStatusError(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
	command, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)
	json := `[
        {
            "result": 1,
            "text": "unable to communicate with the deamon"
        }
    ]`
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Test status-get command when HA status is returned.
func TestGetDHCPStatus(t *testing.T) {
	fa := storktest.NewFakeAgents(mockGetStatusLoadBalancing)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)

	app := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	appStatus, err := GetDHCPStatus(context.Background(), fa, &app)
	require.NoError(t, err)
	require.NotNil(t, appStatus)

	require.Len(t, appStatus, 1)

	status := appStatus[0]

	// Common fields must be always present.
	require.EqualValues(t, 1234, status.Pid)
	require.EqualValues(t, 3024, status.Uptime)
	require.EqualValues(t, 1111, status.Reload)
	require.Equal(t, "dhcp4", status.Daemon)

	// HA status should have been returned.
	require.NotNil(t, status.HAServers)

	// Test HA status of the server receiving the command.
	local := status.HAServers.Local
	require.Equal(t, "primary", local.Role)
	require.Len(t, local.Scopes, 1)
	require.Contains(t, local.Scopes, "server1")
	require.Equal(t, "load-balancing", local.State)

	// Test HA status of the partner.
	remote := status.HAServers.Remote
	require.Equal(t, "secondary", remote.Role)
	require.Len(t, remote.LastScopes, 1)
	require.Contains(t, remote.LastScopes, "server2")
	require.Equal(t, "load-balancing", remote.LastState)
	require.EqualValues(t, 10, remote.Age)
	require.True(t, remote.InTouch)
}

// Test status-get command when HA status is not returned.
func TestGetDHCPStatusNoHA(t *testing.T) {
	fa := storktest.NewFakeAgents(mockGetStatusNoHA)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)

	app := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	appStatus, err := GetDHCPStatus(context.Background(), fa, &app)
	require.NoError(t, err)
	require.NotNil(t, appStatus)

	require.Len(t, appStatus, 1)

	status := appStatus[0]

	// Common fields must be always present.
	require.EqualValues(t, 1234, status.Pid)
	require.EqualValues(t, 3024, status.Uptime)
	require.EqualValues(t, 1111, status.Reload)

	// This time, HA status should not be present.
	require.Nil(t, status.HAServers)
}

// Test the case when the Kea CA is unable to communicate with the
// Kea deamon.
func TestGetDHCPStatusError(t *testing.T) {
	fa := storktest.NewFakeAgents(mockGetStatusError)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)

	app := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	appStatus, err := GetDHCPStatus(context.Background(), fa, &app)
	require.NoError(t, err)
	require.NotNil(t, appStatus)

	require.Empty(t, appStatus)
}
