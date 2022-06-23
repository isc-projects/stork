package hooks

import (
	agentapi "isc.org/stork/api"
)

type BeforeForwardToKeaOverHTTPCallout interface {
	OnBeforeForwardToKeaOverHTTP(in *agentapi.ForwardToKeaOverHTTPReq)
}
