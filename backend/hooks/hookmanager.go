package hooks

import (
	"reflect"

	storkutil "isc.org/stork/util"
)

// Facade for all callout points. It defines the specific calling method for
// each callout point.
type HookManager struct {
	executor *HookExecutor
}

// Constructs the hook manager.
func NewHookManager(supportedTypes []reflect.Type) *HookManager {
	return &HookManager{
		executor: NewHookExecutor(supportedTypes),
	}
}

// Registers all hooks from a given hook directory.
func (hm *HookManager) RegisterCalloutsFromDirectory(directory string) error {
	callouts, err := LoadAllHookCallouts(HookProgramServer, directory)
	if err != nil {
		return err
	}

	hm.RegisterCallouts(callouts)

	return nil
}

// Register callouts.
func (hm *HookManager) RegisterCallouts(callouts []Callout) {
	for _, callout := range callouts {
		hm.executor.registerCallout(callout)
	}
}

// Get accessor of the executor to use with the Call functions.
func (hm *HookManager) GetExecutor() *HookExecutor {
	return hm.executor
}

// Unregisters all callout objects.
func (hm *HookManager) Close() error {
	errs := hm.executor.unregisterAllCallouts()
	return storkutil.CombineErrors("some hooks failed to close", errs)
}
