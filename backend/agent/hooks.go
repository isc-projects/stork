package agent

import (
	"reflect"

	agentapi "isc.org/stork/api"
	"isc.org/stork/hooks"
)

// Callout types
var (
	BeforeForwardToKeaOverHTTPCalloutType reflect.Type = reflect.TypeOf((*hooks.BeforeForwardToKeaOverHTTPCallout)(nil)).Elem()
)

// Hook executor
func newHookExecutor() *hooks.HookExecutor {
	executor := hooks.NewHookExecutor([]reflect.Type{
		BeforeForwardToKeaOverHTTPCalloutType,
	})
	return executor
}

type HookManager struct {
	executor *hooks.HookExecutor
}

// Interface checks
var _ hooks.BeforeForwardToKeaOverHTTPCallout = (*HookManager)(nil)

func newHookManager(executor *hooks.HookExecutor) *HookManager {

	return &HookManager{
		executor: executor,
	}
}

func NewHookManagerFromDirectory(directory string) *HookManager {
	allCallouts := hooks.LoadAllHooks(hooks.HookProgramAgent, directory)
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
	hm.executor.CallSequential(BeforeForwardToKeaOverHTTPCalloutType, func(rawCallout interface{}) {
		callout := rawCallout.(hooks.BeforeForwardToKeaOverHTTPCallout)
		callout.OnBeforeForwardToKeaOverHTTP(in)
	})
}
