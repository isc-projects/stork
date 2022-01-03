package keaconfig

// Check if the map has an expected root node for Control Agent.
func (c *Map) IsControlAgent() bool {
	name, ok := c.GetRootName()
	if !ok {
		return false
	}
	return name == "Control-agent"
}

// Returns an string found at the top level of the configuration under a
// given name. If the given parameter does not exist, the string is empty, and
// the ok value returned is set to false.
func (c *Map) getTopLevelEntryString(entryName string) (out string, ok bool) {
	raw, ok := c.getTopLevelEntry(entryName)
	if ok {
		out, ok = raw.(string)
	}
	return
}

// Returns a HTTP host at the top level of the configuration.
// Some values are normalized to valid IP addresses.
// If the given parameter does not exist, the host is localhost, and
// the ok value returned is set to false.
func (c *Map) GetHTTPHost() (address string, ok bool) {
	address, ok = c.getTopLevelEntryString("http-host")
	if !ok {
		address = "127.0.0.1"
		return
	}

	switch address {
	case "0.0.0.0", "":
		address = "127.0.0.1"
	case "::":
		address = "::1"
	}

	return
}

// Returns a HTTP port at the top level of the configuration.
// If the given parameter does not exist, the port is zero, and
// the ok value returned is set to false.
func (c *Map) GetHTTPPort() (out int64, ok bool) {
	raw, ok := c.getTopLevelEntry("http-port")
	if ok {
		float, ok := raw.(float64)
		if ok {
			out = int64(float)
		}
	}
	return
}

// Returns a trust anchor path at the top level of the configuration.
// If the given parameter does not exist, the output is empty string, and
// the ok value returned is set to false.
func (c *Map) GetTrustAnchor() (out string, ok bool) {
	return c.getTopLevelEntryString("trust-anchor")
}

// Returns a cert file path at the top level of the configuration.
// If the given parameter does not exist, the output is empty string, and
// the ok value returned is set to false.
func (c *Map) GetCertFile() (out string, ok bool) {
	return c.getTopLevelEntryString("cert-file")
}

// Returns a key file path at the top level of the configuration.
// If the given parameter does not exist, the output is empty string, and
// the ok value returned is set to false.
func (c *Map) GetKeyFile() (out string, ok bool) {
	return c.getTopLevelEntryString("key-file")
}

// Returns a cert required flag at the top level of the configuration.
// If the given parameter does not exist, the output is false, and
// the ok value returned is set to false.
func (c *Map) GetCertRequired() (out bool, ok bool) {
	raw, ok := c.getTopLevelEntry("cert-required")
	if ok {
		out, ok = raw.(bool)
	}
	return
}

// Returns true when the Kea Control Agent is configured to use the HTTPS connections.
func (c *Map) UseSecureProtocol() bool {
	trustAnchor, _ := c.GetTrustAnchor()
	certFile, _ := c.GetCertFile()
	keyFile, _ := c.GetKeyFile()
	return len(trustAnchor) != 0 && len(certFile) != 0 && len(keyFile) != 0
}
