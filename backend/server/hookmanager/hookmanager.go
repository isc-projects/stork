package hookmanager

import (
	"reflect"

	"isc.org/stork/hooks/server/authenticationcallout"
	"isc.org/stork/hooksutil"
)

// Facade for all callout points. It defines the specific calling method for
// each callout point.
type HookManager struct {
	hooksutil.HookManager
}

// Constructs the hook manager.
func NewHookManager() *HookManager {
	return &HookManager{
		HookManager: *hooksutil.NewHookManager([]reflect.Type{
			reflect.TypeOf((*authenticationcallout.AuthenticationCallout)(nil)).Elem(),
		}),
	}
}
