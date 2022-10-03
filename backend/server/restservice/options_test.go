package restservice

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
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
					Options: []*models.DHCPOption{
						{
							Code:        5,
							Encapsulate: "option-1002.4.5",
							Fields: []*models.DHCPOptionField{
								{
									FieldType: keaconfig.StringField,
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
											FieldType: keaconfig.Uint32Field,
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
							FieldType: keaconfig.Uint8Field,
							Values:    []string{"1"},
						},
					},
					Options: []*models.DHCPOption{
						{
							Code:        93,
							Encapsulate: "option-94.89.93",
							Fields: []*models.DHCPOptionField{
								{
									FieldType: keaconfig.Uint8Field,
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
					FieldType: keaconfig.StringField,
					Values:    []any{"foo"},
				},
			},
		},
		{
			AlwaysSend: false,
			Code:       1,
			Fields: []dbmodel.DHCPOptionField{
				{
					FieldType: keaconfig.Uint8Field,
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
					FieldType: keaconfig.Uint32Field,
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
	require.Equal(t, keaconfig.StringField, restOptions[0].Fields[0].FieldType)
	require.Len(t, restOptions[0].Fields[0].Values, 1)
	require.Equal(t, "foo", restOptions[0].Fields[0].Values[0])
	require.Len(t, restOptions[0].Options, 1)

	// First level suboption.
	require.Len(t, restOptions[0].Options, 1)
	require.False(t, restOptions[0].Options[0].AlwaysSend)
	require.EqualValues(t, 1, restOptions[0].Options[0].Code)
	require.Len(t, restOptions[0].Options[0].Fields, 1)
	require.Equal(t, keaconfig.Uint8Field, restOptions[0].Options[0].Fields[0].FieldType)
	require.Len(t, restOptions[0].Options[0].Fields[0].Values, 1)
	require.Equal(t, "11", restOptions[0].Options[0].Fields[0].Values[0])
	require.Equal(t, "option-1001.1", restOptions[0].Options[0].Encapsulate)

	// Second level suboption.
	require.Len(t, restOptions[0].Options[0].Options, 1)
	require.False(t, restOptions[0].Options[0].Options[0].AlwaysSend)
	require.EqualValues(t, 2, restOptions[0].Options[0].Options[0].Code)
	require.Len(t, restOptions[0].Options[0].Options[0].Fields, 1)
	require.Equal(t, keaconfig.Uint32Field, restOptions[0].Options[0].Options[0].Fields[0].FieldType)
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
		{"hex-bytes", keaconfig.HexBytesField, []any{"010203"}, []string{"010203"}},
		{"string", keaconfig.StringField, []any{"foo"}, []string{"foo"}},
		{"bool", keaconfig.BoolField, []any{true}, []string{"true"}},
		{"uint8", keaconfig.Uint8Field, []any{111}, []string{"111"}},
		{"uint16", keaconfig.Uint16Field, []any{65536}, []string{"65536"}},
		{"uint32", keaconfig.Uint32Field, []any{14294967295}, []string{"14294967295"}},
		{"ipv4-address", keaconfig.IPv4AddressField, []any{"192.0.1.2"}, []string{"192.0.1.2"}},
		{"ipv6-address", keaconfig.IPv6AddressField, []any{"3001::"}, []string{"3001::"}},
		{"ipv6-prefix", keaconfig.IPv6PrefixField, []any{"3001::", "64"}, []string{"3001::", "64"}},
		{"psid", keaconfig.PsidField, []any{16111, 12}, []string{"16111", "12"}},
		{"fqdn", keaconfig.FqdnField, []any{"foo.example.org."}, []string{"foo.example.org."}},
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
