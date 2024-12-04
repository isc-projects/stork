package keaconfig

import (
	"fmt"

	"github.com/pkg/errors"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

var (
	_ Subnet = (*Subnet4)(nil)
	_ Subnet = (*Subnet6)(nil)
)

// An interface representing a subnet in Stork, extended with Kea-specific
// subnet DHCP configuration.
type SubnetAccessor interface {
	dhcpmodel.SubnetAccessor
	GetID(int64) int64
	GetKeaParameters(int64) *SubnetParameters
}

// An interface representing a subnet in Kea. It is implemented by the
// Subnet4 and Subnet6. This interface is used in functions that
// identically process the DHCPv4 and DHCPv6 subnets. Their actual
// type is hidden behind this interface.
type Subnet interface {
	GetID() int64
	GetPrefix() string
	GetCanonicalPrefix() (string, error)
	GetPools() []Pool
	GetPDPools() []PDPool
	GetReservations() []Reservation
	GetSubnetParameters() *SubnetParameters
	GetDHCPOptions() []SingleOptionData
	GetUniverse() storkutil.IPType
	GetUserContext() map[string]any
}

// Represents a relay configuration for a subnet in Kea.
type Relay struct {
	IPAddresses []string `json:"ip-addresses"`
}

// Represents DHCPv4-over-DHCPv6 subnet configuration parameters in Kea.
type FourOverSixParameters struct {
	FourOverSixInterface   *string `json:"4o6-interface,omitempty"`
	FourOverSixInterfaceID *string `json:"4o6-interface-id,omitempty"`
	FourOverSixSubnet      *string `json:"4o6-subnet,omitempty"`
}

// Represents DDNS configuration parameters in Kea.
type DDNSParameters struct {
	DDNSGeneratedPrefix        *string  `json:"ddns-generated-prefix,omitempty"`
	DDNSOverrideClientUpdate   *bool    `json:"ddns-override-client-update,omitempty"`
	DDNSOverrideNoUpdate       *bool    `json:"ddns-override-no-update,omitempty"`
	DDNSQualifyingSuffix       *string  `json:"ddns-qualifying-suffix,omitempty"`
	DDNSReplaceClientName      *string  `json:"ddns-replace-client-name,omitempty"`
	DDNSSendUpdates            *bool    `json:"ddns-send-updates,omitempty"`
	DDNSUpdateOnRenew          *bool    `json:"ddns-update-on-renew,omitempty"`
	DDNSUseConflictResolution  *bool    `json:"ddns-use-conflict-resolution,omitempty"`
	DDNSConflictResolutionMode *string  `json:"ddns-conflict-resolution-mode,omitempty"`
	DDNSTTLPercent             *float32 `json:"ddns-ttl-percent,omitempty"`
}

// Represents DDNS configuration parameters in Kea.
type SettableDDNSParameters struct {
	DDNSGeneratedPrefix        *storkutil.Nullable[string]  `json:"ddns-generated-prefix,omitempty"`
	DDNSOverrideClientUpdate   *storkutil.Nullable[bool]    `json:"ddns-override-client-update,omitempty"`
	DDNSOverrideNoUpdate       *storkutil.Nullable[bool]    `json:"ddns-override-no-update,omitempty"`
	DDNSQualifyingSuffix       *storkutil.Nullable[string]  `json:"ddns-qualifying-suffix,omitempty"`
	DDNSReplaceClientName      *storkutil.Nullable[string]  `json:"ddns-replace-client-name,omitempty"`
	DDNSSendUpdates            *storkutil.Nullable[bool]    `json:"ddns-send-updates,omitempty"`
	DDNSUpdateOnRenew          *storkutil.Nullable[bool]    `json:"ddns-update-on-renew,omitempty"`
	DDNSUseConflictResolution  *storkutil.Nullable[bool]    `json:"ddns-use-conflict-resolution,omitempty"`
	DDNSConflictResolutionMode *storkutil.Nullable[string]  `json:"ddns-conflict-resolution-mode,omitempty"`
	DDNSTTLPercent             *storkutil.Nullable[float32] `json:"ddns-ttl-percent,omitempty"`
}

// Represents Kea configuration parameters for hostname manipulation.
type HostnameCharParameters struct {
	HostnameCharReplacement *string `json:"hostname-char-replacement,omitempty"`
	HostnameCharSet         *string `json:"hostname-char-set,omitempty"`
}

// Represents Kea configuration parameters for selecting host reservation modes.
type ReservationParameters struct {
	EarlyGlobalReservationsLookup *bool    `json:"early-global-reservations-lookup,omitempty"`
	HostReservationIdentifiers    []string `json:"host-reservation-identifiers,omitempty"`
	ReservationMode               *string  `json:"reservation-mode,omitempty"`
	ReservationsGlobal            *bool    `json:"reservations-global,omitempty"`
	ReservationsInSubnet          *bool    `json:"reservations-in-subnet,omitempty"`
	ReservationsOutOfPool         *bool    `json:"reservations-out-of-pool,omitempty"`
}

// Represents settable Kea configuration parameters for selecting host reservation
// modes.
type SettableReservationParameters struct {
	EarlyGlobalReservationsLookup *storkutil.Nullable[bool]        `json:"early-global-reservations-lookup,omitempty"`
	HostReservationIdentifiers    *storkutil.NullableArray[string] `json:"host-reservation-identifiers,omitempty"`
	ReservationMode               *storkutil.Nullable[string]      `json:"reservation-mode,omitempty"`
	ReservationsGlobal            *storkutil.Nullable[bool]        `json:"reservations-global,omitempty"`
	ReservationsInSubnet          *storkutil.Nullable[bool]        `json:"reservations-in-subnet,omitempty"`
	ReservationsOutOfPool         *storkutil.Nullable[bool]        `json:"reservations-out-of-pool,omitempty"`
}

// Represents DHCP timer configuration parameters in Kea.
type TimerParameters struct {
	RenewTimer        *int64   `json:"renew-timer,omitempty"`
	RebindTimer       *int64   `json:"rebind-timer,omitempty"`
	T1Percent         *float32 `json:"t1-percent,omitempty"`
	T2Percent         *float32 `json:"t2-percent,omitempty"`
	CalculateTeeTimes *bool    `json:"calculate-tee-times,omitempty"`
}

// Represents DHCP lease caching parameters in Kea.
type CacheParameters struct {
	CacheThreshold *float32 `json:"cache-threshold,omitempty"`
	CacheMaxAge    *int64   `json:"cache-max-age,omitempty"`
}

// Represents settable DHCP lease caching parameters in Kea.
type SettableCacheParameters struct {
	CacheThreshold *storkutil.Nullable[float32] `json:"cache-threshold,omitempty"`
}

// Represents client-class configuration in Kea.
type ClientClassParameters struct {
	ClientClass          *string  `json:"client-class,omitempty"`
	RequireClientClasses []string `json:"require-client-classes,omitempty"`
}

// Represents valid lifetime configuration parameters in Kea.
type ValidLifetimeParameters struct {
	ValidLifetime    *int64 `json:"valid-lifetime,omitempty"`
	MinValidLifetime *int64 `json:"min-valid-lifetime,omitempty"`
	MaxValidLifetime *int64 `json:"max-valid-lifetime,omitempty"`
}

// Represents settable valid lifetime configuration parameters in Kea.
type SettableValidLifetimeParameters struct {
	ValidLifetime    *storkutil.Nullable[int64] `json:"valid-lifetime,omitempty"`
	MinValidLifetime *storkutil.Nullable[int64] `json:"min-valid-lifetime,omitempty"`
	MaxValidLifetime *storkutil.Nullable[int64] `json:"max-valid-lifetime,omitempty"`
}

// Represents preferred lifetime configuration parameters in Kea.
type PreferredLifetimeParameters struct {
	PreferredLifetime    *int64 `json:"preferred-lifetime,omitempty"`
	MinPreferredLifetime *int64 `json:"min-preferred-lifetime,omitempty"`
	MaxPreferredLifetime *int64 `json:"max-preferred-lifetime,omitempty"`
}

// Represents mandatory subnet configuration parameters in Kea.
// Note that ID can be left unspecified by the user. In this case
// it will be auto-generated. So, mandatory means that it is always
// present in the Kea runtime configuration.
type MandatorySubnetParameters struct {
	ID     int64  `json:"id"`
	Subnet string `json:"subnet"`
}

// Represents Kea subnet parameter groups supported by both DHCPv4
// and DHCPv6 servers.
type CommonSubnetParameters struct {
	CacheParameters
	ClientClassParameters
	DDNSParameters
	HostnameCharParameters
	ReservationParameters
	TimerParameters
	ValidLifetimeParameters
	Allocator         *string            `json:"allocator,omitempty"`
	Interface         *string            `json:"interface,omitempty"`
	StoreExtendedInfo *bool              `json:"store-extended-info,omitempty"`
	OptionData        []SingleOptionData `json:"option-data,omitempty"`
	Pools             []Pool             `json:"pools,omitempty"`
	Relay             *Relay             `json:"relay,omitempty"`
	Reservations      []Reservation      `json:"reservations,omitempty"`
	UserContext       map[string]any     `json:"user-context,omitempty"`
}

// Represents an IPv4 subnet in Kea.
type Subnet4 struct {
	CommonSubnetParameters
	FourOverSixParameters
	MandatorySubnetParameters
	Authoritative  *bool   `json:"authoritative,omitempty"`
	BootFileName   *string `json:"boot-file-name,omitempty"`
	MatchClientID  *bool   `json:"match-client-id,omitempty"`
	NextServer     *string `json:"next-server,omitempty"`
	ServerHostname *string `json:"server-hostname,omitempty"`
}

// Represents an IPv6 subnet in Kea.
type Subnet6 struct {
	CommonSubnetParameters
	MandatorySubnetParameters
	PreferredLifetimeParameters
	PDAllocator *string  `json:"pd-allocator,omitempty"`
	InterfaceID *string  `json:"interface-id,omitempty"`
	PDPools     []PDPool `json:"pd-pools,omitempty"`
	RapidCommit *bool    `json:"rapid-commit,omitempty"`
}

// Represents a union of DHCP parameters for the DHCPv4 and
// the DHCPv6 servers. Such a union can be used in the
// Stork database model to hold DHCPv4 and DHCPv6 subnets in
// a common structure (see the dbmodel package).
type SubnetParameters struct {
	CacheParameters
	ClientClassParameters
	DDNSParameters
	FourOverSixParameters
	HostnameCharParameters
	PreferredLifetimeParameters
	ReservationParameters
	TimerParameters
	ValidLifetimeParameters
	Allocator         *string
	Authoritative     *bool
	BootFileName      *string
	Interface         *string
	InterfaceID       *string
	MatchClientID     *bool
	NextServer        *string
	PDAllocator       *string
	RapidCommit       *bool
	Relay             *Relay
	ServerHostname    *string
	StoreExtendedInfo *bool
}

// Represents deleted subnet. It includes the fields required by Kea to
// find the reservation and delete it.
type SubnetCmdsDeletedSubnet struct {
	ID int64 `json:"id"`
}

// Returns a subnet ID.
func (s MandatorySubnetParameters) GetID() int64 {
	return s.ID
}

// Returns the subnet prefix in the format received from Kea.
func (s MandatorySubnetParameters) GetPrefix() string {
	return s.Subnet
}

// Returns a canonical subnet prefix or an error if the prefix is
// invalid.
func (s MandatorySubnetParameters) GetCanonicalPrefix() (string, error) {
	cidr := storkutil.ParseIP(s.Subnet)
	if cidr == nil {
		return "", errors.Errorf("invalid subnet prefix: %s", s.Subnet)
	}
	return cidr.GetNetworkPrefixWithLength(), nil
}

// Returns subnet ID.
func (s *Subnet4) GetID() int64 {
	return s.MandatorySubnetParameters.GetID()
}

// Returns the subnet prefix in the format received from Kea.
func (s *Subnet4) GetPrefix() string {
	return s.Subnet
}

// Returns a canonical IPv4 subnet prefix.
func (s *Subnet4) GetCanonicalPrefix() (string, error) {
	return s.MandatorySubnetParameters.GetCanonicalPrefix()
}

// Returns subnet pools.
func (s *Subnet4) GetPools() []Pool {
	return s.Pools
}

// Returns an empty list of delegated prefix pools. Such pools do not
// exist in the DHCPv4 subnets.
func (s *Subnet4) GetPDPools() []PDPool {
	return []PDPool{}
}

// Returns subnet host reservations.
func (s *Subnet4) GetReservations() []Reservation {
	return s.Reservations
}

// Returns DHCP options.
func (s *Subnet4) GetDHCPOptions() []SingleOptionData {
	return s.OptionData
}

// Returns IPv4 universe.
func (s *Subnet4) GetUniverse() storkutil.IPType {
	return storkutil.IPv4
}

// Returns user-context data.
func (s *Subnet4) GetUserContext() map[string]any {
	return s.UserContext
}

// Returns Kea-specific IPv4 subnet configuration parameters.
func (s *Subnet4) GetSubnetParameters() *SubnetParameters {
	return &SubnetParameters{
		CacheParameters:         s.CacheParameters,
		ClientClassParameters:   s.ClientClassParameters,
		DDNSParameters:          s.DDNSParameters,
		FourOverSixParameters:   s.FourOverSixParameters,
		HostnameCharParameters:  s.HostnameCharParameters,
		Relay:                   s.Relay,
		ReservationParameters:   s.ReservationParameters,
		TimerParameters:         s.TimerParameters,
		ValidLifetimeParameters: s.ValidLifetimeParameters,
		Allocator:               s.Allocator,
		Authoritative:           s.Authoritative,
		BootFileName:            s.BootFileName,
		Interface:               s.Interface,
		MatchClientID:           s.MatchClientID,
		NextServer:              s.NextServer,
		ServerHostname:          s.ServerHostname,
		StoreExtendedInfo:       s.StoreExtendedInfo,
	}
}

// Returns subnet ID.
func (s *Subnet6) GetID() int64 {
	return s.MandatorySubnetParameters.GetID()
}

// Returns the subnet prefix in the format received from Kea.
func (s *Subnet6) GetPrefix() string {
	return s.Subnet
}

// Returns a canonical IPv6 subnet prefix.
func (s *Subnet6) GetCanonicalPrefix() (string, error) {
	return s.MandatorySubnetParameters.GetCanonicalPrefix()
}

// Returns subnet pools.
func (s *Subnet6) GetPools() []Pool {
	return s.Pools
}

// Returns delegated prefix pools for the subnet.
func (s *Subnet6) GetPDPools() []PDPool {
	return s.PDPools
}

// Returns subnet host reservations.
func (s *Subnet6) GetReservations() []Reservation {
	return s.Reservations
}

// Returns DHCP options.
func (s *Subnet6) GetDHCPOptions() []SingleOptionData {
	return s.OptionData
}

// Returns IPv6 universe.
func (s *Subnet6) GetUniverse() storkutil.IPType {
	return storkutil.IPv6
}

// Returns user-context data.
func (s *Subnet6) GetUserContext() map[string]any {
	return s.UserContext
}

// Returns Kea-specific IPv6 subnet configuration parameters.
func (s *Subnet6) GetSubnetParameters() *SubnetParameters {
	return &SubnetParameters{
		CacheParameters:             s.CacheParameters,
		ClientClassParameters:       s.ClientClassParameters,
		DDNSParameters:              s.DDNSParameters,
		HostnameCharParameters:      s.HostnameCharParameters,
		PreferredLifetimeParameters: s.PreferredLifetimeParameters,
		Relay:                       s.Relay,
		ReservationParameters:       s.ReservationParameters,
		TimerParameters:             s.TimerParameters,
		ValidLifetimeParameters:     s.ValidLifetimeParameters,
		Allocator:                   s.Allocator,
		Interface:                   s.Interface,
		InterfaceID:                 s.InterfaceID,
		PDAllocator:                 s.PDAllocator,
		RapidCommit:                 s.RapidCommit,
		StoreExtendedInfo:           s.StoreExtendedInfo,
	}
}

// Creates an IPv4 subnet configuration in Kea from the subnet data model in Stork.
// The daemonID parameter is used to identify a daemon in the Stork subnet whose
// configuration should be converted to the Kea format. The lookup is an interface
// returning definitions for the converted DHCP options. Finally, the subnet is the
// interface representing a subnet data model in Stork (e.g., dbmodel.Subnet should
// implement this interface).
func CreateSubnet4(daemonID int64, lookup DHCPOptionDefinitionLookup, subnet SubnetAccessor) (*Subnet4, error) {
	// Mandatory parameters.
	subnet4 := &Subnet4{
		MandatorySubnetParameters: MandatorySubnetParameters{
			ID:     subnet.GetID(daemonID),
			Subnet: subnet.GetPrefix(),
		},
		CommonSubnetParameters: CommonSubnetParameters{
			UserContext: subnet.GetUserContext(daemonID),
		},
	}
	// Address pools.
	for _, pool := range subnet.GetAddressPools(daemonID) {
		keaPool := Pool{
			Pool: fmt.Sprintf("%s-%s", pool.GetLowerBound(), pool.GetUpperBound()),
		}
		// Pool-level DHCP options.
		for _, option := range pool.GetDHCPOptions() {
			optionData, err := CreateSingleOptionData(daemonID, lookup, option)
			if err != nil {
				return nil, err
			}
			keaPool.OptionData = append(keaPool.OptionData, *optionData)
		}
		// Pool-level Kea-specific parameters.
		keaPoolAccessor := pool.(AddressPool)
		params := keaPoolAccessor.GetKeaParameters()
		if params != nil {
			if params.ClientClass != nil {
				keaPool.ClientClass = params.ClientClass
			}
			keaPool.RequireClientClasses = params.RequireClientClasses
			keaPool.PoolID = params.PoolID
		}
		// Add the pool to the subnet.
		subnet4.Pools = append(subnet4.Pools, keaPool)
	}
	// Subnet-level Kea-specific parameters.
	if params := subnet.GetKeaParameters(daemonID); params != nil {
		subnet4.CommonSubnetParameters.CacheParameters = params.CacheParameters
		subnet4.CommonSubnetParameters.ClientClassParameters = params.ClientClassParameters
		subnet4.CommonSubnetParameters.DDNSParameters = params.DDNSParameters
		subnet4.CommonSubnetParameters.HostnameCharParameters = params.HostnameCharParameters
		subnet4.CommonSubnetParameters.ReservationParameters = params.ReservationParameters
		subnet4.CommonSubnetParameters.TimerParameters = params.TimerParameters
		subnet4.CommonSubnetParameters.ValidLifetimeParameters = params.ValidLifetimeParameters
		subnet4.CommonSubnetParameters.Allocator = params.Allocator
		subnet4.CommonSubnetParameters.Interface = params.Interface
		subnet4.CommonSubnetParameters.StoreExtendedInfo = params.StoreExtendedInfo
		subnet4.CommonSubnetParameters.Relay = params.Relay
		subnet4.FourOverSixParameters = params.FourOverSixParameters
		subnet4.Authoritative = params.Authoritative
		subnet4.BootFileName = params.BootFileName
		subnet4.MatchClientID = params.MatchClientID
		subnet4.NextServer = params.NextServer
		subnet4.ServerHostname = params.ServerHostname
	}
	// Subnet-level DHCP options.
	for _, option := range subnet.GetDHCPOptions(daemonID) {
		optionData, err := CreateSingleOptionData(daemonID, lookup, option)
		if err != nil {
			return nil, err
		}
		subnet4.OptionData = append(subnet4.OptionData, *optionData)
	}
	return subnet4, nil
}

// Creates an IPv6 subnet configuration in Kea from the subnet data model in Stork.
// The daemonID parameter is used to identify a daemon in the Stork subnet whose
// configuration should be converted to the Kea format. The lookup is an interface
// returning definitions for the converted DHCP options. Finally, the subnet is the
// interface representing a subnet data model in Stork (e.g., dbmodel.Subnet should
// implement this interface).
func CreateSubnet6(daemonID int64, lookup DHCPOptionDefinitionLookup, subnet SubnetAccessor) (*Subnet6, error) {
	subnet6 := &Subnet6{
		MandatorySubnetParameters: MandatorySubnetParameters{
			ID:     subnet.GetID(daemonID),
			Subnet: subnet.GetPrefix(),
		},
		CommonSubnetParameters: CommonSubnetParameters{
			UserContext: subnet.GetUserContext(daemonID),
		},
	}
	// Address pools.
	for _, pool := range subnet.GetAddressPools(daemonID) {
		keaPool := Pool{
			Pool: fmt.Sprintf("%s-%s", pool.GetLowerBound(), pool.GetUpperBound()),
		}
		// Pool-level DHCP options.
		for _, option := range pool.GetDHCPOptions() {
			optionData, err := CreateSingleOptionData(daemonID, lookup, option)
			if err != nil {
				return nil, err
			}
			keaPool.OptionData = append(keaPool.OptionData, *optionData)
		}
		// Pool-level Kea-specific parameters.
		keaPoolAccessor := pool.(AddressPool)
		params := keaPoolAccessor.GetKeaParameters()
		if params != nil {
			if params.ClientClass != nil {
				keaPool.ClientClass = params.ClientClass
			}
			keaPool.RequireClientClasses = params.RequireClientClasses
			keaPool.PoolID = params.PoolID
		}
		// Add the pool to the subnet.
		subnet6.Pools = append(subnet6.Pools, keaPool)
	}
	// Delegated prefix pools.
	for _, pool := range subnet.GetPrefixPools(daemonID) {
		// Pool prefix.
		prefix, length, err := pool.GetModel().GetPrefix()
		if err != nil {
			return nil, err
		}
		// Excluded prefix.
		excludedPrefix, excludedPrefixLength, err := pool.GetModel().GetExcludedPrefix()
		if err != nil {
			return nil, err
		}
		keaPool := PDPool{
			Prefix:            prefix,
			PrefixLen:         length,
			DelegatedLen:      pool.GetModel().DelegatedLen,
			ExcludedPrefix:    excludedPrefix,
			ExcludedPrefixLen: excludedPrefixLength,
		}
		// Pool-level DHCP options.
		for _, option := range pool.GetDHCPOptions() {
			optionData, err := CreateSingleOptionData(daemonID, lookup, option)
			if err != nil {
				return nil, err
			}
			keaPool.OptionData = append(keaPool.OptionData, *optionData)
		}
		// Pool-level Kea-specific parameters.
		keaPoolAccessor := pool.(PrefixPool)
		params := keaPoolAccessor.GetKeaParameters()
		if params != nil {
			if params.ClientClass != nil {
				keaPool.ClientClass = params.ClientClass
			}
			keaPool.RequireClientClasses = params.RequireClientClasses
			keaPool.PoolID = params.PoolID
		}
		// Add the pool to the subnet.
		subnet6.PDPools = append(subnet6.PDPools, keaPool)
	}
	// Subnet-level Kea-specific parameters.
	if params := subnet.GetKeaParameters(daemonID); params != nil {
		subnet6.CommonSubnetParameters.CacheParameters = params.CacheParameters
		subnet6.CommonSubnetParameters.ClientClassParameters = params.ClientClassParameters
		subnet6.CommonSubnetParameters.DDNSParameters = params.DDNSParameters
		subnet6.CommonSubnetParameters.HostnameCharParameters = params.HostnameCharParameters
		subnet6.CommonSubnetParameters.ReservationParameters = params.ReservationParameters
		subnet6.CommonSubnetParameters.TimerParameters = params.TimerParameters
		subnet6.CommonSubnetParameters.ValidLifetimeParameters = params.ValidLifetimeParameters
		subnet6.CommonSubnetParameters.Allocator = params.Allocator
		subnet6.CommonSubnetParameters.Interface = params.Interface
		subnet6.CommonSubnetParameters.StoreExtendedInfo = params.StoreExtendedInfo
		subnet6.CommonSubnetParameters.Relay = params.Relay
		subnet6.PreferredLifetimeParameters = params.PreferredLifetimeParameters
		subnet6.PDAllocator = params.PDAllocator
		subnet6.InterfaceID = params.InterfaceID
		subnet6.RapidCommit = params.RapidCommit
	}
	// Subnet-level DHCP options.
	for _, option := range subnet.GetDHCPOptions(daemonID) {
		optionData, err := CreateSingleOptionData(daemonID, lookup, option)
		if err != nil {
			return nil, err
		}
		subnet6.OptionData = append(subnet6.OptionData, *optionData)
	}
	return subnet6, nil
}

// Converts a subnet in Stork to a structure accepted by the subnet4-del and
// subnet6-del commands in Kea.
func CreateSubnetCmdsDeletedSubnet(daemonID int64, subnet SubnetAccessor) (deletedSubnet *SubnetCmdsDeletedSubnet, err error) {
	var subnetID int64
	if subnetID = subnet.GetID(daemonID); subnetID == 0 {
		err = errors.Errorf("daemon %d is not associated with the subnet %s", daemonID, subnet.GetPrefix())
		return
	}
	// Create the subnet instance.
	deletedSubnet = &SubnetCmdsDeletedSubnet{
		ID: subnetID,
	}
	return
}
