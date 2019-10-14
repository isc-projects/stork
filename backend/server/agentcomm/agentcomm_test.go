package agentcomm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectingToAgent(t *testing.T) {
	var agents ConnectedAgents
	agents.Init()

	// connect one agent and check if it is in agents map
	agents.GetConnectedAgent("127.0.0.1:8080")
	_, ok := agents.AgentsMap["127.0.0.1:8080"]
	require.True(t, ok)
}
