package keaconfig

import (
	"encoding/json"
	"strings"
)

// A type representing a collection of the hook libraries configured in
// the Kea server.
type HookLibraries []HookLibrary

// A structure representing a configuration of a single hook library.
type HookLibrary struct {
	Library    string          `json:"library"`
	Parameters json.RawMessage `json:"parameters"`
}

// A generic function parsing the hook library configuration into a custom
// structure, specific to the hook library. The last parameter is set false
// when the hook library is not configured.
func getHookLibraryWithParameters[T any](libraries *HookLibraries, name string) (path string, params T, ok bool) {
	lib, rawParams, exists := libraries.getHookLibrary(name)
	if exists {
		if len(rawParams) > 0 {
			err := json.Unmarshal(rawParams, &params)
			if err != nil {
				return
			}
		}
		path = lib
		ok = exists
	}
	return
}

// Searches for a hook library with the matching name. The third parameter is
// false when the library is not found. Otherwise it is true. The library
// parameters are returned in the raw form.
func (hl HookLibraries) getHookLibrary(name string) (path string, params json.RawMessage, ok bool) {
	for _, lib := range hl {
		if strings.Contains(lib.Library, name) {
			path = lib.Library
			params = lib.Parameters
			ok = true
			return
		}
	}
	return
}

// Returns the information about a hooks library having a specified name
// if it exists in the configuration. The name parameter designates the
// name of the library, e.g. libdhcp_ha. The returned values include the
// path to the library, library configuration and the flag indicating
// whether the library exists or not.
func (hl HookLibraries) GetHookLibrary(name string) (path string, params map[string]any, ok bool) {
	lib, rawParams, exists := hl.getHookLibrary(name)
	if exists {
		_ = json.Unmarshal(rawParams, &params)
		path = lib
		ok = exists
	}
	return
}

// Returns a configuration of the HA hook library in the parsed form.
// The last parameter is set false when the hook library is not configured.
func (hl HookLibraries) GetHAHookLibrary() (path string, params HALibraryParams, ok bool) {
	return getHookLibraryWithParameters[HALibraryParams](&hl, "libdhcp_ha")
}

// Returns a configuration of the legal log hook library in the parsed form.
// The last parameter is set false when the hook library is not configured.
func (hl HookLibraries) GetLeaseCmdsHookLibrary() (path string, params LeaseCmdsHookParams, ok bool) {
	return getHookLibraryWithParameters[LeaseCmdsHookParams](&hl, "libdhcp_lease_cmds")
}

// Returns a configuration of the legal log hook library in the parsed form.
// The last parameter is set false when the hook library is not configured.
func (hl HookLibraries) GetLegalLogHookLibrary() (path string, params LegalLogHookParams, ok bool) {
	return getHookLibraryWithParameters[LegalLogHookParams](&hl, "libdhcp_legal_log")
}
