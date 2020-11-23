package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/pkg/errors"
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
	output = "unknown command"
	return []byte(output), nil
}

// mockRndcError mocks an error.
func mockRndcError(command []string) ([]byte, error) {
	log.Debugf("mock rndc: error")

	return []byte(""), errors.Errorf("mocking an error")
}

// mockRndcEmpty mocks empty output.
func mockRndcEmpty(command []string) ([]byte, error) {
	log.Debugf("mock rndc: empty")

	return []byte(""), nil
}

// Initializes StorkAgent instance and context used by the tests.
func setupAgentTest(rndc CommandExecutor) (*StorkAgent, context.Context) {
	httpClient := NewHTTPClient()
	rndcClient := NewRndcClient(rndc)
	gock.InterceptClient(httpClient.client)

	fam := FakeAppMonitor{}
	sa := &StorkAgent{
		AppMonitor:     &fam,
		HTTPClient:     httpClient,
		RndcClient:     rndcClient,
		logTailer:      newLogTailer(),
		keaInterceptor: newKeaInterceptor(),
	}
	ctx := context.Background()
	return sa, ctx
}

func (fam *FakeAppMonitor) GetApps() []*App {
	return fam.Apps
}

func (fam *FakeAppMonitor) Shutdown() {
}

func (fam *FakeAppMonitor) Start(storkAgent *StorkAgent) {
}

// makeAccessPoint is an utility to make single element app access point slice.
func makeAccessPoint(tp, address, key string, port int64) (ap []AccessPoint) {
	return append(ap, AccessPoint{
		Type:    tp,
		Address: address,
		Port:    port,
		Key:     key,
	})
}

func TestNewStorkAgent(t *testing.T) {
	fam := &FakeAppMonitor{}
	sa := NewStorkAgent(fam)
	require.NotNil(t, sa.AppMonitor)
	require.NotNil(t, sa.HTTPClient)
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
		Type:         AppTypeKea,
		AccessPoints: makeAccessPoint(AccessPointControl, "1.2.3.1", "", 1234),
	})

	accessPoints := makeAccessPoint(AccessPointControl, "2.3.4.4", "abcd", 2345)
	accessPoints = append(accessPoints, AccessPoint{
		Type:    AccessPointStatistics,
		Address: "2.3.4.5",
		Port:    2346,
		Key:     "",
	})

	apps = append(apps, &App{
		Type:         AppTypeBind9,
		AccessPoints: accessPoints,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps
	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
	require.Equal(t, stork.Version, rsp.AgentVersion)
	require.Len(t, rsp.Apps, 2)

	keaApp := rsp.Apps[0]
	require.Len(t, keaApp.AccessPoints, 1)
	point := keaApp.AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "1.2.3.1", point.Address)
	require.EqualValues(t, 1234, point.Port)
	require.Empty(t, point.Key)

	bind9App := rsp.Apps[1]
	require.Len(t, bind9App.AccessPoints, 2)
	// sorted by port
	point = bind9App.AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "2.3.4.4", point.Address)
	require.EqualValues(t, 2345, point.Port)
	require.Equal(t, "abcd", point.Key)
	point = bind9App.AccessPoints[1]
	require.Equal(t, AccessPointStatistics, point.Type)
	require.Equal(t, "2.3.4.5", point.Address)
	require.EqualValues(t, 2346, point.Port)
	require.Empty(t, point.Key)
}

// Helper function for unzipping buffers. It does not return
// any error, it expects that everything will go fine.
func doGunzip(data []byte) string {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		panic("problem with gunzip: NewReader")
	}
	unpackedResp, err := ioutil.ReadAll(zr)
	if err != nil {
		panic("problem with gunzip: ReadAll")
	}
	if err := zr.Close(); err != nil {
		panic("problem with gunzip: Close")
	}
	return string(unpackedResp)
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
	require.JSONEq(t, "[{\"result\":0}]", doGunzip(rsp.KeaResponses[0].Response))
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
	require.JSONEq(t, "[{\"HttpCode\":\"Bad Request\"}]", doGunzip(rsp.KeaResponses[0].Response))
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
	require.Len(t, doGunzip(rsp.KeaResponses[0].Response), 0)
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
	require.Len(t, rsp.KeaResponses[0].Response, 0)
}

// Test successful forwarding stats request to named.
func TestForwardToNamedStatsSuccess(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

	// Expect appropriate content type and the body. If they are not matched
	// an error will be raised.
	defer gock.Off()
	gock.New("http://localhost:45634/").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": 0}})

	// Forward the request with the expected body.
	req := &agentapi.ForwardToNamedStatsReq{
		Url:               "http://localhost:45634/json/v1",
		NamedStatsRequest: &agentapi.NamedStatsRequest{Request: ""},
	}

	// Named should respond with non-empty body and the status code 200.
	// This should result in no error and the body should be available
	// in the response.
	rsp, err := sa.ForwardToNamedStats(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.NotNil(t, rsp.NamedStatsResponse)
	require.JSONEq(t, "[{\"result\":0}]", rsp.NamedStatsResponse.Response)
}

// Test forwarding command to named when HTTP 400 (Bad Request) status
// code is returned.
func TestForwardToNamedStatsBadRequest(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

	defer gock.Off()
	gock.New("http://localhost:45634/json/v1").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(400).
		JSON([]map[string]string{{"HttpCode": "Bad Request"}})

	req := &agentapi.ForwardToNamedStatsReq{
		Url:               "http://localhost:45634/json/v1",
		NamedStatsRequest: &agentapi.NamedStatsRequest{Request: ""},
	}

	// The response to the forwarded command should contain HTTP
	// status code 400, but that should not raise an error in the
	// agent.
	rsp, err := sa.ForwardToNamedStats(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.NotNil(t, rsp.NamedStatsResponse)
	require.JSONEq(t, "[{\"HttpCode\":\"Bad Request\"}]", rsp.NamedStatsResponse.Response)
	require.NotEqual(t, 0, rsp.NamedStatsResponse.Status.Code)
}

// Test forwarding command to named statistics-cahnnel when no body is returned.
func TestForwardToNamedStatsHTTPEmptyBody(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

	defer gock.Off()
	gock.New("http://localhost:45634/json/v1").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(200)

	req := &agentapi.ForwardToNamedStatsReq{
		Url:               "http://localhost:45634/json/v1",
		NamedStatsRequest: &agentapi.NamedStatsRequest{Request: ""},
	}

	// Forward the command to named statistics-channel.
	// The response contains no body, but this should not result in an
	// error. The command sender should deal with this as well as with
	// other issues with the response formatting.
	rsp, err := sa.ForwardToNamedStats(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.NotNil(t, rsp.NamedStatsResponse)
	require.Len(t, rsp.NamedStatsResponse.Response, 0)
}

// Test forwarding statistics request when named is unavailable.
func TestForwardToNamedStatsNoNamed(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

	req := &agentapi.ForwardToNamedStatsReq{
		Url:               "http://localhost:45634/json/v1",
		NamedStatsRequest: &agentapi.NamedStatsRequest{Request: ""},
	}

	// Named is unreachable, so we'll have to signal an error to the sender.
	// The response should be empty.
	rsp, err := sa.ForwardToNamedStats(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.NotNil(t, rsp.NamedStatsResponse)
	require.NotEqual(t, 0, rsp.NamedStatsResponse.Status.Code)
	require.Len(t, rsp.NamedStatsResponse.Response, 0)
}

// Test a successful rndc command.
func TestForwardRndcCommandSuccess(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)
	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		Address:     "127.0.0.1",
		Port:        1234,
		Key:         "hmac-sha256:abcd",
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

// Test rndc command failed to forward.
func TestForwardRndcCommandError(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndcError)
	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		Address:     "127.0.0.1",
		Port:        1234,
		Key:         "hmac-sha256:abcd",
		RndcRequest: cmd,
	}

	// Expect an error status code and some message.
	rsp, err := sa.ForwardRndcCommand(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Equal(t, agentapi.Status_ERROR, rsp.Status.Code)
	require.NotEmpty(t, rsp.Status.Message)
}

// Test rndc command successfully forwarded, but bad response.
func TestForwardRndcCommandEmpty(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndcEmpty)
	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		Address:     "127.0.0.1",
		Port:        1234,
		Key:         "hmac-sha256:abcd",
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

// Test that the tail of the text file can be fetched.
func TestTailTextFile(t *testing.T) {
	sa, ctx := setupAgentTest(mockRndc)

	filename := fmt.Sprintf("test%d.log", rand.Int63())
	f, err := os.Create(filename)
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(filename)
	}()

	fmt.Fprintln(f, "This is a file")
	fmt.Fprintln(f, "which is used")
	fmt.Fprintln(f, "in testing TailTextFile")

	sa.logTailer.allow(filename)

	// Forward the request with the expected body.
	req := &agentapi.TailTextFileReq{
		Offset: 38,
		Path:   filename,
	}

	rsp, err := sa.TailTextFile(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Len(t, rsp.Lines, 2)
	require.Equal(t, "which is used", rsp.Lines[0])
	require.Equal(t, "in testing TailTextFile", rsp.Lines[1])

	// Test the case when the offset is beyond the file size.
	req = &agentapi.TailTextFileReq{
		Offset: 200,
		Path:   filename,
	}

	rsp, err = sa.TailTextFile(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Len(t, rsp.Lines, 3)
	require.Equal(t, "This is a file", rsp.Lines[0])
	require.Equal(t, "which is used", rsp.Lines[1])
	require.Equal(t, "in testing TailTextFile", rsp.Lines[2])
}

// Aux function checks if a list of expected strings is present in the string.
func checkOutput(output string, exp []string, reason string) bool {
	for _, x := range exp {
		fmt.Printf("Checking if %s exists in %s.\n", x, reason)
		if !strings.Contains(output, x) {
			fmt.Printf("ERROR: Expected string [%s] not found in %s.\n", x, reason)
			return false
		}
	}
	return true
}

// This is the list of all parameters we expect to be supported by stork-agent.
func getExpectedSwitches() []string {
	return []string{
		"-v", "--version", "--listen-prometheus-only", "--listen-stork-only",
		"--host", "--port", "--prometheus-kea-exporter-host", "--prometheus-kea-exporter-port",
		"--prometheus-kea-exporter-interval", "--prometheus-bind9-exporter-host",
		"--prometheus-bind9-exporter-port", "--prometheus-bind9-exporter-interval",
	}
}

// Location of the stork-agent binary.
const AgentBin = "../cmd/stork-agent/stork-agent"

// Location of the stork-agent man page.
const AgentMan = "../../doc/man/stork-agent.8.rst"

// This test checks if stork-agent -h reports all expected command-line switches.
func TestCommandLineSwitches(t *testing.T) {
	// Run the --help version and get its output.
	agentCmd := exec.Command(AgentBin, "-h")
	output, err := agentCmd.Output()
	require.NoError(t, err)

	// Now check that all expected command-line switches are really there.
	require.True(t, checkOutput(string(output), getExpectedSwitches(), "stork-agent -h output"))
}

// This test checks if all expected command-line switches are documented in the man page.
func TestCommandLineSwitchesDoc(t *testing.T) {
	// Read the contents of the man page
	file, err := os.Open(AgentMan)
	require.NoError(t, err)
	man, err := ioutil.ReadAll(file)
	require.NoError(t, err)

	// And check that all expected switches are mentioned there.
	require.True(t, checkOutput(string(man), getExpectedSwitches(), "stork-agent.8.rst"))
}

// This test checks if stork-agent --version (and -v) report expected version.
func TestCommandLineVersion(t *testing.T) {
	// Let's repeat the test twice (for -v and then for --version)
	for _, opt := range []string{"-v", "--version"} {
		fmt.Printf("Checking %s\n", opt)

		// Run the agent with specific switch.
		agentCmd := exec.Command(AgentBin, opt)
		output, err := agentCmd.Output()
		require.NoError(t, err)

		// Clean up the output (remove end of line)
		ver := strings.TrimSpace(string(output))

		// Check if it equals expected version.
		require.Equal(t, ver, stork.Version)
	}
}
