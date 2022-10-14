package hookmanager

import (
	"reflect"

	"isc.org/stork/hooks"
	"isc.org/stork/hooks/server/authenticationcallout"
	"isc.org/stork/hooksutil"
	storkutil "isc.org/stork/util"
)

// Constructs a new hook executor and register the callouts supported by the
// Stork server.
func newHookExecutor() *hooksutil.HookExecutor {
	executor := hooksutil.NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*authenticationcallout.AuthenticationCallout)(nil)).Elem(),
	})
	return executor
}

// Facade for all callout points. It defines the specific calling method for
// each callout point.
type HookManager struct {
	executor *hooksutil.HookExecutor
}

// Constructs the hook manager.
func newHookManager(executor *hooksutil.HookExecutor) *HookManager {
	return &HookManager{
		executor: executor,
	}
}

// Constructs the hook manager without registering any callout points.
func NewHookManager() *HookManager {
	return newHookManager(newHookExecutor())
}

// Loads all hooks from a given hook directory and construct the hook manager.
func NewHookManagerFromDirectory(directory string) (*HookManager, error) {
	allCallouts, err := hooksutil.LoadAllHooks(hooks.HookProgramServer, directory)
	if err != nil {
		return nil, err
	}
	return NewHookManagerFromCallouts(allCallouts), nil
}

// Constructs the hook manager using a list of the objects with callout points
// implementations.
func NewHookManagerFromCallouts(allCallouts []hooks.Callout) *HookManager {
	executor := newHookExecutor()
	for _, callouts := range allCallouts {
		executor.RegisterCallouts(callouts)
	}
	return newHookManager(executor)
}

// Unregisters all callout objects.
func (hm *HookManager) Close() error {
	errs := hm.executor.UnregisterAllCallouts()
	return storkutil.CombineErrors("some hooks failed to close", errs)
}
