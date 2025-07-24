package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test getting address and port from an inet clause.
func TestGetAddressAndPort(t *testing.T) {
	inetClause := &InetClause{
		Address: "127.0.0.1",
		Port:    storkutil.Ptr("53"),
	}
	address, port := inetClause.GetAddressAndPort(defaultControlsPort)
	require.Equal(t, "127.0.0.1", address)
	require.EqualValues(t, 53, port)
}

// Test getting address and port from an inet clause with a default port.
func TestGetAddressAndPortDefaultPort(t *testing.T) {
	inetClause := &InetClause{
		Address: "192.0.2.1",
	}
	address, port := inetClause.GetAddressAndPort(int64(333))
	require.Equal(t, "192.0.2.1", address)
	require.EqualValues(t, 333, port)
}

// Test getting address and port from an inet clause with a default port
// and an asterisk.
func TestGetAddressAndPortDefaultPortAsterisk(t *testing.T) {
	inetClause := &InetClause{
		Address: "192.0.2.1",
		Port:    storkutil.Ptr("*"),
	}
	address, port := inetClause.GetAddressAndPort(int64(444))
	require.Equal(t, "192.0.2.1", address)
	require.EqualValues(t, 444, port)
}

// Test getting address and port from an inet clause with a non-numeric port.
func TestGetAddressAndPortNonNumericPort(t *testing.T) {
	inetClause := &InetClause{
		Address: "192.0.2.1",
		Port:    storkutil.Ptr("53a"),
	}
	address, port := inetClause.GetAddressAndPort(int64(444))
	require.Equal(t, "192.0.2.1", address)
	require.EqualValues(t, 444, port)
}
