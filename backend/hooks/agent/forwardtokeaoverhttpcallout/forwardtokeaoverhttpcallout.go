package forwardtokeaoverhttpcallout

import (
	agentapi "isc.org/stork/api"
)

type BeforeForwardToKeaOverHTTPCallout interface {
	// Called before forwarding a command to Kea over HTTP.
	OnBeforeForwardToKeaOverHTTP(in *agentapi.ForwardToKeaOverHTTPReq)
}
