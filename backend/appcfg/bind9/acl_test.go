package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the acl statement is formatted correctly.
func TestACLFormat(t *testing.T) {
	acl := &ACL{
		Name: "trusted-networks",
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					IPAddressOrACLName: "127.0.0.1",
				},
			},
		},
	}
	output := acl.getFormattedOutput(nil)
	require.NotNil(t, output)
	cfgEq(t, `acl "trusted-networks" { "127.0.0.1"; };`, output)
}

// Test that serializing an acl statement with nil values does not panic.
func TestACLFormatNilValues(t *testing.T) {
	acl := &ACL{}
	require.NotPanics(t, func() { acl.getFormattedOutput(nil) })
}
