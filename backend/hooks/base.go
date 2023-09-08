package hooks

import "isc.org/stork"

// Defines the standard set of methods that all callout carriers must provide.
type CalloutCarrier interface {
	// Close the carrier and free all used resources. The carrier should not be
	// used after calling this method.
	Close() error
}

// Defines the standard set of methods that all hook settings must provide.
type HookSettings interface {
	// Nothing common but the HookSettings type is more descriptive than just
	// any.
}

type (
	// A function that creates a callout carrier object with the callout
	// implementations.
	// Accepts the settings object - it should be an instance
	// returned by the 'CLIFlags' function with provided values of the
	// members. It should be nil if the hook doesn't require configuring.
	HookLoadFunction = func(settings HookSettings) (CalloutCarrier, error)
	// Returns a compatible program identifier and version of the binary. It
	// must be safe to call before the Load function.
	HookGetVersionFunction = func() (string, string)
	// Returns the CLI flags. It must be a pointer to structure.
	// The object defines the accepted CLI and environment variables in the
	// form compatible with the go-flag library. The function is optional and
	// may be omitted if the hook doesn't require configuring.
	HookCreateCLIFlagsFunction = func() HookSettings
)

const (
	// An embedded name for the Load function.
	HookLoadFunctionName = "Load"
	// An embedded name for the GetVersion function.
	HookGetVersionFunctionName = "GetVersion"
	// An embedded name for the CreateCLIFlags function.
	HookCreateCLIFlagsFunctionName = "CreateCLIFlags"
	// Identifier of the Stork Agent program.
	HookProgramAgent = "Stork Agent"
	// Identifier of the Stork Server program.
	HookProgramServer = "Stork Server"
)

// The current Stork version. Forwarded from the top Stork package.
const StorkVersion = stork.Version
