package agentcomm

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"isc.org/stork/api"
)

// Setup function for the unit tests. It creates a fake agent running at
// 127.0.0.1:8080. The returned function performs a test teardown and
// should be invoked when the unit test finishes.
func setupGrpcliTestCase(t *testing.T) (*MockAgentClient, *connectedAgentsData, func()) {
	settings := AgentsSettings{}
	agents := NewConnectedAgents(&settings)

	// pre-add an agent
	addr := "127.0.0.1:8080"
	agent, err := agents.GetConnectedAgent(addr)
	require.NoError(t, err)

	// create mock AgentClient and patch agent to point to it
	ctrl := gomock.NewController(t)
	mockAgentClient := NewMockAgentClient(ctrl)
	agent.Client = mockAgentClient

	return mockAgentClient, agents, func() {
		ctrl.Finish()
	}
}

//go:generate mockgen -package=agentcomm -destination=api_mock.go isc.org/stork/api AgentClient

func TestGetState(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

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

// Test that a command can be successfully forwarded to Kea and the response
// can be parsed.
func TestForwardToKeaOverHttp(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

	rsp := agentapi.ForwardToKeaOverHttpRsp{
		KeaResponse: `[
            {
                "result": 1,
                "text": "operation failed"
            },
            {
                "result": 0,
                "text": "operation succeeded",
                "arguments": {
                    "success": true
                }
            }
        ]`,
	}
	mockAgentClient.EXPECT().ForwardToKeaOverHttp(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	ctx := context.Background()
	command, _ := NewKeaCommand("test-command", nil, nil)
	actualResponse, err := agents.ForwardToKeaOverHttp(ctx, "http://localhost:8000/", command, "127.0.0.1", 8080)
	require.NoError(t, err)
	require.NotNil(t, actualResponse)

	responseList := actualResponse
	require.Equal(t, 2, len(responseList))

	require.Equal(t, 1, responseList[0].Result)
	require.Equal(t, "operation failed", responseList[0].Text)
	require.Nil(t, responseList[0].Arguments)

	require.Equal(t, 0, responseList[1].Result)
	require.Equal(t, "operation succeeded", responseList[1].Text)
	require.NotNil(t, responseList[1].Arguments)
	require.Equal(t, 1, len(*responseList[1].Arguments))
	require.Contains(t, *responseList[1].Arguments, "success")
}

// Test that the error is returned when the response to the forwarded Kea command
// is malformed.
func TestForwardToKeaOverHttpInvalidResponse(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

	rsp := agentapi.ForwardToKeaOverHttpRsp{
		KeaResponse: `[
            {
                "result": "a string"
            }
        ]`,
	}
	mockAgentClient.EXPECT().ForwardToKeaOverHttp(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	ctx := context.Background()
	command, _ := NewKeaCommand("test-command", nil, nil)
	actualResponse, err := agents.ForwardToKeaOverHttp(ctx, "http://localhost:8080/", command, "127.0.0.1", 8080)
	require.Error(t, err)
	require.Nil(t, actualResponse)
}
