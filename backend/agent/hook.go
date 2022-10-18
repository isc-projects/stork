package agent

import (
	"reflect"

	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
)

// Facade for all callout points. It defines the specific calling method for
// each callout point.
type HookManager struct {
	hooks.HookManager
}

// Interface checks.
var _ BeforeForwardToKeaOverHTTPCallout = (*HookManager)(nil)

// Constructs new hook manager.
func NewHookManager() *HookManager {
	return &HookManager{
		HookManager: *hooks.NewHookManager([]reflect.Type{
			reflect.TypeOf((*BeforeForwardToKeaOverHTTPCallout)(nil)).Elem(),
		}),
	}
}

// Callout point executed before forwarding a command to Kea over HTTP.
func (hm *HookManager) OnBeforeForwardToKeaOverHTTP(in *agentapi.ForwardToKeaOverHTTPReq) {
	hooks.CallSequential(hm.GetExecutor(), func(callout BeforeForwardToKeaOverHTTPCallout) int {
		callout.OnBeforeForwardToKeaOverHTTP(in)
		return 0
	})
}
