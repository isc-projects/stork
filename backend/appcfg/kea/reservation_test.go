package keaconfig

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

// Test host returning static values and implementing Host interface.
type TestHost struct{}

// Returns static host identifiers.
func (host TestHost) GetHostIdentifiers() []struct {
	Type  string
	Value []byte
} {
	return []struct {
		Type  string
		Value []byte
	}{
		{
			Type:  "hw-address",
			Value: []byte{1, 2, 3, 4, 5, 6},
		},
		{
			Type:  "duid",
			Value: []byte{2, 2, 2, 2, 2, 2},
		},
		{
			Type:  "circuit-id",
			Value: []byte{1, 1, 1, 1, 1, 1},
		},
		{
			Type:  "client-id",
			Value: []byte{1, 2, 3, 4},
		},
		{
			Type:  "flex-id",
			Value: []byte{9, 9, 9, 9},
		},
	}
}

// Returns static IP reservation of various kinds.
func (host TestHost) GetIPReservations() []string {
	return []string{"2001:db8:1::1", "3000::/16", "2001:db8:2::2", "3001::/16", "192.0.2.1", "10.0.0.1"}
}

// Returns static hostname.
func (host TestHost) GetHostname() string {
	return "hostname.example.org"
}

// Returns static subnet ID.
func (host TestHost) GetSubnetID(int64) (int64, error) {
	return int64(123), nil
}

// Returns static DHCP options.
func (host TestHost) GetDHCPOptions(int64) (options []DHCPOption) {
	testOptions := []testDHCPOption{
		{
			code:        5,
			encapsulate: "dhcp4",
			fields: []testDHCPOptionField{
				*newTestDHCPOptionField("ipv4-address", "192.0.2.1"),
			},
		},
		{
			code:        7,
			encapsulate: "dhcp4",
			fields: []testDHCPOptionField{
				*newTestDHCPOptionField("ipv4-address", "10.0.0.1"),
			},
		},
	}
	for _, to := range testOptions {
		options = append(options, to)
	}
	return
}

// Test conversion of the host to Kea reservation.
func TestCreateReservation(t *testing.T) {
	var (
		lookup testDHCPOptionDefinitionLookup
		host   TestHost
	)
	reservation, err := CreateReservation(1, lookup, host)
	require.NoError(t, err)
	require.NotNil(t, reservation)
	require.Equal(t, "010203040506", reservation.HWAddress)
	require.Equal(t, "020202020202", reservation.DUID)
	require.Equal(t, "010101010101", reservation.CircuitID)
	require.Equal(t, "01020304", reservation.ClientID)
	require.Equal(t, "09090909", reservation.FlexID)
	require.Equal(t, "192.0.2.1", reservation.IPAddress)
	require.Len(t, reservation.IPAddresses, 2)
	require.Equal(t, "2001:db8:1::1", reservation.IPAddresses[0])
	require.Equal(t, "2001:db8:2::2", reservation.IPAddresses[1])
	require.Len(t, reservation.Prefixes, 2)
	require.Equal(t, "3000::/16", reservation.Prefixes[0])
	require.Equal(t, "3001::/16", reservation.Prefixes[1])
	require.Equal(t, "hostname.example.org", reservation.Hostname)
	require.Len(t, reservation.OptionData, 2)
}

// Test conversion of the host to Kea reservation that can be used
// in host_cmds command.
func TestCreateHostCmdsReservation(t *testing.T) {
	var (
		lookup testDHCPOptionDefinitionLookup
		host   TestHost
	)
	reservation, err := CreateHostCmdsReservation(1, lookup, host)
	require.NoError(t, err)
	require.NotNil(t, reservation)
	require.Equal(t, "010203040506", reservation.HWAddress)
	require.Equal(t, "192.0.2.1", reservation.IPAddress)
	require.Len(t, reservation.IPAddresses, 2)
	require.Equal(t, "2001:db8:1::1", reservation.IPAddresses[0])
	require.Equal(t, "2001:db8:2::2", reservation.IPAddresses[1])
	require.Len(t, reservation.Prefixes, 2)
	require.Equal(t, "3000::/16", reservation.Prefixes[0])
	require.Equal(t, "3001::/16", reservation.Prefixes[1])
	require.Equal(t, "hostname.example.org", reservation.Hostname)
	require.EqualValues(t, 123, reservation.SubnetID)
}
