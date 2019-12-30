package kea

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/database/model"
)

// Helper struct to mock Agents behavior.
type FakeAgents struct {
}

func (fa *FakeAgents) Shutdown() {}
func (fa *FakeAgents) GetConnectedAgent(address string) (*agentcomm.Agent, error) {
	return nil, nil
}
func (fa *FakeAgents) GetState(ctx context.Context, address string, agentPort int64) (*agentcomm.State, error) {
	state := agentcomm.State{
		Cpus: 1,
		Memory: 4,
	}
	return &state, nil
}
func (fa *FakeAgents) ForwardToKeaOverHttp(ctx context.Context, caURL string, command *agentcomm.KeaCommand,
	address string, agentPort int64) (agentcomm.KeaResponseList, error) {
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
	return list, nil
}


// Check if GetConfig returns anything that make sense.
func TestGetConfig(t *testing.T) {
	ctx := context.Background()
	fa := FakeAgents{}
	dbApp := dbmodel.App{
		Details: dbmodel.AppKea{
			Daemons: []dbmodel.KeaDaemon{
				{
					Name: "dhcp4",
				},
			},
		},
	}

	list, err := GetConfig(ctx , &fa, &dbApp)
	require.NoError(t, err)
	require.NotNil(t, list)
}


// Check if GetDaemonHooks returns hooks for given daemon.
func TestGetDaemonHooks(t *testing.T) {
	ctx := context.Background()
	fa := FakeAgents{}
	dbApp := dbmodel.App{
		Details: dbmodel.AppKea{
			Daemons: []dbmodel.KeaDaemon{
				{
					Name: "dhcp4",
				},
			},
		},
	}

	hooksMap, err := GetDaemonHooks(ctx , &fa, &dbApp)
	require.NoError(t, err)
	require.NotNil(t, hooksMap)
	hooks, ok := hooksMap["dhcp4"]
	require.True(t, ok)
	require.Len(t, hooks, 1)
	require.Equal(t, "hook_abc.so", hooks[0])
}
