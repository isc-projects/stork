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

// Searches for the compatible hook libraries in a given directory and extracts
// the their CLI flags. The hooks are not loaded.
//
// Returns the collected CLI flags for each compatible hook in the directory.
// The members of returned hook settings should be set to expected values. The
// CLI flags related to each member are described by their tags. The keys of
// the returned map uniquely identify the hooks.
// The map with configured hook settings should be passed to the call of the
// RegisterHooksFromDirectory method.
// The map keys must be preserved. They are used to assign the settings to
// specific hooks.
//
// The related hook settings will be nil if the particular hook doesn't support
// configuring. The nil value may be preserved in the map passed to the
// RegisterHooksFromDirectory method.
func (hm *HookManager) CollectCLIFlagsFromDirectory(program, directory string) (map[string]hooks.HookSettings, error) {
	allSettings, err := hm.walker.CollectCLIFlags(program, directory)
	if err != nil {
		return nil, err
	}
	return allSettings, nil
}

// Registers all hooks from a given hook directory.
//
// The settings map passed to this method must be indexed by the same keys as
// returned by the CollectCLIFlagsFromDirectory method.
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
