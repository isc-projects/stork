package keaconfig

var _ commonConfigAccessor = (*CtrlAgentConfig)(nil)

// Represents Kea Control Agent's configuration.
type CtrlAgentConfig struct {
	ManagementControlSockets *ManagementControlSockets `json:"control-sockets,omitempty"`
	HTTPHost                 *string                   `json:"http-host,omitempty"`
	HTTPPort                 *int64                    `json:"http-port,omitempty"`
	TrustAnchor              *string                   `json:"trust-anchor,omitempty"`
	CertFile                 *string                   `json:"cert-file,omitempty"`
	KeyFile                  *string                   `json:"key-file,omitempty"`
	CertRequired             *bool                     `json:"cert-required,omitempty"`
	HookLibraries            []HookLibrary             `json:"hooks-libraries"`
	Loggers                  []Logger                  `json:"loggers"`
	Authentication           *Authentication           `json:"authentication,omitempty"`
}

// Represents settable Kea Control Agent's configuration.
type SettableCtrlAgentConfig struct{}

// Returns the configured management control sockets.
func (c *CtrlAgentConfig) GetManagementControlSockets() *ManagementControlSockets {
	return c.ManagementControlSockets
}

// Returns the control socket on which the Kea Control Agent listens for
// incoming connections. The Kea CA has no such entry, but all necessary data
// are defined at the top level of the configuration.
func (c *CtrlAgentConfig) GetListeningControlSockets() []ControlSocket {
	useSecureProtocol := c.TrustAnchor != nil && c.CertFile != nil && c.KeyFile != nil &&
		len(*c.TrustAnchor) != 0 && len(*c.CertFile) != 0 && len(*c.KeyFile) != 0

	socketType := "http"
	if useSecureProtocol {
		socketType = "https"
	}

	address := "127.0.0.1"
	if c.HTTPHost != nil {
		address = *c.HTTPHost
	}
	switch address {
	case "0.0.0.0", "":
		address = "127.0.0.1"
	case "::":
		address = "::1"
	}

	port := int64(8000)
	if c.HTTPPort != nil {
		port = *c.HTTPPort
	}

	return []ControlSocket{
		{
			SocketType:     socketType,
			SocketAddress:  &address,
			SocketPort:     &port,
			TrustAnchor:    c.TrustAnchor,
			CertFile:       c.CertFile,
			KeyFile:        c.KeyFile,
			CertRequired:   c.CertRequired,
			Authentication: c.Authentication,
		},
	}
}

// Returns the hook libraries configured in the Kea Control Agent.
func (c *CtrlAgentConfig) GetHookLibraries() HookLibraries {
	return c.HookLibraries
}

// Returns the loggers configured in the Kea Control Agent.
func (c *CtrlAgentConfig) GetLoggers() []Logger {
	return c.Loggers
}
