package agentcomm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"isc.org/stork/api"
)

//go:generate mockgen -package=agentcomm -destination=api_mock.go isc.org/stork/api AgentClient

func TestGetVersion(t *testing.T) {
	agents := NewConnectedAgents()

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
	mockAgentClient.EXPECT().GetState(gomock.Any(), gomock.Any()).
		Return(&agentapi.GetStateRsp{AgentVersion: expVer}, nil)

	// Check response
	state, err := agents.GetState(addr)
	require.NoError(t, err)
	require.Equal(t, state.AgentVersion, expVer)
}
