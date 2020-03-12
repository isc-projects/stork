package dbmodel

import (
	require "github.com/stretchr/testify/require"

	"testing"
)

// Returns test Kea configuration lacking hooks libraries configurations.
func getTestConfigWithoutHooks(t *testing.T) *KeaConfig {
	configStr := `{
        "Dhcp4": {
            "valid-lifetime": 1000
        }
    }`

	cfg, err := NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration with empty list of hooks libraries.
func getTestConfigEmptyHooks(t *testing.T) *KeaConfig {
	configStr := `{
        "Dhcp4": {
            "valid-lifetime": 1000,
            "high-availability": [ ]
        }
    }`

	cfg, err := NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration including two hooks libraries.
func getTestConfigWithHooks(t *testing.T) *KeaConfig {
	configStr := `{
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_lease_cmds.so"
                },
                {
                    "library": "/usr/lib/kea/libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "load-balancing",
                            "heartbeat-delay": 10000,
                            "max-response-delay": 10000,
                            "max-ack-delay": 5000,
                            "max-unacked-clients": 5,
                            "peers": [{
                                "name": "server1",
                                "url": "http://192.168.56.33:8000/",
                                "role": "primary",
                                "auto-failover": true
                            }, {
                                "name": "server2",
                                "url": "http://192.168.56.66:8000/",
                                "role": "secondary",
                                "auto-failover": true
                            }, {
                                "name": "server3",
                                "url": "http://192.168.56.99:8000/",
                                "role": "backup",
                                "auto-failover": false
                            }]
                        }]
                    }
                }
            ]
        }
    }`

	cfg, err := NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test HA configuration which lacks some parameters.
func getTestMinimalHAConfig(t *testing.T) *KeaConfig {
	configStr := `{
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "load-balancing",
                            "heartbeat-delay": 10000
                        }]
                    }
                }
            ]
        }
    }`

	cfg, err := NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Tests that the configuration root key can be found.
func TestGetRootName(t *testing.T) {
	cfg, err := NewKeaConfigFromJSON(`
        {
            "Logging": { },
            "Dhcp4": {
                "subnet4": [ ]
            }
        }
    `)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	root, ok := cfg.GetRootName()
	require.True(t, ok)
	require.Equal(t, "Dhcp4", root)
}

// Tests that Logging key is ignored as non-root key.
func TestGetRootNameNoRoot(t *testing.T) {
	cfg, err := NewKeaConfigFromJSON(`
        {
            "Logging": { }
        }
    `)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	root, ok := cfg.GetRootName()
	require.False(t, ok)
	require.Empty(t, root)
}

// Test that the subnet ID can be extracted from the Kea configuration for
// an IPv4 subnet having specified prefix.
func TestGetLocalIPv4SubnetID(t *testing.T) {
	cfg := getTestConfigWithIPv4Subnets(t)

	require.EqualValues(t, 567, cfg.GetLocalSubnetID("10.1.0.000/16"))
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

// Test that a list of configurations of all hooks libraries can be retrieved
// from the Kea configuration.
func TestGetHooksLibraries(t *testing.T) {
	cfg := getTestConfigWithHooks(t)

	libraries := cfg.GetHooksLibraries()
	require.Len(t, libraries, 2)

	library := libraries[0]
	require.Equal(t, "/usr/lib/kea/libdhcp_lease_cmds.so", library.Library)
	require.Empty(t, library.Parameters)

	library = libraries[1]
	require.Equal(t, "/usr/lib/kea/libdhcp_ha.so", library.Library)
	require.Contains(t, library.Parameters, "high-availability")
}

// Test that empty list of hooks is returned when the configuration lacks
// hooks-libraries parameter.
func TestGetHooksLibrariesNoHooks(t *testing.T) {
	cfg := getTestConfigWithoutHooks(t)

	libraries := cfg.GetHooksLibraries()
	require.Empty(t, libraries)
}

// Test that configuration of the selected hooks library can be retrieved
// from the Kea configuration.
func TestGetHooksLibrary(t *testing.T) {
	cfg := getTestConfigWithHooks(t)

	path, params, ok := cfg.GetHooksLibrary("libdhcp_ha")
	require.True(t, ok)
	require.Equal(t, "/usr/lib/kea/libdhcp_ha.so", path)
	require.Contains(t, params, "high-availability")
}

// Test the case when Kea configuration contains empty hooks list and
// one of the hooks is requested by name.
func TestGetHooksLibraryEmptyHooks(t *testing.T) {
	cfg := getTestConfigEmptyHooks(t)

	path, params, ok := cfg.GetHooksLibrary("libdhcp_ha")
	require.False(t, ok)
	require.Empty(t, path)
	require.Empty(t, params)
}

// Test that the configuration of the HA hooks library can be retrieved
// and parsed.
func TestGetHAHooksLibrary(t *testing.T) {
	cfg := getTestConfigWithHooks(t)

	path, params, ok := cfg.GetHAHooksLibrary()
	require.True(t, ok)

	require.NotNil(t, params.ThisServerName)
	require.NotNil(t, params.Mode)
	require.NotNil(t, params.HeartbeatDelay)
	require.NotNil(t, params.MaxResponseDelay)
	require.NotNil(t, params.MaxAckDelay)
	require.NotNil(t, params.MaxUnackedClients)

	require.Equal(t, "/usr/lib/kea/libdhcp_ha.so", path)
	require.Equal(t, "server1", *params.ThisServerName)
	require.Equal(t, "load-balancing", *params.Mode)
	require.Equal(t, 10000, *params.HeartbeatDelay)
	require.Equal(t, 10000, *params.MaxResponseDelay)
	require.Equal(t, 5000, *params.MaxAckDelay)
	require.Equal(t, 5, *params.MaxUnackedClients)
	require.Len(t, params.Peers, 3)

	peersFound := make(map[string]bool)

	for _, peer := range params.Peers {
		require.NotNil(t, peer.Name)
		require.NotNil(t, peer.URL)
		require.NotNil(t, peer.Role)
		require.NotNil(t, peer.AutoFailover)
		peersFound[*peer.Name] = true
		switch *peer.Name {
		case "server1":
			require.Equal(t, "http://192.168.56.33:8000/", *peer.URL)
			require.Equal(t, "primary", *peer.Role)
			require.True(t, *peer.AutoFailover)
		case "server2":
			require.Equal(t, "http://192.168.56.66:8000/", *peer.URL)
			require.Equal(t, "secondary", *peer.Role)
			require.True(t, *peer.AutoFailover)
		case "server3":
			require.Equal(t, "http://192.168.56.99:8000/", *peer.URL)
			require.Equal(t, "backup", *peer.Role)
			require.False(t, *peer.AutoFailover)
		}
	}

	// There should be three distinct peers found.
	require.Len(t, peersFound, 3)
}

// Test that parameters not included in the HA configuration are set
// to nil.
func TestGetHAHooksLibraryOptionals(t *testing.T) {
	cfg := getTestMinimalHAConfig(t)

	_, params, ok := cfg.GetHAHooksLibrary()
	require.True(t, ok)

	// These parameters were specified.
	require.NotNil(t, params.ThisServerName)
	require.NotNil(t, params.Mode)
	require.NotNil(t, params.HeartbeatDelay)

	// These parameters weren't.
	require.Nil(t, params.MaxResponseDelay)
	require.Nil(t, params.MaxAckDelay)
	require.Nil(t, params.MaxUnackedClients)
}

// Test the case when Kea configuration contains empty hooks list and
// HA hooks library is requested.
func TestGetHAHooksLibraryEmptyHooks(t *testing.T) {
	cfg := getTestConfigEmptyHooks(t)

	path, params, ok := cfg.GetHAHooksLibrary()
	require.False(t, ok)
	require.Empty(t, path)
	require.Empty(t, params.ThisServerName)
}

// Checks if the HA peer structure validation works as expected.
func TestPeerParametersSet(t *testing.T) {
	p := Peer{}
	require.False(t, p.IsSet())

	name := "server1"
	p.Name = &name
	require.False(t, p.IsSet())

	url := "http://example.org/"
	p.URL = &url
	require.False(t, p.IsSet())

	role := "primary"
	p.Role = &role
	require.True(t, p.IsSet())

	autoFailover := true
	p.AutoFailover = &autoFailover
	require.True(t, p.IsSet())
}

// Checks if the HA configuration validation works as expected.
func TestHAConfigParametersSet(t *testing.T) {
	cfg := KeaConfigHA{}

	require.False(t, cfg.IsSet())

	thisServerName := "server1"
	cfg.ThisServerName = &thisServerName
	require.False(t, cfg.IsSet())

	haMode := "load-balancing"
	cfg.Mode = &haMode
	require.True(t, cfg.IsSet())

	p := Peer{}
	cfg.Peers = append(cfg.Peers, p)
	require.False(t, cfg.IsSet())
}

// Verifies that the shared network instance can be created by parsing
// Kea configuration.
func TestNewSharedNetworkFromKea(t *testing.T) {
	rawNetwork := map[string]interface{}{
		"name": "foo",
		"subnet6": []map[string]interface{}{
			{
				"id":     1,
				"subnet": "2001:db8:2::/64",
			},
			{
				"id":     2,
				"subnet": "2001:db8:1::/64",
			},
		},
	}

	parsedNetwork, err := NewSharedNetworkFromKea(&rawNetwork, 6)
	require.NoError(t, err)
	require.NotNil(t, parsedNetwork)
	require.Equal(t, "foo", parsedNetwork.Name)
	require.EqualValues(t, 6, parsedNetwork.Family)
	require.Len(t, parsedNetwork.Subnets, 2)

	require.Zero(t, parsedNetwork.Subnets[0].ID)
	require.Equal(t, "2001:db8:2::/64", parsedNetwork.Subnets[0].Prefix)
	require.Zero(t, parsedNetwork.Subnets[1].ID)
	require.Equal(t, "2001:db8:1::/64", parsedNetwork.Subnets[1].Prefix)
}

// Test that subnets within a shared network are verified to catch
// those which family is not matching with the shared network family.
func TestNewSharedNetworkFromKeaFamilyClash(t *testing.T) {
	rawNetwork := map[string]interface{}{
		"name": "foo",
		"subnet4": []map[string]interface{}{
			{
				"id":     1,
				"subnet": "192.0.2.0/24",
			},
		},
		"subnet6": []map[string]interface{}{
			{
				"id":     2,
				"subnet": "2001:db8:1::/64",
			},
		},
	}

	parsedNetwork, err := NewSharedNetworkFromKea(&rawNetwork, 4)
	require.Error(t, err)
	require.Nil(t, parsedNetwork)
}

// Verifies that the subnet instance can be created by parsing Kea
// configuration.
func TestNewSubnetFromKea(t *testing.T) {
	rawSubnet := map[string]interface{}{
		"id":     1,
		"subnet": "2001:db8:1::/64",
		"pools": []interface{}{
			map[string]interface{}{
				"pool": "2001:db8:1:1::/120",
			},
		},
		"pd-pools": []interface{}{
			map[string]interface{}{
				"prefix":        "2001:db8:1:1::",
				"prefix-len":    96,
				"delegated-len": 120,
			},
		},
		"reservations": []interface{}{
			map[string]interface{}{
				"duid": "01:02:03:04:05:06",
				"ip-addresses": []interface{}{
					"2001:db8:1::1",
					"2001:db8:1::2",
				},
				"prefixes": []interface{}{
					"3000:1::/64",
					"3000:2::/64",
				},
			},
		},
	}

	parsedSubnet, err := NewSubnetFromKea(&rawSubnet)
	require.NoError(t, err)
	require.NotNil(t, parsedSubnet)
	require.Zero(t, parsedSubnet.ID)
	require.Equal(t, "2001:db8:1::/64", parsedSubnet.Prefix)
	require.Len(t, parsedSubnet.AddressPools, 1)
	require.Equal(t, "2001:db8:1:1::", parsedSubnet.AddressPools[0].LowerBound)
	require.Equal(t, "2001:db8:1:1::ff", parsedSubnet.AddressPools[0].UpperBound)

	require.Len(t, parsedSubnet.PrefixPools, 1)
	require.Equal(t, "2001:db8:1:1::/96", parsedSubnet.PrefixPools[0].Prefix)
	require.EqualValues(t, 120, parsedSubnet.PrefixPools[0].DelegatedLen)

	require.Len(t, parsedSubnet.Hosts, 1)
	require.Len(t, parsedSubnet.Hosts[0].HostIdentifiers, 1)
	require.Equal(t, "duid", parsedSubnet.Hosts[0].HostIdentifiers[0].Type)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6}, parsedSubnet.Hosts[0].HostIdentifiers[0].Value)

	require.Len(t, parsedSubnet.Hosts[0].IPReservations, 4)
	require.Equal(t, "2001:db8:1::1", parsedSubnet.Hosts[0].IPReservations[0].Address)
	require.Equal(t, "2001:db8:1::2", parsedSubnet.Hosts[0].IPReservations[1].Address)
	require.Equal(t, "3000:1::/64", parsedSubnet.Hosts[0].IPReservations[2].Address)
	require.Equal(t, "3000:2::/64", parsedSubnet.Hosts[0].IPReservations[3].Address)
}
