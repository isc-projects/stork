package keaconfig

import (
	"fmt"

	"github.com/pkg/errors"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// An interface representing a subnet in Stork, extended with Kea-specific
// subnet DHCP configuration.
type Subnet interface {
	dhcpmodel.SubnetAccessor
	GetID(int64) int64
	GetKeaParameters(int64) *SubnetParameters
}

// Represents a relay configuration for a subnet in Kea.
type Relay struct {
	IPAddresses []string `mapstructure:"ip-addresses" json:"ip-addresses"`
}

// Represents DHCPv4-over-DHCPv6 subnet configuration parameters in Kea.
type FourOverSixParameters struct {
	FourOverSixInterface   *string `mapstructure:"4o6-interface" json:"4o6-interface,omitempty"`
	FourOverSixInterfaceID *string `mapstructure:"4o6-interface-id" json:"4o6-interface-id,omitempty"`
	FourOverSixSubnet      *string `mapstructure:"4o6-subnet" json:"4o6-subnet,omitempty"`
}

// Represents DDNS configuration parameters in Kea.
type DDNSParameters struct {
	DDNSGeneratedPrefix       *string `mapstructure:"ddns-generated-prefix" json:"ddns-generated-prefix,omitempty"`
	DDNSOverrideClientUpdate  *bool   `mapstructure:"ddns-override-client-update" json:"ddns-override-client-update,omitempty"`
	DDNSOverrideNoUpdate      *bool   `mapstructure:"ddns-override-no-update" json:"ddns-override-no-update,omitempty"`
	DDNSQualifyingSuffix      *string `mapstructure:"ddns-qualifying-suffix" json:"ddns-qualifying-suffix,omitempty"`
	DDNSReplaceClientName     *string `mapstructure:"ddns-replace-client-name" json:"ddns-replace-client-name,omitempty"`
	DDNSSendUpdates           *bool   `mapstructure:"ddns-send-updates" json:"ddns-send-updates,omitempty"`
	DDNSUpdateOnReview        *bool   `mapstructure:"ddns-update-on-renew" json:"ddns-update-on-renew,omitempty"`
	DDNSUseConflictResolution *bool   `mapstructure:"ddns-use-conflict-resolution" json:"ddns-use-conflict-resolution,omitempty"`
}

// Represents Kea configuration parameters for hostname manipulation.
type HostnameCharParameters struct {
	HostnameCharReplacement *string `mapstructure:"hostname-char-replacement" json:"hostname-char-replacement,omitempty"`
	HostnameCharSet         *string `mapstructure:"hostname-char-set" json:"hostname-char-set,omitempty"`
}

// Represents Kea configuration parameters for selecting host reservation modes.
type ReservationParameters struct {
	ReservationMode       *string `mapstructure:"reservation-mode" json:"reservation-mode,omitempty"`
	ReservationsGlobal    *bool   `mapstructure:"reservations-global" json:"reservations-global,omitempty"`
	ReservationsInSubnet  *bool   `mapstructure:"reservations-in-subnet" json:"reservations-in-subnet,omitempty"`
	ReservationsOutOfPool *bool   `mapstructure:"reservations-out-of-pool" json:"reservations-out-of-pool,omitempty"`
}

// Represents DHCP timer configuration parameters in Kea.
type TimerParameters struct {
	RenewTimer        *int64   `mapstructure:"renew-timer" json:"renew-timer,omitempty"`
	RebindTimer       *int64   `mapstructure:"rebind-timer" json:"rebind-timer,omitempty"`
	T1Percent         *float32 `mapstructure:"t1-percent" json:"t1-percent,omitempty"`
	T2Percent         *float32 `mapstructure:"t2-percent" json:"t2-percent,omitempty"`
	CalculateTeeTimes *bool    `mapstructure:"calculate-tee-times" json:"calculate-tee-times,omitempty"`
}

// Represents DHCP lease caching parameters in Kea.
type CacheParameters struct {
	CacheThreshold *float32 `mapstructure:"cache-threshold" json:"cache-threshold,omitempty"`
	CacheMaxAge    *int64   `mapstructure:"cache-max-age" json:"cache-max-age,omitempty"`
}

// Represents client-class configuration in Kea.
type ClientClassParameters struct {
	ClientClass          *string  `mapstructure:"client-class" json:"client-class,omitempty"`
	RequireClientClasses []string `mapstructure:"require-client-classes" json:"require-client-classes,omitempty"`
}

// Represents valid lifetime configuration parameters in Kea.
type ValidLifetimeParameters struct {
	ValidLifetime    *int64 `mapstructure:"valid-lifetime" json:"valid-lifetime,omitempty"`
	MinValidLifetime *int64 `mapstructure:"min-valid-lifetime" json:"min-valid-lifetime,omitempty"`
	MaxValidLifetime *int64 `mapstructure:"max-valid-lifetime" json:"max-valid-lifetime,omitempty"`
}

// Represents preferred lifetime configuration parameters in Kea.
type PreferredLifetimeParameters struct {
	PreferredLifetime    *int64 `mapstructure:"preferred-lifetime" json:"preferred-lifetime,omitempty"`
	MinPreferredLifetime *int64 `mapstructure:"min-preferred-lifetime" json:"min-preferred-lifetime,omitempty"`
	MaxPreferredLifetime *int64 `mapstructure:"max-preferred-lifetime" json:"max-preferred-lifetime,omitempty"`
}

// Represents mandatory subnet configuration parameters in Kea.
// Note that ID can be left unspecified by the user. In this case
// it will be auto-generated. So, mandatory means that it is always
// present in the Kea runtime configuration.
type MandatorySubnetParameters struct {
	ID     int64  `mapstructure:"id" json:"id"`
	Subnet string `mapstructure:"subnet" json:"subnet"`
}

// Represents Kea subnet parameter groups supported by both DHCPv4
// and DHCPv6 servers.
type CommonSubnetParameters struct {
	CacheParameters         `mapstructure:",squash"`
	ClientClassParameters   `mapstructure:",squash"`
	DDNSParameters          `mapstructure:",squash"`
	HostnameCharParameters  `mapstructure:",squash"`
	ReservationParameters   `mapstructure:",squash"`
	TimerParameters         `mapstructure:",squash"`
	ValidLifetimeParameters `mapstructure:",squash"`
	Allocator               *string            `mapstructure:"allocator" json:"allocator,omitempty"`
	Interface               *string            `mapstructure:"interface" json:"interface,omitempty"`
	StoreExtendedInfo       *bool              `mapstructure:"store-extended-info" json:"store-extended-info,omitempty"`
	OptionData              []SingleOptionData `mapstructure:"option-data" json:"option-data,omitempty"`
	Pools                   []Pool             `mapstructure:"pools" json:"pools,omitempty"`
	Relay                   *Relay             `mapstructure:"relay" json:"relay,omitempty"`
	Reservations            []Reservation      `mapstructure:"reservations" json:"reservations,omitempty"`
}

// Represents an IPv4 subnet in Kea.
type Subnet4 struct {
	CommonSubnetParameters    `mapstructure:",squash"`
	FourOverSixParameters     `mapstructure:",squash"`
	MandatorySubnetParameters `mapstructure:",squash"`
	Authoritative             *bool   `mapstructure:"authoritative" json:"authoritative,omitempty"`
	BootFileName              *string `mapstructure:"boot-file-name" json:"boot-file-name,omitempty"`
	MatchClientID             *bool   `mapstructure:"match-client-id" json:"match-client-id,omitempty"`
	NextServer                *string `mapstructure:"next-server" json:"next-server,omitempty"`
	ServerHostname            *string `mapstructure:"server-hostname" json:"server-hostname,omitempty"`
}

// Represents an IPv6 subnet in Kea.
type Subnet6 struct {
	CommonSubnetParameters      `mapstructure:",squash"`
	MandatorySubnetParameters   `mapstructure:",squash"`
	PreferredLifetimeParameters `mapstructure:",squash"`
	PDAllocator                 *string  `mapstructure:"pd-allocator" json:"pd-allocator,omitempty"`
	InterfaceID                 *string  `mapstructure:"interface-id" json:"interface-id,omitempty"`
	PDPools                     []PDPool `mapstructure:"pd-pools" json:"pd-pools,omitempty"`
	RapidCommit                 *bool    `mapstructure:"rapid-commit" json:"rapid-commit,omitempty"`
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

// Returns a canonical subnet prefix or an error if the prefix is
// invalid.
func (s MandatorySubnetParameters) GetCanonicalPrefix() (string, error) {
	cidr := storkutil.ParseIP(s.Subnet)
	if cidr == nil {
		return "", errors.Errorf("invalid subnet prefix: %s", s.Subnet)
	}
	return cidr.GetNetworkPrefixWithLength(), nil
}

// Returns a canonical IPv4 subnet prefix.
func (s *Subnet4) GetCanonicalPrefix() (string, error) {
	return s.MandatorySubnetParameters.GetCanonicalPrefix()
}

// Returns a canonical IPv6 subnet prefix.
func (s *Subnet6) GetCanonicalPrefix() (string, error) {
	return s.MandatorySubnetParameters.GetCanonicalPrefix()
}

// Returns Kea-specific IPv4 subnet configuration parameters.
func (s Subnet4) GetSubnetParameters() *SubnetParameters {
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

// Returns Kea-specific IPv6 subnet configuration parameters.
func (s Subnet6) GetSubnetParameters() *SubnetParameters {
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
func CreateSubnet4(daemonID int64, lookup DHCPOptionDefinitionLookup, subnet Subnet) (*Subnet4, error) {
	// Mandatory parameters.
	subnet4 := &Subnet4{
		MandatorySubnetParameters: MandatorySubnetParameters{
			ID:     subnet.GetID(daemonID),
			Subnet: subnet.GetPrefix(),
		},
		CommonSubnetParameters: CommonSubnetParameters{},
	}
	// Address pools.
	for _, pool := range subnet.GetAddressPools() {
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
				keaPool.ClientClass = *params.ClientClass
			}
			keaPool.RequireClientClasses = params.RequireClientClasses
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
func CreateSubnet6(daemonID int64, lookup DHCPOptionDefinitionLookup, subnet Subnet) (*Subnet6, error) {
	subnet6 := &Subnet6{
		MandatorySubnetParameters: MandatorySubnetParameters{
			ID:     subnet.GetID(daemonID),
			Subnet: subnet.GetPrefix(),
		},
	}
	// Address pools.
	for _, pool := range subnet.GetAddressPools() {
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
				keaPool.ClientClass = *params.ClientClass
			}
			keaPool.RequireClientClasses = params.RequireClientClasses
		}
		// Add the pool to the subnet.
		subnet6.Pools = append(subnet6.Pools, keaPool)
	}
	// Delegated prefix pools.
	for _, pool := range subnet.GetPrefixPools() {
		// Pool prefix.
		prefix, length, err := pool.GetModel().GetPrefix()
		if err != nil {
			return nil, err
		}
		// Excluded prefix.
		excludedPrefix, excludedPrefixLength, err := pool.GetModel().GetDelegatedPrefix()
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
				keaPool.ClientClass = *params.ClientClass
			}
			keaPool.RequireClientClasses = params.RequireClientClasses
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
