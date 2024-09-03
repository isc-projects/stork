package keaconfig

var _ commonConfigAccessor = (*D2Config)(nil)

// Represents a D2 (DHCP-DDNS) Kea configuration.
type D2Config struct {
	HookLibraries []HookLibrary `json:"hooks-libraries,omitempty"`
	Loggers       []Logger      `json:"loggers,omitempty"`
}

// Represents settable D2 (DHCP-DDNS) Kea configuration.
type SettableD2Config struct{}

// Returns the hook libraries configured in the D2 server.
func (c *D2Config) GetHookLibraries() HookLibraries {
	return c.HookLibraries
}

// Returns the loggers configured in the D2 server.
func (c *D2Config) GetLoggers() []Logger {
	return c.Loggers
}
