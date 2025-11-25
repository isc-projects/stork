package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test checking if the options contains no-parse directives.
func TestOptionsHasNoParse(t *testing.T) {
	options := &Options{
		Clauses: []*OptionClause{
			{NoParse: &NoParse{}},
		},
	}
	require.True(t, options.HasNoParse())
}

// Test checking if the options does not contain no-parse directives.
func TestOptionsHasNoParseNone(t *testing.T) {
	options := &Options{}
	require.False(t, options.HasNoParse())
}

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
					Variant: "listen-on",
					Port:    storkutil.Ptr(int64(53)),
				},
			},
			{
				AllowTransfer: &AllowTransfer{
					Port: storkutil.Ptr(int64(55)),
				},
			},
			{
				ListenOn: &ListenOn{
					Variant: "listen-on-v6",
					Port:    storkutil.Ptr(int64(54)),
				},
			},
			{
				ListenOn: &ListenOn{
					Variant: "listen-on",
					Port:    storkutil.Ptr(int64(56)),
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

// Test getting the response-policy clause from options.
func TestOptionsGetResponsePolicy(t *testing.T) {
	options := &Options{
		Clauses: []*OptionClause{
			{
				AllowTransfer: &AllowTransfer{
					Port: storkutil.Ptr(int64(53)),
				},
			},
			{
				ResponsePolicy: &ResponsePolicy{
					Zones: []*ResponsePolicyZone{
						{
							Zone: "rpz.example.com",
						},
					},
				},
			},
		},
	}
	responsePolicy := options.GetResponsePolicy()
	require.NotNil(t, responsePolicy)
	require.Len(t, responsePolicy.Zones, 1)
}

// Test that serializing an options statement with nil values does not panic.
func TestOptionsFormatNilValues(t *testing.T) {
	options := &Options{}
	require.NotPanics(t, func() { options.getFormattedOutput(nil) })
}

// Test that serializing an option clause with nil values does not panic.
func TestOptionClauseFormatNilValues(t *testing.T) {
	optionClause := &OptionClause{}
	require.NotPanics(t, func() { optionClause.getFormattedOutput(nil) })
}
