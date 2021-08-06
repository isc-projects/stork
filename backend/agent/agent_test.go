package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc/security/advancedtls"
	"gopkg.in/h2non/gock.v1"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
)

type FakeAppMonitor struct {
	Apps []App
}

// mockRndc mocks successful rndc output.
func mockRndc(command []string) ([]byte, error) {
	var output string

	if len(command) > 0 && command[len(command)-1] == "status" {
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
func setupAgentTest() (*StorkAgent, context.Context) {
	httpClient := NewHTTPClient()
	gock.InterceptClient(httpClient.client)

	fam := FakeAppMonitor{}
	sa := &StorkAgent{
		AppMonitor:     &fam,
		HTTPClient:     httpClient,
		logTailer:      newLogTailer(),
		keaInterceptor: newKeaInterceptor(),
	}
	sa.Setup()
	ctx := context.Background()
	return sa, ctx
}

func (fam *FakeAppMonitor) GetApps() []App {
	return fam.Apps
}

// Stub function for AppMonitor. It behaves in the same way as original one.
func (fam *FakeAppMonitor) GetApp(appType, apType, address string, port int64) App {
	for _, app := range fam.Apps {
		if app.GetBaseApp().Type != appType {
			continue
		}
		for _, ap := range app.GetBaseApp().AccessPoints {
			if ap.Type == apType && ap.Address == address && ap.Port == port {
				return app
			}
		}
	}
	return nil
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

// Check if NewStorkAgent can be invoked and sets SA members.
func TestNewStorkAgent(t *testing.T) {
	fam := &FakeAppMonitor{}
	var settings cli.Context
	sa := NewStorkAgent(&settings, fam)
	require.NotNil(t, sa.AppMonitor)
	require.NotNil(t, sa.HTTPClient)
}

// Check if an agent returns a response to a ping message..
func TestPing(t *testing.T) {
	sa, ctx := setupAgentTest()
	args := &agentapi.PingReq{}
	rsp, err := sa.Ping(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, rsp)
}

// Check if GetState works.
func TestGetState(t *testing.T) {
	sa, ctx := setupAgentTest()

	// app monitor is empty, no apps should be returned by GetState
	rsp, err := sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
	require.Empty(t, rsp.Apps)

	// add some apps to app monitor so GetState should return something
	var apps []App
	apps = append(apps, &KeaApp{
		BaseApp: BaseApp{
			Type:         AppTypeKea,
			AccessPoints: makeAccessPoint(AccessPointControl, "1.2.3.1", "", 1234),
		},
		HTTPClient: nil,
	})

	accessPoints := makeAccessPoint(AccessPointControl, "2.3.4.4", "abcd", 2345)
	accessPoints = append(accessPoints, AccessPoint{
		Type:    AccessPointStatistics,
		Address: "2.3.4.5",
		Port:    2346,
		Key:     "",
	})

	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient: nil,
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
	sa, ctx := setupAgentTest()

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
	sa, ctx := setupAgentTest()

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
	sa, ctx := setupAgentTest()

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
	sa, ctx := setupAgentTest()

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
	sa, ctx := setupAgentTest()

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

// Test forwarding command to named statistics-channel when no body is returned.
func TestForwardToNamedStatsHTTPEmptyBody(t *testing.T) {
	sa, ctx := setupAgentTest()

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
	sa, ctx := setupAgentTest()

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
	sa, ctx := setupAgentTest()

	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "_", 1234)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient: NewRndcClient(mockRndc),
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

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
	sa, ctx := setupAgentTest()

	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "_", 1234)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient: NewRndcClient(mockRndcError),
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

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

// Test rndc command when there is no app.
func TestForwardRndcCommandNoApp(t *testing.T) {
	sa, ctx := setupAgentTest()

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
	require.EqualValues(t, "Cannot find BIND 9 app", rsp.Status.Message)
}

// Test rndc command successfully forwarded, but bad response.
func TestForwardRndcCommandEmpty(t *testing.T) {
	sa, ctx := setupAgentTest()

	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "_", 1234)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient: NewRndcClient(mockRndcEmpty),
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

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
	sa, ctx := setupAgentTest()

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
		"--host", "--port", "--prometheus-kea-exporter-address", "--prometheus-kea-exporter-port",
		"--prometheus-kea-exporter-interval", "--prometheus-bind9-exporter-address",
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

// Check if stork-agent uses --host parameter.
func TestHostParam(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run agent with '--host 127.1.2.3'
	agentCmd := exec.CommandContext(ctx, AgentBin, "--host", "127.1.2.3")
	out, _ := agentCmd.Output()

	// Check if in the output there is 127.1.2.3.
	require.Contains(t, string(out), "127.1.2.3")
}

// Check if stork-agent uses --port parameter.
func TestPortParam(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run agent with '--port 9876'
	agentCmd := exec.CommandContext(ctx, AgentBin, "--port", "9876")
	out, _ := agentCmd.Output()

	// Check if in the output there is 9876.
	require.Contains(t, string(out), "9876")
}

// Checks if getRootCertificates:
// - returns an error if the cert file doesn't exist,
// - returns an error if the cert contents are invalid,
// - reads and returns correct certificate successfully.
func TestGetRootCertificates(t *testing.T) {
	params := &advancedtls.GetRootCAsParams{}

	// missing cert file error
	_, err := getRootCertificates(params)
	require.EqualError(t, err, "could not read CA certificate: /var/lib/stork-agent/certs/ca.pem: open /var/lib/stork-agent/certs/ca.pem: no such file or directory")

	// prepare temp dir for cert files
	tmpDir, err := ioutil.TempDir("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0755)
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")

	// store bad cert
	err = ioutil.WriteFile(RootCAFile, []byte("CACertPEM"), 0600)
	require.NoError(t, err)
	_, err = getRootCertificates(params)
	require.EqualError(t, err, "failed to append client certs")

	// store correct cert
	var CACertPEM []byte = []byte(`-----BEGIN CERTIFICATE-----
MIIFFjCCAv6gAwIBAgIBATANBgkqhkiG9w0BAQsFADAzMQswCQYDVQQGEwJVUzES
MBAGA1UEChMJSVNDIFN0b3JrMRAwDgYDVQQDEwdSb290IENBMB4XDTIwMTIwODA4
MDc1M1oXDTMwMTIwODA4MDgwM1owMzELMAkGA1UEBhMCVVMxEjAQBgNVBAoTCUlT
QyBTdG9yazEQMA4GA1UEAxMHUm9vdCBDQTCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBALgcYkndHQGFmLk8yi8+yetaCBI1cLG/nm+hwjh5C2rh3lqqDziG
qRmcITxkEbCFujbxJzlaXop1MeXwg2YJMky3WM1GWomVKv3jOVR+GkQG70pp0qpt
JmU2CuXoNhwMFA0H22CG8pPRiilUGPI7RLXaLWpA8D+AslfPHR9TG00HbJ86Bi3g
m4/uPiGdcHS6Q+wmKQRsKs6wAKSmlCrvmaKfmVOkxpuKyuKgjmIKoCwY3gYL1T8L
idvVePvbP/Z2SRQOVbSV8eMaYuk+uFwGKq8thLHs8bIEKhrIGlzDss6ZlPotTi2V
I6e6lb06oFuCSfhBaiHPw2sldwYvE/I8MkWUAuWtBgNvVE/e64FgJb1lGIzJpYMj
5jUp9Z13INsXy9zA8nKyZAK4fI6vlQGRg3bERn+S4Q6HXQor9Ma8QWxsqbdiC9dt
pxpzyx11tWg0jEgzCEBfk9IZjlGqyCdX5Z9pshHkQZ9VeK+DG0s6tYEm7BO1ssQD
+qbJS2PJq4Cwe82a6gO+lDz8A+xiXk8dJeTb8hf/c1NY192rqSLewI8oaHOLKEQg
XNSPEEkQqtIqn92Y5oKhLYKmYkwfOgldpj0XQQ3YwUnsOCfy2wRVNRg6VYnbjca8
rSy58t2MfovKWz9UcKhpnXefSdMgR7VhGv0ekDddGIfONn153uyjN/LpAgMBAAGj
NTAzMBIGA1UdEwEB/wQIMAYBAf8CAQEwHQYDVR0OBBYEFILkrDPZAlboeF+nav7C
Rf7nN1W+MA0GCSqGSIb3DQEBCwUAA4ICAQCDfvIgo70Y0Mi+Rs0mF6114z2gGQ7a
7/VnxV9w9uIjuaARq42E2DemFs5e72tPIfT9UWncgs5ZfyO5w2tjRpUOaVCSS5VY
93qzXBfTsqgkrkVRwec4qqZxpNqpsL9u2ZIfsSJ3BJWFV3Zq/3cOrDulfR5bk0G4
hYo/GDyLHjNalBFpetJSIk7l0VOkr2CBUvxKBOP0U1IQGXd+NL/8zW6UB6OitqNL
/tO+JztOpjo6ZYKJGZvxyL/3FUsiHmd8UwqAjnFjQRd3w0gseyqWDgILXQaDXQ5D
vs2oK+HheJv4h6CdrcIdWlWRKoZP3odZyWB0l31kpMbgYC/tMPYebG6mjPx+/S4m
7L+K27zmm2wItUaWI12ky2FPgeW78ALoKDYWmQ+CnpBNE1iFUf4qRzmypu77DmmM
bLgLFj8Bb50j0/zciPO7+D1h6hCPxwXdfQk0tnWBqjImmK3enbkEsw77kF8MkNjr
Hka0EeTt0hyEFKGgJ7jVdbjLFnRzre63q1GuQbLkOibyjf9WS/1ljv1Ps82aWeE+
rh78iXtpm8c/2IqrI37sLbAIs08iPj8ULV57RbcZI7iTYFIjKwPlWL8O2U1mopYP
RXkm1+W4cMzZS14MLfmacBHnI7Z4mRKvc+zEdco/l4omlszafmUXxnCOmqZlhqbm
/p0vFt1oteWWSQ==
-----END CERTIFICATE-----`)
	err = ioutil.WriteFile(RootCAFile, CACertPEM, 0600)
	require.NoError(t, err)

	// all should be ok
	result, err := getRootCertificates(params)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TrustCerts)
}

// Checks if getIdentityCertificatesForServer:
// - returns an error if the key file doesn't exist,
// - returns an error if the key or cert contents are invalid,
// - reads and returns correct key and certificate pair successfully.
func TestGetIdentityCertificatesForServer(t *testing.T) {
	info := &tls.ClientHelloInfo{}

	// missing key files
	_, err := getIdentityCertificatesForServer(info)
	require.EqualError(t, err, "could not load key PEM file: /var/lib/stork-agent/certs/key.pem: open /var/lib/stork-agent/certs/key.pem: no such file or directory")

	// prepare temp dir for cert files
	tmpDir, err := ioutil.TempDir("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0755)
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")

	// store bad content to files
	err = ioutil.WriteFile(KeyPEMFile, []byte("KeyPEMFile"), 0600)
	require.NoError(t, err)
	err = ioutil.WriteFile(CertPEMFile, []byte("CertPEMFile"), 0600)
	require.NoError(t, err)
	_, err = getIdentityCertificatesForServer(info)
	require.EqualError(t, err, "could not setup TLS key pair: tls: failed to find any PEM data in certificate input")

	// store proper content
	var certPEM []byte = []byte(`-----BEGIN CERTIFICATE-----
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
wQ==
-----END CERTIFICATE-----`)
	var keyPEM []byte = []byte(`-----BEGIN PRIVATE KEY-----
MIIJQgIBADANBgkqhkiG9w0BAQEFAASCCSwwggkoAgEAAoICAQDR8yndmAonFo0d
KWS3WQ3r60lIwKPOZwsdJy+2+eNrmZixYJ+CdlvH3/AVSBRJfYx14NFrHcRUsbW+
hn63kUwT3XHluLTs+QJWSaWa1zTLTJqiaEiPZI/xliQrTYoAV00jJip7CDWr0xpA
PpBwJmhJLrlwnxxZ6XlYLlGjyp+aImugYVQ+3xs4p18LcAmwf/+CyCPdp0rs6bUI
Emo99DwLvI1avWDkbzT3JAVgk3Kc4Jp3eZ/gGRWBBa0eSM5zr11G3xOouPFpe++e
pMsLdjrYgnt1PZBy8DPi5hL/7ltdfdWGvGkIeq1Y0n987P482nOizYoHhrSPKbz+
dL3e0ifIvxUUVrGmCnSefm4cwxW7GzDAZzUwZGa/qk24oEPeAi4zDrUeSdK6WTjF
ev+g6PTQDjifL1jYZHjxyn0+itDzcHqU9lIZUT5CzdenOhEEu3StUskoHOlq3tz2
bkG1hxnHX/CTbczbx1ave9XNSnZw3lCoAPGiL+Ra9Zaov+VVfhTTMv4uxGrjOV4d
DUTveu6mc+75E5mLBjmkGjtsD3H3e/xHTIdiOZd0emgr4PD8yXQqbDKybcvOOhLZ
uNFQPIE4gqzyGMb9BniOkECASLNBKgZcmtibkdwIghQATh+WbYhhOx+DyY+Dd/tG
LE1Q+Wf5sd2Oc/C59W6zDOKfmXmNXwIDAQABAoICAQDLXDWZJsPuyLE3Jfkgf2o0
slrx1WbVboodWu+k1LesacK1TVo0DGEqYYczlfXQmYOMSo+Oqe6Z+uiH+86SEHMY
as8ALMFTKH9TBVMbgIjqwvClj015V3b2EvBF4X1ihy14dmd/dJxIKtqqj+9oMkuh
V1jX9caIcNXQzEzX0lR2ABEv8BaiL4k2fyhY89Tu2YytKR9Ue87fXCC2COBP0lq3
I5Pn6LgJjI5JNOLggPHrcsMsJurtLl7d8pmVVABlnd9D3qA0Na/g9ONNT2I9X+/v
97OOBGv+aRxZE3Ij5MUq8c/6ClXSmMF/36UNZKF+YDrR3zVrxNbwNQWTk5C2W+mb
kAV15nAsF1RF+BX2KDLTeMnk72iiOho9BPXHbbiSJktHJHlNOd/cqic2R0P+1QAP
PMjKTQLYxci1BgofdBYB2lbdB/V+BtIsJ3TwaUXsQvgLwgU67LqameDZ+k8ROtUl
wZqKpsgnQpZ5eJtnXtpc4U2r9F+Kj718JpIYCZKY22rQgycNf8PKBD5jVY87QXhq
7qP071t2jXnIBE8Jb9/EeCa+pKTV6PlpVdX1DpISb69U640dUGPHjqu1xEOQIpiI
/+fyinbicLpwD8GYjnMjhV9/72Ka47fmvyOSPzo8hZUxII1X5iAvGYpp/Jpb6XBU
RKg8xW+fg43hVNiC65iGgQKCAQEA7Gwza7+bDtsJxsho2qT1cqjXL0UVaCq9ak+8
eHgYf3f7TeyG7OQS7BTWtgDcFfDp9Tdyry9A8ma7Cza3Xvw9u1beuJShUZBFLbZ+
vpbNRlcwP8A/BepQz0Qo3AVfjCIDugTHM0qy4aBqXdX+ynFD63C2bXTF6dJo75Jj
xYkyQOLb1rMVjd/G3P5sklG9RF+fR0UHi/vSYg7mWf0Dq3igzDDMi0BlMBE8+BqT
ciYef7Q3NQjYq1MqVwhRQcH9vA7tKgQzWp8pugEBftD8S1cnGR6psIwpyThEhyFv
GJo/n6Fo5QbalGq7ogJSKOlIJZx/izLkJl1VL8qfns1fg7v5CQKCAQEA41XHm8gw
T+Y/I//6stEf0P+MVKD1tiSfTiTX2LByLp1i43arvWhiid7LD7zEqLEY29XhudOO
szPR1haYuqIhjvhzbllV0NWQeeT9YyiSNQ63t/RvtjhWZ8Ffi2yV8s/iyT814bY0
2wKV+EPhPDsBipfhkNxDjDUxNWq1EvcdeN2FOa2HEgR5RpAd1T3IbDpi/wy3N/3W
rGy6NbbJcHygwsjGPBXPhjAZkqFu3GPys/MZZve+edhD7e68y1r90jIsytS9otsm
meBeFenR70+Tf6YWrJppsVXPb+uwlr2nVNDHZ1zcfYogjLs4tApogGFPzZKfy4Jn
X1kkvmiFE7n1JwKCAQBEXI0JzN+DDibnia93+VbXjqaaDnnAIwueH+w5UVCUGxdZ
Utk4ykIGbYggHGOHHKApvZy1tw4qiTXwaiPfnUQkVVwVNzTmJrc6HpjLd0Nn4XIc
HPScO0Kei/Dcndkg5fz53sPSuvi6cO4Qr/36f4HKJE87mxZXI/Yfv86FocQcKvyy
Ohozac9Qu2idbnExwgyGSRmDio8st243+wcCn+Cu6jVa1oXrvjBI9TZJPWh4OJ32
AdbUwzls7QTB5Nv/crl0+r32qCsik4PhLYCmME8n3kvmtsCmZFS8VhiPnppjCAMS
pkaxv6L9l3o2Ri4MYhInJ9H8neQx6374Jh5GMyYxAoIBACZB6E6iGOdJUzTmvjTb
lqQgbWhMki0t6pVHBAAWaZDIsbyf2vUMHREgqkGiveG5s/pC+zK/lJM51EVYFinK
YSVjUGGwrQ1w81hgHfhS+o/tQyO1AhvDTV82nrKi+nUbYQoHFjU+6ZQ10jEukzgE
ohTFzJMJTmDJDtfzdjeT2KTfeq0jM8jncdVbKXoaZKE6DjDn3emRUVBBF/E0KqBA
iPleumWgMgVeEN+pRTPXqh94eLzoUmjE6WGgPKtoS7DU+s7DkIpYoR1iMdM0Pz0r
wiHIPKadcc4DJ96o5lXn4sIWRIhzizOhTCsC0t8RpVZ9ieWJmFSyRF06bkGQ61xP
fh8CggEAEt7swXZznLParkDyWj0hjMNU5YLSPBikuZ+HtX2baVwlx7h7GnfwdmnD
TSzXQBRmQOgS/1Cntx5ol5ce/FUckP7ynqmTm3hDEgq3VT7Vc5KRUUyQLL93ft0T
K+pJ/hjBApOMnytqJttNz9qPs9jtaFkH0hnIPuwO3VIFi2qVhQM3KTGUl1MliXWL
iUmebg7yevOh8nkHR3B6GuCoiVQORYtVQo6p60i6oqXSz7tx/mlMqbV5o1hcd8iE
WESmigg1ZXkl20NEmfDBVZO2O41ODdM+raNVGgtESV4BStc8LO7K3Z4/OcoplV6I
H/Njg8CqtOWDeTVICuUq60wkbEkxYg==
-----END PRIVATE KEY-----`)
	err = ioutil.WriteFile(KeyPEMFile, keyPEM, 0600)
	require.NoError(t, err)
	err = ioutil.WriteFile(CertPEMFile, certPEM, 0600)
	require.NoError(t, err)

	// now it should work
	certs, err := getIdentityCertificatesForServer(info)
	require.NoError(t, err)
	require.NotEmpty(t, certs)
}

// Check if newGRPCServerWithTLS can create gRPC server.
func TestNewGRPCServerWithTLS(t *testing.T) {
	srv, err := newGRPCServerWithTLS()
	require.NoError(t, err)
	require.NotNil(t, srv)
}
