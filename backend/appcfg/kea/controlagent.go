package keaconfig

var _ commonConfigAccessor = (*CtrlAgentConfig)(nil)

// Represents Kea Control Agent's configuration.
type CtrlAgentConfig struct {
	ControlSockets *ControlSockets `json:"control-sockets,omitempty"`
	HTTPHost       *string         `json:"http-host,omitempty"`
	HTTPPort       *int64          `json:"http-port,omitempty"`
	TrustAnchor    *string         `json:"trust-anchor,omitempty"`
	CertFile       *string         `json:"cert-file,omitempty"`
	KeyFile        *string         `json:"key-file,omitempty"`
	CertRequired   *bool           `json:"cert-required,omitempty"`
	HookLibraries  []HookLibrary   `json:"hooks-libraries"`
	Loggers        []Logger        `json:"loggers"`
	Authentication *Authentication `json:"authentication,omitempty"`
}

// Represents settable Kea Control Agent's configuration.
type SettableCtrlAgentConfig struct{}

// A structure representing the configuration of multiple control sockets
// in the Kea Control Agent.
type ControlSockets struct {
	D2      *ControlSocket `json:"d2,omitempty"`
	Dhcp4   *ControlSocket `json:"dhcp4,omitempty"`
	Dhcp6   *ControlSocket `json:"dhcp6,omitempty"`
	NetConf *ControlSocket `json:"netconf,omitempty"`
}

// A structure representing a configuration of a single control socket in
// the  Kea Control Agent.
type ControlSocket struct {
	SocketName string `json:"socket-name"`
	SocketType string `json:"socket-type"`
}

// A structure representing the client credentials in the Kea Control Agent.
type ClientCredentials struct {
	User         *string `json:"user"`
	Password     *string `json:"password"`
	UserFile     *string `json:"user-file"`
	PasswordFile *string `json:"password-file"`
}

// A structure representing a configuration of the authentication credentials
// in the Kea Control Agent.
type Authentication struct {
	Type      string              `json:"type"`
	Realm     string              `json:"realm"`
	Directory *string             `json:"directory"`
	Clients   []ClientCredentials `json:"clients"`
}

// Indicates whether the basic auth method is used.
func (a Authentication) IsBasicAuth() bool {
	return a.Type == "basic"
}

// Returns a list of daemons for which sockets have been configured.
func (cs *ControlSockets) GetConfiguredDaemonNames() (names []string) {
	if cs == nil {
		return
	}

	if cs.D2 != nil {
		names = append(names, "d2")
	}
	if cs.Dhcp4 != nil {
		names = append(names, "dhcp4")
	}
	if cs.Dhcp6 != nil {
		names = append(names, "dhcp6")
	}
	if cs.NetConf != nil {
		names = append(names, "netconf")
	}

	return
}

// Returns true if any control socket is configured.
func (cs *ControlSockets) HasAnyConfiguredDaemon() bool {
	return cs != nil && (cs.D2 != nil || cs.Dhcp4 != nil || cs.Dhcp6 != nil || cs.NetConf != nil)
}

// Returns the configured control sockets.
func (c *CtrlAgentConfig) GetControlSockets() *ControlSockets {
	return c.ControlSockets
}

// Returns the hook libraries configured in the Kea Control Agent.
func (c *CtrlAgentConfig) GetHookLibraries() HookLibraries {
	return c.HookLibraries
}

// Returns the loggers configured in the Kea Control Agent.
func (c *CtrlAgentConfig) GetLoggers() []Logger {
	return c.Loggers
}

// Returns an HTTP host at the top level of the configuration.
// Some values are normalized to valid IP addresses.
// If the given parameter does not exist, the host is localhost, and
// the ok value returned is set to false.
func (c *Config) GetHTTPHost() (address string, ok bool) {
	if !c.IsCtrlAgent() || c.HTTPHost == nil {
		address = "127.0.0.1"
		return
	}
	address = *c.HTTPHost
	switch address {
	case "0.0.0.0", "":
		address = "127.0.0.1"
	case "::":
		address = "::1"
	}
	ok = true
	return
}

// Returns an HTTP port at the top level of the configuration.
// If the given parameter does not exist, the port is zero, and
// the ok value returned is set to false.
func (c *Config) GetHTTPPort() (out int64, ok bool) {
	if c.IsCtrlAgent() && c.HTTPPort != nil {
		out = *c.HTTPPort
		ok = true
	}
	return
}

// Returns a trust anchor path at the top level of the configuration.
// If the given parameter does not exist, the output is empty string, and
// the ok value returned is set to false.
func (c *Config) GetTrustAnchor() (out string, ok bool) {
	if c.IsCtrlAgent() && c.TrustAnchor != nil {
		out = *c.TrustAnchor
		ok = true
	}
	return
}

// Returns a cert file path at the top level of the configuration.
// If the given parameter does not exist, the output is empty string, and
// the ok value returned is set to false.
func (c *Config) GetCertFile() (out string, ok bool) {
	if c.IsCtrlAgent() && c.CertFile == nil {
		return "", false
	}
	return *c.CertFile, true
}

// Returns a key file path at the top level of the configuration.
// If the given parameter does not exist, the output is empty string, and
// the ok value returned is set to false.
func (c *Config) GetKeyFile() (out string, ok bool) {
	if c.IsCtrlAgent() && c.KeyFile == nil {
		return "", false
	}
	return *c.KeyFile, true
}

// Returns a cert required flag at the top level of the configuration.
// If the given parameter does not exist, the output is false, and
// the ok value returned is set to false.
func (c *Config) GetCertRequired() (out bool, ok bool) {
	if c.IsCtrlAgent() && c.CertRequired == nil {
		return
	}
	out = *c.CertRequired
	ok = true
	return
}

// Returns true when the Kea Control Agent is configured to use the HTTPS connections.
func (c *Config) UseSecureProtocol() bool {
	trustAnchor, _ := c.GetTrustAnchor()
	certFile, _ := c.GetCertFile()
	keyFile, _ := c.GetKeyFile()
	return len(trustAnchor) != 0 && len(certFile) != 0 && len(keyFile) != 0
}

// Return basic auth credentials if provided in the configuration.
func (c *Config) GetBasicAuthenticationDetails() *Authentication {
	if !c.IsCtrlAgent() || c.Authentication == nil || !c.Authentication.IsBasicAuth() {
		return nil
	}

	return c.Authentication
}
