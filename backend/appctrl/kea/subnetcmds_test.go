package keactrl

import (
	"testing"

	require "github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	storkutil "isc.org/stork/util"
)

// Tests network4-add command.
func TestNewCommandNetwork4Add(t *testing.T) {
	command := NewCommandNetwork4Add(&keaconfig.SharedNetwork4{
		Name:          "foo",
		Authoritative: storkutil.Ptr(true),
	}, DHCPv4)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "network4-add",
		"service": ["dhcp4"],
		"arguments": {
			"shared-networks": [
				{
					"name": "foo",
					"authoritative": true
				}
			]
		}
	}`, command.Marshal())
}

// Tests network6-add command.
func TestNewCommandNetwork6Add(t *testing.T) {
	command := NewCommandNetwork6Add(&keaconfig.SharedNetwork6{
		Name:        "foo",
		PDAllocator: storkutil.Ptr("flq"),
	}, DHCPv6)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "network6-add",
		"service": ["dhcp6"],
		"arguments": {
			"shared-networks": [
				{
					"name": "foo",
					"pd-allocator": "flq"
				}
			]
		}
	}`, command.Marshal())
}

// Tests network4-del command.
func TestNewCommandNetwork4Del(t *testing.T) {
	command := NewCommandNetwork4Del(&keaconfig.SubnetCmdsDeletedSharedNetwork{
		Name:          "foo",
		SubnetsAction: keaconfig.SharedNetworkSubnetsActionDelete,
	}, DHCPv4)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "network4-del",
		"service": ["dhcp4"],
		"arguments": {
			"name": "foo",
			"subnets-action": "delete"
		}
	}`, command.Marshal())
}

// Tests network6-del command.
func TestNewCommandNetwork6Del(t *testing.T) {
	command := NewCommandNetwork6Del(&keaconfig.SubnetCmdsDeletedSharedNetwork{
		Name:          "foo",
		SubnetsAction: keaconfig.SharedNetworkSubnetsActionKeep,
	}, DHCPv6)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "network6-del",
		"service": ["dhcp6"],
		"arguments": {
			"name": "foo",
			"subnets-action": "keep"
		}
	}`, command.Marshal())
}

// Tests network4-subnet-add command.
func TestNewCommandNetwork4SubnetAdd(t *testing.T) {
	command := NewCommandNetwork4SubnetAdd("foo", 123, DHCPv4)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "network4-subnet-add",
		"service": ["dhcp4"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, command.Marshal())
}

// Tests network6-subnet-add command.
func TestNewCommandNetwork6SubnetAdd(t *testing.T) {
	command := NewCommandNetwork6SubnetAdd("foo", 123, DHCPv6)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "network6-subnet-add",
		"service": ["dhcp6"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, command.Marshal())
}

// Tests network4-subnet-del command.
func TestNewCommandNetwork4SubnetDel(t *testing.T) {
	command := NewCommandNetwork4SubnetDel("foo", 123, DHCPv4)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "network4-subnet-del",
		"service": ["dhcp4"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, command.Marshal())
}

// Tests network6-subnet-del command.
func TestNewCommandNetwork6SubnetDel(t *testing.T) {
	command := NewCommandNetwork6SubnetDel("foo", 123, DHCPv6)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "network6-subnet-del",
		"service": ["dhcp6"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, command.Marshal())
}

// Tests network4-subnet-del command.
func TestNewCommandNetworkSubnetDelFamily4(t *testing.T) {
	command := NewCommandNetworkSubnetDel(4, "foo", 123, DHCPv4)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "network4-subnet-del",
		"service": ["dhcp4"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, command.Marshal())
}

// Tests network6-subnet-del command.
func TestNewCommandNetworkSubnetDelFamily6(t *testing.T) {
	// The network6-subnet-del command should be returned for different families.
	for family := range []int64{6, 1, 0} {
		command := NewCommandNetworkSubnetDel(family, "foo", 123, DHCPv6)
		require.NotNil(t, command)
		require.JSONEq(t, `{
		"command": "network6-subnet-del",
		"service": ["dhcp6"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, command.Marshal())
	}
}

// Tests subnet4-add command.
func TestNewCommandSubnet4Add(t *testing.T) {
	command := NewCommandSubnet4Add(&keaconfig.Subnet4{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			ID:     2,
			Subnet: "192.0.2.0/24",
		},
	}, DHCPv4)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "subnet4-add",
		"service": ["dhcp4"],
		"arguments": {
			"subnet4": [
				{
					"id": 2,
					"subnet": "192.0.2.0/24"
				}
			]
		}
	}`, command.Marshal())
}

// Tests subnet6-add command.
func TestNewCommandSubnet6Add(t *testing.T) {
	command := NewCommandSubnet6Add(&keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			ID:     2,
			Subnet: "2001:db8:1::/64",
		},
	}, DHCPv6)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "subnet6-add",
		"service": ["dhcp6"],
		"arguments": {
			"subnet6": [
				{
					"id": 2,
					"subnet": "2001:db8:1::/64"
				}
			]
		}
	}`, command.Marshal())
}

// Tests subnet4-del command.
func TestNewCommandSubnet4Del(t *testing.T) {
	command := NewCommandSubnet4Del(&keaconfig.SubnetCmdsDeletedSubnet{
		ID: 2,
	}, DHCPv4)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "subnet4-del",
		"service": ["dhcp4"],
		"arguments": {
			"id": 2
		}
	}`, command.Marshal())
}

// Tests subnet6-del command.
func TestNewCommandSubnet6Del(t *testing.T) {
	command := NewCommandSubnet6Del(&keaconfig.SubnetCmdsDeletedSubnet{
		ID: 4,
	}, "dhcp6")
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "subnet6-del",
		"service": ["dhcp6"],
		"arguments": {
			"id": 4
		}
	}`, command.Marshal())
}

// Tests subnet4-del command.
func TestNewCommandSubnetDelFamily4(t *testing.T) {
	command := NewCommandSubnetDel(4, &keaconfig.SubnetCmdsDeletedSubnet{
		ID: 2,
	}, DHCPv4)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "subnet4-del",
		"service": ["dhcp4"],
		"arguments": {
			"id": 2
		}
	}`, command.Marshal())
}

// Tests subnet6-del command.
func TestNewCommandSubnetDelFamily6(t *testing.T) {
	// The subnet6-del command should be returned for different families.
	for family := range []int64{6, 1, 0} {
		command := NewCommandSubnetDel(family, &keaconfig.SubnetCmdsDeletedSubnet{
			ID: 4,
		}, "dhcp6")
		require.NotNil(t, command)
		require.JSONEq(t, `{
		"command": "subnet6-del",
		"service": ["dhcp6"],
		"arguments": {
			"id": 4
		}
	}`, command.Marshal())
	}
}

// Tests subnet4-update command.
func TestNewCommandSubnet4Update(t *testing.T) {
	command := NewCommandSubnet4Update(&keaconfig.Subnet4{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			ID:     2,
			Subnet: "192.0.2.0/24",
		},
	}, DHCPv4)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "subnet4-update",
		"service": ["dhcp4"],
		"arguments": {
			"subnet4": [
				{
					"id": 2,
					"subnet": "192.0.2.0/24"
				}
			]
		}
	}`, command.Marshal())
}

// Tests subnet6-update command.
func TestNewCommandSubnet6Update(t *testing.T) {
	command := NewCommandSubnet6Update(&keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			ID:     2,
			Subnet: "2001:db8:1::/64",
		},
	}, DHCPv6)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "subnet6-update",
		"service": ["dhcp6"],
		"arguments": {
			"subnet6": [
				{
					"id": 2,
					"subnet": "2001:db8:1::/64"
				}
			]
		}
	}`, command.Marshal())
}
