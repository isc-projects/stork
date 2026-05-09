package keactrl

import (
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
)

// Tests that remote-subnet4-set command is constructed correctly.
// The shared network name property must be included in the serialized command
// even if it is empty.
func TestNewCommandRemoteSubnet4Set(t *testing.T) {
	subnet := keaconfig.CreateConfigBackendSubnet4(&keaconfig.Subnet4{
		Subnet4KnownParameters: keaconfig.Subnet4KnownParameters{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     1,
				Subnet: "192.0.2.0/24",
			},
		},
	}, "")
	command := NewCommandRemoteSubnet4Set(subnet, []string{"all"}, daemonname.DHCPv4)
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	marshalled, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "remote-subnet4-set",
		"service": ["dhcp4"],
		"arguments": {
			"subnets": [
				{
					"id": 1,
					"subnet": "192.0.2.0/24",
					"shared-network-name": ""
				}
			],
			"server-tags": ["all"]
		}
	}`, string(marshalled))
}

// Tests that remote-subnet4-set command serializes CB-specific fields correctly.
func TestNewCommandRemoteSubnet4SetWithSharedNetwork(t *testing.T) {
	subnet := keaconfig.CreateConfigBackendSubnet4(&keaconfig.Subnet4{
		Subnet4KnownParameters: keaconfig.Subnet4KnownParameters{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     2,
				Subnet: "10.0.0.0/8",
			},
		},
	}, "mynet")
	command := NewCommandRemoteSubnet4Set(subnet, []string{"server"}, daemonname.DHCPv4)
	require.NotNil(t, command)
	marshalled, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "remote-subnet4-set",
		"service": ["dhcp4"],
		"arguments": {
			"subnets": [
				{
					"id": 2,
					"subnet": "10.0.0.0/8",
					"shared-network-name": "mynet"
				}
			],
			"server-tags": ["server"]
		}
	}`, string(marshalled))
}

// Tests that remote-subnet6-set command is constructed correctly.
// The shared network name property must be included in the serialized command
// even if it is empty.
func TestNewCommandRemoteSubnet6Set(t *testing.T) {
	subnet := keaconfig.CreateConfigBackendSubnet6(&keaconfig.Subnet6{
		Subnet6KnownParameters: keaconfig.Subnet6KnownParameters{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     3,
				Subnet: "2001:db8:1::/64",
			},
		},
	}, "")
	command := NewCommandRemoteSubnet6Set(subnet, []string{"all"}, daemonname.DHCPv6)
	require.NotNil(t, command)
	marshalled, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "remote-subnet6-set",
		"service": ["dhcp6"],
		"arguments": {
			"subnets": [
				{
					"id": 3,
					"subnet": "2001:db8:1::/64",
					"shared-network-name": ""
				}
			],
			"server-tags": ["all"]
		}
	}`, string(marshalled))
}

// Tests that remote-subnet6-set command serializes CB-specific fields correctly.
func TestNewCommandRemoteSubnet6SetWithSharedNetwork(t *testing.T) {
	subnet := keaconfig.CreateConfigBackendSubnet6(&keaconfig.Subnet6{
		Subnet6KnownParameters: keaconfig.Subnet6KnownParameters{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     4,
				Subnet: "2001:db8:2::/64",
			},
		},
	}, "ipv6net")
	command := NewCommandRemoteSubnet6Set(subnet, []string{"server"}, daemonname.DHCPv6)
	require.NotNil(t, command)
	marshalled, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "remote-subnet6-set",
		"service": ["dhcp6"],
		"arguments": {
			"subnets": [
				{
					"id": 4,
					"subnet": "2001:db8:2::/64",
					"shared-network-name": "ipv6net"
				}
			],
			"server-tags": ["server"]
		}
	}`, string(marshalled))
}
