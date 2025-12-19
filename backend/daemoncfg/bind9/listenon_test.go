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
	require.True(t, (*listenOnClauses)[0].Includes("127.0.0.1"))
	require.False(t, (*listenOnClauses)[0].Includes("0.0.0.0"))
	require.False(t, (*listenOnClauses)[0].Includes("::1"))
	require.False(t, (*listenOnClauses)[0].Includes("::"))
}

// Test that the default listen-on clause is returned for port 53.
func TestGetMatchingListenOnDefault(t *testing.T) {
	listenOnClauses := GetDefaultListenOnClauses()
	listenOn := listenOnClauses.GetMatchingListenOnClause(53)
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
	listenOn := listenOnClauses.GetMatchingListenOnClause(53)
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
	listenOn := listenOnClauses.GetMatchingListenOnClause(53)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "127.0.0.1", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), listenOn.GetPort())
}

// Test that a listen-on clause with any ACL is preferred over a listen-on clause
// with a specific non-loopback address.
func TestGetMatchingListenOnAnyAddress(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "192.0.2.1"}},
			},
		},
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "any"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOnClause(53)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "any", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), listenOn.GetPort())
}

// Test that a listen-on clause that includes none is not returned.
func TestGetMatchingListenOnNone(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{
					{IPAddressOrACLName: "127.0.0.1"},
					{IPAddressOrACLName: "none"},
				},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOnClause(53)
	require.Nil(t, listenOn)
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
	listenOn := listenOnClauses.GetMatchingListenOnClause(853)
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
			Variant: "listen-on-v6",
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "2001:db8:1::1"}},
			},
		},
		&ListenOn{
			Variant: "listen-on-v6",
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "::"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOnClause(53)
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
			Variant: "listen-on-v6",
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "2001:db8:1::1"}},
			},
		},
		&ListenOn{
			Variant: "listen-on-v6",
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "::1"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOnClause(53)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "::1", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), listenOn.GetPort())
}

// Test that port number affects selection of the listen-on-v6 clause.
func TestGetMatchingListenOnMultipleLoopbackAddressPortNumberIPv6(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			Variant: "listen-on-v6",
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "2001:db8:1::1"}},
			},
			Port: storkutil.Ptr(int64(853)),
		},
		&ListenOn{
			Variant: "listen-on-v6",
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "::1"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOnClause(853)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "2001:db8:1::1", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(853), listenOn.GetPort())
}

// Test that a listen-on-v6 clause with any ACL is preferred over a listen-on clause
// with a specific non-loopback address.
func TestGetMatchingListenOnAnyAddressIPv6(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			Variant: "listen-on-v6",
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "2001:db8:1::1"}},
			},
		},
		&ListenOn{
			Variant: "listen-on-v6",
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{{IPAddressOrACLName: "any"}},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOnClause(53)
	require.NotNil(t, listenOn)
	require.Len(t, listenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "any", listenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.Equal(t, int64(53), listenOn.GetPort())
}

// Test that a listen-on-v6 clause that includes none is not returned.
func TestGetMatchingListenOnNoneIPv6(t *testing.T) {
	listenOnClauses := ListenOnClauses{
		&ListenOn{
			Variant: "listen-on-v6",
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{
					{IPAddressOrACLName: "::1"},
					{IPAddressOrACLName: "none"},
				},
			},
		},
	}
	listenOn := listenOnClauses.GetMatchingListenOnClause(53)
	require.Nil(t, listenOn)
}

// Test that the listen-on clause is formatted correctly.
func TestListenOnFormat(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on",
		Port:    storkutil.Ptr(int64(853)),
		Proxy:   storkutil.Ptr("plain"),
		TLS:     storkutil.Ptr("domain.name"),
		HTTP:    storkutil.Ptr("myserver"),
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{{IPAddressOrACLName: "127.0.0.1"}},
		},
	}
	output := listenOn.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `listen-on port 853 proxy plain tls domain.name http myserver { "127.0.0.1"; };`, output)
}

// Test that the listen-on clause is formatted correctly when no optional flags are specified.
func TestListenOnFormatNoOptionalFlags(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{{IPAddressOrACLName: "127.0.0.1"}},
		},
	}
	output := listenOn.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `listen-on { "127.0.0.1"; };`, output)
}

// Test that the listen-on-v6 clause is formatted correctly.
func TestListenOnFormatIPv6(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on-v6",
		Port:    storkutil.Ptr(int64(853)),
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{{IPAddressOrACLName: "::1"}},
		},
	}
	output := listenOn.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `listen-on-v6 port 853 { "::1"; };`, output)
}

// Test that serializing a listen-on clause with nil values does not panic.
func TestListenOnFormatNilValues(t *testing.T) {
	listenOn := &ListenOn{}
	require.NotPanics(t, func() { listenOn.getFormattedOutput(nil) })
}

// Test that local loopback address is preferred over other addresses.
func TestGetPreferredIPAddressLocalLoopback(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{IPAddressOrACLName: "192.0.2.1"},
				{IPAddressOrACLName: "key"},
				{IPAddressOrACLName: "127.0.0.1"},
			},
		},
	}
	allowTransferMatchList := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddressOrACLName: "192.0.2.1"},
		},
	}
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransferMatchList)
	require.Equal(t, "127.0.0.1", preferredIPAddress)
}

// Test that zero address is preferred over other addresses, and the
// local loopback address is returned in such a case.
func TestGetPreferredIPAddressZeroAddress(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{IPAddressOrACLName: "192.0.2.1"},
				{IPAddressOrACLName: "key"},
				{IPAddressOrACLName: "0.0.0.0"},
			},
		},
	}
	allowTransferMatchList := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddressOrACLName: "192.0.2.1"},
		},
	}
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransferMatchList)
	require.Equal(t, "127.0.0.1", preferredIPAddress)
}

// Test that local loopback address is returned when any keyword is
// specified.
func TestGetPreferredIPAddressAny(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{IPAddressOrACLName: "192.0.2.1"},
				{IPAddressOrACLName: "key"},
				{IPAddressOrACLName: "any"},
			},
		},
	}
	allowTransferMatchList := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddressOrACLName: "192.0.2.1"},
		},
	}
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransferMatchList)
	require.Equal(t, "127.0.0.1", preferredIPAddress)
}

// Test that an IP address is returned when preceding match list element
// is not an IP address.
func TestGetPreferredIPAddressNotAnIPAddress(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{IPAddressOrACLName: "key"},
				{IPAddressOrACLName: "192.0.2.1"},
			},
		},
	}
	allowTransferMatchList := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddressOrACLName: "key"},
			{IPAddressOrACLName: "192.0.2.1"},
		},
	}
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransferMatchList)
	require.Equal(t, "192.0.2.1", preferredIPAddress)
}

// Test that local loopback address is preferred over other addresses.
func TestGetPreferredIPAddressLocalLoopbackIPv6(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on-v6",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{IPAddressOrACLName: "2001:db8:1::1"},
				{IPAddressOrACLName: "key"},
				{IPAddressOrACLName: "::1"},
			},
		},
	}
	allowTransferMatchList := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddressOrACLName: "2001:db8:1::1"},
		},
	}
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransferMatchList)
	require.Equal(t, "::1", preferredIPAddress)
}

// Test that zero address is preferred over other addresses, and the
// local loopback address is returned in such a case.
func TestGetPreferredIPAddressZeroAddressIPv6(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on-v6",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{IPAddressOrACLName: "2001:db8:1::1"},
				{IPAddressOrACLName: "key"},
				{IPAddressOrACLName: "::"},
			},
		},
	}
	allowTransferMatchList := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddressOrACLName: "2001:db8:1::1"},
		},
	}
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransferMatchList)
	require.Equal(t, "::1", preferredIPAddress)
}

// Test that local loopback address is returned when any keyword is
// specified.
func TestGetPreferredIPAddressAnyIPv6(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on-v6",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{IPAddressOrACLName: "2001:db8:1::1"},
				{IPAddressOrACLName: "key"},
				{IPAddressOrACLName: "any"},
			},
		},
	}
	allowTransferMatchList := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddressOrACLName: "2001:db8:1::1"},
		},
	}
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransferMatchList)
	require.Equal(t, "::1", preferredIPAddress)
}

// Test that an IP address is returned when preceding match list element
// is not an IP address.
func TestGetPreferredIPAddressNotAnIPAddressIPv6(t *testing.T) {
	listenOn := &ListenOn{
		Variant: "listen-on-v6",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{IPAddressOrACLName: "key"},
				{IPAddressOrACLName: "2001:db8:1::1"},
			},
		},
	}
	allowTransferMatchList := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddressOrACLName: "key"},
			{IPAddressOrACLName: "2001:db8:1::1"},
		},
	}
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransferMatchList)
	require.Equal(t, "2001:db8:1::1", preferredIPAddress)
}
