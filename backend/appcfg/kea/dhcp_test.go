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
	cfg := &SettableDHCPv4Config{}
	cfg.SetAllocator(storkutil.Ptr("flq"))
	require.NotNil(t, cfg.Allocator)
	require.EqualValues(t, "flq", *cfg.Allocator.GetValue())
	cfg.SetAllocator(nil)
	require.Nil(t, cfg.Allocator.GetValue())
}

// Test setting cache threshold.
func TestSetDHCPv4CacheThreshold(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetCacheThreshold(storkutil.Ptr(float32(0.2)))
	require.NotNil(t, cfg.CacheThreshold)
	require.NotNil(t, cfg.CacheThreshold.GetValue())
	require.EqualValues(t, float32(0.2), *cfg.CacheThreshold.GetValue())
	cfg.SetCacheThreshold(nil)
	require.Nil(t, cfg.CacheThreshold.GetValue())
}

// Test setting boolean flag indicating if DDNS updates should be sent.
func TestSetDHCPv4DDNSSendUpdates(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSSendUpdates(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSSendUpdates)
	require.NotNil(t, cfg.DDNSSendUpdates.GetValue())
	require.True(t, *cfg.DDNSSendUpdates.GetValue())
	cfg.SetDDNSSendUpdates(nil)
	require.Nil(t, cfg.DDNSSendUpdates.GetValue())
}

// Test setting boolean flag indicating whether the DHCP server should override the
// client's wish to not update the DNS.
func TestSetDHCPv4DDNSOverrideNoUpdate(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSOverrideNoUpdate(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSOverrideNoUpdate)
	require.NotNil(t, cfg.DDNSOverrideNoUpdate.GetValue())
	require.True(t, *cfg.DDNSOverrideNoUpdate.GetValue())
	cfg.SetDDNSOverrideNoUpdate(nil)
	require.Nil(t, cfg.DDNSOverrideNoUpdate.GetValue())
}

// Test setting the boolean flag indicating whether the DHCP server should ignore the
// client's wish to update the DNS on its own.
func TestSetDHCPv4DDNSOverrideClientUpdate(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSOverrideClientUpdate(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSOverrideClientUpdate)
	require.NotNil(t, cfg.DDNSOverrideClientUpdate.GetValue())
	require.True(t, *cfg.DDNSOverrideClientUpdate.GetValue())
	cfg.SetDDNSOverrideClientUpdate(nil)
	require.Nil(t, cfg.DDNSOverrideClientUpdate.GetValue())
}

// Test setting the enumeration specifying whether the server should honor
// the hostname or Client FQDN sent by the client or replace this name.
func TestSetDHCPv4DDNSReplaceClientName(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSReplaceClientName(storkutil.Ptr("never"))
	require.NotNil(t, cfg.DDNSReplaceClientName)
	require.NotNil(t, cfg.DDNSReplaceClientName.GetValue())
	require.Equal(t, "never", *cfg.DDNSReplaceClientName.GetValue())
	cfg.SetDDNSReplaceClientName(nil)
	require.Nil(t, cfg.DDNSReplaceClientName.GetValue())
}

// Test setting a prefix to be prepended to the generated Client FQDN.
func TestSetDHCPv4DDNSGeneratedPrefix(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSGeneratedPrefix(storkutil.Ptr("myhost.example.org"))
	require.NotNil(t, cfg.DDNSGeneratedPrefix)
	require.NotNil(t, cfg.DDNSGeneratedPrefix.GetValue())
	require.Equal(t, "myhost.example.org", *cfg.DDNSGeneratedPrefix.GetValue())
	cfg.SetDDNSGeneratedPrefix(nil)
	require.Nil(t, cfg.DDNSGeneratedPrefix.GetValue())
}

// Test setting a suffix appended to the partial name sent to the DNS.
func TestSetDHCPv4DDNSQualifyingSuffix(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSQualifyingSuffix(storkutil.Ptr("example.org"))
	require.NotNil(t, cfg.DDNSQualifyingSuffix)
	require.NotNil(t, cfg.DDNSQualifyingSuffix.GetValue())
	require.Equal(t, "example.org", *cfg.DDNSQualifyingSuffix.GetValue())
	cfg.SetDDNSQualifyingSuffix(nil)
	require.Nil(t, cfg.DDNSQualifyingSuffix.GetValue())
}

// Test setting a boolean flag, which when true instructs the server to always
// update DNS when leases are renewed, even if the DNS information
// has not changed.
func TestSetDHCPv4DDNSUpdateOnRenew(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSUpdateOnRenew(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSUpdateOnRenew)
	require.NotNil(t, cfg.DDNSUpdateOnRenew.GetValue())
	require.True(t, *cfg.DDNSUpdateOnRenew.GetValue())
	cfg.SetDDNSUpdateOnRenew(nil)
	require.Nil(t, cfg.DDNSUpdateOnRenew.GetValue())
}

// Test setting a boolean flag which is passed to kea-dhcp-ddns with each DDNS
// update request, to indicate whether DNS update conflict
// resolution as described in RFC 4703 should be employed for the
// given update request.
func TestSetDHCPv4DDNSUseConflictResolution(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSUseConflictResolution(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSUseConflictResolution)
	require.NotNil(t, cfg.DDNSUseConflictResolution.GetValue())
	require.True(t, *cfg.DDNSUseConflictResolution.GetValue())
	cfg.SetDDNSUseConflictResolution(nil)
	require.Nil(t, cfg.DDNSUseConflictResolution.GetValue())
}

// Test setting a DDNS conflict resolution mode.
func TestSetDHCPv4DDNSConflictResolutionMode(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSConflictResolutionMode(storkutil.Ptr("check-with-dhcid"))
	require.NotNil(t, cfg.DDNSConflictResolutionMode)
	require.NotNil(t, cfg.DDNSConflictResolutionMode.GetValue())
	require.Equal(t, "check-with-dhcid", *cfg.DDNSConflictResolutionMode.GetValue())
	cfg.SetDDNSConflictResolutionMode(nil)
	require.Nil(t, cfg.DDNSConflictResolutionMode.GetValue())
}

// Test setting the the percent of the lease's lifetime to use for the DNS TTL.
func TestSetDHCPv4DDNSTTLPercent(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDDNSTTLPercent(storkutil.Ptr(float32(0.2)))
	require.NotNil(t, cfg.DDNSTTLPercent)
	require.NotNil(t, cfg.DDNSTTLPercent.GetValue())
	require.EqualValues(t, float32(0.2), *cfg.DDNSTTLPercent.GetValue())
	cfg.SetDDNSTTLPercent(nil)
	require.Nil(t, cfg.DDNSTTLPercent.GetValue())
}

// Test enabling connectivity with the DHCP DDNS daemon.
func TestSetDHCPv4DDNSEnableUpdates(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDHCPDDNSEnableUpdates(storkutil.Ptr(true))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().EnableUpdates)
	require.NotNil(t, cfg.DHCPDDNS.GetValue().EnableUpdates.GetValue())
	require.True(t, *cfg.DHCPDDNS.GetValue().EnableUpdates.GetValue())
	cfg.SetDHCPDDNSEnableUpdates(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().EnableUpdates.GetValue())
}

// Test setting the IP address on which D2 listens for requests.
func TestSetDHCPv4DDNSServerIP(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDHCPDDNSServerIP(storkutil.Ptr("192.0.2.1"))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().ServerIP)
	require.NotNil(t, cfg.DHCPDDNS.GetValue().ServerIP.GetValue())
	require.Equal(t, "192.0.2.1", *cfg.DHCPDDNS.GetValue().ServerIP.GetValue())
	cfg.SetDHCPDDNSServerIP(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().ServerIP.GetValue())
}

// Test setting the port on which D2 listens for requests.
func TestSetDHCPv4DDNSServerPort(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDHCPDDNSServerPort(storkutil.Ptr(int64(8080)))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().ServerPort)
	require.EqualValues(t, 8080, *cfg.DHCPDDNS.GetValue().ServerPort.GetValue())
	cfg.SetDHCPDDNSServerPort(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().ServerPort.GetValue())
}

// Test setting the IP address which DHCP server uses to send requests to D2.
func TestSetDHCPv4DDNSSenderIP(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDHCPDDNSSenderIP(storkutil.Ptr("192.0.2.1"))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().SenderIP)
	require.Equal(t, "192.0.2.1", *cfg.DHCPDDNS.GetValue().SenderIP.GetValue())
	cfg.SetDHCPDDNSSenderIP(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().SenderIP.GetValue())
}

// Test setting the port which DHCP server uses to send requests to D2.
func TestSetDHCPv4DDNSSenderPort(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDHCPDDNSSenderPort(storkutil.Ptr(int64(8080)))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().SenderPort)
	require.EqualValues(t, 8080, *cfg.DHCPDDNS.GetValue().SenderPort.GetValue())
	cfg.SetDHCPDDNSSenderPort(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().SenderPort.GetValue())
}

// Test setting the maximum number of requests allowed to queue while waiting
// to be sent to D2.
func TestSetDHCPv4DDNSMaxQueueSize(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDHCPDDNSMaxQueueSize(storkutil.Ptr(int64(8080)))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().MaxQueueSize)
	require.EqualValues(t, 8080, *cfg.DHCPDDNS.GetValue().MaxQueueSize.GetValue())
	cfg.SetDHCPDDNSMaxQueueSize(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().MaxQueueSize.GetValue())
}

// Test setting the socket protocol to use when sending requests to D2.
func TestSetDHCPv4DDNSNCRProtocol(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDHCPDDNSNCRProtocol(storkutil.Ptr("UDP"))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().NCRProtocol)
	require.Equal(t, "UDP", *cfg.DHCPDDNS.GetValue().NCRProtocol.GetValue())
	cfg.SetDHCPDDNSNCRProtocol(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().NCRProtocol.GetValue())
}

// Test setting the packet format to use when sending requests to D2.
func TestSetDHCPv4DDNSNCRFormat(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDHCPDDNSNCRFormat(storkutil.Ptr("JSON"))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().NCRFormat)
	require.Equal(t, "JSON", *cfg.DHCPDDNS.GetValue().NCRFormat.GetValue())
	cfg.SetDHCPDDNSNCRFormat(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().NCRFormat.GetValue())
}

// Test setting the DHCP DDNS configuration.
func TestSetDHCPv4DDNS(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	dhcpDDNS := &SettableDHCPDDNS{}
	cfg.SetDHCPDDNS(dhcpDDNS)
	require.NotNil(t, cfg.DHCPDDNS)
	require.Equal(t, dhcpDDNS, cfg.DHCPDDNS.GetValue())
	cfg.SetDHCPDDNS(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue())
}

// Test setting the number of seconds since the last removal of the expired
// leases, when the next removal should occur.
func TestSetDHCPv4ELPFlushReclaimedTimerWaitTime(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetELPFlushReclaimedTimerWaitTime(storkutil.Ptr(int64(123)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().FlushReclaimedTimerWaitTime)
	require.EqualValues(t, 123, *cfg.ExpiredLeasesProcessing.GetValue().FlushReclaimedTimerWaitTime.GetValue())
	cfg.SetELPFlushReclaimedTimerWaitTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().FlushReclaimedTimerWaitTime.GetValue())
}

// Test setting the length of time in seconds to keep expired leases in the
// lease database (lease affinity).
func TestSetDHCPv4ELPHoldReclaimedTime(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetELPHoldReclaimedTime(storkutil.Ptr(int64(123)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().HoldReclaimedTime)
	require.EqualValues(t, 123, *cfg.ExpiredLeasesProcessing.GetValue().HoldReclaimedTime.GetValue())
	cfg.SetELPHoldReclaimedTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().HoldReclaimedTime.GetValue())
}

// Test setting the maximum number of expired leases that can be processed in
// a single attempt to clean up expired leases from the lease database.
func TestSetDHCPv4ELPMaxReclaimLeases(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetELPMaxReclaimLeases(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimLeases.GetValue())
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimLeases.GetValue())
	cfg.SetELPMaxReclaimLeases(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimLeases.GetValue())
}

// Test setting the maximum time in milliseconds that a single attempt to clean
// up expired leases from the lease database may take.
func TestSetDHCPv4ELPMaxReclaimTime(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetELPMaxReclaimTime(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimTime)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimTime.GetValue())
	cfg.SetELPMaxReclaimTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimTime.GetValue())
}

// Test setting the length of time in seconds since the last attempt to process
// expired leases before initiating the next attempt.
func TestSetDHCPv4ELPReclaimTimerWaitTime(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetELPReclaimTimerWaitTime(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().ReclaimTimerWaitTime)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.GetValue().ReclaimTimerWaitTime.GetValue())
	cfg.SetELPReclaimTimerWaitTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().ReclaimTimerWaitTime.GetValue())
}

// Test setting the maximum number of expired lease-processing cycles which didn't
// result in full cleanup of the expired leases from the lease database,
// after which a warning message is issued.
func TestSetDHCPv4ELPUnwarnedReclaimCycles(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetELPUnwarnedReclaimCycles(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().UnwarnedReclaimCycles)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.GetValue().UnwarnedReclaimCycles.GetValue())
	cfg.SetELPUnwarnedReclaimCycles(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().UnwarnedReclaimCycles.GetValue())
}

// Test setting the expired leases processing structure.
func TestSetDHCPv4ExpiredLeasesProcessing(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	expiredLeasesProcessing := &SettableExpiredLeasesProcessing{}
	cfg.SetExpiredLeasesProcessing(expiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.Equal(t, expiredLeasesProcessing, cfg.ExpiredLeasesProcessing.GetValue())
	cfg.SetExpiredLeasesProcessing(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue())
}

// Test setting a boolean flag enabling an early global reservations lookup.
func TestSetDHCPv4EarlyGlobalReservationsLookup(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetEarlyGlobalReservationsLookup(storkutil.Ptr(false))
	require.NotNil(t, cfg.EarlyGlobalReservationsLookup)
	require.NotNil(t, cfg.EarlyGlobalReservationsLookup.GetValue())
	require.False(t, *cfg.EarlyGlobalReservationsLookup.GetValue())
	cfg.SetEarlyGlobalReservationsLookup(nil)
	require.Nil(t, cfg.EarlyGlobalReservationsLookup.GetValue())
}

// Test setting host reservation identifiers to be used for host reservation lookup.
func TestSetDHCPv4HostReservationIdentifiers(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetHostReservationIdentifiers([]string{"hw-address", "client-id"})
	require.NotNil(t, cfg.HostReservationIdentifiers)
	require.NotNil(t, cfg.HostReservationIdentifiers.GetValue())
	require.ElementsMatch(t, cfg.HostReservationIdentifiers.GetValue(), []string{"hw-address", "client-id"})
	cfg.SetHostReservationIdentifiers(nil)
	require.Nil(t, cfg.HostReservationIdentifiers.GetValue())
}

// Test setting the boolean flag enabling global reservations.
func TestSetDHCPv4ReservationsGlobal(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetReservationsGlobal(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsGlobal)
	require.NotNil(t, cfg.ReservationsGlobal.GetValue())
	require.False(t, *cfg.ReservationsGlobal.GetValue())
	cfg.SetReservationsGlobal(nil)
	require.Nil(t, cfg.ReservationsGlobal.GetValue())
}

// Test setting the boolean flag enabling in-subnet reservations.
func TestSetDHCPv4ReservationsInSubnet(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetReservationsInSubnet(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsInSubnet)
	require.NotNil(t, cfg.ReservationsInSubnet.GetValue())
	require.False(t, *cfg.ReservationsInSubnet.GetValue())
	cfg.SetReservationsInSubnet(nil)
	require.Nil(t, cfg.ReservationsInSubnet.GetValue())
}

// Test setting the boolean flag enabling out-of-pool reservations.
func TestSetDHCPv4ReservationsOutOfPool(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetReservationsOutOfPool(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsOutOfPool)
	require.NotNil(t, cfg.ReservationsOutOfPool.GetValue())
	require.False(t, *cfg.ReservationsOutOfPool.GetValue())
	cfg.SetReservationsOutOfPool(nil)
	require.Nil(t, cfg.ReservationsOutOfPool.GetValue())
}

// Test setting DHCPv4 valid lifetime.
func TestSetDHCPv4ValidLifetime(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetValidLifetime(storkutil.Ptr(int64(1111)))
	require.NotNil(t, cfg.ValidLifetime)
	require.NotNil(t, cfg.ValidLifetime.GetValue())
	require.EqualValues(t, 1111, *cfg.ValidLifetime.GetValue())
	cfg.SetValidLifetime(nil)
	require.Nil(t, cfg.ValidLifetime.GetValue())
}

// Test setting DHCPv4 authoritative flag.
func TestSetDHCPv4Authoritative(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetAuthoritative(storkutil.Ptr(true))
	require.NotNil(t, cfg.Authoritative)
	require.NotNil(t, cfg.Authoritative.GetValue())
	require.True(t, *cfg.Authoritative.GetValue())
	cfg.SetAuthoritative(nil)
	require.Nil(t, cfg.Authoritative.GetValue())
}

// Test setting DHCPv4 echoing client ID.
func TestSetDHCPv4EchoClientID(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetEchoClientID(storkutil.Ptr(true))
	require.NotNil(t, cfg.EchoClientID)
	require.NotNil(t, cfg.EchoClientID.GetValue())
	require.True(t, *cfg.EchoClientID.GetValue())
	cfg.SetEchoClientID(nil)
	require.Nil(t, cfg.EchoClientID.GetValue())
}

// Test setting DHCPv4 option data.
func TestSetDHCPv4OptionData(t *testing.T) {
	cfg := &SettableDHCPv4Config{}
	cfg.SetDHCPOptions([]SingleOptionData{
		{
			Name:       "routers",
			AlwaysSend: true,
			Code:       3,
			CSVFormat:  true,
			Data:       "foobar",
			Space:      "dhcp4",
		},
	})
	options := cfg.OptionData.GetValue()
	require.Len(t, options, 1)
	require.Equal(t, "routers", options[0].Name)
	require.True(t, options[0].AlwaysSend)
	require.EqualValues(t, 3, options[0].Code)
	require.True(t, options[0].CSVFormat)
	require.Equal(t, "foobar", options[0].Data)
	require.Equal(t, "dhcp4", options[0].Space)

	cfg.SetDHCPOptions(nil)
	require.Nil(t, cfg.OptionData.GetValue())
}

// Test setting DHCPv4 option data.
func TestSetDHCPv4Options(t *testing.T) {
	// Arrange
	cfg := &SettableDHCPv4Config{}

	// Act
	cfg.SetDHCPOptions([]SingleOptionData{
		{
			Name:       "routers",
			AlwaysSend: true,
			Code:       3,
			CSVFormat:  true,
			Data:       "foobar",
			Space:      "dhcp4",
		},
	})

	// Assert
	options := cfg.OptionData.GetValue()
	require.Len(t, options, 1)
	option := options[0]
	require.Equal(t, "routers", option.Name)
	require.True(t, option.AlwaysSend)
	require.EqualValues(t, 3, option.Code)
	require.True(t, option.CSVFormat)
	require.Equal(t, "foobar", option.Data)
	require.Equal(t, "dhcp4", option.Space)
}

// Test setting nil as DHCPv4 option data.
func TestSetDHCPv4NilOptions(t *testing.T) {
	// Arrange
	cfg := &SettableDHCPv4Config{}

	// Act
	cfg.SetDHCPOptions(nil)

	// Assert
	require.Nil(t, cfg.OptionData.GetValue())
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
	cfg := &SettableDHCPv6Config{}
	cfg.SetAllocator(storkutil.Ptr("flq"))
	require.NotNil(t, cfg.Allocator)
	require.EqualValues(t, "flq", *cfg.Allocator.GetValue())
	cfg.SetAllocator(nil)
	require.Nil(t, cfg.Allocator.GetValue())
}

// Test setting cache threshold.
func TestSetDHCPv6CacheThreshold(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetCacheThreshold(storkutil.Ptr(float32(0.2)))
	require.NotNil(t, cfg.CacheThreshold)
	require.NotNil(t, cfg.CacheThreshold.GetValue())
	require.EqualValues(t, float32(0.2), *cfg.CacheThreshold.GetValue())
	cfg.SetCacheThreshold(nil)
	require.Nil(t, cfg.CacheThreshold.GetValue())
}

// Test setting boolean flag indicating if DDNS updates should be sent.
func TestSetDHCPv6DDNSSendUpdates(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSSendUpdates(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSSendUpdates)
	require.NotNil(t, cfg.DDNSSendUpdates.GetValue())
	require.True(t, *cfg.DDNSSendUpdates.GetValue())
	cfg.SetDDNSSendUpdates(nil)
	require.Nil(t, cfg.DDNSSendUpdates.GetValue())
}

// Test setting boolean flag indicating whether the DHCP server should override the
// client's wish to not update the DNS.
func TestSetDHCPv6DDNSOverrideNoUpdate(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSOverrideNoUpdate(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSOverrideNoUpdate)
	require.NotNil(t, cfg.DDNSOverrideNoUpdate.GetValue())
	require.True(t, *cfg.DDNSOverrideNoUpdate.GetValue())
	cfg.SetDDNSOverrideNoUpdate(nil)
	require.Nil(t, cfg.DDNSOverrideNoUpdate.GetValue())
}

// Test setting the boolean flag indicating whether the DHCP server should ignore the
// client's wish to update the DNS on its own.
func TestSetDHCPv6DDNSOverrideClientUpdate(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSOverrideClientUpdate(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSOverrideClientUpdate)
	require.NotNil(t, cfg.DDNSOverrideClientUpdate.GetValue())
	require.True(t, *cfg.DDNSOverrideClientUpdate.GetValue())
	cfg.SetDDNSOverrideClientUpdate(nil)
	require.Nil(t, cfg.DDNSOverrideClientUpdate.GetValue())
}

// Test setting the enumeration specifying whether the server should honor
// the hostname or Client FQDN sent by the client or replace this name.
func TestSetDHCPv6DDNSReplaceClientName(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSReplaceClientName(storkutil.Ptr("never"))
	require.NotNil(t, cfg.DDNSReplaceClientName)
	require.NotNil(t, cfg.DDNSReplaceClientName.GetValue())
	require.Equal(t, "never", *cfg.DDNSReplaceClientName.GetValue())
	cfg.SetDDNSReplaceClientName(nil)
	require.Nil(t, cfg.DDNSReplaceClientName.GetValue())
}

// Test setting a prefix to be prepended to the generated Client FQDN.
func TestSetDHCPv6DDNSGeneratedPrefix(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSGeneratedPrefix(storkutil.Ptr("myhost.example.org"))
	require.NotNil(t, cfg.DDNSGeneratedPrefix)
	require.NotNil(t, cfg.DDNSGeneratedPrefix.GetValue())
	require.Equal(t, "myhost.example.org", *cfg.DDNSGeneratedPrefix.GetValue())
	cfg.SetDDNSGeneratedPrefix(nil)
	require.Nil(t, cfg.DDNSGeneratedPrefix.GetValue())
}

// Test setting a suffix appended to the partial name sent to the DNS.
func TestSetDHCPv6DDNSQualifyingSuffix(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSQualifyingSuffix(storkutil.Ptr("example.org"))
	require.NotNil(t, cfg.DDNSQualifyingSuffix)
	require.NotNil(t, cfg.DDNSQualifyingSuffix.GetValue())
	require.Equal(t, "example.org", *cfg.DDNSQualifyingSuffix.GetValue())
	cfg.SetDDNSQualifyingSuffix(nil)
	require.Nil(t, cfg.DDNSQualifyingSuffix.GetValue())
}

// Test setting a boolean flag, which when true instructs the server to always
// update DNS when leases are renewed, even if the DNS information
// has not changed.
func TestSetDHCPv6DDNSUpdateOnRenew(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSUpdateOnRenew(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSUpdateOnRenew)
	require.NotNil(t, cfg.DDNSUpdateOnRenew.GetValue())
	require.True(t, *cfg.DDNSUpdateOnRenew.GetValue())
	cfg.SetDDNSUpdateOnRenew(nil)
	require.Nil(t, cfg.DDNSUpdateOnRenew.GetValue())
}

// Test setting a boolean flag which is passed to kea-dhcp-ddns with each DDNS
// update request, to indicate whether DNS update conflict
// resolution as described in RFC 4703 should be employed for the
// given update request.
func TestSetDHCPv6DDNSUseConflictResolution(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSUseConflictResolution(storkutil.Ptr(true))
	require.NotNil(t, cfg.DDNSUseConflictResolution)
	require.NotNil(t, cfg.DDNSUseConflictResolution.GetValue())
	require.True(t, *cfg.DDNSUseConflictResolution.GetValue())
	cfg.SetDDNSUseConflictResolution(nil)
	require.Nil(t, cfg.DDNSUseConflictResolution.GetValue())
}

// Test setting a DDNS conflict resolution mode.
func TestSetDHCPv6DDNSConflictResolutionMode(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSConflictResolutionMode(storkutil.Ptr("check-with-dhcid"))
	require.NotNil(t, cfg.DDNSConflictResolutionMode)
	require.NotNil(t, cfg.DDNSConflictResolutionMode.GetValue())
	require.Equal(t, "check-with-dhcid", *cfg.DDNSConflictResolutionMode.GetValue())
	cfg.SetDDNSConflictResolutionMode(nil)
	require.Nil(t, cfg.DDNSConflictResolutionMode.GetValue())
}

// Test setting the the percent of the lease's lifetime to use for the DNS TTL.
func TestSetDHCPv6DDNSTTLPercent(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDDNSTTLPercent(storkutil.Ptr(float32(0.2)))
	require.NotNil(t, cfg.DDNSTTLPercent)
	require.NotNil(t, cfg.DDNSTTLPercent.GetValue())
	require.EqualValues(t, float32(0.2), *cfg.DDNSTTLPercent.GetValue())
	cfg.SetDDNSTTLPercent(nil)
	require.Nil(t, cfg.DDNSTTLPercent.GetValue())
}

// Test enabling connectivity with the DHCP DDNS daemon.
func TestSetDHCPv6DDNSEnableUpdates(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDHCPDDNSEnableUpdates(storkutil.Ptr(true))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().EnableUpdates)
	require.NotNil(t, cfg.DHCPDDNS.GetValue().EnableUpdates.GetValue())
	require.True(t, *cfg.DHCPDDNS.GetValue().EnableUpdates.GetValue())
	cfg.SetDHCPDDNSEnableUpdates(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().EnableUpdates.GetValue())
}

// Test setting the IP address on which D2 listens for requests.
func TestSetDHCPv6DDNSServerIP(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDHCPDDNSServerIP(storkutil.Ptr("192.0.2.1"))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().ServerIP)
	require.NotNil(t, cfg.DHCPDDNS.GetValue().ServerIP.GetValue())
	require.Equal(t, "192.0.2.1", *cfg.DHCPDDNS.GetValue().ServerIP.GetValue())
	cfg.SetDHCPDDNSServerIP(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().ServerIP.GetValue())
}

// Test setting the port on which D2 listens for requests.
func TestSetDHCPv6DDNSServerPort(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDHCPDDNSServerPort(storkutil.Ptr(int64(8080)))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().ServerPort)
	require.EqualValues(t, 8080, *cfg.DHCPDDNS.GetValue().ServerPort.GetValue())
	cfg.SetDHCPDDNSServerPort(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().ServerPort.GetValue())
}

// Test setting the IP address which DHCP server uses to send requests to D2.
func TestSetDHCPv6DDNSSenderIP(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDHCPDDNSSenderIP(storkutil.Ptr("2001:db8:1::1"))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().SenderIP)
	require.Equal(t, "2001:db8:1::1", *cfg.DHCPDDNS.GetValue().SenderIP.GetValue())
	cfg.SetDHCPDDNSSenderIP(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().SenderIP.GetValue())
}

// Test setting the port which DHCP server uses to send requests to D2.
func TestSetDHCPv6DDNSSenderPort(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDHCPDDNSSenderPort(storkutil.Ptr(int64(8080)))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().SenderPort)
	require.EqualValues(t, 8080, *cfg.DHCPDDNS.GetValue().SenderPort.GetValue())
	cfg.SetDHCPDDNSSenderPort(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().SenderPort.GetValue())
}

// Test setting the maximum number of requests allowed to queue while waiting
// to be sent to D2.
func TestSetDHCPv6DDNSMaxQueueSize(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDHCPDDNSMaxQueueSize(storkutil.Ptr(int64(8080)))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().MaxQueueSize)
	require.EqualValues(t, 8080, *cfg.DHCPDDNS.GetValue().MaxQueueSize.GetValue())
	cfg.SetDHCPDDNSMaxQueueSize(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().MaxQueueSize.GetValue())
}

// Test setting the socket protocol to use when sending requests to D2.
func TestSetDHCPv6DDNSNCRProtocol(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDHCPDDNSNCRProtocol(storkutil.Ptr("UDP"))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().NCRProtocol)
	require.Equal(t, "UDP", *cfg.DHCPDDNS.GetValue().NCRProtocol.GetValue())
	cfg.SetDHCPDDNSNCRProtocol(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().NCRProtocol.GetValue())
}

// Test setting the packet format to use when sending requests to D2.
func TestSetDHCPv6DDNSNCRFormat(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDHCPDDNSNCRFormat(storkutil.Ptr("JSON"))
	require.NotNil(t, cfg.DHCPDDNS)
	require.NotNil(t, cfg.DHCPDDNS.GetValue())
	require.NotNil(t, cfg.DHCPDDNS.GetValue().NCRFormat)
	require.Equal(t, "JSON", *cfg.DHCPDDNS.GetValue().NCRFormat.GetValue())
	cfg.SetDHCPDDNSNCRFormat(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue().NCRFormat.GetValue())
}

// Test setting the DHCP DDNS configuration.
func TestSetDHCPv6DDNS(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	dhcpDDNS := &SettableDHCPDDNS{}
	cfg.SetDHCPDDNS(dhcpDDNS)
	require.NotNil(t, cfg.DHCPDDNS)
	require.Equal(t, dhcpDDNS, cfg.DHCPDDNS.GetValue())
	cfg.SetDHCPDDNS(nil)
	require.Nil(t, cfg.DHCPDDNS.GetValue())
}

// Test setting the number of seconds since the last removal of the expired
// leases, when the next removal should occur.
func TestSetDHCPv6ELPFlushReclaimedTimerWaitTime(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetELPFlushReclaimedTimerWaitTime(storkutil.Ptr(int64(123)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().FlushReclaimedTimerWaitTime)
	require.EqualValues(t, 123, *cfg.ExpiredLeasesProcessing.GetValue().FlushReclaimedTimerWaitTime.GetValue())
	cfg.SetELPFlushReclaimedTimerWaitTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().FlushReclaimedTimerWaitTime.GetValue())
}

// Test setting the length of time in seconds to keep expired leases in the
// lease database (lease affinity).
func TestSetDHCPv6ELPHoldReclaimedTime(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetELPHoldReclaimedTime(storkutil.Ptr(int64(123)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().HoldReclaimedTime)
	require.EqualValues(t, 123, *cfg.ExpiredLeasesProcessing.GetValue().HoldReclaimedTime.GetValue())
	cfg.SetELPHoldReclaimedTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().HoldReclaimedTime.GetValue())
}

// Test setting the maximum number of expired leases that can be processed in
// a single attempt to clean up expired leases from the lease database.
func TestSetDHCPv6ELPMaxReclaimLeases(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetELPMaxReclaimLeases(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimLeases.GetValue())
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimLeases.GetValue())
	cfg.SetELPMaxReclaimLeases(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimLeases.GetValue())
}

// Test setting the maximum time in milliseconds that a single attempt to clean
// up expired leases from the lease database may take.
func TestSetDHCPv6ELPMaxReclaimTime(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetELPMaxReclaimTime(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimTime)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimTime.GetValue())
	cfg.SetELPMaxReclaimTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().MaxReclaimTime.GetValue())
}

// Test setting the length of time in seconds since the last attempt to process
// expired leases before initiating the next attempt.
func TestSetDHCPv6ELPReclaimTimerWaitTime(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetELPReclaimTimerWaitTime(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().ReclaimTimerWaitTime)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.GetValue().ReclaimTimerWaitTime.GetValue())
	cfg.SetELPReclaimTimerWaitTime(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().ReclaimTimerWaitTime.GetValue())
}

// Test setting the maximum number of expired lease-processing cycles which didn't
// result in full cleanup of the expired leases from the lease database,
// after which a warning message is issued.
func TestSetDHCPv6ELPUnwarnedReclaimCycles(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetELPUnwarnedReclaimCycles(storkutil.Ptr(int64(234)))
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue())
	require.NotNil(t, cfg.ExpiredLeasesProcessing.GetValue().UnwarnedReclaimCycles)
	require.EqualValues(t, 234, *cfg.ExpiredLeasesProcessing.GetValue().UnwarnedReclaimCycles.GetValue())
	cfg.SetELPUnwarnedReclaimCycles(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue().UnwarnedReclaimCycles.GetValue())
}

// Test setting the expired leases processing structure.
func TestSetDHCPv6ExpiredLeasesProcessing(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	expiredLeasesProcessing := &SettableExpiredLeasesProcessing{}
	cfg.SetExpiredLeasesProcessing(expiredLeasesProcessing)
	require.NotNil(t, cfg.ExpiredLeasesProcessing)
	require.Equal(t, expiredLeasesProcessing, cfg.ExpiredLeasesProcessing.GetValue())
	cfg.SetExpiredLeasesProcessing(nil)
	require.Nil(t, cfg.ExpiredLeasesProcessing.GetValue())
}

// Test setting a boolean flag enabling an early global reservations lookup.
func TestSetDHCPv6EarlyGlobalReservationsLookup(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetEarlyGlobalReservationsLookup(storkutil.Ptr(false))
	require.NotNil(t, cfg.EarlyGlobalReservationsLookup)
	require.NotNil(t, cfg.EarlyGlobalReservationsLookup.GetValue())
	require.False(t, *cfg.EarlyGlobalReservationsLookup.GetValue())
	cfg.SetEarlyGlobalReservationsLookup(nil)
	require.Nil(t, cfg.EarlyGlobalReservationsLookup.GetValue())
}

// Test setting host reservation identifiers to be used for host reservation lookup.
func TestSetDHCPv6HostReservationIdentifiers(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetHostReservationIdentifiers([]string{"hw-address", "client-id"})
	require.NotNil(t, cfg.HostReservationIdentifiers)
	require.NotNil(t, cfg.HostReservationIdentifiers.GetValue())
	require.ElementsMatch(t, cfg.HostReservationIdentifiers.GetValue(), []string{"hw-address", "client-id"})
	cfg.SetHostReservationIdentifiers(nil)
	require.Nil(t, cfg.HostReservationIdentifiers.GetValue())
}

// Test setting the boolean flag enabling global reservations.
func TestSetDHCPv6ReservationsGlobal(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetReservationsGlobal(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsGlobal)
	require.NotNil(t, cfg.ReservationsGlobal.GetValue())
	require.False(t, *cfg.ReservationsGlobal.GetValue())
	cfg.SetReservationsGlobal(nil)
	require.Nil(t, cfg.ReservationsGlobal.GetValue())
}

// Test setting the boolean flag enabling in-subnet reservations.
func TestSetDHCPv6ReservationsInSubnet(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetReservationsInSubnet(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsInSubnet)
	require.NotNil(t, cfg.ReservationsInSubnet.GetValue())
	require.False(t, *cfg.ReservationsInSubnet.GetValue())
	cfg.SetReservationsInSubnet(nil)
	require.Nil(t, cfg.ReservationsInSubnet.GetValue())
}

// Test setting the boolean flag enabling out-of-pool reservations.
func TestSetDHCPv6ReservationsOutOfPool(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetReservationsOutOfPool(storkutil.Ptr(false))
	require.NotNil(t, cfg.ReservationsOutOfPool)
	require.NotNil(t, cfg.ReservationsOutOfPool.GetValue())
	require.False(t, *cfg.ReservationsOutOfPool.GetValue())
	cfg.SetReservationsOutOfPool(nil)
	require.Nil(t, cfg.ReservationsOutOfPool.GetValue())
}

// Test setting DHCPv4 valid lifetime.
func TestSetDHCPv6ValidLifetime(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetValidLifetime(storkutil.Ptr(int64(1111)))
	require.NotNil(t, cfg.ValidLifetime)
	require.NotNil(t, cfg.ValidLifetime.GetValue())
	require.EqualValues(t, 1111, *cfg.ValidLifetime.GetValue())
	cfg.SetValidLifetime(nil)
	require.Nil(t, cfg.ValidLifetime.GetValue())
}

// Test setting allocator for prefix delegation.
func TestSetDHCPv6PDAllocator(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetPDAllocator(storkutil.Ptr("flq"))
	require.NotNil(t, cfg.PDAllocator)
	require.NotNil(t, cfg.PDAllocator.GetValue())
	require.EqualValues(t, "flq", *cfg.PDAllocator.GetValue())
	cfg.SetPDAllocator(nil)
	require.Nil(t, cfg.PDAllocator.GetValue())
}

// Test setting DHCPv6 option data.
func TestSetDHCPv6OptionData(t *testing.T) {
	cfg := &SettableDHCPv6Config{}
	cfg.SetDHCPOptions([]SingleOptionData{
		{
			Name:       "routers",
			AlwaysSend: true,
			Code:       3,
			CSVFormat:  true,
			Data:       "foobar",
			Space:      "dhcp6",
		},
	})
	options := cfg.OptionData.GetValue()
	require.Len(t, options, 1)
	require.Equal(t, "routers", options[0].Name)
	require.True(t, options[0].AlwaysSend)
	require.EqualValues(t, 3, options[0].Code)
	require.True(t, options[0].CSVFormat)
	require.Equal(t, "foobar", options[0].Data)
	require.Equal(t, "dhcp6", options[0].Space)

	cfg.SetDHCPOptions(nil)
	require.Nil(t, cfg.OptionData.GetValue())
}

// Test setting DHCPv6 option data.
func TestSetDHCPv6Options(t *testing.T) {
	// Arrange
	cfg := &SettableDHCPv6Config{}

	// Act
	cfg.SetDHCPOptions([]SingleOptionData{
		{
			Name:       "routers",
			AlwaysSend: true,
			Code:       3,
			CSVFormat:  true,
			Data:       "foobar",
			Space:      "dhcp6",
		},
	})

	// Assert
	options := cfg.OptionData.GetValue()
	require.Len(t, options, 1)
	option := options[0]
	require.Equal(t, "routers", option.Name)
	require.True(t, option.AlwaysSend)
	require.EqualValues(t, 3, option.Code)
	require.True(t, option.CSVFormat)
	require.Equal(t, "foobar", option.Data)
	require.Equal(t, "dhcp6", option.Space)
}

// Test setting nil as DHCPv6 option data.
func TestSetDHCPv6NilOptions(t *testing.T) {
	// Arrange
	cfg := &SettableDHCPv6Config{}

	// Act
	cfg.SetDHCPOptions(nil)

	// Assert
	require.Nil(t, cfg.OptionData.GetValue())
}
