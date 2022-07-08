package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
	agentapi "isc.org/stork/api"
)

type TestCalloutOnBeforeForwardToKeaOverHTTP struct {
	t          *testing.T
	callCounts uint64
}

func newTestCalloutOnBeforeForwardToKeaOverHTTP(t *testing.T) *TestCalloutOnBeforeForwardToKeaOverHTTP {
	return &TestCalloutOnBeforeForwardToKeaOverHTTP{
		t:          t,
		callCounts: 0,
	}
}

func (tc *TestCalloutOnBeforeForwardToKeaOverHTTP) OnBeforeForwardToKeaOverHTTP(r *agentapi.ForwardToKeaOverHTTPReq) {
	require.NotNil(tc.t, r)
	tc.callCounts++
}

func TestOnBeforeForwardToKeaOverHTTPHook(t *testing.T) {
	// Arrange
	callout := newTestCalloutOnBeforeForwardToKeaOverHTTP(t)
	sa, ctx := setupAgentTestWithCallouts([]interface{}{callout})
	req := &agentapi.ForwardToKeaOverHTTPReq{
		Url:         "http://localhost:45634/",
		KeaRequests: []*agentapi.KeaRequest{{Request: "{ \"command\": \"list-commands\"}"}},
	}

	// Act
	_, _ = sa.ForwardToKeaOverHTTP(ctx, req)

	// Arrange
	require.EqualValues(t, 1, callout.callCounts)
}
