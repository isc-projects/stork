package hookmanager

import (
	"reflect"

	"isc.org/stork/hooks/server/authenticationcallouts"
	"isc.org/stork/hooksutil"
)

// Facade for all callouts. It defines the specific calling method for
// each callout.
type HookManager struct {
	hooksutil.HookManager
}

// Constructs the hook manager.
func NewHookManager() *HookManager {
	return &HookManager{
		HookManager: *hooksutil.NewHookManager([]reflect.Type{
			reflect.TypeOf((*authenticationcallouts.AuthenticationCallouts)(nil)).Elem(),
		}),
	}
}
