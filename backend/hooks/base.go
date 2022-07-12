package hooks

import "isc.org/stork"

type Closer interface {
	Close() error
}
type (
	HookLoadFunction    = func() (Closer, error)
	HookVersionFunction = func() (string, string)
)

const (
	HookLoadFunctionName    = "Load"
	HookVersionFunctionName = "Version"
	HookProgramAgent        = "Stork Agent"
	HookProgramServer       = "Stork Server"
)

const StorkVersion = stork.Version
