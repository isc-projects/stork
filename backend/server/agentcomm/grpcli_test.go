package agentcomm

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"isc.org/stork/api"
)

//go:generate mockgen -package=agentcomm -destination=api_mock.go isc.org/stork/api AgentClient

func TestGetState(t *testing.T) {
	settings := AgentsSettings{}
	agents := NewConnectedAgents(&settings)

	// pre-add an agent
	addr := "127.0.0.1:8080"
	agent, err := agents.GetConnectedAgent(addr)
	require.NoError(t, err)

	// create mock AgentClient and patch agent to point to it
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockAgentClient := NewMockAgentClient(ctrl)
	agent.Client = mockAgentClient

	// Call GetState
	expVer := "123"
	rsp := agentapi.GetStateRsp{
		AgentVersion: expVer,
		Apps: []*agentapi.App{
			{
				Version: "1.2.3",
				App: &agentapi.App_Kea{
					Kea: &agentapi.AppKea{
					},
				},
			},
		},
	}
	mockAgentClient.EXPECT().GetState(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	// Check response
	ctx := context.Background()
	state, err := agents.GetState(ctx, "127.0.0.1", 8080)
	require.NoError(t, err)
	require.Equal(t, state.AgentVersion, expVer)
}
