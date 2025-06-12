package pdnsconfig

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test getting webserver configuration when it is enabled.
func TestGetWebserverConfig(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		api = yes
		webserver = yes
		webserver-address = 192.0.2.1
		webserver-port = 8082
	`))
	require.NoError(t, err)

	address, port, ok := cfg.GetWebserverConfig()
	require.True(t, ok)
	require.NotNil(t, address)
	require.Equal(t, "192.0.2.1", *address)
	require.NotNil(t, port)
	require.EqualValues(t, 8082, *port)
}

// Test getting webserver configuration when API is disabled.
func TestGetWebserverAPIDisabled(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		api = no
		webserver = yes
		webserver-address = 192.0.2.1
		webserver-port = 8082
	`))
	require.NoError(t, err)
	address, port, ok := cfg.GetWebserverConfig()
	require.False(t, ok)
	require.Nil(t, address)
	require.Nil(t, port)
}

// Test getting webserver configuration when webserver is disabled.
func TestGetWebserverWebserverDisabled(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		api = yes
		webserver = no
		webserver-address = 192.0.2.1
		webserver-port = 8082
	`))
	require.NoError(t, err)
	address, port, ok := cfg.GetWebserverConfig()
	require.False(t, ok)
	require.Nil(t, address)
	require.Nil(t, port)
}

// Test getting webserver configuration when webserver address is zero. It should
// return the localhost address.
func TestGetWebserverConfigZeroIPv4Address(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		api = yes
		webserver = yes
		webserver-address = 0.0.0.0
		webserver-port = 8082
	`))
	require.NoError(t, err)
	address, port, ok := cfg.GetWebserverConfig()
	require.True(t, ok)
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1", *address)
	require.NotNil(t, port)
	require.EqualValues(t, 8082, *port)
}

// Test getting webserver configuration when webserver address is zero. It should
// return the IPv6 localhost address.
func TestGetWebserverConfigZeroIPv6Address(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		api = yes
		webserver = yes
		webserver-address = ::
		webserver-port = 8082
	`))
	require.NoError(t, err)
	address, port, ok := cfg.GetWebserverConfig()
	require.True(t, ok)
	require.NotNil(t, address)
	require.Equal(t, "::1", *address)
	require.NotNil(t, port)
	require.EqualValues(t, 8082, *port)
}

// Test getting webserver configuration when webserver address is invalid. It
// should return the localhost address and the default port.
func TestGetWebserverConfigZeroInvalidAddressPort(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		api = yes
		webserver = yes
		webserver-address = invalid
		webserver-port = invalid
	`))
	require.NoError(t, err)
	address, port, ok := cfg.GetWebserverConfig()
	require.True(t, ok)
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1", *address)
	require.NotNil(t, port)
	require.EqualValues(t, 8081, *port)
}

// Test getting webserver configuration when webserver address and port are not
// specified. It should return the localhost address and the default port.
func TestGetWebserverConfigDefaults(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		api
		webserver
	`))
	require.NoError(t, err)
	address, port, ok := cfg.GetWebserverConfig()
	require.True(t, ok)
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1", *address)
	require.NotNil(t, port)
	require.EqualValues(t, 8081, *port)
}
