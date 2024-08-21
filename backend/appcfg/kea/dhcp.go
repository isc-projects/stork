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
	_ dhcpConfigModifier   = (*SettableDHCPv4Config)(nil)
	_ dhcp4ConfigModifier  = (*SettableDHCPv4Config)(nil)
	_ dhcpConfigModifier   = (*SettableDHCPv6Config)(nil)
	_ dhcp6ConfigModifier  = (*SettableDHCPv6Config)(nil)
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
	// Sets a DDNS conflict resolution mode.
	SetDDNSConflictResolutionMode(ddnsConflictResolutionMode *string)
	// Sets the the percent of the lease's lifetime to use for the DNS TTL.
	SetDDNSTTLPercent(ddnsTTLPercent *float32)
	// Enables connectivity with the DHCP DDNS daemon and sending DNS updates.
	SetDHCPDDNSEnableUpdates(enableUpdates *bool)
	// Sets the IP address on which D2 listens for requests.
	SetDHCPDDNSServerIP(serverIP *string)
	// Sets the port on which D2 listens for requests.
	SetDHCPDDNSServerPort(serverPort *int64)
	// Sets the IP address which DHCP server uses to send requests to D2.
	SetDHCPDDNSSenderIP(senderIP *string)
	// Sets the port which DHCP server uses to send requests to D2.
	SetDHCPDDNSSenderPort(senderPort *int64)
	// Sets the maximum number of requests allowed to queue while waiting to be sent to D2.
	SetDHCPDDNSMaxQueueSize(maxQueueSize *int64)
	// Sets the socket protocol to use when sending requests to D2.
	SetDHCPDDNSNCRProtocol(ncrProtocol *string)
	// Sets the packet format to use when sending requests to D2.
	SetDHCPDDNSNCRFormat(ncrFormat *string)
	// Sets the DHCP DDNS structure.
	SetDHCPDDNS(dhcpDDNS *SettableDHCPDDNS)
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
	// result in full cleanup of the expired leases from the lease database,
	// after which a warning message is issued.
	SetELPUnwarnedReclaimCycles(unwarnedReclaimCycles *int64)
	// Sets the expired leases processing structure.
	SetExpiredLeasesProcessing(expiredLeasesProcessing *SettableExpiredLeasesProcessing)
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
	// Sets DHCP options data.
	SetDHCPOptions(options []SingleOptionData)
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
	ServerHostname  *string            `json:"server-hostname,omitempty"`
	SharedNetworks  []SharedNetwork4   `json:"shared-networks,omitempty"`
	Subnet4         []Subnet4          `json:"subnet4,omitempty"`
	Subnet4ByPrefix map[string]Subnet4 `json:"-"`
}

// Represents settable Kea DHCPv4 configuration.
type SettableDHCPv4Config struct {
	SettableCommonDHCPConfig
	Authoritative *storkutil.Nullable[bool]   `json:"authoritative,omitempty"`
	BootFileName  *storkutil.Nullable[string] `json:"boot-file-name,omitempty"`
	EchoClientID  *storkutil.Nullable[bool]   `json:"echo-client-id,omitempty"`
	MatchClientID *storkutil.Nullable[bool]   `json:"match-client-id,omitempty"`
	NextServer    *storkutil.Nullable[string] `json:"next-server,omitempty"`
}

// Represents Kea DHCPv6 configuration.
type DHCPv6Config struct {
	CommonDHCPConfig
	PreferredLifetimeParameters
	PDAllocator     *string            `json:"pd-allocator,omitempty"`
	RapidCommit     *bool              `json:"rapid-commit,omitempty"`
	SharedNetworks  []SharedNetwork6   `json:"shared-networks,omitempty"`
	Subnet6         []Subnet6          `json:"subnet6,omitempty"`
	Subnet6ByPrefix map[string]Subnet6 `json:"-"`
}

// Represents settable Kea DHCPv6 configuration.
type SettableDHCPv6Config struct {
	SettableCommonDHCPConfig
	PDAllocator *storkutil.Nullable[string] `json:"pd-allocator,omitempty"`
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
	DHCPDDNS                *DHCPDDNS                `json:"dhcp-ddns,omitempty"`
	ExpiredLeasesProcessing *ExpiredLeasesProcessing `json:"expired-leases-processing,omitempty"`
	HostsDatabase           *Database                `json:"hosts-database,omitempty"`
	HostsDatabases          []Database               `json:"hosts-databases,omitempty"`
	HookLibraries           []HookLibrary            `json:"hooks-libraries,omitempty"`
	LeaseDatabase           *Database                `json:"lease-database,omitempty"`
	Loggers                 []Logger                 `json:"loggers,omitempty"`
	MultiThreading          *MultiThreading          `json:"multi-threading,omitempty"`
	OptionData              []SingleOptionData       `json:"option-data,omitempty"`
	Reservations            []Reservation            `json:"reservations,omitempty"`
	StoreExtendedInfo       *bool                    `json:"store-extended-info,omitempty"`
}

// Represents settable common configuration parameters for the DHCPv4 and DHCPv6 servers.
type SettableCommonDHCPConfig struct {
	SettableCacheParameters
	SettableDDNSParameters
	SettableReservationParameters
	SettableValidLifetimeParameters
	Allocator               *storkutil.Nullable[string]                          `json:"allocator,omitempty"`
	DHCPDDNS                *storkutil.Nullable[SettableDHCPDDNS]                `json:"dhcp-ddns,omitempty"`
	ExpiredLeasesProcessing *storkutil.Nullable[SettableExpiredLeasesProcessing] `json:"expired-leases-processing,omitempty"`
	OptionData              *storkutil.NullableArray[SingleOptionData]           `json:"option-data,omitempty"`
}

// Represents DHCP DDNS configuration parameters for the DHCPv4 and DHCPv6 servers.
type DHCPDDNS struct {
	EnableUpdates *bool   `json:"enable-updates,omitempty"`
	ServerIP      *string `json:"server-ip,omitempty"`
	ServerPort    *int64  `json:"server-port,omitempty"`
	SenderIP      *string `json:"sender-ip,omitempty"`
	SenderPort    *int64  `json:"sender-port,omitempty"`
	MaxQueueSize  *int64  `json:"max-queue-size,omitempty"`
	NCRProtocol   *string `json:"ncr-protocol,omitempty"`
	NCRFormat     *string `json:"ncr-format,omitempty"`
}

// Represents settable DHCP DDNS configuration parameters for the DHCPv4 and
// DHCPv6 servers.
type SettableDHCPDDNS struct {
	EnableUpdates *storkutil.Nullable[bool]   `json:"enable-updates,omitempty"`
	ServerIP      *storkutil.Nullable[string] `json:"server-ip,omitempty"`
	ServerPort    *storkutil.Nullable[int64]  `json:"server-port,omitempty"`
	SenderIP      *storkutil.Nullable[string] `json:"sender-ip,omitempty"`
	SenderPort    *storkutil.Nullable[int64]  `json:"sender-port,omitempty"`
	MaxQueueSize  *storkutil.Nullable[int64]  `json:"max-queue-size,omitempty"`
	NCRProtocol   *storkutil.Nullable[string] `json:"ncr-protocol,omitempty"`
	NCRFormat     *storkutil.Nullable[string] `json:"ncr-format,omitempty"`
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

// Represents the collection of settable settings pertaining to the expired
// leases processing.
type SettableExpiredLeasesProcessing struct {
	FlushReclaimedTimerWaitTime *storkutil.Nullable[int64] `json:"flush-reclaimed-timer-wait-time,omitempty"`
	HoldReclaimedTime           *storkutil.Nullable[int64] `json:"hold-reclaimed-time,omitempty"`
	MaxReclaimLeases            *storkutil.Nullable[int64] `json:"max-reclaim-leases,omitempty"`
	MaxReclaimTime              *storkutil.Nullable[int64] `json:"max-reclaim-time,omitempty"`
	ReclaimTimerWaitTime        *storkutil.Nullable[int64] `json:"reclaim-timer-wait-time,omitempty"`
	UnwarnedReclaimCycles       *storkutil.Nullable[int64] `json:"unwarned-reclaim-cycles,omitempty"`
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

// Finds a DHCPv4 subnet by prefix. It returns nil pointer if the subnet
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
func (c *SettableDHCPv4Config) SetAllocator(allocator *string) {
	c.Allocator = storkutil.NewNullable(allocator)
}

// Sets cache threshold.
func (c *SettableDHCPv4Config) SetCacheThreshold(cacheThreshold *float32) {
	c.CacheThreshold = storkutil.NewNullable(cacheThreshold)
}

// Sets boolean flag indicating if DDNS updates should be sent.
func (c *SettableDHCPv4Config) SetDDNSSendUpdates(ddnsSendUpdates *bool) {
	c.SettableCommonDHCPConfig.DDNSSendUpdates = storkutil.NewNullable(ddnsSendUpdates)
}

// Sets boolean flag indicating whether the DHCP server should override the
// client's wish to not update the DNS.
func (c *SettableDHCPv4Config) SetDDNSOverrideNoUpdate(ddnsOverrideNoUpdate *bool) {
	c.SettableCommonDHCPConfig.DDNSOverrideNoUpdate = storkutil.NewNullable(ddnsOverrideNoUpdate)
}

// Sets the boolean flag indicating whether the DHCP server should ignore the
// client's wish to update the DNS on its own.
func (c *SettableDHCPv4Config) SetDDNSOverrideClientUpdate(ddnsOverrideClientUpdate *bool) {
	c.SettableCommonDHCPConfig.DDNSOverrideClientUpdate = storkutil.NewNullable(ddnsOverrideClientUpdate)
}

// Sets the enumeration specifying whether the server should honor
// the hostname or Client FQDN sent by the client or replace this name.
func (c *SettableDHCPv4Config) SetDDNSReplaceClientName(ddnsReplaceClientName *string) {
	c.SettableCommonDHCPConfig.DDNSReplaceClientName = storkutil.NewNullable(ddnsReplaceClientName)
}

// Sets a prefix to be prepended to the generated Client FQDN.
func (c *SettableDHCPv4Config) SetDDNSGeneratedPrefix(ddnsGeneratedPrefix *string) {
	c.SettableCommonDHCPConfig.DDNSGeneratedPrefix = storkutil.NewNullable(ddnsGeneratedPrefix)
}

// Sets a suffix appended to the partial name sent to the DNS.
func (c *SettableDHCPv4Config) SetDDNSQualifyingSuffix(ddnsQualifyingSuffix *string) {
	c.SettableCommonDHCPConfig.DDNSQualifyingSuffix = storkutil.NewNullable(ddnsQualifyingSuffix)
}

// Sets a boolean flag, which when true instructs the server to always
// update DNS when leases are renewed, even if the DNS information
// has not changed.
func (c *SettableDHCPv4Config) SetDDNSUpdateOnRenew(ddnsUpdateOnRenew *bool) {
	c.SettableCommonDHCPConfig.DDNSUpdateOnRenew = storkutil.NewNullable(ddnsUpdateOnRenew)
}

// Sets a boolean flag which is passed to kea-dhcp-ddns with each DDNS
// update request, to indicate whether DNS update conflict
// resolution as described in RFC 4703 should be employed for the
// given update request.
func (c *SettableDHCPv4Config) SetDDNSUseConflictResolution(ddnsUseConflictResolution *bool) {
	c.SettableCommonDHCPConfig.DDNSUseConflictResolution = storkutil.NewNullable(ddnsUseConflictResolution)
}

// Sets a DDNS conflict resolution mode.
func (c *SettableDHCPv4Config) SetDDNSConflictResolutionMode(ddnsConflictResolutionMode *string) {
	c.SettableCommonDHCPConfig.DDNSConflictResolutionMode = storkutil.NewNullable(ddnsConflictResolutionMode)
}

// Sets the the percent of the lease's lifetime to use for the DNS TTL.
func (c *SettableDHCPv4Config) SetDDNSTTLPercent(ddnsTTLPercent *float32) {
	c.SettableCommonDHCPConfig.DDNSTTLPercent = storkutil.NewNullable(ddnsTTLPercent)
}

// Enables connectivity with the DHCP DDNS daemon and sending DNS updates.
func (c *SettableDHCPv4Config) SetDHCPDDNSEnableUpdates(enableUpdates *bool) {
	c.ensureDHCPDDNS().EnableUpdates = storkutil.NewNullable(enableUpdates)
}

// Sets the IP address on which D2 listens for requests.
func (c *SettableDHCPv4Config) SetDHCPDDNSServerIP(serverIP *string) {
	c.ensureDHCPDDNS().ServerIP = storkutil.NewNullable(serverIP)
}

// Sets the port on which D2 listens for requests.
func (c *SettableDHCPv4Config) SetDHCPDDNSServerPort(serverPort *int64) {
	c.ensureDHCPDDNS().ServerPort = storkutil.NewNullable(serverPort)
}

// Sets the IP address which DHCP server uses to send requests to D2.
func (c *SettableDHCPv4Config) SetDHCPDDNSSenderIP(senderIP *string) {
	c.ensureDHCPDDNS().SenderIP = storkutil.NewNullable(senderIP)
}

// Sets the port which DHCP server uses to send requests to D2.
func (c *SettableDHCPv4Config) SetDHCPDDNSSenderPort(senderPort *int64) {
	c.ensureDHCPDDNS().SenderPort = storkutil.NewNullable(senderPort)
}

// Sets the maximum number of requests allowed to queue while waiting to be sent to D2.
func (c *SettableDHCPv4Config) SetDHCPDDNSMaxQueueSize(maxQueueSize *int64) {
	c.ensureDHCPDDNS().MaxQueueSize = storkutil.NewNullable(maxQueueSize)
}

// Sets the socket protocol to use when sending requests to D2.
func (c *SettableDHCPv4Config) SetDHCPDDNSNCRProtocol(ncrProtocol *string) {
	c.ensureDHCPDDNS().NCRProtocol = storkutil.NewNullable(ncrProtocol)
}

// Sets the packet format to use when sending requests to D2.
func (c *SettableDHCPv4Config) SetDHCPDDNSNCRFormat(ncrFormat *string) {
	c.ensureDHCPDDNS().NCRFormat = storkutil.NewNullable(ncrFormat)
}

// Initializes the structure holding DHCP DDNS configuration.
func (c *SettableDHCPv4Config) ensureDHCPDDNS() *SettableDHCPDDNS {
	if c.DHCPDDNS == nil {
		c.DHCPDDNS = storkutil.NewNullable(&SettableDHCPDDNS{})
	}
	return c.DHCPDDNS.GetValue()
}

// Sets the DHCP DDNS structure.
func (c *SettableDHCPv4Config) SetDHCPDDNS(dhcpDDNS *SettableDHCPDDNS) {
	c.DHCPDDNS = storkutil.NewNullable(dhcpDDNS)
}

// Sets the number of seconds since the last removal of the expired
// leases, when the next removal should occur.
func (c *SettableDHCPv4Config) SetELPFlushReclaimedTimerWaitTime(flushReclaimedTimerWaitTime *int64) {
	c.ensureExpiredLeasesProcessing().FlushReclaimedTimerWaitTime = storkutil.NewNullable(flushReclaimedTimerWaitTime)
}

// Sets the length of time in seconds to keep expired leases in the
// lease database (lease affinity).
func (c *SettableDHCPv4Config) SetELPHoldReclaimedTime(holdReclaimedTime *int64) {
	c.ensureExpiredLeasesProcessing().HoldReclaimedTime = storkutil.NewNullable(holdReclaimedTime)
}

// Sets the maximum number of expired leases that can be processed in
// a single attempt to clean up expired leases from the lease database.
func (c *SettableDHCPv4Config) SetELPMaxReclaimLeases(maxReclaimedLeases *int64) {
	c.ensureExpiredLeasesProcessing().MaxReclaimLeases = storkutil.NewNullable(maxReclaimedLeases)
}

// Sets the maximum time in milliseconds that a single attempt to clean
// up expired leases from the lease database may take.
func (c *SettableDHCPv4Config) SetELPMaxReclaimTime(maxReclaimTime *int64) {
	c.ensureExpiredLeasesProcessing().MaxReclaimTime = storkutil.NewNullable(maxReclaimTime)
}

// Sets the length of time in seconds since the last attempt to process
// expired leases before initiating the next attempt.
func (c *SettableDHCPv4Config) SetELPReclaimTimerWaitTime(reclaimTimerWaitTime *int64) {
	c.ensureExpiredLeasesProcessing().ReclaimTimerWaitTime = storkutil.NewNullable(reclaimTimerWaitTime)
}

// Sets the maximum number of expired lease-processing cycles which didn't
// result in full cleanup of the expired leases from the lease database,
// after which a warning message is issued.
func (c *SettableDHCPv4Config) SetELPUnwarnedReclaimCycles(unwarnedReclaimCycles *int64) {
	c.ensureExpiredLeasesProcessing().UnwarnedReclaimCycles = storkutil.NewNullable(unwarnedReclaimCycles)
}

// Initializes the structure holding expired leases processing settings.
func (c *SettableDHCPv4Config) ensureExpiredLeasesProcessing() *SettableExpiredLeasesProcessing {
	if c.ExpiredLeasesProcessing == nil {
		c.ExpiredLeasesProcessing = storkutil.NewNullable(&SettableExpiredLeasesProcessing{})
	}
	return c.ExpiredLeasesProcessing.GetValue()
}

// Sets the expired leases processing structure.
func (c *SettableDHCPv4Config) SetExpiredLeasesProcessing(expiredLeasesProcessing *SettableExpiredLeasesProcessing) {
	c.ExpiredLeasesProcessing = storkutil.NewNullable(expiredLeasesProcessing)
}

// Sets the boolean flag enabling global reservations.
func (c *SettableDHCPv4Config) SetReservationsGlobal(reservationsGlobal *bool) {
	c.SettableReservationParameters.ReservationsGlobal = storkutil.NewNullable(reservationsGlobal)
}

// Sets the boolean flag enabling in-subnet reservations.
func (c *SettableDHCPv4Config) SetReservationsInSubnet(reservationsInSubnet *bool) {
	c.SettableReservationParameters.ReservationsInSubnet = storkutil.NewNullable(reservationsInSubnet)
}

// Sets the boolean flag enabling out-of-pool reservations.
func (c *SettableDHCPv4Config) SetReservationsOutOfPool(reservationsOutOfPool *bool) {
	c.SettableReservationParameters.ReservationsOutOfPool = storkutil.NewNullable(reservationsOutOfPool)
}

// Sets a boolean flag enabling an early global reservations lookup.
func (c *SettableDHCPv4Config) SetEarlyGlobalReservationsLookup(earlyGlobalReservationsLookup *bool) {
	c.SettableReservationParameters.EarlyGlobalReservationsLookup = storkutil.NewNullable(earlyGlobalReservationsLookup)
}

// Sets host reservation identifiers to be used for host reservation lookup.
func (c *SettableDHCPv4Config) SetHostReservationIdentifiers(hostReservationIdentifiers []string) {
	c.SettableReservationParameters.HostReservationIdentifiers = storkutil.NewNullableArray(hostReservationIdentifiers)
}

// Sets global valid lifetime.
func (c *SettableDHCPv4Config) SetValidLifetime(validLifetime *int64) {
	c.SettableValidLifetimeParameters.ValidLifetime = storkutil.NewNullable(validLifetime)
}

// Sets global DHCP option data.
func (c *SettableDHCPv4Config) SetDHCPOptions(options []SingleOptionData) {
	c.OptionData = storkutil.NewNullableArray(options)
}

// Sets a boolean flag indicating whether the server is authoritative.
func (c *SettableDHCPv4Config) SetAuthoritative(authoritative *bool) {
	c.Authoritative = storkutil.NewNullable(authoritative)
}

// Sets a boolean flag indicating whether the server should return client
// ID in its responses.
func (c *SettableDHCPv4Config) SetEchoClientID(echoClientID *bool) {
	c.EchoClientID = storkutil.NewNullable(echoClientID)
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

// Finds a DHCPv6 subnet by prefix. It returns nil pointer if the subnet
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
func (c *SettableDHCPv6Config) SetAllocator(allocator *string) {
	c.Allocator = storkutil.NewNullable(allocator)
}

// Sets cache threshold.
func (c *SettableDHCPv6Config) SetCacheThreshold(cacheThreshold *float32) {
	c.CacheThreshold = storkutil.NewNullable(cacheThreshold)
}

// Sets boolean flag indicating if DDNS updates should be sent.
func (c *SettableDHCPv6Config) SetDDNSSendUpdates(ddnsSendUpdates *bool) {
	c.SettableCommonDHCPConfig.DDNSSendUpdates = storkutil.NewNullable(ddnsSendUpdates)
}

// Sets boolean flag indicating whether the DHCP server should override the
// client's wish to not update the DNS.
func (c *SettableDHCPv6Config) SetDDNSOverrideNoUpdate(ddnsOverrideNoUpdate *bool) {
	c.SettableCommonDHCPConfig.DDNSOverrideNoUpdate = storkutil.NewNullable(ddnsOverrideNoUpdate)
}

// Sets the boolean flag indicating whether the DHCP server should ignore the
// client's wish to update the DNS on its own.
func (c *SettableDHCPv6Config) SetDDNSOverrideClientUpdate(ddnsOverrideClientUpdate *bool) {
	c.SettableCommonDHCPConfig.DDNSOverrideClientUpdate = storkutil.NewNullable(ddnsOverrideClientUpdate)
}

// Sets the enumeration specifying whether the server should honor
// the hostname or Client FQDN sent by the client or replace this name.
func (c *SettableDHCPv6Config) SetDDNSReplaceClientName(ddnsReplaceClientName *string) {
	c.SettableCommonDHCPConfig.DDNSReplaceClientName = storkutil.NewNullable(ddnsReplaceClientName)
}

// Sets a prefix to be prepended to the generated Client FQDN.
func (c *SettableDHCPv6Config) SetDDNSGeneratedPrefix(ddnsGeneratedPrefix *string) {
	c.SettableCommonDHCPConfig.DDNSGeneratedPrefix = storkutil.NewNullable(ddnsGeneratedPrefix)
}

// Sets a suffix appended to the partial name sent to the DNS.
func (c *SettableDHCPv6Config) SetDDNSQualifyingSuffix(ddnsQualifyingSuffix *string) {
	c.SettableCommonDHCPConfig.DDNSQualifyingSuffix = storkutil.NewNullable(ddnsQualifyingSuffix)
}

// Sets a boolean flag, which when true instructs the server to always
// update DNS when leases are renewed, even if the DNS information
// has not changed.
func (c *SettableDHCPv6Config) SetDDNSUpdateOnRenew(ddnsUpdateOnRenew *bool) {
	c.SettableCommonDHCPConfig.DDNSUpdateOnRenew = storkutil.NewNullable(ddnsUpdateOnRenew)
}

// Sets a boolean flag which is passed to kea-dhcp-ddns with each DDNS
// update request, to indicate whether DNS update conflict
// resolution as described in RFC 4703 should be employed for the
// given update request.
func (c *SettableDHCPv6Config) SetDDNSUseConflictResolution(ddnsUseConflictResolution *bool) {
	c.SettableCommonDHCPConfig.DDNSUseConflictResolution = storkutil.NewNullable(ddnsUseConflictResolution)
}

// Sets a DDNS conflict resolution mode.
func (c *SettableDHCPv6Config) SetDDNSConflictResolutionMode(ddnsConflictResolutionMode *string) {
	c.SettableCommonDHCPConfig.DDNSConflictResolutionMode = storkutil.NewNullable(ddnsConflictResolutionMode)
}

// Sets the the percent of the lease's lifetime to use for the DNS TTL.
func (c *SettableDHCPv6Config) SetDDNSTTLPercent(ddnsTTLPercent *float32) {
	c.SettableCommonDHCPConfig.DDNSTTLPercent = storkutil.NewNullable(ddnsTTLPercent)
}

// Enables connectivity with the DHCP DDNS daemon and sending DNS updates.
func (c *SettableDHCPv6Config) SetDHCPDDNSEnableUpdates(enableUpdates *bool) {
	c.ensureDHCPDDNS().EnableUpdates = storkutil.NewNullable(enableUpdates)
}

// Sets the IP address on which D2 listens for requests.
func (c *SettableDHCPv6Config) SetDHCPDDNSServerIP(serverIP *string) {
	c.ensureDHCPDDNS().ServerIP = storkutil.NewNullable(serverIP)
}

// Sets the port on which D2 listens for requests.
func (c *SettableDHCPv6Config) SetDHCPDDNSServerPort(serverPort *int64) {
	c.ensureDHCPDDNS().ServerPort = storkutil.NewNullable(serverPort)
}

// Sets the IP address which DHCP server uses to send requests to D2.
func (c *SettableDHCPv6Config) SetDHCPDDNSSenderIP(senderIP *string) {
	c.ensureDHCPDDNS().SenderIP = storkutil.NewNullable(senderIP)
}

// Sets the port which DHCP server uses to send requests to D2.
func (c *SettableDHCPv6Config) SetDHCPDDNSSenderPort(senderPort *int64) {
	c.ensureDHCPDDNS().SenderPort = storkutil.NewNullable(senderPort)
}

// Sets the maximum number of requests allowed to queue while waiting to be sent to D2.
func (c *SettableDHCPv6Config) SetDHCPDDNSMaxQueueSize(maxQueueSize *int64) {
	c.ensureDHCPDDNS().MaxQueueSize = storkutil.NewNullable(maxQueueSize)
}

// Sets the socket protocol to use when sending requests to D2.
func (c *SettableDHCPv6Config) SetDHCPDDNSNCRProtocol(ncrProtocol *string) {
	c.ensureDHCPDDNS().NCRProtocol = storkutil.NewNullable(ncrProtocol)
}

// Sets the packet format to use when sending requests to D2.
func (c *SettableDHCPv6Config) SetDHCPDDNSNCRFormat(ncrFormat *string) {
	c.ensureDHCPDDNS().NCRFormat = storkutil.NewNullable(ncrFormat)
}

// Initializes the structure holding DHCP DDNS configuration.
func (c *SettableDHCPv6Config) ensureDHCPDDNS() *SettableDHCPDDNS {
	if c.DHCPDDNS == nil {
		c.DHCPDDNS = storkutil.NewNullable(&SettableDHCPDDNS{})
	}
	return c.DHCPDDNS.GetValue()
}

// Sets the DHCP DDNS structure.
func (c *SettableDHCPv6Config) SetDHCPDDNS(dhcpDDNS *SettableDHCPDDNS) {
	c.DHCPDDNS = storkutil.NewNullable(dhcpDDNS)
}

// Sets the number of seconds since the last removal of the expired
// leases, when the next removal should occur.
func (c *SettableDHCPv6Config) SetELPFlushReclaimedTimerWaitTime(flushReclaimedTimerWaitTime *int64) {
	c.ensureExpiredLeasesProcessing().FlushReclaimedTimerWaitTime = storkutil.NewNullable(flushReclaimedTimerWaitTime)
}

// Sets the length of time in seconds to keep expired leases in the
// lease database (lease affinity).
func (c *SettableDHCPv6Config) SetELPHoldReclaimedTime(holdReclaimedTime *int64) {
	c.ensureExpiredLeasesProcessing().HoldReclaimedTime = storkutil.NewNullable(holdReclaimedTime)
}

// Sets the maximum number of expired leases that can be processed in
// a single attempt to clean up expired leases from the lease database.
func (c *SettableDHCPv6Config) SetELPMaxReclaimLeases(maxReclaimedLeases *int64) {
	c.ensureExpiredLeasesProcessing().MaxReclaimLeases = storkutil.NewNullable(maxReclaimedLeases)
}

// Sets the maximum time in milliseconds that a single attempt to clean
// up expired leases from the lease database may take.
func (c *SettableDHCPv6Config) SetELPMaxReclaimTime(maxReclaimTime *int64) {
	c.ensureExpiredLeasesProcessing().MaxReclaimTime = storkutil.NewNullable(maxReclaimTime)
}

// Sets the length of time in seconds since the last attempt to process
// expired leases before initiating the next attempt.
func (c *SettableDHCPv6Config) SetELPReclaimTimerWaitTime(reclaimTimerWaitTime *int64) {
	c.ensureExpiredLeasesProcessing().ReclaimTimerWaitTime = storkutil.NewNullable(reclaimTimerWaitTime)
}

// Sets the maximum number of expired lease-processing cycles which didn't
// result in full cleanup of the expired leases from the lease database,
// after which a warning message is issued.
func (c *SettableDHCPv6Config) SetELPUnwarnedReclaimCycles(unwarnedReclaimCycles *int64) {
	c.ensureExpiredLeasesProcessing().UnwarnedReclaimCycles = storkutil.NewNullable(unwarnedReclaimCycles)
}

// Initializes the structure holding expired leases processing settings.
func (c *SettableDHCPv6Config) ensureExpiredLeasesProcessing() *SettableExpiredLeasesProcessing {
	if c.ExpiredLeasesProcessing == nil {
		c.ExpiredLeasesProcessing = storkutil.NewNullable(&SettableExpiredLeasesProcessing{})
	}
	return c.ExpiredLeasesProcessing.GetValue()
}

// Sets the expired leases processing structure.
func (c *SettableDHCPv6Config) SetExpiredLeasesProcessing(expiredLeasesProcessing *SettableExpiredLeasesProcessing) {
	c.ExpiredLeasesProcessing = storkutil.NewNullable(expiredLeasesProcessing)
}

// Sets the boolean flag enabling global reservations.
func (c *SettableDHCPv6Config) SetReservationsGlobal(reservationsGlobal *bool) {
	c.SettableReservationParameters.ReservationsGlobal = storkutil.NewNullable(reservationsGlobal)
}

// Sets the boolean flag enabling in-subnet reservations.
func (c *SettableDHCPv6Config) SetReservationsInSubnet(reservationsInSubnet *bool) {
	c.SettableReservationParameters.ReservationsInSubnet = storkutil.NewNullable(reservationsInSubnet)
}

// Sets the boolean flag enabling out-of-pool reservations.
func (c *SettableDHCPv6Config) SetReservationsOutOfPool(reservationsOutOfPool *bool) {
	c.SettableReservationParameters.ReservationsOutOfPool = storkutil.NewNullable(reservationsOutOfPool)
}

// Sets a boolean flag enabling an early global reservations lookup.
func (c *SettableDHCPv6Config) SetEarlyGlobalReservationsLookup(earlyGlobalReservationsLookup *bool) {
	c.SettableReservationParameters.EarlyGlobalReservationsLookup = storkutil.NewNullable(earlyGlobalReservationsLookup)
}

// Sets host reservation identifiers to be used for host reservation lookup.
func (c *SettableDHCPv6Config) SetHostReservationIdentifiers(hostReservationIdentifiers []string) {
	c.SettableReservationParameters.HostReservationIdentifiers = storkutil.NewNullableArray(hostReservationIdentifiers)
}

// Sets global valid lifetime.
func (c *SettableDHCPv6Config) SetValidLifetime(validLifetime *int64) {
	c.SettableValidLifetimeParameters.ValidLifetime = storkutil.NewNullable(validLifetime)
}

// Sets global DHCP option data.
func (c *SettableDHCPv6Config) SetDHCPOptions(options []SingleOptionData) {
	c.OptionData = storkutil.NewNullableArray(options)
}

// Sets allocator for prefix delegation.
func (c *SettableDHCPv6Config) SetPDAllocator(pdAllocator *string) {
	c.PDAllocator = storkutil.NewNullable(pdAllocator)
}
