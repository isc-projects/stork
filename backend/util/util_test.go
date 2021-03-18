package storkutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that HostWithPort function generates proper output.
func TestHostWithPortURL(t *testing.T) {
	require.Equal(t, "http://localhost:1000/", HostWithPortURL("localhost", 1000))
	require.Equal(t, "http://192.0.2.0:1/", HostWithPortURL("192.0.2.0", 1))
}

// Test parsing URL into host and port.
func TestParseURL(t *testing.T) {
	host, port := ParseURL("https://xyz:8080/")
	require.Equal(t, "xyz", host)
	require.EqualValues(t, 8080, port)

	host, port = ParseURL("https://[2001:db8:1::]:8080")
	require.Equal(t, "2001:db8:1::", host)
	require.EqualValues(t, 8080, port)

	host, port = ParseURL("http://host.example.org/")
	require.Equal(t, "host.example.org", host)
	require.Zero(t, port)
}

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
	parsed, prefix, ok := ParseIP("192.0.2.0/24")
	require.Equal(t, "192.0.2.0/24", parsed)
	require.True(t, prefix)
	require.True(t, ok)

	parsed, prefix, ok = ParseIP("192.0.2.1/32")
	require.Equal(t, "192.0.2.1", parsed)
	require.False(t, prefix)
	require.True(t, ok)

	parsed, prefix, ok = ParseIP("2001:db8:1::/48")
	require.Equal(t, "2001:db8:1::/48", parsed)
	require.True(t, prefix)
	require.True(t, ok)

	parsed, prefix, ok = ParseIP("2001:db8:1::/128")
	require.Equal(t, "2001:db8:1::", parsed)
	require.False(t, prefix)
	require.True(t, ok)
}

// Test conversion of a string consisting of a string of hexadecimal
// digits with and without whitespace and with and without colons
// is successful. Also test that conversion of a string having
// invalid format is unsuccessful.
func TestFormatMACAddress(t *testing.T) {
	// Whitespace.
	formatted, ok := FormatMACAddress("01 02 03 04 05 06")
	require.True(t, ok)
	require.Equal(t, "01:02:03:04:05:06", formatted)

	// Correct format already.
	formatted, ok = FormatMACAddress("01:02:03:04:05:06")
	require.True(t, ok)
	require.Equal(t, "01:02:03:04:05:06", formatted)

	// No separator.
	formatted, ok = FormatMACAddress("aabbccddeeff")
	require.True(t, ok)
	require.Equal(t, "aa:bb:cc:dd:ee:ff", formatted)

	// Non-hexadecimal digits present.
	_, ok = FormatMACAddress("ab:cd:ef:gh")
	require.False(t, ok)

	// Invalid separator.
	_, ok = FormatMACAddress("01,02,03,04,05,06")
	require.False(t, ok)
}

// Test detection whether the text comprises an identifier
// consisting of hexadecimal digits and optionally a whitespace
// or colons.
func TestIsHexIdentifier(t *testing.T) {
	require.True(t, IsHexIdentifier("01:02:03"))
	require.True(t, IsHexIdentifier("01 e2 03"))
	require.True(t, IsHexIdentifier("abcdef "))
	require.True(t, IsHexIdentifier("12"))
	require.True(t, IsHexIdentifier(" abcd:ef"))
	require.False(t, IsHexIdentifier(" "))
	require.False(t, IsHexIdentifier("1234gh"))
	require.False(t, IsHexIdentifier("12:56:"))
	require.False(t, IsHexIdentifier("12:56:9"))
	require.False(t, IsHexIdentifier("ab,cd"))
	require.False(t, IsHexIdentifier("ab: cd"))
}

// Check if BytesToHex works.
func TestBytesToHex(t *testing.T) {
	bytesArray := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	str := BytesToHex(bytesArray)
	require.Equal(t, "0102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F20", str)
}
