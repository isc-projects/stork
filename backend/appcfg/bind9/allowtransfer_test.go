package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that zone transfer is disabled when allow-transfer is not specified.
func TestAllowTransferIsDisabledEmpty(t *testing.T) {
	at := &AllowTransfer{
		AdressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{},
		},
	}
	require.True(t, at.IsDisabled())
}

// Test that zone transfer is disabled when allow-transfer is specified as none.
func TestAllowTransferIsDisabledNone(t *testing.T) {
	at := &AllowTransfer{
		AdressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					ACLName: "none",
				},
			},
		},
	}
	require.True(t, at.IsDisabled())
}

// Test that zone transfer is enabled when allow-transfer is specified as a valid ACL,
// even if another ACL contains none.
func TestAllowTransferIsNotDisabled(t *testing.T) {
	at := &AllowTransfer{
		AdressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					ACLName: "none",
				},
				{
					IPAddress: "127.0.0.1",
				},
			},
		},
	}
	require.False(t, at.IsDisabled())
}
