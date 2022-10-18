package hookmanager

import (
	"reflect"

	"isc.org/stork/hooks"
	"isc.org/stork/server/callouts/authenticationcallout"
)

// Facade for all callout points. It defines the specific calling method for
// each callout point.
type HookManager struct {
	hooks.HookManager
}

// Constructs the hook manager.
func NewHookManager() *HookManager {
	return &HookManager{
		HookManager: *hooks.NewHookManager([]reflect.Type{
			reflect.TypeOf((*authenticationcallout.AuthenticationCallout)(nil)).Elem(),
		}),
	}
}
