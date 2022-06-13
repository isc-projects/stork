package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	storkutil "isc.org/stork/util"
)

// Test that the DHCPOption model implements expected interfaces.
func TestDHCPOptionInterface(t *testing.T) {
	// Create an option.
	option := DHCPOption{
		AlwaysSend:  true,
		Code:        1776,
		Encapsulate: "bar",
		Name:        "foo",
		Space:       "dhcp6",
		Universe:    storkutil.IPv6,
		Fields: []DHCPOptionField{
			{
				FieldType: keaconfig.StringField,
				Values:    []interface{}{"bar"},
			},
		},
	}
	// Validate returned values.
	require.True(t, option.IsAlwaysSend())
	require.EqualValues(t, 1776, option.GetCode())
	require.Equal(t, "bar", option.GetEncapsulate())
	require.Equal(t, "foo", option.GetName())
	require.Equal(t, "dhcp6", option.GetSpace())
	require.Equal(t, storkutil.IPv6, option.GetUniverse())
	require.Len(t, option.GetFields(), 1)
	require.Equal(t, keaconfig.StringField, option.GetFields()[0].GetFieldType())
	require.Len(t, option.GetFields()[0].GetValues(), 1)
	require.Equal(t, "bar", option.GetFields()[0].GetValues()[0])
}
