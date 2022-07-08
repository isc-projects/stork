package main

import (
	"isc.org/stork/hooks"
)

func Load() (hooks.Closer, error) {
	return &callouts{}, nil
}

func Version() (string, string) {
	return hooks.HookProgramAgent, hooks.StorkVersion
}

// Type guards
var _ hooks.HookLoadFunction = Load
var _ hooks.HookVersionFunction = Version
