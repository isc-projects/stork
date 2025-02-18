package agent

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"os"
	"path"
	"slices"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/security/advancedtls"
	"google.golang.org/grpc/status"
	"gopkg.in/h2non/gock.v1"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
	"isc.org/stork/appdata/bind9stats"
	"isc.org/stork/hooks"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=agent -destination=serverstreamingservermock_test.go google.golang.org/grpc ServerStreamingServer

type FakeAppMonitor struct {
	Apps       []App
	HTTPClient *httpClient
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
	bind9StatsClient := NewBind9StatsClient()
	gock.InterceptClient(bind9StatsClient.innerClient.GetClient())

	httpClientConfig := HTTPClientConfig{SkipTLSVerification: true}
	httpClient := NewHTTPClient(httpClientConfig)
	gock.InterceptClient(httpClient.client)

	cleanupCerts, _ := GenerateSelfSignedCerts()

	fam := FakeAppMonitor{
		Apps: []App{
			&KeaApp{
				BaseApp: BaseApp{
					Type:         AppTypeKea,
					AccessPoints: makeAccessPoint(AccessPointControl, "localhost", "", 45634, false),
					Pid:          42,
				},
				HTTPClient: httpClient,
			},
			&Bind9App{
				BaseApp: BaseApp{
					Type:         AppTypeBind9,
					AccessPoints: makeAccessPoint(AccessPointControl, "localhost", "abcd", 45635, false),
					Pid:          43,
				},
			},
		},
	}

	sa := &StorkAgent{
		AppMonitor:          &fam,
		bind9StatsClient:    bind9StatsClient,
		KeaHTTPClientConfig: httpClientConfig,
		logTailer:           newLogTailer(),
		keaInterceptor:      newKeaInterceptor(),
		hookManager:         NewHookManager(),
	}

	sa.hookManager.RegisterCalloutCarriers(calloutCarriers)
	err := sa.SetupGRPCServer()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	return sa, ctx, cleanupCerts
}

// Stub function for AppMonitor. It returns a fixed list of apps.
func (fam *FakeAppMonitor) GetApps() []App {
	return fam.Apps
}

// Stub function for AppMonitor. It behaves in the same way as original one.
func (fam *FakeAppMonitor) GetApp(appType, apType, address string, port int64) App {
	for _, app := range fam.GetApps() {
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
	bind9StatsClient := NewBind9StatsClient()
	keaHTTPClientConfig := HTTPClientConfig{}
	sa := NewStorkAgent(
		"foo", 42, fam, bind9StatsClient, keaHTTPClientConfig, NewHookManager(), "",
	)
	require.NotNil(t, sa.AppMonitor)
	require.Equal(t, bind9StatsClient, sa.bind9StatsClient)
	require.Equal(t, keaHTTPClientConfig, sa.KeaHTTPClientConfig)
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
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = nil

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
		HTTPClient: newHTTPClientWithDefaults(),
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

	app := fam.GetApp(AppTypeKea, AccessPointControl, "1.2.3.1", 1234).(*KeaApp)
	app.HTTPClient = NewHTTPClient(HTTPClientConfig{
		BasicAuth: basicAuthCredentials{User: "foo", Password: "bar"},
	})

	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	// Deprecated parameter. Always false.
	require.False(t, rsp.AgentUsesHTTPCredentials)
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
		MatchHeader("Accept", "application/json").
		Get("json/v1").
		Reply(200).
		JSON([]map[string]int{{"result": 0}})

	// Forward the request with the expected body.
	req := &agentapi.ForwardToNamedStatsReq{
		Url:               "http://localhost:45634/",
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
	gock.New("http://localhost:45634/").
		MatchHeader("Accept", "application/json").
		Get("/json/v1").
		Reply(400).
		JSON([]map[string]string{{"HttpCode": "Bad Request"}})

	req := &agentapi.ForwardToNamedStatsReq{
		Url:               "http://localhost:45634/",
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
	gock.New("http://localhost:45634/").
		MatchHeader("Accept", "application/json").
		Get("/json/v1").
		Reply(200)

	req := &agentapi.ForwardToNamedStatsReq{
		Url:               "http://localhost:45634/",
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
		Url:               "http://localhost:45634/",
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
	params := &advancedtls.ConnectionInfo{}

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
	params := &advancedtls.ConnectionInfo{}
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

// Test that the error is returned if the GRPC TLS certificate files are
// missing.
func TestNewGRPCServerWithTLSMissingCerts(t *testing.T) {
	// Arrange
	cleanup := RememberPaths()
	defer cleanup()
	sb := testutil.NewSandbox()
	defer sb.Close()

	KeyPEMFile = path.Join(sb.BasePath, "key-not-exists.pem")
	CertPEMFile = path.Join(sb.BasePath, "cert-not-exists.pem")
	RootCAFile = path.Join(sb.BasePath, "rootCA-not-exists.pem")
	AgentTokenFile = path.Join(sb.BasePath, "agentToken-not-exists")
	ServerCertFingerprintFile = path.Join(sb.BasePath, "server-cert-not-exists.sha256")

	// Act
	server, err := newGRPCServerWithTLS()

	// Assert
	require.ErrorContains(t, err, "the agent cannot start due to missing certificates")
	// Mention about the register command.
	require.ErrorContains(t, err, "stork-agent register")
	// Mention about the --server-url flag.
	require.ErrorContains(t, err, "--server-url")
	require.Nil(t, server)
}

// Test that the error is returned if the GRPC TLS certificate files are
// invalid.
func TestNewGRPCServerWithTLSInvalidCerts(t *testing.T) {
	// Arrange
	cleanup, _ := GenerateSelfSignedCerts()
	defer cleanup()
	certStore := NewCertStoreDefault()

	// Make the cert store invalid.
	certStore.RemoveServerCertFingerprint()

	// Act
	server, err := newGRPCServerWithTLS()

	// Assert
	require.ErrorContains(t, err, "cannot start due to invalid certificates")
	// Recommend to re-register the agent.
	require.ErrorContains(t, err, "stork-agent register")
	require.Nil(t, server)
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

// Test that the agent cannot be set up with invalid certificates.
func TestAgentSetupInvalidCerts(t *testing.T) {
	// Arrange
	sa, _, teardown := setupAgentTest()
	defer teardown()
	certStore := NewCertStoreDefault()
	certStore.RemoveServerCertFingerprint()

	// Act
	err := sa.SetupGRPCServer()

	// Assert
	require.ErrorContains(t, err, "cert store is not valid")
}

// Tests the peer verification function creation.
func TestCreateVerifyPeer(t *testing.T) {
	// Arrange
	allowedFingerprint := [32]byte{42}

	// Act
	verify := createVerifyPeer(allowedFingerprint)

	// Assert
	require.NotNil(t, verify)
}

// The verification function must deny access if the extended key usage is
// missing.
func TestVerifyPeerMissingExtendedKeyUsage(t *testing.T) {
	// Arrange
	cert := &x509.Certificate{Raw: []byte("foo")}
	fingerprint := sha256.Sum256(cert.Raw)

	verify := createVerifyPeer(fingerprint)

	// Act
	rsp, err := verify(&advancedtls.HandshakeVerificationInfo{
		Leaf: cert,
	})

	// Assert
	require.Nil(t, rsp)
	require.ErrorContains(t, err, "peer certificate does not have the extended key usage set")
}

// The verification function must deny access if the certificate fingerprint
// doesn't match the allowed one.
func TestVerifyPeerFingerprintMismatch(t *testing.T) {
	// Arrange
	cert := &x509.Certificate{
		Raw:         []byte("foo"),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	fingerprint := [32]byte{42}

	verify := createVerifyPeer(fingerprint)

	// Act
	rsp, err := verify(&advancedtls.HandshakeVerificationInfo{
		Leaf: cert,
	})

	// Assert
	require.Nil(t, rsp)
	require.ErrorContains(t, err, "peer certificate fingerprint does not match the allowed one")
}

// Test that the verification function allows access if the certificate meets
// the requirements.
func TestVerifyPeerCorrectCertificate(t *testing.T) {
	// Arrange
	cert := &x509.Certificate{
		Raw:         []byte("foo"),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	fingerprint := sha256.Sum256(cert.Raw)

	verify := createVerifyPeer(fingerprint)

	// Act
	rsp, err := verify(&advancedtls.HandshakeVerificationInfo{
		Leaf: cert,
	})

	// Assert
	require.NotNil(t, rsp)
	require.NoError(t, err)
}

// Test receiving a stream of zones filtered by view name.
func TestReceiveZonesFilterByView(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	slices.SortFunc(defaultZones, func(zone1, zone2 *bind9stats.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 app with the inventory.
	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "key", 1234, false)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		zoneInventory: inventory,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	// Mock the streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.Zone](ctrl)

	// The zones from the _default view should be returned in order.
	var mocks []any
	for _, zone := range defaultZones {
		apiZone := &agentapi.Zone{
			Name:           zone.Name(),
			Class:          zone.Class,
			Serial:         zone.Serial,
			Type:           zone.Type,
			Loaded:         zone.Loaded.Unix(),
			View:           "_default",
			TotalZoneCount: 10,
		}
		mocks = append(mocks, mock.EXPECT().Send(apiZone).Return(nil))
	}
	gomock.InOrder(mocks...)

	// Run the actual test.
	err = sa.ReceiveZones(&agentapi.ReceiveZonesReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ViewName:       "_default",
		LoadedAfter:    time.Date(2024, 2, 3, 15, 19, 0, 0, time.UTC).Unix(),
	}, mock)
	require.NoError(t, err)
}

// Test receiving a stream of zones filtered by loading time.
func TestReceiveZonesFilterByLoadedAfter(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	slices.SortFunc(defaultZones, func(zone1, zone2 *bind9stats.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 app with the inventory.
	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "key", 1234, false)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		zoneInventory: inventory,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	// Mock the streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.Zone](ctrl)

	// Run the actual test.
	err = sa.ReceiveZones(&agentapi.ReceiveZonesReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		LoadedAfter:    time.Date(2025, 2, 3, 15, 19, 0, 0, time.UTC).Unix(),
	}, mock)
	require.NoError(t, err)
}

// Test receiving a stream of zones filtered by lower bound and limit.
func TestReceiveZonesFilterLowerBound(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	slices.SortFunc(defaultZones, func(zone1, zone2 *bind9stats.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 app with the inventory.
	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "key", 1234, false)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		zoneInventory: inventory,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	// Mock the streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.Zone](ctrl)

	// The zones from the _default view should be returned in order
	// starting from 6th zones up to 9th.
	var mocks []any
	for _, zone := range defaultZones[6:9] {
		apiZone := &agentapi.Zone{
			Name:           zone.Name(),
			Class:          zone.Class,
			Serial:         zone.Serial,
			Type:           zone.Type,
			Loaded:         zone.Loaded.Unix(),
			View:           "_default",
			TotalZoneCount: 10,
		}
		mocks = append(mocks, mock.EXPECT().Send(apiZone).Return(nil))
	}
	gomock.InOrder(mocks...)

	// Receive 3 zones ordered after 5th zone.
	err = sa.ReceiveZones(&agentapi.ReceiveZonesReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		LowerBound:     defaultZones[5].Name(),
		Limit:          3,
	}, mock)
	require.NoError(t, err)
}

// Test that an error is returned when zone inventory is nil.
func TestReceiveZonesNilZoneInventory(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 app without zone inventory.
	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "key", 1234, false)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		zoneInventory: nil,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	// Mock the streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.Zone](ctrl)

	// Run the actual test.
	err := sa.ReceiveZones(&agentapi.ReceiveZonesReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ViewName:       "_default",
		LoadedAfter:    time.Date(2024, 2, 3, 15, 19, 0, 0, time.UTC).Unix(),
	}, mock)
	require.Contains(t, err.Error(), "attempted to receive DNS zones from an app for which zone inventory was not instantiated")
}

// Test that an error is returned when the app is not a DNS server.
func TestReceiveZonesUnsupportedApp(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add an app that is not a DNS server.
	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "key", 1234, false)
	var apps []App
	apps = append(apps, &KeaApp{
		BaseApp: BaseApp{
			Type:         AppTypeKea,
			AccessPoints: accessPoints,
		},
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	// Mock the streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.Zone](ctrl)

	// Run the actual test.
	err := sa.ReceiveZones(&agentapi.ReceiveZonesReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
	}, mock)
	require.Error(t, err)
	require.Contains(t, err.Error(), "attempted to receive DNS zones from an unsupported app")
}

// Test that specific error is returned when the zone inventory was not initialized
// while trying to receive the zones.
func TestReceiveZonesZoneInventoryNotInited(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	slices.SortFunc(defaultZones, func(zone1, zone2 *bind9stats.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 app with the inventory.
	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "key", 1234, false)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		zoneInventory: inventory,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	// Mock the streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.Zone](ctrl)

	// Run the actual test.
	err := sa.ReceiveZones(&agentapi.ReceiveZonesReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ViewName:       "_default",
		LoadedAfter:    time.Date(2024, 2, 3, 15, 19, 0, 0, time.UTC).Unix(),
	}, mock)
	require.Error(t, err)
	require.Equal(t, "rpc error: code = FailedPrecondition desc = zone inventory has not been initialized yet", err.Error())

	// The zone inventory was not initialized so we expect that it is returned
	// as an error over gRPC.
	s := status.Convert(err)
	details := s.Details()
	require.Len(t, details, 1)
	info, ok := details[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	require.Equal(t, "ZONE_INVENTORY_NOT_INITED", info.Reason)
}

// Test that specific error is returned when the zone inventory was busy
// while trying to receive the zones.
func TestReceiveZonesZoneInventoryBusy(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	slices.SortFunc(defaultZones, func(zone1, zone2 *bind9stats.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	// Start receiving zones but don't complete it. It turns the inventory
	// into "busy" state.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = inventory.receiveZones(ctx, nil)
	require.NoError(t, err)

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 app with the inventory.
	accessPoints := makeAccessPoint(AccessPointControl, "127.0.0.1", "_", 1234, false)
	var apps []App
	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		zoneInventory: inventory,
	})
	fam, _ := sa.AppMonitor.(*FakeAppMonitor)
	fam.Apps = apps

	// Mock the streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.Zone](ctrl)

	// Run the actual test.
	err = sa.ReceiveZones(&agentapi.ReceiveZonesReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ViewName:       "_default",
		LoadedAfter:    time.Date(2024, 2, 3, 15, 19, 0, 0, time.UTC).Unix(),
	}, mock)
	require.Error(t, err)
	require.Equal(t, "rpc error: code = Unavailable desc = cannot transition to the RECEIVING_ZONES state while the zone inventory is in RECEIVING_ZONES state", err.Error())

	// The zone inventory was busy so we expect that it is returned as an
	// error over gRPC.
	s := status.Convert(err)
	details := s.Details()
	require.Len(t, details, 1)
	info, ok := details[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	require.Equal(t, "ZONE_INVENTORY_BUSY_ERROR", info.Reason)
}
