package dbmodel

import (
	"testing"

	require "github.com/stretchr/testify/require"

	keaconfig "isc.org/stork/appcfg/kea"
)

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
			map[string]interface{}{
				"hw-address": "01:01:01:01:01:01",
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
	require.Len(t, parsedSubnet.Hosts[0].HostIdentifiers, 2)
	require.Equal(t, "duid", parsedSubnet.Hosts[0].HostIdentifiers[0].Type)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6}, parsedSubnet.Hosts[0].HostIdentifiers[0].Value)
	require.Equal(t, "hw-address", parsedSubnet.Hosts[0].HostIdentifiers[1].Type)
	require.Equal(t, []byte{1, 1, 1, 1, 1, 1}, parsedSubnet.Hosts[0].HostIdentifiers[1].Value)

	require.Len(t, parsedSubnet.Hosts[0].IPReservations, 4)
	require.Equal(t, "2001:db8:1::1", parsedSubnet.Hosts[0].IPReservations[0].Address)
	require.Equal(t, "2001:db8:1::2", parsedSubnet.Hosts[0].IPReservations[1].Address)
	require.Equal(t, "3000:1::/64", parsedSubnet.Hosts[0].IPReservations[2].Address)
	require.Equal(t, "3000:2::/64", parsedSubnet.Hosts[0].IPReservations[3].Address)
}

// Verifies that the host instance can be created by parsing Kea
// configuration.
func TestNewHostFromKea(t *testing.T) {
	rawHost := map[string]interface{}{
		"duid": "01:02:03:04",
		"ip-addresses": []interface{}{
			"2001:db8:1::1",
			"2001:db8:1::2",
		},
		"prefixes": []interface{}{
			"3000:1::/64",
			"3000:2::/64",
		},
		"hostname": "hostname.example.org",
	}

	parsedHost, err := NewHostFromKea(&rawHost)
	require.NoError(t, err)
	require.NotNil(t, parsedHost)

	require.Len(t, parsedHost.HostIdentifiers, 1)
	require.Equal(t, "duid", parsedHost.HostIdentifiers[0].Type)
	require.Len(t, parsedHost.IPReservations, 4)
	require.Equal(t, "2001:db8:1::1", parsedHost.IPReservations[0].Address)
	require.Equal(t, "2001:db8:1::2", parsedHost.IPReservations[1].Address)
	require.Equal(t, "3000:1::/64", parsedHost.IPReservations[2].Address)
	require.Equal(t, "3000:2::/64", parsedHost.IPReservations[3].Address)
	require.Equal(t, "hostname.example.org", parsedHost.Hostname)
}

// Test that log targets can be created from parsed Kea logger config.
func TestNewLogTargetsFromKea(t *testing.T) {
	logger := keaconfig.Logger{
		Name: "logger-name",
		OutputOptions: []keaconfig.LoggerOutputOptions{
			{
				Output: "stdout",
			},
			{
				Output: "/tmp/log",
			},
		},
		Severity: "DEBUG",
	}

	targets := NewLogTargetsFromKea(logger)
	require.Len(t, targets, 2)
	require.Equal(t, "logger-name", targets[0].Name)
	require.Equal(t, "stdout", targets[0].Output)
	require.Equal(t, "debug", targets[0].Severity)
	require.Equal(t, "logger-name", targets[1].Name)
	require.Equal(t, "/tmp/log", targets[1].Output)
	require.Equal(t, "debug", targets[1].Severity)
}
