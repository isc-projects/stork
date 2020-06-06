package agentcomm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectingToAgent(t *testing.T) {
	settings := AgentsSettings{}
	agents := NewConnectedAgents(&settings)
	defer agents.Shutdown()

	// connect one agent and check if it is in agents map
	agent, err := agents.GetConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	_, ok := agents.(*connectedAgentsData).AgentsMap["127.0.0.1:8080"]
	require.True(t, ok)

	// Initially, there should be no stats.
	require.NotNil(t, agent)
	require.Zero(t, agent.Stats.CurrentErrors)
	require.Empty(t, agent.Stats.AppCommStats)
}
