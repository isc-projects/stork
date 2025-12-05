package agentcomm

import (
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"iter"
	"strings"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	agentapi "isc.org/stork/api"
	bind9config "isc.org/stork/daemoncfg/bind9"
	"isc.org/stork/daemoncfg/dnsconfig"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/daemondata/bind9stats"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	dbmodel "isc.org/stork/server/database/model"
	storktest "isc.org/stork/server/test/dbmodel"
	testutil "isc.org/stork/testutil"
)

//go:embed testdata/valid-zone.json
var validZoneData []byte

// Stub error used in tests.
type testError struct{}

// Converts the error to string.
func (err *testError) Error() string {
	return "test error"
}

// Setup function for the unit tests.
func setupGrpcliTestCase(ctrl *gomock.Controller) (*MockAgentClient, *connectedAgentsImpl) {
	caCertPEM, serverCertPEM, serverKeyPEM, _ := generateSelfSignedCerts()

	mockAgentClient := NewMockAgentClient(ctrl)
	mockAgentsConnector := NewMockAgentConnector(ctrl)
	mockAgentsConnector.EXPECT().connect().AnyTimes().Return(nil)
	mockAgentsConnector.EXPECT().close().AnyTimes()
	mockAgentsConnector.EXPECT().createClient().AnyTimes().Return(mockAgentClient)

	settings := AgentsSettings{}
	fec := &storktest.FakeEventCenter{}
	agents := newConnectedAgentsImpl(&settings, fec, caCertPEM, serverCertPEM, serverKeyPEM)
	agents.setConnectorFactory(func(string) agentConnector {
		return mockAgentsConnector
	})

	return mockAgentClient, agents
}

// Gomock-compatible matcher that asserts the GRPC call options. The assertion
// passes if the option to compress the content with the GZIP method is
// provided.
type gzipMatcher struct{}

// Interface check.
var _ gomock.Matcher = (*gzipMatcher)(nil)

// Constructs a new GZIP matcher instance.
func newGZIPMatcher() gomock.Matcher {
	return &gzipMatcher{}
}

// Checks if the provided argument contains the GRPC call option to compress
// the content.
func (*gzipMatcher) Matches(data any) bool {
	options, ok := data.([]grpc.CallOption)
	if !ok {
		// Argument is not an option list or the options are not provided.
		return false
	}
	// Search for compress option.
	for _, option := range options {
		compressorOption, ok := option.(grpc.CompressorCallOption)
		if !ok {
			// It isn't a compress option. Go to next.
			continue
		}
		// The compress option is found. Assert the compression method.
		return compressorOption.CompressorType == "gzip"
	}
	// The compress option is not found.
	return false
}

// Returns a string representation of the matcher for log purposes.
func (*gzipMatcher) String() string {
	return "gzip matcher"
}

//go:generate mockgen -package=agentcomm -destination=apimock_test.go -source=../../api/agent_grpc.pb.go isc.org/stork/api AgentClient
//go:generate mockgen -package=agentcomm -destination=agentcommmock_test.go -source=agentcomm.go -mock_names=agentConnector=MockAgentConnector agentConnector
//go:generate mockgen -package=agentcomm -destination=serverstreamingclientmock_test.go google.golang.org/grpc ServerStreamingClient

// Check if Ping works.
func TestPing(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// prepare expectations
	rsp := agentapi.PingRsp{}
	mockAgentClient.EXPECT().Ping(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	// call ping
	ctx := context.Background()
	err := agents.Ping(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})
	require.NoError(t, err)
}

// Test an error case for Ping.
func TestPingError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// prepare expectations
	mockAgentClient.EXPECT().Ping(gomock.Any(), gomock.Any()).AnyTimes().
		Return(nil, pkgerrors.Errorf("ping failed"))

	// call ping
	ctx := context.Background()
	err := agents.Ping(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})
	require.Error(t, err)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.EqualValues(t, 1, agent.stats.GetTotalAgentErrorCount())
}

// Check if GetState works.
func TestGetState(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// Prepare expectations.
	expectedVersion := "123"
	response := agentapi.GetStateRsp{
		AgentVersion: expectedVersion,
		Daemons: []*agentapi.Daemon{
			{
				Name: string(daemonname.DHCPv4),
				AccessPoints: []*agentapi.AccessPoint{{
					Type:     string(dbmodel.AccessPointControl),
					Address:  "1.2.3.4",
					Port:     1234,
					Protocol: string(protocoltype.HTTPS),
				}},
			},
			{
				Name: string(daemonname.Bind9),
				AccessPoints: []*agentapi.AccessPoint{{
					Type:     string(dbmodel.AccessPointControl),
					Address:  "1.2.3.5",
					Port:     4321,
					Protocol: string(protocoltype.RNDC),
				}},
			},
		},
	}
	mockAgentClient.EXPECT().
		GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&response, nil)

	ctx := context.Background()

	// Act
	state, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, expectedVersion, state.AgentVersion)
	require.Equal(t, daemonname.DHCPv4, state.Daemons[0].Name)
	require.Len(t, state.Daemons, 2)

	require.Equal(t, daemonname.DHCPv4, state.Daemons[0].Name)
	require.Len(t, state.Daemons[0].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPointControl, state.Daemons[0].AccessPoints[0].Type)
	require.Equal(t, "1.2.3.4", state.Daemons[0].AccessPoints[0].Address)
	require.EqualValues(t, 1234, state.Daemons[0].AccessPoints[0].Port)
	require.Equal(t, protocoltype.HTTPS, state.Daemons[0].AccessPoints[0].Protocol)

	require.Equal(t, daemonname.Bind9, state.Daemons[1].Name)
	require.Len(t, state.Daemons[1].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPointControl, state.Daemons[1].AccessPoints[0].Type)
	require.Equal(t, "1.2.3.5", state.Daemons[1].AccessPoints[0].Address)
	require.EqualValues(t, 4321, state.Daemons[1].AccessPoints[0].Port)
	require.Equal(t, protocoltype.RNDC, state.Daemons[1].AccessPoints[0].Protocol)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
}

// Test that the call to the agent is retried if the connection error occurs.
func TestGetStateRetryOnError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()
	ctx := context.Background()

	gomock.InOrder(
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(nil, pkgerrors.New("error")),
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(&agentapi.GetStateRsp{
				AgentVersion: "2.5.0",
				Daemons: []*agentapi.Daemon{
					{
						Name: string(daemonname.DHCPv4),
						AccessPoints: []*agentapi.AccessPoint{{
							Type:    string(dbmodel.AccessPointControl),
							Address: "1.2.3.4",
							Port:    1234,
						}},
					},
				},
			}, nil),
	)

	// Act
	state, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, "2.5.0", state.AgentVersion)
	require.Equal(t, daemonname.DHCPv4, state.Daemons[0].Name)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
}

// Test that the error is returned when trying to get state from an unknown agent.
func TestGetStateCommunicateWithUnknownAgent(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()
	ctx := context.Background()

	gomock.InOrder(
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(nil, pkgerrors.New("initial error")),
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(nil, pkgerrors.New("unknown agent")),
	)

	// Act
	state, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Assert
	require.ErrorContains(t, err, "unknown agent")
	require.Nil(t, state)
	// Check that the event is raised.
	events := agents.eventCenter.(*storktest.FakeEventCenter).Events
	require.Len(t, events, 1)
	event := events[0]
	require.Equal(t, "communication with Stork agent on <machine id=\"0\" address=\"127.0.0.1\" hostname=\"\"> to get state failed", event.Text)
	require.Equal(t, dbmodel.EvError, event.Level)
	require.Contains(t, event.Details, "unknown agent")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.EqualValues(t, 1, agent.stats.GetTotalAgentErrorCount())
}

// Test that the event is raised when the connection to an agent is restored.
func TestGetStateConnectionReset(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()
	ctx := context.Background()

	gomock.InOrder(
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(nil, pkgerrors.New("first error")),
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(nil, pkgerrors.New("second error")),
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(&agentapi.GetStateRsp{
				AgentVersion: "2.5.0",
				Daemons: []*agentapi.Daemon{
					{
						Name: string(daemonname.DHCPv4),
						AccessPoints: []*agentapi.AccessPoint{{
							Type:    string(dbmodel.AccessPointControl),
							Address: "1.2.3.4",
							Port:    1234,
						}},
					},
				},
			}, nil),
	)

	// Make a failed call to set the error state.
	_, _ = agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Act
	state, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, "2.5.0", state.AgentVersion)
	require.Equal(t, daemonname.DHCPv4, state.Daemons[0].Name)
	// Check that the event is raised.
	events := agents.eventCenter.(*storktest.FakeEventCenter).Events
	require.Len(t, events, 2)
	event := events[1]
	require.Equal(t, "communication with Stork agent on <machine id=\"0\" address=\"127.0.0.1\" hostname=\"\"> to get state succeeded", event.Text)
	require.Equal(t, dbmodel.EvWarning, event.Level)
	require.Empty(t, event.Details)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
}

// Test that the event is not raised when the connectivity error continues.
func TestGetStateConnectionErrorContinued(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()
	ctx := context.Background()

	gomock.InOrder(
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(nil, pkgerrors.New("first error")),
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(nil, pkgerrors.New("second error")),
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(nil, pkgerrors.New("third error")),
		mockAgentClient.EXPECT().
			GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
			Return(nil, pkgerrors.New("fourth error")),
	)

	// Make a failed call to set the error state.
	_, _ = agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Act
	state, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Assert
	require.ErrorContains(t, err, "fourth error")
	require.Nil(t, state)
	// Check that the event is not raised twice.
	events := agents.eventCenter.(*storktest.FakeEventCenter).Events
	require.Len(t, events, 1)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.EqualValues(t, 2, agent.stats.GetTotalAgentErrorCount())
}

// Test that the error is returned when the response to GetState is malformed.
func TestGetStateBadResponse(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockAgentClient.EXPECT().
		GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return((*agentapi.GetStateRsp)(nil), nil)

	ctx := context.Background()

	// Act
	state, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Assert
	require.ErrorContains(t, err, "wrong response to get state")
	require.Nil(t, state)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
}

// Test that the response from the legacy agents monitoring Kea is handled
// correctly.
func TestGetStateLegacyKeaResponse(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// Prepare expectations.
	response := agentapi.GetStateRsp{
		AgentVersion: "2.5.1",
		Daemons: []*agentapi.Daemon{
			{
				Name: "kea",
				AccessPoints: []*agentapi.AccessPoint{{
					Type:              string(dbmodel.AccessPointControl),
					Address:           "1.2.3.4",
					Port:              1234,
					UseSecureProtocol: true,
				}},
			},
		},
	}
	mockAgentClient.EXPECT().
		GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&response, nil)

	ctx := context.Background()

	// Act
	state, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, state.Daemons, 1)
	require.Equal(t, daemonname.CA, state.Daemons[0].Name)
	require.Len(t, state.Daemons[0].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPointControl, state.Daemons[0].AccessPoints[0].Type)
	require.Equal(t, "1.2.3.4", state.Daemons[0].AccessPoints[0].Address)
	require.EqualValues(t, 1234, state.Daemons[0].AccessPoints[0].Port)
	require.Equal(t, protocoltype.HTTPS, state.Daemons[0].AccessPoints[0].Protocol)
}

// Test that the response from the legacy agents monitoring BIND9 is handled
// correctly.
func TestGetStateLegacyBIND9Response(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// Prepare expectations.
	response := agentapi.GetStateRsp{
		AgentVersion: "2.5.1",
		Daemons: []*agentapi.Daemon{
			{
				Name: "bind9",
				AccessPoints: []*agentapi.AccessPoint{{
					Type:              string(dbmodel.AccessPointControl),
					Address:           "1.2.3.4",
					Port:              1234,
					UseSecureProtocol: false,
				}},
			},
		},
	}
	mockAgentClient.EXPECT().
		GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&response, nil)

	ctx := context.Background()

	// Act
	state, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, state.Daemons, 1)
	require.Equal(t, daemonname.Bind9, state.Daemons[0].Name)
	require.Len(t, state.Daemons[0].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPointControl, state.Daemons[0].AccessPoints[0].Type)
	require.Equal(t, "1.2.3.4", state.Daemons[0].AccessPoints[0].Address)
	require.EqualValues(t, 1234, state.Daemons[0].AccessPoints[0].Port)
	require.Equal(t, protocoltype.RNDC, state.Daemons[0].AccessPoints[0].Protocol)
}

// Test that a command can be successfully forwarded to Kea and the response
// can be parsed.
func TestForwardToKeaOverHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rawResponseV4 := []byte(`
		{
			"result": 1,
			"text": "operation failed"
		}
	`)

	rawResponseV6 := []byte(`{
			"result": 0,
			"text": "operation succeeded",
			"arguments": {
				"success": true
			}
		}
	`)

	rspV4 := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{
			{
				Status: &agentapi.Status{
					Code: 0,
				},
				Response: rawResponseV4,
			},
		},
	}

	rspV6 := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{
			{
				Status: &agentapi.Status{
					Code: 0,
				},
				Response: rawResponseV6,
			},
		},
	}

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rspV4, nil)

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rspV6, nil)

	ctx := context.Background()
	commandV4 := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.DHCPv4)
	commandV6 := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.DHCPv6)
	var responseV4 keactrl.Response
	var responseV6 keactrl.Response
	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.DHCPv4,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{commandV4}, &responseV4)
	require.NoError(t, err)
	require.NotNil(t, responseV4)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.Error(t, cmdsResult.CmdsErrors[0])

	require.Equal(t, keactrl.ResponseError, responseV4.Result)
	require.Equal(t, "operation failed", responseV4.Text)
	require.Nil(t, responseV4.Arguments)

	dbDaemon.Name = daemonname.DHCPv6
	cmdsResult, err = agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{commandV6}, &responseV6)
	require.NoError(t, err)
	require.NotNil(t, responseV6)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.NoError(t, cmdsResult.CmdsErrors[0])

	require.NotNil(t, responseV6)
	require.Equal(t, keactrl.ResponseSuccess, responseV6.Result)
	require.Equal(t, "operation succeeded", responseV6.Text)
	require.NotNil(t, responseV6.Arguments)
	argumentMap := map[string]any{}
	err = json.Unmarshal(responseV6.Arguments, &argumentMap)
	require.NoError(t, err)
	require.Len(t, argumentMap, 1)
	require.Contains(t, argumentMap, "success")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.CA))
	require.EqualValues(t, 1, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv6))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.D2))
}

// Test that a command can be successfully forwarded to Kea and the response
// can be parsed even if the agent is in a version prior to 2.3.2.
func TestForwardToKeaOverHTTPFromOldAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// The Kea responses are still wrapped in a JSON array for old agents.
	rawResponseV4 := []byte(`[
		{
			"result": 1,
			"text": "operation failed"
		}
	]`)

	rawResponseV6 := []byte(`[{
			"result": 0,
			"text": "operation succeeded",
			"arguments": {
				"success": true
			}
		}
	]`)

	rspV4 := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{
			{
				Status: &agentapi.Status{
					Code: 0,
				},
				Response: rawResponseV4,
			},
		},
	}

	rspV6 := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{
			{
				Status: &agentapi.Status{
					Code: 0,
				},
				Response: rawResponseV6,
			},
		},
	}

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rspV4, nil)

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rspV6, nil)

	ctx := context.Background()
	commandV4 := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.DHCPv4)
	commandV6 := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.DHCPv6)
	var responseV4 keactrl.Response
	var responseV6 keactrl.Response
	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.DHCPv4,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{commandV4}, &responseV4)
	require.NoError(t, err)
	require.NotNil(t, responseV4)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.Error(t, cmdsResult.CmdsErrors[0])

	require.Equal(t, keactrl.ResponseError, responseV4.Result)
	require.Equal(t, "operation failed", responseV4.Text)
	require.Nil(t, responseV4.Arguments)

	dbDaemon.Name = daemonname.DHCPv6
	cmdsResult, err = agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{commandV6}, &responseV6)
	require.NoError(t, err)
	require.NotNil(t, responseV6)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.NoError(t, cmdsResult.CmdsErrors[0])

	require.NotNil(t, responseV6)
	require.Equal(t, keactrl.ResponseSuccess, responseV6.Result)
	require.Equal(t, "operation succeeded", responseV6.Text)
	require.NotNil(t, responseV6.Arguments)
	argumentMap := map[string]any{}
	err = json.Unmarshal(responseV6.Arguments, &argumentMap)
	require.NoError(t, err)
	require.Len(t, argumentMap, 1)
	require.Contains(t, argumentMap, "success")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.CA))
	require.EqualValues(t, 1, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv6))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.D2))
}

// Test that two commands at once can be successfully forwarded to the same Kea
// daemon and the response can be parsed.
func TestForwardToKeaOverHTTPWith2Cmds(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	response := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{
			{
				Status: &agentapi.Status{
					Code: 0,
				},
				Response: []byte(`{
					"result": 1,
					"text": "operation failed"
				}`),
			},
			{
				Status: &agentapi.Status{
					Code: 0,
				},
				Response: []byte(`{
					"result": 0,
					"text": "operation succeeded",
					"arguments": {
						"success": true
					}
				}`),
			},
		},
	}

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&response, nil)

	ctx := context.Background()
	command1 := keactrl.NewCommandBase(keactrl.CommandName("first-command"), daemonname.DHCPv4)
	command2 := keactrl.NewCommandBase(keactrl.CommandName("second-command"), daemonname.DHCPv4)
	var actualResponse1 keactrl.Response
	var actualResponse2 keactrl.Response
	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.DHCPv4,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Act
	cmdsResult, err := agents.ForwardToKeaOverHTTP(
		ctx, dbDaemon,
		[]keactrl.SerializableCommand{command1, command2},
		&actualResponse1, &actualResponse2,
	)

	// Assert
	require.NoError(t, err)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 2)

	require.Error(t, cmdsResult.CmdsErrors[0])
	require.NotNil(t, actualResponse1)
	require.Equal(t, keactrl.ResponseError, actualResponse1.Result)
	require.Equal(t, "operation failed", actualResponse1.Text)

	require.NotNil(t, actualResponse2.Arguments)
	require.NotNil(t, actualResponse2)
	require.NoError(t, cmdsResult.CmdsErrors[1])
	require.Equal(t, keactrl.ResponseSuccess, actualResponse2.Result)
	require.Equal(t, "operation succeeded", actualResponse2.Text)
	require.NotNil(t, actualResponse2.Arguments)

	argumentMap := map[string]any{}
	err = json.Unmarshal(actualResponse2.Arguments, &argumentMap)
	require.NoError(t, err)
	require.Len(t, argumentMap, 1)
	require.Contains(t, argumentMap, "success")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.EqualValues(t, 1, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv6))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.D2))
}

// Test that the error is returned when the response to the forwarded Kea command
// is malformed.
func TestForwardToKeaOverHTTPInvalidResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: []byte(`{
                "result": "a string"
            }`),
		}},
	}
	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	command := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.DHCPv4)
	var actualResponse keactrl.Response
	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.DHCPv4,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{command}, &actualResponse)
	require.NoError(t, err)
	require.NotNil(t, cmdsResult)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	// and now for our command we get an error
	require.Error(t, cmdsResult.CmdsErrors[0])

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.CA))
	require.EqualValues(t, 1, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv6))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.D2))
}

// Test that the error is returned when the response to the forwarded Kea command
// contains a non-success HTTP status code.
func TestForwardToKeaOverHTTPBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code:    agentapi.Status_ERROR,
				Message: "received non-success status code 400 from Kea, with status text: 400 Bad Request; url: http://localhost:45634/",
			},
		}},
	}
	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	command := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.CA)
	var actualResponse keactrl.Response
	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.CA,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{command}, &actualResponse)
	require.NoError(t, err)
	require.NotNil(t, cmdsResult)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.Error(t, cmdsResult.CmdsErrors[0])
	require.Contains(t, cmdsResult.CmdsErrors[0].Error(), "received non-success status code 400 from Kea")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.EqualValues(t, 1, agent.stats.GetKeaStats().GetErrorCount(daemonname.CA))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv6))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.D2))
}

// Test that communication errors are counted when Stork sends commands the
// DHCP daemon through the Kea CA (prior Kea 3.0.0) and the daemon is
// unreachable.
func TestForwardToKeaOverHTTPUnreachableDaemonPrior3_0_0(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code: agentapi.Status_OK,
			},
			Response: []byte(`{
				"result": 1,
				"text": "unable to forward command to the dhcp4 service: No such file or directory. The server is likely to be offline"
			}`),
		}},
	}

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	command := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.DHCPv4)
	var actualResponse keactrl.Response

	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.DHCPv4,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Act
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{command}, &actualResponse)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cmdsResult)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.ErrorContains(t, cmdsResult.CmdsErrors[0], "unable to forward command to the dhcp4 service")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.Zero(t, 0, agent.stats.GetKeaStats().GetErrorCount(daemonname.CA))
	require.EqualValues(t, 1, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
}

// Test that communication errors are counted when Stork sends commands the
// DHCP daemon through the Kea CA (prior Kea 3.0.0) and CA is unreachable.
func TestForwardToKeaOverHTTPUnreachableCAPrior3_0_0(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code: agentapi.Status_ERROR,
				Message: "failed to forward commands to Kea: " +
					"failed to send command to Kea: " +
					"failed to send command to Kea: http://127.0.0.1:8000/: " +
					"problem sending POST to http://127.0.0.1:8000/: Post \"http://127.0.0.1:8000/\": " +
					"dial tcp 127.0.0.1:8000: connect: connection refused",
			},
		}},
	}

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	command := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.DHCPv4)
	var actualResponse keactrl.Response

	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.DHCPv4,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Act
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{command}, &actualResponse)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cmdsResult)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.ErrorContains(t, cmdsResult.CmdsErrors[0], "failed to forward commands to Kea")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	// The error occurred in communication between Stork agent and the Kea CA.
	// Unfortunately, we cannot determine which daemon failed when the
	// communication is through the CA, so we increment only the total error
	// count for target daemon.
	require.EqualValues(t, 1, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.CA))
}

// Test that communication errors are counted when Stork sends commands the
// DHCP daemon directly and the daemon is unreachable. It simulates the
// situation when the DHCP daemon is down but the Stork agent hasn't performed
// the daemon re-detection yet.
func TestForwardToKeaOverHTTPUnreachableDaemonPost3_0_0(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: agentapi.Status_OK,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code: agentapi.Status_ERROR,
				Message: "failed to forward commands to Kea: " +
					"failed to send command to Kea: " +
					"failed to connect to unix socket: /var/run/kea/kea4-ctrl-socket: " +
					"dial unix /var/run/kea/kea4-ctrl-socket: connect: no such file or directory",
			},
		}},
	}

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	command := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.DHCPv4)
	var actualResponse keactrl.Response

	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.DHCPv4,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Act
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{command}, &actualResponse)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cmdsResult)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.ErrorContains(t, cmdsResult.CmdsErrors[0], "failed to forward commands to Kea")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.CA))
	require.EqualValues(t, 1, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
}

// Test that communication errors are counted when Stork sends commands the
// DHCP daemon directly and the daemon is unreachable. It simulates the
// situation when the DHCP daemon is down and the Stork agent has performed
// the daemon re-detection already.
func TestForwardToKeaOverHTTPUnreachableDaemonPost3_0_0AfterRedetection(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: agentapi.Status_OK,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code:    agentapi.Status_ERROR,
				Message: "cannot find Kea daemon",
			},
		}},
	}

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	command := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemonname.DHCPv4)
	var actualResponse keactrl.Response

	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.DHCPv4,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Act
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbDaemon, []keactrl.SerializableCommand{command}, &actualResponse)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cmdsResult)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.ErrorContains(t, cmdsResult.CmdsErrors[0], "cannot find Kea daemon")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.Zero(t, agent.stats.GetKeaStats().GetErrorCount(daemonname.CA))
	require.EqualValues(t, 1, agent.stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
}

// Test that a statistics request can be successfully forwarded to named
// statistics-channel and the output can be parsed.
func TestForwardToNamedStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.ForwardToNamedStatsRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		NamedStatsResponse: &agentapi.NamedStatsResponse{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: `{
                             "json-stats-version": "1.2.",
                             "views": {
                                 "_default": {
                                     "resolver": {
                                         "cachestats": {
                                             "CacheHits": 11,
                                             "CacheMisses": 12
                                         }
                                     }
                                 }
                             }
                        }`,
		},
	}

	// Mock the gRPC call to the Stork agent. Ensure that the request is
	// correct by specifying a custom matcher.
	mockAgentClient.EXPECT().
		ForwardToNamedStats(gomock.Any(), gomock.Cond(func(req *agentapi.ForwardToNamedStatsReq) bool {
			//nolint:staticcheck
			return req.Url == "http://localhost:8000/" &&
				req.StatsAddress == "localhost" &&
				req.StatsPort == 8000 &&
				req.RequestType == agentapi.ForwardToNamedStatsReq_SERVER
		}), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	actualResponse := NamedStatsGetResponse{}
	err := agents.ForwardToNamedStats(ctx,
		dbmodel.Daemon{
			ID:   1,
			Name: daemonname.Bind9,
			Machine: &dbmodel.Machine{
				Address:   "127.0.0.1",
				AgentPort: 8080,
			},
			AccessPoints: []*dbmodel.AccessPoint{{
				Type:     dbmodel.AccessPointStatistics,
				Address:  "localhost",
				Port:     8000,
				Protocol: protocoltype.HTTP,
			}},
		}, agentapi.ForwardToNamedStatsReq_SERVER, &actualResponse)
	require.NoError(t, err)
	require.NotNil(t, actualResponse)
	require.Len(t, *actualResponse.Views, 1)
	require.Contains(t, *actualResponse.Views, "_default")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.Zero(t, agent.stats.GetBind9Stats().GetErrorCount(dbmodel.AccessPointControl))
	require.Zero(t, agent.stats.GetBind9Stats().GetErrorCount(dbmodel.AccessPointStatistics))
}

// Test that the error is returned when the response to the forwarded
// named statistics request is malformed.
func TestForwardToNamedStatsInvalidResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.ForwardToNamedStatsRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		NamedStatsResponse: &agentapi.NamedStatsResponse{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: `{
                          "views": "not the views you are looking for",
            }`,
		},
	}
	mockAgentClient.EXPECT().
		ForwardToNamedStats(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	actualResponse := NamedStatsGetResponse{}
	err := agents.ForwardToNamedStats(ctx,
		dbmodel.Daemon{
			ID:   1,
			Name: daemonname.Bind9,
			Machine: &dbmodel.Machine{
				Address:   "127.0.0.1",
				AgentPort: 8080,
			},
			AccessPoints: []*dbmodel.AccessPoint{{
				Type:    dbmodel.AccessPointStatistics,
				Address: "localhost",
				Port:    8000,
			}},
		}, agentapi.ForwardToNamedStatsReq_DEFAULT, &actualResponse)
	require.Error(t, err)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.Zero(t, agent.stats.GetBind9Stats().GetErrorCount(dbmodel.AccessPointControl))
	require.EqualValues(t, 1, agent.stats.GetBind9Stats().GetErrorCount(dbmodel.AccessPointStatistics))
}

// Test that a command can be successfully forwarded to rndc and the response
// can be parsed.
func TestForwardRndcCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.ForwardRndcCommandRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		RndcResponse: &agentapi.RndcResponse{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: "all good",
		},
	}

	mockAgentClient.EXPECT().ForwardRndcCommand(
		gomock.Any(), gomock.Any(), newGZIPMatcher(),
	).Return(&rsp, nil)

	ctx := context.Background()
	dbDaemon := &dbmodel.Daemon{
		Name: daemonname.Bind9,
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "127.0.0.1",
			Port:    953,
			Key:     "",
		}},
	}

	out, err := agents.ForwardRndcCommand(ctx, dbDaemon, "test")
	require.NoError(t, err)
	require.Equal(t, out.Output, "all good")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalAgentErrorCount())
	require.Zero(t, agent.stats.GetBind9Stats().GetErrorCount(dbmodel.AccessPointControl))
	require.Zero(t, agent.stats.GetBind9Stats().GetErrorCount(dbmodel.AccessPointStatistics))
}

// Test the gRPC call which fetches the tail of the specified text file.
func TestTailTextFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := agentapi.TailTextFileRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		Lines: []string{
			"Text returned by",
			"mock agent client",
		},
	}

	mockAgentClient.EXPECT().
		TailTextFile(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	tail, err := agents.TailTextFile(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}, "/tmp/log.txt", 2)
	require.NoError(t, err)
	require.Len(t, tail, 2)

	require.Equal(t, "Text returned by", tail[0])
	require.Equal(t, "mock agent client", tail[1])
}

// Test the error case for the gRPC call fetching the tail of the
// specified text file.
func TestTailTextFileError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockAgentClient.EXPECT().
		TailTextFile(gomock.Any(), gomock.Any(), newGZIPMatcher()).AnyTimes().
		Return(nil, pkgerrors.New("tail error"))

	ctx := context.Background()
	_, err := agents.TailTextFile(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}, "/tmp/log.txt", 2)
	require.Error(t, err)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.EqualValues(t, 1, agent.stats.GetTotalAgentErrorCount())
}

// Test getting first Kea error found in the KeaCmdsResult structure.
func TestKeaCmdsResultGetFirstError(t *testing.T) {
	var result *KeaCmdsResult

	// For nil result there is no error.
	require.NoError(t, result.GetFirstError())

	// Same for empty result.
	result = &KeaCmdsResult{}
	require.NoError(t, result.GetFirstError())

	// Set some errors at various levels.
	result = &KeaCmdsResult{
		Error: pkgerrors.New("first error"),
		CmdsErrors: []error{
			nil,
			pkgerrors.New("second error"),
			pkgerrors.New("third error"),
		},
	}
	// First error goes first.
	first := result.GetFirstError()
	require.ErrorContains(t, first, "first error")

	// Remove the first error. We should now get the second one.
	result.Error = nil
	first = result.GetFirstError()
	require.ErrorContains(t, first, "second error")

	// Repeat the test for next error.
	result.CmdsErrors[1] = nil
	first = result.GetFirstError()
	require.ErrorContains(t, first, "third error")
}

// Test that an error is returned when specified access point does not exist.
func TestReceiveZonesNonExistingAccessPoint(t *testing.T) {
	// Create an daemon without the access point.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockStreamingClient := NewMockServerStreamingClient[agentapi.Zone](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(0)

	// The iterator should return an error that there is no access point available
	// for this daemon.
	for zone, err := range agents.ReceiveZones(context.Background(), daemon, nil) {
		require.ErrorContains(t, err, "access point")
		require.Nil(t, zone)
	}
}

// Test that an error is returned when establishing connection fails.
func TestReceiveZonesConnectionError(t *testing.T) {
	caCertPEM, serverCertPEM, serverKeyPEM, err := generateSelfSignedCerts()
	require.NoError(t, err)

	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockAgentClient := NewMockAgentClient(ctrl)
	mockAgentConnector := NewMockAgentConnector(ctrl)
	mockAgentConnector.EXPECT().connect().AnyTimes().Return(&testError{})
	mockAgentConnector.EXPECT().close().AnyTimes()
	mockAgentConnector.EXPECT().createClient().AnyTimes().Return(mockAgentClient)

	agents := newConnectedAgentsImpl(&AgentsSettings{}, &storktest.FakeEventCenter{}, caCertPEM, serverCertPEM, serverKeyPEM)
	agents.setConnectorFactory(func(string) agentConnector {
		return mockAgentConnector
	})

	mockStreamingClient := NewMockServerStreamingClient[agentapi.Zone](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(0)

	// The iterator should return an error during an attempt to connect.
	for zone, err := range agents.ReceiveZones(context.Background(), daemon, nil) {
		var testError *testError
		require.ErrorAs(t, err, &testError)
		require.Nil(t, zone)
	}
}

// Test that an error is returned when getting a gRPC stream fails.
func TestReceiveZonesGetStreamError(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockStreamingClient := NewMockServerStreamingClient[agentapi.Zone](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(2).Return(nil, &testError{})

	// The iterator should return an error returned by ReceiveZones().
	for zone, err := range agents.ReceiveZones(context.Background(), daemon, nil) {
		require.ErrorContains(t, err, "test error")
		require.Nil(t, zone)
	}
}

// Test that it is explicitly communicated via an error that the zone
// inventory hasn't been initialized.
func TestReceiveZonesZoneInventoryNotInited(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockStreamingClient := NewMockServerStreamingClient[agentapi.Zone](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)

	// Create an error returned over gRPC indicating that the zone inventory
	// hasn't been initialized.
	st := status.New(codes.FailedPrecondition, "zone inventory not initialized")
	ds, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "ZONE_INVENTORY_NOT_INITED",
	})
	require.NoError(t, err)
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(2).Return(nil, ds.Err())

	// The iterator should return ZoneInventoryNotInitedError.
	for zone, err := range agents.ReceiveZones(context.Background(), daemon, nil) {
		var zoneInventoryNotInitedError *ZoneInventoryNotInitedError
		require.ErrorAs(t, err, &zoneInventoryNotInitedError)
		require.Nil(t, zone)
	}
}

// Test that it is explicitly communicated via an error that the zone
// inventory is busy.
func TestReceiveZonesZoneInventoryBusy(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockStreamingClient := NewMockServerStreamingClient[agentapi.Zone](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)

	// Create an error returned over gRPC indicating that the zone inventory
	// hasn't been initialized.
	st := status.New(codes.Unavailable, "zone inventory busy")
	ds, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "ZONE_INVENTORY_BUSY",
	})
	require.NoError(t, err)
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(2).Return(nil, ds.Err())

	// The iterator should return ZoneInventoryBusyError.
	for zone, err := range agents.ReceiveZones(context.Background(), daemon, nil) {
		var zoneInventoryBusyError *ZoneInventoryBusyError
		require.ErrorAs(t, err, &zoneInventoryBusyError)
		require.Nil(t, zone)
	}
}

// Test that other gRPC status errors are handled properly.
func TestReceiveZonesOtherStatusError(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockStreamingClient := NewMockServerStreamingClient[agentapi.Zone](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)

	// Create a generic status error returned over gRPC.
	st := status.New(codes.Internal, "internal server error")
	ds, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "SOME_OTHER_ERROR",
	})
	require.NoError(t, err)
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(2).Return(nil, ds.Err())

	// The iterator should return the original error without special handling.
	for zone, err := range agents.ReceiveZones(context.Background(), daemon, nil) {
		require.ErrorContains(t, err, "internal server error")
		require.Nil(t, zone)
	}
}

// Test that an error is returned when getting a zone over the stream fails.
func TestReceiveZonesZoneInventoryReceiveZoneError(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockStreamingClient := NewMockServerStreamingClient[agentapi.Zone](ctrl)
	mockStreamingClient.EXPECT().Recv().Times(1).Return(nil, &testError{})
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(1).Return(mockStreamingClient, nil)

	// The iterator should return an error returned by Recv().
	for zone, err := range agents.ReceiveZones(context.Background(), daemon, nil) {
		var testError *testError
		require.ErrorAs(t, err, &testError)
		require.Nil(t, zone)
	}
}

// Test successful reception of all zones over the stream.
func TestReceiveZones(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Generate a bunch of zones to be returned over the stream.
	generatedZones := testutil.GenerateRandomZones(100)

	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// Create the client mock.
	mockStreamingClient := NewMockServerStreamingClient[agentapi.Zone](ctrl)
	// The mock will return a sequence of zones.
	var mocks []any
	for _, zone := range generatedZones {
		zone := &agentapi.Zone{
			Name:           zone.Name,
			Class:          zone.Class,
			Serial:         zone.Serial,
			Type:           zone.Type,
			Loaded:         time.Date(2025, 1, 5, 15, 19, 0, 0, time.UTC).Unix(),
			Rpz:            true,
			View:           "_default",
			TotalZoneCount: 100,
		}
		mocks = append(mocks, mockStreamingClient.EXPECT().Recv().Return(zone, nil))
	}
	// The last item returned by the mock must be io.EOF indicating the
	// end of the stream.
	mocks = append(mocks, mockStreamingClient.EXPECT().Recv().Return(nil, io.EOF))

	// Make sure the zones are returned in order.
	gomock.InOrder(mocks...)

	// Return the mocked client when ReceiveZones() called.
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).AnyTimes().Return(mockStreamingClient, nil)

	// Collect the zones returned over the stream.
	var zones []*bind9stats.ExtendedZone
	for zone, err := range agents.ReceiveZones(context.Background(), daemon, nil) {
		require.NoError(t, err)
		require.NotNil(t, zone)
		zones = append(zones, zone)
	}
	// Make sure all zones have been returned.
	require.Len(t, zones, len(generatedZones))

	// Validate returned zones.
	for i, zone := range zones {
		require.Equal(t, generatedZones[i].Name, zone.Name())
		require.Equal(t, generatedZones[i].Class, zone.Class)
		require.Equal(t, generatedZones[i].Serial, zone.Serial)
		require.Equal(t, generatedZones[i].Type, zone.Type)
		require.True(t, zone.RPZ)
		require.Equal(t, time.Date(2025, 1, 5, 15, 19, 0, 0, time.UTC), zone.Loaded)
		require.Equal(t, "_default", zone.ViewName)
		require.EqualValues(t, 100, zone.TotalZoneCount)
	}
}

// Test that an error is returned when specified access point does not exist.
func TestReceiveZoneRRsNonExistingAccessPoint(t *testing.T) {
	// Create an daemon without the access point.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveZoneRRsRsp](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(0)

	// The iterator should return an error that there is no access point available
	// for this daemon.
	for rrs, err := range agents.ReceiveZoneRRs(context.Background(), daemon, "example.com", "_default") {
		require.ErrorContains(t, err, "access point")
		require.Nil(t, rrs)
	}
}

// Test that an error is returned when establishing connection fails.
func TestReceiveZoneRRsConnectionError(t *testing.T) {
	caCertPEM, serverCertPEM, serverKeyPEM, err := generateSelfSignedCerts()
	require.NoError(t, err)

	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockAgentClient := NewMockAgentClient(ctrl)
	mockAgentConnector := NewMockAgentConnector(ctrl)
	mockAgentConnector.EXPECT().connect().AnyTimes().Return(&testError{})
	mockAgentConnector.EXPECT().close().AnyTimes()
	mockAgentConnector.EXPECT().createClient().AnyTimes().Return(mockAgentClient)

	agents := newConnectedAgentsImpl(&AgentsSettings{}, &storktest.FakeEventCenter{}, caCertPEM, serverCertPEM, serverKeyPEM)
	agents.setConnectorFactory(func(string) agentConnector {
		return mockAgentConnector
	})

	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveZoneRRsRsp](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(0)

	// The iterator should return an error during an attempt to connect.
	for rrs, err := range agents.ReceiveZoneRRs(context.Background(), daemon, "example.com", "_default") {
		var testError *testError
		require.ErrorAs(t, err, &testError)
		require.Nil(t, rrs)
	}
}

// Test that an error is returned when getting a gRPC stream fails.
func TestReceiveZoneRRsGetStreamError(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveZoneRRsRsp](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)
	mockAgentClient.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Any()).Times(2).Return(nil, &testError{})

	// The iterator should return an error returned by ReceiveZones().
	for rrs, err := range agents.ReceiveZoneRRs(context.Background(), daemon, "example.com", "_default") {
		require.ErrorContains(t, err, "test error")
		require.Nil(t, rrs)
	}
}

// Test that it is explicitly communicated via an error that the zone
// inventory hasn't been initialized.
func TestReceiveZoneRRsZoneInventoryNotInited(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// Create an error returned over gRPC indicating that the zone inventory
	// hasn't been initialized.
	st := status.New(codes.FailedPrecondition, "zone inventory not initialized")
	ds, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "ZONE_INVENTORY_NOT_INITED",
	})
	require.NoError(t, err)
	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveZoneRRsRsp](ctrl)
	mockStreamingClient.EXPECT().Recv().Times(1).Return(nil, ds.Err())
	mockAgentClient.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Any()).Times(1).Return(mockStreamingClient, nil)

	// The iterator should return ZoneInventoryNotInitedError.
	for rrs, err := range agents.ReceiveZoneRRs(context.Background(), daemon, "example.com", "_default") {
		var zoneInventoryNotInitedError *ZoneInventoryNotInitedError
		require.ErrorAs(t, err, &zoneInventoryNotInitedError)
		require.Nil(t, rrs)
	}
}

// Test that it is explicitly communicated via an error that the zone
// inventory is busy.
func TestReceiveZoneRRsZoneInventoryBusy(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// Create an error returned over gRPC indicating that the zone inventory
	// hasn't been initialized.
	st := status.New(codes.Unavailable, "zone inventory busy")
	ds, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "ZONE_INVENTORY_BUSY",
	})
	require.NoError(t, err)
	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveZoneRRsRsp](ctrl)
	mockStreamingClient.EXPECT().Recv().Times(1).Return(nil, ds.Err())
	mockAgentClient.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Any()).Times(1).Return(mockStreamingClient, nil)

	// The iterator should return ZoneInventoryBusyError.
	for rrs, err := range agents.ReceiveZoneRRs(context.Background(), daemon, "example.com", "_default") {
		var zoneInventoryBusyError *ZoneInventoryBusyError
		require.ErrorAs(t, err, &zoneInventoryBusyError)
		require.Nil(t, rrs)
	}
}

// Test that other gRPC status errors are handled properly.
func TestReceiveZoneRRsOtherStatusError(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// Create a generic status error returned over gRPC.
	st := status.New(codes.Internal, "internal server error")
	ds, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "SOME_OTHER_ERROR",
	})
	require.NoError(t, err)
	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveZoneRRsRsp](ctrl)
	mockStreamingClient.EXPECT().Recv().Times(1).Return(nil, ds.Err())
	mockAgentClient.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Any()).Times(1).Return(mockStreamingClient, nil)

	// The iterator should return the original error without special handling.
	for rrs, err := range agents.ReceiveZoneRRs(context.Background(), daemon, "example.com", "_default") {
		require.ErrorContains(t, err, "internal server error")
		require.Nil(t, rrs)
	}
}

// Test that an error is returned when getting zone contents over the stream fails.
func TestReceiveZoneRRsZoneInventoryReceiveError(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveZoneRRsRsp](ctrl)
	mockStreamingClient.EXPECT().Recv().Times(1).Return(nil, &testError{})
	mockAgentClient.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Any()).Times(1).Return(mockStreamingClient, nil)

	// The iterator should return an error returned by Recv().
	for rrs, err := range agents.ReceiveZoneRRs(context.Background(), daemon, "example.com", "_default") {
		var testError *testError
		require.ErrorAs(t, err, &testError)
		require.Nil(t, rrs)
	}
}

// Test successful reception of zone contents over the stream.
func TestReceiveZoneRRs(t *testing.T) {
	// Create an daemon.
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// Get the example zone contents from the file.
	var rrs []string
	err := json.Unmarshal(validZoneData, &rrs)
	require.NoError(t, err)

	// Create the client mock.
	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveZoneRRsRsp](ctrl)
	// The mock will return a sequence of RRs.
	var mocks []any
	for _, rr := range rrs {
		rr := &agentapi.ReceiveZoneRRsRsp{
			Rrs: []string{rr},
		}
		mocks = append(mocks, mockStreamingClient.EXPECT().Recv().Return(rr, nil))
	}
	// The last item returned by the mock must be io.EOF indicating the
	// end of the stream.
	mocks = append(mocks, mockStreamingClient.EXPECT().Recv().Return(nil, io.EOF))

	// Make sure the RRs are returned in order.
	gomock.InOrder(mocks...)

	// Return the mocked client when ReceiveZoneRRs() called.
	mockAgentClient.EXPECT().ReceiveZoneRRs(gomock.Any(), gomock.Any()).AnyTimes().Return(mockStreamingClient, nil)

	// Collect the RRs returned over the stream.
	var contents []*dnsconfig.RR
	for receivedRRs, err := range agents.ReceiveZoneRRs(context.Background(), daemon, "zone1", "_default") {
		require.NoError(t, err)
		require.NotNil(t, receivedRRs)
		contents = append(contents, receivedRRs...)
	}
	// Make sure all RRs have been returned.
	require.Len(t, contents, len(rrs))

	// Validate returned RRs.
	for i, rr := range contents {
		// Replace tabs with spaces in the original RR.
		original := strings.Join(strings.Fields(rrs[i]), " ")
		require.Equal(t, original, rr.GetString())
	}
}

// Test successfully getting the PowerDNS server information.
func TestGetPowerDNSServerInfo(t *testing.T) {
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	rsp := &agentapi.GetPowerDNSServerInfoRsp{
		Type:             "PowerDNS",
		Id:               "127.0.0.1",
		DaemonType:       "pdns",
		Version:          "4.7.0",
		Url:              "http://127.0.0.1:8081",
		ConfigURL:        "http://127.0.0.1:8081/config",
		ZonesURL:         "http://127.0.0.1:8081/zones",
		AutoprimariesURL: "http://127.0.0.1:8081/autoprimaries",
		Uptime:           1234,
	}
	mockAgentClient.EXPECT().GetPowerDNSServerInfo(gomock.Any(), gomock.Any(), newGZIPMatcher()).AnyTimes().Return(rsp, nil)

	serverInfo, err := agents.GetPowerDNSServerInfo(context.Background(), daemon)
	require.NoError(t, err)
	require.NotNil(t, serverInfo)
	require.Equal(t, "PowerDNS", serverInfo.Type)
	require.Equal(t, "127.0.0.1", serverInfo.ID)
	require.Equal(t, "pdns", serverInfo.DaemonType)
	require.Equal(t, "4.7.0", serverInfo.Version)
	require.Equal(t, "http://127.0.0.1:8081", serverInfo.URL)
	require.Equal(t, "http://127.0.0.1:8081/config", serverInfo.ConfigURL)
	require.Equal(t, "http://127.0.0.1:8081/zones", serverInfo.ZonesURL)
	require.Equal(t, "http://127.0.0.1:8081/autoprimaries", serverInfo.AutoprimariesURL)
	require.EqualValues(t, 1234, serverInfo.Uptime)
}

// Test that an error is returned when trying to get the PowerDNS server
// when there is no access point.
func TestGetPowerDNSServerInfoNoAccessPoint(t *testing.T) {
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
	}

	ctrl := gomock.NewController(t)
	_, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	serverInfo, err := agents.GetPowerDNSServerInfo(context.Background(), daemon)
	require.Error(t, err)
	require.ErrorContains(t, err, "no access point")
	require.Nil(t, serverInfo)
}

// Test that an error is returned when trying to get the PowerDNS server
// when the gRPC call fails.
func TestGetPowerDNSServerInfoErrorResponse(t *testing.T) {
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	mockAgentClient.EXPECT().GetPowerDNSServerInfo(gomock.Any(), gomock.Any(), newGZIPMatcher()).AnyTimes().Return(nil, &testError{})

	serverInfo, err := agents.GetPowerDNSServerInfo(context.Background(), daemon)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, serverInfo)
}

// Test successfully receiving BIND 9 configuration over the stream for
// a single file type.
func TestReceiveBind9FormattedConfigOneFile(t *testing.T) {
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Mock the gRPC client receiving a single configuration file followed by EOF.
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()
	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveBind9ConfigRsp](ctrl)

	// The first chunk is the configuration file preamble.
	mockStreamingClient.EXPECT().Recv().DoAndReturn(func() (*agentapi.ReceiveBind9ConfigRsp, error) {
		return &agentapi.ReceiveBind9ConfigRsp{
			Response: &agentapi.ReceiveBind9ConfigRsp_File{
				File: &agentapi.ReceiveBind9ConfigFile{
					FileType: agentapi.Bind9ConfigFileType_CONFIG,
				},
			},
		}, nil
	})
	// The second chunk is the configuration file contents.
	mockStreamingClient.EXPECT().Recv().DoAndReturn(func() (*agentapi.ReceiveBind9ConfigRsp, error) {
		return &agentapi.ReceiveBind9ConfigRsp{
			Response: &agentapi.ReceiveBind9ConfigRsp_Line{
				Line: "options { ... };",
			},
		}, nil
	})

	// Create a unique context (with a value), so we can later verify that
	// this particular context is used for the request.
	type requestCtxKey string
	requestCtx := context.WithValue(context.Background(), requestCtxKey("key"), "requestCtx")

	// The last chunk is EOF.
	mockStreamingClient.EXPECT().Recv().Return(nil, io.EOF)
	// Return the mocked client when ReceiveBind9Config() called.
	mockAgentClient.EXPECT().ReceiveBind9Config(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, req *agentapi.ReceiveBind9ConfigReq, opts ...any) (grpc.ServerStreamingClient[agentapi.ReceiveBind9ConfigRsp], error) {
		// Make sure that the request is properly populated.
		require.NotNil(t, req)
		require.Nil(t, req.Filter)
		require.Nil(t, req.FileSelector)
		// Make sure that the correct context was passed to the request.
		require.Equal(t, requestCtx, ctx)
		return mockStreamingClient, nil
	})

	// Collect the chunks from the stream.
	next, cancel := iter.Pull2(agents.ReceiveBind9FormattedConfig(requestCtx, daemon, nil, nil))
	defer cancel()

	// Configuration file preamble.
	rsp, err, ok := next()
	require.True(t, ok)
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Response)
	require.IsType(t, &agentapi.ReceiveBind9ConfigRsp_File{}, rsp.Response)
	file := rsp.Response.(*agentapi.ReceiveBind9ConfigRsp_File).File
	require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, file.FileType)

	// Configuration file contents.
	rsp, err, ok = next()
	require.True(t, ok)
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Response)
	require.IsType(t, &agentapi.ReceiveBind9ConfigRsp_Line{}, rsp.Response)
	line := rsp.Response.(*agentapi.ReceiveBind9ConfigRsp_Line).Line
	require.Equal(t, "options { ... };", line)

	// The stream should be exhausted.
	rsp, err, ok = next()
	require.False(t, ok)
	require.NoError(t, err)
	require.Nil(t, rsp)
}

// Test successfully receiving BIND 9 configuration over the stream for
// two file types.
func TestGetBind9FormattedConfigTwoFiles(t *testing.T) {
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Mock the gRPC client receiving two configuration files followed by EOF.
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()
	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveBind9ConfigRsp](ctrl)

	// The first chunk is the configuration file preamble.
	mockStreamingClient.EXPECT().Recv().DoAndReturn(func() (*agentapi.ReceiveBind9ConfigRsp, error) {
		return &agentapi.ReceiveBind9ConfigRsp{
			Response: &agentapi.ReceiveBind9ConfigRsp_File{
				File: &agentapi.ReceiveBind9ConfigFile{
					FileType: agentapi.Bind9ConfigFileType_CONFIG,
				},
			},
		}, nil
	})
	// The second chunk is the configuration file contents.
	mockStreamingClient.EXPECT().Recv().DoAndReturn(func() (*agentapi.ReceiveBind9ConfigRsp, error) {
		return &agentapi.ReceiveBind9ConfigRsp{
			Response: &agentapi.ReceiveBind9ConfigRsp_Line{
				Line: "options { ... };",
			},
		}, nil
	})
	// The third chunk is the rndc key file preamble.
	mockStreamingClient.EXPECT().Recv().DoAndReturn(func() (*agentapi.ReceiveBind9ConfigRsp, error) {
		return &agentapi.ReceiveBind9ConfigRsp{
			Response: &agentapi.ReceiveBind9ConfigRsp_File{
				File: &agentapi.ReceiveBind9ConfigFile{
					FileType: agentapi.Bind9ConfigFileType_RNDC_KEY,
				},
			},
		}, nil
	})
	// The fourth chunk is the rndc key file contents.
	mockStreamingClient.EXPECT().Recv().DoAndReturn(func() (*agentapi.ReceiveBind9ConfigRsp, error) {
		return &agentapi.ReceiveBind9ConfigRsp{
			Response: &agentapi.ReceiveBind9ConfigRsp_Line{
				Line: "rndc-key;",
			},
		}, nil
	})
	// The last chunk is EOF.
	mockStreamingClient.EXPECT().Recv().Return(nil, io.EOF)

	// Return the mocked client when ReceiveBind9Config() called.
	mockAgentClient.EXPECT().ReceiveBind9Config(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, req *agentapi.ReceiveBind9ConfigReq, opts ...any) (grpc.ServerStreamingClient[agentapi.ReceiveBind9ConfigRsp], error) {
		// Make sure that the request is properly populated.
		require.NotNil(t, req)
		require.Nil(t, req.Filter)
		require.NotNil(t, req.FileSelector)
		require.Len(t, req.FileSelector.FileTypes, 2)
		require.Contains(t, req.FileSelector.FileTypes, agentapi.Bind9ConfigFileType_CONFIG)
		require.Contains(t, req.FileSelector.FileTypes, agentapi.Bind9ConfigFileType_RNDC_KEY)
		return mockStreamingClient, nil
	})

	// Collect the chunks from the stream.
	next, cancel := iter.Pull2(agents.ReceiveBind9FormattedConfig(context.Background(), daemon, bind9config.NewFileTypeSelector(bind9config.FileTypeConfig, bind9config.FileTypeRndcKey), nil))
	defer cancel()

	// Configuration file preamble.
	rsp, err, ok := next()
	require.True(t, ok)
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Response)
	require.IsType(t, &agentapi.ReceiveBind9ConfigRsp_File{}, rsp.Response)
	file := rsp.Response.(*agentapi.ReceiveBind9ConfigRsp_File).File
	require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, file.FileType)

	// Configuration file contents.
	rsp, err, ok = next()
	require.True(t, ok)
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Response)
	require.IsType(t, &agentapi.ReceiveBind9ConfigRsp_Line{}, rsp.Response)
	line := rsp.Response.(*agentapi.ReceiveBind9ConfigRsp_Line).Line
	require.Equal(t, "options { ... };", line)

	// RNDC key file preamble.
	rsp, err, ok = next()
	require.True(t, ok)
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Response)
	require.IsType(t, &agentapi.ReceiveBind9ConfigRsp_File{}, rsp.Response)
	file = rsp.Response.(*agentapi.ReceiveBind9ConfigRsp_File).File
	require.Equal(t, agentapi.Bind9ConfigFileType_RNDC_KEY, file.FileType)

	// RNDC key file contents.
	rsp, err, ok = next()
	require.True(t, ok)
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Response)
	require.IsType(t, &agentapi.ReceiveBind9ConfigRsp_Line{}, rsp.Response)
	line = rsp.Response.(*agentapi.ReceiveBind9ConfigRsp_Line).Line
	require.Equal(t, "rndc-key;", line)

	// The stream should be exhausted.
	rsp, err, ok = next()
	require.False(t, ok)
	require.NoError(t, err)
	require.Nil(t, rsp)
}

// Test successfully receiving BIND 9 configuration over the stream with
// filtering.
func TestGetBind9FormattedConfigFiltering(t *testing.T) {
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Mock the gRPC client receiving a single configuration file with filtering.
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()
	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveBind9ConfigRsp](ctrl)

	// The first chunk is the configuration file preamble.
	mockStreamingClient.EXPECT().Recv().DoAndReturn(func() (*agentapi.ReceiveBind9ConfigRsp, error) {
		return &agentapi.ReceiveBind9ConfigRsp{
			Response: &agentapi.ReceiveBind9ConfigRsp_File{
				File: &agentapi.ReceiveBind9ConfigFile{
					FileType: agentapi.Bind9ConfigFileType_CONFIG,
				},
			},
		}, nil
	})
	// The last chunk is EOF. The file is empty.
	mockStreamingClient.EXPECT().Recv().Return(nil, io.EOF)

	mockAgentClient.EXPECT().ReceiveBind9Config(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, req *agentapi.ReceiveBind9ConfigReq, opts ...any) (grpc.ServerStreamingClient[agentapi.ReceiveBind9ConfigRsp], error) {
		require.NotNil(t, req)
		require.NotNil(t, req.FileSelector)
		require.Len(t, req.FileSelector.FileTypes, 1)
		require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, req.FileSelector.FileTypes[0])
		require.NotNil(t, req.Filter)
		require.Len(t, req.Filter.FilterTypes, 1)
		require.Equal(t, agentapi.ReceiveBind9ConfigFilter_CONFIG, req.Filter.FilterTypes[0])
		return mockStreamingClient, nil
	})

	// Collect the chunks from the stream.
	next, cancel := iter.Pull2(agents.ReceiveBind9FormattedConfig(context.Background(), daemon, bind9config.NewFileTypeSelector(bind9config.FileTypeConfig), bind9config.NewFilter(bind9config.FilterTypeConfig)))
	defer cancel()

	// Configuration file preamble.
	rsp, err, ok := next()
	require.True(t, ok)
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.NotNil(t, rsp.Response)
	require.IsType(t, &agentapi.ReceiveBind9ConfigRsp_File{}, rsp.Response)
	file := rsp.Response.(*agentapi.ReceiveBind9ConfigRsp_File).File
	require.Equal(t, agentapi.Bind9ConfigFileType_CONFIG, file.FileType)

	// The stream should be exhausted.
	rsp, err, ok = next()
	require.False(t, ok)
	require.NoError(t, err)
	require.Nil(t, rsp)
}

// Test that an error is returned when trying to open the stream to receive
// the BIND 9 configuration from the agent.
func TestReceiveBind9FormattedConfigOpenStreamError(t *testing.T) {
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Mock the gRPC client returning an error.
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// Return an error when trying to open the stream.
	mockAgentClient.EXPECT().ReceiveBind9Config(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, &testError{})

	// Collect the chunks from the stream.
	next, cancel := iter.Pull2(agents.ReceiveBind9FormattedConfig(context.Background(), daemon, nil, nil))
	defer cancel()

	// The error should be propagated.
	rsp, err, ok := next()
	require.True(t, ok)
	require.ErrorContains(t, err, "failed to open stream to receive BIND 9 configuration from the agent: test error")
	require.Nil(t, rsp)
}

// Test that an error is returned when trying to receive the BIND 9 formatted config
// when the gRPC call fails.
func TestReceiveBind9FormattedConfigErrorResponse(t *testing.T) {
	daemon := &dbmodel.Daemon{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}

	// Mock the gRPC client returning an error.
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()
	mockStreamingClient := NewMockServerStreamingClient[agentapi.ReceiveBind9ConfigRsp](ctrl)
	mockStreamingClient.EXPECT().Recv().Return(nil, &testError{})

	// Return the mocked client when ReceiveBind9Config() called.
	mockAgentClient.EXPECT().ReceiveBind9Config(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(mockStreamingClient, nil)

	// Collect the chunks from the stream.
	next, cancel := iter.Pull2(agents.ReceiveBind9FormattedConfig(context.Background(), daemon, nil, nil))
	defer cancel()

	// The error should be propagated.
	rsp, err, ok := next()
	require.True(t, ok)
	require.ErrorContains(t, err, "failed to receive BIND 9 configuration from the agent: test error")
	require.Nil(t, rsp)
}

// Test getting the name of the daemon.
func TestDaemonGetName(t *testing.T) {
	daemon := Daemon{
		Name: daemonname.DHCPv4,
	}
	require.Equal(t, daemonname.DHCPv4, daemon.GetName())
}

// Test getting the access point of the daemon.
func TestDaemonGetAccessPoint(t *testing.T) {
	accessPoint := dbmodel.AccessPoint{
		Type:    dbmodel.AccessPointControl,
		Address: "127.0.0.1",
		Port:    8080,
	}
	daemon := Daemon{
		Name:         daemonname.DHCPv4,
		AccessPoints: []dbmodel.AccessPoint{accessPoint},
	}

	// Test getting existing access point.
	dbAccessPoint, err := daemon.GetAccessPoint(dbmodel.AccessPointControl)
	require.NoError(t, err)
	require.Equal(t, &accessPoint, dbAccessPoint)

	// Test getting non-existing access point.
	dbAccessPoint, err = daemon.GetAccessPoint(dbmodel.AccessPointStatistics)
	require.Error(t, err)
	require.Nil(t, dbAccessPoint)
	require.Contains(t, err.Error(), "no statistics access point for daemon dhcp4")
}

// Test getting the machine tag of the daemon.
func TestDaemonGetMachineTag(t *testing.T) {
	machine := &dbmodel.Machine{
		Address: "192.0.2.1",
	}
	daemon := Daemon{
		Machine: machine,
	}
	require.Equal(t, machine, daemon.GetMachineTag())
}
