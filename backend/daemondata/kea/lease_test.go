package keadata

import (
	"testing"

	require "github.com/stretchr/testify/require"

	agentapi "isc.org/stork/api"
	storkutil "isc.org/stork/util"
)

func TestNewLease4(t *testing.T) {
	// Act
	lease := NewLease4(
		"127.0.0.1",
		"00:00:00:00:00:00",
		"",
		1,
		2,
		3,
		false,
		true,
		"host.example",
		3,
		map[string]any{
			"potato": "mashed",
		},
	)

	leaseWithClientID := NewLease4(
		"127.0.0.2",
		"",
		"01:01:01:01:01:01:01:01",
		1,
		2,
		3,
		true,
		false,
		"host2.example",
		3,
		nil,
	)

	// Assert
	require.Equal(t, storkutil.IPv4, lease.Family)
	require.Equal(t, uint64(1), lease.CLTT)
	require.Nil(t, lease.DUID)
	require.Equal(t, "00:00:00:00:00:00", lease.HWAddress)
	require.Equal(t, "", lease.ClientID.String())
	require.Equal(t, uint8(0), lease.PrefixLength)
	require.Equal(t, LeaseStateReleased, lease.State)
	require.Equal(t, uint32(3), lease.LocalSubnetID)
	require.Equal(t, uint32(2), lease.ValidLifetime)
	require.Equal(t, false, lease.FqdnFwd)
	require.Equal(t, true, lease.FqdnRev)
	require.Equal(t, "host.example", lease.Hostname)
	require.Equal(t, map[string]any{"potato": "mashed"}, lease.UserContext)

	require.Equal(t, "", leaseWithClientID.HWAddress)
	require.Equal(t, "01:01:01:01:01:01:01:01", leaseWithClientID.ClientID.String())
}

func TestNewLease6(t *testing.T) {
	// Act
	hwtype := uint32(4)
	lease := NewLease6(
		"::1",
		"00:00:00:00:00:00:00:00",
		"IA_NA",
		6,
		7,
		8,
		9,
		10,
		64,
		false,
		true,
		"host3.example",
		"00:11:22:33:44:55",
		2,
		map[string]any{"potato": "boiled"},
		&hwtype,
		"DOCSIS MODEM",
	)

	// Assert
	require.Equal(t, storkutil.IPv6, lease.Family)
	require.Nil(t, lease.ClientID)
	require.Equal(t, uint64(6), lease.CLTT)
	require.Equal(t, "00:00:00:00:00:00:00:00", lease.DUID.String())
	require.Equal(t, uint8(64), lease.PrefixLength)
	require.Equal(t, LeaseStateExpiredReclaimed, lease.State)
	require.Equal(t, uint32(8), lease.LocalSubnetID)
	require.Equal(t, uint32(7), lease.ValidLifetime)
	require.Equal(t, false, lease.FqdnFwd)
	require.Equal(t, true, lease.FqdnRev)
	require.Equal(t, "host3.example", lease.Hostname)
	require.Equal(t, "00:11:22:33:44:55", lease.HWAddress)
	require.Equal(t, map[string]any{"potato": "boiled"}, lease.UserContext)
	require.EqualValues(t, 4, *lease.HWType)
	require.Equal(t, "DOCSIS MODEM", lease.HWAddressSource)
}

func TestToGRPC(t *testing.T) {
	// Arrange
	duid := "00:01:02:03:04:05:06:07"
	clientID := "09:08:07:06:05:04:03"
	hwtype := uint32(6)
	input := Lease{
		Type:            "IA_PD",
		Family:          storkutil.IPv6,
		IPAddress:       "fe80::7",
		HWAddress:       "00:11:22:33:44:55",
		DUID:            NewColonSepHexStr(&duid),
		CLTT:            100,
		ValidLifetime:   3600,
		LocalSubnetID:   9,
		State:           0,
		PrefixLength:    64,
		ClientID:        NewColonSepHexStr(&clientID),
		FqdnFwd:         false,
		FqdnRev:         true,
		IAID:            5,
		Hostname:        "host4.example",
		HWType:          &hwtype,
		HWAddressSource: "Subscriber ID",
		UserContext:     map[string]any{"potato": "fried"},
	}

	// Act
	result := input.ToGRPC()

	// Assert
	require.Equal(t, "IA_PD", result.Type)
	require.Equal(t, agentapi.Lease_IPAddrFamily(storkutil.IPv6), result.Family)
	require.Equal(t, input.IPAddress, result.IpAddress)
	require.Equal(t, input.DUID.String(), result.Duid)
	require.Equal(t, uint64(input.ValidLifetime), result.ValidLifetime)
	require.Equal(t, input.LocalSubnetID, result.SubnetID)
	require.EqualValues(t, input.State, result.State)
	require.Equal(t, uint32(input.PrefixLength), result.PrefixLen)
	require.Equal(t, input.ClientID.String(), result.ClientID)
	require.Equal(t, false, result.FqdnFwd)
	require.Equal(t, true, result.FqdnRev)
	require.Equal(t, uint32(5), result.Iaid)
	require.Equal(t, "host4.example", result.Hostname)
	require.EqualValues(t, 6, *result.HwType)
	require.Equal(t, "Subscriber ID", result.HwAddressSource)
	require.Equal(t, "{\"potato\":\"fried\"}", result.UserContext)
}
