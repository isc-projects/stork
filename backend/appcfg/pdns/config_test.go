package pdnsconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test getting a parsed boolean value.
func TestConfigGetBool(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-enabled": {
			{
				boolValue: storkutil.Ptr(true),
			},
			{
				stringValue: storkutil.Ptr("foo"),
			},
		},
	})
	require.NotNil(t, config)

	t.Run("get existing key", func(t *testing.T) {
		enabled := config.GetBool("api-enabled")
		require.NotNil(t, enabled)
		require.True(t, *enabled)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		enabled := config.GetBool("non-existing-key")
		require.Nil(t, enabled)
	})
}

// Test that nil value is returned when there is no boolean value
// for a key.
func TestConfigGetBoolEmpty(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-enabled": {},
	})
	require.NotNil(t, config)

	enabled := config.GetBool("api-enabled")
	require.Nil(t, enabled)
}

// Test getting a parsed integer value.
func TestConfigGetInt64(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-port": {
			{
				int64Value: storkutil.Ptr(int64(8080)),
			},
			{
				stringValue: storkutil.Ptr("foo"),
			},
		},
	})
	require.NotNil(t, config)

	t.Run("get existing key", func(t *testing.T) {
		port := config.GetInt64("api-port")
		require.NotNil(t, port)
		require.Equal(t, int64(8080), *port)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		port := config.GetInt64("non-existing-key")
		require.Nil(t, port)
	})
}

// Test that nil value is returned when there is no integer value
// for a key.
func TestConfigGetInt64Empty(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-port": {},
	})
	require.NotNil(t, config)

	port := config.GetInt64("api-port")
	require.Nil(t, port)
}

// Test getting a parsed string value.
func TestConfigGetString(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-key": {
			{
				stringValue: storkutil.Ptr("stork"),
			},
			{
				stringValue: storkutil.Ptr("foo"),
			},
		},
	})
	require.NotNil(t, config)

	t.Run("get existing key", func(t *testing.T) {
		apiKey := config.GetString("api-key")
		require.NotNil(t, apiKey)
		require.Equal(t, "stork", *apiKey)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		apiKey := config.GetString("non-existing-key")
		require.Nil(t, apiKey)
	})
}

// Test that nil value is returned when there is no string value
// for a key.
func TestConfigGetStringEmpty(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-key": {},
	})
	require.NotNil(t, config)

	apiKey := config.GetString("api-key")
	require.Nil(t, apiKey)
}

// Test getting all values for a key.
func TestConfigGetValues(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-key": {
			{
				stringValue: storkutil.Ptr("stork"),
			},
			{
				stringValue: storkutil.Ptr("kea"),
			},
		},
	})
	require.NotNil(t, config)

	values := config.GetValues("api-key")
	require.Len(t, values, 2)
	stork := values[0].GetString()
	require.NotNil(t, stork)
	require.Equal(t, "stork", *stork)
	kea := values[1].GetString()
	require.NotNil(t, kea)
	require.Equal(t, "kea", *kea)
}

// Test that an empty slice is returned when there are no values
// for a key.
func TestConfigGetValuesEmpty(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-key": {},
	})
	require.NotNil(t, config)

	values := config.GetValues("api-key")
	require.Len(t, values, 0)
}

// Test that IPv4 localhost address is returned when the allow-axfr-ips
// parameter contains the 127.0.0.0/8 range.
func TestConfigGetAXFRCredentialsAllowIPv4Range(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"allow-axfr-ips": {
			{
				stringValue: storkutil.Ptr("192.0.2.0/24"),
			},
			{
				stringValue: storkutil.Ptr("127.0.0.0/8"),
			},
			{
				stringValue: storkutil.Ptr("::1"),
			},
		},
	})
	require.NotNil(t, config)
	address, keyName, algorithm, secret, err := config.GetAXFRCredentials("", "example.com")
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1:53", *address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
}

// Test that IPv4 localhost address is returned when the allow-axfr-ips
// parameter contains the 127.0.0.1 address.
func TestConfigGetAXFRCredentialsAllowIPv4Address(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"allow-axfr-ips": {
			{
				stringValue: storkutil.Ptr("127.0.0.1"),
			},
		},
	})
	require.NotNil(t, config)
	address, keyName, algorithm, secret, err := config.GetAXFRCredentials("", "example.com")
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1:53", *address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
}

// Test that IPv4 localhost address is returned when the allow-axfr-ips
// parameter contains the 127.0.0.1 address and the local port is specified.
func TestConfigGetAXFRCredentialsAllowIPv4AddressLocalPort(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"allow-axfr-ips": {
			{
				stringValue: storkutil.Ptr("127.0.0.1"),
			},
		},
		"local-port": {
			{
				int64Value: storkutil.Ptr(int64(5353)),
			},
		},
	})
	require.NotNil(t, config)
	address, keyName, algorithm, secret, err := config.GetAXFRCredentials("", "example.com")
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1:5353", *address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
}

// Test that IPv4 localhost address is returned when the allow-axfr-ips
// parameter is not specified and the local port is specified.
func TestConfigGetAXFRCredentialsNoAllowIPv4AddressLocalPort(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"local-port": {
			{
				int64Value: storkutil.Ptr(int64(5353)),
			},
		},
	})
	require.NotNil(t, config)
	address, keyName, algorithm, secret, err := config.GetAXFRCredentials("", "example.com")
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1:5353", *address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
}

// Test that IPv6 localhost address is returned when the allow-axfr-ips
// parameter contains the ::/120 range.
func TestConfigGetAXFRCredentialsAllowIPv6Range(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"allow-axfr-ips": {
			{
				stringValue: storkutil.Ptr("::/120"),
			},
		},
	})
	require.NotNil(t, config)
	address, keyName, algorithm, secret, err := config.GetAXFRCredentials("", "example.com")
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "[::1]:53", *address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
}

// Test that IPv6 localhost address is returned when the allow-axfr-ips
// parameter contains the ::1 address.
func TestConfigGetAXFRCredentialsAllowIPv6Address(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"allow-axfr-ips": {
			{
				stringValue: storkutil.Ptr("192.0.2.0/24"),
			},
			{
				stringValue: storkutil.Ptr("::1"),
			},
			{
				stringValue: storkutil.Ptr("127.0.0.0/8"),
			},
		},
	})
	require.NotNil(t, config)
	address, keyName, algorithm, secret, err := config.GetAXFRCredentials("", "example.com")
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "[::1]:53", *address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
}

// Test that an error is returned when the disable-axfr parameter is set to
// true.
func TestConfigGetAXFRCredentialsDisableAXFR(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"disable-axfr": {
			{
				boolValue: storkutil.Ptr(true),
			},
		},
	})
	require.NotNil(t, config)
	address, keyName, algorithm, secret, err := config.GetAXFRCredentials("", "example.com")
	require.Error(t, err)
	require.ErrorContains(t, err, "disable-axfr is set to disable zone transfers")
	require.Nil(t, address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
}

// Test returning the API key.
func TestConfigGetAPIKey(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-key": {
			{
				stringValue: storkutil.Ptr("stork"),
			},
		},
	})
	require.NotNil(t, config)
	apiKey := config.GetAPIKey()
	require.Equal(t, "stork", apiKey)
}
