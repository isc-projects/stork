package kea

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	//log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storktest "isc.org/stork/server/test"
)

// Kea servers' response to config-get command. The argument indicates if
// it is a response from a single server or two servers.
func mockGetConfigResponse(daemons int, response interface{}) {
	list := response.(*agentcomm.KeaResponseList)

	*list = agentcomm.KeaResponseList{
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
					},
				},
			},
		},
	}
	if daemons > 1 {
		*list = append(*list, agentcomm.KeaResponse{
			KeaResponseHeader: agentcomm.KeaResponseHeader{
				Result: 0,
				Daemon: "dhcp6",
			},
			Arguments: &map[string]interface{}{
				"Dhcp6": map[string]interface{}{
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
		})
	}
}

// Check if GetConfig returns response to the forwarded command.
func TestGetConfig(t *testing.T) {
	ctx := context.Background()

	// check getting config of 1 daemon
	fa := storktest.NewFakeAgents(func(response interface{}) {
		mockGetConfigResponse(1, response)
	})

	dbApp := dbmodel.App{
		CtrlAddress: "192.0.2.0",
		CtrlPort:    1234,
		Details: dbmodel.AppKea{
			Daemons: []dbmodel.KeaDaemon{
				{
					Name: "dhcp4",
				},
			},
		},
	}

	daemons := make(agentcomm.KeaDaemons)
	daemons["dhcp4"] = true

	list, err := GetConfig(ctx, fa, &dbApp, &daemons)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list, 1)

	require.Equal(t, "http://192.0.2.0:1234/", fa.RecordedURL)
	require.Equal(t, "config-get", fa.RecordedCommand)

	// check getting configs of 2 daemons
	fa = storktest.NewFakeAgents(func(response interface{}) {
		mockGetConfigResponse(2, response)
	})
	dbApp = dbmodel.App{
		Details: dbmodel.AppKea{
			Daemons: []dbmodel.KeaDaemon{
				{
					Name: "dhcp4",
				},
				{
					Name: "dhcp6",
				},
			},
		},
	}

	daemons["dhcp6"] = true

	list, err = GetConfig(ctx, fa, &dbApp, &daemons)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list, 2)
}

// Check if GetDaemonHooks returns hooks for given daemon.
func TestGetDaemonHooks(t *testing.T) {
	ctx := context.Background()
	// check getting config of 1 daemon
	fa := storktest.NewFakeAgents(func(response interface{}) {
		mockGetConfigResponse(1, response)
	})
	dbApp := dbmodel.App{
		Details: dbmodel.AppKea{
			Daemons: []dbmodel.KeaDaemon{
				{
					Name: "dhcp4",
				},
			},
		},
	}

	hooksMap, err := GetDaemonHooks(ctx, fa, &dbApp)
	require.NoError(t, err)
	require.NotNil(t, hooksMap)
	hooks, ok := hooksMap["dhcp4"]
	require.True(t, ok)
	require.Len(t, hooks, 1)
	require.Equal(t, "hook_abc.so", hooks[0])

	// check getting configs of 2 daemons
	fa = storktest.NewFakeAgents(func(response interface{}) {
		mockGetConfigResponse(2, response)
	})
	dbApp = dbmodel.App{
		Details: dbmodel.AppKea{
			Daemons: []dbmodel.KeaDaemon{
				{
					Name: "dhcp6",
				},
				{
					Name: "dhcp4",
				},
			},
		},
	}

	hooksMap, err = GetDaemonHooks(ctx, fa, &dbApp)
	require.NoError(t, err)
	require.NotNil(t, hooksMap)
	hooks, ok = hooksMap["dhcp4"]
	require.True(t, ok)
	require.Len(t, hooks, 1)
	require.Equal(t, "hook_abc.so", hooks[0])
	hooks, ok = hooksMap["dhcp6"]
	require.True(t, ok)
	require.Len(t, hooks, 2)
	require.Contains(t, hooks, "hook_abc.so")
	require.Contains(t, hooks, "hook_def.so")
}
