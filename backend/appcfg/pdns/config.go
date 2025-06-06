package pdnsconfig

// Config represents a parsed PowerDNS configuration.
type Config struct {
	values map[string][]ParsedValue
}

// Instantiates the configuration from a map of key/values pairs.
func newConfig(values map[string][]ParsedValue) *Config {
	return &Config{
		values: values,
	}
}

// Returns a boolean value for a key. If the key is not found or it is
// not a boolean, nil is returned.
func (c *Config) GetBool(key string) *bool {
	if value, ok := c.values[key]; ok && len(value) > 0 {
		return value[0].GetBool()
	}
	return nil
}

// Returns an integer value for a key. If the key is not found or it is
// not an integer, nil is returned.
func (c *Config) GetInt64(key string) *int64 {
	if value, ok := c.values[key]; ok && len(value) > 0 {
		return value[0].GetInt64()
	}
	return nil
}

// Returns a string value for a key. If the key is not found or it is
// not a string, nil is returned.
func (c *Config) GetString(key string) *string {
	if value, ok := c.values[key]; ok && len(value) > 0 {
		return value[0].GetString()
	}
	return nil
}

// Returns all values for a key. If the key is not found, an empty
// slice is returned.
func (c *Config) GetValues(key string) []ParsedValue {
	if values, ok := c.values[key]; ok {
		return values
	}
	return []ParsedValue{}
}

// ParsedValue represents a parsed value from a PowerDNS configuration.
// It is one of the values specified after equal sign for a given key.
type ParsedValue struct {
	boolValue   *bool
	int64Value  *int64
	stringValue *string
}

// Returns a boolean value. If the value is not a boolean, nil is returned.
func (v *ParsedValue) GetBool() *bool {
	if v.boolValue != nil {
		return v.boolValue
	}
	return nil
}

// Returns an integer value. If the value is not an integer, nil is returned.
func (v *ParsedValue) GetInt64() *int64 {
	if v.int64Value != nil {
		return v.int64Value
	}
	return nil
}

// Returns a string value. If the value is not a string, nil is returned.
func (v *ParsedValue) GetString() *string {
	if v.stringValue != nil {
		return v.stringValue
	}
	return nil
}
