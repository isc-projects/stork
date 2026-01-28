package keadata

import (
	"testing"

	require "github.com/stretchr/testify/require"

	agentapi "isc.org/stork/api"
)

func TestNewLease4(t *testing.T) {
	// Act
	lease := NewLease4(
		"127.0.0.1",
		"00:00:00:00:00:00",
		1,
		2,
		3,
		3,
	)

	// Assert
	require.Equal(t, LeaseIPv4, lease.IPVersion)
	require.Equal(t, "", lease.ClientID)
	require.Equal(t, uint64(1), lease.CLTT)
	require.Equal(t, "", lease.DUID)
	require.Equal(t, "", lease.Hostname)
	require.Equal(t, "00:00:00:00:00:00", lease.HWAddress)
	require.Equal(t, uint8(0), lease.PrefixLength)
	require.Equal(t, 3, lease.State)
	require.Equal(t, uint32(3), lease.SubnetID)
	require.Equal(t, uint32(2), lease.ValidLifetime)
}

func TestNewLease6(t *testing.T) {
	// Act
	lease := NewLease6(
		"::1",
		"00:00:00:00:00:00:00:00",
		6,
		7,
		8,
		2,
		64,
	)

	// Assert
	require.Equal(t, LeaseIPv6, lease.IPVersion)
	require.Equal(t, "", lease.ClientID)
	require.Equal(t, uint64(6), lease.CLTT)
	require.Equal(t, "00:00:00:00:00:00:00:00", lease.DUID)
	require.Equal(t, "", lease.Hostname)
	require.Equal(t, "", lease.HWAddress)
	require.Equal(t, uint8(64), lease.PrefixLength)
	require.Equal(t, 2, lease.State)
	require.Equal(t, uint32(8), lease.SubnetID)
	require.Equal(t, uint32(7), lease.ValidLifetime)
}

func TestToGRPC(t *testing.T) {
	// Arrange
	input := Lease{
		IPVersion:     LeaseIPv6,
		IPAddress:     "fe80::7",
		DUID:          "00:01:02:03:04:05:06:07",
		CLTT:          100,
		ValidLifetime: 3600,
		SubnetID:      9,
		State:         0,
		PrefixLength:  64,
	}

	// Act
	result := input.ToGRPC()

	// Assert
	require.Equal(t, agentapi.Lease_IPVersion(LeaseIPv6), result.IpVersion)
	require.Equal(t, input.IPAddress, result.IpAddress)
	require.Equal(t, input.DUID, result.Duid)
	require.Equal(t, uint64(input.ValidLifetime), result.ValidLifetime)
	require.Equal(t, input.SubnetID, result.SubnetID)
	require.Equal(t, uint32(input.State), result.State)
	require.Equal(t, uint32(input.PrefixLength), result.PrefixLen)
	require.Empty(t, result.HwAddress)
}
