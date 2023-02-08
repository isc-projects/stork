package keaconfig

import (
	"testing"

	require "github.com/stretchr/testify/require"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
)

//go:generate mockgen -package=keaconfig_test -destination=optiondefmock_test.go isc.org/stork/appcfg/kea DHCPOptionDefinition
//go:generate mockgen -package=keaconfig_test -destination=optiondeflookupmock_test.go isc.org/stork/appcfg/kea DHCPOptionDefinitionLookup

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
// an array of records. When the position is greater than the size of
// the record, the next record field types are returned.
func TestDHCPOptionDefinitionFieldTypeRecordArray(t *testing.T) {
	def := &dhcpOptionDefinition{
		Array:      true,
		OptionType: RecordOption,
		RecordTypes: []DHCPOptionType{
			Uint8Option,
			Uint16Option,
			Uint32Option,
		},
	}
	for i := 0; i < 3; i++ {
		offset := i * len(def.RecordTypes)
		fieldType, ok := GetDHCPOptionDefinitionFieldType(def, offset)
		require.True(t, ok)
		require.Equal(t, dhcpmodel.Uint8Field, fieldType)

		fieldType, ok = GetDHCPOptionDefinitionFieldType(def, offset+1)
		require.True(t, ok)
		require.Equal(t, dhcpmodel.Uint16Field, fieldType)

		fieldType, ok = GetDHCPOptionDefinitionFieldType(def, offset+2)
		require.True(t, ok)
		require.Equal(t, dhcpmodel.Uint32Field, fieldType)
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
