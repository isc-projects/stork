package keaconfig

import (
	"encoding/json"

	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

var (
	_ commonConfigAccessor = (*DHCPv4Config)(nil)
	_ dhcpConfigAccessor   = (*DHCPv4Config)(nil)
	_ commonConfigAccessor = (*DHCPv6Config)(nil)
	_ dhcpConfigAccessor   = (*DHCPv6Config)(nil)
	_ dhcpConfigModifier   = (*DHCPv4Config)(nil)
	_ dhcp4ConfigModifier  = (*DHCPv4Config)(nil)
	_ dhcpConfigModifier   = (*DHCPv6Config)(nil)
	_ dhcp6ConfigModifier  = (*DHCPv6Config)(nil)
)

// An interface for fetching the parts of the DHCP configurations.
// Both DHCPv4 and DHCPv6 configuration structures must implement
// this interface. It is used by the keaconfig.Config.
type dhcpConfigAccessor interface {
	// Returns common configuration structures for DHCPv4 and DHCPv6.
	GetCommonDHCPConfig() CommonDHCPConfig
	// Returns a slice of interfaces to the SharedNetwork4 or SharedNetwork6,
	// depending on the server type.
	GetSharedNetworks(includeRootSubnets bool) []SharedNetwork
	// Searches for a subnet with a given prefix.
	GetSubnetByPrefix(prefix string) Subnet
	// Returns a slice of interfaces to the Subnet4 or Subnet6, depending
	// on the server type.
	GetSubnets() []Subnet
	// Returns a slice of the option data.
	GetDHCPOptions() []SingleOptionData
}

// An interface for setting the parts of the DHCP configurations. Both
// DHCPv4 and DHCPv6 configuration structures must implement this
// interface.
type dhcpConfigModifier interface {
	// Sets an allocator.
	SetAllocator(allocator *string)
	// Sets cache threshold.
	SetCacheThreshold(cacheThreshold *float32)
	// Sets boolean flag indicating if DDNS updates should be sent.
	SetDDNSSendUpdates(ddnsSendUpdates *bool)
	// Sets boolean flag indicating whether the DHCP server should override the
	// client's wish to not update the DNS.
	SetDDNSOverrideNoUpdate(ddnsOverrideNoUpdate *bool)
	// Sets the boolean flag indicating whether the DHCP server should ignore the
	// client's wish to update the DNS on its own.
	SetDDNSOverrideClientUpdate(ddnsOverrideClientUpdate *bool)
	// Sets the enumeration specifying whether the server should honor
	// the hostname or Client FQDN sent by the client or replace this name.
	SetDDNSReplaceClientName(ddnsReplaceClientName *string)
	// Sets a prefix to be prepended to the generated Client FQDN.
	SetDDNSGeneratedPrefix(ddnsGeneratedPrefix *string)
	// Sets a suffix appended to the partial name sent to the DNS.
	SetDDNSQualifyingSuffix(ddnsQualifyingSuffix *string)
	// Sets a boolean flag, which when true instructs the server to always
	// update DNS when leases are renewed, even if the DNS information
	// has not changed.
	SetDDNSUpdateOnRenew(ddnsUpdateOnRenew *bool)
	// Sets a boolean flag which is passed to kea-dhcp-ddns with each DDNS
	// update request, to indicate whether DNS update conflict
	// resolution as described in RFC 4703 should be employed for the
	// given update request.
	SetDDNSUseConflictResolution(ddnsUseConflictResolution *bool)
	// Sets the the percent of the lease's lifetime to use for the DNS TTL.
	SetDDNSTTLPercent(ddnsTTLPercent *float32)
	// Sets the number of seconds since the last removal of the expired
	// leases, when the next removal should occur.
	SetELPFlushReclaimedTimerWaitTime(flushReclaimedTimerWaitTime *int64)
	// Sets the length of time in seconds to keep expired leases in the
	// lease database (lease affinity).
	SetELPHoldReclaimedTime(holdReclaimedTime *int64)
	// Sets the maximum number of expired leases that can be processed in
	// a single attempt to clean up expired leases from the lease database.
	SetELPMaxReclaimLeases(maxReclaimedLeases *int64)
	// Sets the maximum time in milliseconds that a single attempt to clean
	// up expired leases from the lease database may take.
	SetELPMaxReclaimTime(maxReclaimTime *int64)
	// Sets the length of time in seconds since the last attempt to process
	// expired leases before initiating the next attempt.
	SetELPReclaimTimerWaitTime(reclaimTimerWaitTime *int64)
	// Sets the maximum number of expired lease-processing cycles which didn't
	// result in full cleanup of the exired leases from the lease database,
	// after which a warning message is issued.
	SetELPUnwarnedReclaimCycles(unwarnedReclaimCycles *int64)
	// Sets the expired leases processing structure.
	SetExpiredLeasesProcessing(expiredLeasesProcessing *ExpiredLeasesProcessing)
	// Sets a boolean flag enabling an early global reservations lookup.
	SetEarlyGlobalReservationsLookup(earlyGlobalReservationsLookup *bool)
	// Sets host reservation identifiers to be used for host reservation lookup.
	SetHostReservationIdentifiers(hostReservationIdentifiers []string)
	// Sets the boolean flag enabling global reservations.
	SetReservationsGlobal(reservationsGlobal *bool)
	// Sets the boolean flag enabling in-subnet reservations.
	SetReservationsInSubnet(reservationsInSubnet *bool)
	// Sets the boolean flag enabling out-of-pool reservations.
	SetReservationsOutOfPool(reservationsOutOfPool *bool)
	// Sets global valid lifetime.
	SetValidLifetime(validLifetime *int64)
}

// An interface for setting the parts of the DHCPv4 configurations. It is
// must be implemented by the DHCPv4 configuration store.
type dhcp4ConfigModifier interface {
	SetAuthoritative(authoritative *bool)
	SetEchoClientID(echoClientID *bool)
}

// An interface for setting the parts of the DHCPv6 configurations. It is
// must be implemented by the DHCPv6 configuration store.
type dhcp6ConfigModifier interface {
	SetPDAllocator(pdAllocator *string)
}

// Represents Kea DHCPv4 configuration.
type DHCPv4Config struct {
	CommonDHCPConfig
	Authoritative   *bool              `json:"authoritative,omitempty"`
	BootFileName    *string            `json:"boot-file-name,omitempty"`
	EchoClientID    *bool              `json:"echo-client-id,omitempty"`
	MatchClientID   *bool              `json:"match-client-id,omitempty"`
	NextServer      *string            `json:"next-server,omitempty"`
	OptionData      []SingleOptionData `json:"option-data,omitempty"`
	ServerHostname  *string            `json:"server-hostname,omitempty"`
	SharedNetworks  []SharedNetwork4   `json:"shared-networks,omitempty"`
	Subnet4         []Subnet4          `json:"subnet4,omitempty"`
	Subnet4ByPrefix map[string]Subnet4 `json:"-"`
}

// Represents Kea DHCPv6 configuration.
type DHCPv6Config struct {
	CommonDHCPConfig
	PreferredLifetimeParameters
	PDAllocator     *string            `json:"pd-allocator,omitempty"`
	RapidCommit     *bool              `json:"rapid-commit,omitempty"`
	OptionData      []SingleOptionData `json:"option-data,omitempty"`
	SharedNetworks  []SharedNetwork6   `json:"shared-networks,omitempty"`
	Subnet6         []Subnet6          `json:"subnet6,omitempty"`
	Subnet6ByPrefix map[string]Subnet6 `json:"-"`
}

// Represents common configuration parameters for the DHCPv4 and DHCPv6 servers.
type CommonDHCPConfig struct {
	CacheParameters
	DDNSParameters
	HostnameCharParameters
	ReservationParameters
	TimerParameters
	ValidLifetimeParameters
	Allocator               *string                  `json:"allocator,omitempty"`
	ClientClasses           []ClientClass            `json:"client-classes,omitempty"`
	ConfigControl           *ConfigControl           `json:"config-control,omitempty"`
	ControlSocket           *ControlSocket           `json:"control-socket,omitempty"`
	ExpiredLeasesProcessing *ExpiredLeasesProcessing `json:"expired-leases-processing,omitempty"`
	HostsDatabase           *Database                `json:"hosts-database,omitempty"`
	HostsDatabases          []Database               `json:"hosts-databases,omitempty"`
	HookLibraries           []HookLibrary            `json:"hooks-libraries,omitempty"`
	LeaseDatabase           *Database                `json:"lease-database,omitempty"`
	Loggers                 []Logger                 `json:"loggers,omitempty"`
	MultiThreading          *MultiThreading          `json:"multi-threading,omitempty"`
	Reservations            []Reservation            `json:"reservations,omitempty"`
	StoreExtendedInfo       *bool                    `json:"store-extended-info,omitempty"`
}

// Represents the collection of settings pertaining to the expired
// leases processing.
type ExpiredLeasesProcessing struct {
	FlushReclaimedTimerWaitTime *int64 `json:"flush-reclaimed-timer-wait-time,omitempty"`
	HoldReclaimedTime           *int64 `json:"hold-reclaimed-time,omitempty"`
	MaxReclaimLeases            *int64 `json:"max-reclaim-leases,omitempty"`
	MaxReclaimTime              *int64 `json:"max-reclaim-time,omitempty"`
	ReclaimTimerWaitTime        *int64 `json:"reclaim-timer-wait-time,omitempty"`
	UnwarnedReclaimCycles       *int64 `json:"unwarned-reclaim-cycles,omitempty"`
}

// Represents the global DHCP multi-threading parameters.
type MultiThreading struct {
	EnableMultiThreading *bool `json:"enable-multi-threading,omitempty"`
	ThreadPoolSize       *int  `json:"thread-pool-size,omitempty"`
	PacketQueueSize      *int  `json:"packet-queue-size,omitempty"`
}

// Unmarshals the DHCPv4 configuration and builds an index of the
// subnets by prefix.
func (c *DHCPv4Config) UnmarshalJSON(data []byte) error {
	type t DHCPv4Config
	if err := json.Unmarshal(data, (*t)(c)); err != nil {
		return errors.Wrapf(err, "problem unmarshalling the DHCPv4 configuration")
	}
	c.Subnet4ByPrefix = make(map[string]Subnet4)
	for i := range c.Subnet4 {
		if prefix, err := c.Subnet4[i].GetCanonicalPrefix(); err == nil {
			c.Subnet4ByPrefix[prefix] = c.Subnet4[i]
		}
	}
	for _, sn := range c.SharedNetworks {
		for _, s := range sn.Subnet4 {
			subnet := s
			if prefix, err := s.GetCanonicalPrefix(); err == nil {
				c.Subnet4ByPrefix[prefix] = subnet
			}
		}
	}
	return nil
}

// Returns the hooks libraries configured in the DHCPv4 server.
func (c *DHCPv4Config) GetHookLibraries() HookLibraries {
	return c.HookLibraries
}

// Returns the loggers configured in the DHCPv4 server.
func (c *DHCPv4Config) GetLoggers() []Logger {
	return c.Loggers
}

// Returns the DHCPv4 configuration parameters common for the  DHCPv4 and
// DHCPv6 servers.
func (c *DHCPv4Config) GetCommonDHCPConfig() CommonDHCPConfig {
	return c.CommonDHCPConfig
}

// Returns a slice of interfaces to the DHCPv4 shared networks.
func (c *DHCPv4Config) GetSharedNetworks(includeRootSubnets bool) (sharedNetworks []SharedNetwork) {
	for i := range c.SharedNetworks {
		sharedNetworks = append(sharedNetworks, &c.SharedNetworks[i])
	}
	if includeRootSubnets {
		sharedNetworks = append(sharedNetworks, &SharedNetwork4{
			Subnet4: c.Subnet4,
		})
	}
	return
}

// Finds a DHCPv4 subnet by prefix. It returns nil ponter if the subnet
// is not found.
func (c *DHCPv4Config) GetSubnetByPrefix(prefix string) Subnet {
	if cidr := storkutil.ParseIP(prefix); cidr != nil {
		if subnet, ok := c.Subnet4ByPrefix[cidr.GetNetworkPrefixWithLength()]; ok {
			return &subnet
		}
	}
	return nil
}

// Returns a slice of interfaces to the DHCPv4 subnets.
func (c *DHCPv4Config) GetSubnets() (subnets []Subnet) {
	for i := range c.Subnet4 {
		subnets = append(subnets, &c.Subnet4[i])
	}
	return
}

// Returns a slice of DHCP option data.
func (c *DHCPv4Config) GetDHCPOptions() (options []SingleOptionData) {
	return c.OptionData
}

// Sets an allocator.
func (c *DHCPv4Config) SetAllocator(allocator *string) {
	c.Allocator = allocator
}

// Sets cache threshold.
func (c *DHCPv4Config) SetCacheThreshold(cacheThreshold *float32) {
	c.CacheParameters.CacheThreshold = cacheThreshold
}

// Sets boolean flag indicating if DDNS updates should be sent.
func (c *DHCPv4Config) SetDDNSSendUpdates(ddnsSendUpdates *bool) {
	c.CommonDHCPConfig.DDNSSendUpdates = ddnsSendUpdates
}

// Sets boolean flag indicating whether the DHCP server should override the
// client's wish to not update the DNS.
func (c *DHCPv4Config) SetDDNSOverrideNoUpdate(ddnsOverrideNoUpdate *bool) {
	c.CommonDHCPConfig.DDNSOverrideNoUpdate = ddnsOverrideNoUpdate
}

// Sets the boolean flag indicating whether the DHCP server should ignore the
// client's wish to update the DNS on its own.
func (c *DHCPv4Config) SetDDNSOverrideClientUpdate(ddnsOverrideClientUpdate *bool) {
	c.CommonDHCPConfig.DDNSOverrideClientUpdate = ddnsOverrideClientUpdate
}

// Sets the enumeration specifying whether the server should honor
// the hostname or Client FQDN sent by the client or replace this name.
func (c *DHCPv4Config) SetDDNSReplaceClientName(ddnsReplaceClientName *string) {
	c.CommonDHCPConfig.DDNSReplaceClientName = ddnsReplaceClientName
}

// Sets a prefix to be prepended to the generated Client FQDN.
func (c *DHCPv4Config) SetDDNSGeneratedPrefix(ddnsGeneratedPrefix *string) {
	c.CommonDHCPConfig.DDNSGeneratedPrefix = ddnsGeneratedPrefix
}

// Sets a suffix appended to the partial name sent to the DNS.
func (c *DHCPv4Config) SetDDNSQualifyingSuffix(ddnsQualifyingSuffix *string) {
	c.CommonDHCPConfig.DDNSQualifyingSuffix = ddnsQualifyingSuffix
}

// Sets a boolean flag, which when true instructs the server to always
// update DNS when leases are renewed, even if the DNS information
// has not changed.
func (c *DHCPv4Config) SetDDNSUpdateOnRenew(ddnsUpdateOnRenew *bool) {
	c.CommonDHCPConfig.DDNSUpdateOnRenew = ddnsUpdateOnRenew
}

// Sets a boolean flag which is passed to kea-dhcp-ddns with each DDNS
// update request, to indicate whether DNS update conflict
// resolution as described in RFC 4703 should be employed for the
// given update request.
func (c *DHCPv4Config) SetDDNSUseConflictResolution(ddnsUseConflictResolution *bool) {
	c.CommonDHCPConfig.DDNSUseConflictResolution = ddnsUseConflictResolution
}

// Sets the the percent of the lease's lifetime to use for the DNS TTL.
func (c *DHCPv4Config) SetDDNSTTLPercent(ddnsTTLPercent *float32) {
	c.CommonDHCPConfig.DDNSTTLPercent = ddnsTTLPercent
}

// Initializes the structure holding expired leases processing settings.
func (c *DHCPv4Config) ensureExpiredLeasesProcessing() *ExpiredLeasesProcessing {
	if c.ExpiredLeasesProcessing == nil {
		c.ExpiredLeasesProcessing = &ExpiredLeasesProcessing{}
	}
	return c.ExpiredLeasesProcessing
}

// Sets the number of seconds since the last removal of the expired
// leases, when the next removal should occur.
func (c *DHCPv4Config) SetELPFlushReclaimedTimerWaitTime(flushReclaimedTimerWaitTime *int64) {
	c.ensureExpiredLeasesProcessing().FlushReclaimedTimerWaitTime = flushReclaimedTimerWaitTime
}

// Sets the length of time in seconds to keep expired leases in the
// lease database (lease affinity).
func (c *DHCPv4Config) SetELPHoldReclaimedTime(holdReclaimedTime *int64) {
	c.ensureExpiredLeasesProcessing().HoldReclaimedTime = holdReclaimedTime
}

// Sets the maximum number of expired leases that can be processed in
// a single attempt to clean up expired leases from the lease database.
func (c *DHCPv4Config) SetELPMaxReclaimLeases(maxReclaimedLeases *int64) {
	c.ensureExpiredLeasesProcessing().MaxReclaimLeases = maxReclaimedLeases
}

// Sets the maximum time in milliseconds that a single attempt to clean
// up expired leases from the lease database may take.
func (c *DHCPv4Config) SetELPMaxReclaimTime(maxReclaimTime *int64) {
	c.ensureExpiredLeasesProcessing().MaxReclaimTime = maxReclaimTime
}

// Sets the length of time in seconds since the last attempt to process
// expired leases before initiating the next attempt.
func (c *DHCPv4Config) SetELPReclaimTimerWaitTime(reclaimTimerWaitTime *int64) {
	c.ensureExpiredLeasesProcessing().ReclaimTimerWaitTime = reclaimTimerWaitTime
}

// Sets the maximum number of expired lease-processing cycles which didn't
// result in full cleanup of the exired leases from the lease database,
// after which a warning message is issued.
func (c *DHCPv4Config) SetELPUnwarnedReclaimCycles(unwarnedReclaimCycles *int64) {
	c.ensureExpiredLeasesProcessing().UnwarnedReclaimCycles = unwarnedReclaimCycles
}

// Sets the expired leases processing structure.
func (c *DHCPv4Config) SetExpiredLeasesProcessing(expiredLeasesProcessing *ExpiredLeasesProcessing) {
	c.ExpiredLeasesProcessing = expiredLeasesProcessing
}

// Sets the boolean flag enabling global reservations.
func (c *DHCPv4Config) SetReservationsGlobal(reservationsGlobal *bool) {
	c.ReservationParameters.ReservationsGlobal = reservationsGlobal
}

// Sets the boolean flag enabling in-subnet reservations.
func (c *DHCPv4Config) SetReservationsInSubnet(reservationsInSubnet *bool) {
	c.ReservationParameters.ReservationsInSubnet = reservationsInSubnet
}

// Sets the boolean flag enabling out-of-pool reservations.
func (c *DHCPv4Config) SetReservationsOutOfPool(reservationsOutOfPool *bool) {
	c.ReservationParameters.ReservationsOutOfPool = reservationsOutOfPool
}

// Sets a boolean flag enabling an early global reservations lookup.
func (c *DHCPv4Config) SetEarlyGlobalReservationsLookup(earlyGlobalReservationsLookup *bool) {
	c.ReservationParameters.EarlyGlobalReservationsLookup = earlyGlobalReservationsLookup
}

// Sets host reservation identifiers to be used for host reservation lookup.
func (c *DHCPv4Config) SetHostReservationIdentifiers(hostReservationIdentifiers []string) {
	c.ReservationParameters.HostReservationIdentifiers = hostReservationIdentifiers
}

// Sets global valid lifetime.
func (c *DHCPv4Config) SetValidLifetime(validLiftime *int64) {
	c.ValidLifetimeParameters.ValidLifetime = validLiftime
}

// Sets a boolean flag indicating whether the server is authoritative.
func (c *DHCPv4Config) SetAuthoritative(authoritative *bool) {
	c.Authoritative = authoritative
}

// Sets a boolean flag indicating whether the server should return client
// ID in its responses.
func (c *DHCPv4Config) SetEchoClientID(echoClientID *bool) {
	c.EchoClientID = echoClientID
}

// Unmarshals the DHCPv6 configuration and builds an index of the
// subnets by prefix.
func (c *DHCPv6Config) UnmarshalJSON(data []byte) error {
	type t DHCPv6Config
	if err := json.Unmarshal(data, (*t)(c)); err != nil {
		return errors.Wrapf(err, "problem unmarshalling the DHCPv6 configuration")
	}
	c.Subnet6ByPrefix = make(map[string]Subnet6)
	for i := range c.Subnet6 {
		if prefix, err := c.Subnet6[i].GetCanonicalPrefix(); err == nil {
			c.Subnet6ByPrefix[prefix] = c.Subnet6[i]
		}
	}
	for _, sn := range c.SharedNetworks {
		for _, s := range sn.Subnet6 {
			subnet := s
			if prefix, err := s.GetCanonicalPrefix(); err == nil {
				c.Subnet6ByPrefix[prefix] = subnet
			}
		}
	}
	return nil
}

// Returns the hooks libraries configured in the DHCPv6 server.
func (c *DHCPv6Config) GetHookLibraries() HookLibraries {
	return c.HookLibraries
}

// Returns the loggers configured in the DHCPv6 server.
func (c *DHCPv6Config) GetLoggers() []Logger {
	return c.Loggers
}

// Returns the DHCPv6 configuration parameters common for the  DHCPv4 and
// DHCPv6 servers.
func (c *DHCPv6Config) GetCommonDHCPConfig() CommonDHCPConfig {
	return c.CommonDHCPConfig
}

// Returns a slice of interfaces to the DHCPv6 shared networks.
func (c *DHCPv6Config) GetSharedNetworks(includeRootSubnets bool) (sharedNetworks []SharedNetwork) {
	for i := range c.SharedNetworks {
		sharedNetworks = append(sharedNetworks, &c.SharedNetworks[i])
	}
	if includeRootSubnets {
		sharedNetworks = append(sharedNetworks, &SharedNetwork6{
			Subnet6: c.Subnet6,
		})
	}
	return
}

// Finds a DHCPv6 subnet by prefix. It returns nil ponter if the subnet
// is not found.
func (c *DHCPv6Config) GetSubnetByPrefix(prefix string) Subnet {
	if cidr := storkutil.ParseIP(prefix); cidr != nil {
		if subnet, ok := c.Subnet6ByPrefix[cidr.GetNetworkPrefixWithLength()]; ok {
			return &subnet
		}
	}
	return nil
}

// Returns a slice of interfaces to the DHCPv6 subnets.
func (c *DHCPv6Config) GetSubnets() (subnets []Subnet) {
	for i := range c.Subnet6 {
		subnets = append(subnets, &c.Subnet6[i])
	}
	return
}

// Returns a slice of DHCP option data.
func (c *DHCPv6Config) GetDHCPOptions() (options []SingleOptionData) {
	return c.OptionData
}

// Sets an allocator.
func (c *DHCPv6Config) SetAllocator(allocator *string) {
	c.Allocator = allocator
}

// Sets cache threshold.
func (c *DHCPv6Config) SetCacheThreshold(cacheThreshold *float32) {
	c.CacheParameters.CacheThreshold = cacheThreshold
}

// Sets boolean flag indicating if DDNS updates should be sent.
func (c *DHCPv6Config) SetDDNSSendUpdates(ddnsSendUpdates *bool) {
	c.CommonDHCPConfig.DDNSSendUpdates = ddnsSendUpdates
}

// Sets boolean flag indicating whether the DHCP server should override the
// client's wish to not update the DNS.
func (c *DHCPv6Config) SetDDNSOverrideNoUpdate(ddnsOverrideNoUpdate *bool) {
	c.CommonDHCPConfig.DDNSOverrideNoUpdate = ddnsOverrideNoUpdate
}

// Sets the boolean flag indicating whether the DHCP server should ignore the
// client's wish to update the DNS on its own.
func (c *DHCPv6Config) SetDDNSOverrideClientUpdate(ddnsOverrideClientUpdate *bool) {
	c.CommonDHCPConfig.DDNSOverrideClientUpdate = ddnsOverrideClientUpdate
}

// Sets the enumeration specifying whether the server should honor
// the hostname or Client FQDN sent by the client or replace this name.
func (c *DHCPv6Config) SetDDNSReplaceClientName(ddnsReplaceClientName *string) {
	c.CommonDHCPConfig.DDNSReplaceClientName = ddnsReplaceClientName
}

// Sets a prefix to be prepended to the generated Client FQDN.
func (c *DHCPv6Config) SetDDNSGeneratedPrefix(ddnsGeneratedPrefix *string) {
	c.CommonDHCPConfig.DDNSGeneratedPrefix = ddnsGeneratedPrefix
}

// Sets a suffix appended to the partial name sent to the DNS.
func (c *DHCPv6Config) SetDDNSQualifyingSuffix(ddnsQualifyingSuffix *string) {
	c.CommonDHCPConfig.DDNSQualifyingSuffix = ddnsQualifyingSuffix
}

// Sets a boolean flag, which when true instructs the server to always
// update DNS when leases are renewed, even if the DNS information
// has not changed.
func (c *DHCPv6Config) SetDDNSUpdateOnRenew(ddnsUpdateOnRenew *bool) {
	c.CommonDHCPConfig.DDNSUpdateOnRenew = ddnsUpdateOnRenew
}

// Sets a boolean flag which is passed to kea-dhcp-ddns with each DDNS
// update request, to indicate whether DNS update conflict
// resolution as described in RFC 4703 should be employed for the
// given update request.
func (c *DHCPv6Config) SetDDNSUseConflictResolution(ddnsUseConflictResolution *bool) {
	c.CommonDHCPConfig.DDNSUseConflictResolution = ddnsUseConflictResolution
}

// Sets the the percent of the lease's lifetime to use for the DNS TTL.
func (c *DHCPv6Config) SetDDNSTTLPercent(ddnsTTLPercent *float32) {
	c.CommonDHCPConfig.DDNSTTLPercent = ddnsTTLPercent
}

// Initializes the structure holding expired leases processing settings.
func (c *DHCPv6Config) ensureExpiredLeasesProcessing() *ExpiredLeasesProcessing {
	if c.ExpiredLeasesProcessing == nil {
		c.ExpiredLeasesProcessing = &ExpiredLeasesProcessing{}
	}
	return c.ExpiredLeasesProcessing
}

// Sets the number of seconds since the last removal of the expired
// leases, when the next removal should occur.
func (c *DHCPv6Config) SetELPFlushReclaimedTimerWaitTime(flushReclaimedTimerWaitTime *int64) {
	c.ensureExpiredLeasesProcessing().FlushReclaimedTimerWaitTime = flushReclaimedTimerWaitTime
}

// Sets the length of time in seconds to keep expired leases in the
// lease database (lease affinity).
func (c *DHCPv6Config) SetELPHoldReclaimedTime(holdReclaimedTime *int64) {
	c.ensureExpiredLeasesProcessing().HoldReclaimedTime = holdReclaimedTime
}

// Sets the maximum number of expired leases that can be processed in
// a single attempt to clean up expired leases from the lease database.
func (c *DHCPv6Config) SetELPMaxReclaimLeases(maxReclaimedLeases *int64) {
	c.ensureExpiredLeasesProcessing().MaxReclaimLeases = maxReclaimedLeases
}

// Sets the maximum time in milliseconds that a single attempt to clean
// up expired leases from the lease database may take.
func (c *DHCPv6Config) SetELPMaxReclaimTime(maxReclaimTime *int64) {
	c.ensureExpiredLeasesProcessing().MaxReclaimTime = maxReclaimTime
}

// Sets the length of time in seconds since the last attempt to process
// expired leases before initiating the next attempt.
func (c *DHCPv6Config) SetELPReclaimTimerWaitTime(reclaimTimerWaitTime *int64) {
	c.ensureExpiredLeasesProcessing().ReclaimTimerWaitTime = reclaimTimerWaitTime
}

// Sets the maximum number of expired lease-processing cycles which didn't
// result in full cleanup of the exired leases from the lease database,
// after which a warning message is issued.
func (c *DHCPv6Config) SetELPUnwarnedReclaimCycles(unwarnedReclaimCycles *int64) {
	c.ensureExpiredLeasesProcessing().UnwarnedReclaimCycles = unwarnedReclaimCycles
}

// Sets the expired leases processing structure.
func (c *DHCPv6Config) SetExpiredLeasesProcessing(expiredLeasesProcessing *ExpiredLeasesProcessing) {
	c.ExpiredLeasesProcessing = expiredLeasesProcessing
}

// Sets the boolean flag enabling global reservations.
func (c *DHCPv6Config) SetReservationsGlobal(reservationsGlobal *bool) {
	c.ReservationParameters.ReservationsGlobal = reservationsGlobal
}

// Sets the boolean flag enabling in-subnet reservations.
func (c *DHCPv6Config) SetReservationsInSubnet(reservationsInSubnet *bool) {
	c.ReservationParameters.ReservationsInSubnet = reservationsInSubnet
}

// Sets the boolean flag enabling out-of-pool reservations.
func (c *DHCPv6Config) SetReservationsOutOfPool(reservationsOutOfPool *bool) {
	c.ReservationParameters.ReservationsOutOfPool = reservationsOutOfPool
}

// Sets a boolean flag enabling an early global reservations lookup.
func (c *DHCPv6Config) SetEarlyGlobalReservationsLookup(earlyGlobalReservationsLookup *bool) {
	c.ReservationParameters.EarlyGlobalReservationsLookup = earlyGlobalReservationsLookup
}

// Sets host reservation identifiers to be used for host reservation lookup.
func (c *DHCPv6Config) SetHostReservationIdentifiers(hostReservationIdentifiers []string) {
	c.ReservationParameters.HostReservationIdentifiers = hostReservationIdentifiers
}

// Sets global valid lifetime.
func (c *DHCPv6Config) SetValidLifetime(validLiftime *int64) {
	c.ValidLifetimeParameters.ValidLifetime = validLiftime
}

// Sets allocator for prefix delegation.
func (c *DHCPv6Config) SetPDAllocator(pdAllocator *string) {
	c.PDAllocator = pdAllocator
}
