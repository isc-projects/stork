package agent

import (
	"reflect"

	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
	"isc.org/stork/hooks/agent/forwardtokeaoverhttpcallout"
	"isc.org/stork/hooksutil"
)

// Constructs the hook executor and register the callouts supported by Stork agent.
func newHookExecutor() *hooksutil.HookExecutor {
	executor := hooksutil.NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*forwardtokeaoverhttpcallout.BeforeForwardToKeaOverHTTPCallout)(nil)).Elem(),
	})
	return executor
}

// Facade for all callout points. It defines the specific calling method for
// each callout point.
type HookManager struct {
	executor *hooksutil.HookExecutor
}

// Interface checks.
var _ forwardtokeaoverhttpcallout.BeforeForwardToKeaOverHTTPCallout = (*HookManager)(nil)

// Constructs new hook manager.
func newHookManager(executor *hooksutil.HookExecutor) *HookManager {
	return &HookManager{
		executor: executor,
	}
}

// Loads the hooks from a given directory and constructs the hook manager.
func NewHookManagerFromDirectory(directory string) (*HookManager, error) {
	allCallouts, err := hooksutil.LoadAllHooks(hooks.HookProgramAgent, directory)
	if err != nil {
		return nil, err
	}
	return NewHookManagerFromCallouts(allCallouts), nil
}

// Constructs the hook manager using the list of objects with the callout
// points implementations.
func NewHookManagerFromCallouts(allCallouts []any) *HookManager {
	executor := newHookExecutor()
	for _, callouts := range allCallouts {
		executor.RegisterCallouts(callouts)
	}
	return newHookManager(executor)
}

// Unregisters all callout objects.
func (hm *HookManager) Close() {
	hm.executor.UnregisterAllCallouts()
}

// Callout point executed before forwarding a command to Kea over HTTP.
func (hm *HookManager) OnBeforeForwardToKeaOverHTTP(in *agentapi.ForwardToKeaOverHTTPReq) {
	hooksutil.CallSequential(hm.executor, func(callout forwardtokeaoverhttpcallout.BeforeForwardToKeaOverHTTPCallout) int {
		callout.OnBeforeForwardToKeaOverHTTP(in)
		return 0
	})
}
