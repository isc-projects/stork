package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that zone transfer is disabled when allow-transfer is not specified.
func TestAllowTransferIsDisabledEmpty(t *testing.T) {
	at := &AllowTransfer{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{},
		},
	}
	require.True(t, at.IsDisabled())
}

// Test that zone transfer is disabled when allow-transfer is specified as none.
func TestAllowTransferIsDisabledNone(t *testing.T) {
	at := &AllowTransfer{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					IPAddressOrACLName: "none",
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
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					IPAddressOrACLName: "none",
				},
				{
					IPAddressOrACLName: "127.0.0.1",
				},
			},
		},
	}
	require.False(t, at.IsDisabled())
}

// Test that the allow-transfer clause with the port and transport options
// is formatted correctly.
func TestAllowTransferFormat(t *testing.T) {
	allowTransfer := &AllowTransfer{
		Port:      storkutil.Ptr(int64(53)),
		Transport: storkutil.Ptr("tcp"),
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					IPAddressOrACLName: "127.0.0.1",
				},
				{
					KeyID: "test-key",
				},
			},
		},
	}
	output := allowTransfer.getFormattedOutput(nil)
	require.NotNil(t, output)
	cfgEq(t, `allow-transfer port 53 transport tcp { "127.0.0.1"; key "test-key"; };`, output)
}

// Test that the allow-transfer clause without the port and transport options
// is formatted correctly.
func TestAllowTransferFormatNoPortTransport(t *testing.T) {
	allowTransfer := &AllowTransfer{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					IPAddressOrACLName: "127.0.0.1",
				},
			},
		},
	}
	output := allowTransfer.getFormattedOutput(nil)
	require.NotNil(t, output)
	cfgEq(t, `allow-transfer { "127.0.0.1"; };`, output)
}

// Test that serializing an allow-transfer clause with nil values does not panic.
func TestAllowTransferFormatNilValues(t *testing.T) {
	allowTransfer := &AllowTransfer{}
	require.NotPanics(t, func() { allowTransfer.getFormattedOutput(nil) })
}
