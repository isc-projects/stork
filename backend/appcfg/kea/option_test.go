package keaconfig

import (
	"math"
	"testing"

	require "github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// DHCP option field used in the tests implementing the DHCPOptionField
// interface.
type testDHCPOptionField struct {
	fieldType string
	values    []interface{}
}

// DHCP option used in the tests implementing the DHCPOption interface.
type testDHCPOption struct {
	alwaysSend  bool
	code        uint16
	encapsulate string
	fields      []testDHCPOptionField
	name        string
	space       string
}

// DHCP option definition lookup used in the tests.
type testDHCPOptionDefinitionLookup struct {
	hasDefinition bool
}

// Convenience function creating new testDHCPOption instance.
func newTestDHCPOptionField(fieldType string, values ...interface{}) *testDHCPOptionField {
	return &testDHCPOptionField{
		fieldType: fieldType,
		values:    values,
	}
}

// Returns option field type.
func (field testDHCPOptionField) GetFieldType() string {
	return field.fieldType
}

// Returns option field values.
func (field testDHCPOptionField) GetValues() []interface{} {
	return field.values
}

// Checks if the option is always returned to a DHCP client, regardless
// if the client has requested it or not.
func (option testDHCPOption) IsAlwaysSend() bool {
	return option.alwaysSend
}

// Returns option code.
func (option testDHCPOption) GetCode() uint16 {
	return option.code
}

// Returns an encapsulated option space name.
func (option testDHCPOption) GetEncapsulate() string {
	return option.encapsulate
}

// Returns option fields belonging to the option.
func (option testDHCPOption) GetFields() (returnedFields []DHCPOptionField) {
	for _, field := range option.fields {
		returnedFields = append(returnedFields, field)
	}
	return returnedFields
}

// Returns option name.
func (option testDHCPOption) GetName() string {
	return option.name
}

// Returns option space name.
func (option testDHCPOption) GetSpace() string {
	return option.space
}

// Returns option universe (i.e., IPv4 or IPv6).
func (option testDHCPOption) GetUniverse() storkutil.IPType {
	return storkutil.IPv4
}

// Checks if a definition of the specified option exists for the
// given daemon.
func (lookup testDHCPOptionDefinitionLookup) DefinitionExists(daemonID int64, option DHCPOption) bool {
	return lookup.hasDefinition
}

// Test that a DHCP option in the Kea format is created from the Stork's
// option representation. It creates an option with many different option
// fields and simulates the case that a definition for this option exists.
func TestCreateSingleOptionDataMultiplFields(t *testing.T) {
	option := &testDHCPOption{
		alwaysSend:  true,
		code:        1600,
		encapsulate: "foo",
		fields: []testDHCPOptionField{
			{
				fieldType: "uint8",
				values:    []interface{}{123},
			},
			{
				fieldType: "uint16",
				values:    []interface{}{234},
			},
			{
				fieldType: "uint32",
				values:    []interface{}{369},
			},
			{
				fieldType: "bool",
				values:    []interface{}{true},
			},
			{
				fieldType: "ipv4-address",
				values:    []interface{}{"192.0.2.1"},
			},
			{
				fieldType: "ipv6-address",
				values:    []interface{}{"3000:12::"},
			},
			{
				fieldType: "ipv6-prefix",
				values:    []interface{}{"3001::", 64},
			},
			{
				fieldType: "psid",
				values:    []interface{}{1644, 12},
			},
			{
				fieldType: "fqdn",
				values:    []interface{}{"foobar.example.org"},
			},
			{
				fieldType: "string",
				values:    []interface{}{"foobar"},
			},
		},
		name:  "bar",
		space: "foobar",
	}

	lookup := &testDHCPOptionDefinitionLookup{
		hasDefinition: true,
	}

	// Convert the option from the Stork to Kea format.
	data, err := CreateSingleOptionData(1, lookup, option)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Make sure that the conversion was correct.
	require.True(t, data.AlwaysSend)
	require.EqualValues(t, 1600, data.Code)
	require.True(t, data.CSVFormat)
	require.Equal(t, "foobar", data.Space)
	require.Equal(t, "bar", data.Name)

	// Make sure that the option data were set correctly.
	require.Equal(t, "123,234,369,true,192.0.2.1,3000:12::,3001::/64,1644/12,foobar.example.org,foobar", data.Data)
}

// Test the option conversion from the Stork to Kea format when the option
// comprises a field in hex-bytes format.
func TestCreateSingleOptionDataHexBytesField(t *testing.T) {
	option := &testDHCPOption{
		code: 1678,
		fields: []testDHCPOptionField{
			{
				fieldType: "hex-bytes",
				values:    []interface{}{"01:02:03:04"},
			},
		},
	}

	lookup := &testDHCPOptionDefinitionLookup{
		hasDefinition: true,
	}

	// Convert the option from the Stork to Kea format.
	data, err := CreateSingleOptionData(1, lookup, option)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Make sure the option was converted ok.
	require.False(t, data.AlwaysSend)
	require.EqualValues(t, 1678, data.Code)
	require.True(t, data.CSVFormat)
	require.Empty(t, data.Space)
	require.Empty(t, data.Name)

	// The colons should have been sanitized.
	require.Equal(t, "01020304", data.Data)
}

// Test that a DHCP option in the Kea format is created from the Stork's
// option representation. It creates an option with many different option
// fields and simulates the case that a definition for this DOES NOT exist.
func TestCreateSingleOptionDataNoDefinition(t *testing.T) {
	option := &testDHCPOption{
		alwaysSend:  true,
		code:        16,
		encapsulate: "foobar",
		fields: []testDHCPOptionField{
			{
				fieldType: "uint8",
				values:    []interface{}{123},
			},
			{
				fieldType: "uint16",
				values:    []interface{}{234},
			},
			{
				fieldType: "uint32",
				values:    []interface{}{369},
			},
			{
				fieldType: "bool",
				values:    []interface{}{true},
			},
			{
				fieldType: "ipv4-address",
				values:    []interface{}{"192.0.2.1"},
			},
			{
				fieldType: "ipv6-address",
				values:    []interface{}{"3000:12::"},
			},
			{
				fieldType: "ipv6-prefix",
				values:    []interface{}{"3001::", 64},
			},
			{
				fieldType: "psid",
				values:    []interface{}{1644, 12},
			},
			{
				fieldType: "fqdn",
				values:    []interface{}{"foobar.example.org"},
			},
			{
				fieldType: "string",
				values:    []interface{}{"foobar"},
			},
		},
		name:  "bar",
		space: "foo",
	}

	lookup := &testDHCPOptionDefinitionLookup{
		hasDefinition: false,
	}

	// Convert the option from the Stork to Kea format.
	data, err := CreateSingleOptionData(1, lookup, option)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Make sure that the conversion was correct.
	require.True(t, data.AlwaysSend)
	require.EqualValues(t, 16, data.Code)
	require.False(t, data.CSVFormat)
	require.Equal(t, "foo", data.Space)
	require.Equal(t, "bar", data.Name)

	// Make sure that the option data were converted to the hex format.
	require.Equal(t, "7B00EA0000017101C0000201300000120000000000000000000000003001000000000000000000000000000040066C0C06666F6F626172076578616D706C65036F7267666F6F626172", data.Data)
}

// Test that a hex-bytes option field is converted to Kea format successfully.
func TestConvertHexBytesField(t *testing.T) {
	// Colons are allowed.
	value, err := convertHexBytesField(*newTestDHCPOptionField(HexBytesField, "00:01:02:03:04"))
	require.NoError(t, err)
	require.Equal(t, "0001020304", value)

	// Spaces are allowed.
	value, err = convertHexBytesField(*newTestDHCPOptionField(HexBytesField, "00 01 02 03 04"))
	require.NoError(t, err)
	require.Equal(t, "0001020304", value)

	// No separators are also allowed.
	value, err = convertHexBytesField(*newTestDHCPOptionField(HexBytesField, "0001020304"))
	require.NoError(t, err)
	require.Equal(t, "0001020304", value)
}

// Test that conversion of a malformed hex-bytes option field yields an error.
func TestConvertHexBytesFieldMalformed(t *testing.T) {
	// It must have a single value.
	_, err := convertHexBytesField(*newTestDHCPOptionField(HexBytesField, "010203", "010203"))
	require.Error(t, err)

	// Having no values is wrong.
	_, err = convertHexBytesField(*newTestDHCPOptionField(HexBytesField))
	require.Error(t, err)

	// Non-hex string.
	_, err = convertHexBytesField(*newTestDHCPOptionField(HexBytesField, "wrong"))
	require.Error(t, err)

	// Not a string.
	_, err = convertHexBytesField(*newTestDHCPOptionField(HexBytesField, 525))
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
	value, err := convertIPv4AddressField(*newTestDHCPOptionField(HexBytesField, "192.0.2.1"), false)
	require.NoError(t, err)
	require.Equal(t, "C0000201", value)
}

// Test that an IPv4 option field is converted to a text format.
func TestIPv4AddressFieldToText(t *testing.T) {
	value, err := convertIPv4AddressField(*newTestDHCPOptionField(HexBytesField, "192.0.2.1"), true)
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", value)
}

// Test that conversion of a malformed IPv4 option field yields an error.
func TestIPv4AddressFieldMalformed(t *testing.T) {
	_, err := convertIPv4AddressField(*newTestDHCPOptionField(IPv4AddressField, "192.0.2.1", "192.0.2.2"), false)
	require.Error(t, err)

	_, err = convertIPv4AddressField(*newTestDHCPOptionField(IPv4AddressField), false)
	require.Error(t, err)

	_, err = convertIPv4AddressField(*newTestDHCPOptionField(IPv4AddressField, "2001:db8:1::1"), false)
	require.Error(t, err)
}

// Test that an IPv6 option field is converted to a hex format.
func TestIPv6AddressFieldToHex(t *testing.T) {
	value, err := convertIPv6AddressField(*newTestDHCPOptionField(HexBytesField, "2001:db8:1::1"), false)
	require.NoError(t, err)
	require.Equal(t, "20010DB8000100000000000000000001", value)
}

// Test that an IPv6 option field is converted to a text format.
func TestIPv6AddressFieldToText(t *testing.T) {
	value, err := convertIPv6AddressField(*newTestDHCPOptionField(HexBytesField, "2001:db8:1::1"), true)
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
	_, err := convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, "3001::"), false)
	require.Error(t, err)

	_, err = convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, 64), false)
	require.Error(t, err)

	_, err = convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, "3001::", 129), false)
	require.Error(t, err)

	_, err = convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField), false)
	require.Error(t, err)

	_, err = convertIPv6PrefixField(*newTestDHCPOptionField(IPv6PrefixField, "192.0.2.1", 32), false)
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
	_, err := convertPsidField(*newTestDHCPOptionField(PsidField, 1000), false)
	require.Error(t, err)

	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, "1000", 12), false)
	require.Error(t, err)

	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, 1000, "12"), false)
	require.Error(t, err)

	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, 1000, 1200), false)
	require.Error(t, err)

	_, err = convertPsidField(*newTestDHCPOptionField(PsidField, 165535, 12), false)
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
