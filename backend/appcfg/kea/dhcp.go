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
}

// Represents Kea DHCPv4 configuration.
type DHCPv4Config struct {
	CommonDHCPConfig
	SharedNetworks  []SharedNetwork4   `json:"shared-networks"`
	Subnet4         []Subnet4          `json:"subnet4"`
	Subnet4ByPrefix map[string]Subnet4 `json:"-"`
}

// Represents Kea DHCPv6 configuration.
type DHCPv6Config struct {
	CommonDHCPConfig
	SharedNetworks  []SharedNetwork6   `json:"shared-networks"`
	Subnet6         []Subnet6          `json:"subnet6"`
	Subnet6ByPrefix map[string]Subnet6 `json:"-"`
}

// Represents common configuration parameters for the DHCPv4 and DHCPv6 servers.
type CommonDHCPConfig struct {
	ReservationParameters
	ClientClasses  []ClientClass   `json:"client-classes"`
	ConfigControl  *ConfigControl  `json:"config-control"`
	ControlSocket  *ControlSocket  `json:"control-socket"`
	HostsDatabase  *Database       `json:"hosts-database"`
	HostsDatabases []Database      `json:"hosts-databases"`
	HookLibraries  []HookLibrary   `json:"hooks-libraries"`
	LeaseDatabase  *Database       `json:"lease-database"`
	Loggers        []Logger        `json:"loggers"`
	MultiThreading *MultiThreading `json:"multi-threading"`
	Reservations   []Reservation   `json:"reservations"`
}

// Represents the global DHCP multi-threading parameters.
type MultiThreading struct {
	EnableMultiThreading *bool `json:"enable-multi-threading"`
	ThreadPoolSize       *int  `json:"thread-pool-size"`
	PacketQueueSize      *int  `json:"packet-queue-size"`
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
