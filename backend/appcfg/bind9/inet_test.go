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
	address, port := inetClause.GetConnectableAddressAndPort(defaultControlsPort)
	require.Equal(t, "127.0.0.1", address)
	require.EqualValues(t, 53, port)
}

// Test getting address and port from an inet clause with a default port.
func TestGetAddressAndPortDefaultPort(t *testing.T) {
	inetClause := &InetClause{
		Address: "192.0.2.1",
	}
	address, port := inetClause.GetConnectableAddressAndPort(int64(333))
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
	address, port := inetClause.GetConnectableAddressAndPort(int64(444))
	require.Equal(t, "192.0.2.1", address)
	require.EqualValues(t, 444, port)
}

// Test getting address and port from an inet clause with a non-numeric port.
func TestGetAddressAndPortNonNumericPort(t *testing.T) {
	inetClause := &InetClause{
		Address: "192.0.2.1",
		Port:    storkutil.Ptr("53a"),
	}
	address, port := inetClause.GetConnectableAddressAndPort(int64(444))
	require.Equal(t, "192.0.2.1", address)
	require.EqualValues(t, 444, port)
}

// Test that the 127.0.0.1 is returned when asterisk is specified.
func TestGetAddressAndPortAsterisk(t *testing.T) {
	inetClause := &InetClause{
		Address: "*",
	}
	address, port := inetClause.GetConnectableAddressAndPort(int64(444))
	require.Equal(t, "127.0.0.1", address)
	require.EqualValues(t, 444, port)
}

// Test that the 127.0.0.1 is returned when IPv4 zero address is specified.
func TestGetAddressAndPortIPv4Zero(t *testing.T) {
	inetClause := &InetClause{
		Address: "0.0.0.0",
	}
	address, port := inetClause.GetConnectableAddressAndPort(int64(444))
	require.Equal(t, "127.0.0.1", address)
	require.EqualValues(t, 444, port)
}

// Test that the ::1 is returned when IPv6 zero address is specified.
func TestGetAddressAndPortIPv6Zero(t *testing.T) {
	inetClause := &InetClause{
		Address: "::",
	}
	address, port := inetClause.GetConnectableAddressAndPort(int64(444))
	require.Equal(t, "::1", address)
	require.EqualValues(t, 444, port)
}

// Test that the full inet clause is formatted correctly.
func TestInetClauseFormat(t *testing.T) {
	inetClause := &InetClause{
		Address: "127.0.0.1",
		Port:    storkutil.Ptr("53"),
		Allow: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					KeyID: "guest",
				},
			},
		},
		Keys: &Keys{
			KeyNames: []string{"trusted"},
		},
		ReadOnly: storkutil.Ptr(Boolean(true)),
	}
	output := inetClause.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `inet "127.0.0.1" port 53 allow { key "guest"; } keys { "trusted"; } read-only true;`, output)
}

// Test that the inet clause is formatted correctly when no optional flags are specified.
func TestInetClauseFormatNoOptionalFlags(t *testing.T) {
	inetClause := &InetClause{
		Address: "127.0.0.1",
	}
	output := inetClause.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `inet "127.0.0.1";`, output)
}

// Test that serializing an inet clause with nil values does not panic.
func TestInetClauseFormatNilValues(t *testing.T) {
	inetClause := &InetClause{}
	require.NotPanics(t, func() { inetClause.getFormattedOutput(nil) })
}
