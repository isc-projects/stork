package agent

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
)

func TestOnBeforeForwardToKeaOverHTTPCallout(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockBeforeForwardToKeaOverHTTPCallouts(ctrl)
	mock.
		EXPECT().
		OnBeforeForwardToKeaOverHTTP(context.Background(), gomock.Any()).
		Times(1)

	sa, ctx := setupAgentTestWithHooks([]hooks.CalloutCarrier{mock})
	req := &agentapi.ForwardToKeaOverHTTPReq{
		Url:         "http://localhost:45634/",
		KeaRequests: []*agentapi.KeaRequest{{Request: "{ \"command\": \"list-commands\"}"}},
	}

	// Act
	_, _ = sa.ForwardToKeaOverHTTP(ctx, req)

	// Assert
	// Call assertion inside a mock.
}
