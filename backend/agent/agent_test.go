package agent

import (
	"context"
	"fmt"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
)

type FakeAppMonitor struct {
	Apps []*App
}

// mockRndc mocks successful rndc output.
func mockRndc(command []string) ([]byte, error) {
	var output string

	if command[len(command)-1] == "status" {
		output = "server is up and running"
		return []byte(output), nil
	}

	// unknown command.
	output = fmt.Sprintf("unknown command")
	return []byte(output), nil
}

// mockRndcError mocks an error.
func mockRndcError(command []string) ([]byte, error) {
	log.Debugf("mock rndc: error")

	return []byte(""), fmt.Errorf("mocking an error")
}

// mockRndcEmpty mocks empty output.
func mockRndcEmpty(command []string) ([]byte, error) {
	log.Debugf("mock rndc: empty")

	return []byte(""), nil
}

// Initializes StorkAgent instance and context used by the tests.
func setupAgentTest(rndc CommandExecutor) (*StorkAgent, context.Context) {
	caClient := NewCAClient()
	rndcClient := NewRndcClient(rndc)
	gock.InterceptClient(caClient.client)

	fsm := FakeAppMonitor{}
	sa := &StorkAgent{
		AppMonitor: &fsm,
		CAClient:   caClient,
		RndcClient: rndcClient,
	}
	ctx := context.Background()
	return sa, ctx
}

func (fsm *FakeAppMonitor) GetApps() []*App {
	return fsm.Apps
}

func (fsm *FakeAppMonitor) Shutdown() {
}

func TestNewStorkAgent(t *testing.T) {
	sa := NewStorkAgent()
	require.NotNil(t, sa.AppMonitor)
	require.NotNil(t, sa.CAClient)
	require.NotNil(t, sa.RndcClient)
}

func TestGetState(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

	// app monitor is empty, no apps should be returned by GetState
	rsp, err := sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
	require.Empty(t, rsp.Apps)

	// add some apps to app monitor so GetState should return something
	var apps []*App
	apps = append(apps, &App{
		Type:        "kea",
		CtrlAddress: "1.2.3.1",
		CtrlPort:    1234,
	})
	apps = append(apps, &App{
		Type:        "bind9",
		CtrlAddress: "2.3.4.4",
		CtrlPort:    2345,
		CtrlKey:     "abcd",
	})
	fsm, _ := sa.AppMonitor.(*FakeAppMonitor)
	fsm.Apps = apps
	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
	require.Equal(t, stork.Version, rsp.AgentVersion)
	require.Equal(t, 2, len(rsp.Apps))

	keaApp := rsp.Apps[0]
	bind9App := rsp.Apps[1]
	require.Equal(t, "1.2.3.1", keaApp.CtrlAddress)
	require.Equal(t, int64(1234), keaApp.CtrlPort)
	require.Empty(t, keaApp.CtrlKey)
	require.Equal(t, "2.3.4.4", bind9App.CtrlAddress)
	require.Equal(t, int64(2345), bind9App.CtrlPort)
	require.Equal(t, "abcd", bind9App.CtrlKey)
}

// Test forwarding command to Kea when HTTP 200 status code
// is returned.
func TestForwardToKeaOverHTTPSuccess(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

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
	req := &agentapi.ForwardToKeaOverHTTPReq{
		Url:         "http://localhost:45634/",
		KeaRequests: []*agentapi.KeaRequest{{Request: "{ \"command\": \"list-commands\"}"}},
	}

	// Kea should respond with non-empty body and the status code 200.
	// This should result in no error and the body should be available
	// in the response.
	rsp, err := sa.ForwardToKeaOverHTTP(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Len(t, rsp.KeaResponses, 1)
	require.JSONEq(t, "[{\"result\":0}]", rsp.KeaResponses[0].Response)
}

// Test forwarding command to Kea when HTTP 400 (Bad Request) status
// code is returned.
func TestForwardToKeaOverHTTPBadRequest(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(400).
		JSON([]map[string]string{{"HttpCode": "Bad Request"}})

	req := &agentapi.ForwardToKeaOverHTTPReq{
		Url:         "http://localhost:45634/",
		KeaRequests: []*agentapi.KeaRequest{{Request: "{ \"command\": \"list-commands\"}"}},
	}

	// The response to the forwarded command should contain HTTP
	// status code 400, but that should not raise an error in the
	// agent.
	rsp, err := sa.ForwardToKeaOverHTTP(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Len(t, rsp.KeaResponses, 1)
	require.JSONEq(t, "[{\"HttpCode\":\"Bad Request\"}]", rsp.KeaResponses[0].Response)
}

// Test forwarding command to Kea when no body is returned.
func TestForwardToKeaOverHTTPEmptyBody(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(200)

	req := &agentapi.ForwardToKeaOverHTTPReq{
		Url:         "http://localhost:45634/",
		KeaRequests: []*agentapi.KeaRequest{{Request: "{ \"command\": \"list-commands\"}"}},
	}

	// Forward the command to Kea. The response contains no body, but
	// this should not result in an error. The command sender should
	// deal with this as well as with other issues with the response
	// formatting.
	rsp, err := sa.ForwardToKeaOverHTTP(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Len(t, rsp.KeaResponses, 1)
	require.Equal(t, 0, len(rsp.KeaResponses[0].Response))
}

// Test forwarding command when Kea is unavailable.
func TestForwardToKeaOverHTTPNoKea(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

	req := &agentapi.ForwardToKeaOverHTTPReq{
		Url:         "http://localhost:45634/",
		KeaRequests: []*agentapi.KeaRequest{{Request: "{ \"command\": \"list-commands\"}"}},
	}

	// Kea is unreachable, so we'll have to signal an error to the sender.
	// The response should be empty.
	rsp, err := sa.ForwardToKeaOverHTTP(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Len(t, rsp.KeaResponses, 1)
	require.NotEqual(t, 0, rsp.KeaResponses[0].Status.Code)
	require.Equal(t, 0, len(rsp.KeaResponses[0].Response))
}

func TestForwardRndcCommandSuccess(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)
	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		CtrlAddress: "127.0.0.1",
		CtrlPort:    1234,
		CtrlKey:     "hmac-md5:abcd",
		RndcRequest: cmd,
	}

	// Expect no error, an OK status code, and an empty status message.
	rsp, err := sa.ForwardRndcCommand(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Equal(t, agentapi.Status_OK, rsp.Status.Code)
	require.Empty(t, rsp.Status.Message)
	// Check expected output.
	require.Equal(t, rsp.RndcResponse.Response, "server is up and running")

	// Empty request.
	cmd = &agentapi.RndcRequest{Request: ""}
	req.RndcRequest = cmd
	rsp, err = sa.ForwardRndcCommand(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Equal(t, agentapi.Status_OK, rsp.Status.Code)
	require.Empty(t, rsp.Status.Message)
	require.Equal(t, rsp.RndcResponse.Response, "unknown command")

	// Unknown request.
	cmd = &agentapi.RndcRequest{Request: "foobar"}
	req.RndcRequest = cmd
	rsp, err = sa.ForwardRndcCommand(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Equal(t, agentapi.Status_OK, rsp.Status.Code)
	require.Empty(t, rsp.Status.Message)
	require.Equal(t, rsp.RndcResponse.Response, "unknown command")
}

func TestForwardRndcCommandError(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndcError)
	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		CtrlAddress: "127.0.0.1",
		CtrlPort:    1234,
		CtrlKey:     "hmac-md5:abcd",
		RndcRequest: cmd,
	}

	// Expect an error status code and some message.
	rsp, err := sa.ForwardRndcCommand(ctx, req)
	require.NotNil(t, rsp)
	require.Error(t, err)
	require.Equal(t, agentapi.Status_ERROR, rsp.Status.Code)
	require.NotEmpty(t, rsp.Status.Message)
}

func TestForwardRndcCommandEmpty(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndcEmpty)
	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		CtrlAddress: "127.0.0.1",
		CtrlPort:    1234,
		CtrlKey:     "hmac-md5:abcd",
		RndcRequest: cmd,
	}

	// Empty output is not normal, but we are just forwarding, so expect
	// no error, an OK status code, and an empty status message.
	rsp, err := sa.ForwardRndcCommand(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Equal(t, agentapi.Status_OK, rsp.Status.Code)
	require.Empty(t, rsp.Status.Message)
}
