package hooks

import "isc.org/stork"

type Callout interface {
	Close() error
}
type (
	HookLoadFunction    = func() (Callout, error)
	HookVersionFunction = func() (string, string)
)

const (
	HookLoadFunctionName    = "Load"
	HookVersionFunctionName = "Version"
	HookProgramAgent        = "Stork Agent"
	HookProgramServer       = "Stork Server"
)

const StorkVersion = stork.Version
