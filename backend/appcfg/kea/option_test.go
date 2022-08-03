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

	// Empty string.
	_, err = convertHexBytesField(*newTestDHCPOptionField(HexBytesField, ""))
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
	_, err = convertHexBytesField(*newTestDHCPOptionField(StringField, ""))
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
	_, err = convertHexBytesField(*newTestDHCPOptionField(IPv4AddressField, ""))
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
	_, err = convertHexBytesField(*newTestDHCPOptionField(IPv6PrefixField, "", 32))
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

// Test that an option received from Kea is correctly parsed into the Stork's
// representation of an option.
func TestCreateDHCPOptionCSV(t *testing.T) {
	optionData := SingleOptionData{
		AlwaysSend: true,
		Code:       244,
		CSVFormat:  true,
		Data:       "192.0.2.1, xyz, true, 1020, 3000::/64, 90/2, foobar.example.com., 2001:db8:1::12",
		Name:       "foo",
		Space:      "bar",
	}
	option := CreateDHCPOption(optionData, storkutil.IPv4)
	require.True(t, option.IsAlwaysSend())
	require.EqualValues(t, 244, option.GetCode())
	require.Equal(t, "foo", option.GetName())
	require.Equal(t, "bar", option.GetSpace())
	require.Equal(t, storkutil.IPv4, option.GetUniverse())
	require.Equal(t, "bar.244", option.GetEncapsulate())

	fields := option.GetFields()
	require.Len(t, fields, 8)
	require.Equal(t, IPv4AddressField, fields[0].GetFieldType())
	require.Len(t, fields[0].GetValues(), 1)
	require.Equal(t, "192.0.2.1", fields[0].GetValues()[0])

	require.Equal(t, StringField, fields[1].GetFieldType())
	require.Len(t, fields[1].GetValues(), 1)
	require.Equal(t, "xyz", fields[1].GetValues()[0])

	require.Equal(t, BoolField, fields[2].GetFieldType())
	require.Len(t, fields[2].GetValues(), 1)
	require.Equal(t, true, fields[2].GetValues()[0])

	require.Equal(t, Uint32Field, fields[3].GetFieldType())
	require.Len(t, fields[1].GetValues(), 1)
	require.EqualValues(t, 1020, fields[3].GetValues()[0])

	require.Equal(t, IPv6PrefixField, fields[4].GetFieldType())
	require.Len(t, fields[4].GetValues(), 2)
	require.Equal(t, "3000::", fields[4].GetValues()[0])
	require.EqualValues(t, 64, fields[4].GetValues()[1])

	require.Equal(t, PsidField, fields[5].GetFieldType())
	require.Len(t, fields[5].GetValues(), 2)
	require.EqualValues(t, 90, fields[5].GetValues()[0])
	require.EqualValues(t, 2, fields[5].GetValues()[1])

	require.Equal(t, FqdnField, fields[6].GetFieldType())
	require.Len(t, fields[6].GetValues(), 1)
	require.EqualValues(t, "foobar.example.com.", fields[6].GetValues()[0])

	require.Equal(t, IPv6AddressField, fields[7].GetFieldType())
	require.Len(t, fields[7].GetValues(), 1)
	require.EqualValues(t, "2001:db8:1::12", fields[7].GetValues()[0])
}

// Test that an option in a hex bytes format received from Kea is correctly parsed
// into the Stork's representation of an option.
func TestCreateDHCPOptionHex(t *testing.T) {
	optionData := SingleOptionData{
		AlwaysSend: false,
		Code:       2048,
		CSVFormat:  false,
		Data:       "01 02 03 04 05 06 07 08 09 0A",
		Name:       "foobar",
		Space:      "baz",
	}
	option := CreateDHCPOption(optionData, storkutil.IPv6)
	require.False(t, option.IsAlwaysSend())
	require.EqualValues(t, 2048, option.GetCode())
	require.Equal(t, "foobar", option.GetName())
	require.Equal(t, "baz", option.GetSpace())
	require.Equal(t, storkutil.IPv6, option.GetUniverse())
	require.Equal(t, "baz.2048", option.GetEncapsulate())

	fields := option.GetFields()
	require.Len(t, fields, 1)
	require.Equal(t, HexBytesField, fields[0].GetFieldType())
	require.Len(t, fields[0].GetValues(), 1)
	require.Equal(t, "0102030405060708090A", fields[0].GetValues()[0])
}

// Test that an empty option received from Kea is correctly parsed into the
// Stork's representation of an option.
func TestCreateDHCPOptionEmpty(t *testing.T) {
	optionData := SingleOptionData{
		Code:      333,
		CSVFormat: true,
		Name:      "foobar",
		Space:     "baz",
	}
	option := CreateDHCPOption(optionData, storkutil.IPv6)
	require.False(t, option.IsAlwaysSend())
	require.EqualValues(t, 333, option.GetCode())
	require.Equal(t, "foobar", option.GetName())
	require.Equal(t, "baz", option.GetSpace())
	require.Equal(t, storkutil.IPv6, option.GetUniverse())
	require.Equal(t, "baz.333", option.GetEncapsulate())
	require.Empty(t, option.GetFields())
}

// Test encapsulated option space setting for top-level DHCPv4 options.
func TestCreateDHCPOptionEncapsulateDHCPv4TopLevel(t *testing.T) {
	optionData := SingleOptionData{
		Code:  253,
		Space: DHCPv4OptionSpace,
	}
	option := CreateDHCPOption(optionData, storkutil.IPv4)
	require.Equal(t, "option-253", option.GetEncapsulate())
}

// Test encapsulated option space setting for DHCPv4 suboptions.
func TestCreateDHCPOptionEncapsulateDHCPv4Suboption(t *testing.T) {
	optionData := SingleOptionData{
		Code:  1,
		Space: "option-253",
	}
	option := CreateDHCPOption(optionData, storkutil.IPv4)
	require.Equal(t, "option-253.1", option.GetEncapsulate())
}

// Test encapsulated option space setting for top-level DHCPv6 options.
func TestCreateDHCPOptionEncapsulateDHCPv6TopLevel(t *testing.T) {
	optionData := SingleOptionData{
		Code:  1024,
		Space: DHCPv6OptionSpace,
	}
	option := CreateDHCPOption(optionData, storkutil.IPv6)
	require.Equal(t, "option-1024", option.GetEncapsulate())
}

// Test encapsulated option space setting for DHCPv6 suboptions.
func TestCreateDHCPOptionEncapsulateDHCPv6Suboption(t *testing.T) {
	optionData := SingleOptionData{
		Code:  1,
		Space: "option-1024",
	}
	option := CreateDHCPOption(optionData, storkutil.IPv6)
	require.Equal(t, "option-1024.1", option.GetEncapsulate())
}
