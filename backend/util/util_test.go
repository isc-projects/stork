package storkutil

import (
	"encoding/json"
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

// Test conversion from hex to bytes.
func TestHexToBytes(t *testing.T) {
	require.EqualValues(t, HexToBytes("00:01:02:03:04:05:06"), []byte{0, 1, 2, 3, 4, 5, 6})
	require.EqualValues(t, HexToBytes("ffeeaa"), []byte{0xff, 0xee, 0xaa})
	require.Empty(t, HexToBytes("dog"))
}

// Test read a configuration without import statements.
func TestReadConfigurationWithoutIncludes(t *testing.T) {
	path := "configs/config-without-includes.json"
	raw, err := ReadConfigurationWithIncludes(path)
	require.NoError(t, err)

	var content interface{}
	json.Unmarshal([]byte(raw), &content)
	data := content.(map[string]interface{})

	require.Contains(t, data, "foo", "bar", "baz")
	foo := int(data["foo"].(float64))
	require.EqualValues(t, foo, 42)
	bar := data["bar"].(string)
	require.EqualValues(t, bar, "24")
	baz := data["baz"].(bool)
	require.EqualValues(t, baz, true)
}

// Test read a configuration with include statements.
func TestReadConfigurationWithIncludes(t *testing.T) {
	path := "configs/config-with-includes.json"
	raw, err := ReadConfigurationWithIncludes(path)
	require.NoError(t, err)

	var content interface{}
	json.Unmarshal([]byte(raw), &content)
	data := content.(map[string]interface{})
	require.Contains(t, data, "biz", "buz", "boz")

	// Non-imported content
	biz := data["biz"].(string)
	require.EqualValues(t, biz, "zib")
	boz := data["boz"].(string)
	require.EqualValues(t, boz, "zob")

	// Imported content
	buz := data["buz"].(map[string]interface{})
	require.Contains(t, buz, "foo", "bar", "baz")
	foo := int(buz["foo"].(float64))
	require.EqualValues(t, foo, 42)
	bar := buz["bar"].(string)
	require.EqualValues(t, bar, "24")
	baz := buz["baz"].(bool)
	require.EqualValues(t, baz, true)
}

// Test read a configuration with nested import statements.
func TestReadConfigurationWithNestedIncludes(t *testing.T) {
	path := "configs/config-with-nested-includes.json"
	raw, err := ReadConfigurationWithIncludes(path)
	require.NoError(t, err)

	var content interface{}
	json.Unmarshal([]byte(raw), &content)
	data := content.(map[string]interface{})
	require.Contains(t, data, "ban")

	// Non-imported content
	ban := data["ban"].([]interface{})
	require.EqualValues(t, len(ban), 5)
	require.Equal(t, ban[0], float64(0))
	require.Equal(t, ban[1], float64(1))
	require.Equal(t, ban[2], float64(2))
	require.Equal(t, ban[4], float64(4))

	// 1-level nesting
	firstNest := ban[3].(map[string]interface{})
	require.Contains(t, firstNest, "biz", "buz", "boz")
	biz := firstNest["biz"].(string)
	require.EqualValues(t, biz, "zib")
	boz := firstNest["boz"].(string)
	require.EqualValues(t, boz, "zob")

	// 2-level nesting
	buz := firstNest["buz"].(map[string]interface{})
	require.Contains(t, buz, "foo", "bar", "baz")
	foo := int(buz["foo"].(float64))
	require.EqualValues(t, foo, 42)
	bar := buz["bar"].(string)
	require.EqualValues(t, bar, "24")
	baz := buz["baz"].(bool)
	require.EqualValues(t, baz, true)
}

// Test read a configuration with an infinite loop.
func TestReadConfigurationWithInfiniteLoop(t *testing.T) {
	path := "configs/config-with-infinite-loop.json"
	raw, err := ReadConfigurationWithIncludes(path)
	require.Empty(t, raw)
	require.Error(t, err)
}

// Test read a configuration with multiple the same import statements.
func TestReadConfigurationWithMultipleTheSameIncludes(t *testing.T) {
	path := "configs/config-with-multiple-the-same-includes.json"
	raw, err := ReadConfigurationWithIncludes(path)
	require.NoError(t, err)

	var content interface{}
	err = json.Unmarshal([]byte(raw), &content)
	require.NoError(t, err)
	data := content.(map[string]interface{})
	require.Contains(t, data, "biz", "buz", "boz")

	for _, key := range []string{"biz", "buz", "boz"} {
		nested := data[key].(map[string]interface{})
		require.Contains(t, nested, "foo", "bar", "baz")
		foo := int(nested["foo"].(float64))
		require.EqualValues(t, foo, 42)
		bar := nested["bar"].(string)
		require.EqualValues(t, bar, "24")
		baz := nested["baz"].(bool)
		require.EqualValues(t, baz, true)
	}
}

// Test read a configuration with an import statement related to a non-existing file.
func TestReadConfigurationWithNonExistingIncludes(t *testing.T) {
	path := "configs/config-with-non-existing-includes.json"
	_, err := ReadConfigurationWithIncludes(path)
	require.Error(t, err)
}
