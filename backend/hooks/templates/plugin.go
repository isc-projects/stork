package main

import (
	"isc.org/stork/hooks"
)

// Loads an object with the callout points implementations.
func Load() (any, error) {
	return &callout{}, nil
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
