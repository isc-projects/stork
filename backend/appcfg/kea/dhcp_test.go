package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that configured hook libraries are retrieved from a DHCPv4
// server configuration.
func TestGetDHCPv4HookLibraries(t *testing.T) {
	cfg := &DHCPv4Config{
		CommonDHCPConfig: CommonDHCPConfig{
			HookLibraries: []HookLibrary{
				{
					Library: "libdhcp_lease_cmds",
				},
			},
		},
	}
	hooks := cfg.GetHookLibraries()
	require.Len(t, hooks, 1)
	require.Equal(t, "libdhcp_lease_cmds", hooks[0].Library)
}

// Test that the configured loggers are retrieved from a DHCPv4
// server configuration.
func TestGetDHCPv4Loggers(t *testing.T) {
	cfg := &DHCPv4Config{
		CommonDHCPConfig: CommonDHCPConfig{
			Loggers: []Logger{
				{
					Name:       "kea-dhcp4",
					Severity:   "DEBUG",
					DebugLevel: 99,
				},
			},
		},
	}
	libraries := cfg.GetLoggers()
	require.Len(t, libraries, 1)
	require.Equal(t, "kea-dhcp4", libraries[0].Name)
	require.Equal(t, "DEBUG", libraries[0].Severity)
	require.EqualValues(t, 99, libraries[0].DebugLevel)
}

// Test that common DHCPv4 configuration parameters are retrieved.
func TestGetDHCPv4CommonConfig(t *testing.T) {
	cfg := &DHCPv4Config{
		CommonDHCPConfig: CommonDHCPConfig{
			ClientClasses: []ClientClass{
				{
					Name: "foo",
				},
			},
		},
	}
	common := cfg.GetCommonDHCPConfig()
	require.Len(t, common.ClientClasses, 1)
	require.Equal(t, "foo", common.ClientClasses[0].Name)
}

// Test getting shared networks from a DHCPv4 configuration.
func TestGetDHCPv4SharedNetworks(t *testing.T) {
	cfg := &DHCPv4Config{
		SharedNetworks: []SharedNetwork4{
			{
				Name: "foo",
				Subnet4: []Subnet4{
					{
						MandatorySubnetParameters: MandatorySubnetParameters{
							ID:     1,
							Subnet: "192.0.2.0/24",
						},
					},
				},
			},
		},
		Subnet4: []Subnet4{
			{
				MandatorySubnetParameters: MandatorySubnetParameters{
					ID:     2,
					Subnet: "192.0.3.0/24",
				},
			},
		},
	}

	t.Run("without root subnets", func(t *testing.T) {
		sharedNetworks := cfg.GetSharedNetworks(false)
		require.Len(t, sharedNetworks, 1)
		require.Equal(t, "foo", sharedNetworks[0].GetName())
		require.Len(t, sharedNetworks[0].GetSubnets(), 1)
		require.EqualValues(t, 1, sharedNetworks[0].GetSubnets()[0].GetID())
		require.Equal(t, "192.0.2.0/24", sharedNetworks[0].GetSubnets()[0].GetPrefix())
	})

	t.Run("with root subnets", func(t *testing.T) {
		sharedNetworks := cfg.GetSharedNetworks(true)
		require.Len(t, sharedNetworks, 2)
		require.Equal(t, "foo", sharedNetworks[0].GetName())
		require.Len(t, sharedNetworks[0].GetSubnets(), 1)
		require.EqualValues(t, 1, sharedNetworks[0].GetSubnets()[0].GetID())
		require.Equal(t, "192.0.2.0/24", sharedNetworks[0].GetSubnets()[0].GetPrefix())
		require.Empty(t, sharedNetworks[1].GetName())
		require.Len(t, sharedNetworks[1].GetSubnets(), 1)
		require.EqualValues(t, 2, sharedNetworks[1].GetSubnets()[0].GetID())
		require.Equal(t, "192.0.3.0/24", sharedNetworks[1].GetSubnets()[0].GetPrefix())
	})
}

// Test getting subnets from the DHCPv4 server configuration.
func TestGetDHCPv4Subnets(t *testing.T) {
	cfg := &DHCPv4Config{
		Subnet4: []Subnet4{
			{
				MandatorySubnetParameters: MandatorySubnetParameters{
					ID:     2,
					Subnet: "192.0.2.0/24",
				},
			},
		},
	}
	subnets := cfg.GetSubnets()
	require.Len(t, subnets, 1)
	require.EqualValues(t, 2, subnets[0].GetID())
	require.EqualValues(t, "192.0.2.0/24", subnets[0].GetPrefix())
}

// Test setting DHCPv4 allocator.
func TestSetDHCPv4Allocator(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetAllocator(storkutil.Ptr("flq"))
	require.NotNil(t, cfg.Allocator)
	require.EqualValues(t, "flq", *cfg.Allocator)
	cfg.SetAllocator(nil)
	require.Nil(t, cfg.Allocator)
}

// Test setting cache threshold.
func TestSetDHCPv4CacheThreshold(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetCacheThreshold(storkutil.Ptr(float32(0.2)))
	require.NotNil(t, cfg.CacheThreshold)
	require.EqualValues(t, float32(0.2), *cfg.CacheThreshold)
	cfg.SetCacheThreshold(nil)
	require.Nil(t, cfg.CacheThreshold)
}

// Test setting boolean flag indicating if DDNS updates should be sent.
func TestSetDHCPv4DDNSSendUpdates(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetDDNSSendUpdates(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSSendUpdates)
	require.True(t, *cfg.DDNSSendUpdates)
	cfg.SetDDNSSendUpdates(nil)
	require.Nil(t, cfg.DDNSSendUpdates)
}

// Test setting boolean flag indicating whether the DHCP server should override the
// client's wish to not update the DNS.
func TestSetDHCPv4DDNSOverrideNoUpdate(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetDDNSOverrideNoUpdate(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSOverrideNoUpdate)
	require.True(t, *cfg.DDNSOverrideNoUpdate)
	cfg.SetDDNSOverrideNoUpdate(nil)
	require.Nil(t, cfg.DDNSOverrideNoUpdate)
}

// Test setting the boolean flag indicating whether the DHCP server should ignore the
// client's wish to update the DNS on its own.
func TestSetDHCPv4DDNSOverrideClientUpdate(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetDDNSOverrideClientUpdate(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSOverrideClientUpdate)
	require.True(t, *cfg.DDNSOverrideClientUpdate)
	cfg.SetDDNSOverrideClientUpdate(nil)
	require.Nil(t, cfg.DDNSOverrideClientUpdate)
}

// Test setting the enumeration specifying whether the server should honor
// the hostname or Client FQDN sent by the client or replace this name.
func TestSetDHCPv4DDNSReplaceClientName(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetDDNSReplaceClientName(storkutil.Ptr("never"))
	require.NotNil(t, cfg.DDNSReplaceClientName)
	require.Equal(t, "never", *cfg.DDNSReplaceClientName)
	cfg.SetDDNSReplaceClientName(nil)
	require.Nil(t, cfg.DDNSReplaceClientName)
}

// Test setting a prefix to be prepended to the generated Client FQDN.
func TestSetDHCPv4DDNSGeneratedPrefix(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetDDNSGeneratedPrefix(storkutil.Ptr("myhost.example.org"))
	require.NotNil(t, cfg.DDNSGeneratedPrefix)
	require.Equal(t, "myhost.example.org", *cfg.DDNSGeneratedPrefix)
	cfg.SetDDNSGeneratedPrefix(nil)
	require.Nil(t, cfg.DDNSGeneratedPrefix)
}

// Test setting a suffix appended to the partial name sent to the DNS.
func TestSetDHCPv4DDNSQualifyingSuffix(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetDDNSQualifyingSuffix(storkutil.Ptr("example.org"))
	require.NotNil(t, cfg.DDNSQualifyingSuffix)
	require.Equal(t, "example.org", *cfg.DDNSQualifyingSuffix)
	cfg.SetDDNSQualifyingSuffix(nil)
	require.Nil(t, cfg.DDNSQualifyingSuffix)
}

// Test setting a boolean flag, which when true instructs the server to always
// update DNS when leases are renewed, even if the DNS information
// has not changed.
func TestSetDHCPv4DDNSUpdateOnRenew(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetDDNSUpdateOnRenew(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSUpdateOnRenew)
	require.True(t, *cfg.DDNSUpdateOnRenew)
	cfg.SetDDNSUpdateOnRenew(nil)
	require.Nil(t, cfg.DDNSUpdateOnRenew)
}

// Test setting a boolean flag which is passed to kea-dhcp-ddns with each DDNS
// update request, to indicate whether DNS update conflict
// resolution as described in RFC 4703 should be employed for the
// given update request.
func TestSetDHCPv4DDNSUseConflictResolution(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetDDNSUseConflictResolution(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSUseConflictResolution)
	require.True(t, *cfg.DDNSUseConflictResolution)
	cfg.SetDDNSUseConflictResolution(nil)
	require.Nil(t, cfg.DDNSUseConflictResolution)
}

// Test setting the the percent of the lease's lifetime to use for the DNS TTL.
func TestSetDHCPv4DDNSTTLPercent(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetDDNSTTLPercent(storkutil.Ptr(float32(0.2)))
	require.NotNil(t, cfg.DDNSTTLPercent)
	require.EqualValues(t, float32(0.2), *cfg.DDNSTTLPercent)
	cfg.SetDDNSTTLPercent(nil)
	require.Nil(t, cfg.DDNSTTLPercent)
}

// Test setting the number of seconds since the last removal of the expired
// leases, when the next removal should occur.
func TestSetDHCPv4ELPFlushReclaimedTimerWaitTime(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetELPFlushReclaimedTimerWaitTime(storkutil.Ptr(int64(123)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.FlushReclaimedTimerWaitTime)
	require.EqualValues(t, 123, *cfg.ExpiredLeasesProcessing.FlushReclaimedTimerWaitTime)
	cfg.SetELPFlushReclaimedTimerWaitTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.FlushReclaimedTimerWaitTime)
}

// Test setting the length of time in seconds to keep expired leases in the
// lease database (lease affinity).
func TestSetDHCPv4ELPHoldReclaimedTime(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetELPHoldReclaimedTime(storkutil.Ptr(int64(123)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.HoldReclaimedTime)
	require.EqualValues(t, 123, *cfg.ExpiredLeasesProcessing.HoldReclaimedTime)
	cfg.SetELPHoldReclaimedTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.HoldReclaimedTime)
}

// Test setting the maximum number of expired leases that can be processed in
// a single attempt to clean up expired leases from the lease database.
func TestSetDHCPv4ELPMaxReclaimLeases(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetELPMaxReclaimLeases(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.MaxReclaimLeases)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.MaxReclaimLeases)
	cfg.SetELPMaxReclaimLeases(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.MaxReclaimLeases)
}

// Test setting the maximum time in milliseconds that a single attempt to clean
// up expired leases from the lease database may take.
func TestSetDHCPv4ELPMaxReclaimTime(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetELPMaxReclaimTime(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.MaxReclaimTime)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.MaxReclaimTime)
	cfg.SetELPMaxReclaimTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.MaxReclaimTime)
}

// Test setting the length of time in seconds since the last attempt to process
// expired leases before initiating the next attempt.
func TestSetDHCPv4ELPReclaimTimerWaitTime(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetELPReclaimTimerWaitTime(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.ReclaimTimerWaitTime)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.ReclaimTimerWaitTime)
	cfg.SetELPReclaimTimerWaitTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.ReclaimTimerWaitTime)
}

// Test setting the maximum number of expired lease-processing cycles which didn't
// result in full cleanup of the exired leases from the lease database,
// after which a warning message is issued.
func TestSetDHCPv4ELPUnwarnedReclaimCycles(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetELPUnwarnedReclaimCycles(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.UnwarnedReclaimCycles)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.UnwarnedReclaimCycles)
	cfg.SetELPUnwarnedReclaimCycles(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.UnwarnedReclaimCycles)
}

// Test setting the expired leases processing structure.
func TestSetDHCPv4ExpiredLeasesProcessing(t *testing.T) {
	cfg := &DHCPv4Config{}
	expiredLeasesProcessing := &ExpiredLeasesProcessing{}
	cfg.SetExpiredLeasesProcessing(expiredLeasesProcessing)
	require.Equal(t, expiredLeasesProcessing, cfg.ExpiredLeasesProcessing)
	cfg.SetExpiredLeasesProcessing(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing)
}

// Test setting a boolean flag enabling an early global reservations lookup.
func TestSetDHCPv4EarlyGlobalReservationsLookup(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetEarlyGlobalReservationsLookup(storkutil.Ptr(false))
	require.NotNil(t, cfg.EarlyGlobalReservationsLookup)
	require.False(t, *cfg.EarlyGlobalReservationsLookup)
	cfg.SetEarlyGlobalReservationsLookup(nil)
	require.Nil(t, cfg.EarlyGlobalReservationsLookup)
}

// Test setting host reservation identifiers to be used for host reservation lookup.
func TestSetDHCPv4HostReservationIdentifiers(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetHostReservationIdentifiers([]string{"hw-address", "client-id"})
	require.NotNil(t, cfg.HostReservationIdentifiers)
	require.ElementsMatch(t, cfg.HostReservationIdentifiers, []string{"hw-address", "client-id"})
	cfg.SetHostReservationIdentifiers(nil)
	require.Nil(t, cfg.HostReservationIdentifiers)
}

// Test setting the boolean flag enabling global reservations.
func TestSetDHCPv4ReservationsGlobal(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetReservationsGlobal(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsGlobal)
	require.False(t, *cfg.ReservationsGlobal)
	cfg.SetReservationsGlobal(nil)
	require.Nil(t, cfg.ReservationsGlobal)
}

// Test setting the boolean flag enabling in-subnet reservations.
func TestSetDHCPv4ReservationsInSubnet(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetReservationsInSubnet(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsInSubnet)
	require.False(t, *cfg.ReservationsInSubnet)
	cfg.SetReservationsInSubnet(nil)
	require.Nil(t, cfg.ReservationsInSubnet)
}

// Test setting the boolean flag enabling out-of-pool reservations.
func TestSetDHCPv4ReservationsOutOfPool(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetReservationsOutOfPool(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsOutOfPool)
	require.False(t, *cfg.ReservationsOutOfPool)
	cfg.SetReservationsOutOfPool(nil)
	require.Nil(t, cfg.ReservationsOutOfPool)
}

// Test setting DHCPv4 valid lifetime.
func TestSetDHCPv4ValidLifetime(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetValidLifetime(storkutil.Ptr(int64(1111)))
	require.NotNil(t, cfg.ValidLifetime)
	require.EqualValues(t, 1111, *cfg.ValidLifetime)
	cfg.SetValidLifetime(nil)
	require.Nil(t, cfg.ValidLifetime)
}

// Test setting DHCPv4 authoritative flag.
func TestSetDHCPv4Authoritative(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetAuthoritative(storkutil.Ptr(true))
	require.NotNil(t, cfg.Authoritative)
	require.True(t, *cfg.Authoritative)
	cfg.SetAuthoritative(nil)
	require.Nil(t, cfg.Authoritative)
}

// Test setting DHCPv4 echoing client ID.
func TestSetDHCPv4EchoClientID(t *testing.T) {
	cfg := &DHCPv4Config{}
	cfg.SetEchoClientID(storkutil.Ptr(true))
	require.NotNil(t, cfg.EchoClientID)
	require.True(t, *cfg.EchoClientID)
	cfg.SetEchoClientID(nil)
	require.Nil(t, cfg.EchoClientID)
}

// Test getting configured hook libraries from the DHCPv6 server configuration.
func TestGetDHCPv6HookLibraries(t *testing.T) {
	cfg := &DHCPv6Config{
		CommonDHCPConfig: CommonDHCPConfig{
			HookLibraries: []HookLibrary{
				{
					Library: "libdhcp_ha",
				},
			},
		},
	}
	hooks := cfg.GetHookLibraries()
	require.Len(t, hooks, 1)
	require.Equal(t, "libdhcp_ha", hooks[0].Library)
}

// Test getting configured loggers from the DHCPv6 server configuration.
func TestGetDHCPv6Loggers(t *testing.T) {
	cfg := &DHCPv6Config{
		CommonDHCPConfig: CommonDHCPConfig{
			Loggers: []Logger{
				{
					Name:       "kea-dhcp6",
					Severity:   "INFO",
					DebugLevel: 65,
				},
			},
		},
	}
	libraries := cfg.GetLoggers()
	require.Len(t, libraries, 1)

	require.Equal(t, "kea-dhcp6", libraries[0].Name)
	require.Equal(t, "INFO", libraries[0].Severity)
	require.EqualValues(t, 65, libraries[0].DebugLevel)
}

// Test getting common DHCPv6 configuration parameters.
func TestGetDHCPv6CommonConfig(t *testing.T) {
	cfg := &DHCPv6Config{
		CommonDHCPConfig: CommonDHCPConfig{
			ClientClasses: []ClientClass{
				{
					Name: "bar",
				},
			},
		},
	}
	common := cfg.GetCommonDHCPConfig()
	require.Len(t, common.ClientClasses, 1)
	require.Equal(t, "bar", common.ClientClasses[0].Name)
}

// Test getting shared networks from the DHCPv6 server configuration.
func TestGetDHCPv6SharedNetworks(t *testing.T) {
	cfg := &DHCPv6Config{
		SharedNetworks: []SharedNetwork6{
			{
				Name: "foo",
				Subnet6: []Subnet6{
					{
						MandatorySubnetParameters: MandatorySubnetParameters{
							ID:     1,
							Subnet: "2001:db8:1::/64",
						},
					},
				},
			},
		},
		Subnet6: []Subnet6{
			{
				MandatorySubnetParameters: MandatorySubnetParameters{
					ID:     2,
					Subnet: "2001:db8:2::/64",
				},
			},
		},
	}

	t.Run("without root subnets", func(t *testing.T) {
		sharedNetworks := cfg.GetSharedNetworks(false)
		require.Len(t, sharedNetworks, 1)
		require.Equal(t, "foo", sharedNetworks[0].GetName())
		require.Len(t, sharedNetworks[0].GetSubnets(), 1)
		require.EqualValues(t, 1, sharedNetworks[0].GetSubnets()[0].GetID())
		require.Equal(t, "2001:db8:1::/64", sharedNetworks[0].GetSubnets()[0].GetPrefix())
	})

	t.Run("with root subnets", func(t *testing.T) {
		sharedNetworks := cfg.GetSharedNetworks(true)
		require.Len(t, sharedNetworks, 2)
		require.Equal(t, "foo", sharedNetworks[0].GetName())
		require.Len(t, sharedNetworks[0].GetSubnets(), 1)
		require.EqualValues(t, 1, sharedNetworks[0].GetSubnets()[0].GetID())
		require.Equal(t, "2001:db8:1::/64", sharedNetworks[0].GetSubnets()[0].GetPrefix())
		require.Empty(t, sharedNetworks[1].GetName())
		require.Len(t, sharedNetworks[1].GetSubnets(), 1)
		require.EqualValues(t, 2, sharedNetworks[1].GetSubnets()[0].GetID())
		require.Equal(t, "2001:db8:2::/64", sharedNetworks[1].GetSubnets()[0].GetPrefix())
	})
}

// Test getting subnets from the DHCPv6 server configuration.
func TestGetDHCPv6Subnets(t *testing.T) {
	cfg := &DHCPv6Config{
		Subnet6: []Subnet6{
			{
				MandatorySubnetParameters: MandatorySubnetParameters{
					ID:     2,
					Subnet: "2001:db8:2::/64",
				},
			},
		},
	}

	subnets := cfg.GetSubnets()
	require.Len(t, subnets, 1)
	require.EqualValues(t, 2, subnets[0].GetID())
	require.EqualValues(t, "2001:db8:2::/64", subnets[0].GetPrefix())
}

// Test setting DHCPv6 allocator.
func TestSetDHCPv6Allocator(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetAllocator(storkutil.Ptr("flq"))
	require.NotNil(t, cfg.Allocator)
	require.EqualValues(t, "flq", *cfg.Allocator)
	cfg.SetAllocator(nil)
	require.Nil(t, cfg.Allocator)
}

// Test setting cache threshold.
func TestSetDHCPv6CacheThreshold(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetCacheThreshold(storkutil.Ptr(float32(0.2)))
	require.NotNil(t, cfg.CacheThreshold)
	require.EqualValues(t, float32(0.2), *cfg.CacheThreshold)
	cfg.SetCacheThreshold(nil)
	require.Nil(t, cfg.CacheThreshold)
}

// Test setting boolean flag indicating if DDNS updates should be sent.
func TestSetDHCPv6DDNSSendUpdates(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetDDNSSendUpdates(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSSendUpdates)
	require.True(t, *cfg.DDNSSendUpdates)
	cfg.SetDDNSSendUpdates(nil)
	require.Nil(t, cfg.DDNSSendUpdates)
}

// Test setting boolean flag indicating whether the DHCP server should override the
// client's wish to not update the DNS.
func TestSetDHCPv6DDNSOverrideNoUpdate(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetDDNSOverrideNoUpdate(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSOverrideNoUpdate)
	require.True(t, *cfg.DDNSOverrideNoUpdate)
	cfg.SetDDNSOverrideNoUpdate(nil)
	require.Nil(t, cfg.DDNSOverrideNoUpdate)
}

// Test setting the boolean flag indicating whether the DHCP server should ignore the
// client's wish to update the DNS on its own.
func TestSetDHCPv6DDNSOverrideClientUpdate(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetDDNSOverrideClientUpdate(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSOverrideClientUpdate)
	require.True(t, *cfg.DDNSOverrideClientUpdate)
	cfg.SetDDNSOverrideClientUpdate(nil)
	require.Nil(t, cfg.DDNSOverrideClientUpdate)
}

// Test setting the enumeration specifying whether the server should honor
// the hostname or Client FQDN sent by the client or replace this name.
func TestSetDHCPv6DDNSReplaceClientName(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetDDNSReplaceClientName(storkutil.Ptr("never"))
	require.NotNil(t, cfg.DDNSReplaceClientName)
	require.Equal(t, "never", *cfg.DDNSReplaceClientName)
	cfg.SetDDNSReplaceClientName(nil)
	require.Nil(t, cfg.DDNSReplaceClientName)
}

// Test setting a prefix to be prepended to the generated Client FQDN.
func TestSetDHCPv6DDNSGeneratedPrefix(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetDDNSGeneratedPrefix(storkutil.Ptr("never"))
	require.NotNil(t, cfg.DDNSGeneratedPrefix)
	require.Equal(t, "never", *cfg.DDNSGeneratedPrefix)
	cfg.SetDDNSGeneratedPrefix(nil)
	require.Nil(t, cfg.DDNSGeneratedPrefix)
}

// Test setting a suffix appended to the partial name sent to the DNS.
func TestSetDHCPv6DDNSQualifyingSuffix(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetDDNSQualifyingSuffix(storkutil.Ptr("example.org"))
	require.NotNil(t, cfg.DDNSQualifyingSuffix)
	require.Equal(t, "example.org", *cfg.DDNSQualifyingSuffix)
	cfg.SetDDNSQualifyingSuffix(nil)
	require.Nil(t, cfg.DDNSQualifyingSuffix)
}

// Test setting a boolean flag, which when true instructs the server to always
// update DNS when leases are renewed, even if the DNS information
// has not changed.
func TestSetDHCPv6DDNSUpdateOnRenew(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetDDNSUpdateOnRenew(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSUpdateOnRenew)
	require.True(t, *cfg.DDNSUpdateOnRenew)
	cfg.SetDDNSUpdateOnRenew(nil)
	require.Nil(t, cfg.DDNSUpdateOnRenew)
}

// Test setting a boolean flag which is passed to kea-dhcp-ddns with each DDNS
// update request, to indicate whether DNS update conflict
// resolution as described in RFC 4703 should be employed for the
// given update request.
func TestSetDHCPv6DDNSUseConflictResolution(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetDDNSUseConflictResolution(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSUseConflictResolution)
	require.True(t, *cfg.DDNSUseConflictResolution)
	cfg.SetDDNSUseConflictResolution(nil)
	require.Nil(t, cfg.DDNSUseConflictResolution)
}

// Test setting the the percent of the lease's lifetime to use for the DNS TTL.
func TestSetDHCPv6DDNSTTLPercent(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetDDNSTTLPercent(storkutil.Ptr(float32(0.2)))
	require.NotNil(t, cfg.DDNSTTLPercent)
	require.EqualValues(t, float32(0.2), *cfg.DDNSTTLPercent)
	cfg.SetDDNSTTLPercent(nil)
	require.Nil(t, cfg.DDNSTTLPercent)
}

// Test setting the number of seconds since the last removal of the expired
// leases, when the next removal should occur.
func TestSetDHCPv6ELPFlushReclaimedTimerWaitTime(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetELPFlushReclaimedTimerWaitTime(storkutil.Ptr(int64(123)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.FlushReclaimedTimerWaitTime)
	require.EqualValues(t, 123, *cfg.ExpiredLeasesProcessing.FlushReclaimedTimerWaitTime)
	cfg.SetELPFlushReclaimedTimerWaitTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.FlushReclaimedTimerWaitTime)
}

// Test setting the length of time in seconds to keep expired leases in the
// lease database (lease affinity).
func TestSetDHCPv6ELPHoldReclaimedTime(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetELPHoldReclaimedTime(storkutil.Ptr(int64(123)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.HoldReclaimedTime)
	require.EqualValues(t, 123, *cfg.ExpiredLeasesProcessing.HoldReclaimedTime)
	cfg.SetELPHoldReclaimedTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.HoldReclaimedTime)
}

// Test setting the maximum number of expired leases that can be processed in
// a single attempt to clean up expired leases from the lease database.
func TestSetDHCPv6ELPMaxReclaimLeases(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetELPMaxReclaimLeases(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.MaxReclaimLeases)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.MaxReclaimLeases)
	cfg.SetELPMaxReclaimLeases(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.MaxReclaimLeases)
}

// Test setting the maximum time in milliseconds that a single attempt to clean
// up expired leases from the lease database may take.
func TestSetDHCPv6ELPMaxReclaimTime(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetELPMaxReclaimTime(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.MaxReclaimTime)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.MaxReclaimTime)
	cfg.SetELPMaxReclaimTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.MaxReclaimTime)
}

// Test setting the length of time in seconds since the last attempt to process
// expired leases before initiating the next attempt.
func TestSetDHCPv6ELPReclaimTimerWaitTime(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetELPReclaimTimerWaitTime(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.ReclaimTimerWaitTime)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.ReclaimTimerWaitTime)
	cfg.SetELPReclaimTimerWaitTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.ReclaimTimerWaitTime)
}

// Test setting the maximum number of expired lease-processing cycles which didn't
// result in full cleanup of the exired leases from the lease database,
// after which a warning message is issued.
func TestSetDHCPv6ELPUnwarnedReclaimCycles(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetELPUnwarnedReclaimCycles(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.UnwarnedReclaimCycles)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.UnwarnedReclaimCycles)
	cfg.SetELPUnwarnedReclaimCycles(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.UnwarnedReclaimCycles)
}

// Test setting the expired leases processing structure.
func TestSetDHCPv6ExpiredLeasesProcessing(t *testing.T) {
	cfg := &DHCPv6Config{}
	expiredLeasesProcessing := &ExpiredLeasesProcessing{}
	cfg.SetExpiredLeasesProcessing(expiredLeasesProcessing)
	require.Equal(t, expiredLeasesProcessing, cfg.ExpiredLeasesProcessing)
	cfg.SetExpiredLeasesProcessing(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing)
}

// Test setting a boolean flag enabling an early global reservations lookup.
func TestSetDHCPv6EarlyGlobalReservationsLookup(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetEarlyGlobalReservationsLookup(storkutil.Ptr(false))
	require.NotNil(t, cfg.EarlyGlobalReservationsLookup)
	require.False(t, *cfg.EarlyGlobalReservationsLookup)
	cfg.SetEarlyGlobalReservationsLookup(nil)
	require.Nil(t, cfg.EarlyGlobalReservationsLookup)
}

// Test setting host reservation identifiers to be used for host reservation lookup.
func TestSetDHCPv6HostReservationIdentifiers(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetHostReservationIdentifiers([]string{"hw-address", "client-id"})
	require.NotNil(t, cfg.HostReservationIdentifiers)
	require.ElementsMatch(t, cfg.HostReservationIdentifiers, []string{"hw-address", "client-id"})
	cfg.SetHostReservationIdentifiers(nil)
	require.Nil(t, cfg.HostReservationIdentifiers)
}

// Test setting the boolean flag enabling global reservations.
func TestSetDHCPv6ReservationsGlobal(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetReservationsGlobal(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsGlobal)
	require.False(t, *cfg.ReservationsGlobal)
	cfg.SetReservationsGlobal(nil)
	require.Nil(t, cfg.ReservationsGlobal)
}

// Test setting the boolean flag enabling in-subnet reservations.
func TestSetDHCPv6ReservationsInSubnet(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetReservationsInSubnet(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsInSubnet)
	require.False(t, *cfg.ReservationsInSubnet)
	cfg.SetReservationsInSubnet(nil)
	require.Nil(t, cfg.ReservationsInSubnet)
}

// Test setting the boolean flag enabling out-of-pool reservations.
func TestSetDHCPv6ReservationsOutOfPool(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetReservationsOutOfPool(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsOutOfPool)
	require.False(t, *cfg.ReservationsOutOfPool)
	cfg.SetReservationsOutOfPool(nil)
	require.Nil(t, cfg.ReservationsOutOfPool)
}

// Test setting DHCPv6 valid lifetime.
func TestSetDHCPv6ValidLifetime(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetValidLifetime(storkutil.Ptr(int64(1111)))
	require.NotNil(t, cfg.ValidLifetime)
	require.EqualValues(t, 1111, *cfg.ValidLifetime)
}

// Test setting allocator for prefix delegation.
func TestSetDHCPv6PDAllocator(t *testing.T) {
	cfg := &DHCPv6Config{}
	cfg.SetPDAllocator(storkutil.Ptr("flq"))
	require.NotNil(t, cfg.PDAllocator)
	require.EqualValues(t, "flq", *cfg.PDAllocator)
	cfg.SetPDAllocator(nil)
	require.Nil(t, cfg.PDAllocator)
}
