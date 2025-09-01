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
