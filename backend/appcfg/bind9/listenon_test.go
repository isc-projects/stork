package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test getting default listen-on clause. It must contain an IPv4 local
// loopback address and port 53.
func TestGetDefaultListenOnClauses(t *testing.T) {
	listenOnClauses := GetDefaultListenOnClauses()
	require.Len(t, *listenOnClauses, 1)
	require.Len(t, (*listenOnClauses)[0].AddressMatchList.Elements, 1)
	require.Equal(t, "127.0.0.1", (*listenOnClauses)[0].AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), (*listenOnClauses)[0].GetPort())
	require.True(t, (*listenOnClauses)[0].IncludesIPAddress("127.0.0.1"))
	require.False(t, (*listenOnClauses)[0].IncludesIPAddress("0.0.0.0"))
	require.False(t, (*listenOnClauses)[0].IncludesIPAddress("::1"))
	require.False(t, (*listenOnClauses)[0].IncludesIPAddress("::"))
}

// Test that the default listen-on clause is returned for port 53.
func TestGetMatchingListenOnDefault(t *testing.T) {
	listenOnClauses := GetDefaultListenOnClauses()
	listenOn := listenOnClauses.GetMatchingListenOn(53)
	require.NotNil(t, listenOn)
	require.Len(t, *listenOnClauses, 1)
	require.Len(t, (*listenOnClauses)[0].AddressMatchList.Elements, 1)
	require.Equal(t, "127.0.0.1", (*listenOnClauses)[0].AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), listenOn.GetPort())
}

// Test that getting a listen-on clause when multiple clauses exist and the
// second one has the preferred IP address.
func TestGetMatchingListenOnMultipleZeroAddress(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "192.0.2.1"}},
			},
		},
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "0.0.0.0"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOn(53)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "0.0.0.0", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), listenOn.GetPort())
}

// Test that getting a listen-on clause when multiple clauses exist and the
// second one has the preferred IP address.
func TestGetMatchingListenOnMultipleLoopbackAddress(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "192.0.2.1"}},
			},
		},
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "127.0.0.1"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOn(53)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "127.0.0.1", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), listenOn.GetPort())
}

// Test that non-standard port number affects selection of the listen-on
// clause.
func TestGetMatchingListenOnMultipleLoopbackAddressPortNumber(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "192.0.2.1"}},
			},
			Port: storkutil.Ptr(int64(853)),
		},
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "127.0.0.1"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOn(853)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "192.0.2.1", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(853), listenOn.GetPort())
}

// Test that getting a listen-on-v6 clause when multiple clauses exist and the
// second one has the preferred IP address.
func TestGetMatchingListenOnMultipleZeroAddressIPv6(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "2001:db8:1::1"}},
			},
		},
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "::"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOn(53)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "::", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), listenOn.GetPort())
}

// Test that getting a listen-on-v6 clause when multiple clauses exist and the
// second one has the preferred IP address.
func TestGetMatchingListenOnMultipleLoopbackAddressIPv6(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "2001:db8:1::1"}},
			},
		},
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "::1"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOn(53)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "::1", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), listenOn.GetPort())
}

// Test that port number affects selection of the listen-on-v6 clause.
func TestGetMatchingListenOnMultipleLoopbackAddressPortNumberIPv6(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "2001:db8:1::1"}},
			},
			Port: storkutil.Ptr(int64(853)),
		},
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "::1"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOn(853)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "2001:db8:1::1", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(853), listenOn.GetPort())
}
