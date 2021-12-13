package storkutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests function converting an address to CIDR.
func TestMakeCIDR(t *testing.T) {
	cidr, err := MakeCIDR("192.0.2.123")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.123/32", cidr)

	cidr, err = MakeCIDR("192.0.2.0/24")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.0/24", cidr)

	cidr, err = MakeCIDR("2001:db8:1::1")
	require.NoError(t, err)
	require.Equal(t, "2001:db8:1::1/128", cidr)

	cidr, err = MakeCIDR("2001:db8:1::/64")
	require.NoError(t, err)
	require.Equal(t, "2001:db8:1::/64", cidr)
}

// Test that IP address or prefix can be parsed.
func TestParseIP(t *testing.T) {
	parsedIP := ParseIP("192.0.2.0/24")
	require.NotNil(t, parsedIP)
	require.Equal(t, IPv4, parsedIP.Protocol)
	require.Equal(t, "192.0.2.0/24", parsedIP.NetworkAddress)
	require.Equal(t, "192.0.2.0", parsedIP.NetworkPrefix)
	require.EqualValues(t, 24, parsedIP.PrefixLength)
	require.True(t, parsedIP.Prefix)
	require.True(t, parsedIP.CIDR)

	parsedIP = ParseIP("192.0.2.1/32")
	require.NotNil(t, parsedIP)
	require.Equal(t, IPv4, parsedIP.Protocol)
	require.Equal(t, "192.0.2.1", parsedIP.NetworkAddress)
	require.Equal(t, "192.0.2.1", parsedIP.NetworkPrefix)
	require.EqualValues(t, 32, parsedIP.PrefixLength)
	require.False(t, parsedIP.Prefix)
	require.True(t, parsedIP.CIDR)

	parsedIP = ParseIP("192.0.2.1")
	require.NotNil(t, parsedIP)
	require.Equal(t, IPv4, parsedIP.Protocol)
	require.Equal(t, "192.0.2.1", parsedIP.NetworkAddress)
	require.Equal(t, "192.0.2.1", parsedIP.NetworkPrefix)
	require.EqualValues(t, 32, parsedIP.PrefixLength)
	require.False(t, parsedIP.Prefix)
	require.False(t, parsedIP.CIDR)

	parsedIP = ParseIP("2001:db8:1::/48")
	require.NotNil(t, parsedIP)
	require.Equal(t, IPv6, parsedIP.Protocol)
	require.Equal(t, "2001:db8:1::/48", parsedIP.NetworkAddress)
	require.Equal(t, "2001:db8:1::", parsedIP.NetworkPrefix)
	require.EqualValues(t, 48, parsedIP.PrefixLength)
	require.True(t, parsedIP.Prefix)
	require.True(t, parsedIP.CIDR)

	parsedIP = ParseIP("2001:db8:1::/128")
	require.NotNil(t, parsedIP)
	require.Equal(t, IPv6, parsedIP.Protocol)
	require.Equal(t, "2001:db8:1::", parsedIP.NetworkAddress)
	require.Equal(t, "2001:db8:1::", parsedIP.NetworkPrefix)
	require.EqualValues(t, 128, parsedIP.PrefixLength)
	require.False(t, parsedIP.Prefix)
	require.True(t, parsedIP.CIDR)

	parsedIP = ParseIP("2001:db8:1::")
	require.NotNil(t, parsedIP)
	require.Equal(t, IPv6, parsedIP.Protocol)
	require.Equal(t, "2001:db8:1::", parsedIP.NetworkAddress)
	require.Equal(t, "2001:db8:1::", parsedIP.NetworkPrefix)
	require.EqualValues(t, 128, parsedIP.PrefixLength)
	require.False(t, parsedIP.Prefix)
	require.False(t, parsedIP.CIDR)

	require.Nil(t, ParseIP(""))
	require.Nil(t, ParseIP("192.0.2.0/xy"))
	require.Nil(t, ParseIP("192.0.2.0/"))
}

// Test that the IP range in both supported formats is parsed
// correctly.
func TestParseIPRange(t *testing.T) {
	// IPv4 case.
	lb, ub, err := ParseIPRange("192.0.2.10 - 192.0.2.55")
	require.NoError(t, err)
	require.NotNil(t, lb)
	require.NotNil(t, ub)
	require.Equal(t, "192.0.2.10", lb.String())
	require.Equal(t, "192.0.2.55", ub.String())

	// IPv6 case with some odd spacing.
	lb, ub, err = ParseIPRange("2001:db8:1:1::1000 -2001:db8:1:2::EEEE")
	require.NoError(t, err)
	require.NotNil(t, lb)
	require.NotNil(t, ub)
	require.Equal(t, "2001:db8:1:1::1000", lb.String())
	require.Equal(t, "2001:db8:1:2::eeee", ub.String())

	// Check that the range can be specified as prefix.
	lb, ub, err = ParseIPRange("3000:1::/32")
	require.NoError(t, err)
	require.NotNil(t, lb)
	require.NotNil(t, ub)
	require.Equal(t, "3000:1::", lb.String())
	require.Equal(t, "3000:1:ffff:ffff:ffff:ffff:ffff:ffff", ub.String())

	// Two hyphens and 3 addresses is wrong.
	_, _, err = ParseIPRange("192.0.2.0-192.0.2.100-192.0.3.100")
	require.Error(t, err)

	// No upper bound.
	_, _, err = ParseIPRange("192.0.2.0- ")
	require.Error(t, err)

	// Mix of IPv4 and IPv6 is wrong.
	_, _, err = ParseIPRange("192.0.2.0-2001:db8:1::100")
	require.Error(t, err)
}

// Test that it can be determined whether an IPv4 address is within
// the range.
func TestIPv4InRange(t *testing.T) {
	// 192.0.2.100 is within the range of 192.0.2.10 - 192.0.2.200.
	parsedIP := ParseIP("192.0.2.100")
	lb, ub, err := ParseIPRange("192.0.2.10 - 192.0.2.200")
	require.NoError(t, err)
	require.True(t, parsedIP.IsInRange(lb, ub))

	// 192.0.2.201 is off by one (out of range).
	parsedIP = ParseIP("192.0.2.201")
	require.NoError(t, err)
	require.False(t, parsedIP.IsInRange(lb, ub))

	// 192.0.2.9 is also off by one.
	parsedIP = ParseIP("192.0.2.9")
	require.NoError(t, err)
	require.False(t, parsedIP.IsInRange(lb, ub))

	// IPv6 address is always out of an IPv4 range.
	parsedIP = ParseIP("2001:db8:1::1")
	require.NoError(t, err)
	require.False(t, parsedIP.IsInRange(lb, ub))
}

// Test that it can be determined whether an IPv6 address is within
// the range.
func TestIPv6InRange(t *testing.T) {
	// 2001:db8:1::164 is within the range 2001:db8:1::100 - 2001:db8:1::200.
	parsedIP := ParseIP("2001:db8:1::164")
	lb, ub, err := ParseIPRange("2001:db8:1::100 - 2001:db8:1::200")
	require.NoError(t, err)
	require.True(t, parsedIP.IsInRange(lb, ub))

	// This address is above the upper bound.
	parsedIP = ParseIP("2001:db8:1::ffff:ffff")
	require.NoError(t, err)
	require.False(t, parsedIP.IsInRange(lb, ub))

	// This address is below the lower bound.
	parsedIP = ParseIP("2001:db8:1::")
	require.NoError(t, err)
	require.False(t, parsedIP.IsInRange(lb, ub))

	// IPv4 address is always out of an IPv6 range.
	parsedIP = ParseIP("192.0.2.1")
	lb, ub, err = ParseIPRange("2001:db8:1::100 - 2001:db8:1::200")
	require.NoError(t, err)
	require.False(t, parsedIP.IsInRange(lb, ub))

	// Prefix is always out of range of addresses.
	parsedIP = ParseIP("2001:db8:1::/120")
	require.NoError(t, err)
	lb, ub, err = ParseIPRange("2001:db8:1:: - 2001:db8:1::ffff")
	require.NoError(t, err)
	require.False(t, parsedIP.IsInRange(lb, ub))
}

// Test that it can be determined whether a prefix is within the
// prefix length specified using the prefix, prefix length and the
// delegated length.
func TestPrefixInRange(t *testing.T) {
	// Delegated prefix matches the container prefix.
	parsedIP := ParseIP("2001:db8:1::/96")
	require.True(t, parsedIP.IsInPrefixRange("2001:db8:1::", 64, 96))

	// The delegated lengths match but the prefix length doesn't.
	// In fact it is invalid because it is higher than the delegated
	// prefix length.
	parsedIP = ParseIP("2001:db8:1::/64")
	require.False(t, parsedIP.IsInPrefixRange("2001:db8:1::", 80, 64))

	// The prefixes don't match.
	parsedIP = ParseIP("2001:db8:2::/96")
	require.False(t, parsedIP.IsInPrefixRange("2001:db8:1::", 64, 96))

	// Prefix is in range.
	parsedIP = ParseIP("2001:db8:1:0:2::/96")
	require.True(t, parsedIP.IsInPrefixRange("2001:db8:1::", 64, 96))

	// An IP address is always out of the prefix range.
	parsedIP = ParseIP("2001:db8:1:0:2::")
	require.False(t, parsedIP.IsInPrefixRange("2001:db8:1::", 64, 96))
}
