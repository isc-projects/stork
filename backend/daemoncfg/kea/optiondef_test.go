package keaconfig

import (
	"testing"

	require "github.com/stretchr/testify/require"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
)

//go:generate mockgen -package=keaconfig_test -destination=optiondefmock_test.go isc.org/stork/daemoncfg/kea DHCPOptionDefinition
//go:generate mockgen -package=keaconfig_test -destination=optiondeflookupmock_test.go isc.org/stork/daemoncfg/kea DHCPOptionDefinitionLookup

// Test DHCPOptionDefinition interface.
func TestDHCPOptionDefinition(t *testing.T) {
	def := &dhcpOptionDefinition{
		Array:       true,
		Code:        12,
		Encapsulate: "foo",
		Name:        "baz",
		RecordTypes: []DHCPOptionType{
			Uint8Option,
		},
		Space:      "bar",
		OptionType: RecordOption,
	}
	require.NotNil(t, def)
	require.True(t, def.GetArray())
	require.EqualValues(t, 12, def.GetCode())
	require.Equal(t, "foo", def.GetEncapsulate())
	require.Equal(t, "baz", def.GetName())
	require.Len(t, def.GetRecordTypes(), 1)
	require.Equal(t, Uint8Option, def.GetRecordTypes()[0])
	require.Equal(t, "bar", def.GetSpace())
	require.Equal(t, RecordOption, def.GetType())
}

// Check that option field type is not returned for an empty option.
func TestDHCPOptionDefinitionFieldTypeEmpty(t *testing.T) {
	def := &dhcpOptionDefinition{
		OptionType: EmptyOption,
	}
	for i := 0; i < 2; i++ {
		fieldType, ok := GetDHCPOptionDefinitionFieldType(def, i)
		require.False(t, ok)
		require.Empty(t, fieldType)
	}
}

// Check that option field type is returned for the simple option
// comprising a single option field and that it is not returned
// when the position is greater than 0.
func TestDHCPOptionDefinitionFieldTypeSimple(t *testing.T) {
	def := &dhcpOptionDefinition{
		OptionType: StringOption,
	}
	fieldType, ok := GetDHCPOptionDefinitionFieldType(def, 0)
	require.True(t, ok)
	require.Equal(t, dhcpmodel.StringField, fieldType)

	fieldType, ok = GetDHCPOptionDefinitionFieldType(def, 1)
	require.False(t, ok)
	require.Empty(t, fieldType)
}

// Check that the same option field type is returned regardless of
// the option for an option comprising an array.
func TestDHCPOptionDefinitionFieldTypeSimpleArray(t *testing.T) {
	def := &dhcpOptionDefinition{
		Array:      true,
		OptionType: Uint8Option,
	}
	for i := 0; i < 3; i++ {
		fieldType, ok := GetDHCPOptionDefinitionFieldType(def, i)
		require.True(t, ok)
		require.Equal(t, dhcpmodel.Uint8Field, fieldType)
	}
}

// Check that record option field types are returned for the option
// comprising an record of fields.
func TestDHCPOptionDefinitionFieldTypeRecord(t *testing.T) {
	def := &dhcpOptionDefinition{
		OptionType: RecordOption,
		RecordTypes: []DHCPOptionType{
			PsidOption,
			StringOption,
		},
	}
	fieldType, ok := GetDHCPOptionDefinitionFieldType(def, 0)
	require.True(t, ok)
	require.Equal(t, dhcpmodel.PsidField, fieldType)

	fieldType, ok = GetDHCPOptionDefinitionFieldType(def, 1)
	require.True(t, ok)
	require.Equal(t, dhcpmodel.StringField, fieldType)

	fieldType, ok = GetDHCPOptionDefinitionFieldType(def, 2)
	require.False(t, ok)
	require.Empty(t, fieldType)
}

// Check that option field types are returned for the option comprising
// a record with a trailing array field. Per Kea's own semantics, "array"
// on a record option means the *last* field type repeats for any values
// beyond the record's fixed fields; the record does not cycle back to its
// first field type. Regression test for a bug where slp-directory-agent
// (record-types "bool, ipv4-address", array) misidentified the third and
// later CSV values (meant to be additional ipv4-address values) as bool,
// because the record was wrapping back to position 0 via a modulo
// operation on every position, causing Stork to reject the entire subnet
// containing this option during a config pull.
func TestDHCPOptionDefinitionFieldTypeRecordArray(t *testing.T) {
	def := &dhcpOptionDefinition{
		Array:      true,
		OptionType: RecordOption,
		RecordTypes: []DHCPOptionType{
			BoolOption,
			IPv4AddressOption,
		},
	}
	fieldType, ok := GetDHCPOptionDefinitionFieldType(def, 0)
	require.True(t, ok)
	require.Equal(t, dhcpmodel.BoolField, fieldType)

	for i := 1; i < 5; i++ {
		fieldType, ok := GetDHCPOptionDefinitionFieldType(def, i)
		require.True(t, ok)
		require.Equal(t, dhcpmodel.IPv4AddressField, fieldType)
	}
}

// Check that false is returned for the record option that lacks
// actual record.
func TestDHCPOptionDefinitionFieldTypeRecordNoRecordTypes(t *testing.T) {
	def := &dhcpOptionDefinition{
		OptionType:  RecordOption,
		RecordTypes: []DHCPOptionType{},
	}

	fieldType, ok := GetDHCPOptionDefinitionFieldType(def, 0)
	require.False(t, ok)
	require.Empty(t, fieldType)
}
