package keaconfig_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
)

//go:generate mockgen -package=keaconfig_test -destination=sharednetworkmock_test.go isc.org/stork/appcfg/kea SharedNetworkAccessor

// Returns a JSON structure with all configurable DHCPv4 shared network parameters
// in Kea. It has been initially created from the Kea's all-keys.json file and then
// slightly modified.
func getAllKeysSharedNetwork4() string {
	return `
	{
		"allocator": "random",
		"authoritative": false,
		"boot-file-name": "/dev/null",
		"client-class": "foobar",
		"ddns-generated-prefix": "myhost",
		"ddns-override-client-update": true,
		"ddns-override-no-update": true,
		"ddns-qualifying-suffix": "example.org",
		"ddns-replace-client-name": "never",
		"ddns-send-updates": true,
		"ddns-update-on-renew": true,
		"ddns-use-conflict-resolution": true,
		"hostname-char-replacement": "x",
		"hostname-char-set": "[^A-Za-z0-9.-]",
		"interface": "eth0",
		"match-client-id": true,
		"name": "my-secret-network",
		"next-server": "192.0.2.123",
		"option-data": [
			{
				"always-send": true,
				"code": 3,
				"csv-format": true,
				"data": "192.0.3.1",
				"name": "routers",
				"space": "dhcp4"
			}
		],
		"relay": {
			"ip-addresses": [
				"192.168.56.1"
			]
		},
		"rebind-timer": 41,
		"renew-timer": 31,
		"calculate-tee-times": true,
		"t1-percent": 0.5,
		"t2-percent": 0.75,
		"cache-threshold": 0.25,
		"cache-max-age": 1000,
		"reservation-mode": "all",
		"reservations-global": true,
		"reservations-in-subnet": true,
		"reservations-out-of-pool": true,
		"require-client-classes": [ "late" ],
		"store-extended-info": true,
		"server-hostname": "myhost.example.org",
        "valid-lifetime": 6001,
        "min-valid-lifetime": 4001,
        "max-valid-lifetime": 8001
	}
`
}

// Returns a JSON structure with all configurable DHCPv6 shared network parameters
// in Kea. It has been initially created from the Kea's all-keys.json file and then
// slightly modified.
func getAllKeysSharedNetwork6() string {
	return `
	{
		"allocator": "random",
		"pd-allocator": "iterative",
		"client-class": "foobar",
		"ddns-generated-prefix": "myhost",
		"ddns-override-client-update": true,
		"ddns-override-no-update": true,
		"ddns-qualifying-suffix": "example.org",
		"ddns-replace-client-name": "never",
		"ddns-send-updates": true,
		"ddns-update-on-renew": true,
		"ddns-use-conflict-resolution": true,
		"hostname-char-replacement": "x",
		"hostname-char-set": "[^A-Za-z0-9.-]",
		"interface": "eth0",
		"interface-id": "ethx",
		"name": "my-secret-network",
		"option-data": [
			{
				"always-send": true,
				"code": 7,
				"csv-format": true,
				"data": "15",
				"name": "preference",
				"space": "dhcp6"
			}
		],
		"preferred-lifetime": 2000,
		"min-preferred-lifetime": 1500,
		"max-preferred-lifetime": 2500,
		"rapid-commit": true,
		"relay": {
			"ip-addresses": [
				"2001:db8:0:f::1"
			]
		},
		"rebind-timer": 41,
		"renew-timer": 31,
		"calculate-tee-times": true,
		"t1-percent": 0.5,
		"t2-percent": 0.75,
		"cache-threshold": 0.25,
		"cache-max-age": 10,
		"reservation-mode": "all",
		"reservations-global": true,
		"reservations-in-subnet": true,
		"reservations-out-of-pool": true,
		"require-client-classes": [ "late" ],
		"store-extended-info": true,
		"valid-lifetime": 6001,
		"min-valid-lifetime": 4001,
		"max-valid-lifetime": 8001
	}
`
}

// Test that Kea shared network configuration is properly decoded into the
// keaconfig.SharedNetwork4 structure.
func TestDecodeAllKeysSharedNetwork4(t *testing.T) {
	params := keaconfig.SharedNetwork4{}
	err := json.Unmarshal([]byte(getAllKeysSharedNetwork4()), &params)
	require.NoError(t, err)

	require.Equal(t, "my-secret-network", params.Name)
	require.Equal(t, "random", *params.Allocator)
	require.Equal(t, "/dev/null", *params.BootFileName)
	require.Equal(t, "foobar", *params.ClientClass)
	require.Equal(t, "myhost", *params.DDNSGeneratedPrefix)
	require.True(t, *params.DDNSOverrideClientUpdate)
	require.True(t, *params.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *params.DDNSQualifyingSuffix)
	require.Equal(t, "never", *params.DDNSReplaceClientName)
	require.True(t, *params.DDNSSendUpdates)
	require.True(t, *params.DDNSUpdateOnRenew)
	require.True(t, *params.DDNSUseConflictResolution)
	require.Equal(t, "x", *params.HostnameCharReplacement)
	require.Equal(t, "[^A-Za-z0-9.-]", *params.HostnameCharSet)
	require.Equal(t, "eth0", *params.Interface)
	require.True(t, *params.MatchClientID)
	require.Equal(t, "192.0.2.123", *params.NextServer)
	require.True(t, *params.StoreExtendedInfo)
	require.Len(t, params.OptionData, 1)
	require.True(t, params.OptionData[0].AlwaysSend)
	require.EqualValues(t, 3, params.OptionData[0].Code)
	require.True(t, params.OptionData[0].CSVFormat)
	require.Equal(t, "192.0.3.1", params.OptionData[0].Data)
	require.Equal(t, "routers", params.OptionData[0].Name)
	require.Equal(t, "dhcp4", params.OptionData[0].Space)
	require.EqualValues(t, 41, *params.RebindTimer)
	require.Len(t, params.Relay.IPAddresses, 1)
	require.Equal(t, "192.168.56.1", params.Relay.IPAddresses[0])
	require.EqualValues(t, 31, *params.RenewTimer)
	require.True(t, *params.ReservationsInSubnet)
	require.True(t, *params.ReservationsOutOfPool)
	require.True(t, *params.CalculateTeeTimes)
	require.EqualValues(t, 0.5, *params.T1Percent)
	require.EqualValues(t, 0.75, *params.T2Percent)
	require.EqualValues(t, 0.25, *params.CacheThreshold)
	require.EqualValues(t, 1000, *params.CacheMaxAge)
	require.Len(t, params.RequireClientClasses, 1)
	require.Equal(t, "late", params.RequireClientClasses[0])
	require.Equal(t, "myhost.example.org", *params.ServerHostname)
	require.EqualValues(t, 6001, *params.ValidLifetime)
	require.EqualValues(t, 4001, *params.MinValidLifetime)
	require.EqualValues(t, 8001, *params.MaxValidLifetime)
}

// Test that Kea shared network configuration is properly decoded into the
// keaconfig.SharedNetwork6 structure.
func TestDecodeAllKeysSharedNetwork6(t *testing.T) {
	params := keaconfig.SharedNetwork6{}
	err := json.Unmarshal([]byte(getAllKeysSharedNetwork6()), &params)
	require.NoError(t, err)

	require.Equal(t, "my-secret-network", params.Name)
	require.Equal(t, "random", *params.Allocator)
	require.Equal(t, "iterative", *params.PDAllocator)
	require.Equal(t, "foobar", *params.ClientClass)
	require.Equal(t, "myhost", *params.DDNSGeneratedPrefix)
	require.True(t, *params.DDNSOverrideClientUpdate)
	require.True(t, *params.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *params.DDNSQualifyingSuffix)
	require.Equal(t, "never", *params.DDNSReplaceClientName)
	require.True(t, *params.DDNSSendUpdates)
	require.True(t, *params.DDNSUpdateOnRenew)
	require.True(t, *params.DDNSUseConflictResolution)
	require.Equal(t, "x", *params.HostnameCharReplacement)
	require.Equal(t, "[^A-Za-z0-9.-]", *params.HostnameCharSet)
	require.Equal(t, "eth0", *params.Interface)
	require.Equal(t, "ethx", *params.InterfaceID)
	require.True(t, *params.StoreExtendedInfo)
	require.Len(t, params.OptionData, 1)
	require.True(t, params.OptionData[0].AlwaysSend)
	require.EqualValues(t, 7, params.OptionData[0].Code)
	require.True(t, params.OptionData[0].CSVFormat)
	require.Equal(t, "15", params.OptionData[0].Data)
	require.Equal(t, "preference", params.OptionData[0].Name)
	require.Equal(t, "dhcp6", params.OptionData[0].Space)
	require.EqualValues(t, 2000, *params.PreferredLifetime)
	require.EqualValues(t, 1500, *params.MinPreferredLifetime)
	require.EqualValues(t, 2500, *params.MaxPreferredLifetime)
	require.True(t, *params.RapidCommit)
	require.EqualValues(t, 41, *params.RebindTimer)
	require.EqualValues(t, 31, *params.RenewTimer)
	require.True(t, *params.ReservationsInSubnet)
	require.True(t, *params.ReservationsOutOfPool)
	require.True(t, *params.CalculateTeeTimes)
	require.EqualValues(t, 0.5, *params.T1Percent)
	require.EqualValues(t, 0.75, *params.T2Percent)
	require.EqualValues(t, 0.25, *params.CacheThreshold)
	require.EqualValues(t, 10, *params.CacheMaxAge)
	require.Len(t, params.RequireClientClasses, 1)
	require.Equal(t, "late", params.RequireClientClasses[0])
	require.EqualValues(t, 6001, *params.ValidLifetime)
	require.EqualValues(t, 4001, *params.MinValidLifetime)
	require.EqualValues(t, 8001, *params.MaxValidLifetime)
}

// Test that the Kea DHCPv4 shared network configuration parameters are
// returned in the keaconfig.SharedNetworkParameters union.
func TestGetParametersSharedNetwork4(t *testing.T) {
	sharedNetwork4 := keaconfig.SharedNetwork4{}
	err := json.Unmarshal([]byte(getAllKeysSharedNetwork4()), &sharedNetwork4)
	require.NoError(t, err)

	params := sharedNetwork4.GetSharedNetworkParameters()
	require.NotNil(t, params)

	require.Equal(t, "random", *params.Allocator)
	require.Equal(t, "/dev/null", *params.BootFileName)
	require.Equal(t, "foobar", *params.ClientClass)
	require.Equal(t, "myhost", *params.DDNSGeneratedPrefix)
	require.True(t, *params.DDNSOverrideClientUpdate)
	require.True(t, *params.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *params.DDNSQualifyingSuffix)
	require.Equal(t, "never", *params.DDNSReplaceClientName)
	require.True(t, *params.DDNSSendUpdates)
	require.True(t, *params.DDNSUpdateOnRenew)
	require.True(t, *params.DDNSUseConflictResolution)
	require.Equal(t, "x", *params.HostnameCharReplacement)
	require.Equal(t, "[^A-Za-z0-9.-]", *params.HostnameCharSet)
	require.Equal(t, "eth0", *params.Interface)
	require.True(t, *params.MatchClientID)
	require.Equal(t, "192.0.2.123", *params.NextServer)
	require.True(t, *params.StoreExtendedInfo)
	require.EqualValues(t, 41, *params.RebindTimer)
	require.Len(t, params.Relay.IPAddresses, 1)
	require.Equal(t, "192.168.56.1", params.Relay.IPAddresses[0])
	require.EqualValues(t, 31, *params.RenewTimer)
	require.True(t, *params.ReservationsInSubnet)
	require.True(t, *params.ReservationsOutOfPool)
	require.True(t, *params.CalculateTeeTimes)
	require.EqualValues(t, 0.5, *params.T1Percent)
	require.EqualValues(t, 0.75, *params.T2Percent)
	require.EqualValues(t, 0.25, *params.CacheThreshold)
	require.EqualValues(t, 1000, *params.CacheMaxAge)
	require.Len(t, params.RequireClientClasses, 1)
	require.Equal(t, "late", params.RequireClientClasses[0])
	require.Equal(t, "myhost.example.org", *params.ServerHostname)
	require.EqualValues(t, 6001, *params.ValidLifetime)
	require.EqualValues(t, 4001, *params.MinValidLifetime)
	require.EqualValues(t, 8001, *params.MaxValidLifetime)
}

// Test that the Kea DHCPv6 shared network configuration parameters are
// returned in the keaconfig.SharedNetworkParameters union.
func TestGetParametersSharedNetwork6(t *testing.T) {
	sharedNetwork6 := keaconfig.SharedNetwork6{}
	err := json.Unmarshal([]byte(getAllKeysSharedNetwork6()), &sharedNetwork6)
	require.NoError(t, err)

	params := sharedNetwork6.GetSharedNetworkParameters()
	require.NotNil(t, params)

	require.Equal(t, "random", *params.Allocator)
	require.Equal(t, "iterative", *params.PDAllocator)
	require.Equal(t, "foobar", *params.ClientClass)
	require.Equal(t, "myhost", *params.DDNSGeneratedPrefix)
	require.True(t, *params.DDNSOverrideClientUpdate)
	require.True(t, *params.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *params.DDNSQualifyingSuffix)
	require.Equal(t, "never", *params.DDNSReplaceClientName)
	require.True(t, *params.DDNSSendUpdates)
	require.True(t, *params.DDNSUpdateOnRenew)
	require.True(t, *params.DDNSUseConflictResolution)
	require.Equal(t, "x", *params.HostnameCharReplacement)
	require.Equal(t, "[^A-Za-z0-9.-]", *params.HostnameCharSet)
	require.Equal(t, "eth0", *params.Interface)
	require.Equal(t, "ethx", *params.InterfaceID)
	require.True(t, *params.StoreExtendedInfo)
	require.EqualValues(t, 2000, *params.PreferredLifetime)
	require.EqualValues(t, 1500, *params.MinPreferredLifetime)
	require.EqualValues(t, 2500, *params.MaxPreferredLifetime)
	require.True(t, *params.RapidCommit)
	require.EqualValues(t, 41, *params.RebindTimer)
	require.EqualValues(t, 31, *params.RenewTimer)
	require.True(t, *params.ReservationsInSubnet)
	require.True(t, *params.ReservationsOutOfPool)
	require.True(t, *params.CalculateTeeTimes)
	require.EqualValues(t, 0.5, *params.T1Percent)
	require.EqualValues(t, 0.75, *params.T2Percent)
	require.EqualValues(t, 0.25, *params.CacheThreshold)
	require.EqualValues(t, 10, *params.CacheMaxAge)
	require.Len(t, params.RequireClientClasses, 1)
	require.Equal(t, "late", params.RequireClientClasses[0])
	require.EqualValues(t, 6001, *params.ValidLifetime)
	require.EqualValues(t, 4001, *params.MinValidLifetime)
	require.EqualValues(t, 8001, *params.MaxValidLifetime)
}

// Test converting an DHCPv4 shared network in Stork into the shared network
// configuration in Kea.
func TestCreateSharedNetwork4(t *testing.T) {
	controller := gomock.NewController(t)

	// Mock a shared network in Stork.
	mock := NewMockSharedNetworkAccessor(controller)

	// Shared network name.
	mock.EXPECT().GetName().Return("my-secret-network")
	// Return shared-network-level Kea parameters.
	mock.EXPECT().GetKeaParameters(gomock.Eq(int64(1))).Return(&keaconfig.SharedNetworkParameters{
		CacheParameters: keaconfig.CacheParameters{
			CacheMaxAge:    ptr[int64](1001),
			CacheThreshold: ptr[float32](0.25),
		},
		ClientClassParameters: keaconfig.ClientClassParameters{
			ClientClass:          ptr("myclass"),
			RequireClientClasses: []string{"foo"},
		},
		DDNSParameters: keaconfig.DDNSParameters{
			DDNSGeneratedPrefix:       ptr("example.com"),
			DDNSOverrideClientUpdate:  ptr(true),
			DDNSOverrideNoUpdate:      ptr(true),
			DDNSQualifyingSuffix:      ptr("example.org"),
			DDNSReplaceClientName:     ptr("never"),
			DDNSSendUpdates:           ptr(true),
			DDNSUseConflictResolution: ptr(true),
		},
		HostnameCharParameters: keaconfig.HostnameCharParameters{
			HostnameCharReplacement: ptr("xyz"),
			HostnameCharSet:         ptr("[A-z]"),
		},
		ReservationParameters: keaconfig.ReservationParameters{
			ReservationMode:       ptr("out-of-pool"),
			ReservationsGlobal:    ptr(true),
			ReservationsInSubnet:  ptr(true),
			ReservationsOutOfPool: ptr(true),
		},
		TimerParameters: keaconfig.TimerParameters{
			CalculateTeeTimes: ptr(true),
			RebindTimer:       ptr[int64](300),
			RenewTimer:        ptr[int64](200),
			T1Percent:         ptr[float32](0.32),
			T2Percent:         ptr[float32](0.44),
		},
		ValidLifetimeParameters: keaconfig.ValidLifetimeParameters{
			MaxValidLifetime: ptr[int64](1000),
			MinValidLifetime: ptr[int64](500),
			ValidLifetime:    ptr[int64](1001),
		},
		Allocator:     ptr("iterative"),
		Authoritative: ptr(true),
		BootFileName:  ptr("/tmp/bootfile"),
		Interface:     ptr("etx0"),
		InterfaceID:   ptr("id-foo"),
		MatchClientID: ptr(true),
		NextServer:    ptr("192.0.2.1"),
		Relay: &keaconfig.Relay{
			IPAddresses: []string{"10.0.0.1"},
		},
		ServerHostname:    ptr("hostname.example.org"),
		StoreExtendedInfo: ptr(true),
	})
	// Return subnet-level DHCP options.
	mock.EXPECT().GetDHCPOptions(gomock.Any()).Return([]dhcpmodel.DHCPOptionAccessor{
		keaconfig.DHCPOption{
			AlwaysSend:  true,
			Code:        5,
			Encapsulate: "foo",
			Fields:      []dhcpmodel.DHCPOptionFieldAccessor{},
			Space:       "dhcp4",
		},
	})

	// Mock a subnet within the shared network.
	subnetMock := NewMockSubnetAccessor(controller)
	subnetMock.EXPECT().GetID(gomock.Any()).Return(int64(5))
	subnetMock.EXPECT().GetPrefix().Return("192.0.2.0/24")
	subnetMock.EXPECT().GetAddressPools(gomock.Any()).Return([]dhcpmodel.AddressPoolAccessor{})
	subnetMock.EXPECT().GetKeaParameters(gomock.Any()).Return(&keaconfig.SubnetParameters{})
	subnetMock.EXPECT().GetDHCPOptions(gomock.Any()).Return([]dhcpmodel.DHCPOptionAccessor{})
	subnetMock.EXPECT().GetUserContext(gomock.Any()).Return(nil)

	mock.EXPECT().GetSubnets(gomock.Any()).Return([]keaconfig.SubnetAccessor{
		subnetMock,
	})

	// Do not return option definitions. This is not the area of the code
	// that we want to test here.
	lookupMock := NewMockDHCPOptionDefinitionLookup(controller)
	lookupMock.EXPECT().DefinitionExists(gomock.Any(), gomock.Any()).AnyTimes().Return(false)

	// Convert the subnet from the Stork format to the Kea format.
	network4, err := keaconfig.CreateSharedNetwork4(1, lookupMock, mock)
	require.NoError(t, err)
	require.NotNil(t, *network4)

	// Make sure that the conversion was correct.
	require.Equal(t, "my-secret-network", network4.Name)
	require.Equal(t, "iterative", *network4.Allocator)
	require.True(t, *network4.Authoritative)
	require.Equal(t, "/tmp/bootfile", *network4.BootFileName)
	require.EqualValues(t, 1001, *network4.CacheMaxAge)
	require.EqualValues(t, 0.25, *network4.CacheThreshold)
	require.True(t, *network4.CalculateTeeTimes)
	require.Equal(t, "myclass", *network4.ClientClass)
	require.Equal(t, "example.com", *network4.DDNSGeneratedPrefix)
	require.True(t, *network4.DDNSOverrideClientUpdate)
	require.True(t, *network4.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *network4.DDNSQualifyingSuffix)
	require.Equal(t, "never", *network4.DDNSReplaceClientName)
	require.True(t, *network4.DDNSSendUpdates)
	require.True(t, *network4.DDNSUseConflictResolution)
	require.Equal(t, "xyz", *network4.HostnameCharReplacement)
	require.Equal(t, "[A-z]", *network4.HostnameCharSet)
	require.Equal(t, "etx0", *network4.Interface)
	require.True(t, *network4.MatchClientID)
	require.EqualValues(t, 1000, *network4.MaxValidLifetime)
	require.EqualValues(t, 500, *network4.MinValidLifetime)
	require.Equal(t, "192.0.2.1", *network4.NextServer)
	require.Len(t, network4.OptionData, 1)
	require.EqualValues(t, 5, network4.OptionData[0].Code)
	require.Equal(t, "dhcp4", network4.OptionData[0].Space)
	require.EqualValues(t, 300, *network4.RebindTimer)
	require.Len(t, network4.Relay.IPAddresses, 1)
	require.Equal(t, "10.0.0.1", network4.Relay.IPAddresses[0])
	require.EqualValues(t, 200, *network4.RenewTimer)
	require.Len(t, network4.RequireClientClasses, 1)
	require.Equal(t, "foo", network4.RequireClientClasses[0])
	require.Equal(t, "out-of-pool", *network4.ReservationMode)
	require.True(t, *network4.ReservationsGlobal)
	require.True(t, *network4.ReservationsInSubnet)
	require.True(t, *network4.ReservationsOutOfPool)
	require.Equal(t, "hostname.example.org", *network4.ServerHostname)
	require.True(t, *network4.StoreExtendedInfo)
	require.Equal(t, float32(0.32), *network4.T1Percent)
	require.Equal(t, float32(0.44), *network4.T2Percent)
	require.EqualValues(t, 1001, *network4.ValidLifetime)

	require.Len(t, network4.Subnet4, 1)
	require.EqualValues(t, 5, network4.Subnet4[0].ID)
	require.Equal(t, "192.0.2.0/24", network4.Subnet4[0].Subnet)
}

// Test converting an DHCPv6 shared network in Stork into the shared network
// configuration in Kea.
func TestCreateSharedNetwork6(t *testing.T) {
	controller := gomock.NewController(t)

	// Mock a subnet in Stork.
	mock := NewMockSharedNetworkAccessor(controller)

	// Shared network name.
	mock.EXPECT().GetName().Return("my-secret-network")
	// Return shared-network-level Kea parameters.
	mock.EXPECT().GetKeaParameters(gomock.Eq(int64(1))).Return(&keaconfig.SharedNetworkParameters{
		CacheParameters: keaconfig.CacheParameters{
			CacheMaxAge:    ptr[int64](1001),
			CacheThreshold: ptr[float32](0.25),
		},
		ClientClassParameters: keaconfig.ClientClassParameters{
			ClientClass:          ptr("myclass"),
			RequireClientClasses: []string{"foo"},
		},
		DDNSParameters: keaconfig.DDNSParameters{
			DDNSGeneratedPrefix:       ptr("example.com"),
			DDNSOverrideClientUpdate:  ptr(true),
			DDNSOverrideNoUpdate:      ptr(true),
			DDNSQualifyingSuffix:      ptr("example.org"),
			DDNSReplaceClientName:     ptr("never"),
			DDNSSendUpdates:           ptr(true),
			DDNSUseConflictResolution: ptr(true),
		},
		HostnameCharParameters: keaconfig.HostnameCharParameters{
			HostnameCharReplacement: ptr("xyz"),
			HostnameCharSet:         ptr("[A-z]"),
		},
		PreferredLifetimeParameters: keaconfig.PreferredLifetimeParameters{
			MaxPreferredLifetime: ptr[int64](800),
			MinPreferredLifetime: ptr[int64](300),
			PreferredLifetime:    ptr[int64](801),
		},
		ReservationParameters: keaconfig.ReservationParameters{
			ReservationMode:       ptr("out-of-pool"),
			ReservationsGlobal:    ptr(true),
			ReservationsInSubnet:  ptr(true),
			ReservationsOutOfPool: ptr(true),
		},
		TimerParameters: keaconfig.TimerParameters{
			CalculateTeeTimes: ptr(true),
			RebindTimer:       ptr[int64](300),
			RenewTimer:        ptr[int64](200),
			T1Percent:         ptr[float32](0.32),
			T2Percent:         ptr[float32](0.44),
		},
		ValidLifetimeParameters: keaconfig.ValidLifetimeParameters{
			MaxValidLifetime: ptr[int64](1000),
			MinValidLifetime: ptr[int64](500),
			ValidLifetime:    ptr[int64](1001),
		},
		Allocator:   ptr("iterative"),
		PDAllocator: ptr("random"),
		Interface:   ptr("etx0"),
		InterfaceID: ptr("id-foo"),
		RapidCommit: ptr(true),
		Relay: &keaconfig.Relay{
			IPAddresses: []string{"3000::1"},
		},
		ServerHostname:    ptr("hostname.example.org"),
		StoreExtendedInfo: ptr(true),
	})
	// Return subnet-level DHCP options.
	mock.EXPECT().GetDHCPOptions(gomock.Any()).Return([]dhcpmodel.DHCPOptionAccessor{
		keaconfig.DHCPOption{
			AlwaysSend:  true,
			Code:        5,
			Encapsulate: "foo",
			Fields:      []dhcpmodel.DHCPOptionFieldAccessor{},
			Space:       "dhcp6",
		},
	})

	// Mock a subnet within the shared network.
	subnetMock := NewMockSubnetAccessor(controller)
	subnetMock.EXPECT().GetID(gomock.Any()).Return(int64(5))
	subnetMock.EXPECT().GetPrefix().Return("2001:db8:1::/64")
	subnetMock.EXPECT().GetAddressPools(gomock.Any()).Return([]dhcpmodel.AddressPoolAccessor{})
	subnetMock.EXPECT().GetPrefixPools(gomock.Any()).Return([]dhcpmodel.PrefixPoolAccessor{})
	subnetMock.EXPECT().GetKeaParameters(gomock.Any()).Return(&keaconfig.SubnetParameters{})
	subnetMock.EXPECT().GetDHCPOptions(gomock.Any()).Return([]dhcpmodel.DHCPOptionAccessor{})
	subnetMock.EXPECT().GetUserContext(gomock.Any()).Return(nil)

	mock.EXPECT().GetSubnets(gomock.Any()).Return([]keaconfig.SubnetAccessor{
		subnetMock,
	})

	// Do not return option definitions. This is not the area of the code
	// that we want to test here.
	lookupMock := NewMockDHCPOptionDefinitionLookup(controller)
	lookupMock.EXPECT().DefinitionExists(gomock.Any(), gomock.Any()).AnyTimes().Return(false)

	// Convert the subnet from the Stork format to the Kea format.
	network6, err := keaconfig.CreateSharedNetwork6(1, lookupMock, mock)
	require.NoError(t, err)
	require.NotNil(t, network6)

	// Make sure that the conversion was correct.
	require.Equal(t, "my-secret-network", network6.Name)
	require.Equal(t, "iterative", *network6.Allocator)
	require.EqualValues(t, 1001, *network6.CacheMaxAge)
	require.EqualValues(t, 0.25, *network6.CacheThreshold)
	require.True(t, *network6.CalculateTeeTimes)
	require.Equal(t, "myclass", *network6.ClientClass)
	require.Equal(t, "example.com", *network6.DDNSGeneratedPrefix)
	require.True(t, *network6.DDNSOverrideClientUpdate)
	require.True(t, *network6.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *network6.DDNSQualifyingSuffix)
	require.Equal(t, "never", *network6.DDNSReplaceClientName)
	require.True(t, *network6.DDNSSendUpdates)
	require.True(t, *network6.DDNSUseConflictResolution)
	require.Equal(t, "xyz", *network6.HostnameCharReplacement)
	require.Equal(t, "[A-z]", *network6.HostnameCharSet)
	require.Equal(t, "etx0", *network6.Interface)
	require.EqualValues(t, 1000, *network6.MaxValidLifetime)
	require.EqualValues(t, 500, *network6.MinValidLifetime)
	require.Len(t, network6.OptionData, 1)
	require.EqualValues(t, 5, network6.OptionData[0].Code)
	require.Equal(t, "dhcp6", network6.OptionData[0].Space)
	require.Len(t, network6.RequireClientClasses, 1)
	require.Equal(t, "foo", network6.RequireClientClasses[0])
	require.EqualValues(t, 300, *network6.RebindTimer)
	require.Len(t, network6.Relay.IPAddresses, 1)
	require.Equal(t, "3000::1", network6.Relay.IPAddresses[0])
	require.EqualValues(t, 200, *network6.RenewTimer)
	require.Len(t, network6.RequireClientClasses, 1)
	require.Equal(t, "foo", network6.RequireClientClasses[0])
	require.Equal(t, "out-of-pool", *network6.ReservationMode)
	require.True(t, *network6.ReservationsGlobal)
	require.True(t, *network6.ReservationsInSubnet)
	require.True(t, *network6.ReservationsOutOfPool)
	require.True(t, *network6.StoreExtendedInfo)
	require.Equal(t, float32(0.32), *network6.T1Percent)
	require.Equal(t, float32(0.44), *network6.T2Percent)
	require.EqualValues(t, 1001, *network6.ValidLifetime)

	require.Len(t, network6.Subnet6, 1)
	require.EqualValues(t, 5, network6.Subnet6[0].ID)
	require.Equal(t, "2001:db8:1::/64", network6.Subnet6[0].Subnet)
}

// Test conversion of the shared network to a structure used when deleting
// the shared network from Kea with the subnets.
func TestCreateSubnetCmdsDeletedSharedNetwork(t *testing.T) {
	controller := gomock.NewController(t)

	// Mock a shared network in Stork.
	mock := NewMockSharedNetworkAccessor(controller)

	// Shared network name.
	mock.EXPECT().GetName().Return("foo")

	sharedNetwork := keaconfig.CreateSubnetCmdsDeletedSharedNetwork(1, mock, keaconfig.SharedNetworkSubnetsActionDelete)
	require.NotNil(t, sharedNetwork)

	require.Equal(t, "foo", sharedNetwork.Name)
	require.EqualValues(t, "delete", sharedNetwork.SubnetsAction)
}

// Test conversion of the shared network to a structure used when deleting
// the shared network from Kea with preserving subnets.
func TestCreateSubnetCmdsDeletedSharedNetworkKeepSubnets(t *testing.T) {
	controller := gomock.NewController(t)

	// Mock a shared network in Stork.
	mock := NewMockSharedNetworkAccessor(controller)

	// Shared network name.
	mock.EXPECT().GetName().Return("bar")

	sharedNetwork := keaconfig.CreateSubnetCmdsDeletedSharedNetwork(1, mock, keaconfig.SharedNetworkSubnetsActionKeep)
	require.NotNil(t, sharedNetwork)

	require.Equal(t, "bar", sharedNetwork.Name)
	require.EqualValues(t, "keep", sharedNetwork.SubnetsAction)
}
