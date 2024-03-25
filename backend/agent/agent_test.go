package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/security/advancedtls"
	"gopkg.in/h2non/gock.v1"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
	"isc.org/stork/testutil"
)

type FakeAppMonitor struct {
	Apps []App
}

// Initializes StorkAgent instance and context used by the tests.
// Returns the teardown function that must be called to clean up the
// mocked paths.
func setupAgentTest() (*StorkAgent, context.Context, func()) {
	return setupAgentTestWithHooks(nil)
}

// Initializes StorkAgent instance and context used by the tests. Loads the
// given list of callout carriers (hooks' contents).
func setupAgentTestWithHooks(calloutCarriers []hooks.CalloutCarrier) (*StorkAgent, context.Context, func()) {
	httpClient := NewHTTPClient()
	httpClient.SetSkipTLSVerification(true)
	gock.InterceptClient(httpClient.client)

	restorePaths := RememberPaths()
	sb := testutil.NewSandbox()

	// Replace the default paths.
	RootCAFile = path.Join(sb.BasePath, "ca-not-exists.pem")
	CertPEMFile = path.Join(sb.BasePath, "cert-not-exists.pem")
	KeyPEMFile = path.Join(sb.BasePath, "key-not-exists.pem")
	AgentTokenFile = path.Join(sb.BasePath, "agent-token-not-exists")
	CredentialsFile = path.Join(sb.BasePath, "credentials-not-exists.json")
	ServerCertFingerprintFile = path.Join(sb.BasePath, "server-cert-not-exists.sha256")

	fam := FakeAppMonitor{}
	sa := &StorkAgent{
		AppMonitor:        &fam,
		GeneralHTTPClient: httpClient,
		KeaHTTPClient:     httpClient,
		logTailer:         newLogTailer(),
		keaInterceptor:    newKeaInterceptor(),
		hookManager:       NewHookManager(),
	}

	sa.hookManager.RegisterCalloutCarriers(calloutCarriers)
	sa.Setup()
	ctx := context.Background()
	return sa, ctx, func() {
		sb.Close()
		restorePaths()
	}
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
func makeAccessPoint(tp, address, key string, port int64, useSecureProtocol bool) (ap []AccessPoint) {
	return append(ap, AccessPoint{
		Type:              tp,
		Address:           address,
		Port:              port,
		Key:               key,
		UseSecureProtocol: useSecureProtocol,
	})
}

// Check if NewStorkAgent can be invoked and sets SA members.
func TestNewStorkAgent(t *testing.T) {
	fam := &FakeAppMonitor{}
	generalHTTPClient := NewHTTPClient()
	keaHTTPClient := NewHTTPClient()
	sa := NewStorkAgent("foo", 42, fam, generalHTTPClient, keaHTTPClient, NewHookManager())
	require.NotNil(t, sa.AppMonitor)
	require.Equal(t, generalHTTPClient, sa.GeneralHTTPClient)
	require.Equal(t, keaHTTPClient, sa.KeaHTTPClient)
}

// Check if an agent returns a response to a ping message..
func TestPing(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

	args := &agentapi.PingReq{}
	rsp, err := sa.Ping(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, rsp)
}

// Check if GetState works.
func TestGetState(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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
			AccessPoints: makeAccessPoint(AccessPointControl, "1.2.3.1", "", 1234, false),
		},
		HTTPClient: nil,
	})

	accessPoints := makeAccessPoint(AccessPointControl, "2.3.4.4", "abcd", 2345, true)
	accessPoints = append(accessPoints, AccessPoint{
		Type:              AccessPointStatistics,
		Address:           "2.3.4.5",
		Port:              2346,
		Key:               "foo",
		UseSecureProtocol: false,
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
	require.False(t, rsp.AgentUsesHTTPCredentials)
	require.Len(t, rsp.Apps, 2)

	keaApp := rsp.Apps[0]
	require.Len(t, keaApp.AccessPoints, 1)
	point := keaApp.AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "1.2.3.1", point.Address)
	require.False(t, point.UseSecureProtocol)
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
	require.True(t, point.UseSecureProtocol)
	point = bind9App.AccessPoints[1]
	require.Equal(t, AccessPointStatistics, point.Type)
	require.Equal(t, "2.3.4.5", point.Address)
	require.EqualValues(t, 2346, point.Port)
	require.False(t, point.UseSecureProtocol)
	require.EqualValues(t, "foo", point.Key)

	// Recreate Stork agent.
	sa, ctx, teardown = setupAgentTest()
	defer teardown()

	// Prepare credentials file.
	restorePaths := RememberPaths()
	defer restorePaths()
	sb := testutil.NewSandbox()
	defer sb.Close()

	content := `{
		"basic_auth": [
			{
				"ip": "10.0.0.1",
				"port": 42,
				"user": "foo",
				"password": "bar"
			}
		]
	}`

	CredentialsFile, _ = sb.Write("credentials.json", content)

	ok, err := sa.KeaHTTPClient.LoadCredentials()
	require.NoError(t, err)
	require.True(t, ok)

	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.True(t, rsp.AgentUsesHTTPCredentials)
}

// Test forwarding command to Kea when HTTP 200 status code
// is returned.
func TestForwardToKeaOverHTTPSuccess(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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
	require.JSONEq(t, "[{\"result\":0}]", string(rsp.KeaResponses[0].Response))
}

// Test forwarding command to Kea when HTTP 400 (Bad Request) status
// code is returned.
func TestForwardToKeaOverHTTPBadRequest(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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
	require.JSONEq(t, "[{\"HttpCode\":\"Bad Request\"}]", string(rsp.KeaResponses[0].Response))
}

// Test forwarding command to Kea when no body is returned.
func TestForwardToKeaOverHTTPEmptyBody(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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
	require.Len(t, string(rsp.KeaResponses[0].Response), 0)
}

// Test forwarding command when Kea is unavailable.
func TestForwardToKeaOverHTTPNoKea(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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
	sa, ctx, teardown := setupAgentTest()
	defer teardown()
	executor := newTestCommandExecutorDefault()
	rndcClient := NewRndcClient(executor)
	rndcClient.BaseCommand = []string{"/rndc"}

	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "_", 1234, false)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient: rndcClient,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		Address:     "127.0.0.1",
		Port:        1234,
		RndcRequest: cmd,
	}

	// Expect no error, an OK status code, and an empty status message.
	rsp, err := sa.ForwardRndcCommand(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.Equal(t, agentapi.Status_OK, rsp.Status.Code)
	require.Empty(t, rsp.Status.Message)
	// Check expected output.
	require.Equal(t, rsp.RndcResponse.Response, "Server is up and running")

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
	sa, ctx, teardown := setupAgentTest()
	defer teardown()
	executor := newTestCommandExecutorDefault().
		setRndcStatus("", errors.Errorf("mocking an error"))
	rndcClient := NewRndcClient(executor)
	rndcClient.BaseCommand = []string{"/rndc"}

	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "_", 1234, false)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient: rndcClient,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		Address:     "127.0.0.1",
		Port:        1234,
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
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		Address:     "127.0.0.1",
		Port:        1234,
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
	sa, ctx, teardown := setupAgentTest()
	defer teardown()
	executor := newTestCommandExecutorDefault().setRndcStatus("", nil)
	rndcClient := NewRndcClient(executor)
	rndcClient.BaseCommand = []string{"/rndc"}

	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "_", 1234, false)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient: rndcClient,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	cmd := &agentapi.RndcRequest{Request: "status"}

	req := &agentapi.ForwardRndcCommandReq{
		Address:     "127.0.0.1",
		Port:        1234,
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
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

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

// Checks if getRootCertificates:
// - returns an error if the cert file doesn't exist.
func TestGetRootCertificatesForMissingOrInvalidFiles(t *testing.T) {
	params := &advancedtls.GetRootCAsParams{}

	// Prepare temp dir for cert files.
	tmpDir, err := os.MkdirTemp("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0o755)
	restoreCerts := RememberPaths()
	defer restoreCerts()
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")

	// Missing cert file error.
	certStore := NewCertStoreDefault()
	getRootCertificates := createGetRootCertificatesHandler(certStore)
	_, err = getRootCertificates(params)
	require.ErrorContains(t, err, "could not read the root CA")
	require.ErrorContains(t, err, fmt.Sprintf("open %s/certs/ca.pem: no such file or directory", tmpDir))

	// store bad cert
	err = os.WriteFile(RootCAFile, []byte("CACertPEM"), 0o600)
	require.NoError(t, err)
	_, err = getRootCertificates(params)
	require.EqualError(t, err, "failed to append root CA")
}

// Checks if getRootCertificates reads and returns correct certificate successfully.
func TestGetRootCertificates(t *testing.T) {
	cleanup, err := GenerateSelfSignedCerts()
	require.NoError(t, err)
	defer cleanup()

	// All should be ok.
	certStore := NewCertStoreDefault()
	getRootCertificates := createGetRootCertificatesHandler(certStore)
	params := &advancedtls.GetRootCAsParams{}
	result, err := getRootCertificates(params)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TrustCerts)
}

// Checks if getIdentityCertificatesForServer:
// - returns an error if the key file doesn't exist,
// - returns an error if the key or cert contents are invalid.
func TestGetIdentityCertificatesForServerForMissingOrInvalid(t *testing.T) {
	info := &tls.ClientHelloInfo{}

	// Prepare temp dir for cert files.
	tmpDir, err := os.MkdirTemp("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0o755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0o755)
	restoreCerts := RememberPaths()
	defer restoreCerts()
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")

	// Missing key files.
	certStore := NewCertStoreDefault()
	getIdentityCertificatesForServer := createGetIdentityCertificatesForServerHandler(certStore)
	_, err = getIdentityCertificatesForServer(info)
	require.ErrorContains(t, err, "could not read the private key")
	require.ErrorContains(t, err, fmt.Sprintf("open %s/certs/key.pem: no such file or directory", tmpDir))

	// Store bad content to files.
	err = os.WriteFile(KeyPEMFile, []byte("KeyPEMFile"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(CertPEMFile, []byte("CertPEMFile"), 0o600)
	require.NoError(t, err)
	_, err = getIdentityCertificatesForServer(info)
	require.EqualError(t, err, "could not setup TLS key pair: tls: failed to find any PEM data in certificate input")
}

// Checks if getIdentityCertificatesForServer reads and returns
// correct key and certificate pair successfully.
func TestGetIdentityCertificatesForServer(t *testing.T) {
	cleanup, err := GenerateSelfSignedCerts()
	require.NoError(t, err)
	defer cleanup()

	// Now it should work.
	certStore := NewCertStoreDefault()
	getIdentityCertificatesForServer := createGetIdentityCertificatesForServerHandler(certStore)
	info := &tls.ClientHelloInfo{}
	certs, err := getIdentityCertificatesForServer(info)
	require.NoError(t, err)
	require.NotEmpty(t, certs)
}

// Check if newGRPCServerWithTLS can create gRPC server.
func TestNewGRPCServerWithTLS(t *testing.T) {
	cleanup, err := GenerateSelfSignedCerts()
	require.NoError(t, err)
	defer cleanup()

	srv, err := newGRPCServerWithTLS()
	require.NoError(t, err)
	require.NotNil(t, srv)
}

// Check if the Stork Agent prints the host and port parameters.
func TestHostAndPortParams(t *testing.T) {
	// Arrange
	sa, _, teardown := setupAgentTest()
	defer teardown()
	sa.Host = "127.0.0.1"
	sa.Port = 9876

	// We shut down the server before starting. It causes the serve
	// call fails and doesn't block the execution.
	sa.Shutdown(false)

	// Act
	var serveErr error
	stdout, _, err := testutil.CaptureOutput(func() {
		serveErr = sa.Serve()
	})

	// Assert
	require.Error(t, serveErr)
	require.NoError(t, err)
	stdoutStr := string(stdout)
	require.Contains(t, stdoutStr, "127.0.0.1")
	require.Contains(t, stdoutStr, "9876")
}
