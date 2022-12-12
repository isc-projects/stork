package hooks

import "isc.org/stork"

// Defines the standard set of methods that all callout carriers must provide.
type CalloutCarrier interface {
	// Close the carrier and free all used resources. The carrier should not be
	// used after calling this method.
	Close() error
}

type (
	// A function that creates a callout carrier object with the callout
	// implementations.
	HookLoadFunction = func() (CalloutCarrier, error)
	// Returns a compatible program identifier and version of the binary. It
	// must be safe to call before the Load function.
	HookVersionFunction = func() (string, string)
)

const (
	// An embedded name for the Load function.
	HookLoadFunctionName = "Load"
	// An embedded name for the Version function.
	HookVersionFunctionName = "Version"
	// Identifier of the Stork Agent program.
	HookProgramAgent = "Stork Agent"
	// Identifier of the Stork Server program.
	HookProgramServer = "Stork Server"
)

// The current Stork version. Forwarded from the top Stork package.
const StorkVersion = stork.Version
