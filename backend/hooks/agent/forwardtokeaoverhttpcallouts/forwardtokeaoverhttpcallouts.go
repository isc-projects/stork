package forwardtokeaoverhttpcallouts

import (
	"context"

	agentapi "isc.org/stork/api"
)

// The callout specification used before forwarding a request to Kea.
type BeforeForwardToKeaOverHTTPCallouts interface {
	// Called before forwarding a command to Kea over HTTP.
	OnBeforeForwardToKeaOverHTTP(ctx context.Context, in *agentapi.ForwardToKeaOverHTTPReq) error
}
