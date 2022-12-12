package forwardtokeaoverhttpcallouts

import (
	"context"

	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
)

type BeforeForwardToKeaOverHTTPCallouts interface {
	hooks.CalloutCarrier
	// Called before forwarding a command to Kea over HTTP.
	OnBeforeForwardToKeaOverHTTP(ctx context.Context, in *agentapi.ForwardToKeaOverHTTPReq)
}
