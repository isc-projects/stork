package main

import (
	"isc.org/stork/hooks"
)

// Loads a callout carrier (an object with the callout specification implementations).
func Load(settings hooks.HookSettings) (hooks.CalloutCarrier, error) {
	return &calloutCarrier{}, nil
}

// Returns an application name and expected version.
func Version() (string, string) {
	return hooks.HookProgramAgent, hooks.StorkVersion
}

// Optional support for providing settings.
// Returns a prototype of the settings. The prototype includes the tags
// with definitions of the parameters. This method (if provided) is called
// BEFORE loading the hook.
func ProtoSettings() hooks.HookSettings {
	return nil
}

// Type guards.
var (
	_ hooks.HookLoadFunction          = Load
	_ hooks.HookVersionFunction       = Version
	_ hooks.HookProtoSettingsFunction = ProtoSettings
)
