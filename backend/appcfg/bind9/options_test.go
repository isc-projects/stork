package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test getting the allow-transfer clause from options.
func TestOptionsGetAllowTransferPort(t *testing.T) {
	options := &Options{
		Clauses: []*OptionClause{
			{
				ListenOn: &ListenOn{
					Port: storkutil.Ptr(int64(53)),
				},
			},
			{
				AllowTransfer: &AllowTransfer{
					Port: storkutil.Ptr(int64(54)),
				},
			},
		},
	}
	allowTransfer := options.GetAllowTransfer()
	require.NotNil(t, allowTransfer)
	require.Equal(t, int64(54), *allowTransfer.Port)
}

// Test getting the listen-on and listen-on-v6 clauses from options.
func TestOptionsGetListenOnSet(t *testing.T) {
	options := &Options{
		Clauses: []*OptionClause{
			{
				ListenOn: &ListenOn{
					Port: storkutil.Ptr(int64(53)),
				},
			},
			{
				AllowTransfer: &AllowTransfer{
					Port: storkutil.Ptr(int64(55)),
				},
			},
			{
				ListenOnV6: &ListenOn{
					Port: storkutil.Ptr(int64(54)),
				},
			},
			{
				ListenOn: &ListenOn{
					Port: storkutil.Ptr(int64(56)),
				},
			},
		},
	}
	listenOnSet := options.GetListenOnSet()
	require.NotNil(t, listenOnSet)
	require.Len(t, *listenOnSet, 3)
	require.Equal(t, int64(53), (*listenOnSet)[0].GetPort())
	require.Equal(t, int64(54), (*listenOnSet)[1].GetPort())
	require.Equal(t, int64(56), (*listenOnSet)[2].GetPort())
}
