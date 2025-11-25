package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test checking if the zone contains no-parse directives.
func TestZoneHasNoParse(t *testing.T) {
	zone := &Zone{
		Clauses: []*ZoneClause{
			{NoParse: &NoParse{}},
		},
	}
	require.True(t, zone.HasNoParse())
}

// Test checking if the zone does not contain no-parse directives.
func TestZoneHasNoParseNone(t *testing.T) {
	zone := &Zone{}
	require.False(t, zone.HasNoParse())
}

// Tests that allow-transfer is returned when specified.
func TestZoneGetAllowTransfer(t *testing.T) {
	zone := &Zone{
		Clauses: []*ZoneClause{
			{
				AllowTransfer: &AllowTransfer{
					Port: storkutil.Ptr(int64(53)),
				},
			},
			{
				Option: &Option{},
			},
		},
	}
	allowTransfer := zone.GetAllowTransfer()
	require.NotNil(t, allowTransfer)
	require.Equal(t, int64(53), *allowTransfer.Port)
}

// Tests that allow-transfer is not returned when not specified.
func TestZoneGetAllowTransferLacking(t *testing.T) {
	zone := &Zone{
		Clauses: []*ZoneClause{
			{
				Option: &Option{},
			},
		},
	}
	allowTransfer := zone.GetAllowTransfer()
	require.Nil(t, allowTransfer)
}

// Test that the zone is formatted correctly.
func TestZoneGetFormattedOutput(t *testing.T) {
	zone := &Zone{
		Name:  "example.com",
		Class: "IN",
		Clauses: []*ZoneClause{
			{
				Option: &Option{
					Identifier: "type",
					Switches: []OptionSwitch{
						{
							IdentSwitch: storkutil.Ptr("forward"),
						},
					},
				},
			},
		},
	}
	output := zone.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `zone "example.com" IN {
		type forward;
	};`, output)
}

// Test that serializing a zone with nil values does not panic.
func TestZoneFormatNilValues(t *testing.T) {
	zone := &Zone{}
	require.NotPanics(t, func() { zone.getFormattedOutput(nil) })
}

// Test that serializing a zone clause with nil values does not panic.
func TestZoneClauseFormatNilValues(t *testing.T) {
	zoneClause := &ZoneClause{}
	require.NotPanics(t, func() { zoneClause.getFormattedOutput(nil) })
}
