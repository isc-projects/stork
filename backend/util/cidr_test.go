package storkutil

import (
	"math"
	"math/big"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests function converting an address to CIDR.
func TestMakeCIDR(t *testing.T) {
	t.Run("IPv4 address", func(t *testing.T) {
		cidr, err := MakeCIDR("192.0.2.123")
		require.NoError(t, err)
		require.Equal(t, "192.0.2.123/32", cidr)
	})

	t.Run("IPv4 subnet", func(t *testing.T) {
		cidr, err := MakeCIDR("192.0.2.0/24")
		require.NoError(t, err)
		require.Equal(t, "192.0.2.0/24", cidr)
	})

	t.Run("IPv4 zero address", func(t *testing.T) {
		cidr, err := MakeCIDR("0.0.0.0")
		require.NoError(t, err)
		require.Equal(t, "0.0.0.0/32", cidr)
	})

	t.Run("IPv4 zero subnet", func(t *testing.T) {
		cidr, err := MakeCIDR("0.0.0.0/0")
		require.NoError(t, err)
		require.Equal(t, "0.0.0.0/0", cidr)
	})

	t.Run("IPv6 address", func(t *testing.T) {
		cidr, err := MakeCIDR("2001:db8:1::1")
		require.NoError(t, err)
		require.Equal(t, "2001:db8:1::1/128", cidr)
	})

	t.Run("IPv6 subnet", func(t *testing.T) {
		cidr, err := MakeCIDR("2001:db8:1::/64")
		require.NoError(t, err)
		require.Equal(t, "2001:db8:1::/64", cidr)
	})

	t.Run("IPv6 zero address", func(t *testing.T) {
		cidr, err := MakeCIDR("::")
		require.NoError(t, err)
		require.Equal(t, "::/128", cidr)
	})

	t.Run("IPv6 zero subnet", func(t *testing.T) {
		cidr, err := MakeCIDR("::/0")
		require.NoError(t, err)
		require.Equal(t, "::/0", cidr)
	})
}

// Test that IP address or prefix can be parsed.
func TestParseIP(t *testing.T) {
	t.Run("IPv4 subnet", func(t *testing.T) {
		parsedIP := ParseIP("192.0.2.0/24")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv4, parsedIP.Protocol)
		require.Equal(t, "192.0.2.0/24", parsedIP.NetworkAddress)
		require.Equal(t, "192.0.2.0", parsedIP.NetworkPrefix)
		require.EqualValues(t, 24, parsedIP.PrefixLength)
		require.True(t, parsedIP.Prefix)
		require.True(t, parsedIP.CIDR)
	})

	t.Run("IPv4 address with mask", func(t *testing.T) {
		parsedIP := ParseIP("192.0.2.1/32")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv4, parsedIP.Protocol)
		require.Equal(t, "192.0.2.1", parsedIP.NetworkAddress)
		require.Equal(t, "192.0.2.1", parsedIP.NetworkPrefix)
		require.EqualValues(t, 32, parsedIP.PrefixLength)
		require.False(t, parsedIP.Prefix)
		require.True(t, parsedIP.CIDR)
	})

	t.Run("IPv4 address", func(t *testing.T) {
		parsedIP := ParseIP("192.0.2.1")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv4, parsedIP.Protocol)
		require.Equal(t, "192.0.2.1", parsedIP.NetworkAddress)
		require.Equal(t, "192.0.2.1", parsedIP.NetworkPrefix)
		require.EqualValues(t, 32, parsedIP.PrefixLength)
		require.False(t, parsedIP.Prefix)
		require.False(t, parsedIP.CIDR)
	})

	t.Run("IPv4 zero address with mask", func(t *testing.T) {
		parsedIP := ParseIP("0.0.0.0/0")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv4, parsedIP.Protocol)
		require.Equal(t, "0.0.0.0/0", parsedIP.NetworkAddress)
		require.Equal(t, "0.0.0.0", parsedIP.NetworkPrefix)
		require.EqualValues(t, 32, parsedIP.PrefixLength)
		require.True(t, parsedIP.Prefix)
		require.True(t, parsedIP.CIDR)
	})

	t.Run("IPv4 zero address", func(t *testing.T) {
		parsedIP := ParseIP("0.0.0.0")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv4, parsedIP.Protocol)
		require.Equal(t, "0.0.0.0", parsedIP.NetworkAddress)
		require.Equal(t, "0.0.0.0", parsedIP.NetworkPrefix)
		require.EqualValues(t, 32, parsedIP.PrefixLength)
		require.False(t, parsedIP.Prefix)
		require.False(t, parsedIP.CIDR)
	})

	t.Run("IPv6 subnet", func(t *testing.T) {
		parsedIP := ParseIP("2001:db8:1::/48")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv6, parsedIP.Protocol)
		require.Equal(t, "2001:db8:1::/48", parsedIP.NetworkAddress)
		require.Equal(t, "2001:db8:1::", parsedIP.NetworkPrefix)
		require.EqualValues(t, 48, parsedIP.PrefixLength)
		require.True(t, parsedIP.Prefix)
		require.True(t, parsedIP.CIDR)
	})

	t.Run("IPv6 address with mask", func(t *testing.T) {
		parsedIP := ParseIP("2001:db8:1::/128")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv6, parsedIP.Protocol)
		require.Equal(t, "2001:db8:1::", parsedIP.NetworkAddress)
		require.Equal(t, "2001:db8:1::", parsedIP.NetworkPrefix)
		require.EqualValues(t, 128, parsedIP.PrefixLength)
		require.False(t, parsedIP.Prefix)
		require.True(t, parsedIP.CIDR)
	})

	t.Run("IPv6 address", func(t *testing.T) {
		parsedIP := ParseIP("2001:db8:1::")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv6, parsedIP.Protocol)
		require.Equal(t, "2001:db8:1::", parsedIP.NetworkAddress)
		require.Equal(t, "2001:db8:1::", parsedIP.NetworkPrefix)
		require.EqualValues(t, 128, parsedIP.PrefixLength)
		require.False(t, parsedIP.Prefix)
		require.False(t, parsedIP.CIDR)
	})

	t.Run("IPv6 zero address with mask", func(t *testing.T) {
		parsedIP := ParseIP("::/0")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv6, parsedIP.Protocol)
		require.Equal(t, "::/0", parsedIP.NetworkAddress)
		require.Equal(t, "::", parsedIP.NetworkPrefix)
		require.EqualValues(t, 128, parsedIP.PrefixLength)
		require.True(t, parsedIP.Prefix)
		require.True(t, parsedIP.CIDR)
	})

	t.Run("IPv6 zero address", func(t *testing.T) {
		parsedIP := ParseIP("::")
		require.NotNil(t, parsedIP)
		require.Equal(t, IPv6, parsedIP.Protocol)
		require.Equal(t, "::", parsedIP.NetworkAddress)
		require.Equal(t, "::", parsedIP.NetworkPrefix)
		require.EqualValues(t, 128, parsedIP.PrefixLength)
		require.False(t, parsedIP.Prefix)
		require.False(t, parsedIP.CIDR)
	})

	t.Run("empty string", func(t *testing.T) {
		require.Nil(t, ParseIP(""))
	})

	t.Run("invalid IPv4 masks", func(t *testing.T) {
		require.Nil(t, ParseIP("192.0.2.0/xy"))
		require.Nil(t, ParseIP("192.0.2.0/"))
	})
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

// Test that the network prefixes are correctly converted to binary strings.
func TestPrefixToBinaryForValidPrefixes(t *testing.T) {
	// Arrange
	fourZeroOctets := "00000000" + "00000000" + "00000000" + "00000000"
	v4InV6Prefix := fourZeroOctets + fourZeroOctets +
		"00000000" + "00000000" + "11111111" + "11111111"

	data := [][]string{
		{"168.1.2.0/24", v4InV6Prefix + "10101000" + "00000001" + "00000010"},
		{"168.1.0.0/16", v4InV6Prefix + "10101000" + "00000001"},
		{"168.0.0.0/8", v4InV6Prefix + "10101000"},
		{"168.1.2.0/26", v4InV6Prefix + "10101000" + "00000001" + "00000010" + "00"},
		{"168.1.2.1/24", v4InV6Prefix + "10101000" + "00000001" + "00000010"},
		{"3001::/80", "00110000" + "00000001" + fourZeroOctets + fourZeroOctets},
		{"3001::1/80", "00110000" + "00000001" + fourZeroOctets + fourZeroOctets},
	}

	for _, entry := range data {
		prefix := entry[0]
		expected := entry[1]

		t.Run(prefix, func(t *testing.T) {
			// Act
			cidr := ParseIP(prefix)
			actual := cidr.GetNetworkPrefixAsBinary()

			// Assert
			require.EqualValues(t, expected, actual)
		})
	}
}

// Test that the invalid network prefixes are converted to empty strings.
func TestPrefixToBinaryForInvalidPrefixes(t *testing.T) {
	// Arrange
	cidr := ParseIP("192.168.0.0/24")

	data := []struct {
		prefix string
		length int
	}{
		{"foobar", 24},
		{"192..168.0.0", 24},
		{"512.512.0.0", 24},
		{"192.168.0.0", 512},
		{"192.168.0.0", 0},
		{"192.168.0.0", -1},
	}

	for _, entry := range data {
		t.Run(entry.prefix, func(t *testing.T) {
			cidr.NetworkPrefix = entry.prefix
			cidr.PrefixLength = entry.length

			// Act
			actual := cidr.GetNetworkPrefixAsBinary()

			// Assert
			require.Empty(t, actual)
		})
	}
}

// Test that the prefix with length is properly returned.
func TestGetPrefixWithLength(t *testing.T) {
	// Arrange
	prefixes := []string{
		"10.0.0.0/8",
		"42.42.0.0/16",
		"192.168.1.0/24",
		"127.0.0.1/32",
		"fe80::/64",
		"3001:1000::/80",
		"2001::42/128",
	}

	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			cidr := ParseIP(prefix)

			// Act
			returnedPrefix := cidr.GetNetworkPrefixWithLength()

			// Assert
			require.EqualValues(t, returnedPrefix, prefix)
		})
	}
}

// Test that the subnet string is formatted properly.
func TestFormatCIDRNotation(t *testing.T) {
	require.EqualValues(t, "fe80::/64", FormatCIDRNotation("fe80::", 64))
	require.EqualValues(t, "10.0.0.0/8", FormatCIDRNotation("10.0.0.0", 8))
	// The address is not converted to canonical form and the mask isn't
	// validated.
	require.EqualValues(t, "8.8.8.8/4242", FormatCIDRNotation("8.8.8.8", 4242))
}

// Test calculating the size of the IPv4 range with lower bound equals to the
// upper one.
func TestCalculateRangeSizeLowerEqualsToUpperBound(t *testing.T) {
	// Arrange
	lowerBound := net.ParseIP("10.0.0.1")
	upperBound := net.ParseIP("10.0.0.1")

	// Act
	size := CalculateRangeSize(lowerBound, upperBound)

	// Assert
	require.EqualValues(t, big.NewInt(1), size)
}

// Test that the size of the IPv4 range is calculated properly.
func TestCalculateRangeSizeIPv4(t *testing.T) {
	// Arrange
	lowerBound := net.ParseIP("10.0.0.1")
	upperBound := net.ParseIP("10.0.1.0")

	// Act
	size := CalculateRangeSize(lowerBound, upperBound)

	// Assert
	require.EqualValues(t, big.NewInt(256), size)
}

// Test that the size of the range with swapped bounds is not positive.
func TestCalculateRangeSizeSwappedBounds(t *testing.T) {
	// Arrange
	lowerBound := net.ParseIP("10.0.0.10")
	upperBound := net.ParseIP("10.0.0.1")

	// Act
	size := CalculateRangeSize(lowerBound, upperBound)

	// Assert
	require.EqualValues(t, big.NewInt(-8), size)
}

// Test that the size of the IPv6 range is calculated properly.
func TestCalculateRangeSizeIPv6(t *testing.T) {
	// Arrange
	lowerBound := net.ParseIP("fe80::")
	upperBound := net.ParseIP("fe80:0:0:1::")

	// Act
	size := CalculateRangeSize(lowerBound, upperBound)

	// Assert
	require.EqualValues(
		t,
		big.NewInt(0).Add(
			big.NewInt(0).SetUint64(math.MaxUint64),
			big.NewInt(2),
		), size)
}

// Test that the size of the delegated prefix pool is calculated properly.
func TestCalculateDelegatedPrefixRangeSize(t *testing.T) {
	t.Run("delegated length lowers than prefix length", func(t *testing.T) {
		require.Equal(t, big.NewInt(0), CalculateDelegatedPrefixRangeSize(2, 1))
	})

	t.Run("delegated length equals prefix length", func(t *testing.T) {
		require.Equal(t, big.NewInt(1), CalculateDelegatedPrefixRangeSize(42, 42))
	})

	t.Run("small number of delegated prefixes", func(t *testing.T) {
		require.Equal(t,
			big.NewInt(4),
			CalculateDelegatedPrefixRangeSize(42, 44),
		)
	})

	t.Run("big number of delegated prefixes", func(t *testing.T) {
		require.Equal(t,
			big.NewInt(0).Mul(
				big.NewInt(0).Add(
					big.NewInt(0).SetUint64(math.MaxUint64),
					big.NewInt(1),
				),
				big.NewInt(2),
			),
			CalculateDelegatedPrefixRangeSize(1, 66),
		)
	})
}
