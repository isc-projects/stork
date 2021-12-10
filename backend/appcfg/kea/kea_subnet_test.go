package keaconfig

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

// Returns test Kea configuration including multiple IPv4 subnets.
func getTestConfigWithIPv4Subnets(t *testing.T) *Map {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet4": [
                        {
                            "id": 567,
                            "subnet": "10.1.0.0/16"
                        },
                        {
                            "id": 678,
                            "subnet": "10.2.0.0/16"
                        }
                    ]
                },
                {
                    "name": "bar",
                    "subnet4": [
                        {
                            "id": 789,
                            "subnet": "10.3.0.0/16"
                        },
                        {
                            "id": 890,
                            "subnet": "10.4.0.0/16"
                        }
                    ]
                }
            ],
            "subnet4": [
                {
                    "id": 123,
                    "subnet": "192.0.2.0/24"
                },
                {
                    "id": 234,
                    "subnet": "192.0.3.0/24"
                },
                {
                    "id": 345,
                    "subnet": "10.0.0.0/8"
                }
            ]
        }
    }`

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration including multiple IPv6 subnets.
func getTestConfigWithIPv6Subnets(t *testing.T) *Map {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet6": [
                        {
                            "id": 567,
                            "subnet": "3000:1::/32"
                        },
                        {
                            "id": 678,
                            "subnet": "3000:2::/32"
                        }
                    ]
                },
                {
                    "name": "bar",
                    "subnet6": [
                        {
                            "id": 789,
                            "subnet": "3000:3::/32"
                        },
                        {
                            "id": 890,
                            "subnet": "3000:4::/32"
                        }
                    ]
                }
            ],
            "subnet6": [
                {
                    "id": 123,
                    "subnet": "2001:db8:1::/64"
                },
                {
                    "id": 234,
                    "subnet": "2001:db8:2::/64"
                },
                {
                    "id": 345,
                    "subnet": "2001:db8:3::/64"
                }
            ]
        }
    }`

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Test that the subnet ID can be extracted from the Kea configuration for
// an IPv4 subnet having specified prefix.
func TestGetLocalIPv4SubnetID(t *testing.T) {
	cfg := getTestConfigWithIPv4Subnets(t)

	require.EqualValues(t, 567, cfg.GetLocalSubnetID("10.1.0.0/16"))
	require.EqualValues(t, 678, cfg.GetLocalSubnetID("10.2.0.1/16"))
	require.EqualValues(t, 123, cfg.GetLocalSubnetID("192.0.2.0/24"))
	require.EqualValues(t, 234, cfg.GetLocalSubnetID("192.0.3.0/24"))
	require.EqualValues(t, 345, cfg.GetLocalSubnetID("10.0.0.0/8"))
	require.EqualValues(t, 0, cfg.GetLocalSubnetID("10.0.0.0/16"))
}

// Test that the subnet ID can be extracted from the Kea configuration for
// an IPv6 subnet having specified prefix.
func TestGetLocalIPv6SubnetID(t *testing.T) {
	cfg := getTestConfigWithIPv6Subnets(t)

	require.EqualValues(t, 567, cfg.GetLocalSubnetID("3000:0001::/32"))
	require.EqualValues(t, 678, cfg.GetLocalSubnetID("3000:2::/32"))
	require.EqualValues(t, 123, cfg.GetLocalSubnetID("2001:db8:1:0::/64"))
	require.EqualValues(t, 234, cfg.GetLocalSubnetID("2001:db8:2::/64"))
	require.EqualValues(t, 345, cfg.GetLocalSubnetID("2001:db8:3::/64"))
	require.EqualValues(t, 0, cfg.GetLocalSubnetID("2001:db8:4::/64"))
}

// Test that it is possible to parse IPv4 shared-networks list into a custom
// structure.
func TestDecodeIPv4SharedNetworks(t *testing.T) {
	cfg := getTestConfigWithIPv4Subnets(t)

	networks := []struct {
		Name    string
		Subnet4 []struct {
			Subnet string
		}
	}{}
	err := cfg.DecodeSharedNetworks(&networks)
	require.NoError(t, err)
	require.Len(t, networks, 2)
	require.Len(t, networks[0].Subnet4, 2)
	require.Equal(t, "10.1.0.0/16", networks[0].Subnet4[0].Subnet)
	require.Equal(t, "10.2.0.0/16", networks[0].Subnet4[1].Subnet)
	require.Len(t, networks[1].Subnet4, 2)
	require.Equal(t, "10.3.0.0/16", networks[1].Subnet4[0].Subnet)
	require.Equal(t, "10.4.0.0/16", networks[1].Subnet4[1].Subnet)
}

// Test that it is possible to parse IPv6 shared-networks list into a custom
// structure.
func TestDecodeIPv6SharedNetworks(t *testing.T) {
	cfg := getTestConfigWithIPv6Subnets(t)

	networks := []struct {
		Name    string
		Subnet6 []struct {
			Subnet string
		}
	}{}
	err := cfg.DecodeSharedNetworks(&networks)
	require.NoError(t, err)
	require.Len(t, networks, 2)
	require.Len(t, networks[0].Subnet6, 2)
	require.Equal(t, "3000:1::/32", networks[0].Subnet6[0].Subnet)
	require.Equal(t, "3000:2::/32", networks[0].Subnet6[1].Subnet)
	require.Len(t, networks[1].Subnet6, 2)
	require.Equal(t, "3000:3::/32", networks[1].Subnet6[0].Subnet)
	require.Equal(t, "3000:4::/32", networks[1].Subnet6[1].Subnet)
}

// Test that an error is returned when the shared-networks list
// is malformed, i.e., does not match the specified structure.
func TestDecodeMalformedSharedNetworks(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": 1234
                }
            ]
        }
    }`

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	networks := []struct {
		Name string
	}{}
	err = cfg.DecodeSharedNetworks(&networks)
	require.Error(t, err)
}
