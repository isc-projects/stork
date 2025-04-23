package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

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
