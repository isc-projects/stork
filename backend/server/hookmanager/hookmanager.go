package hookmanager

import (
	"reflect"

	"isc.org/stork/hooks"
	"isc.org/stork/hooks/server/authenticationcallout"
	"isc.org/stork/hooksutil"
)

func newHookExecutor() *hooksutil.HookExecutor {
	executor := hooksutil.NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*authenticationcallout.AuthenticationCallout)(nil)).Elem(),
	})
	return executor
}

type HookManager struct {
	executor *hooksutil.HookExecutor
}

func newHookManager(executor *hooksutil.HookExecutor) *HookManager {
	return &HookManager{
		executor: executor,
	}
}

func NewHookManagerFromDirectory(directory string) *HookManager {
	allCallouts := hooksutil.LoadAllHooks(hooks.HookProgramServer, directory)
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
