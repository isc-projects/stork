package bind9config

// Returns the string representation of the unparsed contents.
func (c *RawContents) GetString() string {
	if c != nil {
		return string(*c)
	}
	return ""
}
