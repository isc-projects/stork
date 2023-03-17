package keaconfig

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"testing"

	require "github.com/stretchr/testify/require"
)

//go:embed keaconfig_test_dhcp4_all_keys.json
var allKeysDHCPv4 []byte

//go:embed keaconfig_test_dhcp6_all_keys.json
var allKeysDHCPv6 []byte

// Returns test Kea configuration with empty list of hooks libraries.
func getTestConfigEmptyHooks(t *testing.T) *Config {
	configStr := `{
        "Dhcp4": {
            "valid-lifetime": 1000,
            "high-availability": [ ]
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration including multiple IPv4 subnets.
func getTestConfigWithIPv4Subnets(t *testing.T) *Config {
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
                            "subnet": "10.2.1.0/16"
                        }
                    ],
					"option-data": [
						{
							"code": 5,
							"name": "name-servers",
							"space": "dhcp4",
							"data": "192.0.2.1"
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

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns a configuration including global DHCP reservations.
func getTestConfigWithGlobalReservations(t *testing.T, rootName string) *Config {
	configStr := `{
		"%s": {
			"reservations": [
				{
					"hw-address": "01:02:03:04:05:06",
					"hostname": "foo.example.org"
				},
				{
					"duid": "01:01:01:01",
					"client-classes": ["bar"]
				}
			]
		}
	}`
	cfg, err := NewConfig(fmt.Sprintf(configStr, rootName))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns a configuration with loggers.
func getTestConfigWithLoggers(t *testing.T, rootName string) *Config {
	configStr := `{
        "%s": {
            "loggers": [
                {
                    "name": "kea-dhcp4",
                    "output_options": [
                        {
                            "output": "stdout"
                        }
                    ],
                    "severity": "WARN"
                },
                {
                    "name": "kea-dhcp4.bad-packets",
                    "output_options": [
                        {
                            "output": "/tmp/badpackets.log"
                        }
                    ],
                    "severity": "DEBUG",
                    "debuglevel": 99
                }
            ]
        }
    }`

	configStr = fmt.Sprintf(configStr, rootName)
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	return cfg
}

// Returns test Kea configuration including multiple IPv6 subnets.
func getTestConfigWithIPv6Subnets(t *testing.T) *Config {
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
                    ],
					"option-data": [
						{
							"code": 33,
							"name": "bcmcs-server-dns",
							"data": "foo.example.org",
							"space": "dhcp6"
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
                    "subnet": "2001:db8:1:0::/64"
                },
                {
                    "id": 234,
                    "subnet": "2001:db8:2::/64",
                    "pd-pools": [
                        {
                            "prefix": "3000::/16",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ]
                },
                {
                    "id": 345,
                    "subnet": "2001:db8:3::/64"
                }
            ]
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Test that Kea DHCPv4 configuration is recognised and parsed.
func TestDecodeDHCPv4(t *testing.T) {
	var config Config
	err := json.Unmarshal(allKeysDHCPv4, &config)
	require.NoError(t, err)

	require.NotNil(t, config.DHCPv4Config)

	marshalled, err := json.Marshal(config)
	require.NoError(t, err)

	require.JSONEq(t, string(allKeysDHCPv4), string(marshalled))
}

// Test that Kea DHCPv6 configuration is recognised and parsed.
func TestDecodeDHCPv6(t *testing.T) {
	var config Config
	err := json.Unmarshal(allKeysDHCPv6, &config)
	require.NoError(t, err)

	require.NotNil(t, config.DHCPv6Config)

	marshalled, err := json.Marshal(config)
	require.NoError(t, err)

	require.JSONEq(t, string(allKeysDHCPv6), string(marshalled))
}

// Tests that the configuration can contain comments.
func TestNewConfigWithComments(t *testing.T) {
	configStr := `{
		"Dhcp4": {
			// A comment.
		}
	}`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

// Test instantiating a new config from a map.
func TestNewConfigFromMap(t *testing.T) {
	cfgMap := &map[string]any{
		"Dhcp4": map[string]any{
			"subnet4": []any{
				map[string]any{
					"id":     123,
					"subnet": "192.0.2.0/24",
				},
			},
		},
	}
	cfg := NewConfigFromMap(cfgMap)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Raw)
	require.NotNil(t, cfg.DHCPv4Config)
	subnets := cfg.GetSubnets()
	require.Len(t, subnets, 1)
	require.EqualValues(t, 123, subnets[0].GetID())
	require.Equal(t, "192.0.2.0/24", subnets[0].GetPrefix())
}

// Test that DHCPv4 config is returned as a common data accessor.
func TestGetCommonConfigAccessorDHCPv4(t *testing.T) {
	cfg := &Config{
		DHCPv4Config: &DHCPv4Config{},
	}
	require.NotNil(t, cfg)
	accessor := cfg.getCommonConfigAccessor()
	require.NotNil(t, accessor)
	require.Equal(t, cfg.DHCPv4Config, accessor)
}

// Test that DHCPv6 config is returned as a common data accessor.
func TestGetCommonConfigAccessorDHCPv6(t *testing.T) {
	cfg := &Config{
		DHCPv6Config: &DHCPv6Config{},
	}
	require.NotNil(t, cfg)
	accessor := cfg.getCommonConfigAccessor()
	require.NotNil(t, accessor)
	require.Equal(t, cfg.DHCPv6Config, accessor)
}

// Test that Control Agent config is returned as a common data accessor.
func TestGetCommonConfigAccessorCtrlAgent(t *testing.T) {
	cfg := &Config{
		CtrlAgentConfig: &CtrlAgentConfig{},
	}
	require.NotNil(t, cfg)
	accessor := cfg.getCommonConfigAccessor()
	require.NotNil(t, accessor)
	require.Equal(t, cfg.CtrlAgentConfig, accessor)
}

// Test that D2 config is returned as a common data accessor.
func TestGetCommonConfigAccessorD2(t *testing.T) {
	cfg := &Config{
		D2Config: &D2Config{},
	}
	require.NotNil(t, cfg)
	accessor := cfg.getCommonConfigAccessor()
	require.NotNil(t, accessor)
	require.Equal(t, cfg.D2Config, accessor)
}

// Verifies that a list of loggers is parsed correctly for a daemon.
func TestGetLoggers(t *testing.T) {
	cfg := getTestConfigWithLoggers(t, "Dhcp4")
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.DHCPv4Config)

	loggers := cfg.GetLoggers()
	require.Len(t, loggers, 2)

	require.Equal(t, "kea-dhcp4", loggers[0].Name)
	require.Len(t, loggers[0].OutputOptions, 1)
	require.Equal(t, "stdout", loggers[0].OutputOptions[0].Output)
	require.Equal(t, "WARN", loggers[0].Severity)
	require.Zero(t, loggers[0].DebugLevel)

	require.Equal(t, "kea-dhcp4.bad-packets", loggers[1].Name)
	require.Len(t, loggers[1].OutputOptions, 1)
	require.Equal(t, "/tmp/badpackets.log", loggers[1].OutputOptions[0].Output)
	require.Equal(t, "DEBUG", loggers[1].Severity)
	require.Equal(t, 99, loggers[1].DebugLevel)
}

// Verifies that a list of loggers is parsed correctly for a daemon.
func TestGetControlSockets(t *testing.T) {
	configStr := `{
        "Control-agent": {
            "control-sockets": {
                "dhcp4": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v4"
                },
                "dhcp6": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v6"
                },
                "d2": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-d2"
                }
            }
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.CtrlAgentConfig)

	sockets := cfg.GetControlSockets()

	require.NotNil(t, sockets.D2)
	require.Equal(t, "unix", sockets.D2.SocketType)
	require.Equal(t, "/path/to/the/unix/socket-d2", sockets.D2.SocketName)

	require.NotNil(t, sockets.Dhcp4)
	require.Equal(t, "unix", sockets.Dhcp4.SocketType)
	require.Equal(t, "/path/to/the/unix/socket-v4", sockets.Dhcp4.SocketName)

	require.NotNil(t, sockets.Dhcp6)
	require.Equal(t, "unix", sockets.Dhcp6.SocketType)
	require.Equal(t, "/path/to/the/unix/socket-v6", sockets.Dhcp6.SocketName)

	require.Nil(t, sockets.NetConf)
}

// Verifies that the list of daemons for which control sockets are specified
// is returned correctly.
func TestConfiguredDaemonNames(t *testing.T) {
	// Initialize all 4 supported sockets.
	configStr := `{
        "Control-agent": {
            "control-sockets": {
                "dhcp4": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v4"
                },
                "dhcp6": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v6"
                },
                "d2": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-d2"
                },
                "netconf": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-netconf"
                }
            }
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	sockets := cfg.GetControlSockets()

	names := sockets.GetConfiguredDaemonNames()
	require.Len(t, names, 4)

	require.Contains(t, names, "dhcp4")
	require.Contains(t, names, "dhcp6")
	require.Contains(t, names, "d2")
	require.Contains(t, names, "netconf")

	// Reduce the number of configured sockets.
	configStr = `{
        "Control-agent": {
            "control-sockets": {
                "dhcp4": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v4"
                },
                "d2": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-d2"
                }
            }
        }
    }`

	cfg, err = NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	sockets = cfg.GetControlSockets()

	// This time only two sockets have been configured.
	names = sockets.GetConfiguredDaemonNames()
	require.Len(t, names, 2)

	require.Contains(t, names, "dhcp4")
	require.Contains(t, names, "d2")
}

// Test that all database connections configurations are parsed and returned
// correctly: lease-database, hosts-database, hosts-databases, config-databases
// and forensic logging config.
func TestGetAllDatabases(t *testing.T) {
	// Create template configuration into which we will be inserting
	// different configurations in different tests.
	configTemplate := `{
        "Dhcp4": {
            %s
            %s
            %s
            "config-control": {
                %s
            },
            "hooks-libraries": [
                %s
            ]
        }
    }`
	leaseDatabase := `"lease-database": {
        "type": "mysql",
        "name": "kea-lease-mysql"
    },`
	hostsDatabase := `"hosts-database": {
        "type": "mysql",
        "name": "kea-hosts-mysql",
        "host": "mysql.example.org"
    },`
	hostsDatabases := `"hosts-databases": [
        {
            "type": "postgresql",
            "name": "kea-hosts-pgsql",
            "host": "localhost"
        },
        {
            "type": "mysql",
            "name": "kea-hosts-mysql"
        }
    ],`
	configDatabases := `"config-databases": [
        {
            "type": "mysql",
            "name": "kea-hosts-mysql"
        },
        {
            "type": "postgresql",
            "name": "kea-hosts-pgsql",
            "host": "localhost"
        }
    ]`
	legalConfig := `{
        "library": "/usr/lib/kea/libdhcp_legal_log.so",
        "parameters": {
            "path": "/tmp/legal_log.log"
        }
    }`

	// All configurations used together.
	t.Run("all configs present", func(t *testing.T) {
		configStr := fmt.Sprintf(configTemplate, leaseDatabase, hostsDatabase, hostsDatabases, configDatabases, legalConfig)
		cfg, err := NewConfig(configStr)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		databases := cfg.GetAllDatabases()
		require.NotNil(t, databases.Lease)
		require.Len(t, databases.Hosts, 1)
		require.Len(t, databases.Config, 2)
		require.NotNil(t, databases.Forensic)
	})

	// No database configuration.
	t.Run("no configs present", func(t *testing.T) {
		configStr := fmt.Sprintf(configTemplate, "", "", "", "", "")
		cfg, err := NewConfig(configStr)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		databases := cfg.GetAllDatabases()
		require.Nil(t, databases.Lease)
		require.Empty(t, databases.Hosts)
		require.Empty(t, databases.Config)
		require.Nil(t, databases.Forensic)
	})

	// lease-database
	t.Run("lease-database only", func(t *testing.T) {
		configStr := fmt.Sprintf(configTemplate, leaseDatabase, "", "", "", "")
		cfg, err := NewConfig(configStr)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		databases := cfg.GetAllDatabases()
		require.NotNil(t, databases.Lease)
		require.Empty(t, databases.Hosts)
		require.Empty(t, databases.Config)
		require.Nil(t, databases.Forensic)

		require.Empty(t, databases.Lease.Path)
		require.Equal(t, "mysql", databases.Lease.Type)
		require.Equal(t, "kea-lease-mysql", databases.Lease.Name)
		require.Equal(t, "localhost", databases.Lease.Host)
	})

	// hosts-database
	t.Run("hosts-database only", func(t *testing.T) {
		configStr := fmt.Sprintf(configTemplate, "", hostsDatabase, "", "", "")
		cfg, err := NewConfig(configStr)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		databases := cfg.GetAllDatabases()
		require.Nil(t, databases.Lease)
		require.Len(t, databases.Hosts, 1)
		require.Empty(t, databases.Config)
		require.Nil(t, databases.Forensic)

		require.Empty(t, databases.Hosts[0].Path)
		require.Equal(t, "mysql", databases.Hosts[0].Type)
		require.Equal(t, "kea-hosts-mysql", databases.Hosts[0].Name)
		require.Equal(t, "mysql.example.org", databases.Hosts[0].Host)
	})

	// hosts-databases
	t.Run("hosts-databases only", func(t *testing.T) {
		configStr := fmt.Sprintf(configTemplate, "", "", hostsDatabases, "", "")
		cfg, err := NewConfig(configStr)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		databases := cfg.GetAllDatabases()
		require.Nil(t, databases.Lease)
		require.Len(t, databases.Hosts, 2)
		require.Empty(t, databases.Config)
		require.Nil(t, databases.Forensic)

		require.Empty(t, databases.Hosts[0].Path)
		require.Equal(t, "postgresql", databases.Hosts[0].Type)
		require.Equal(t, "kea-hosts-pgsql", databases.Hosts[0].Name)
		require.Equal(t, "localhost", databases.Hosts[0].Host)

		require.Empty(t, databases.Hosts[1].Path)
		require.Equal(t, "mysql", databases.Hosts[1].Type)
		require.Equal(t, "kea-hosts-mysql", databases.Hosts[1].Name)
		require.Equal(t, "localhost", databases.Hosts[1].Host)
	})

	// config-databases
	t.Run("config-databases only", func(t *testing.T) {
		configStr := fmt.Sprintf(configTemplate, "", "", "", configDatabases, "")
		cfg, err := NewConfig(configStr)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		databases := cfg.GetAllDatabases()
		require.Nil(t, databases.Lease)
		require.Empty(t, databases.Hosts)
		require.Len(t, databases.Config, 2)
		require.Nil(t, databases.Forensic)

		require.Empty(t, databases.Config[0].Path)
		require.Equal(t, "mysql", databases.Config[0].Type)
		require.Equal(t, "kea-hosts-mysql", databases.Config[0].Name)
		require.Equal(t, "localhost", databases.Config[0].Host)

		require.Empty(t, databases.Config[1].Path)
		require.Equal(t, "postgresql", databases.Config[1].Type)
		require.Equal(t, "kea-hosts-pgsql", databases.Config[1].Name)
		require.Equal(t, "localhost", databases.Config[1].Host)
	})

	// legal logging hook
	t.Run("legal logging only", func(t *testing.T) {
		configStr := fmt.Sprintf(configTemplate, "", "", "", "", legalConfig)
		cfg, err := NewConfig(configStr)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		databases := cfg.GetAllDatabases()
		require.Nil(t, databases.Lease)
		require.Empty(t, databases.Hosts)
		require.Empty(t, databases.Config)
		require.NotNil(t, databases.Forensic)

		require.Equal(t, "/tmp/legal_log.log", databases.Forensic.Path)
		require.Empty(t, databases.Forensic.Type)
		require.Empty(t, databases.Forensic.Name)
		require.Empty(t, databases.Forensic.Host)
	})
}

// Test parsing global reservation modes when all of them
// are explicitly set.
func TestGetGlobalReservationModesEnableAll(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservations-global": true,
            "reservations-in-subnet": true,
            "reservations-out-of-pool": true,
            "reservation-mode": "disabled"
        }
    }`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationParameters()
	require.NotNil(t, modes)
	// The new settings take precedence over the deprecated
	// reservation-mode setting.
	val, set := modes.IsGlobal()
	require.True(t, val)
	require.True(t, set)
	val, set = modes.IsInSubnet()
	require.True(t, val)
	require.True(t, set)
	val, set = modes.IsOutOfPool()
	require.True(t, val)
	require.True(t, set)
}

// Test parsing global reservation modes when all of them
// are explicitly disabled.
func TestGetGlobalReservationModesDisableAll(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservations-global": false,
            "reservations-in-subnet": false,
            "reservations-out-of-pool": false,
            "reservation-mode": "out-of-pool"
        }
    }`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationParameters()
	require.NotNil(t, modes)
	// The new settings take precedence over the deprecated
	// reservation-mode setting.
	val, set := modes.IsGlobal()
	require.False(t, val)
	require.True(t, set)
	val, set = modes.IsInSubnet()
	require.False(t, val)
	require.True(t, set)
	val, set = modes.IsOutOfPool()
	require.False(t, val)
	require.True(t, set)
}

// Test parsing the deprecated reservation-mode set to disabled.
func TestGetGlobalReservationModesDeprecatedDisabled(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservation-mode": "disabled"
        }
    }`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationParameters()
	require.NotNil(t, modes)
	val, set := modes.IsGlobal()
	require.False(t, val)
	require.True(t, set)
	val, set = modes.IsInSubnet()
	require.False(t, val)
	require.True(t, set)
	val, set = modes.IsOutOfPool()
	require.False(t, val)
	require.True(t, set)
}

// Test parsing the deprecated reservation-mode set to global.
func TestGetGlobalReservationModesDeprecatedGlobal(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservation-mode": "global"
        }
    }`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationParameters()
	require.NotNil(t, modes)
	val, set := modes.IsGlobal()
	require.True(t, val)
	require.True(t, set)
	val, set = modes.IsInSubnet()
	require.False(t, val)
	require.True(t, set)
	val, set = modes.IsOutOfPool()
	require.False(t, val)
	require.True(t, set)
}

// Test parsing the deprecated reservation-mode set to out-of-pool.
func TestGetGlobalReservationModesDeprecatedOutOfPool(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservation-mode": "out-of-pool"
        }
    }`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationParameters()
	require.NotNil(t, modes)
	val, set := modes.IsGlobal()
	require.False(t, val)
	require.True(t, set)
	val, set = modes.IsInSubnet()
	require.True(t, val)
	require.True(t, set)
	val, set = modes.IsOutOfPool()
	require.True(t, val)
	require.True(t, set)
}

// Test parsing the deprecated reservation-mode set to all.
func TestGetGlobalReservationModesDeprecatedAll(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservation-mode": "all"
        }
    }`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationParameters()
	require.NotNil(t, modes)
	val, set := modes.IsGlobal()
	require.False(t, val)
	require.True(t, set)
	val, set = modes.IsInSubnet()
	require.True(t, val)
	require.True(t, set)
	val, set = modes.IsOutOfPool()
	require.False(t, val)
	require.True(t, set)
}

// Test parsing the configuration when host reservation modes aren't
// explicitly specified.
func TestGetGlobalReservationModesDefaults(t *testing.T) {
	configStr := `{
        "Dhcp4": { }
    }`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationParameters()
	require.NotNil(t, modes)
	val, set := modes.IsGlobal()
	require.False(t, val)
	require.False(t, set)
	val, set = modes.IsInSubnet()
	require.True(t, val)
	require.False(t, set)
	val, set = modes.IsOutOfPool()
	require.False(t, val)
	require.False(t, set)
}

// Test a function implementing host reservation mode checking using
// Kea inheritance scheme.
func TestIsInAnyReservationModes(t *testing.T) {
	modes := []ReservationParameters{
		{
			ReservationsOutOfPool: nil,
		},
		{
			ReservationsOutOfPool: new(bool),
		},
		{
			ReservationsOutOfPool: new(bool),
		},
	}
	*modes[2].ReservationsOutOfPool = true

	require.True(t, IsInAnyReservationModes(func(modes ReservationParameters) (bool, bool) {
		return modes.IsOutOfPool()
	}, modes[0], modes[1], modes[2]))

	require.True(t, IsInAnyReservationModes(func(modes ReservationParameters) (bool, bool) {
		return modes.IsOutOfPool()
	}, modes[0], modes[2], modes[1]))

	require.False(t, IsInAnyReservationModes(func(modes ReservationParameters) (bool, bool) {
		return modes.IsOutOfPool()
	}, modes[1], modes[0]))

	require.False(t, IsInAnyReservationModes(func(modes ReservationParameters) (bool, bool) {
		return modes.IsOutOfPool()
	}, modes[0], modes[0]))
}

// Test that the sensitive data are hidden.
func TestHideSensitiveData(t *testing.T) {
	// Arrange
	config, err := NewConfig(`{
		"foo": "bar",
		"password": "xxx",
		"token": "",
		"secret": "aaa",
		"first": {
			"foo": "baz",
			"Password": 42,
			"Token": null,
			"Secret": "bbb",
			"second": {
				"foo": "biz",
				"passworD": true,
				"tokeN": "yyy",
				"secreT": "ccc"
			}
		}
	}`)
	require.NoError(t, err)

	// Act
	config.HideSensitiveData()
	data := config.Raw

	// Assert
	// Top level
	require.EqualValues(t, "bar", data["foo"])
	require.EqualValues(t, nil, data["password"])
	require.EqualValues(t, nil, data["token"])
	require.EqualValues(t, nil, data["secret"])
	// First level of the nesting
	first := data["first"].(map[string]interface{})
	require.EqualValues(t, "baz", first["foo"])
	require.EqualValues(t, nil, first["Password"])
	require.EqualValues(t, nil, first["Token"])
	require.EqualValues(t, nil, first["Secret"])
	// Second level of the nesting
	second := first["second"].(map[string]interface{})
	require.EqualValues(t, "biz", second["foo"])
	require.EqualValues(t, nil, second["passworD"])
	require.EqualValues(t, nil, second["tokeN"])
	require.EqualValues(t, nil, second["secreT"])
}

// Test that client classes list can be extracted from the
// Kea configuration.
func TestGetClientClasses(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "client-classes": [
				{
					"name": "foo"
				},
				{
					"name": "bar"
				}
			]
        }
    }`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	clientClasses := cfg.GetClientClasses()
	require.Len(t, clientClasses, 2)
}

// Test that empty set of client classes is returned when there is
// no client-classes entry in the configuration.
func TestGetClientClassesNonExisting(t *testing.T) {
	configStr := `{
		"Dhcp4": {
		}
	}`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	clientClasses := cfg.GetClientClasses()
	require.Empty(t, clientClasses)
}

// Test that the subnet ID can be extracted from the Kea configuration for
// an IPv4 subnet having specified prefix.
func TestGetLocalIPv4SubnetID(t *testing.T) {
	cfg := getTestConfigWithIPv4Subnets(t)

	require.EqualValues(t, 567, cfg.GetSubnetByPrefix("10.1.0.0/16").GetID())
	require.EqualValues(t, 678, cfg.GetSubnetByPrefix("10.2.1.0/16").GetID())
	require.EqualValues(t, 123, cfg.GetSubnetByPrefix("192.0.2.0/24").GetID())
	require.EqualValues(t, 234, cfg.GetSubnetByPrefix("192.0.3.0/24").GetID())
	require.EqualValues(t, 345, cfg.GetSubnetByPrefix("10.0.0.0/8").GetID())
	require.Nil(t, cfg.GetSubnetByPrefix("10.0.0.0/16"))
}

// Test that the subnet ID can be extracted from the Kea configuration for
// an IPv6 subnet having specified prefix.
func TestGetLocalIPv6SubnetID(t *testing.T) {
	cfg := getTestConfigWithIPv6Subnets(t)

	require.EqualValues(t, 567, cfg.GetSubnetByPrefix("3000:1::/32").GetID())
	require.EqualValues(t, 678, cfg.GetSubnetByPrefix("3000:2:0::/32").GetID())
	require.EqualValues(t, 123, cfg.GetSubnetByPrefix("2001:db8:1::/64").GetID())
	require.EqualValues(t, 234, cfg.GetSubnetByPrefix("2001:db8:2::/64").GetID())
	require.EqualValues(t, 345, cfg.GetSubnetByPrefix("2001:db8:3::/64").GetID())
	require.Nil(t, cfg.GetSubnetByPrefix("2001:db8:4::/64"))
}

// Test that the top-level multi-threading parameters are returned properly.
func TestGetMultiThreadingEntry(t *testing.T) {
	// Arrange
	configStr := `{
		"Dhcp4": {
			"multi-threading": {
			   "enable-multi-threading": true,
			   "thread-pool-size": 4,
			   "packet-queue-size": 16
			}
		}
	}`

	config, err := NewConfig(configStr)
	require.NoError(t, err)

	// Act
	multiThreading := config.GetMultiThreading()

	// Assert
	require.NotNil(t, multiThreading)
	require.NotNil(t, multiThreading.EnableMultiThreading)
	require.True(t, *multiThreading.EnableMultiThreading)
	require.NotNil(t, multiThreading.ThreadPoolSize)
	require.EqualValues(t, 4, *multiThreading.ThreadPoolSize)
	require.NotNil(t, multiThreading.PacketQueueSize)
	require.EqualValues(t, 16, *multiThreading.PacketQueueSize)
}

// Test that the top-level multi-threading structure is returned even if it
// includes no parameters.
func TestGetMultiThreadingEntryMissingParameters(t *testing.T) {
	// Arrange
	configStr := `{ "Dhcp4": { "multi-threading": { } } }`
	config, _ := NewConfig(configStr)

	// Act
	multiThreading := config.GetMultiThreading()

	// Assert
	require.NotNil(t, multiThreading)
	require.Nil(t, multiThreading.EnableMultiThreading)
	require.Nil(t, multiThreading.PacketQueueSize)
	require.Nil(t, multiThreading.ThreadPoolSize)
}

// Test that the top-level multi-threading parameters are nil if the
// multi-threading entry is missing.
func TestGetMultiThreadingEntryNotExists(t *testing.T) {
	// Arrange
	configStr := `{ "Dhcp4": { } }`
	config, _ := NewConfig(configStr)

	// Act
	multiThreading := config.GetMultiThreading()

	// Assert
	require.Nil(t, multiThreading)
}

// Test getting all shared networks from the DHCPv4 config.
func TestGetSharedNetworks4(t *testing.T) {
	cfg := getTestConfigWithIPv4Subnets(t)
	require.NotNil(t, cfg)

	sharedNetworks := cfg.GetSharedNetworks(false)
	require.Len(t, sharedNetworks, 2)

	for i := 0; i < len(sharedNetworks); i++ {
		require.IsType(t, (*SharedNetwork4)(nil), sharedNetworks[i])
	}
	require.Equal(t, "foo", sharedNetworks[0].GetName())
	require.Len(t, sharedNetworks[0].GetSubnets(), 2)
	require.Len(t, sharedNetworks[0].GetDHCPOptions(), 1)
	require.Equal(t, "bar", sharedNetworks[1].GetName())
	require.Len(t, sharedNetworks[1].GetSubnets(), 2)
}

// Test getting all shared networks from the DHCPv6 config.
func TestGetSharedNetworks6(t *testing.T) {
	cfg := getTestConfigWithIPv6Subnets(t)
	require.NotNil(t, cfg)

	sharedNetworks := cfg.GetSharedNetworks(false)
	require.Len(t, sharedNetworks, 2)

	for i := 0; i < len(sharedNetworks); i++ {
		require.IsType(t, (*SharedNetwork6)(nil), sharedNetworks[i])
	}
	require.Equal(t, "foo", sharedNetworks[0].GetName())
	require.Len(t, sharedNetworks[0].GetSubnets(), 2)
	require.Len(t, sharedNetworks[0].GetDHCPOptions(), 1)
	require.Equal(t, "bar", sharedNetworks[1].GetName())
	require.Len(t, sharedNetworks[1].GetSubnets(), 2)
	require.Empty(t, sharedNetworks[1].GetDHCPOptions())
}

// Test getting all top-level IPv4 subnets from the DHCPv4 config.
func TestGetSubnets4(t *testing.T) {
	cfg := getTestConfigWithIPv4Subnets(t)
	require.NotNil(t, cfg)

	subnets := cfg.GetSubnets()
	require.Len(t, subnets, 3)

	for i := 0; i < len(subnets); i++ {
		require.IsType(t, (*Subnet4)(nil), subnets[i])
	}
	require.EqualValues(t, 123, subnets[0].GetID())
	require.Equal(t, "192.0.2.0/24", subnets[0].GetPrefix())
	require.EqualValues(t, 234, subnets[1].GetID())
	require.Equal(t, "192.0.3.0/24", subnets[1].GetPrefix())
	require.EqualValues(t, 345, subnets[2].GetID())
	require.Equal(t, "10.0.0.0/8", subnets[2].GetPrefix())
}

// Test getting all top-level IPv6 subnets from the DHCPv4 config.
func TestGetSubnets6(t *testing.T) {
	cfg := getTestConfigWithIPv6Subnets(t)

	require.NotNil(t, cfg)

	subnets := cfg.GetSubnets()
	require.Len(t, subnets, 3)

	for i := 0; i < len(subnets); i++ {
		require.IsType(t, (*Subnet6)(nil), subnets[i])
	}
	require.EqualValues(t, 123, subnets[0].GetID())
	require.Equal(t, "2001:db8:1:0::/64", subnets[0].GetPrefix())
	require.EqualValues(t, 234, subnets[1].GetID())
	require.Equal(t, "2001:db8:2::/64", subnets[1].GetPrefix())
	require.EqualValues(t, 345, subnets[2].GetID())
	require.Equal(t, "2001:db8:3::/64", subnets[2].GetPrefix())
}

// Test getting global DHCPv4 host reservations.
func TestGetReservations4(t *testing.T) {
	cfg := getTestConfigWithGlobalReservations(t, "Dhcp4")
	require.NotNil(t, cfg)

	reservations := cfg.GetReservations()
	require.Len(t, reservations, 2)
	require.Equal(t, "01:02:03:04:05:06", reservations[0].HWAddress)
	require.Equal(t, "foo.example.org", reservations[0].Hostname)
	require.Equal(t, "01:01:01:01", reservations[1].DUID)
	require.Len(t, reservations[1].ClientClasses, 1)
	require.Equal(t, "bar", reservations[1].ClientClasses[0])
}

// Test getting global DHCPv6 host reservations.
func TestGetReservations6(t *testing.T) {
	cfg := getTestConfigWithGlobalReservations(t, "Dhcp6")
	require.NotNil(t, cfg)

	reservations := cfg.GetReservations()
	require.Len(t, reservations, 2)
	require.Equal(t, "01:02:03:04:05:06", reservations[0].HWAddress)
	require.Equal(t, "foo.example.org", reservations[0].Hostname)
	require.Equal(t, "01:01:01:01", reservations[1].DUID)
	require.Len(t, reservations[1].ClientClasses, 1)
	require.Equal(t, "bar", reservations[1].ClientClasses[0])
}
