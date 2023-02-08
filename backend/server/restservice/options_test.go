package restservice

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	storkutil "isc.org/stork/util"
)

// Test successful conversion of DHCPv4 options received over the REST API to
// the database model.
func TestFlattenDHCPv4Options(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	// Create options with suboptions and different option field types.
	restOptions := []*models.DHCPOption{
		{
			AlwaysSend:  true,
			Code:        1001,
			Encapsulate: "option-1001",
			Fields: []*models.DHCPOptionField{
				{
					FieldType: dhcpmodel.StringField,
					Values:    []string{"foo"},
				},
			},
			Options: []*models.DHCPOption{
				{
					Code:        1,
					Encapsulate: "option-1001.1",
					Fields: []*models.DHCPOptionField{
						{
							FieldType: dhcpmodel.BinaryField,
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
							FieldType: dhcpmodel.BoolField,
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
					FieldType: dhcpmodel.Uint16Field,
					Values:    []string{"755"},
				},
			},
			Options: []*models.DHCPOption{
				{
					Code:        3,
					Encapsulate: "option-1002.3",
					Fields: []*models.DHCPOptionField{
						{
							FieldType: dhcpmodel.Uint8Field,
							Values:    []string{"123"},
						},
						{
							FieldType: dhcpmodel.PsidField,
							Values:    []string{"1622", "12"},
						},
						{
							FieldType: dhcpmodel.FqdnField,
							Values:    []string{"foo.example.org."},
						},
						{
							FieldType: dhcpmodel.Int8Field,
							Values:    []string{"-123"},
						},
						{
							FieldType: dhcpmodel.Int16Field,
							Values:    []string{"-234"},
						},
						{
							FieldType: dhcpmodel.Int32Field,
							Values:    []string{"-345"},
						},
					},
					Universe: 4,
				},
				{
					Code:        4,
					Encapsulate: "option-1002.4",
					Fields: []*models.DHCPOptionField{
						{
							FieldType: dhcpmodel.Uint32Field,
							Values:    []string{"166535"},
						},
						{
							FieldType: dhcpmodel.IPv6PrefixField,
							Values:    []string{"3001::", "64"},
						},
						{
							FieldType: dhcpmodel.IPv4AddressField,
							Values:    []string{"192.0.2.2"},
						},
					},
					Universe: 4,
					Options: []*models.DHCPOption{
						{
							Code:        5,
							Encapsulate: "option-1002.4.5",
							Fields: []*models.DHCPOptionField{
								{
									FieldType: dhcpmodel.StringField,
									Values:    []string{"baz"},
								},
							},
							Universe: 4,
							Options: []*models.DHCPOption{
								// This option is at 4-th recursion level, thus it should be
								// excluded from the result.
								{
									Code:        6,
									Encapsulate: "option-1002.4.5.6",
									Fields: []*models.DHCPOptionField{
										{
											FieldType: dhcpmodel.Uint32Field,
											Values:    []string{"12"},
										},
									},
									Universe: 4,
								},
							},
						},
					},
				},
			},
			Universe: 4,
		},
	}
	// Convert and flatten the structure.
	options, err := rapi.flattenDHCPOptions("dhcp4", restOptions, 0)
	require.NoError(t, err)
	require.Len(t, options, 7)

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
	require.Len(t, options[2].Fields, 6)
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
	require.Len(t, options[2].Fields[3].Values, 1)
	require.Equal(t, int8(-123), options[2].Fields[3].Values[0])
	require.Len(t, options[2].Fields[4].Values, 1)
	require.Equal(t, int16(-234), options[2].Fields[4].Values[0])
	require.Len(t, options[2].Fields[5].Values, 1)
	require.Equal(t, int32(-345), options[2].Fields[5].Values[0])

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

	require.False(t, options[4].AlwaysSend)
	require.EqualValues(t, 5, options[4].Code)
	require.Len(t, options[4].Fields, 1)
	require.Len(t, options[4].Fields[0].Values, 1)
	require.EqualValues(t, "baz", options[4].Fields[0].Values[0])
	require.Equal(t, "option-1002.4.5", options[4].Encapsulate)
	require.Equal(t, "option-1002.4", options[4].Space)
	require.Equal(t, storkutil.IPv4, options[4].Universe)

	require.True(t, options[5].AlwaysSend)
	require.EqualValues(t, 1001, options[5].Code)
	require.Len(t, options[5].Fields, 1)
	require.Len(t, options[5].Fields[0].Values, 1)
	require.EqualValues(t, "foo", options[5].Fields[0].Values[0])
	require.Equal(t, "option-1001", options[5].Encapsulate)
	require.Equal(t, "dhcp4", options[5].Space)
	require.Equal(t, storkutil.IPv4, options[5].Universe)

	require.False(t, options[6].AlwaysSend)
	require.EqualValues(t, 1002, options[6].Code)
	require.Len(t, options[6].Fields, 1)
	require.Len(t, options[6].Fields[0].Values, 1)
	require.EqualValues(t, 755, options[6].Fields[0].Values[0])
	require.Equal(t, "option-1002", options[6].Encapsulate)
	require.Equal(t, "dhcp4", options[6].Space)
	require.Equal(t, storkutil.IPv4, options[6].Universe)
}

// Test successful conversion of standard DHCPv6 options received over the
// REST API to the database model. Standard options have their definitions
// used to set the encapsulate field.
func TestFlattenDHCPv6Options(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	// Create options with suboptions and different option field types.
	restOptions := []*models.DHCPOption{
		{
			AlwaysSend:  true,
			Code:        94,
			Encapsulate: "option-94",
			Fields:      []*models.DHCPOptionField{},
			Options: []*models.DHCPOption{
				{
					Code:        89,
					Encapsulate: "option-94.89",
					Fields: []*models.DHCPOptionField{
						{
							FieldType: dhcpmodel.Uint8Field,
							Values:    []string{"1"},
						},
					},
					Options: []*models.DHCPOption{
						{
							Code:        93,
							Encapsulate: "option-94.89.93",
							Fields: []*models.DHCPOptionField{
								{
									FieldType: dhcpmodel.Uint8Field,
									Values:    []string{"2"},
								},
							},
							Universe: 6,
						},
					},
					Universe: 6,
				},
			},
			Universe: 6,
		},
	}
	// Convert and flatten the structure.
	options, err := rapi.flattenDHCPOptions("dhcp6", restOptions, 0)
	require.NoError(t, err)
	require.Len(t, options, 3)

	// Sort the options by code because their order is not guaranteed.
	sort.Slice(options, func(i, j int) bool {
		return options[i].Code < options[j].Code
	})
	require.False(t, options[0].AlwaysSend)
	require.EqualValues(t, 89, options[0].Code)
	require.Len(t, options[0].Fields, 1)
	require.Len(t, options[0].Fields[0].Values, 1)
	require.EqualValues(t, 1, options[0].Fields[0].Values[0])
	require.Equal(t, "s46-cont-mape-options", options[0].Space)
	require.Equal(t, "s46-rule-options", options[0].Encapsulate)
	require.Equal(t, storkutil.IPv6, options[0].Universe)

	require.False(t, options[1].AlwaysSend)
	require.EqualValues(t, 93, options[1].Code)
	require.Len(t, options[1].Fields, 1)
	require.Len(t, options[1].Fields[0].Values, 1)
	require.EqualValues(t, 2, options[1].Fields[0].Values[0])
	require.Equal(t, "s46-rule-options", options[1].Space)
	require.Empty(t, options[1].Encapsulate)
	require.Equal(t, storkutil.IPv6, options[1].Universe)

	require.True(t, options[2].AlwaysSend)
	require.EqualValues(t, 94, options[2].Code)
	require.Empty(t, options[2].Fields)
	require.Equal(t, "dhcp6", options[2].Space)
	require.Equal(t, "s46-cont-mape-options", options[2].Encapsulate)
	require.Equal(t, storkutil.IPv6, options[2].Universe)
}

// Test negative scenarios of conversion of the DHCP options from the
// REST API format to the database model.
func TestFlattenDHCPOptionsInvalidValues(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	type test struct {
		testName  string
		fieldType string
		values    []string
	}
	tests := []test{
		{"non uint8 value", dhcpmodel.Uint8Field, []string{"foo"}},
		{"non uint16 value", dhcpmodel.Uint16Field, []string{"foo"}},
		{"non uint32 value", dhcpmodel.Uint32Field, []string{"foo"}},
		{"non int8 value", dhcpmodel.Int8Field, []string{"foo"}},
		{"non int16 value", dhcpmodel.Int16Field, []string{"foo"}},
		{"non int32 value", dhcpmodel.Int32Field, []string{"foo"}},
		{"uint8 out of range", dhcpmodel.Uint8Field, []string{"256"}},
		{"uint16 out of range", dhcpmodel.Uint16Field, []string{"65536"}},
		{"uint32 out of range", dhcpmodel.Uint32Field, []string{"14294967295"}},
		{"int8 out of range", dhcpmodel.Int8Field, []string{"256"}},
		{"int16 out of range", dhcpmodel.Int16Field, []string{"65536"}},
		{"int32 out of range", dhcpmodel.Int32Field, []string{"14294967295"}},
		{"invalid bool", dhcpmodel.BoolField, []string{"19"}},
		{"prefix lacks length", dhcpmodel.IPv6PrefixField, []string{"3001::"}},
		{"prefix length out of range", dhcpmodel.IPv6PrefixField, []string{"3001::", "280"}},
		{"psid lacks length", dhcpmodel.PsidField, []string{"1600"}},
		{"psid out of range", dhcpmodel.PsidField, []string{"65536", "12"}},
		{"psid length out of range", dhcpmodel.PsidField, []string{"12", "1000"}},
		{"no values", dhcpmodel.StringField, []string{}},
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
			options, err := rapi.flattenDHCPOptions("dhcp4", restOptions, 0)
			require.Error(t, err)
			require.Nil(t, options)
		})
	}
}

// Test that a DHCP option model is successfully converted to a REST API format.
func TestUnflattenDHCPOptions(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	options := []dbmodel.DHCPOption{
		{
			AlwaysSend:  true,
			Code:        1001,
			Encapsulate: "option-1001",
			Fields: []dbmodel.DHCPOptionField{
				{
					FieldType: dhcpmodel.StringField,
					Values:    []any{"foo"},
				},
			},
		},
		{
			AlwaysSend: false,
			Code:       1,
			Fields: []dbmodel.DHCPOptionField{
				{
					FieldType: dhcpmodel.Uint8Field,
					Values:    []any{11},
				},
			},
			Space:       "option-1001",
			Encapsulate: "option-1001.1",
		},
		{
			AlwaysSend: false,
			Code:       2,
			Fields: []dbmodel.DHCPOptionField{
				{
					FieldType: dhcpmodel.Uint32Field,
					Values:    []any{22},
				},
			},
			Space:       "option-1001.1",
			Encapsulate: "option-1001.1.2",
		},
	}

	// Convert.
	restOptions := rapi.unflattenDHCPOptions(options, "", 0)
	require.Len(t, restOptions, 1)
	require.True(t, restOptions[0].AlwaysSend)
	require.EqualValues(t, 1001, restOptions[0].Code)
	require.EqualValues(t, "option-1001", restOptions[0].Encapsulate)
	require.Len(t, restOptions[0].Fields, 1)
	require.Equal(t, dhcpmodel.StringField, restOptions[0].Fields[0].FieldType)
	require.Len(t, restOptions[0].Fields[0].Values, 1)
	require.Equal(t, "foo", restOptions[0].Fields[0].Values[0])
	require.Len(t, restOptions[0].Options, 1)

	// First level suboption.
	require.Len(t, restOptions[0].Options, 1)
	require.False(t, restOptions[0].Options[0].AlwaysSend)
	require.EqualValues(t, 1, restOptions[0].Options[0].Code)
	require.Len(t, restOptions[0].Options[0].Fields, 1)
	require.Equal(t, dhcpmodel.Uint8Field, restOptions[0].Options[0].Fields[0].FieldType)
	require.Len(t, restOptions[0].Options[0].Fields[0].Values, 1)
	require.Equal(t, "11", restOptions[0].Options[0].Fields[0].Values[0])
	require.Equal(t, "option-1001.1", restOptions[0].Options[0].Encapsulate)

	// Second level suboption.
	require.Len(t, restOptions[0].Options[0].Options, 1)
	require.False(t, restOptions[0].Options[0].Options[0].AlwaysSend)
	require.EqualValues(t, 2, restOptions[0].Options[0].Options[0].Code)
	require.Len(t, restOptions[0].Options[0].Options[0].Fields, 1)
	require.Equal(t, dhcpmodel.Uint32Field, restOptions[0].Options[0].Options[0].Fields[0].FieldType)
	require.Len(t, restOptions[0].Options[0].Options[0].Fields[0].Values, 1)
	require.Equal(t, "22", restOptions[0].Options[0].Options[0].Fields[0].Values[0])
	require.Equal(t, "option-1001.1.2", restOptions[0].Options[0].Options[0].Encapsulate)
}

// Test that option field values of different types are correctly converted
// into REST API format.
func TestUnflattenDHCPOptionsVariousFieldTypes(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	type test struct {
		testName    string
		fieldType   string
		inputValues []any
		values      []string
	}
	tests := []test{
		{"binary", dhcpmodel.BinaryField, []any{"010203"}, []string{"010203"}},
		{"string", dhcpmodel.StringField, []any{"foo"}, []string{"foo"}},
		{"bool", dhcpmodel.BoolField, []any{true}, []string{"true"}},
		{"uint8", dhcpmodel.Uint8Field, []any{111}, []string{"111"}},
		{"uint16", dhcpmodel.Uint16Field, []any{65536}, []string{"65536"}},
		{"uint32", dhcpmodel.Uint32Field, []any{14294967295}, []string{"14294967295"}},
		{"int8", dhcpmodel.Int8Field, []any{-111}, []string{"-111"}},
		{"int16", dhcpmodel.Int16Field, []any{-2323}, []string{"-2323"}},
		{"int32", dhcpmodel.Int32Field, []any{-235}, []string{"-235"}},
		{"ipv4-address", dhcpmodel.IPv4AddressField, []any{"192.0.1.2"}, []string{"192.0.1.2"}},
		{"ipv6-address", dhcpmodel.IPv6AddressField, []any{"3001::"}, []string{"3001::"}},
		{"ipv6-prefix", dhcpmodel.IPv6PrefixField, []any{"3001::", "64"}, []string{"3001::", "64"}},
		{"psid", dhcpmodel.PsidField, []any{16111, 12}, []string{"16111", "12"}},
		{"fqdn", dhcpmodel.FqdnField, []any{"foo.example.org."}, []string{"foo.example.org."}},
	}
	for _, test := range tests {
		fieldType := test.fieldType
		inputValues := test.inputValues
		values := test.values
		t.Run(test.testName, func(t *testing.T) {
			options := []dbmodel.DHCPOption{
				{
					Code: 1001,
					Fields: []dbmodel.DHCPOptionField{
						{
							FieldType: fieldType,
							Values:    inputValues,
						},
					},
				},
			}
			restOptions := rapi.unflattenDHCPOptions(options, "", 0)
			require.Len(t, restOptions, 1)
			require.Len(t, restOptions[0].Fields, 1)
			require.Equal(t, fieldType, restOptions[0].Fields[0].FieldType)
			require.Equal(t, values, restOptions[0].Fields[0].Values)
		})
	}
}

// Test that maximum recursion level is respected while converting DHCP options
// to the REST API format.
func TestUnflattenDHCPOptionsRecursionLevel(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	options := []dbmodel.DHCPOption{
		{
			Code:        1001,
			Encapsulate: "option-1001",
		},
		{
			Code:        1,
			Space:       "option-1001",
			Encapsulate: "option-1001.1",
		},
		{
			Code:        2,
			Space:       "option-1001.1",
			Encapsulate: "option-1001.1.2",
		},
		{
			Code:        3,
			Space:       "option-1001.1.2",
			Encapsulate: "option-1001.1.3",
		},
	}

	restOptions := rapi.unflattenDHCPOptions(options, "", 0)
	require.Len(t, restOptions, 1)
	require.Len(t, restOptions[0].Options, 1)
	require.Len(t, restOptions[0].Options[0].Options, 1)
	require.Zero(t, restOptions[0].Options[0].Options[0].Options)
}
