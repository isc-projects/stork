package agent

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/security/advancedtls"
	"google.golang.org/grpc/status"
	"gopkg.in/h2non/gock.v1"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
	bind9config "isc.org/stork/daemoncfg/bind9"
	pdnsdata "isc.org/stork/daemondata/pdns"
	"isc.org/stork/datamodel/daemonname"
	dnsmodel "isc.org/stork/datamodel/dns"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/hooks"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=agent -destination=serverstreamingservermock_test.go google.golang.org/grpc ServerStreamingServer
//go:generate mockgen -package=agent -destination=dnsconfigaccessormock_test.go -mock_names=dnsConfigAccessor=MockDNSConfigAccessor isc.org/stork/agent dnsConfigAccessor

//go:embed testdata/valid-zone.json
var validZoneData []byte

type FakeMonitor struct {
	Daemons []Daemon
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

	pdnsClient := newPDNSClient()
	gock.InterceptClient(pdnsClient.innerClient.GetClient())

	httpClientConfig := HTTPClientConfig{SkipTLSVerification: true, Interceptor: gock.InterceptClient}

	cleanupCerts, _ := GenerateSelfSignedCerts()

	keaAccessPoint := AccessPoint{
		Type:     AccessPointControl,
		Address:  "localhost",
		Port:     45634,
		Protocol: protocoltype.HTTP,
	}

	fdm := FakeMonitor{
		Daemons: []Daemon{
			&keaDaemon{
				daemon: daemon{
					Name:         daemonname.DHCPv4,
					AccessPoints: []AccessPoint{keaAccessPoint},
				},
				connector: newKeaConnector(keaAccessPoint, httpClientConfig),
			},
			&Bind9Daemon{
				dnsDaemonImpl: dnsDaemonImpl{
					daemon: daemon{
						Name: daemonname.Bind9,
						AccessPoints: []AccessPoint{{
							Type:     AccessPointControl,
							Address:  "localhost",
							Port:     45635,
							Protocol: protocoltype.RNDC,
						}},
					},
				},
			},
		},
	}

	sa := &StorkAgent{
		Monitor:             &fdm,
		bind9StatsClient:    bind9StatsClient,
		pdnsClient:          pdnsClient,
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
	return sa, ctx, func() {
		cleanupCerts()
		gock.Off()
	}
}

// Stub function for Monitor. It returns a fixed list of daemons.
func (fdm *FakeMonitor) GetDaemons() []Daemon {
	return fdm.Daemons
}

// Stub function for Monitor. It behaves in the same way as original one.
func (fdm *FakeMonitor) GetDaemonByAccessPoint(apType, address string, port int64) Daemon {
	for _, daemon := range fdm.GetDaemons() {
		for _, ap := range daemon.GetAccessPoints() {
			if ap.Type == apType && ap.Address == address && ap.Port == port {
				return daemon
			}
		}
	}
	return nil
}

func (fdm *FakeMonitor) Shutdown() {
}

func (fdm *FakeMonitor) Start(context.Context, agentManager) {
}

// A matcher for PowerDNS zones. It excludes the Loaded field which is
// dynamically set by the agent, and not returned by the PowerDNS API.
type powerDNSZoneMatcher struct {
	expected *agentapi.Zone
}

// Checks of the zone matches the expected zone.
func (m *powerDNSZoneMatcher) Matches(actual any) bool {
	zone, ok := actual.(*agentapi.Zone)
	if !ok {
		return false
	}
	return zone.Name == m.expected.Name &&
		zone.Class == m.expected.Class &&
		zone.Serial == m.expected.Serial &&
		zone.Type == m.expected.Type &&
		zone.View == m.expected.View &&
		zone.TotalZoneCount == m.expected.TotalZoneCount
}

// Returns expected zone data.
func (m *powerDNSZoneMatcher) String() string {
	return fmt.Sprintf("Zone with Name=%s, Class=%s, Serial=%d, Type=%s, View=%s, TotalZoneCount=%d (Loaded ignored)",
		m.expected.Name, m.expected.Class, m.expected.Serial, m.expected.Type, m.expected.View, m.expected.TotalZoneCount)
}

// Check if NewStorkAgent can be invoked and sets SA members.
func TestNewStorkAgent(t *testing.T) {
	fdm := &FakeMonitor{}
	bind9StatsClient := NewBind9StatsClient()
	keaHTTPClientConfig := HTTPClientConfig{}
	sa := NewStorkAgent(
		"foo", 42, fdm, bind9StatsClient, NewHookManager(),
	)
	require.NotNil(t, sa.Monitor)
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
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = nil

	// daemon monitor is empty, no daemons should be returned by GetState
	rsp, err := sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
	require.Empty(t, rsp.Daemons)

	// add some daemons to daemon monitor so GetState should return something
	var daemons []Daemon
	daemons = append(daemons, &keaDaemon{
		daemon: daemon{
			Name: daemonname.DHCPv4,
			AccessPoints: []AccessPoint{{
				Type:     AccessPointControl,
				Address:  "1.2.3.1",
				Port:     1234,
				Protocol: protocoltype.HTTP,
			}},
		},
	})

	accessPoints := []AccessPoint{
		{
			Type:     AccessPointControl,
			Address:  "2.3.4.4",
			Port:     2345,
			Key:      "abcd",
			Protocol: protocoltype.HTTPS,
		},
		{
			Type:     AccessPointStatistics,
			Address:  "2.3.4.5",
			Port:     2346,
			Key:      "foo",
			Protocol: protocoltype.HTTP,
		},
	}

	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name:         daemonname.Bind9,
				AccessPoints: accessPoints,
			},
		},
	})
	fdm.Daemons = daemons
	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
	require.Equal(t, stork.Version, rsp.AgentVersion)
	require.False(t, rsp.AgentUsesHTTPCredentials) //nolint:staticcheck,deprecated
	require.Len(t, rsp.Daemons, 2)

	daemonKea := rsp.Daemons[0]
	require.Len(t, daemonKea.AccessPoints, 1)
	point := daemonKea.AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "1.2.3.1", point.Address)
	require.False(t, point.UseSecureProtocol) //nolint:staticcheck,deprecated
	require.EqualValues(t, 1234, point.Port)
	require.Empty(t, point.Key)

	daemonBind9 := rsp.Daemons[1]
	require.Len(t, daemonBind9.AccessPoints, 2)
	// sorted by port
	point = daemonBind9.AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "2.3.4.4", point.Address)
	require.EqualValues(t, 2345, point.Port)
	require.Equal(t, "abcd", point.Key)
	require.True(t, point.UseSecureProtocol) //nolint:staticcheck,deprecated
	point = daemonBind9.AccessPoints[1]
	require.Equal(t, AccessPointStatistics, point.Type)
	require.Equal(t, "2.3.4.5", point.Address)
	require.EqualValues(t, 2346, point.Port)
	require.False(t, point.UseSecureProtocol) //nolint:staticcheck,deprecated
	require.EqualValues(t, "foo", point.Key)

	// Recreate Stork agent.
	sa, ctx, teardown = setupAgentTest()
	defer teardown()

	daemon := fdm.GetDaemonByAccessPoint(AccessPointControl, "1.2.3.1", 1234).(*keaDaemon)
	daemon.connector = newKeaConnector(daemon.AccessPoints[0], HTTPClientConfig{
		BasicAuth: basicAuthCredentials{User: "foo", Password: "bar"},
	})

	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	// Deprecated parameter. Always false.
	require.False(t, rsp.AgentUsesHTTPCredentials) //nolint:staticcheck,deprecated
}

// Check if GetState works even if the daemon has multiple access points of
// the same type.
func TestGetStateMultipleAccessPointsSameType(t *testing.T) {
	// Arrange
	sa, ctx, teardown := setupAgentTest()
	defer teardown()
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = nil

	// daemon monitor is empty, no daemons should be returned by GetState
	rsp, err := sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
	require.Empty(t, rsp.Daemons)

	// add some daemons to daemon monitor so GetState should return something
	var daemons []Daemon
	daemons = append(daemons, &keaDaemon{
		daemon: daemon{
			Name: daemonname.DHCPv4,
			AccessPoints: []AccessPoint{
				{
					Type:     AccessPointControl,
					Address:  "/var/run/kea/kea-dhcp4.sock",
					Protocol: protocoltype.Socket,
				},
				{
					Type:     AccessPointControl,
					Address:  "1.2.3.1",
					Port:     1234,
					Protocol: protocoltype.HTTP,
				},
			},
		},
	})
	fdm.Daemons = daemons

	// Act
	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})

	// Assert
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
	require.Equal(t, stork.Version, rsp.AgentVersion)
	require.Len(t, rsp.Daemons, 1)
	daemon := rsp.Daemons[0]
	require.Len(t, daemon.AccessPoints, 1)
	accessPoint := daemon.AccessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, "unix", accessPoint.Protocol)
	require.Equal(t, "/var/run/kea/kea-dhcp4.sock", accessPoint.Address)
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
	keaResponse := rsp.KeaResponses[0]
	status := keaResponse.GetStatus()
	require.Zero(t, status.Code, status.Message)
	require.JSONEq(t, "{\"result\":0, \"text\":\"\"}", string(keaResponse.Response))
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
	require.Equal(t, agentapi.Status_ERROR, rsp.KeaResponses[0].Status.Code)
	require.Contains(t, rsp.KeaResponses[0].Status.Message, "received non-success status code 400 from Kea, with status text: 400 Bad Request; url: http://localhost:45634/")
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

// Test forwarding statistics request to named for different request types.
func TestForwardToNamedStatsForDifferentRequestTypes(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

	tests := []struct {
		name        string
		requestType agentapi.ForwardToNamedStatsReq_RequestType
		paths       []string
	}{
		{"default", agentapi.ForwardToNamedStatsReq_DEFAULT, []string{""}},
		{"status", agentapi.ForwardToNamedStatsReq_STATUS, []string{"/status"}},
		{"server", agentapi.ForwardToNamedStatsReq_SERVER, []string{"/server"}},
		{"zones", agentapi.ForwardToNamedStatsReq_ZONES, []string{"/zones"}},
		{"network", agentapi.ForwardToNamedStatsReq_NETWORK, []string{"/net"}},
		{"memory", agentapi.ForwardToNamedStatsReq_MEMORY, []string{"/mem"}},
		{"traffic", agentapi.ForwardToNamedStatsReq_TRAFFIC, []string{"/traffic"}},
		{"server and traffic", agentapi.ForwardToNamedStatsReq_SERVER_AND_TRAFFIC, []string{"/server", "/traffic"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Expect appropriate content type and the body. If they are not matched
			// an error will be raised.
			defer gock.Off()
			for _, path := range test.paths {
				gock.New("http://localhost:45634/").
					MatchHeader("Accept", "application/json").
					Get(fmt.Sprintf("json/v1%s", path)).
					Reply(200).
					JSON(map[string]int{"result": 0})
			}

			// Forward the request with the expected body.
			req := &agentapi.ForwardToNamedStatsReq{
				Url:               "http://localhost:45634/",
				RequestType:       test.requestType,
				StatsAddress:      "localhost",
				StatsPort:         45634,
				NamedStatsRequest: &agentapi.NamedStatsRequest{Request: ""},
			}

			// Named should respond with non-empty body and the status code 200.
			// This should result in no error and the body should be available
			// in the response.
			rsp, err := sa.ForwardToNamedStats(ctx, req)
			require.NotNil(t, rsp)
			require.NoError(t, err)
			require.NotNil(t, rsp.NamedStatsResponse)
			require.JSONEq(t, "{\"result\":0}", rsp.NamedStatsResponse.Response)
		})
	}
}

// Test that a response error is properly handled when forwarding a request
// to get combined server and traffic statistics.
func TestForwardToNamedStatsServerAndTrafficError(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()

	defer gock.Off()

	// First request is successful.
	gock.New("http://localhost:45634/").
		MatchHeader("Accept", "application/json").
		Get("json/v1/server").
		Reply(200).
		JSON(map[string]int{"result": 0})

	// Second request is not successful.
	gock.New("http://localhost:45634/").
		MatchHeader("Accept", "application/json").
		Get("json/v1/traffic").
		Reply(404).
		JSON(map[string]string{"error": "Not Found"})

	req := &agentapi.ForwardToNamedStatsReq{
		Url:               "http://localhost:45634/",
		RequestType:       agentapi.ForwardToNamedStatsReq_SERVER_AND_TRAFFIC,
		StatsAddress:      "localhost",
		StatsPort:         45634,
		NamedStatsRequest: &agentapi.NamedStatsRequest{Request: ""},
	}

	// The response should contain the error even though the first request was successful.
	rsp, err := sa.ForwardToNamedStats(ctx, req)
	require.NotNil(t, rsp)
	require.NoError(t, err)
	require.NotNil(t, rsp.NamedStatsResponse)
	require.NotNil(t, rsp.NamedStatsResponse.Status)
	require.Equal(t, agentapi.Status_ERROR, rsp.NamedStatsResponse.Status.Code)
	require.Empty(t, rsp.NamedStatsResponse.Response)
}

// Test a successful rndc command.
func TestForwardRndcCommandSuccess(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()
	executor := newTestCommandExecutorDefault()
	rndcClient := NewRndcClient(executor)
	rndcClient.BaseCommand = []string{"/rndc"}

	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Protocol: protocoltype.RNDC,
				}},
			},
		},
		rndcClient: rndcClient,
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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

	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Protocol: protocoltype.RNDC,
				}},
			},
		},
		rndcClient: rndcClient,
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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

// Test rndc command when there is no daemon.
func TestForwardRndcCommandNoDaemon(t *testing.T) {
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
	require.EqualValues(t, "cannot find BIND 9 daemon", rsp.Status.Message)
}

// Test rndc command successfully forwarded, but bad response.
func TestForwardRndcCommandEmpty(t *testing.T) {
	sa, ctx, teardown := setupAgentTest()
	defer teardown()
	executor := newTestCommandExecutorDefault().setRndcStatus("", nil)
	rndcClient := NewRndcClient(executor)
	rndcClient.BaseCommand = []string{"/rndc"}

	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Protocol: protocoltype.RNDC,
				}},
			},
		},
		rndcClient: rndcClient,
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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
	slices.SortFunc(defaultZones, func(zone1, zone2 *dnsmodel.Zone) int {
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
	config := parseDefaultBind9Config(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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

// Test receiving a stream of zones from PowerDNS.
func TestReceiveZonesPDNS(t *testing.T) {
	// Setup server response.
	randomZones := generateRandomPDNSZones(20)
	slices.SortFunc(randomZones, func(zone1, zone2 *pdnsdata.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	pdnsClient, off := setGetViewsPowerDNSResponseOK(t, randomZones)
	defer off()

	// Create zone inventory.
	config := parseDefaultPDNSConfig(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, pdnsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a PowerDNS daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Protocol: protocoltype.HTTP,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Mock the streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.Zone](ctrl)

	// The zones from the localhost view should be returned in order.
	var mocks []any
	for _, zone := range randomZones {
		apiZone := &agentapi.Zone{
			Name:           zone.Name(),
			Class:          "IN",
			Serial:         zone.Serial,
			Type:           zone.Kind,
			Loaded:         time.Now().Unix(),
			View:           "localhost",
			TotalZoneCount: int64(len(randomZones)),
		}
		mocks = append(mocks, mock.EXPECT().Send(&powerDNSZoneMatcher{apiZone}).Return(nil))
	}
	gomock.InOrder(mocks...)

	// Run the actual test.
	err = sa.ReceiveZones(&agentapi.ReceiveZonesReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ViewName:       "localhost",
		LoadedAfter:    time.Date(2024, 2, 3, 15, 19, 0, 0, time.UTC).Unix(),
	}, mock)
	require.NoError(t, err)
}

// Test receiving a stream of RPZ zones.
func TestReceiveRPZZones(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	slices.SortFunc(defaultZones, func(zone1, zone2 *dnsmodel.Zone) int {
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

	sa, _, teardown := setupAgentTest()
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create zone inventory with the configuration that marks each zone as RPZ.
	rpzMock := NewMockDNSConfigAccessor(ctrl)
	rpzMock.EXPECT().IsRPZ(gomock.Any(), gomock.Any()).AnyTimes().Return(true)
	rpzMock.EXPECT().GetAPIKey().AnyTimes().Return("")

	inventory := newZoneInventory(newZoneInventoryStorageMemory(), rpzMock, bind9StatsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	// Add a BIND9 daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Mock the streaming server.
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
			Rpz:            true,
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
	slices.SortFunc(defaultZones, func(zone1, zone2 *dnsmodel.Zone) int {
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
	config := parseDefaultBind9Config(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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
	slices.SortFunc(defaultZones, func(zone1, zone2 *dnsmodel.Zone) int {
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
	config := parseDefaultBind9Config(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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

	// Add a BIND9 daemon without zone inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: nil,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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
	require.Contains(t, err.Error(), "attempted to receive DNS zones from a daemon for which zone inventory was not instantiated")
}

// Test that an error is returned when the daemon is not a DNS server.
func TestReceiveZonesUnsupportedDaemon(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a daemon that is not a DNS server.
	var daemons []Daemon
	daemons = append(daemons, &keaDaemon{
		daemon: daemon{
			Name: daemonname.DHCPv4,
			AccessPoints: []AccessPoint{{
				Type:     AccessPointControl,
				Address:  "127.0.0.1",
				Port:     1234,
				Key:      "key",
				Protocol: protocoltype.HTTP,
			}},
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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
	require.Contains(t, err.Error(), "attempted to receive DNS zones from an unsupported daemon")
}

// Test that specific error is returned when the zone inventory was not initialized
// while trying to receive the zones.
func TestReceiveZonesZoneInventoryNotInited(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	slices.SortFunc(defaultZones, func(zone1, zone2 *dnsmodel.Zone) int {
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
	config := parseDefaultBind9Config(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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
	slices.SortFunc(defaultZones, func(zone1, zone2 *dnsmodel.Zone) int {
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
	config := parseDefaultBind9Config(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	done, err := inventory.populate(true)
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

	// Add a BIND9 daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

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
	require.Equal(t, "ZONE_INVENTORY_BUSY", info.Reason)
}

// Test successfully receiving a stream of zone RRs.
func TestReceiveZoneRRs(t *testing.T) {
	// Setup server response for populating the zone inventory.
	trustedZones := generateRandomZones(10)
	slices.SortFunc(trustedZones, func(zone1, zone2 *dnsmodel.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	guestZones := generateRandomZones(10)
	response := map[string]any{
		"views": map[string]any{
			"trusted": map[string]any{
				"zones": trustedZones,
			},
			"guest": map[string]any{
				"zones": guestZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	config := parseDefaultBind9Config(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	// Get the example zone contents from the file.
	var rrs []string
	err := json.Unmarshal(validZoneData, &rrs)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Replace the default AXFR executor to mock the AXFR response.
	axfrExecutor := NewMockZoneInventoryAXFRExecutor(ctrl)
	axfrExecutor.EXPECT().run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(transfer *dns.Transfer, message *dns.Msg, address string) (chan *dns.Envelope, error) {
		require.NotNil(t, transfer.TsigSecret)
		require.Len(t, transfer.TsigSecret, 1)
		require.Contains(t, transfer.TsigSecret, "trusted-key.")
		require.Equal(t, transfer.TsigSecret["trusted-key."], "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=")
		require.Len(t, message.Question, 1)
		require.Contains(t, message.Question[0].Name, trustedZones[0].Name())
		require.Equal(t, "127.0.0.1:53", address)
		ch := make(chan *dns.Envelope)
		go func() {
			for _, rr := range rrs {
				rr, err := dns.NewRR(rr)
				require.NoError(t, err)
				ch <- &dns.Envelope{
					RR: []dns.RR{rr},
				}
			}
			close(ch)
		}()
		return ch, nil
	})
	inventory.axfrExecutor = axfrExecutor

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	// Add a BIND9 daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Mock the streaming server.
	mock := NewMockServerStreamingServer[agentapi.ReceiveZoneRRsRsp](ctrl)

	// The zone RRs should be returned in order.
	var mocks []any
	for i, rr := range rrs {
		rsp := &agentapi.ReceiveZoneRRsRsp{
			Rrs: []string{rr},
		}
		mocks = append(mocks, mock.EXPECT().Send(rsp).DoAndReturn(func(rsp *agentapi.ReceiveZoneRRsRsp) error {
			require.Equal(t, rr, rrs[i])
			return nil
		}))
	}
	gomock.InOrder(mocks...)

	// Run the actual test.
	err = sa.ReceiveZoneRRs(&agentapi.ReceiveZoneRRsReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ZoneName:       trustedZones[0].Name(),
		ViewName:       "trusted",
	}, mock)
	require.NoError(t, err)
}

// Test successfully receiving a stream of zone RRs from PowerDNS.
func TestReceiveZoneRRsPowerDNS(t *testing.T) {
	// Setup server response for populating the zone inventory.
	randomZones := generateRandomPDNSZones(10)
	slices.SortFunc(randomZones, func(zone1, zone2 *pdnsdata.Zone) int {
		return storkutil.CompareNames(zone1.ZoneName, zone2.ZoneName)
	})
	pdnsClient, off := setGetViewsPowerDNSResponseOK(t, randomZones)
	defer off()

	// Create zone inventory.
	config := parseDefaultPDNSConfig(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, pdnsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	// Get the example zone contents from the file.
	var rrs []string
	err := json.Unmarshal(validZoneData, &rrs)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Replace the default AXFR executor to mock the AXFR response.
	axfrExecutor := NewMockZoneInventoryAXFRExecutor(ctrl)
	axfrExecutor.EXPECT().run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(transfer *dns.Transfer, message *dns.Msg, address string) (chan *dns.Envelope, error) {
		require.Len(t, message.Question, 1)
		require.Contains(t, message.Question[0].Name, randomZones[0].ZoneName)
		require.Equal(t, "127.0.0.1:53", address)
		ch := make(chan *dns.Envelope)
		go func() {
			for _, rr := range rrs {
				rr, err := dns.NewRR(rr)
				require.NoError(t, err)
				ch <- &dns.Envelope{
					RR: []dns.RR{rr},
				}
			}
			close(ch)
		}()
		return ch, nil
	})
	inventory.axfrExecutor = axfrExecutor

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	// Add a PowerDNS daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Protocol: protocoltype.HTTP,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Mock the streaming server.
	mock := NewMockServerStreamingServer[agentapi.ReceiveZoneRRsRsp](ctrl)

	// The zone RRs should be returned in order.
	var mocks []any
	for i, rr := range rrs {
		rsp := &agentapi.ReceiveZoneRRsRsp{
			Rrs: []string{rr},
		}
		mocks = append(mocks, mock.EXPECT().Send(rsp).DoAndReturn(func(rsp *agentapi.ReceiveZoneRRsRsp) error {
			require.Equal(t, rr, rrs[i])
			return nil
		}))
	}
	gomock.InOrder(mocks...)

	// Run the actual test.
	err = sa.ReceiveZoneRRs(&agentapi.ReceiveZoneRRsReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ZoneName:       randomZones[0].ZoneName,
		ViewName:       "localhost",
	}, mock)
	require.NoError(t, err)
}

// Test that an error is returned when zone inventory is nil.
func TestReceiveZoneRRsNilZoneInventory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 daemon with the nil zone inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Protocol: protocoltype.RNDC,
				}},
			},
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Mock the streaming server.
	mock := NewMockServerStreamingServer[agentapi.ReceiveZoneRRsRsp](ctrl)

	// Run the actual test.
	err := sa.ReceiveZoneRRs(&agentapi.ReceiveZoneRRsReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ZoneName:       "example.com",
		ViewName:       "trusted",
	}, mock)
	require.Contains(t, err.Error(), "attempted to receive DNS zone RRs from a daemon for which zone inventory was not instantiated")
}

// Test that specific error is returned when the zone inventory was not initialized
// while trying to receive the zone RRs.
func TestReceiveZoneRRsZoneInventoryNotInited(t *testing.T) {
	// Setup server response for populating the zone inventory.
	trustedZones := generateRandomZones(10)
	slices.SortFunc(trustedZones, func(zone1, zone2 *dnsmodel.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	guestZones := generateRandomZones(10)
	response := map[string]any{
		"views": map[string]any{
			"trusted": map[string]any{
				"zones": trustedZones,
			},
			"guest": map[string]any{
				"zones": guestZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory but do not populate it.
	config := parseDefaultBind9Config(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

	// Get the example zone contents from the file.
	var rrs []string
	err := json.Unmarshal(validZoneData, &rrs)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Replace the default AXFR executor to mock the AXFR response.
	axfrExecutor := NewMockZoneInventoryAXFRExecutor(ctrl)
	axfrExecutor.EXPECT().run(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(0)
	inventory.axfrExecutor = axfrExecutor

	// Add a BIND9 daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Mock the streaming server.
	mock := NewMockServerStreamingServer[agentapi.ReceiveZoneRRsRsp](ctrl)
	mock.EXPECT().Send(gomock.Any()).MaxTimes(0)

	// Run the actual test.
	err = sa.ReceiveZoneRRs(&agentapi.ReceiveZoneRRsReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ZoneName:       trustedZones[0].Name(),
		ViewName:       "trusted",
	}, mock)
	require.Contains(t, err.Error(), "rpc error: code = FailedPrecondition desc = zone inventory has not been initialized yet")

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
// while trying to receive the zone RRs.
func TestReceiveZoneRRsZoneInventoryBusy(t *testing.T) {
	// Setup server response for populating the zone inventory.
	trustedZones := generateRandomZones(10)
	slices.SortFunc(trustedZones, func(zone1, zone2 *dnsmodel.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	guestZones := generateRandomZones(10)
	response := map[string]any{
		"views": map[string]any{
			"trusted": map[string]any{
				"zones": trustedZones,
			},
			"guest": map[string]any{
				"zones": guestZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	config := parseDefaultBind9Config(t)
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	inventory.start()
	defer inventory.stop()

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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Replace the default AXFR executor to mock the AXFR response.
	// Expect that the AXFR request is not executed.
	axfrExecutor := NewMockZoneInventoryAXFRExecutor(ctrl)
	axfrExecutor.EXPECT().run(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(0)
	inventory.axfrExecutor = axfrExecutor

	// Add a BIND9 daemon with the inventory.
	var daemons []Daemon
	daemons = append(daemons, &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "127.0.0.1",
					Port:     1234,
					Key:      "key",
					Protocol: protocoltype.RNDC,
				}},
			},
			zoneInventory: inventory,
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Mock the streaming server.
	mock := NewMockServerStreamingServer[agentapi.ReceiveZoneRRsRsp](ctrl)
	mock.EXPECT().Send(gomock.Any()).MaxTimes(0)

	// Run the actual test.
	err = sa.ReceiveZoneRRs(&agentapi.ReceiveZoneRRsReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		ZoneName:       trustedZones[0].Name(),
		ViewName:       "trusted",
	}, mock)
	require.Contains(t, err.Error(), "rpc error: code = Unavailable desc = failed to submit AXFR request to the worker pool: zone transfer is not possible because the zone inventory is in RECEIVING_ZONES state")

	// The zone inventory was busy so we expect that it is returned as an
	// error over gRPC.
	s := status.Convert(err)
	details := s.Details()
	require.Len(t, details, 1)
	info, ok := details[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	require.Equal(t, "ZONE_INVENTORY_BUSY", info.Reason)
}

// Test that the PowerDNS server information is returned and
// parsed successfully.
func TestGetPowerDNSServerInfo(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	defer gock.Off()
	gock.New("http://localhost:1234/").
		MatchHeader("X-API-Key", "stork").
		Get("api/v1/servers/localhost").
		Reply(http.StatusOK).
		JSON(map[string]any{
			"type":              "Server",
			"version":           "4.7.3",
			"id":                "localhost",
			"daemon_type":       "authoritative",
			"url":               "http://localhost:1234/api/v1/servers/localhost",
			"config_url":        "http://localhost:1234/api/v1/servers/localhost/config",
			"zones_url":         "http://localhost:1234/api/v1/servers/localhost/zones",
			"autoprimaries_url": "http://localhost:1234/api/v1/servers/localhost/autoprimaries",
		})

	gock.New("http://localhost:1234/").
		MatchHeader("X-API-Key", "stork").
		Get("api/v1/servers/localhost/statistics").
		MatchParam("statistic", "uptime").
		Reply(http.StatusOK).
		JSON([]map[string]any{
			{
				"name":  "uptime",
				"value": "1234",
			},
		})

	// Add a PowerDNS daemon.
	var daemons []Daemon
	daemons = append(daemons, &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "localhost",
					Port:     1234,
					Key:      "stork",
					Protocol: protocoltype.HTTP,
				}},
			},
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Get the server information.
	rsp, err := sa.GetPowerDNSServerInfo(context.Background(), &agentapi.GetPowerDNSServerInfoReq{
		WebserverAddress: "localhost",
		WebserverPort:    1234,
	})
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.Equal(t, "Server", rsp.Type)
	require.Equal(t, "4.7.3", rsp.Version)
	require.Equal(t, "localhost", rsp.Id)
	require.Equal(t, "authoritative", rsp.DaemonType)
	require.Equal(t, "http://localhost:1234/api/v1/servers/localhost", rsp.Url)
	require.Equal(t, "http://localhost:1234/api/v1/servers/localhost/config", rsp.ConfigURL)
	require.Equal(t, "http://localhost:1234/api/v1/servers/localhost/zones", rsp.ZonesURL)
	require.Equal(t, "http://localhost:1234/api/v1/servers/localhost/autoprimaries", rsp.AutoprimariesURL)
	require.EqualValues(t, 1234, rsp.Uptime)
}

// Test that the correct error is returned when the specified PowerDNS
// server was not found by the control address and port.
func TestGetPowerDNSServerInfoNoDaemon(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = []Daemon{}

	// Get the server information.
	rsp, err := sa.GetPowerDNSServerInfo(context.Background(), &agentapi.GetPowerDNSServerInfoReq{
		WebserverAddress: "localhost",
		WebserverPort:    1234,
	})
	require.Error(t, err)
	require.Nil(t, rsp)

	// Make sure that the correct status and details were returned.
	st := status.Convert(err)
	require.Equal(t, codes.FailedPrecondition, st.Code())
	require.Equal(t, "PowerDNS server localhost:1234 not found", st.Message())
	details := st.Details()
	require.Len(t, details, 1)
	info, ok := details[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	require.Equal(t, "DAEMON_NOT_FOUND", info.Reason)
}

// Test that the correct error is returned when the API key is not configured
// for the PowerDNS server.
func TestGetPowerDNSServerInfoNoAPIKey(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a PowerDNS daemon with no API key.
	var daemons []Daemon
	daemons = append(daemons, &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "localhost",
					Port:     1234,
					Key:      "",
					Protocol: protocoltype.HTTP,
				}},
			},
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Get the server information.
	rsp, err := sa.GetPowerDNSServerInfo(context.Background(), &agentapi.GetPowerDNSServerInfoReq{
		WebserverAddress: "localhost",
		WebserverPort:    1234,
	})
	require.Error(t, err)
	require.Nil(t, rsp)

	// Make sure that the correct status and details were returned.
	st := status.Convert(err)
	require.Equal(t, codes.FailedPrecondition, st.Code())
	require.Equal(t, "API key not configured for PowerDNS server localhost:1234", st.Message())
	details := st.Details()
	require.Len(t, details, 1)
	info, ok := details[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	require.Equal(t, "API_KEY_NOT_CONFIGURED", info.Reason)
}

// Test that the correct error is returned when the PowerDNS server
// returns an error response.
func TestGetPowerDNSServerInfoErrorResponse(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	defer gock.Off()
	gock.New("http://localhost:1234/").
		MatchHeader("X-API-Key", "stork").
		Get("api/v1/servers/localhost").
		Reply(http.StatusInternalServerError).
		BodyString("Internal server error")

	// Add a PowerDNS daemon.
	var daemons []Daemon
	daemons = append(daemons, &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "localhost",
					Port:     1234,
					Key:      "stork",
					Protocol: protocoltype.HTTP,
				}},
			},
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Get the server information.
	rsp, err := sa.GetPowerDNSServerInfo(context.Background(), &agentapi.GetPowerDNSServerInfoReq{
		WebserverAddress: "localhost",
		WebserverPort:    1234,
	})
	require.Error(t, err)
	require.Nil(t, rsp)

	// Make sure that the correct status was returned.
	st := status.Convert(err)
	require.Equal(t, codes.Unknown, st.Code())
	require.Equal(t, "Internal server error", st.Message())
	details := st.Details()
	require.Empty(t, details)
}

// Test that the correct error is returned when the PowerDNS server
// returns an error response for the statistics endpoint.
func TestGetPowerDNSServerInfoStatisticsErrorResponse(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	defer gock.Off()
	gock.New("http://localhost:1234/").
		MatchHeader("X-API-Key", "stork").
		Get("api/v1/servers/localhost").
		Reply(http.StatusOK).
		JSON(map[string]any{
			"type":              "Server",
			"version":           "4.7.3",
			"id":                "localhost",
			"daemon_type":       "authoritative",
			"url":               "http://localhost:1234/api/v1/servers/localhost",
			"config_url":        "http://localhost:1234/api/v1/servers/localhost/config",
			"zones_url":         "http://localhost:1234/api/v1/servers/localhost/zones",
			"autoprimaries_url": "http://localhost:1234/api/v1/servers/localhost/autoprimaries",
		})

	gock.New("http://localhost:1234/").
		MatchHeader("X-API-Key", "stork").
		Get("api/v1/servers/localhost/statistics").
		MatchParam("statistic", "uptime").
		Reply(http.StatusInternalServerError).
		BodyString("Internal server error")

	// Add a PowerDNS daemon.
	var daemons []Daemon
	daemons = append(daemons, &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
				AccessPoints: []AccessPoint{{
					Type:     AccessPointControl,
					Address:  "localhost",
					Port:     1234,
					Key:      "stork",
					Protocol: protocoltype.HTTP,
				}},
			},
		},
	})
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = daemons

	// Get the server information.
	rsp, err := sa.GetPowerDNSServerInfo(context.Background(), &agentapi.GetPowerDNSServerInfoReq{
		WebserverAddress: "localhost",
		WebserverPort:    1234,
	})
	require.Error(t, err)
	require.Nil(t, rsp)

	// Make sure that the correct status was returned.
	st := status.Convert(err)
	require.Equal(t, codes.Unknown, st.Code())
	require.Equal(t, "Internal server error", st.Message())
	details := st.Details()
	require.Empty(t, details)
}

// In this structure we will collect the information about the received files
// as we get them over the stream.
type receivedBind9File struct {
	fileType   agentapi.Bind9ConfigFileType
	sourcePath string
	contents   []string
}

// Test getting BIND 9 configuration from a server without filters.
func TestReceiveBind9ConfigNoFiltersNorFileTypes(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Create a BIND 9 daemon.
	accessPoints := []AccessPoint{{
		Type:     AccessPointControl,
		Address:  "127.0.0.1",
		Key:      "key",
		Port:     1234,
		Protocol: protocoltype.RNDC,
	}}

	daemon := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name:         daemonname.Bind9,
				AccessPoints: accessPoints,
			},
		},
		bind9Config:   parseDefaultBind9Config(t),
		rndcKeyConfig: parseDefaultBind9RNDCKeyConfig(t),
	}

	fam, _ := sa.Monitor.(*FakeMonitor)
	fam.Daemons = []Daemon{daemon}

	// Mock the server streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.ReceiveBind9ConfigRsp](ctrl)

	var receivedFiles []*receivedBind9File

	// Mock the function sending the actual data over the stream. In this mock
	// we collect the information about the received files and their contents.
	mock.EXPECT().Send(gomock.Any()).AnyTimes().DoAndReturn(func(rsp *agentapi.ReceiveBind9ConfigRsp) error {
		switch r := rsp.Response.(type) {
		case *agentapi.ReceiveBind9ConfigRsp_File:
			// Received file preamble.
			receivedFile := &receivedBind9File{
				fileType:   r.File.FileType,
				sourcePath: r.File.SourcePath,
			}
			receivedFiles = append(receivedFiles, receivedFile)
		case *agentapi.ReceiveBind9ConfigRsp_Line:
			// Received file contents chunk. Append it to the last received file.
			require.NotEmpty(t, receivedFiles)
			receivedFiles[len(receivedFiles)-1].contents = append(receivedFiles[len(receivedFiles)-1].contents, r.Line)
		}
		return nil
	})
	// Specify multiple filters and file types.
	err := sa.ReceiveBind9Config(&agentapi.ReceiveBind9ConfigReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		Filter: &agentapi.ReceiveBind9ConfigFilter{
			FilterTypes: []agentapi.ReceiveBind9ConfigFilter_FilterType{},
		},
	}, mock)
	require.NoError(t, err)
	require.Len(t, receivedFiles, 2)
	require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, receivedFiles[0].fileType)
	require.Contains(t, receivedFiles[0].sourcePath, "named.conf")
	require.NotEmpty(t, receivedFiles[0].contents)
	require.Equal(t, agentapi.Bind9ConfigFileType_RNDC_KEY, receivedFiles[1].fileType)
	require.Contains(t, receivedFiles[1].sourcePath, "rndc.key")
	require.NotEmpty(t, receivedFiles[1].contents)

	// Make sure that the returned configuration is valid.
	bind9Config, err := bind9config.NewParser().Parse("", "", strings.NewReader(strings.Join(receivedFiles[0].contents, "\n")))
	require.NoError(t, err)
	require.NotNil(t, bind9Config)

	// Make sure that other configuration than views and zones is returned.
	controls := bind9Config.GetControls()
	require.NotNil(t, controls)

	// Make sure that views are returned.
	view := bind9Config.GetView("trusted")
	require.NotNil(t, view)

	// Make sure that zones are returned.
	zone := view.GetZone("example.com")
	require.NotNil(t, zone)

	// Make sure that the rndc.key file is valid.
	rndcKeyConfig, err := bind9config.NewParser().Parse("", "", strings.NewReader(strings.Join(receivedFiles[1].contents, "\n")))
	require.NoError(t, err)
	require.NotNil(t, rndcKeyConfig)

	rndcKey := rndcKeyConfig.GetKey("rndc-key")
	require.NotNil(t, rndcKey)
	algorithm, secret, err := rndcKey.GetAlgorithmSecret()
	require.NoError(t, err)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "UlJY3N2FdJ5cWUT6jQt/OPEnT9ap4b45Pzo1724yYw=", secret)
}

// Test getting BIND 9 configuration from a server with specifying
// multiple filters and file types.
func TestReceiveBind9ConfigMultipleFiltersAndFileTypes(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Create a BIND 9 daemon.
	accessPoints := []AccessPoint{{
		Type:     AccessPointControl,
		Address:  "127.0.0.1",
		Key:      "key",
		Port:     1234,
		Protocol: protocoltype.RNDC,
	}}
	daemon := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name:         daemonname.Bind9,
				AccessPoints: accessPoints,
			},
		},
		bind9Config:   parseDefaultBind9Config(t),
		rndcKeyConfig: parseDefaultBind9RNDCKeyConfig(t),
	}
	fam, _ := sa.Monitor.(*FakeMonitor)
	fam.Daemons = []Daemon{daemon}

	// Mock the server streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.ReceiveBind9ConfigRsp](ctrl)

	var receivedFiles []*receivedBind9File

	// Mock the function sending the actual data over the stream. In this mock
	// we collect the information about the received files and their contents.
	mock.EXPECT().Send(gomock.Any()).AnyTimes().DoAndReturn(func(rsp *agentapi.ReceiveBind9ConfigRsp) error {
		switch r := rsp.Response.(type) {
		case *agentapi.ReceiveBind9ConfigRsp_File:
			// Received file preamble.
			receivedFile := &receivedBind9File{
				fileType:   r.File.FileType,
				sourcePath: r.File.SourcePath,
			}
			receivedFiles = append(receivedFiles, receivedFile)
		case *agentapi.ReceiveBind9ConfigRsp_Line:
			// Received file contents chunk. Append it to the last received file.
			require.NotEmpty(t, receivedFiles)
			receivedFiles[len(receivedFiles)-1].contents = append(receivedFiles[len(receivedFiles)-1].contents, r.Line)
		}
		return nil
	})
	// Specify multiple filters and file types.
	err := sa.ReceiveBind9Config(&agentapi.ReceiveBind9ConfigReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		Filter: &agentapi.ReceiveBind9ConfigFilter{
			FilterTypes: []agentapi.ReceiveBind9ConfigFilter_FilterType{
				agentapi.ReceiveBind9ConfigFilter_CONFIG,
				agentapi.ReceiveBind9ConfigFilter_ZONE,
				agentapi.ReceiveBind9ConfigFilter_VIEW,
			},
		},
		FileSelector: &agentapi.ReceiveBind9ConfigFileSelector{
			FileTypes: []agentapi.Bind9ConfigFileType{
				agentapi.Bind9ConfigFileType_CONFIG,
				agentapi.Bind9ConfigFileType_RNDC_KEY,
			},
		},
	}, mock)
	require.NoError(t, err)
	require.Len(t, receivedFiles, 2)
	require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, receivedFiles[0].fileType)
	require.Contains(t, receivedFiles[0].sourcePath, "named.conf")
	require.NotEmpty(t, receivedFiles[0].contents)
	require.Equal(t, agentapi.Bind9ConfigFileType_RNDC_KEY, receivedFiles[1].fileType)
	require.Contains(t, receivedFiles[1].sourcePath, "rndc.key")
	require.NotEmpty(t, receivedFiles[1].contents)

	// Make sure that the returned configuration is valid.
	bind9Config, err := bind9config.NewParser().Parse("", "", strings.NewReader(strings.Join(receivedFiles[0].contents, "\n")))
	require.NoError(t, err)
	require.NotNil(t, bind9Config)

	// Make sure that other configuration than views and zones is returned.
	controls := bind9Config.GetControls()
	require.NotNil(t, controls)

	// Make sure that views are returned.
	view := bind9Config.GetView("trusted")
	require.NotNil(t, view)

	// Make sure that zones are returned.
	zone := view.GetZone("example.com")
	require.NotNil(t, zone)

	// Make sure that the rndc.key file is valid.
	rndcKeyConfig, err := bind9config.NewParser().Parse("", "", strings.NewReader(strings.Join(receivedFiles[1].contents, "\n")))
	require.NoError(t, err)
	require.NotNil(t, rndcKeyConfig)

	rndcKey := rndcKeyConfig.GetKey("rndc-key")
	require.NotNil(t, rndcKey)
	algorithm, secret, err := rndcKey.GetAlgorithmSecret()
	require.NoError(t, err)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "UlJY3N2FdJ5cWUT6jQt/OPEnT9ap4b45Pzo1724yYw=", secret)
}

// Test getting BIND 9 configuration from a server with specifying
// a single filter and file type.
func TestReceiveBind9ConfigSingleFilterAndFileType(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Create a BIND 9 daemon.
	accessPoints := []AccessPoint{{
		Type:     AccessPointControl,
		Address:  "127.0.0.1",
		Key:      "key",
		Port:     1234,
		Protocol: protocoltype.RNDC,
	}}
	daemon := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name:         daemonname.Bind9,
				AccessPoints: accessPoints,
			},
		},
		bind9Config:   parseDefaultBind9Config(t),
		rndcKeyConfig: parseDefaultBind9RNDCKeyConfig(t),
	}
	fam, _ := sa.Monitor.(*FakeMonitor)
	fam.Daemons = []Daemon{daemon}

	// Mock the server streaming server.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.ReceiveBind9ConfigRsp](ctrl)

	var receivedFiles []*receivedBind9File

	// Mock the function sending the actual data over the stream. In this mock
	// we collect the information about the received files and their contents.
	mock.EXPECT().Send(gomock.Any()).AnyTimes().DoAndReturn(func(rsp *agentapi.ReceiveBind9ConfigRsp) error {
		switch r := rsp.Response.(type) {
		case *agentapi.ReceiveBind9ConfigRsp_File:
			// Received file preamble.
			receivedFile := &receivedBind9File{
				fileType:   r.File.FileType,
				sourcePath: r.File.SourcePath,
			}
			receivedFiles = append(receivedFiles, receivedFile)
		case *agentapi.ReceiveBind9ConfigRsp_Line:
			// Received file contents chunk. Append it to the last received file.
			require.NotEmpty(t, receivedFiles)
			receivedFiles[len(receivedFiles)-1].contents = append(receivedFiles[len(receivedFiles)-1].contents, r.Line)
		}
		return nil
	})
	// Specify a single filter and file type.
	err := sa.ReceiveBind9Config(&agentapi.ReceiveBind9ConfigReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
		Filter: &agentapi.ReceiveBind9ConfigFilter{
			FilterTypes: []agentapi.ReceiveBind9ConfigFilter_FilterType{
				agentapi.ReceiveBind9ConfigFilter_CONFIG,
			},
		},
		FileSelector: &agentapi.ReceiveBind9ConfigFileSelector{
			FileTypes: []agentapi.Bind9ConfigFileType{
				agentapi.Bind9ConfigFileType_CONFIG,
			},
		},
	}, mock)
	require.NoError(t, err)
	require.Len(t, receivedFiles, 1)
	require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, receivedFiles[0].fileType)
	require.Contains(t, receivedFiles[0].sourcePath, "named.conf")
	require.NotEmpty(t, receivedFiles[0].contents)

	// Make sure that the returned configuration is valid.
	bind9Config, err := bind9config.NewParser().Parse("", "", strings.NewReader(strings.Join(receivedFiles[0].contents, "\n")))
	require.NoError(t, err)
	require.NotNil(t, bind9Config)

	// Make sure that other configuration than views and zones is returned.
	controls := bind9Config.GetControls()
	require.NotNil(t, controls)

	// Make sure that views are not returned.
	view := bind9Config.GetView("trusted")
	require.Nil(t, view)
}

// Test getting BIND 9 configuration from a non-existing server.
func TestReceiveBind9ConfigNoApp(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.ReceiveBind9ConfigRsp](ctrl)

	err := sa.ReceiveBind9Config(&agentapi.ReceiveBind9ConfigReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
	}, mock)
	require.Error(t, err)

	st := status.Convert(err)
	require.Equal(t, codes.FailedPrecondition, st.Code())
	require.Equal(t, "BIND 9 server 127.0.0.1:1234 not found", st.Message())
}

// Test getting BIND 9 configuration from a non-BIND9 DNS server.
func TestReceiveBind9ConfigNotBind9Daemon(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a non-BIND9 daemon.
	accessPoints := []AccessPoint{{
		Type:     AccessPointControl,
		Address:  "127.0.0.1",
		Key:      "key",
		Port:     1234,
		Protocol: protocoltype.RNDC,
	}}
	daemon := &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name:         daemonname.PDNS,
				AccessPoints: accessPoints,
			},
		},
	}
	fdm, _ := sa.Monitor.(*FakeMonitor)
	fdm.Daemons = []Daemon{daemon}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.ReceiveBind9ConfigRsp](ctrl)

	err := sa.ReceiveBind9Config(&agentapi.ReceiveBind9ConfigReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
	}, mock)
	require.Error(t, err)

	st := status.Convert(err)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Equal(t, "attempted to get BIND 9 configuration from daemon pdns instead of BIND 9", st.Message())
}

// Test getting BIND 9 configuration from a server for which the
// configuration was not found.
func TestReceiveBind9ConfigNoConfig(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Add a BIND9 daemon with no configuration.
	accessPoints := []AccessPoint{{
		Type:     AccessPointControl,
		Address:  "127.0.0.1",
		Key:      "key",
		Port:     1234,
		Protocol: protocoltype.RNDC,
	}}
	daemon := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name:         daemonname.Bind9,
				AccessPoints: accessPoints,
			},
		},
	}
	fam, _ := sa.Monitor.(*FakeMonitor)
	fam.Daemons = []Daemon{daemon}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockServerStreamingServer[agentapi.ReceiveBind9ConfigRsp](ctrl)

	err := sa.ReceiveBind9Config(&agentapi.ReceiveBind9ConfigReq{
		ControlAddress: "127.0.0.1",
		ControlPort:    1234,
	}, mock)
	require.Error(t, err)

	st := status.Convert(err)
	require.Equal(t, codes.NotFound, st.Code())
	require.Equal(t, "BIND 9 configuration not found for server 127.0.0.1:1234", st.Message())
}

// Test that that call to allow log method of the agent enables access to a log.
func TestAllowLog(t *testing.T) {
	// Arrange
	sa, _, teardown := setupAgentTest()
	defer teardown()

	// Act
	sa.allowLog("test/log/path")

	// Assert
	require.True(t, sa.logTailer.allowed("test/log/path"))
}
