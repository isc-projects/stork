package keaconfig

var _ commonConfigAccessor = (*D2Config)(nil)

// Represents a D2 (DHCP-DDNS) Kea configuration.
type D2Config struct {
	HookLibraries []HookLibrary `json:"hooks-libraries,omitempty"`
	Loggers       []Logger      `json:"loggers,omitempty"`
	// Replaced by ControlSockets in Kea 2.7.2.
	ControlSocket  *ControlSocket  `json:"control-socket,omitempty"`
	ControlSockets []ControlSocket `json:"control-sockets,omitempty"`
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

// Returns the control sockets on which the DHCPv6 server listens for incoming
// connections. If both ControlSockets and ControlSocket are defined, the
// ControlSockets takes precedence. If neither is defined, nil is returned.
func (c *D2Config) GetListeningControlSockets() []ControlSocket {
	if c.ControlSockets != nil {
		return c.ControlSockets
	}
	if c.ControlSocket != nil {
		return []ControlSocket{*c.ControlSocket}
	}
	return nil
}
