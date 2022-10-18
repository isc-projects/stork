package forwardtokeaoverhttpcallout

import (
	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
)

type BeforeForwardToKeaOverHTTPCallout interface {
	hooks.Callout
	// Called before forwarding a command to Kea over HTTP.
	OnBeforeForwardToKeaOverHTTP(in *agentapi.ForwardToKeaOverHTTPReq)
}
