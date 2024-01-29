package keaconfig_test

import (
	"testing"

	require "github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=keaconfig_test -destination=optionmock_test.go isc.org/stork/datamodel/dhcp DHCPOptionAccessor

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
func (option testDHCPOption) GetFields() (returnedFields []dhcpmodel.DHCPOptionFieldAccessor) {
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

// Test that a DHCP option in the Kea format is created from the Stork's
// option representation. It creates an option with many different option
// fields and simulates the case that a definition for this option exists.
func TestCreateSingleOptionDataMultipleFields(t *testing.T) {
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
				fieldType: "int8",
				values:    []interface{}{-123},
			},
			{
				fieldType: "int16",
				values:    []interface{}{-234},
			},
			{
				fieldType: "int32",
				values:    []interface{}{-369},
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

	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().DefinitionExists(gomock.Any(), gomock.Any()).Return(true)

	// Convert the option from the Stork to Kea format.
	data, err := keaconfig.CreateSingleOptionData(1, lookup, option)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Make sure that the conversion was correct.
	require.True(t, data.AlwaysSend)
	require.EqualValues(t, 1600, data.Code)
	require.True(t, data.CSVFormat)
	require.Equal(t, "foobar", data.Space)
	require.Equal(t, "bar", data.Name)

	// Make sure that the option data were set correctly.
	require.Equal(t, "123,234,369,-123,-234,-369,true,192.0.2.1,3000:12::,3001::/64,1644/12,foobar.example.org,foobar", data.Data)
}

// Test the option conversion from the Stork to Kea format when the option
// comprises a field in the binary format.
func TestCreateSingleOptionDataBinaryField(t *testing.T) {
	option := &testDHCPOption{
		code: 1678,
		fields: []testDHCPOptionField{
			{
				fieldType: "binary",
				values:    []interface{}{"01:02:03:04"},
			},
		},
	}

	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().DefinitionExists(gomock.Any(), gomock.Any()).Return(true)

	// Convert the option from the Stork to Kea format.
	data, err := keaconfig.CreateSingleOptionData(1, lookup, option)
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
				fieldType: "int8",
				values:    []interface{}{-123},
			},
			{
				fieldType: "int16",
				values:    []interface{}{-234},
			},
			{
				fieldType: "int32",
				values:    []interface{}{-369},
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

	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().DefinitionExists(gomock.Any(), gomock.Any()).Return(false)

	// Convert the option from the Stork to Kea format.
	data, err := keaconfig.CreateSingleOptionData(1, lookup, option)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Make sure that the conversion was correct.
	require.True(t, data.AlwaysSend)
	require.EqualValues(t, 16, data.Code)
	require.False(t, data.CSVFormat)
	require.Equal(t, "foo", data.Space)
	require.Equal(t, "bar", data.Name)

	// Make sure that the option data were converted to the hex format.
	require.Equal(t, "7B00EA0000017185FF16FFFFFE8F01C0000201300000120000000000000000000000003001000000000000000000000000000040066C0C06666F6F626172076578616D706C65036F7267666F6F626172", data.Data)
}

// Test that an option received from Kea is correctly parsed into the Stork's
// representation of an option.
func TestCreateDHCPOptionCSV(t *testing.T) {
	optionData := keaconfig.SingleOptionData{
		AlwaysSend: true,
		Code:       244,
		CSVFormat:  true,
		Data:       "192.0.2.1, xyz, true, 1020, 3000::/64, 90/2, foobar.example.com., 2001:db8:1::12, -5",
		Name:       "foo",
		Space:      "bar",
	}
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil)
	option, err := keaconfig.CreateDHCPOption(optionData, storkutil.IPv4, lookup)
	require.NoError(t, err)
	require.True(t, option.IsAlwaysSend())
	require.EqualValues(t, 244, option.GetCode())
	require.Equal(t, "foo", option.GetName())
	require.Equal(t, "bar", option.GetSpace())
	require.Equal(t, storkutil.IPv4, option.GetUniverse())
	require.Equal(t, "bar.244", option.GetEncapsulate())

	fields := option.GetFields()
	require.Len(t, fields, 9)
	require.Equal(t, dhcpmodel.IPv4AddressField, fields[0].GetFieldType())
	require.Len(t, fields[0].GetValues(), 1)
	require.Equal(t, "192.0.2.1", fields[0].GetValues()[0])

	require.Equal(t, dhcpmodel.StringField, fields[1].GetFieldType())
	require.Len(t, fields[1].GetValues(), 1)
	require.Equal(t, "xyz", fields[1].GetValues()[0])

	require.Equal(t, dhcpmodel.BoolField, fields[2].GetFieldType())
	require.Len(t, fields[2].GetValues(), 1)
	require.Equal(t, true, fields[2].GetValues()[0])

	require.Equal(t, dhcpmodel.Uint32Field, fields[3].GetFieldType())
	require.Len(t, fields[1].GetValues(), 1)
	require.EqualValues(t, 1020, fields[3].GetValues()[0])

	require.Equal(t, dhcpmodel.IPv6PrefixField, fields[4].GetFieldType())
	require.Len(t, fields[4].GetValues(), 2)
	require.Equal(t, "3000::", fields[4].GetValues()[0])
	require.EqualValues(t, 64, fields[4].GetValues()[1])

	require.Equal(t, dhcpmodel.PsidField, fields[5].GetFieldType())
	require.Len(t, fields[5].GetValues(), 2)
	require.EqualValues(t, 90, fields[5].GetValues()[0])
	require.EqualValues(t, 2, fields[5].GetValues()[1])

	require.Equal(t, dhcpmodel.FqdnField, fields[6].GetFieldType())
	require.Len(t, fields[6].GetValues(), 1)
	require.EqualValues(t, "foobar.example.com.", fields[6].GetValues()[0])

	require.Equal(t, dhcpmodel.IPv6AddressField, fields[7].GetFieldType())
	require.Len(t, fields[7].GetValues(), 1)
	require.EqualValues(t, "2001:db8:1::12", fields[7].GetValues()[0])

	require.Equal(t, dhcpmodel.Int32Field, fields[8].GetFieldType())
	require.Len(t, fields[8].GetValues(), 1)
	require.EqualValues(t, -5, fields[8].GetValues()[0])
}

// Test that an option in a hex bytes format received from Kea is correctly parsed
// into the Stork's representation of an option.
func TestCreateDHCPOptionHex(t *testing.T) {
	optionData := keaconfig.SingleOptionData{
		AlwaysSend: false,
		Code:       2048,
		CSVFormat:  false,
		Data:       "01 02 03 04 05 06 07 08 09 0A",
		Name:       "foobar",
		Space:      "baz",
	}
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil)
	option, err := keaconfig.CreateDHCPOption(optionData, storkutil.IPv6, lookup)
	require.NoError(t, err)
	require.False(t, option.IsAlwaysSend())
	require.EqualValues(t, 2048, option.GetCode())
	require.Equal(t, "foobar", option.GetName())
	require.Equal(t, "baz", option.GetSpace())
	require.Equal(t, storkutil.IPv6, option.GetUniverse())
	require.Equal(t, "baz.2048", option.GetEncapsulate())

	fields := option.GetFields()
	require.Len(t, fields, 1)
	require.Equal(t, dhcpmodel.BinaryField, fields[0].GetFieldType())
	require.Len(t, fields[0].GetValues(), 1)
	require.Equal(t, "0102030405060708090A", fields[0].GetValues()[0])
}

// Test that an empty option received from Kea is correctly parsed into the
// Stork's representation of an option.
func TestCreateDHCPOptionEmpty(t *testing.T) {
	optionData := keaconfig.SingleOptionData{
		Code:      333,
		CSVFormat: true,
		Name:      "foobar",
		Space:     "baz",
	}
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil)
	option, err := keaconfig.CreateDHCPOption(optionData, storkutil.IPv6, lookup)
	require.NoError(t, err)
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
	optionData := keaconfig.SingleOptionData{
		Code:  253,
		Space: dhcpmodel.DHCPv4OptionSpace,
	}
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil)
	option, err := keaconfig.CreateDHCPOption(optionData, storkutil.IPv4, lookup)
	require.NoError(t, err)
	require.Equal(t, "option-253", option.GetEncapsulate())
}

// Test encapsulated option space setting for DHCPv4 suboptions.
func TestCreateDHCPOptionEncapsulateDHCPv4Suboption(t *testing.T) {
	optionData := keaconfig.SingleOptionData{
		Code:  1,
		Space: "option-253",
	}
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil)
	option, err := keaconfig.CreateDHCPOption(optionData, storkutil.IPv4, lookup)
	require.NoError(t, err)
	require.Equal(t, "option-253.1", option.GetEncapsulate())
}

// Test encapsulated option space setting for top-level DHCPv6 options.
func TestCreateDHCPOptionEncapsulateDHCPv6TopLevel(t *testing.T) {
	optionData := keaconfig.SingleOptionData{
		Code:  1024,
		Space: dhcpmodel.DHCPv6OptionSpace,
	}
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil)
	option, err := keaconfig.CreateDHCPOption(optionData, storkutil.IPv6, lookup)
	require.NoError(t, err)
	require.Equal(t, "option-1024", option.GetEncapsulate())
}

// Test encapsulated option space setting for DHCPv6 suboptions.
func TestCreateDHCPOptionEncapsulateDHCPv6Suboption(t *testing.T) {
	optionData := keaconfig.SingleOptionData{
		Code:  1,
		Space: "option-1024",
	}
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil)
	option, err := keaconfig.CreateDHCPOption(optionData, storkutil.IPv6, lookup)
	require.NoError(t, err)
	require.Equal(t, "option-1024.1", option.GetEncapsulate())
}

// Test that a standard option definition is used for setting the
// encapsulated option space and setting the option field types.
func TestCreateStandardDHCPOption(t *testing.T) {
	optionData := keaconfig.SingleOptionData{
		Code:      89,
		CSVFormat: true,
		Data:      "10, 9, 6, 192.0.2.1, 3000::/64",
		Name:      "s46-rule",
		Space:     "s46-cont-mape-options",
	}
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	stdLookup := keaconfig.NewStdDHCPOptionDefinitionLookup()
	lookup.EXPECT().Find(gomock.Any(), gomock.Any()).Return(stdLookup.FindByCodeSpace(optionData.Code, optionData.Space, storkutil.IPv6))
	option, err := keaconfig.CreateDHCPOption(optionData, storkutil.IPv6, lookup)
	require.NoError(t, err)
	require.NotNil(t, option)
	require.False(t, option.IsAlwaysSend())
	require.EqualValues(t, 89, option.GetCode())
	require.Equal(t, "s46-rule", option.GetName())
	require.Equal(t, "s46-cont-mape-options", option.GetSpace())
	require.Equal(t, "s46-rule-options", option.GetEncapsulate())
	require.Equal(t, storkutil.IPv6, option.GetUniverse())

	fields := option.GetFields()
	require.Len(t, fields, 5)
	require.Equal(t, dhcpmodel.Uint8Field, fields[0].GetFieldType())
	require.Len(t, fields[0].GetValues(), 1)
	require.EqualValues(t, 10, fields[0].GetValues()[0])
	require.Equal(t, dhcpmodel.Uint8Field, fields[1].GetFieldType())
	require.Len(t, fields[1].GetValues(), 1)
	require.EqualValues(t, 9, fields[1].GetValues()[0])
	require.Equal(t, dhcpmodel.Uint8Field, fields[2].GetFieldType())
	require.Len(t, fields[2].GetValues(), 1)
	require.EqualValues(t, 6, fields[2].GetValues()[0])
	require.Equal(t, dhcpmodel.IPv4AddressField, fields[3].GetFieldType())
	require.Len(t, fields[3].GetValues(), 1)
	require.Equal(t, "192.0.2.1", fields[3].GetValues()[0])
	require.Equal(t, dhcpmodel.IPv6PrefixField, fields[4].GetFieldType())
	require.Len(t, fields[4].GetValues(), 2)
	require.Equal(t, "3000::", fields[4].GetValues()[0])
	require.Equal(t, 64, fields[4].GetValues()[1])
}

// Test that an option instance is successfully created using an option
// definition with binary field types.
func TestCreateStandardDHCPOptionBinary(t *testing.T) {
	optionData := keaconfig.SingleOptionData{
		Code:      97,
		CSVFormat: true,
		Data:      "1, 010203040102",
		Name:      "uuid-guid",
		Space:     "dhcp4",
	}
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	stdLookup := keaconfig.NewStdDHCPOptionDefinitionLookup()
	lookup.EXPECT().Find(gomock.Any(), gomock.Any()).Return(stdLookup.FindByCodeSpace(optionData.Code, optionData.Space, storkutil.IPv4))
	option, err := keaconfig.CreateDHCPOption(optionData, storkutil.IPv4, lookup)
	require.NoError(t, err)
	require.NotNil(t, option)
	require.False(t, option.IsAlwaysSend())
	require.EqualValues(t, 97, option.GetCode())
	require.Equal(t, "uuid-guid", option.GetName())
	require.Equal(t, "dhcp4", option.GetSpace())
	require.Empty(t, option.GetEncapsulate())
	require.Equal(t, storkutil.IPv4, option.GetUniverse())

	fields := option.GetFields()
	require.Len(t, fields, 2)
	require.Equal(t, dhcpmodel.Uint8Field, fields[0].GetFieldType())
	require.Len(t, fields[0].GetValues(), 1)
	require.EqualValues(t, 1, fields[0].GetValues()[0])
	require.Equal(t, dhcpmodel.BinaryField, fields[1].GetFieldType())
	require.Len(t, fields[1].GetValues(), 1)
	require.EqualValues(t, "010203040102", fields[1].GetValues()[0])
}
