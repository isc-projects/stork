package hooksutil

import (
	"reflect"

	"isc.org/stork/hooks"
	storkutil "isc.org/stork/util"
)

// Facade for all callouts. It defines the specific calling method for
// each callout.
type HookManager struct {
	executor    *HookExecutor
	allSettings map[string]hooks.HookSettings
	walker      *HookWalker
}

// Constructs the hook manager.
func NewHookManager(supportedTypes []reflect.Type) *HookManager {
	return &HookManager{
		executor:    NewHookExecutor(supportedTypes),
		allSettings: map[string]hooks.HookSettings{},
		walker:      NewHookWalker(),
	}
}

// Searches for the compatible hook libraries in a given directory and extracts
// the prototypes of their settings.
// The hooks are not loaded.
func (hm *HookManager) CollectProtoSettingsFromDirectory(program, directory string) error {
	allSettings, err := hm.walker.CollectProtoSettings(program, directory)
	if err != nil {
		return err
	}
	hm.allSettings = allSettings
	return nil
}

// Registers all hooks from a given hook directory.
func (hm *HookManager) RegisterHooksFromDirectory(program, directory string) error {
	carriers, err := hm.walker.LoadAllHooks(program, directory, hm.allSettings)
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
