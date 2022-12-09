package forwardtokeaoverhttpcallout

import (
	"context"

	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
)

type BeforeForwardToKeaOverHTTPCallout interface {
	hooks.Callout
	// Called before forwarding a command to Kea over HTTP.
	OnBeforeForwardToKeaOverHTTP(ctx context.Context, in *agentapi.ForwardToKeaOverHTTPReq)
}
