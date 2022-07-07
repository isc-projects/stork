package agent

import (
	"reflect"

	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
	"isc.org/stork/hooks/agent/forwardtokeaoverhttpcallout"
	"isc.org/stork/hooksutil"
)

// Callout types
var (
	beforeForwardToKeaOverHTTPCalloutType reflect.Type = reflect.TypeOf((*forwardtokeaoverhttpcallout.BeforeForwardToKeaOverHTTPCallout)(nil)).Elem()
)

// Hook executor
func newHookExecutor() *hooksutil.HookExecutor {
	executor := hooksutil.NewHookExecutor([]reflect.Type{
		beforeForwardToKeaOverHTTPCalloutType,
	})
	return executor
}

type HookManager struct {
	executor *hooksutil.HookExecutor
}

// Interface checks
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
	hm.executor.CallSequential(beforeForwardToKeaOverHTTPCalloutType, func(rawCallout interface{}) {
		callout := rawCallout.(forwardtokeaoverhttpcallout.BeforeForwardToKeaOverHTTPCallout)
		callout.OnBeforeForwardToKeaOverHTTP(in)
	})
}
