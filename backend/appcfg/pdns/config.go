package pdnsconfig

import (
	"net"

	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

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

// Gets the target address and the required credentials for the zone transfer.
// Since TSIG keys are not specified in the configuration file, this function
// assumes that the TSIG is disabled for the zone transfers from the local host.
// It checks IP addresses and ranges specified in the allow-axfr-ips. If this
// parameter allows zone transfer from the localhost addresses, the function
// returns 127.0.0.1 or ::1 as a target address. It will return an error if
// AXFR is globally disabled or if the allow-axfr-ips parameter forbids access
// from the localhost addresses. The function ignores the viewName parameter
// until views are supported in PowerDNS and Stork.
func (c *Config) GetAXFRCredentials(viewName string, zoneName string) (address *string, keyName *string, algorithm *string, secret *string, err error) {
	disableAXFR := c.GetBool("disable-axfr")
	if disableAXFR != nil && *disableAXFR {
		// AXFR is globally disabled.
		return nil, nil, nil, nil, errors.Errorf("disable-axfr is set to disable zone transfers")
	}
	allowedIPs := c.GetValues("allow-axfr-ips")
	if allowedIPs == nil {
		// By default, PowerDNS allows AXFR from the localhost.
		return storkutil.Ptr("127.0.0.1"), nil, nil, nil, nil
	}
	for _, value := range allowedIPs {
		allowed := value.GetString()
		if allowed == nil {
			// Invalid value. Get the next one.
			continue
		}
		parsedAllowed := storkutil.ParseIP(*allowed)
		if parsedAllowed == nil {
			// Invalid value.
			continue
		}
		switch {
		// Check for things like 127.0.0.0/8 or 127.0.0.1.
		case parsedAllowed.Prefix && parsedAllowed.IPNet.Contains(net.ParseIP("127.0.0.1")), parsedAllowed.IP.Equal(net.ParseIP("127.0.0.1")):
			return storkutil.Ptr("127.0.0.1"), nil, nil, nil, nil
		// Check for things like ::/120 or ::1.
		case parsedAllowed.Prefix && parsedAllowed.IPNet.Contains(net.ParseIP("::1")), parsedAllowed.IP.Equal(net.ParseIP("::1")):
			return storkutil.Ptr("::1"), nil, nil, nil, nil
		}
	}
	return nil, nil, nil, nil, errors.Errorf("failed to get AXFR credentials for zone %s: allow-axfr-ips allows neither 127.0.0.1 nor ::1", zoneName)
}

// Returns the API key for the statistics channel. This key is included in
// the X-API-Key header.
func (c *Config) GetAPIKey() string {
	if key := c.GetString("api-key"); key != nil {
		return *key
	}
	return ""
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
