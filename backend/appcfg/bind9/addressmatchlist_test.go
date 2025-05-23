package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test checking that the address match list excludes the specified IP address.
func TestAddressMatchListExcludesIPAddress(t *testing.T) {
	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddress: "127.0.0.1", Negation: true},
			{IPAddress: "::1", Negation: false},
		},
	}
	require.True(t, aml.ExcludesIPAddress("127.0.0.1"))
	require.False(t, aml.ExcludesIPAddress("::1"))
	require.False(t, aml.ExcludesIPAddress("192.168.1.1"))
}

// Test checking that the address match list excludes the specified IP address
// when the ACL is none.
func TestAddressMatchListExcludesIPAddressWithNone(t *testing.T) {
	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{ACLName: "none"},
		},
	}
	require.True(t, aml.ExcludesIPAddress("127.0.0.1"))
	require.True(t, aml.ExcludesIPAddress("::1"))
	require.True(t, aml.ExcludesIPAddress("192.168.1.1"))
}

// Test checking that the address match list does not exclude the specified IP
// address when the ACL is any.
func TestAddressMatchListExcludesIPAddressWithAny(t *testing.T) {
	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{ACLName: "any"},
		},
	}
	require.False(t, aml.ExcludesIPAddress("127.0.0.1"))
	require.False(t, aml.ExcludesIPAddress("::1"))
	require.False(t, aml.ExcludesIPAddress("192.168.1.1"))
}
