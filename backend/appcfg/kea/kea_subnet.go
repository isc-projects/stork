package keaconfig

import (
	"bytes"
	"net"

	"github.com/pkg/errors"
)

// Represents address pool structure within Kea configuration.
type Pool struct {
	Pool string
}

// Represents prefix delegation pool structure within Kea configuration.
type PdPool struct {
	Prefix            string
	PrefixLen         int    `mapstructure:"prefix-len"`
	DelegatedLen      int    `mapstructure:"delegated-len"`
	ExcludedPrefix    string `mapstructure:"excluded-prefix"`
	ExcludedPrefixLen int    `mapstructure:"excluded-prefix-len"`
}

// Represents a subnet with pools within Kea configuration.
type Subnet struct {
	ID           int64
	Subnet       string
	ClientClass  string `mapstructure:"client-class"`
	Pools        []Pool
	PdPools      []PdPool `mapstructure:"pd-pools"`
	Reservations []Reservation
}

// Represents a shared network with subnets within Kea configuration.
type SharedNetwork struct {
	Name    string
	Subnet4 []Subnet
	Subnet6 []Subnet
}

// Represents a subnet retrieved from database from app table,
// from config json field.
type KeaSubnet struct {
	ID             int
	AppID          int
	Subnet         string
	Pools          []map[string]interface{}
	SharedNetwork  string
	MachineAddress string
	AgentPort      int64
}

// Represents a shared network retrieved from database from app table,
// from config json field.
type KeaSharedNetwork struct {
	Name           string
	AppID          int
	Subnets        []map[string]interface{}
	MachineAddress string
	AgentPort      int64
}

// Matches the prefix of a subnet with the given IP network. If the match is
// found the local subnet id of that subnet is returned. Otherwise, the value
// of 0 is returned.
func getMatchingSubnetLocalID(subnet interface{}, ipNet *net.IPNet) int64 {
	sn := subnet.(map[string]interface{})

	// Parse the prefix into a common form that can be used for comparison.
	_, localNetwork, err := net.ParseCIDR(sn["subnet"].(string))
	if err != nil {
		return 0
	}
	// Compare the prefix of the subnet we have found and the specified prefix.
	if (localNetwork != nil) && net.IP.Equal(ipNet.IP, localNetwork.IP) &&
		bytes.Equal(ipNet.Mask, localNetwork.Mask) {
		snID, ok := sn["id"]
		if ok {
			return int64(snID.(float64))
		}
		return int64(0)
	}
	// No match.
	return 0
}

// Scans subnets within the Kea configuration and returns the ID of the subnet having
// the specified prefix.
func (c *Map) GetLocalSubnetID(prefix string) int64 {
	_, globalNetwork, err := net.ParseCIDR(prefix)
	if err != nil || globalNetwork == nil {
		return 0
	}

	// Depending on the DHCP server type, we need to use different name of the list
	// holding the subnets.
	rootName, ok := c.GetRootName()
	if !ok {
		return 0
	}
	var subnetParamName string
	switch rootName {
	case RootNameDHCPv4:
		subnetParamName = "subnet4"
	case RootNameDHCPv6:
		subnetParamName = "subnet6"
	default:
		// If this is neither the DHCPv4 nor DHCPv6 server, there is nothing to do.
		return 0
	}

	// First, let's iterate over the subnets which are not associated with any
	// shared network.
	if subnetList, ok := c.GetTopLevelList(subnetParamName); ok {
		for _, s := range subnetList {
			id := getMatchingSubnetLocalID(s, globalNetwork)
			if id > 0 {
				return id
			}
		}
	}

	// No match. Let's get the subnets belonging to the shared networks.
	if networkList, ok := c.GetTopLevelList("shared-networks"); ok {
		for _, n := range networkList {
			if network, ok := n.(map[string]interface{}); ok {
				if subnetList, ok := network[subnetParamName].([]interface{}); ok {
					for _, s := range subnetList {
						id := getMatchingSubnetLocalID(s, globalNetwork)
						if id > 0 {
							return id
						}
					}
				}
			}
		}
	}

	return 0
}

// Parses shared-networks list into the specified structure. The argument
// must be a pointer to a slice of structures reflecting the shared network
// data.
func (c *Map) DecodeSharedNetworks(decodedSharedNetworks interface{}) error {
	if sharedNetworksList, ok := c.GetTopLevelList("shared-networks"); ok {
		if err := decode(sharedNetworksList, decodedSharedNetworks); err != nil {
			return errors.WithMessage(err, "problem parsing shared-networks")
		}
	}
	return nil
}

// Parses subnet4 or subnet6 list into the specified structure. The argument
// must be a pointer to a slice of structures reflecting the subnet
// data.
func (c *Map) DecodeTopLevelSubnets(decodedSubnets interface{}) error {
	rootName, ok := c.GetRootName()
	if !ok {
		return errors.New("missing root node")
	}
	var subnetsList []interface{}
	switch rootName {
	case "Dhcp4":
		subnetsList, ok = c.GetTopLevelList("subnet4")
	case "Dhcp6":
		subnetsList, ok = c.GetTopLevelList("subnet6")
	default:
		return errors.Errorf("invalid configuration root node %s", rootName)
	}
	if ok {
		if err := decode(subnetsList, decodedSubnets); err != nil {
			return errors.WithMessage(err, "problem parsing subnets")
		}
	}
	return nil
}
