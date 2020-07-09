package agentcomm

import (
	"testing"

	"github.com/stretchr/testify/require"

	storktest "isc.org/stork/server/test"
)

// Test that it is possible to connect to a new agent and that the
// statistics can be gathered for this agent.
func TestConnectingToAgent(t *testing.T) {
	settings := AgentsSettings{}
	fec := &storktest.FakeEventCenter{}
	agents := NewConnectedAgents(&settings, fec)
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

	// Let's modify some stats.
	agent.Stats.CurrentErrors++

	// We should be able to get pointer to stats via the convenience
	// function.
	stats := agents.GetConnectedAgentStats("127.0.0.1", 8080)
	require.NotNil(t, stats)
	require.EqualValues(t, 1, agent.Stats.CurrentErrors)
}
