package kea

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	//log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/database/model"
)

// Helper struct to mock Agents behavior.
type FakeAgents struct {
	daemons         int
	recordedURL     string
	recordedCommand string
}

func (fa *FakeAgents) Shutdown() {}
func (fa *FakeAgents) GetConnectedAgent(address string) (*agentcomm.Agent, error) {
	return nil, nil
}
func (fa *FakeAgents) GetState(ctx context.Context, address string, agentPort int64) (*agentcomm.State, error) {
	state := agentcomm.State{
		Cpus:   1,
		Memory: 4,
	}
	return &state, nil
}
func (fa *FakeAgents) ForwardToKeaOverHttp(ctx context.Context, caURL string, command *agentcomm.KeaCommand,
	address string, agentPort int64) (agentcomm.KeaResponseList, error) {
	fa.recordedURL = caURL
	fa.recordedCommand = command.Command
	list := agentcomm.KeaResponseList{
		{
			Result: 0,
			Daemon: "dhcp4",
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
	if fa.daemons > 1 {
		list = append(list, agentcomm.KeaResponse{
			Result: 0,
			Daemon: "dhcp6",
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
	return list, nil
}

// Check if GetConfig returns response to the forwarded command.
func TestGetConfig(t *testing.T) {
	ctx := context.Background()
	fa := FakeAgents{}

	// check getting config of 1 daemon
	fa.daemons = 1
	dbApp := dbmodel.App{
		CtrlPort: 1234,
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

	list, err := GetConfig(ctx, &fa, &dbApp, &daemons)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list, 1)

	require.Equal(t, "http://localhost:1234/", fa.recordedURL)
	require.Equal(t, "config-get", fa.recordedCommand)

	// check getting configs of 2 daemons
	fa.daemons = 2
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

	list, err = GetConfig(ctx, &fa, &dbApp, &daemons)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list, 2)
}

// Check if GetDaemonHooks returns hooks for given daemon.
func TestGetDaemonHooks(t *testing.T) {
	ctx := context.Background()
	fa := FakeAgents{}

	// check getting config of 1 daemon
	fa.daemons = 1
	dbApp := dbmodel.App{
		Details: dbmodel.AppKea{
			Daemons: []dbmodel.KeaDaemon{
				{
					Name: "dhcp4",
				},
			},
		},
	}

	hooksMap, err := GetDaemonHooks(ctx, &fa, &dbApp)
	require.NoError(t, err)
	require.NotNil(t, hooksMap)
	hooks, ok := hooksMap["dhcp4"]
	require.True(t, ok)
	require.Len(t, hooks, 1)
	require.Equal(t, "hook_abc.so", hooks[0])

	// check getting configs of 2 daemons
	fa.daemons = 2
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

	hooksMap, err = GetDaemonHooks(ctx, &fa, &dbApp)
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
