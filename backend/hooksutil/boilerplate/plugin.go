package main

import (
	"isc.org/stork/hooks"
)

// Loads a callout carrier (an object with the callout specification implementations).
func Load() (hooks.CalloutCarrier, error) {
	return &calloutCarrier{}, nil
}

// Returns an application name and expected version.
func Version() (string, string) {
	return hooks.HookProgramAgent, hooks.StorkVersion
}

// Type guards.
var (
	_ hooks.HookLoadFunction    = Load
	_ hooks.HookVersionFunction = Version
)
