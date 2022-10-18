package main

import (
	"isc.org/stork/hooks"
)

func Load() (hooks.Callout, error) {
	return &callout{}, nil
}

func Version() (string, string) {
	return hooks.HookProgramServer, hooks.StorkVersion
}

// Type guards.
var (
	_ hooks.HookLoadFunction    = Load
	_ hooks.HookVersionFunction = Version
)
