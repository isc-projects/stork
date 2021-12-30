package keaconfig

// Check if the map has a expected root node for Control Agent.
func (c *Map) IsControlAgent() bool {
	name, ok := c.GetRootName()
	if !ok {
		return false
	}
	return name == "Control-agent"
}

func (c *Map) getTopLevelEntryString(entryName string) (out string, ok bool) {
	raw, ok := c.getTopLevelEntry(entryName)
	if ok {
		out, ok = raw.(string)
	}
	return
}

func (c *Map) GetHTTPHost() (string, bool) {
	return c.getTopLevelEntryString("http-host")
}

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

func (c *Map) GetTrustAnchor() (out string, ok bool) {
	return c.getTopLevelEntryString("trust-anchor")
}

func (c *Map) GetCertFile() (out string, ok bool) {
	return c.getTopLevelEntryString("cert-file")
}

func (c *Map) GetKeyFile() (out string, ok bool) {
	return c.getTopLevelEntryString("key-file")
}

func (c *Map) GetCertRequired() (out bool, ok bool) {
	raw, ok := c.getTopLevelEntry("cert-required")
	if ok {
		out, ok = raw.(bool)
	}
	return
}
