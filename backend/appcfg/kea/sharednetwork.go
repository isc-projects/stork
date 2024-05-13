package keaconfig

import (
	dhcpmodel "isc.org/stork/datamodel/dhcp"
)

var (
	_ SharedNetwork = (*SharedNetwork4)(nil)
	_ SharedNetwork = (*SharedNetwork6)(nil)
)

// An interface representing a shared network in Stork, extended with Kea-specific
// shared network DHCP configuration.
type SharedNetworkAccessor interface {
	dhcpmodel.SharedNetworkAccessor
	GetName() string
	GetKeaParameters(int64) *SharedNetworkParameters
	GetSubnets(int64) []SubnetAccessor
}

// An interface representing a shared network in Kea. It is implemented
// by the SharedNetwork4 and SharedNetwork6. This interface is used in
// functions that identically process the DHCPv4 and DHCPv6 shared
// networks. Their actual type is hidden behind this interface.
type SharedNetwork interface {
	GetName() string
	GetSubnets() []Subnet
	GetSharedNetworkParameters() *SharedNetworkParameters
	GetDHCPOptions() []SingleOptionData
}

// Represents Kea shared network parameter groups supported by both
// DHCPv4 and DHCPv6 servers.
type CommonSharedNetworkParameters struct {
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
	Relay             *Relay             `json:"relay,omitempty"`
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
	CommonSharedNetworkParameters
	Authoritative  *bool     `json:"authoritative,omitempty"`
	BootFileName   *string   `json:"boot-file-name,omitempty"`
	MatchClientID  *bool     `json:"match-client-id,omitempty"`
	Name           string    `json:"name"`
	NextServer     *string   `json:"next-server,omitempty"`
	ServerHostname *string   `json:"server-hostname,omitempty"`
	Subnet4        []Subnet4 `json:"subnet4,omitempty"`
}

// Represents an IPv6 shared network in Kea.
type SharedNetwork6 struct {
	CommonSharedNetworkParameters
	PreferredLifetimeParameters
	PDAllocator *string   `json:"pd-allocator,omitempty"`
	InterfaceID *string   `json:"interface-id,omitempty"`
	Name        string    `json:"name"`
	RapidCommit *bool     `json:"rapid-commit,omitempty"`
	Subnet6     []Subnet6 `json:"subnet6,omitempty"`
}

// Denotes what to do with the subnets of a deleted shared network.
// Kea supports two types of operations: keep and delete.
type SharedNetworkSubnetsAction string

const (
	// Preserve the subnets of the deleted shared network making them
	// top-level subnets.
	SharedNetworkSubnetsActionKeep SharedNetworkSubnetsAction = "keep"
	// Delete subnets associated with the deleted shared network.
	SharedNetworkSubnetsActionDelete SharedNetworkSubnetsAction = "delete"
)

// Represents deleted shared network. It includes the fields required by Kea to
// find the shared network and delete it.
type SubnetCmdsDeletedSharedNetwork struct {
	Name          string                     `json:"name"`
	SubnetsAction SharedNetworkSubnetsAction `json:"subnets-action"`
}

// Returns shared network name.
func (s SharedNetwork4) GetName() string {
	return s.Name
}

// Returns shared network subnets.
func (s SharedNetwork4) GetSubnets() (subnets []Subnet) {
	for i := range s.Subnet4 {
		subnets = append(subnets, &s.Subnet4[i])
	}
	return
}

// Returns shared network DHCP options.
func (s SharedNetwork4) GetDHCPOptions() []SingleOptionData {
	return s.OptionData
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

// Returns shared network name.
func (s SharedNetwork6) GetName() string {
	return s.Name
}

// Returns shared network subnets.
func (s SharedNetwork6) GetSubnets() (subnets []Subnet) {
	for i := range s.Subnet6 {
		subnets = append(subnets, &s.Subnet6[i])
	}
	return
}

// Returns shared network DHCP options.
func (s SharedNetwork6) GetDHCPOptions() []SingleOptionData {
	return s.OptionData
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
func CreateSharedNetwork4(daemonID int64, lookup DHCPOptionDefinitionLookup, sharedNetwork SharedNetworkAccessor) (*SharedNetwork4, error) {
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
	for _, subnet := range sharedNetwork.GetSubnets(daemonID) {
		subnet4, err := CreateSubnet4(daemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		sharedNetwork4.Subnet4 = append(sharedNetwork4.Subnet4, *subnet4)
	}
	return sharedNetwork4, nil
}

// Creates an IPv6 shared network configuration in Kea from the shared network data
// model in Stork. The daemonID parameter is used to identify a daemon in the Stork
// shared network whose configuration should be converted to the Kea format. The lookup
// is an interface returning definitions for the converted DHCP options. Finally, the
// sharedNetwork is the interface representing a shared network data model in Stork
// (e.g., dbmodel.SharedNetwork should implement this interface).
func CreateSharedNetwork6(daemonID int64, lookup DHCPOptionDefinitionLookup, sharedNetwork SharedNetworkAccessor) (*SharedNetwork6, error) {
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
	for _, subnet := range sharedNetwork.GetSubnets(daemonID) {
		subnet6, err := CreateSubnet6(daemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		sharedNetwork6.Subnet6 = append(sharedNetwork6.Subnet6, *subnet6)
	}
	return sharedNetwork6, nil
}

// Converts a shared network in Stork to a structure accepted by the
// network4-del and network6-del commands in Kea.
func CreateSubnetCmdsDeletedSharedNetwork(daemonID int64, sharedNetwork SharedNetworkAccessor, subnetsAction SharedNetworkSubnetsAction) (deletedSharedNetwork *SubnetCmdsDeletedSharedNetwork) {
	deletedSharedNetwork = &SubnetCmdsDeletedSharedNetwork{
		Name:          sharedNetwork.GetName(),
		SubnetsAction: subnetsAction,
	}
	return
}
