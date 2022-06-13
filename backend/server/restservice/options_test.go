package restservice

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/server/gen/models"
	storkutil "isc.org/stork/util"
)

// Test successful conversion of DHCP options received over the REST API to
// the database model.
func TestFlattenDHCPOptions(t *testing.T) {
	// Create options with suboptions and different option field types.
	restOptions := []*models.DHCPOption{
		{
			AlwaysSend:  true,
			Code:        1001,
			Encapsulate: "option-1001",
			Fields: []*models.DHCPOptionField{
				{
					FieldType: keaconfig.StringField,
					Values:    []string{"foo"},
				},
			},
			Options: []*models.DHCPOption{
				{
					Code:        1,
					Encapsulate: "option-1001.1",
					Fields: []*models.DHCPOptionField{
						{
							FieldType: keaconfig.HexBytesField,
							Values:    []string{"01:02:03"},
						},
					},
					Universe: 4,
				},
				{
					Code:        2,
					Encapsulate: "option-1001.2",
					Fields: []*models.DHCPOptionField{
						{
							FieldType: keaconfig.BoolField,
							Values:    []string{"true"},
						},
					},
					Universe: 4,
				},
			},
			Universe: 4,
		},
		{
			AlwaysSend:  false,
			Code:        1002,
			Encapsulate: "option-1002",
			Fields: []*models.DHCPOptionField{
				{
					FieldType: keaconfig.Uint16Field,
					Values:    []string{"755"},
				},
			},
			Options: []*models.DHCPOption{
				{
					Code:        3,
					Encapsulate: "option-1002.3",
					Fields: []*models.DHCPOptionField{
						{
							FieldType: keaconfig.Uint8Field,
							Values:    []string{"123"},
						},
						{
							FieldType: keaconfig.PsidField,
							Values:    []string{"1622", "12"},
						},
						{
							FieldType: keaconfig.FqdnField,
							Values:    []string{"foo.example.org."},
						},
					},
					Universe: 4,
				},
				{
					Code:        4,
					Encapsulate: "option-1002.4",
					Fields: []*models.DHCPOptionField{
						{
							FieldType: keaconfig.Uint32Field,
							Values:    []string{"166535"},
						},
						{
							FieldType: keaconfig.IPv6PrefixField,
							Values:    []string{"3001::", "64"},
						},
						{
							FieldType: keaconfig.IPv4AddressField,
							Values:    []string{"192.0.2.2"},
						},
					},
					Universe: 4,
				},
			},
			Universe: 4,
		},
	}
	// Convert and flatten the structure.
	options, err := flattenDHCPOptions("dhcp4", restOptions)
	require.NoError(t, err)
	require.Len(t, options, 6)

	// Sort the options by code because their order is not guaranteed.
	sort.Slice(options, func(i, j int) bool {
		return options[i].Code < options[j].Code
	})
	require.False(t, options[0].AlwaysSend)
	require.EqualValues(t, 1, options[0].Code)
	require.Len(t, options[0].Fields, 1)
	require.Len(t, options[0].Fields[0].Values, 1)
	require.EqualValues(t, "01:02:03", options[0].Fields[0].Values[0])
	require.Equal(t, "option-1001", options[0].Space)
	require.Equal(t, "option-1001.1", options[0].Encapsulate)
	require.Equal(t, storkutil.IPv4, options[0].Universe)

	require.False(t, options[1].AlwaysSend)
	require.EqualValues(t, 2, options[1].Code)
	require.Len(t, options[1].Fields, 1)
	require.Len(t, options[1].Fields[0].Values, 1)
	require.Equal(t, true, options[1].Fields[0].Values[0])
	require.Equal(t, "option-1001", options[1].Space)
	require.Equal(t, "option-1001.2", options[1].Encapsulate)
	require.Equal(t, storkutil.IPv4, options[1].Universe)

	require.False(t, options[2].AlwaysSend)
	require.EqualValues(t, 3, options[2].Code)
	require.Len(t, options[2].Fields, 3)
	require.Len(t, options[2].Fields[0].Values, 1)
	require.Equal(t, uint8(123), options[2].Fields[0].Values[0])
	require.Len(t, options[2].Fields[1].Values, 2)
	require.Equal(t, uint16(1622), options[2].Fields[1].Values[0])
	require.Equal(t, uint8(12), options[2].Fields[1].Values[1])
	require.Len(t, options[2].Fields[2].Values, 1)
	require.Equal(t, "foo.example.org.", options[2].Fields[2].Values[0])
	require.Equal(t, "option-1002", options[2].Space)
	require.Equal(t, "option-1002.3", options[2].Encapsulate)
	require.Equal(t, storkutil.IPv4, options[2].Universe)

	require.False(t, options[3].AlwaysSend)
	require.EqualValues(t, 4, options[3].Code)
	require.Len(t, options[3].Fields, 3)
	require.Len(t, options[3].Fields[0].Values, 1)
	require.Equal(t, uint32(166535), options[3].Fields[0].Values[0])
	require.Len(t, options[3].Fields[1].Values, 2)
	require.Equal(t, "3001::", options[3].Fields[1].Values[0])
	require.Equal(t, uint8(64), options[3].Fields[1].Values[1])
	require.Len(t, options[3].Fields[2].Values, 1)
	require.Equal(t, "192.0.2.2", options[3].Fields[2].Values[0])
	require.Equal(t, "option-1002", options[3].Space)
	require.Equal(t, "option-1002.4", options[3].Encapsulate)
	require.Equal(t, storkutil.IPv4, options[3].Universe)

	require.True(t, options[4].AlwaysSend)
	require.EqualValues(t, 1001, options[4].Code)
	require.Len(t, options[4].Fields, 1)
	require.Len(t, options[4].Fields[0].Values, 1)
	require.EqualValues(t, "foo", options[4].Fields[0].Values[0])
	require.Equal(t, "option-1001", options[4].Encapsulate)
	require.Equal(t, "dhcp4", options[4].Space)
	require.Equal(t, storkutil.IPv4, options[4].Universe)

	require.False(t, options[5].AlwaysSend)
	require.EqualValues(t, 1002, options[5].Code)
	require.Len(t, options[5].Fields, 1)
	require.Len(t, options[5].Fields[0].Values, 1)
	require.EqualValues(t, 755, options[5].Fields[0].Values[0])
	require.Equal(t, "option-1002", options[5].Encapsulate)
	require.Equal(t, "dhcp4", options[5].Space)
	require.Equal(t, storkutil.IPv4, options[5].Universe)
}

// Test negative scenarios of conversion of the DHCP options from the
// REST API format to the database model.
func TestFlattenDHCPOptionsInvalidValues(t *testing.T) {
	type test struct {
		testName  string
		fieldType string
		values    []string
	}
	tests := []test{
		{"non uint8 value", keaconfig.Uint8Field, []string{"foo"}},
		{"non uint16 value", keaconfig.Uint16Field, []string{"foo"}},
		{"non uint32 value", keaconfig.Uint32Field, []string{"foo"}},
		{"uint8 out of range", keaconfig.Uint8Field, []string{"256"}},
		{"uint16 out of range", keaconfig.Uint16Field, []string{"65536"}},
		{"uint32 out of range", keaconfig.Uint32Field, []string{"14294967295"}},
		{"invalid bool", keaconfig.BoolField, []string{"19"}},
		{"prefix lacks length", keaconfig.IPv6PrefixField, []string{"3001::"}},
		{"prefix length out of range", keaconfig.IPv6PrefixField, []string{"3001::", "280"}},
		{"psid lacks length", keaconfig.PsidField, []string{"1600"}},
		{"psid out of range", keaconfig.PsidField, []string{"65536", "12"}},
		{"psid length out of range", keaconfig.PsidField, []string{"12", "1000"}},
		{"no values", keaconfig.StringField, []string{}},
	}

	for _, test := range tests {
		fieldType := test.fieldType
		values := test.values
		t.Run(test.testName, func(t *testing.T) {
			restOptions := []*models.DHCPOption{
				{
					Code:        1001,
					Encapsulate: "option-1001",
					Fields: []*models.DHCPOptionField{
						{
							FieldType: fieldType,
							Values:    values,
						},
					},
				},
			}
			options, err := flattenDHCPOptions("dhcp4", restOptions)
			require.Error(t, err)
			require.Nil(t, options)
		})
	}
}
