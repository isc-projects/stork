package agentcomm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"isc.org/stork/api"
)

//go:generate mockgen -package=agentcomm -destination=api_mock.go isc.org/stork/api AgentClient

func TestGetVersion(t *testing.T) {
	var agents ConnectedAgents
	agents.Init()

	// pre-add an agent
	addr := "127.0.0.1:8080"
	agent, err := agents.GetConnectedAgent(addr)
	require.NoError(t, err)

	// create mock AgentClient and patch agent to point to it
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockAgentClient := NewMockAgentClient(ctrl)
	agent.Client = mockAgentClient

	// Call GetVersion
	expVer := "123"
	mockAgentClient.EXPECT().GetVersion(gomock.Any(), gomock.Any()).
		Return(&agentapi.GetVersionRsp{Version: expVer}, nil)

	// Check response
	ver, err := agents.GetVersion(addr)
	require.NoError(t, err)
	require.Equal(t, ver, expVer)
}
