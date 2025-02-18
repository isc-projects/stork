package agentcomm

import (
	"context"
	"io"
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
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/appdata/bind9stats"
	dbmodel "isc.org/stork/server/database/model"
	storktest "isc.org/stork/server/test/dbmodel"
	testutil "isc.org/stork/testutil"
)

// Stub error used in tests.
type testError struct{}

// Converts the error to string.
func (err *testError) Error() string {
	return "test error"
}

// makeAccessPoint is an utility to make single element app access point slice.
func makeAccessPoint(tp, address, key string, port int64) (ap []*agentapi.AccessPoint) {
	return append(ap, &agentapi.AccessPoint{
		Type:    tp,
		Address: address,
		Port:    port,
		Key:     key,
	})
}

// Setup function for the unit tests.
func setupGrpcliTestCase(ctrl *gomock.Controller) (*MockAgentClient, *connectedAgentsImpl) {
	mockAgentClient := NewMockAgentClient(ctrl)
	mockAgentsConnector := NewMockAgentConnector(ctrl)
	mockAgentsConnector.EXPECT().connect().AnyTimes().Return(nil)
	mockAgentsConnector.EXPECT().close().AnyTimes()
	mockAgentsConnector.EXPECT().createClient().AnyTimes().Return(mockAgentClient)

	settings := AgentsSettings{}
	fec := &storktest.FakeEventCenter{}
	agents := newConnectedAgentsImpl(&settings, fec, CACertPEM, ServerCertPEM, ServerKeyPEM)
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
	require.EqualValues(t, 1, agent.stats.GetTotalErrorCount())
}

// Check if GetState works.
func TestGetState(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// prepare expectations
	expVer := "123"
	rsp := agentapi.GetStateRsp{
		AgentVersion: expVer,
		Apps: []*agentapi.App{
			{
				Type:         AppTypeKea,
				AccessPoints: makeAccessPoint(AccessPointControl, "1.2.3.4", "", 1234),
			},
		},
	}
	mockAgentClient.EXPECT().
		GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	// call get state
	ctx := context.Background()
	state, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})
	require.NoError(t, err)
	require.Equal(t, expVer, state.AgentVersion)
	require.Equal(t, AppTypeKea, state.Apps[0].Type)
}

// Test error case for GetState.
func TestGetStateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	// prepare expectations
	mockAgentClient.EXPECT().
		GetState(gomock.Any(), gomock.Any(), newGZIPMatcher()).AnyTimes().
		Return(nil, pkgerrors.New("get state error"))

	// call get state
	ctx := context.Background()
	_, err := agents.GetState(ctx, &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	})
	require.Error(t, err)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.EqualValues(t, 1, agent.stats.GetTotalErrorCount())
}

// Test that a command can be successfully forwarded to Kea and the response
// can be parsed.
func TestForwardToKeaOverHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAgentClient, agents := setupGrpcliTestCase(ctrl)
	defer ctrl.Finish()

	data := []byte(`[
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
	]`)

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: data,
		}},
	}

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	command := keactrl.NewCommandBase(keactrl.CommandName("test-command"), keactrl.DHCPv4, keactrl.DHCPv6)
	actualResponse := keactrl.ResponseList{}
	dbApp := &dbmodel.App{
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
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []keactrl.SerializableCommand{command, command}, &actualResponse)
	require.NoError(t, err)
	require.NotNil(t, actualResponse)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.NoError(t, cmdsResult.CmdsErrors[0])

	responseList := actualResponse
	require.Len(t, responseList, 2)

	require.Equal(t, 1, responseList[0].Result)
	require.Equal(t, "operation failed", responseList[0].Text)
	require.Nil(t, responseList[0].Arguments)

	require.Equal(t, 0, responseList[1].Result)
	require.Equal(t, "operation succeeded", responseList[1].Text)
	require.NotNil(t, responseList[1].Arguments)
	require.Len(t, *responseList[1].Arguments, 1)
	require.Contains(t, *responseList[1].Arguments, "success")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalErrorCount())
	keaCommErrors := agent.stats.GetKeaCommErrorStats(0)
	require.Zero(t, keaCommErrors.GetErrorCount(KeaDaemonCA))
	require.EqualValues(t, 1, keaCommErrors.GetErrorCount(KeaDaemonDHCPv4))
	require.Zero(t, keaCommErrors.GetErrorCount(KeaDaemonDHCPv6))
	require.Zero(t, keaCommErrors.GetErrorCount(KeaDaemonD2))
}

// Test that two commands can be successfully forwarded to Kea and the response
// can be parsed.
func TestForwardToKeaOverHTTPWith2Cmds(t *testing.T) {
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
			Response: []byte(`[
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
        ]`),
		}, {
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: []byte(`[
            {
                "result": 1,
                "text": "operation failed"
            }
        ]`),
		}},
	}

	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	daemons := []keactrl.DaemonName{keactrl.DHCPv4, keactrl.DHCPv6}
	command1 := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemons...)
	command2 := keactrl.NewCommandBase(keactrl.CommandName("test-command"), daemons...)
	actualResponse1 := keactrl.ResponseList{}
	actualResponse2 := keactrl.ResponseList{}
	dbApp := &dbmodel.App{
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
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []keactrl.SerializableCommand{command1, command2}, &actualResponse1, &actualResponse2)
	require.NoError(t, err)
	require.NotNil(t, actualResponse1)
	require.NotNil(t, actualResponse2)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 2)
	require.NoError(t, cmdsResult.CmdsErrors[0])
	require.NoError(t, cmdsResult.CmdsErrors[1])

	responseList := actualResponse1
	require.Len(t, responseList, 2)

	require.Equal(t, 1, responseList[0].Result)
	require.Equal(t, "operation failed", responseList[0].Text)
	require.Nil(t, responseList[0].Arguments)

	require.Equal(t, 0, responseList[1].Result)
	require.Equal(t, "operation succeeded", responseList[1].Text)
	require.NotNil(t, responseList[1].Arguments)
	require.Len(t, *responseList[1].Arguments, 1)
	require.Contains(t, *responseList[1].Arguments, "success")

	responseList = actualResponse2
	require.Len(t, responseList, 1)

	require.Equal(t, 1, responseList[0].Result)
	require.Equal(t, "operation failed", responseList[0].Text)
	require.Nil(t, responseList[0].Arguments)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, 0, agent.stats.GetTotalErrorCount())
	keaCommErrors := agent.stats.GetKeaCommErrorStats(0)
	require.Zero(t, keaCommErrors.GetErrorCount(KeaDaemonCA))
	require.EqualValues(t, 2, keaCommErrors.GetErrorCount(KeaDaemonDHCPv4))
	require.Zero(t, keaCommErrors.GetErrorCount(KeaDaemonDHCPv6))
	require.Zero(t, keaCommErrors.GetErrorCount(KeaDaemonD2))
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
			Response: []byte(`[
            {
                "result": "a string"
            }
        ]`),
		}},
	}
	mockAgentClient.EXPECT().
		ForwardToKeaOverHTTP(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	command := keactrl.NewCommandBase(keactrl.CommandName("test-command"))
	actualResponse := keactrl.ResponseList{}
	dbApp := &dbmodel.App{
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
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []keactrl.SerializableCommand{command}, &actualResponse)
	require.NoError(t, err)
	require.NotNil(t, cmdsResult)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	// and now for our command we get an error
	require.Error(t, cmdsResult.CmdsErrors[0])

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalErrorCount())
	keaCommErrors := agent.stats.GetKeaCommErrorStats(0)
	require.EqualValues(t, 1, keaCommErrors.GetErrorCount(KeaDaemonCA))
	require.Zero(t, keaCommErrors.GetErrorCount(KeaDaemonDHCPv4))
	require.Zero(t, keaCommErrors.GetErrorCount(KeaDaemonDHCPv6))
	require.Zero(t, keaCommErrors.GetErrorCount(KeaDaemonD2))
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

	mockAgentClient.EXPECT().
		ForwardToNamedStats(gomock.Any(), gomock.Any(), newGZIPMatcher()).
		Return(&rsp, nil)

	ctx := context.Background()
	actualResponse := NamedStatsGetResponse{}
	err := agents.ForwardToNamedStats(ctx,
		dbmodel.App{
			ID:   1,
			Type: dbmodel.AppTypeBind9,
			Name: "named",
			Machine: &dbmodel.Machine{
				Address:   "127.0.0.1",
				AgentPort: 8080,
			},
		}, "localhost", 8000, "", &actualResponse)
	require.NoError(t, err)
	require.NotNil(t, actualResponse)
	require.Len(t, *actualResponse.Views, 1)
	require.Contains(t, *actualResponse.Views, "_default")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalErrorCount())
	bind9CommErrors := agent.stats.GetBind9CommErrorStats(1)
	require.Zero(t, bind9CommErrors.GetErrorCount(Bind9ChannelRNDC))
	require.Zero(t, bind9CommErrors.GetErrorCount(Bind9ChannelStats))
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
		dbmodel.App{
			ID:   1,
			Type: dbmodel.AppTypeBind9,
			Name: "named",
			Machine: &dbmodel.Machine{
				Address:   "127.0.0.1",
				AgentPort: 8080,
			},
		}, "localhost", 8000, "", &actualResponse)
	require.Error(t, err)

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalErrorCount())
	bind9CommErrors := agent.stats.GetBind9CommErrorStats(1)
	require.Zero(t, bind9CommErrors.GetErrorCount(Bind9ChannelRNDC))
	require.EqualValues(t, 1, bind9CommErrors.GetErrorCount(Bind9ChannelStats))
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
	dbApp := &dbmodel.App{
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

	out, err := agents.ForwardRndcCommand(ctx, dbApp, "test")
	require.NoError(t, err)
	require.Equal(t, out.Output, "all good")

	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.stats.GetTotalErrorCount())
	bind9CommErrors := agent.stats.GetBind9CommErrorStats(0)
	require.Zero(t, bind9CommErrors.GetErrorCount(Bind9ChannelRNDC))
	require.Zero(t, bind9CommErrors.GetErrorCount(Bind9ChannelStats))
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
	require.EqualValues(t, 1, agent.stats.GetTotalErrorCount())
}

// Check MakeAccessPoint.
func TestMakeAccessPoint(t *testing.T) {
	aps := MakeAccessPoint(dbmodel.AccessPointControl, "1.2.3.4", "abcd", 124)
	require.Len(t, aps, 1)
	ap := aps[0]
	require.EqualValues(t, dbmodel.AccessPointControl, ap.Type)
	require.EqualValues(t, "1.2.3.4", ap.Address)
	require.EqualValues(t, 124, ap.Port)
	require.EqualValues(t, "abcd", ap.Key)
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
	// Create an app without the access point.
	app := &dbmodel.App{
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
	// for this app.
	for zone, err := range agents.ReceiveZones(context.Background(), app, nil) {
		require.ErrorContains(t, err, "access point")
		require.Nil(t, zone)
	}
}

// Test that an error is returned when establishing connection fails.
func TestReceiveZonesConnectionError(t *testing.T) {
	// Create an app.
	app := &dbmodel.App{
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

	agents := newConnectedAgentsImpl(&AgentsSettings{}, &storktest.FakeEventCenter{}, CACertPEM, ServerCertPEM, ServerKeyPEM)
	agents.setConnectorFactory(func(string) agentConnector {
		return mockAgentConnector
	})

	mockStreamingClient := NewMockServerStreamingClient[agentapi.Zone](ctrl)
	// Make sure that the gRPC client is not used.
	mockStreamingClient.EXPECT().Recv().Times(0)
	mockAgentClient.EXPECT().ReceiveZones(gomock.Any(), gomock.Any()).Times(0)

	// The iterator should return an error during an attempt to connect.
	for zone, err := range agents.ReceiveZones(context.Background(), app, nil) {
		var testError *testError
		require.ErrorAs(t, err, &testError)
		require.Nil(t, zone)
	}
}

// Test that an error is returned when getting a gRPC stream fails.
func TestReceiveZonesGetStreamError(t *testing.T) {
	// Create an app.
	app := &dbmodel.App{
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
	for zone, err := range agents.ReceiveZones(context.Background(), app, nil) {
		require.ErrorContains(t, err, "test error")
		require.Nil(t, zone)
	}
}

// Test that it is explicitly communicated via an error that the zone
// inventory hasn't been initialized.
func TestReceiveZonesZoneInventoryNotInited(t *testing.T) {
	// Create an app.
	app := &dbmodel.App{
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
	for zone, err := range agents.ReceiveZones(context.Background(), app, nil) {
		var zoneInventoryNotInitedError *ZoneInventoryNotInitedError
		require.ErrorAs(t, err, &zoneInventoryNotInitedError)
		require.Nil(t, zone)
	}
}

// Test that it is explicitly communicated via an error that the zone
// inventory is busy.
func TestReceiveZonesZoneInventoryBusy(t *testing.T) {
	// Create an app.
	app := &dbmodel.App{
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
	for zone, err := range agents.ReceiveZones(context.Background(), app, nil) {
		var zoneInventoryBusyError *ZoneInventoryBusyError
		require.ErrorAs(t, err, &zoneInventoryBusyError)
		require.Nil(t, zone)
	}
}

// Test that other gRPC status errors are handled properly.
func TestReceiveZonesOtherStatusError(t *testing.T) {
	// Create an app.
	app := &dbmodel.App{
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
	for zone, err := range agents.ReceiveZones(context.Background(), app, nil) {
		require.ErrorContains(t, err, "internal server error")
		require.Nil(t, zone)
	}
}

// Test that an error is returned when getting a zone over the stream fails.
func TestReceiveZonesZoneInventoryReceiveZoneError(t *testing.T) {
	// Create an app.
	app := &dbmodel.App{
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
	for zone, err := range agents.ReceiveZones(context.Background(), app, nil) {
		var testError *testError
		require.ErrorAs(t, err, &testError)
		require.Nil(t, zone)
	}
}

// Test successful reception of all zones over the stream.
func TestReceiveZones(t *testing.T) {
	// Create an app.
	app := &dbmodel.App{
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
	for zone, err := range agents.ReceiveZones(context.Background(), app, nil) {
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
		require.Equal(t, time.Date(2025, 1, 5, 15, 19, 0, 0, time.UTC), zone.Loaded)
		require.Equal(t, "_default", zone.ViewName)
		require.EqualValues(t, 100, zone.TotalZoneCount)
	}
}
