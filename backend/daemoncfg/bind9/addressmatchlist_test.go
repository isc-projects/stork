package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=bind9config -destination=addressmatchlistglobalconfigmock_test.go -mock_names=AddressMatchListGlobalConfigAccessor=MockAddressMatchListGlobalConfigAccessor isc.org/stork/daemoncfg/bind9 AddressMatchListGlobalConfigAccessor

// Test that all DNS clients are denied when address match list contains
// only the none ACL. No client matches none ACL.
func TestAddressMatchListAllowNone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{IPAddressOrACLName: "none"},
		},
	}

	for _, ipAddress := range []string{"127.0.0.1", "::1"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, ipAddress, "")
		require.NoError(t, err)
		require.False(t, isAllowed, "IP address %s should be denied", ipAddress)
	}

	for _, keyID := range []string{"key1", "key2"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "", keyID)
		require.NoError(t, err)
		require.False(t, isAllowed, "Key %s should be denied", keyID)
	}
}

// Test that all clients are denied when address match list contains
// the !none ACL. In order to allow an address, there must be at least
// one positive match.
func TestAddressMatchListDenyNone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation:           true,
				IPAddressOrACLName: "none",
			},
		},
	}
	for _, ipAddress := range []string{"127.0.0.1", "::1"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, ipAddress, "")
		require.NoError(t, err)
		require.False(t, isAllowed, "IP address %s should be denied", ipAddress)
	}

	for _, keyID := range []string{"key1", "key2"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "", keyID)
		require.NoError(t, err)
		require.False(t, isAllowed, "Key %s should be denied", keyID)
	}
}

// Test that a client is allowed when address match list contains
// the !none ACL and the allowed address or key ID is in the address
// match list. No address matches the none ACL, so negation never
// fires, and the matching process moves to next elements which matches
// ::1 and key ID but not the 127.0.0.1.
func TestAddressMatchListDenyNoneAllowAddressAndKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation:           true,
				IPAddressOrACLName: "none",
			},
			{
				IPAddressOrACLName: "::1",
			},
			{
				KeyID: "key1",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "::1", "")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key1")
	require.NoError(t, err)
	require.True(t, isAllowed)
}

// Test that all clients are allowed when address match list contains
// only the any ACL.
func TestAddressMatchListAllowAny(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				IPAddressOrACLName: "any",
			},
		},
	}
	for _, ipAddress := range []string{"127.0.0.1", "::1"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, ipAddress, "")
		require.NoError(t, err)
		require.True(t, isAllowed, "IP address %s should be allowed", ipAddress)
	}

	for _, keyID := range []string{"key1", "key2"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "", keyID)
		require.NoError(t, err)
		require.True(t, isAllowed, "Key %s should be allowed", keyID)
	}
}

// Test that all clients are denied when address match list contains
// the !any ACL. All clients match the any ACL, so negation fires and
// all clients are denied.
func TestAddressMatchListIsIPAddressAndKeyAllowedDenyAny(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation:           true,
				IPAddressOrACLName: "any",
			},
		},
	}
	for _, ipAddress := range []string{"127.0.0.1", "::1"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, ipAddress, "")
		require.NoError(t, err)
		require.False(t, isAllowed, "IP address %s should be denied", ipAddress)
	}

	for _, keyID := range []string{"key1", "key2"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "", keyID)
		require.NoError(t, err)
		require.False(t, isAllowed, "Key %s should be denied", keyID)
	}
}

// Test that all clients are denied even if selected address or key is
// allowed. The !any ACL negates the match of the ::1 address and key1 key,
// so even though ::1 and key1 are later allowed, all clients are denied.
func TestAddressMatchListDenyAnyAllowAddressAndKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation:           true,
				IPAddressOrACLName: "any",
			},
			{
				IPAddressOrACLName: "::1",
			},
			{
				KeyID: "key1",
			},
		},
	}
	for _, ipAddress := range []string{"127.0.0.1", "::1"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, ipAddress, "")
		require.NoError(t, err)
		require.False(t, isAllowed, "IP address %s should be denied", ipAddress)
	}

	for _, keyID := range []string{"key1", "key2"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "", keyID)
		require.NoError(t, err)
		require.False(t, isAllowed, "Key %s should be denied", keyID)
	}
}

// Test that a client is allowed when its address or key is explicitly listed
// in the address match list.
func TestAddressMatchListAllowAddressAndKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				IPAddressOrACLName: "127.0.0.1",
			},
			{
				KeyID: "key1",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key1")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.2", "")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key2")
	require.NoError(t, err)
	require.False(t, isAllowed)
}

// Test that all clients are denied when address match list contains
// one denied address. The 127.0.0.1 is explicitly denied but other
// addresses have no positive matches, so they are also denied.
func TestAddressMatchListDenyAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation:           true,
				IPAddressOrACLName: "127.0.0.1",
			},
		},
	}
	for _, ipAddress := range []string{"127.0.0.1", "::1"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, ipAddress, "")
		require.NoError(t, err)
		require.False(t, isAllowed, "IP address %s should be denied", ipAddress)
	}
}

// Test that all clients are denied when address match list contains
// one denied key. The key1 is explicitly denied but other keys have
// no positive matches, so they are also denied.
func TestAddressMatchListDenyKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation: true,
				KeyID:    "key1",
			},
		},
	}
	for _, keyID := range []string{"key1", "key2"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "", keyID)
		require.NoError(t, err)
		require.False(t, isAllowed, "Key %s should be denied", keyID)
	}
}

// Test that non-denied addresses are allowed when address match list contains
// any ACL after a denied address.
func TestAddressMatchListDenyAddressAllowAny(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation:           true,
				IPAddressOrACLName: "127.0.0.1",
			},
			{
				IPAddressOrACLName: "any",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "::1", "")
	require.NoError(t, err)
	require.True(t, isAllowed)
}

// Test that non-denied keys are allowed when address match list contains
// any ACL after a denied key.
func TestAddressMatchListDenyKeyAllowAny(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation: true,
				KeyID:    "key1",
			},
			{
				IPAddressOrACLName: "any",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key1")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key2")
	require.NoError(t, err)
	require.True(t, isAllowed)
}

// Test that an address or key specified within ACL is allowed and other
// addresses and keys are denied.
func TestAddressMatchListAllowACL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL("myacl").Return(&ACL{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					IPAddressOrACLName: "127.0.0.1",
				},
				{
					KeyID: "key1",
				},
			},
		},
	}).Times(4)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				IPAddressOrACLName: "myacl",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key1")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.2", "")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key2")
	require.NoError(t, err)
	require.False(t, isAllowed)
}

// Test that all clients are denied when the ACL is not found in the
// global config.
func TestAddressMatchListAllowACLNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL("myacl").Return(nil).Times(4)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				IPAddressOrACLName: "myacl",
			},
		},
	}

	for _, ipAddress := range []string{"127.0.0.1", "::1"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, ipAddress, "")
		require.NoError(t, err)
		require.False(t, isAllowed, "IP address %s should be denied", ipAddress)
	}

	for _, keyID := range []string{"key1", "key2"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "", keyID)
		require.NoError(t, err)
		require.False(t, isAllowed, "Key %s should be denied", keyID)
	}
}

// Test that a client is denied when its address or key is negated within
// an ACL and other addresses or keys specified within another ACL are allowed.
func TestAddressMatchListDenyInACLAllowInAnotherACL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL("acl1").Return(&ACL{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					Negation:           true,
					IPAddressOrACLName: "127.0.0.1",
				},
				{
					Negation: true,
					KeyID:    "key1",
				},
			},
		},
	}).AnyTimes()
	globalConfig.EXPECT().GetACL("acl2").Return(&ACL{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					IPAddressOrACLName: "::1",
				},
				{
					KeyID: "key2",
				},
			},
		},
	}).AnyTimes()
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				IPAddressOrACLName: "acl1",
			},
			{
				IPAddressOrACLName: "acl2",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key1")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "::1", "")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key2")
	require.NoError(t, err)
	require.True(t, isAllowed)
}

// Test that a client is denied when its address or key is negated
// within a match list and other addresses or keys specified within
// an ACL are allowed.
func TestAddressMatchListDenyACLAllowInAnotherACL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL("myacl").Return(&ACL{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					IPAddressOrACLName: "::1",
				},
				{
					KeyID: "key2",
				},
			},
		},
	}).Times(2)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation:           true,
				IPAddressOrACLName: "127.0.0.1",
			},
			{
				Negation: true,
				KeyID:    "key1",
			},
			{
				IPAddressOrACLName: "myacl",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key1")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "::1", "")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key2")
	require.NoError(t, err)
	require.True(t, isAllowed)
}

// Test that a client is allowed when its address or key is specified
// within a match list.
func TestAddressMatchListAllowInMatchList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				AddressMatchList: &AddressMatchList{
					Elements: []*AddressMatchListElement{
						{
							IPAddressOrACLName: "127.0.0.1",
						},
						{
							KeyID: "key1",
						},
					},
				},
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key1")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.2", "")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key2")
	require.NoError(t, err)
	require.False(t, isAllowed)
}

// Test that all addresses are denied when address match list contains
// a negated address or key. The 127.0.0.1 and key1 are explicitly denied
// but other addresses and keys have no positive matches, so they are also denied.
func TestAddressMatchListDenyInMatchList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation: true,
				AddressMatchList: &AddressMatchList{
					Elements: []*AddressMatchListElement{
						{
							IPAddressOrACLName: "127.0.0.1",
						},
						{
							KeyID: "key1",
						},
					},
				},
			},
		},
	}
	for _, ipAddress := range []string{"127.0.0.1", "::1"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, ipAddress, "")
		require.NoError(t, err)
		require.False(t, isAllowed, "IP address %s should be denied", ipAddress)
	}

	for _, keyID := range []string{"key1", "key2"} {
		isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "", keyID)
		require.NoError(t, err)
		require.False(t, isAllowed, "Key %s should be denied", keyID)
	}
}

// Test that a clients is denied when its address or key is negated within an
// embedded match list and other addresses and keys specified within a top-level
// match list are allowed.
func TestAddressMatchListIsIPAddressAllowedDenyAddressInMatchListAllowAddressInAnotherAddressMatchList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				AddressMatchList: &AddressMatchList{
					Elements: []*AddressMatchListElement{
						{
							Negation:           true,
							IPAddressOrACLName: "127.0.0.1",
						},
						{
							Negation: true,
							KeyID:    "key1",
						},
					},
				},
			},
			{
				IPAddressOrACLName: "::1",
			},
			{
				KeyID: "key2",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key1")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "::1", "")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "", "key2")
	require.NoError(t, err)
	require.True(t, isAllowed)
}

// Test that the address is matched against the local interface addresses when
// ACL is set to localhost.
func TestAddressMatchListAllowLocalhost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				IPAddressOrACLName: "localhost",
			},
		},
	}

	matchAddress := func(ipAddress string) bool { return ipAddress == "192.0.1.1" }

	match, negative, err := aml.match(0, globalConfig, "192.0.1.1", "", matchAddress, nil)
	require.True(t, match)
	require.False(t, negative)
	require.NoError(t, err)

	match, negative, err = aml.match(0, globalConfig, "192.0.1.2", "", matchAddress, nil)
	require.False(t, match)
	require.False(t, negative)
	require.NoError(t, err)
}

// Test that the address is matched against the local interface networks when
// ACL is set to localnets.
func TestAddressMatchListAllowLocalnets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				IPAddressOrACLName: "localnets",
			},
		},
	}

	matchAddress := func(ipAddress string) bool { return ipAddress == "192.0.1.1" }

	match, negative, err := aml.match(0, globalConfig, "192.0.1.1", "", nil, matchAddress)
	require.True(t, match)
	require.False(t, negative)
	require.NoError(t, err)

	match, negative, err = aml.match(0, globalConfig, "192.0.1.2", "", nil, matchAddress)
	require.False(t, match)
	require.False(t, negative)
	require.NoError(t, err)
}

// Test that all key IDs are collected from the address match list
// containing nested ACLs and match lists. It should return all keys
// regardless if they are negated. It should exclude duplicates.
func TestGetAllKeyIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(1).Return(&ACL{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					Negation: true,
					KeyID:    "key1",
				},
				{
					AddressMatchList: &AddressMatchList{
						Elements: []*AddressMatchListElement{
							{
								KeyID: "key2",
							},
						},
					},
				},
				{
					KeyID: "key3",
				},
			},
		},
	}).Times(1)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				IPAddressOrACLName: "acl1",
			},
			{
				KeyID: "key3",
			},
			{
				IPAddressOrACLName: "192.0.2.1",
			},
			{
				AddressMatchList: &AddressMatchList{
					Elements: []*AddressMatchListElement{
						{
							KeyID: "key4",
						},
					},
				},
			},
		},
	}
	ids, err := aml.getKeyIDs(0, globalConfig)
	require.NoError(t, err)
	require.Len(t, ids, 4)
	require.Contains(t, ids, "key1")
	require.Contains(t, ids, "key2")
	require.Contains(t, ids, "key3")
	require.Contains(t, ids, "key4")
}

// Test that the match is correctly evaluated for a complex case when both
// IP address and key are required to let the client access the server.
func TestAddressMatchListBothIPAddressAndKeyRequired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	// Express the following statement:
	// allow-transfer { !{ !127.0.0.1; any }; key stork; };
	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation: true,
				AddressMatchList: &AddressMatchList{
					Elements: []*AddressMatchListElement{
						{
							Negation:           true,
							IPAddressOrACLName: "127.0.0.1",
						},
						{
							IPAddressOrACLName: "any",
						},
					},
				},
			},
			{
				KeyID: "stork",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "stork")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "key")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.2", "stork")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.2", "key")
	require.NoError(t, err)
	require.False(t, isAllowed)
}

// Test that the match is correctly evaluated for a complex case when both
// IP address and key are required to let the client access the server.
// This is the ACL version of the other test.
func TestAddressMatchListBothIPAddressAndKeyRequiredACL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL("acl1").Return(&ACL{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					Negation:           true,
					IPAddressOrACLName: "127.0.0.1",
				},
				{
					IPAddressOrACLName: "any",
				},
			},
		},
	}).AnyTimes()
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	// Express the following statement:
	// allow-transfer { !acl1; key stork; };
	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				Negation:           true,
				IPAddressOrACLName: "acl1",
			},
			{
				KeyID: "stork",
			},
		},
	}
	isAllowed, err := aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "stork")
	require.NoError(t, err)
	require.True(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.1", "key")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.2", "stork")
	require.NoError(t, err)
	require.False(t, isAllowed)

	isAllowed, err = aml.IsIPAddressAndKeyAllowed(globalConfig, "127.0.0.2", "key")
	require.NoError(t, err)
	require.False(t, isAllowed)
}

// Test that an error is returned when matching in the address match list
// exceeds the maximum recursion level.
func TestAddressMatchListMatchTooMuchRecursion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)

	root := &AddressMatchList{
		Elements: []*AddressMatchListElement{},
	}
	ptr := root

	// Create a chain of address match lists exceeding the maximum recursion level.
	for range maxAddressMatchListRecursionLevel + 1 {
		ptr.Elements = append(ptr.Elements, &AddressMatchListElement{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{},
			},
		})
		// Move the pointer to the next level.
		ptr = ptr.Elements[0].AddressMatchList
	}
	_, _, err := root.match(0, globalConfig, "127.0.0.1", "", storkutil.IsHostIPAddress, storkutil.IsIPAddressInHostNetwork)
	require.ErrorContains(t, err, "address match list recursion level exceeded: 11")
}

// Test that an error is returned when getting the key IDs in the address match list
// exceeds the maximum recursion level.
func TestAddressMatchListGetKeyIDsTooMuchRecursion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey(gomock.Any()).Times(0)

	aml := &AddressMatchList{}
	_, err := aml.getKeyIDs(maxAddressMatchListRecursionLevel, globalConfig)
	require.NoError(t, err)

	_, err = aml.getKeyIDs(maxAddressMatchListRecursionLevel+1, globalConfig)
	require.Error(t, err)
	require.ErrorContains(t, err, "address match list recursion level exceeded: 11")
}

// Test that all keys are collected from the address match list containing nested
// ACLs and match lists. It should return all keys regardless if they are negated.
// It should exclude duplicates.
func TestAddressMatchListGetKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	globalConfig := NewMockAddressMatchListGlobalConfigAccessor(ctrl)
	globalConfig.EXPECT().GetACL(gomock.Any()).Times(0)
	globalConfig.EXPECT().GetKey("key1").Return(&Key{
		Name: "key1",
		Clauses: []*KeyClause{
			{
				Algorithm: "hmac-sha256",
				Secret:    "secret1",
			},
		},
	}).Times(1)
	globalConfig.EXPECT().GetKey("key2").Return(&Key{
		Name: "key2",
		Clauses: []*KeyClause{
			{
				Algorithm: "hmac-sha256",
				Secret:    "secret2",
			},
		},
	}).Times(1)

	aml := &AddressMatchList{
		Elements: []*AddressMatchListElement{
			{
				KeyID: "key1",
			},
			{
				Negation: true,
				KeyID:    "key2",
			},
			{
				KeyID: "key1",
			},
		},
	}
	keys, err := aml.GetKeys(globalConfig)
	require.NoError(t, err)
	require.Len(t, keys, 2)

	require.Equal(t, keys[0].Name, "key1")
	require.Equal(t, keys[0].Clauses[0].Algorithm, "hmac-sha256")
	require.Equal(t, keys[0].Clauses[0].Secret, "secret1")
	require.Equal(t, keys[1].Name, "key2")
	require.Equal(t, keys[1].Clauses[0].Algorithm, "hmac-sha256")
	require.Equal(t, keys[1].Clauses[0].Secret, "secret2")
}
