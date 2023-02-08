package keaconfig

import dhcpmodel "isc.org/stork/datamodel/dhcp"

// An interface representing a shared network in Stork, extended with Kea-specific
// shared network DHCP configuration.
type SharedNetwork interface {
	dhcpmodel.SharedNetworkAccessor
	GetName() string
	GetKeaParameters(int64) *SharedNetworkParameters
}

// Represents Kea shared network parameter groups supported by both
// DHCPv4 and DHCPv6 servers.
type CommonSharedNetworkParameters struct {
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
	Relay                   *Relay             `mapstructure:"relay" json:"relay,omitempty"`
}

// Represents a union of DHCP parameters for the DHCPv4 and
// the DHCPv6 servers. Such a union can be used in the
// Stork database model to hold DHCPv4 and DHCPv6 shared
// networks in a common structure (see the dbmodel package).
type SharedNetworkParameters struct {
	CacheParameters
	ClientClassParameters
	DDNSParameters
	HostnameCharParameters
	PreferredLifetimeParameters
	ReservationParameters
	TimerParameters
	ValidLifetimeParameters
	Authoritative     *bool
	Allocator         *string
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

// Represents an IPv4 shared network in Kea.
type SharedNetwork4 struct {
	CommonSharedNetworkParameters `mapstructure:",squash"`
	Authoritative                 *bool     `mapstructure:"authoritative" json:"authoritative,omitempty"`
	BootFileName                  *string   `mapstructure:"boot-file-name" json:"boot-file-name,omitempty"`
	MatchClientID                 *bool     `mapstructure:"match-client-id" json:"match-client-id,omitempty"`
	Name                          string    `mapstructure:"name" json:"name"`
	NextServer                    *string   `mapstructure:"next-server" json:"next-server,omitempty"`
	ServerHostname                *string   `mapstructure:"server-hostname" json:"server-hostname,omitempty"`
	Subnet4                       []Subnet4 `mapstructure:"subnet4" json:"subnet4,omitempty"`
}

// Represents an IPv6 shared network in Kea.
type SharedNetwork6 struct {
	CommonSharedNetworkParameters `mapstructure:",squash"`
	PreferredLifetimeParameters   `mapstructure:",squash"`
	PDAllocator                   *string   `mapstructure:"pd-allocator" json:"pd-allocator,omitempty"`
	InterfaceID                   *string   `mapstructure:"interface-id" json:"interface-id,omitempty"`
	Name                          string    `mapstructure:"name" json:"name"`
	RapidCommit                   *bool     `mapstructure:"rapid-commit" json:"rapid-commit,omitempty"`
	Subnet6                       []Subnet6 `mapstructure:"subnet6" json:"subnet6,omitempty"`
}

// Returns Kea-specific DHCPv4 shared network configuration parameters.
func (s SharedNetwork4) GetSharedNetworkParameters() *SharedNetworkParameters {
	return &SharedNetworkParameters{
		CacheParameters:         s.CacheParameters,
		ClientClassParameters:   s.ClientClassParameters,
		DDNSParameters:          s.DDNSParameters,
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

// Returns Kea-specific DHCPv6 shared network configuration parameters.
func (s SharedNetwork6) GetSharedNetworkParameters() *SharedNetworkParameters {
	return &SharedNetworkParameters{
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

// Creates an IPv4 shared network configuration in Kea from the shared network data
// model in Stork. The daemonID parameter is used to identify a daemon in the Stork
// shared network whose configuration should be converted to the Kea format. The lookup
// is an interface returning definitions for the converted DHCP options. Finally, the
// sharedNetwork is the interface representing a shared network data model in Stork
// (e.g., dbmodel.SharedNetwork should implement this interface).
func CreateSharedNetwork4(daemonID int64, lookup DHCPOptionDefinitionLookup, sharedNetwork SharedNetwork) (*SharedNetwork4, error) {
	sharedNetwork4 := &SharedNetwork4{
		Name: sharedNetwork.GetName(),
	}
	if params := sharedNetwork.GetKeaParameters(daemonID); params != nil {
		sharedNetwork4.CommonSharedNetworkParameters = CommonSharedNetworkParameters{
			CacheParameters:         params.CacheParameters,
			ClientClassParameters:   params.ClientClassParameters,
			DDNSParameters:          params.DDNSParameters,
			HostnameCharParameters:  params.HostnameCharParameters,
			ReservationParameters:   params.ReservationParameters,
			TimerParameters:         params.TimerParameters,
			ValidLifetimeParameters: params.ValidLifetimeParameters,
			Allocator:               params.Allocator,
			Interface:               params.Interface,
			StoreExtendedInfo:       params.StoreExtendedInfo,
			Relay:                   params.Relay,
		}
		sharedNetwork4.Authoritative = params.Authoritative
		sharedNetwork4.BootFileName = params.BootFileName
		sharedNetwork4.MatchClientID = params.MatchClientID
		sharedNetwork4.NextServer = params.NextServer
		sharedNetwork4.ServerHostname = params.ServerHostname
	}
	for _, option := range sharedNetwork.GetDHCPOptions(daemonID) {
		optionData, err := CreateSingleOptionData(daemonID, lookup, option)
		if err != nil {
			return nil, err
		}
		sharedNetwork4.OptionData = append(sharedNetwork4.OptionData, *optionData)
	}
	return sharedNetwork4, nil
}

// Creates an IPv6 shared network configuration in Kea from the shared network data
// model in Stork. The daemonID parameter is used to identify a daemon in the Stork
// shared network whose configuration should be converted to the Kea format. The lookup
// is an interface returning definitions for the converted DHCP options. Finally, the
// sharedNetwork is the interface representing a shared network data model in Stork
// (e.g., dbmodel.SharedNetwork should implement this interface).
func CreateSharedNetwork6(daemonID int64, lookup DHCPOptionDefinitionLookup, sharedNetwork SharedNetwork) (*SharedNetwork6, error) {
	sharedNetwork6 := &SharedNetwork6{
		Name: sharedNetwork.GetName(),
	}
	if params := sharedNetwork.GetKeaParameters(daemonID); params != nil {
		sharedNetwork6.CommonSharedNetworkParameters = CommonSharedNetworkParameters{
			CacheParameters:         params.CacheParameters,
			ClientClassParameters:   params.ClientClassParameters,
			DDNSParameters:          params.DDNSParameters,
			HostnameCharParameters:  params.HostnameCharParameters,
			ReservationParameters:   params.ReservationParameters,
			TimerParameters:         params.TimerParameters,
			ValidLifetimeParameters: params.ValidLifetimeParameters,
			Allocator:               params.Allocator,
			Interface:               params.Interface,
			StoreExtendedInfo:       params.StoreExtendedInfo,
			Relay:                   params.Relay,
		}
		sharedNetwork6.PreferredLifetimeParameters = params.PreferredLifetimeParameters
		sharedNetwork6.PDAllocator = params.PDAllocator
		sharedNetwork6.InterfaceID = params.InterfaceID
		sharedNetwork6.RapidCommit = params.RapidCommit
	}
	for _, option := range sharedNetwork.GetDHCPOptions(daemonID) {
		optionData, err := CreateSingleOptionData(daemonID, lookup, option)
		if err != nil {
			return nil, err
		}
		sharedNetwork6.OptionData = append(sharedNetwork6.OptionData, *optionData)
	}
	return sharedNetwork6, nil
}
