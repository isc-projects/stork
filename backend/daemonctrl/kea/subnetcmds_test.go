package keactrl

import (
	"testing"

	require "github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
	storkutil "isc.org/stork/util"
)

// Tests network4-add command.
func TestNewCommandNetwork4Add(t *testing.T) {
	command := NewCommandNetwork4Add(&keaconfig.SharedNetwork4{
		Name:          "foo",
		Authoritative: storkutil.Ptr(true),
	}, daemonname.DHCPv4)
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
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
	}`, string(bytes))
}

// Tests network6-add command.
func TestNewCommandNetwork6Add(t *testing.T) {
	command := NewCommandNetwork6Add(&keaconfig.SharedNetwork6{
		Name:        "foo",
		PDAllocator: storkutil.Ptr("flq"),
	}, daemonname.DHCPv6)
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
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
	}`, string(bytes))
}

// Tests network4-del command.
func TestNewCommandNetwork4Del(t *testing.T) {
	command := NewCommandNetwork4Del(&keaconfig.SubnetCmdsDeletedSharedNetwork{
		Name:          "foo",
		SubnetsAction: keaconfig.SharedNetworkSubnetsActionDelete,
	}, daemonname.DHCPv4)
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "network4-del",
		"service": ["dhcp4"],
		"arguments": {
			"name": "foo",
			"subnets-action": "delete"
		}
	}`, string(bytes))
}

// Tests network6-del command.
func TestNewCommandNetwork6Del(t *testing.T) {
	command := NewCommandNetwork6Del(&keaconfig.SubnetCmdsDeletedSharedNetwork{
		Name:          "foo",
		SubnetsAction: keaconfig.SharedNetworkSubnetsActionKeep,
	}, daemonname.DHCPv6)
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "network6-del",
		"service": ["dhcp6"],
		"arguments": {
			"name": "foo",
			"subnets-action": "keep"
		}
	}`, string(bytes))
}

// Tests network4-subnet-add command.
func TestNewCommandNetwork4SubnetAdd(t *testing.T) {
	command := NewCommandNetwork4SubnetAdd("foo", 123, daemonname.DHCPv4)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "network4-subnet-add",
		"service": ["dhcp4"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, string(bytes))
}

// Tests network6-subnet-add command.
func TestNewCommandNetwork6SubnetAdd(t *testing.T) {
	command := NewCommandNetwork6SubnetAdd("foo", 123, daemonname.DHCPv6)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "network6-subnet-add",
		"service": ["dhcp6"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, string(bytes))
}

// Tests network4-subnet-del command.
func TestNewCommandNetwork4SubnetDel(t *testing.T) {
	command := NewCommandNetwork4SubnetDel("foo", 123, daemonname.DHCPv4)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "network4-subnet-del",
		"service": ["dhcp4"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, string(bytes))
}

// Tests network6-subnet-del command.
func TestNewCommandNetwork6SubnetDel(t *testing.T) {
	command := NewCommandNetwork6SubnetDel("foo", 123, daemonname.DHCPv6)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "network6-subnet-del",
		"service": ["dhcp6"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, string(bytes))
}

// Tests network4-subnet-del command.
func TestNewCommandNetworkSubnetDelFamily4(t *testing.T) {
	command := NewCommandNetworkSubnetDel(4, "foo", 123, daemonname.DHCPv4)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "network4-subnet-del",
		"service": ["dhcp4"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, string(bytes))
}

// Tests network6-subnet-del command.
func TestNewCommandNetworkSubnetDelFamily6(t *testing.T) {
	// The network6-subnet-del command should be returned for different families.
	families := []int{6, 1, 0}
	for _, family := range families {
		command := NewCommandNetworkSubnetDel(family, "foo", 123, daemonname.DHCPv6)
		require.NotNil(t, command)
		bytes, err := command.Marshal()
		require.NoError(t, err)
		require.JSONEq(t, `{
		"command": "network6-subnet-del",
		"service": ["dhcp6"],
		"arguments": {
			"id": 123,
			"name": "foo"
		}
	}`, string(bytes))
	}
}

// Tests subnet4-add command.
func TestNewCommandSubnet4Add(t *testing.T) {
	command := NewCommandSubnet4Add(&keaconfig.Subnet4{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			ID:     2,
			Subnet: "192.0.2.0/24",
		},
	}, daemonname.DHCPv4)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
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
	}`, string(bytes))
}

// Tests subnet6-add command.
func TestNewCommandSubnet6Add(t *testing.T) {
	command := NewCommandSubnet6Add(&keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			ID:     2,
			Subnet: "2001:db8:1::/64",
		},
	}, daemonname.DHCPv6)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
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
	}`, string(bytes))
}

// Tests subnet4-del command.
func TestNewCommandSubnet4Del(t *testing.T) {
	command := NewCommandSubnet4Del(&keaconfig.SubnetCmdsDeletedSubnet{
		ID: 2,
	}, daemonname.DHCPv4)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "subnet4-del",
		"service": ["dhcp4"],
		"arguments": {
			"id": 2
		}
	}`, string(bytes))
}

// Tests subnet6-del command.
func TestNewCommandSubnet6Del(t *testing.T) {
	command := NewCommandSubnet6Del(&keaconfig.SubnetCmdsDeletedSubnet{
		ID: 4,
	}, daemonname.DHCPv6)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "subnet6-del",
		"service": ["dhcp6"],
		"arguments": {
			"id": 4
		}
	}`, string(bytes))
}

// Tests subnet4-del command.
func TestNewCommandSubnetDelFamily4(t *testing.T) {
	command := NewCommandSubnetDel(4, &keaconfig.SubnetCmdsDeletedSubnet{
		ID: 2,
	}, daemonname.DHCPv4)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "subnet4-del",
		"service": ["dhcp4"],
		"arguments": {
			"id": 2
		}
	}`, string(bytes))
}

// Tests subnet6-del command.
func TestNewCommandSubnetDelFamily6(t *testing.T) {
	// The subnet6-del command should be returned for different families.
	families := []int{6, 1, 0}
	for _, family := range families {
		command := NewCommandSubnetDel(family, &keaconfig.SubnetCmdsDeletedSubnet{
			ID: 4,
		}, daemonname.DHCPv6)
		require.NotNil(t, command)
		bytes, err := command.Marshal()
		require.NoError(t, err)
		require.JSONEq(t, `{
		"command": "subnet6-del",
		"service": ["dhcp6"],
		"arguments": {
			"id": 4
		}
	}`, string(bytes))
	}
}

// Tests subnet4-update command.
func TestNewCommandSubnet4Update(t *testing.T) {
	command := NewCommandSubnet4Update(&keaconfig.Subnet4{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			ID:     2,
			Subnet: "192.0.2.0/24",
		},
	}, daemonname.DHCPv4)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
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
	}`, string(bytes))
}

// Tests subnet6-update command.
func TestNewCommandSubnet6Update(t *testing.T) {
	command := NewCommandSubnet6Update(&keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			ID:     2,
			Subnet: "2001:db8:1::/64",
		},
	}, daemonname.DHCPv6)
	require.NotNil(t, command)
	bytes, err := command.Marshal()
	require.NoError(t, err)
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
	}`, string(bytes))
}
