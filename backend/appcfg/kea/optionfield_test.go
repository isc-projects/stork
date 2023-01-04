package keaconfig

import (
	"math"
	"testing"

	require "github.com/stretchr/testify/require"
)

// Test parsing various option fields.
func TestParseDHCPOptionField(t *testing.T) {
	field, err := parseDHCPOptionField(BoolField, "true")
	require.NoError(t, err)
	require.Equal(t, BoolField, field.GetFieldType())
	require.Len(t, field.GetValues(), 1)
	require.True(t, field.GetValues()[0].(bool))

	field, err = parseDHCPOptionField(Uint8Field, "123")
	require.NoError(t, err)
	require.Equal(t, Uint8Field, field.GetFieldType())
	require.Len(t, field.GetValues(), 1)
	require.EqualValues(t, 123, field.GetValues()[0].(uint8))

	field, err = parseDHCPOptionField(Uint16Field, "234")
	require.NoError(t, err)
	require.Equal(t, Uint16Field, field.GetFieldType())
	require.Len(t, field.GetValues(), 1)
	require.EqualValues(t, 234, field.GetValues()[0].(uint16))

	field, err = parseDHCPOptionField(Uint32Field, "345")
	require.NoError(t, err)
	require.Equal(t, Uint32Field, field.GetFieldType())
	require.Len(t, field.GetValues(), 1)
	require.EqualValues(t, 345, field.GetValues()[0].(uint32))

	field, err = parseDHCPOptionField(IPv4AddressField, "192.0.2.1")
	require.NoError(t, err)
	require.Equal(t, IPv4AddressField, field.GetFieldType())
	require.Len(t, field.GetValues(), 1)
	require.Equal(t, "192.0.2.1", field.GetValues()[0].(string))

	field, err = parseDHCPOptionField(IPv6AddressField, "2001:db8:1::1")
	require.NoError(t, err)
	require.Equal(t, IPv6AddressField, field.GetFieldType())
	require.Len(t, field.GetValues(), 1)
	require.Equal(t, "2001:db8:1::1", field.GetValues()[0].(string))

	field, err = parseDHCPOptionField(IPv6PrefixField, "3001::/56")
	require.NoError(t, err)
	require.Equal(t, IPv6PrefixField, field.GetFieldType())
	require.Len(t, field.GetValues(), 2)
	require.Equal(t, "3001::", field.GetValues()[0].(string))
	require.Equal(t, 56, field.GetValues()[1].(int))

	field, err = parseDHCPOptionField(FqdnField, "foo.example.org")
	require.NoError(t, err)
	require.Equal(t, FqdnField, field.GetFieldType())
	require.Len(t, field.GetValues(), 1)
	require.Equal(t, "foo.example.org", field.GetValues()[0].(string))

	field, err = parseDHCPOptionField(PsidField, "10/1")
	require.NoError(t, err)
	require.Equal(t, PsidField, field.GetFieldType())
	require.Len(t, field.GetValues(), 2)
	require.EqualValues(t, 10, field.GetValues()[0].(uint16))
	require.EqualValues(t, 1, field.GetValues()[1].(uint8))
}

// Test parsing a boolean option field value.
func TestParseBoolField(t *testing.T) {
	bv, err := parseBoolField("true")
	require.NoError(t, err)
	require.True(t, bv)

	bv, err = parseBoolField("TRUE")
	require.NoError(t, err)
	require.True(t, bv)

	bv, err = parseBoolField("false")
	require.NoError(t, err)
	require.False(t, bv)

	bv, err = parseBoolField("FALSE")
	require.NoError(t, err)
	require.False(t, bv)

	_, err = parseBoolField("foo")
	require.Error(t, err)
}

// Test parsing an uint8 option field value.
func TestParseUint8Field(t *testing.T) {
	iv, err := parseUint8Field("222")
	require.NoError(t, err)
	require.EqualValues(t, 222, iv)

	// Error cases.
	_, err = parseUint8Field("268")
	require.Error(t, err)
	_, err = parseUint8Field("foo")
	require.Error(t, err)
}

// Test parsing an uint16 option field value.
func TestParseUint16Field(t *testing.T) {
	iv, err := parseUint16Field("11222")
	require.NoError(t, err)
	require.EqualValues(t, 11222, iv)

	// Error cases.
	_, err = parseUint16Field("65537")
	require.Error(t, err)
	_, err = parseUint16Field("foo")
	require.Error(t, err)
}

// Test parsing an uint32 option field value.
func TestParseUint32Field(t *testing.T) {
	iv, err := parseUint32Field("3311222")
	require.NoError(t, err)
	require.EqualValues(t, 3311222, iv)

	// Error cases.
	_, err = parseUint32Field("4294967296")
	require.Error(t, err)
	_, err = parseUint32Field("foo")
	require.Error(t, err)
}

// Test parsing IP option field values.
func TestParseIPField(t *testing.T) {
	ip, err := parseIPField("192.0.2.1")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", ip.NetworkAddress)

	ip, err = parseIPField("2001:db8:1::1")
	require.NoError(t, err)
	require.False(t, ip.Prefix)
	require.Equal(t, "2001:db8:1::1", ip.NetworkAddress)
	require.False(t, ip.Prefix)

	ip, err = parseIPField("3000::/56")
	require.NoError(t, err)
	require.True(t, ip.Prefix)
	require.Equal(t, "3000::", ip.NetworkPrefix)
	require.Equal(t, 56, ip.PrefixLength)

	_, err = parseIPField("foo")
	require.Error(t, err)
}

// Test parsing the PSID option field values.
func TestParsePsidField(t *testing.T) {
	psid, psidLen, err := parsePsidField("12/11")
	require.NoError(t, err)
	require.EqualValues(t, 12, psid)
	require.EqualValues(t, 11, psidLen)

	// Error cases.
	_, _, err = parsePsidField("f/11")
	require.Error(t, err)
	_, _, err = parsePsidField("12/a")
	require.Error(t, err)
	_, _, err = parsePsidField("13")
	require.Error(t, err)
	_, _, err = parsePsidField("/13")
	require.Error(t, err)
}

// Test that a binary option field is converted to Kea format successfully.
func TestConvertBinaryField(t *testing.T) {
	// Colons are allowed.
	value, err := convertBinaryField(*newTestDHCPOptionField(BinaryField, "00:01:02:03:04"))
	require.NoError(t, err)
	require.Equal(t, "0001020304", value)

	// Spaces are allowed.
	value, err = convertBinaryField(*newTestDHCPOptionField(BinaryField, "00 01 02 03 04"))
	require.NoError(t, err)
	require.Equal(t, "0001020304", value)

	// No separators are also allowed.
	value, err = convertBinaryField(*newTestDHCPOptionField(BinaryField, "0001020304"))
	require.NoError(t, err)
	require.Equal(t, "0001020304", value)
}

// Test that conversion of a malformed binary option field yields an error.
func TestConvertBinaryFieldMalformed(t *testing.T) {
	// It must have a single value.
	_, err := convertBinaryField(*newTestDHCPOptionField(BinaryField, "010203", "010203"))
	require.Error(t, err)

	// Having no values is wrong.
	_, err = convertBinaryField(*newTestDHCPOptionField(BinaryField))
	require.Error(t, err)

	// Non-hex string.
	_, err = convertBinaryField(*newTestDHCPOptionField(BinaryField, "wrong"))
	require.Error(t, err)

	// Not a string.
	_, err = convertBinaryField(*newTestDHCPOptionField(BinaryField, 525))
	require.Error(t, err)

	// Empty string.
	_, err = convertBinaryField(*newTestDHCPOptionField(BinaryField, ""))
	require.Error(t, err)
}

// Test that a string option field is converted to a hex format.
func TestConvertStringFieldToHex(t *testing.T) {
	value, err := convertStringField(*newTestDHCPOptionField(StringField, "foobar"), false)
	require.NoError(t, err)
	require.Equal(t, "666F6F626172", value)
}

// Test that a string option field is converted to text format.
func TestConvertStringFieldToText(t *testing.T) {
	value, err := convertStringField(*newTestDHCPOptionField(StringField, "foobar"), true)
	require.NoError(t, err)
	require.Equal(t, "foobar", value)
}

// Test that conversion of a malformed string option field yields an error.
func TestConvertStringFieldMalformed(t *testing.T) {
	// It must be a single value.
	_, err := convertStringField(*newTestDHCPOptionField(StringField, "foo", "bar"), false)
	require.Error(t, err)

	// Having no values is wrong.
	_, err = convertStringField(*newTestDHCPOptionField(StringField), false)
	require.Error(t, err)

	// Not a string.
	_, err = convertStringField(*newTestDHCPOptionField(StringField, 123), false)
	require.Error(t, err)

	// Empty string.
	_, err = convertBinaryField(*newTestDHCPOptionField(StringField, ""))
	require.Error(t, err)
}

// Test that a boolean option field is converted to a hex format.
func TestBoolFieldToHex(t *testing.T) {
	// Convert true value.
	value, err := convertBoolField(*newTestDHCPOptionField(BoolField, true), false)
	require.NoError(t, err)
	require.Equal(t, "01", value)

	// Convert false value.
	value, err = convertBoolField(*newTestDHCPOptionField(BoolField, false), false)
	require.NoError(t, err)
	require.Equal(t, "00", value)
}

// Test that a boolean option field is converted to a text format.
func TestBoolFieldToText(t *testing.T) {
	// Convert true value.
	value, err := convertBoolField(*newTestDHCPOptionField(BoolField, true), true)
	require.NoError(t, err)
	require.Equal(t, "true", value)

	// Convert false value.
	value, err = convertBoolField(*newTestDHCPOptionField(BoolField, false), true)
	require.NoError(t, err)
	require.Equal(t, "false", value)
}

// Test that conversion of a malformed boolean option field yields an error.
func TestBoolFieldMalformed(t *testing.T) {
	// It must be a single value.
	_, err := convertBoolField(*newTestDHCPOptionField(BoolField, false, true), false)
	require.Error(t, err)

	// Having no values is wrong.
	_, err = convertBoolField(*newTestDHCPOptionField(BoolField), false)
	require.Error(t, err)

	// Not a boolean value.
	_, err = convertBoolField(*newTestDHCPOptionField(BoolField, 123), false)
	require.Error(t, err)
}

// Test that an uint8 option field is converted to a hex format.
func TestUint8FieldToHex(t *testing.T) {
	value, err := convertUintField(*newTestDHCPOptionField(Uint8Field, 155), false)
	require.NoError(t, err)
	require.Equal(t, "9B", value)
}

// Test that an uint8 option field is converted to a text format.
func TestUint8FieldToText(t *testing.T) {
	value, err := convertUintField(*newTestDHCPOptionField(Uint8Field, 155), true)
	require.NoError(t, err)
	require.Equal(t, "155", value)
}

// Test that conversion of a malformed uint8 option field yields an error.
func TestUint8FieldMalformed(t *testing.T) {
	// It must be a single value.
	_, err := convertUintField(*newTestDHCPOptionField(Uint8Field, 15, 16), false)
	require.Error(t, err)

	// It must be lower than 256.
	_, err = convertUintField(*newTestDHCPOptionField(Uint8Field, 1550), false)
	require.Error(t, err)

	// No value.
	_, err = convertUintField(*newTestDHCPOptionField(Uint8Field), false)
	require.Error(t, err)

	// Not a number.
	_, err = convertUintField(*newTestDHCPOptionField(Uint8Field, "111"), false)
	require.Error(t, err)

	// Floating point number.
	_, err = convertUintField(*newTestDHCPOptionField(Uint8Field, 1.1), false)
	require.Error(t, err)
}

// Test that an uint16 option field is converted to a hex format.
func TestUint16FieldToHex(t *testing.T) {
	value, err := convertUintField(*newTestDHCPOptionField(Uint16Field, 1550), false)
	require.NoError(t, err)
	require.Equal(t, "060E", value)
}

// Test that converted uint16 option field value has 4 digits.
func TestUint16FieldToHexPadding(t *testing.T) {
	value, err := convertUintField(*newTestDHCPOptionField(Uint16Field, 1), false)
	require.NoError(t, err)
	require.Equal(t, "0001", value)
}

// Test that an uint16 option field is converted to a text format.
func TestUint16FieldToText(t *testing.T) {
	value, err := convertUintField(*newTestDHCPOptionField(Uint16Field, 1550), true)
	require.NoError(t, err)
	require.Equal(t, "1550", value)
}

// Test that conversion of a malformed uint16 option field yields an error.
func TestUint16FieldMalformed(t *testing.T) {
	// It must be a single value.
	_, err := convertUintField(*newTestDHCPOptionField(Uint16Field, 150, 1600), false)
	require.Error(t, err)

	// It must be lower or equal max uint16.
	_, err = convertUintField(*newTestDHCPOptionField(Uint16Field, 166535), false)
	require.Error(t, err)

	// No value.
	_, err = convertUintField(*newTestDHCPOptionField(Uint16Field), false)
	require.Error(t, err)

	// Not a number.
	_, err = convertUintField(*newTestDHCPOptionField(Uint16Field, "222"), false)
	require.Error(t, err)
}

// Test that an uint32 option field is converted to a hex format.
func TestUint32FieldToHex(t *testing.T) {
	value, err := convertUintField(*newTestDHCPOptionField(Uint32Field, 65537), false)
	require.NoError(t, err)
	require.Equal(t, "00010001", value)
}

// Test that converted uint32 option field value has 8 digits.
func TestUint32FieldToHexPadding(t *testing.T) {
	value, err := convertUintField(*newTestDHCPOptionField(Uint32Field, 1), false)
	require.NoError(t, err)
	require.Equal(t, "00000001", value)
}

// Test that an uint32 option field is converted to a text format.
func TestUint32FieldToText(t *testing.T) {
	value, err := convertUintField(*newTestDHCPOptionField(Uint32Field, 65537), true)
	require.NoError(t, err)
	require.Equal(t, "65537", value)
}

// Test that conversion of a malformed uint32 option field yields an error.
func TestUint32FieldMalformed(t *testing.T) {
	// It must be a single value.
	_, err := convertUintField(*newTestDHCPOptionField(Uint32Field, 1, 10), false)
	require.Error(t, err)

	// It must be lower than max uint32.
	_, err = convertUintField(*newTestDHCPOptionField(Uint32Field, uint64(math.MaxUint64-5)), false)
	require.Error(t, err)

	// No value.
	_, err = convertUintField(*newTestDHCPOptionField(Uint32Field), false)
	require.Error(t, err)

	// Not a number.
	_, err = convertUintField(*newTestDHCPOptionField(Uint32Field, "222"), false)
	require.Error(t, err)
}

// Test that an IPv4 option field is converted to a hex format.
func TestIPv4AddressFieldToHex(t *testing.T) {
	value, err := convertIPv4AddressField(*newTestDHCPOptionField(BinaryField, "192.0.2.1"), false)
	require.NoError(t, err)
	require.Equal(t, "C0000201", value)
}

// Test that an IPv4 option field is converted to a text format.
func TestIPv4AddressFieldToText(t *testing.T) {
	value, err := convertIPv4AddressField(*newTestDHCPOptionField(BinaryField, "192.0.2.1"), true)
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", value)
}

// Test that conversion of a malformed IPv4 option field yields an error.
func TestIPv4AddressFieldMalformed(t *testing.T) {
	// It must be a single value.
	_, err := convertIPv4AddressField(*newTestDHCPOptionField(IPv4AddressField, "192.0.2.1", "192.0.2.2"), false)
	require.Error(t, err)

	// No value.
	_, err = convertIPv4AddressField(*newTestDHCPOptionField(IPv4AddressField), false)
	require.Error(t, err)

	// IPv6 address.
	_, err = convertIPv4AddressField(*newTestDHCPOptionField(IPv4AddressField, "2001:db8:1::1"), false)
	require.Error(t, err)

	// Empty string.
	_, err = convertBinaryField(*newTestDHCPOptionField(IPv4AddressField, ""))
	require.Error(t, err)
}

// Test that an IPv6 option field is converted to a hex format.
func TestIPv6AddressFieldToHex(t *testing.T) {
	value, err := convertIPv6AddressField(*newTestDHCPOptionField(BinaryField, "2001:db8:1::1"), false)
	require.NoError(t, err)
	require.Equal(t, "20010DB8000100000000000000000001", value)
}

// Test that an IPv6 option field is converted to a text format.
func TestIPv6AddressFieldToText(t *testing.T) {
	value, err := convertIPv6AddressField(*newTestDHCPOptionField(BinaryField, "2001:db8:1::1"), true)
	require.NoError(t, err)
	require.Equal(t, "2001:db8:1::1", value)
}

// Test that conversion of a malformed IPv6 option field yields an error.
func TestIPv6AddressFieldMalformed(t *testing.T) {
	_, err := convertIPv6AddressField(*newTestDHCPOptionField(IPv6AddressField, "2001:db8:1::1", "2001:db8:1::1"), false)
	require.Error(t, err)

	_, err = convertIPv6AddressField(*newTestDHCPOptionField(IPv6AddressField), false)
	require.Error(t, err)

	_, err = convertIPv6AddressField(*newTestDHCPOptionField(IPv6AddressField, "192.0.2.1"), false)
	require.Error(t, err)
}

// Test that an IPv6 prefix option field is converted to a hex format.
func TestIPv6PrefixFieldToHex(t *testing.T) {
	value, err := convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, "3001::", 64), false)
	require.NoError(t, err)
	require.Equal(t, "3001000000000000000000000000000040", value)
}

// Test that an IPv6 prefix option field is converted to text format.
func TestIPv6PrefixFieldToText(t *testing.T) {
	value, err := convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, "3001::", 64), true)
	require.NoError(t, err)
	require.Equal(t, "3001::/64", value)
}

// Test that conversion of a malformed IPv6 prefix option field yields an error.
func TestIPv6PrefixFieldMalformed(t *testing.T) {
	// No prefix length.
	_, err := convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, "3001::"), false)
	require.Error(t, err)

	// No prefix.
	_, err = convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, 64), false)
	require.Error(t, err)

	// Too high prefix length.
	_, err = convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, "3001::", 129), false)
	require.Error(t, err)

	// Negative prefix length.
	_, err = convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, "3001::", -1), false)
	require.Error(t, err)

	// No value.
	_, err = convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField), false)
	require.Error(t, err)

	// IPv4 prefix.
	_, err = convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, "192.0.2.1", 32), false)
	require.Error(t, err)

	// Empty prefix.
	_, err = convertBinaryField(*newTestDHCPOptionField(IPv6PrefixField, "", 32))
	require.Error(t, err)
}

// Test that PSID option field is converted to hex format.
func TestPsidFieldToHex(t *testing.T) {
	value, err := convertPsidField(*newTestDHCPOptionField(PsidField, 1000, 12), false)
	require.NoError(t, err)
	require.Equal(t, "03E80C", value)
}

// Test that PSID option field is converted to text format.
func TestPsidFieldToText(t *testing.T) {
	value, err := convertPsidField(*newTestDHCPOptionField(PsidField, 1000, 12), true)
	require.NoError(t, err)
	require.Equal(t, "1000/12", value)
}

// Test that conversion of a malformed PSID option field yields an error.
func TestPsidFieldMalformed(t *testing.T) {
	// No PSID length.
	_, err := convertPsidField(*newTestDHCPOptionField(PsidField, 1000), false)
	require.Error(t, err)

	// PSID is not a number.
	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, "1000", 12), false)
	require.Error(t, err)

	// PSID length is not a number.
	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, 1000, "12"), false)
	require.Error(t, err)

	// PSID length is too high.
	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, 1000, 1200), false)
	require.Error(t, err)

	// PSID is too high.
	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, 165535, 12), false)
	require.Error(t, err)

	// PSID is negative.
	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, -1, 12), false)
	require.Error(t, err)

	// PSID length is negative.
	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, 1, -2), false)
	require.Error(t, err)
}

// Test that FQDN option field is converted to hex format.
func TestFqdnFieldToHex(t *testing.T) {
	value, err := convertFqdnField(*newTestDHCPOptionField(FqdnField, "foobar.example.org."), false)
	require.NoError(t, err)
	require.Equal(t, "06666F6F626172076578616D706C65036F726700", value)
}

// Test that FQDN option field is converted to text format.
func TestFqdnFieldToText(t *testing.T) {
	value, err := convertFqdnField(*newTestDHCPOptionField(FqdnField, "foobar.example.org."), true)
	require.NoError(t, err)
	require.Equal(t, "foobar.example.org.", value)
}

// Test that conversion of a malformed FQDN option field yields an error.
func TestFqdnFieldMalformed(t *testing.T) {
	_, err := convertFqdnField(*newTestDHCPOptionField(FqdnField, "foobar.example.org.", "foo"), false)
	require.Error(t, err)

	_, err = convertFqdnField(*newTestDHCPOptionField(FqdnField), false)
	require.Error(t, err)

	_, err = convertFqdnField(*newTestDHCPOptionField(FqdnField, 123), false)
	require.Error(t, err)

	_, err = convertFqdnField(*newTestDHCPOptionField("invalid...fqdn"), false)
	require.Error(t, err)
}
