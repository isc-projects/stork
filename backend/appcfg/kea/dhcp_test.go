package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that configured hook libraries are retrieved from a DHCPv4
// server configuration.
func TestGetDHCPv4HookLibraries(t *testing.T) {
	cfg := &DHCPv4Config{
		CommonDHCPConfig: CommonDHCPConfig{
			HookLibraries: []HookLibrary{
				{
					Library: "libdhcp_lease_cmds",
				},
			},
		},
	}
	hooks := cfg.GetHookLibraries()
	require.Len(t, hooks, 1)
	require.Equal(t, "libdhcp_lease_cmds", hooks[0].Library)
}

// Test that the configured loggers are retrieved from a DHCPv4
// server configuration.
func TestGetDHCPv4Loggers(t *testing.T) {
	cfg := &DHCPv4Config{
		CommonDHCPConfig: CommonDHCPConfig{
			Loggers: []Logger{
				{
					Name:       "kea-dhcp4",
					Severity:   "DEBUG",
					DebugLevel: 99,
				},
			},
		},
	}
	libraries := cfg.GetLoggers()
	require.Len(t, libraries, 1)
	require.Equal(t, "kea-dhcp4", libraries[0].Name)
	require.Equal(t, "DEBUG", libraries[0].Severity)
	require.EqualValues(t, 99, libraries[0].DebugLevel)
}

// Test that common DHCPv4 configuration parameters are retrieved.
func TestGetDHCPv4CommonConfig(t *testing.T) {
	cfg := &DHCPv4Config{
		CommonDHCPConfig: CommonDHCPConfig{
			ClientClasses: []ClientClass{
				{
					Name: "foo",
				},
			},
		},
	}
	common := cfg.GetCommonDHCPConfig()
	require.Len(t, common.ClientClasses, 1)
	require.Equal(t, "foo", common.ClientClasses[0].Name)
}

// Test getting shared networks from a DHCPv4 configuration.
func TestGetDHCPv4SharedNetworks(t *testing.T) {
	cfg := &DHCPv4Config{
		SharedNetworks: []SharedNetwork4{
			{
				Name: "foo",
				Subnet4: []Subnet4{
					{
						MandatorySubnetParameters: MandatorySubnetParameters{
							ID:     1,
							Subnet: "192.0.2.0/24",
						},
					},
				},
			},
		},
		Subnet4: []Subnet4{
			{
				MandatorySubnetParameters: MandatorySubnetParameters{
					ID:     2,
					Subnet: "192.0.3.0/24",
				},
			},
		},
	}

	t.Run("without root subnets", func(t *testing.T) {
		sharedNetworks := cfg.GetSharedNetworks(false)
		require.Len(t, sharedNetworks, 1)
		require.Equal(t, "foo", sharedNetworks[0].GetName())
		require.Len(t, sharedNetworks[0].GetSubnets(), 1)
		require.EqualValues(t, 1, sharedNetworks[0].GetSubnets()[0].GetID())
		require.Equal(t, "192.0.2.0/24", sharedNetworks[0].GetSubnets()[0].GetPrefix())
	})

	t.Run("with root subnets", func(t *testing.T) {
		sharedNetworks := cfg.GetSharedNetworks(true)
		require.Len(t, sharedNetworks, 2)
		require.Equal(t, "foo", sharedNetworks[0].GetName())
		require.Len(t, sharedNetworks[0].GetSubnets(), 1)
		require.EqualValues(t, 1, sharedNetworks[0].GetSubnets()[0].GetID())
		require.Equal(t, "192.0.2.0/24", sharedNetworks[0].GetSubnets()[0].GetPrefix())
		require.Empty(t, sharedNetworks[1].GetName())
		require.Len(t, sharedNetworks[1].GetSubnets(), 1)
		require.EqualValues(t, 2, sharedNetworks[1].GetSubnets()[0].GetID())
		require.Equal(t, "192.0.3.0/24", sharedNetworks[1].GetSubnets()[0].GetPrefix())
	})
}

// Test getting subnets from the DHCPv4 server configuration.
func TestGetDHCPv4Subnets(t *testing.T) {
	cfg := &DHCPv4Config{
		Subnet4: []Subnet4{
			{
				MandatorySubnetParameters: MandatorySubnetParameters{
					ID:     2,
					Subnet: "192.0.2.0/24",
				},
			},
		},
	}
	subnets := cfg.GetSubnets()
	require.Len(t, subnets, 1)
	require.EqualValues(t, 2, subnets[0].GetID())
	require.EqualValues(t, "192.0.2.0/24", subnets[0].GetPrefix())
}

// Test getting configured hook libraries from the DHCPv6 server configuration.
func TestGetDHCPv6HookLibraries(t *testing.T) {
	cfg := &DHCPv6Config{
		CommonDHCPConfig: CommonDHCPConfig{
			HookLibraries: []HookLibrary{
				{
					Library: "libdhcp_ha",
				},
			},
		},
	}
	hooks := cfg.GetHookLibraries()
	require.Len(t, hooks, 1)
	require.Equal(t, "libdhcp_ha", hooks[0].Library)
}

// Test getting configured loggers from the DHCPv6 server configuration.
func TestGetDHCPv6Loggers(t *testing.T) {
	cfg := &DHCPv6Config{
		CommonDHCPConfig: CommonDHCPConfig{
			Loggers: []Logger{
				{
					Name:       "kea-dhcp6",
					Severity:   "INFO",
					DebugLevel: 65,
				},
			},
		},
	}
	libraries := cfg.GetLoggers()
	require.Len(t, libraries, 1)

	require.Equal(t, "kea-dhcp6", libraries[0].Name)
	require.Equal(t, "INFO", libraries[0].Severity)
	require.EqualValues(t, 65, libraries[0].DebugLevel)
}

// Test getting common DHCPv6 configuration parameters.
func TestGetDHCPv6CommonConfig(t *testing.T) {
	cfg := &DHCPv6Config{
		CommonDHCPConfig: CommonDHCPConfig{
			ClientClasses: []ClientClass{
				{
					Name: "bar",
				},
			},
		},
	}
	common := cfg.GetCommonDHCPConfig()
	require.Len(t, common.ClientClasses, 1)
	require.Equal(t, "bar", common.ClientClasses[0].Name)
}

// Test getting shared networks from the DHCPv6 server configuration.
func TestGetDHCPv6SharedNetworks(t *testing.T) {
	cfg := &DHCPv6Config{
		SharedNetworks: []SharedNetwork6{
			{
				Name: "foo",
				Subnet6: []Subnet6{
					{
						MandatorySubnetParameters: MandatorySubnetParameters{
							ID:     1,
							Subnet: "2001:db8:1::/64",
						},
					},
				},
			},
		},
		Subnet6: []Subnet6{
			{
				MandatorySubnetParameters: MandatorySubnetParameters{
					ID:     2,
					Subnet: "2001:db8:2::/64",
				},
			},
		},
	}

	t.Run("without root subnets", func(t *testing.T) {
		sharedNetworks := cfg.GetSharedNetworks(false)
		require.Len(t, sharedNetworks, 1)
		require.Equal(t, "foo", sharedNetworks[0].GetName())
		require.Len(t, sharedNetworks[0].GetSubnets(), 1)
		require.EqualValues(t, 1, sharedNetworks[0].GetSubnets()[0].GetID())
		require.Equal(t, "2001:db8:1::/64", sharedNetworks[0].GetSubnets()[0].GetPrefix())
	})

	t.Run("with root subnets", func(t *testing.T) {
		sharedNetworks := cfg.GetSharedNetworks(true)
		require.Len(t, sharedNetworks, 2)
		require.Equal(t, "foo", sharedNetworks[0].GetName())
		require.Len(t, sharedNetworks[0].GetSubnets(), 1)
		require.EqualValues(t, 1, sharedNetworks[0].GetSubnets()[0].GetID())
		require.Equal(t, "2001:db8:1::/64", sharedNetworks[0].GetSubnets()[0].GetPrefix())
		require.Empty(t, sharedNetworks[1].GetName())
		require.Len(t, sharedNetworks[1].GetSubnets(), 1)
		require.EqualValues(t, 2, sharedNetworks[1].GetSubnets()[0].GetID())
		require.Equal(t, "2001:db8:2::/64", sharedNetworks[1].GetSubnets()[0].GetPrefix())
	})
}

// Test getting subnets from the DHCPv6 server configuration.
func TestGetDHCPv6Subnets(t *testing.T) {
	cfg := &DHCPv6Config{
		Subnet6: []Subnet6{
			{
				MandatorySubnetParameters: MandatorySubnetParameters{
					ID:     2,
					Subnet: "2001:db8:2::/64",
				},
			},
		},
	}

	subnets := cfg.GetSubnets()
	require.Len(t, subnets, 1)
	require.EqualValues(t, 2, subnets[0].GetID())
	require.EqualValues(t, "2001:db8:2::/64", subnets[0].GetPrefix())
}
