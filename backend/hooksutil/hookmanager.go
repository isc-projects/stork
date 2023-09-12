package hooksutil

import (
	"reflect"

	"isc.org/stork/hooks"
	storkutil "isc.org/stork/util"
)

// Facade for all callouts. It defines the specific calling method for
// each callout.
type HookManager struct {
	executor *HookExecutor
	walker   *HookWalker
}

// Constructs the hook manager.
func NewHookManager(supportedTypes []reflect.Type) *HookManager {
	return &HookManager{
		executor: NewHookExecutor(supportedTypes),
		walker:   NewHookWalker(),
	}
}

// Registers all hooks from a given hook directory.
//
// The settings map passed to this method must be indexed by the hook names
// returned by the "LibraryManager.GetName" function.
//
// If a particular hook doesn't support configuring, the related hook settings
// may be nil or be omitted.
func (hm *HookManager) RegisterHooksFromDirectory(program, directory string, allSettings map[string]hooks.HookSettings) error {
	carriers, err := hm.walker.LoadAllHooks(program, directory, allSettings)
	if err != nil {
		return err
	}

	hm.RegisterCalloutCarriers(carriers)

	return nil
}

// Register callout carriers.
func (hm *HookManager) RegisterCalloutCarriers(carriers []hooks.CalloutCarrier) {
	for _, carrier := range carriers {
		hm.RegisterCalloutCarrier(carrier)
	}
}

// Register a callout carrier.
func (hm *HookManager) RegisterCalloutCarrier(carrier hooks.CalloutCarrier) {
	hm.executor.registerCalloutCarrier(carrier)
}

// Get accessor of the executor to use with the Call functions.
func (hm *HookManager) GetExecutor() *HookExecutor {
	return hm.executor
}

// Unregisters all callout carriers.
func (hm *HookManager) Close() error {
	errs := hm.executor.unregisterAllCalloutCarriers()
	return storkutil.CombineErrors("some hooks failed to close", errs)
}
