package agent

import (
	"testing"
	"context"

	"gopkg.in/h2non/gock.v1"
	"github.com/stretchr/testify/require"

	"isc.org/stork/api"
	"isc.org/stork"
)

type FakeAppMonitor struct {
	Apps []interface{}
}

// Initializes StorkAgent instance and context used by the tests.
func setupAgentTest() (*StorkAgent, context.Context) {
	fsm := FakeAppMonitor{}
	sa := &StorkAgent{
		AppMonitor: &fsm,
	}
	ctx := context.Background()
	return sa, ctx
}

func (fsm *FakeAppMonitor) GetApps() []interface{} {
	return nil
}

func (fsm *FakeAppMonitor) Shutdown() {
}


func TestGetState(t *testing.T) {
    sa, ctx := setupAgentTest()

	// app monitor is empty, no apps should be returned by GetState
	rsp, err := sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)

	// add some app to app monitor so GetState should return something
	var apps []interface{}
	apps = append(apps, AppKea{
		AppCommon: AppCommon{
			Version: "1.2.3",
			Active: true,
		},
	})
	fsm, _ := sa.AppMonitor.(*FakeAppMonitor)
	fsm.Apps = apps
	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
}

// Test forwarding command to Kea when HTTP 200 status code
// is returned.
func TestForwardToKeaOverHttpSuccess(t *testing.T) {
	sa, ctx := setupAgentTest()

	// Expect appropriate content type and the body. If they are not matched
	// an error will be raised.
	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]string{"command": "list-commands"}).
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": 0}})

	// Forward the request with the expected body.
	req := &agentapi.ForwardToKeaOverHttpReq{
		Url: "http://localhost:45634/",
		KeaRequest: "{ \"command\": \"list-commands\"}",
	}

	// Kea should respond with non-empty body and the status code 200.
	// This should result in no error and the body should be available
	// in the response.
	rsp, err := sa.ForwardToKeaOverHttp(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.JSONEq(t, "[{\"result\":0}]", rsp.KeaResponse)
}

// Test forwarding command to Kea when HTTP 400 (Bad Request) status
// code is returned.
func TestForwardToKeaOverHttpBadRequest(t *testing.T) {
	sa, ctx := setupAgentTest()

	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(400).
		JSON([]map[string]string{{"HttpCode": "Bad Request"}})

	req := &agentapi.ForwardToKeaOverHttpReq{
		Url: "http://localhost:45634/",
		KeaRequest: "{ \"command\": \"list-commands\"}",
	}

	// The response to the forwarded command should contain HTTP
	// status code 400, but that should not raise an error in the
	// agent.
	rsp, err := sa.ForwardToKeaOverHttp(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.JSONEq(t, "[{\"HttpCode\":\"Bad Request\"}]", rsp.KeaResponse)
}

// Test forwarding command to Kea when no body is returned.
func TestForwardToKeaOverHttpEmptyBody(t *testing.T) {
	sa, ctx := setupAgentTest()

	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(200)

	req := &agentapi.ForwardToKeaOverHttpReq{
		Url: "http://localhost:45634/",
		KeaRequest: "{ \"command\": \"list-commands\"}",
	}

	// Forward the command to Kea. The response contains no body, but
	// this should not result in an error. The command sender should
	// deal with this as well as with other issues with the response
	// formatting.
	rsp, err := sa.ForwardToKeaOverHttp(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Equal(t, 0, len(rsp.KeaResponse))
}

// Test forwarding command when Kea is unavailable.
func TestForwardToKeaOverHttpNoKea(t *testing.T) {
	sa, ctx := setupAgentTest()

	req := &agentapi.ForwardToKeaOverHttpReq{
		Url: "http://localhost:45634/",
		KeaRequest: "{ \"command\": \"list-commands\"}",
	}

	// Kea is unreachable, so we'll have to signal an error to the sender.
	// The response should be empty.
	rsp, err := sa.ForwardToKeaOverHttp(ctx, req)
	require.NotNil(t, rsp)
	require.Error(t, err)
	require.Equal(t, 0, len(rsp.KeaResponse))
}
