package forwardtokeaoverhttpcallouts

import (
	"context"

	agentapi "isc.org/stork/api"
)

type BeforeForwardToKeaOverHTTPCallouts interface {
	// Called before forwarding a command to Kea over HTTP.
	OnBeforeForwardToKeaOverHTTP(ctx context.Context, in *agentapi.ForwardToKeaOverHTTPReq) error
}
