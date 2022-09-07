package keaconfig

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

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
