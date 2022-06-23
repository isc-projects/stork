package hooks

import "isc.org/stork"

type HookLoadFunction func() (interface{}, error)
type HookVersionFunction func() (string, string)

const (
	HookLoadFunctionName    = "Load"
	HookVersionFunctionName = "Version"
	HookProgramAgent        = "Stork Agent"
	HookProgramServer       = "Stork Server"
)

const StorkVersion = stork.Version
