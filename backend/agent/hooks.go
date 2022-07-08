package agent

import (
	"reflect"

	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
	"isc.org/stork/hooks/agent/forwardtokeaoverhttpcallout"
	"isc.org/stork/hooksutil"
)

// Hook executor.
func newHookExecutor() *hooksutil.HookExecutor {
	executor := hooksutil.NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*forwardtokeaoverhttpcallout.BeforeForwardToKeaOverHTTPCallout)(nil)).Elem(),
	})
	return executor
}

type HookManager struct {
	executor *hooksutil.HookExecutor
}

// Interface checks.
var _ forwardtokeaoverhttpcallout.BeforeForwardToKeaOverHTTPCallout = (*HookManager)(nil)

func newHookManager(executor *hooksutil.HookExecutor) *HookManager {
	return &HookManager{
		executor: executor,
	}
}

func NewHookManagerFromDirectory(directory string) *HookManager {
	allCallouts := hooksutil.LoadAllHooks(hooks.HookProgramAgent, directory)
	return NewHookManagerFromCallouts(allCallouts)
}

func NewHookManagerFromCallouts(allCallouts []interface{}) *HookManager {
	executor := newHookExecutor()
	for _, callouts := range allCallouts {
		executor.RegisterCallouts(callouts)
	}
	return newHookManager(executor)
}

func (hm *HookManager) Close() {
	hm.executor.UnregisterAllCallouts()
}

func (hm *HookManager) OnBeforeForwardToKeaOverHTTP(in *agentapi.ForwardToKeaOverHTTPReq) {
	hooksutil.CallSequential(hm.executor, func(callout forwardtokeaoverhttpcallout.BeforeForwardToKeaOverHTTPCallout) int {
		callout.OnBeforeForwardToKeaOverHTTP(in)
		return 0
	})
}
