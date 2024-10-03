package keaconfig

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"testing"

	require "github.com/stretchr/testify/require"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

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

// Test that Kea DHCPv4 configuration is recognized and parsed.
func TestDecodeDHCPv4(t *testing.T) {
	var config Config
	err := json.Unmarshal(testutil.AllKeysDHCPv4JSON, &config)
	require.NoError(t, err)

	require.NotNil(t, config.DHCPv4Config)

	marshalled, err := json.Marshal(config)
	require.NoError(t, err)

	require.JSONEq(t, string(testutil.AllKeysDHCPv4JSON), string(marshalled))
}

// Test that Kea DHCPv6 configuration is recognized and parsed.
func TestDecodeDHCPv6(t *testing.T) {
	var config Config
	err := json.Unmarshal(testutil.AllKeysDHCPv6JSON, &config)
	require.NoError(t, err)

	require.NotNil(t, config.DHCPv6Config)

	marshalled, err := json.Marshal(config)
	require.NoError(t, err)

	require.JSONEq(t, string(testutil.AllKeysDHCPv6JSON), string(marshalled))
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

// Test getting raw configuration.
func TestGetRawConfig(t *testing.T) {
	// Set the original config.
	var config Config
	err := json.Unmarshal(testutil.AllKeysDHCPv4JSON, &config)
	require.NoError(t, err)

	// Extract the raw config.
	rawConfig, err := config.GetRawConfig()
	require.NoError(t, err)
	require.NotNil(t, rawConfig)
	require.Equal(t, config.Raw, rawConfig)

	// Make sure it contains the top-level key.
	require.Contains(t, rawConfig, "Dhcp4")

	// Serialize it again and make sure it is equal to the original.
	marshalled, err := json.Marshal(config)
	require.NoError(t, err)

	require.JSONEq(t, string(testutil.AllKeysDHCPv4JSON), string(marshalled))
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
	require.Len(t, loggers[0].GetAllOutputOptions(), 1)
	require.Equal(t, "stdout", loggers[0].GetAllOutputOptions()[0].Output)
	require.Equal(t, "WARN", loggers[0].Severity)
	require.Zero(t, loggers[0].DebugLevel)

	require.Equal(t, "kea-dhcp4.bad-packets", loggers[1].Name)
	require.Len(t, loggers[1].GetAllOutputOptions(), 1)
	require.Equal(t, "/tmp/badpackets.log", loggers[1].GetAllOutputOptions()[0].Output)
	require.Equal(t, "DEBUG", loggers[1].Severity)
	require.Equal(t, 99, loggers[1].DebugLevel)
}

// Verifies that a list of control sockets is parsed correctly for a daemon.
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
	require.NotNil(t, sockets)
	require.True(t, sockets.HasAnyConfiguredDaemon())

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

// Verifies that nil is returned if the control-sockets entry is not configured.
func TestGetControlSocketsForMissingEntry(t *testing.T) {
	configStr := `{ "Control-agent": { } }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.CtrlAgentConfig)

	sockets := cfg.GetControlSockets()
	require.Nil(t, sockets)
	require.False(t, sockets.HasAnyConfiguredDaemon())
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
	require.NotNil(t, sockets)
	require.True(t, sockets.HasAnyConfiguredDaemon())

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
	require.NotNil(t, sockets)
	require.True(t, sockets.HasAnyConfiguredDaemon())

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

// Test that caching parameters are parsed and returned correctly.
func TestGetCacheParameters(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "cache-threshold": 0.5
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetCacheParameters().CacheThreshold)
	require.EqualValues(t, 0.5, *cfg.GetCacheParameters().CacheThreshold)
	require.Nil(t, cfg.GetCacheParameters().CacheMaxAge)
}

// Test that DDNS parameters are parsed and returned correctly.
func TestGetDDNSParameters(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "ddns-generated-prefix": "myhost",
            "ddns-override-client-update": false,
            "ddns-override-no-update": false,
            "ddns-qualifying-suffix": "suffix",
            "ddns-replace-client-name": "never",
            "ddns-send-updates": true,
            "ddns-update-on-renew": true,
            "ddns-use-conflict-resolution": true,
            "ddns-ttl-percent": 0.55
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetDDNSParameters())
	require.NotNil(t, cfg.GetDDNSParameters().DDNSGeneratedPrefix)
	require.Equal(t, "myhost", *cfg.GetDDNSParameters().DDNSGeneratedPrefix)
	require.NotNil(t, cfg.GetDDNSParameters().DDNSOverrideClientUpdate)
	require.False(t, *cfg.GetDDNSParameters().DDNSOverrideClientUpdate)
	require.NotNil(t, cfg.GetDDNSParameters().DDNSOverrideNoUpdate)
	require.False(t, *cfg.GetDDNSParameters().DDNSOverrideNoUpdate)
	require.NotNil(t, cfg.GetDDNSParameters().DDNSQualifyingSuffix)
	require.Equal(t, "suffix", *cfg.GetDDNSParameters().DDNSQualifyingSuffix)
	require.NotNil(t, cfg.GetDDNSParameters().DDNSReplaceClientName)
	require.Equal(t, "never", *cfg.GetDDNSParameters().DDNSReplaceClientName)
	require.NotNil(t, cfg.GetDDNSParameters().DDNSSendUpdates)
	require.True(t, *cfg.GetDDNSParameters().DDNSSendUpdates)
	require.NotNil(t, cfg.GetDDNSParameters().DDNSUpdateOnRenew)
	require.True(t, *cfg.GetDDNSParameters().DDNSUpdateOnRenew)
	require.NotNil(t, cfg.GetDDNSParameters().DDNSUseConflictResolution)
	require.True(t, *cfg.GetDDNSParameters().DDNSUseConflictResolution)
	require.NotNil(t, cfg.GetDDNSParameters().DDNSTTLPercent)
	require.Equal(t, float32(0.55), *cfg.GetDDNSParameters().DDNSTTLPercent)
}

// Test that DHCP DDNS parameters are parsed and returned correctly.
func TestGetDHCPDDNSParameters(t *testing.T) {
	configStr := `{
        "Dhcp4": {
			"dhcp-ddns": {
				"enable-updates": true,
				"max-queue-size": 7,
				"ncr-format": "JSON",
				"ncr-protocol": "UDP",
				"sender-ip": "192.0.2.1",
				"sender-port": 8080,
				"server-ip": "192.0.2.2",
				"server-port": 8081
			}
		}
	}`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetDHCPDDNSParameters())
	require.NotNil(t, cfg.GetDHCPDDNSParameters().EnableUpdates)
	require.True(t, *cfg.GetDHCPDDNSParameters().EnableUpdates)
	require.NotNil(t, cfg.GetDHCPDDNSParameters().MaxQueueSize)
	require.EqualValues(t, 7, *cfg.GetDHCPDDNSParameters().MaxQueueSize)
	require.NotNil(t, cfg.GetDHCPDDNSParameters().NCRFormat)
	require.Equal(t, "JSON", *cfg.GetDHCPDDNSParameters().NCRFormat)
	require.NotNil(t, cfg.GetDHCPDDNSParameters().NCRProtocol)
	require.EqualValues(t, "UDP", *cfg.GetDHCPDDNSParameters().NCRProtocol)
	require.NotNil(t, cfg.GetDHCPDDNSParameters().SenderIP)
	require.Equal(t, "192.0.2.1", *cfg.GetDHCPDDNSParameters().SenderIP)
	require.NotNil(t, cfg.GetDHCPDDNSParameters().SenderPort)
	require.EqualValues(t, 8080, *cfg.GetDHCPDDNSParameters().SenderPort)
	require.NotNil(t, cfg.GetDHCPDDNSParameters().ServerIP)
	require.Equal(t, "192.0.2.2", *cfg.GetDHCPDDNSParameters().ServerIP)
	require.NotNil(t, cfg.GetDHCPDDNSParameters().ServerPort)
	require.EqualValues(t, 8081, *cfg.GetDHCPDDNSParameters().ServerPort)
}

// Test that expiration lease processing parameters are parsed and returned correctly.
func TestGetExpiredLeasesProcessingParameters(t *testing.T) {
	configStr := `{
        "Dhcp4": {
			"expired-leases-processing": {
				"flush-reclaimed-timer-wait-time": 15,
				"hold-reclaimed-time": 3,
				"max-reclaim-leases": 18,
				"max-reclaim-time": 123,
				"reclaim-timer-wait-time": 11,
				"unwarned-reclaim-cycles": 1
			}
		}
	}`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetExpiredLeasesProcessingParameters())
	require.NotNil(t, cfg.GetExpiredLeasesProcessingParameters().FlushReclaimedTimerWaitTime)
	require.EqualValues(t, 15, *cfg.GetExpiredLeasesProcessingParameters().FlushReclaimedTimerWaitTime)
	require.NotNil(t, cfg.GetExpiredLeasesProcessingParameters().HoldReclaimedTime)
	require.EqualValues(t, 3, *cfg.GetExpiredLeasesProcessingParameters().HoldReclaimedTime)
	require.NotNil(t, cfg.GetExpiredLeasesProcessingParameters().MaxReclaimLeases)
	require.EqualValues(t, 18, *cfg.GetExpiredLeasesProcessingParameters().MaxReclaimLeases)
	require.NotNil(t, cfg.GetExpiredLeasesProcessingParameters().MaxReclaimTime)
	require.EqualValues(t, 123, *cfg.GetExpiredLeasesProcessingParameters().MaxReclaimTime)
	require.NotNil(t, cfg.GetExpiredLeasesProcessingParameters().ReclaimTimerWaitTime)
	require.EqualValues(t, 11, *cfg.GetExpiredLeasesProcessingParameters().ReclaimTimerWaitTime)
	require.NotNil(t, cfg.GetExpiredLeasesProcessingParameters().UnwarnedReclaimCycles)
	require.EqualValues(t, 1, *cfg.GetExpiredLeasesProcessingParameters().UnwarnedReclaimCycles)
}

// Test that hostname char parameters are parsed and returned correctly.
func TestGetHostnameCharParameters(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "hostname-char-replacement": "a",
            "hostname-char-set": "bcd"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetHostnameCharParameters().HostnameCharReplacement)
	require.Equal(t, "a", *cfg.GetHostnameCharParameters().HostnameCharReplacement)
	require.NotNil(t, cfg.GetHostnameCharParameters().HostnameCharSet)
	require.Equal(t, "bcd", *cfg.GetHostnameCharParameters().HostnameCharSet)
}

// Test that timer parameters are parsed and returned correctly.
func TestGetTimerParameters(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "calculate-tee-times": true,
            "renew-timer": 60,
            "t2-percent": 0.5
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetTimerParameters().CalculateTeeTimes)
	require.True(t, *cfg.GetTimerParameters().CalculateTeeTimes)
	require.Nil(t, cfg.GetTimerParameters().RebindTimer)
	require.NotNil(t, cfg.GetTimerParameters().RenewTimer)
	require.EqualValues(t, 60, *cfg.GetTimerParameters().RenewTimer)
	require.Nil(t, cfg.GetTimerParameters().T1Percent)
	require.NotNil(t, cfg.GetTimerParameters().T2Percent)
	require.EqualValues(t, 0.5, *cfg.GetTimerParameters().T2Percent)
}

// Test that preferred lifetime parameters are parsed and returned correctly.
func TestGetPreferredLifetimeParameters(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "min-preferred-lifetime": 10,
			"preferred-lifetime": 50
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetPreferredLifetimeParameters().MaxPreferredLifetime)
	require.NotNil(t, cfg.GetPreferredLifetimeParameters().MinPreferredLifetime)
	require.EqualValues(t, 10, *cfg.GetPreferredLifetimeParameters().MinPreferredLifetime)
	require.NotNil(t, cfg.GetPreferredLifetimeParameters().PreferredLifetime)
	require.EqualValues(t, 50, *cfg.GetPreferredLifetimeParameters().PreferredLifetime)
}

// Test that preferred lifetime parameters are ignored for DHCPv4.
func TestGetPreferredLifetimeParametersUnsupported(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "min-preferred-lifetime": 10,
			"preferred-lifetime": 50
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetPreferredLifetimeParameters().MaxPreferredLifetime)
	require.Nil(t, cfg.GetPreferredLifetimeParameters().MinPreferredLifetime)
	require.Nil(t, cfg.GetPreferredLifetimeParameters().PreferredLifetime)
}

// Test that valid lifetime parameters are parsed and returned correctly.
func TestGetValidLifetimeParameters(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "min-valid-lifetime": 10,
            "valid-lifetime": 50
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetValidLifetimeParameters().MaxValidLifetime)
	require.NotNil(t, cfg.GetValidLifetimeParameters().MinValidLifetime)
	require.EqualValues(t, 10, *cfg.GetValidLifetimeParameters().MinValidLifetime)
	require.NotNil(t, cfg.GetValidLifetimeParameters().ValidLifetime)
	require.EqualValues(t, 50, *cfg.GetValidLifetimeParameters().ValidLifetime)
}

// Test that allocator parameter is parsed and returned correctly.
func TestGetAllocator(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "allocator": "random"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetAllocator())
	require.Equal(t, "random", *cfg.GetAllocator())
}

// Test that PD allocator parameter is parsed and returned correctly.
func TestGetPDAllocator(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "pd-allocator": "flq"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetPDAllocator())
	require.Equal(t, "flq", *cfg.GetPDAllocator())
}

// Test that PD allocator parameter is ignored for DHCPv4.
func TestGetPDAllocatorUnsupported(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "pd-allocator": "flq"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetPDAllocator())
}

// Test that the authoritative parameter is parsed and returned correctly.
func TestGetAuthoritative(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "authoritative": true
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetAuthoritative())
	require.True(t, *cfg.GetAuthoritative())
}

// Test that the authoritative parameter is ignored for DHCPv6.
func TestGetAuthoritativeUnsupported(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "authoritative": true
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetAuthoritative())
}

// Test that the boot-file-name parameter is parsed and returned correctly.
func TestGetBootFileName(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "boot-file-name": "/tmp/boot"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetBootFileName())
	require.Equal(t, "/tmp/boot", *cfg.GetBootFileName())
}

// Test that the boot-file-name parameter is ignored for DHCPv6.
func TestGetBootFileNameUnsupported(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "boot-file-name": "/tmp/boot"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetBootFileName())
}

// Test that the match-client-id parameter is parsed and returned correctly.
func TestGetMatchClientID(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "match-client-id": false
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetMatchClientID())
	require.False(t, *cfg.GetMatchClientID())
}

// Test that the match-client-id parameter is ignored for DHCPv6.
func TestGetMatchClientIDUnsupported(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "match-client-id": false
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetMatchClientID())
}

// Test that the next-server parameter is parsed and returned correctly.
func TestGetNextServer(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "next-server": "10.1.1.1"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetNextServer())
	require.Equal(t, "10.1.1.1", *cfg.GetNextServer())
}

// Test that the next-server parameter is ignored for DHCPv6.
func TestGetNextServerUnsupported(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "next-server": "10.1.1.1"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetNextServer())
}

// Test that the server-hostname parameter is parsed and returned correctly.
func TestGetServerHostname(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "server-hostname": "myhost"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetServerHostname())
	require.Equal(t, "myhost", *cfg.GetServerHostname())
}

// Test that the server-hostname parameter is ignored for DHCPv6.
func TestGetServerHostnameUnsupported(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "server-hostname": "myhost"
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetServerHostname())
}

// Test that the rapid-commit parameter is parsed and returned correctly.
func TestGetRapidCommit(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "rapid-commit": true
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetRapidCommit())
	require.True(t, *cfg.GetRapidCommit())
}

// Test that the rapid-commit parameter is ignored for DHCPv4.
func TestGetRapidCommitUnsupported(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "rapid-commit": true
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Nil(t, cfg.GetRapidCommit())
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

// Test that the store-extended-info parameter is parsed and returned correctly.
func TestStoreExtendedInfo(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "store-extended-info": false
        }
    }`

	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.NotNil(t, cfg.GetStoreExtendedInfo())
	require.False(t, *cfg.GetStoreExtendedInfo())
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

// Test that MT is by default disabled in Kea version earlier than 2.3.5.
func TestIsGlobalMultiThreadingEnabledDefault230(t *testing.T) {
	configStr := `{ "Dhcp4": { } }`
	config, _ := NewConfig(configStr)
	require.False(t, config.IsMultiThreadingEnabled(storkutil.NewSemanticVersion(2, 3, 0)))
}

// Test that MT can be explicitly enabled in Kea version earlier than 2.3.5.
func TestIsGlobalMultiThreadingEnabledEnabled230(t *testing.T) {
	configStr := `{
		"Dhcp4": {
			"multi-threading": {
				"enable-multi-threading": true
			}
		}
	}`
	config, _ := NewConfig(configStr)
	require.True(t, config.IsMultiThreadingEnabled(storkutil.NewSemanticVersion(2, 3, 0)))
}

// Test that MT is by default enabled in Kea version later than or equal
// 2.3.5.
func TestIsGlobalMultiThreadingEnabledDefault235(t *testing.T) {
	configStr := `{ "Dhcp4": { } }`
	config, _ := NewConfig(configStr)
	require.True(t, config.IsMultiThreadingEnabled(storkutil.NewSemanticVersion(2, 3, 5)))
}

// Test that MT can be explicitly disabled in Kea version later than 2.3.5.
func TestIsGlobalMultiThreadingEnabledDisabled235(t *testing.T) {
	configStr := `{
		"Dhcp4": {
			"multi-threading": {
				"enable-multi-threading": false
			}
		}
	}`
	config, _ := NewConfig(configStr)
	require.False(t, config.IsMultiThreadingEnabled(storkutil.NewSemanticVersion(2, 3, 0)))
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

// Test getting global DHCPv4 options.
func TestGetDHCPOptions4(t *testing.T) {
	configStr := `{
		"Dhcp4": {
			"option-data": [
				{
					"always-send": false,
					"code": 3,
					"csv-format": true,
					"data": "10.0.0.1",
					"name": "routers",
					"space": "dhcp4"
				},
				{
					"always-send": true,
					"code": 6,
					"csv-format": true,
					"data": "192.0.3.1, 192.0.3.2",
					"name": "domain-name-servers",
					"space": "dhcp4"
				}
			]
		}
	}`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	options := cfg.GetDHCPOptions()
	require.Len(t, options, 2)

	require.False(t, options[0].AlwaysSend)
	require.EqualValues(t, 3, options[0].Code)
	require.True(t, options[0].CSVFormat)
	require.Equal(t, "10.0.0.1", options[0].Data)
	require.Equal(t, "routers", options[0].Name)
	require.Equal(t, dhcpmodel.DHCPv4OptionSpace, options[0].Space)

	require.True(t, options[1].AlwaysSend)
	require.EqualValues(t, 6, options[1].Code)
	require.True(t, options[1].CSVFormat)
	require.Equal(t, "192.0.3.1, 192.0.3.2", options[1].Data)
	require.Equal(t, "domain-name-servers", options[1].Name)
	require.Equal(t, dhcpmodel.DHCPv4OptionSpace, options[0].Space)
}

// Test getting global DHCPv6 options.
func TestGetDHCPOptions6(t *testing.T) {
	configStr := `{
		"Dhcp6": {
			"option-data": [
				{
					"always-send": false,
					"code": 23,
					"csv-format": true,
					"data": "2001:db8:1::1",
					"name": "dns-servers",
					"space": "dhcp6"
				},
				{
					"always-send": true,
					"code": 27,
					"csv-format": true,
					"data": "2001:db8:1::2, 2001:db8:1::3",
					"name": "nis-servers",
					"space": "dhcp6"
				}
			]
		}
	}`
	cfg, err := NewConfig(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	options := cfg.GetDHCPOptions()
	require.Len(t, options, 2)

	require.False(t, options[0].AlwaysSend)
	require.EqualValues(t, 23, options[0].Code)
	require.True(t, options[0].CSVFormat)
	require.Equal(t, "2001:db8:1::1", options[0].Data)
	require.Equal(t, "dns-servers", options[0].Name)
	require.Equal(t, dhcpmodel.DHCPv6OptionSpace, options[0].Space)

	require.True(t, options[1].AlwaysSend)
	require.EqualValues(t, 27, options[1].Code)
	require.True(t, options[1].CSVFormat)
	require.Equal(t, "2001:db8:1::2, 2001:db8:1::3", options[1].Data)
	require.Equal(t, "nis-servers", options[1].Name)
	require.Equal(t, dhcpmodel.DHCPv6OptionSpace, options[0].Space)
}

// Test merging a partial config into current config.
func TestMergeConfig(t *testing.T) {
	source1 := `{
		"Dhcp4": {
			"allocator": "iterative",
			"valid-lifetime": 123,
			"host-identifiers": [ "hw-address", "circuit-id" ],
			"expired-leases-processing": {
				"reclaim-timer-wait-time": 5,
				"max-reclaim-leases": 0
			},
			"hooks-libraries": [
				{
					"library": "/tmp/hooks-library.so"
				}
			]
		},
		"hash": "01020304"
	}`
	source2 := `{
		"Dhcp4": {
			"allocator": "flq",
			"boot-file-name": "/tmp/filename",
			"expired-leases-processing": {
				"max-reclaim-leases": 12,
				"max-reclaim-time": 7
			}
		},
		"hash": "02030405"
	}`
	config1, err := NewConfig(source1)
	require.NoError(t, err)
	require.NotNil(t, config1)

	config2, err := NewConfig(source2)
	require.NoError(t, err)
	require.NotNil(t, config2)

	config1.Merge(config2)
	require.NotNil(t, config1)

	marshalled, err := json.Marshal(config1)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"Dhcp4": {
			"allocator": "flq",
			"boot-file-name": "/tmp/filename",
			"valid-lifetime": 123,
			"host-identifiers": [ "hw-address", "circuit-id" ],
			"expired-leases-processing": {
				"reclaim-timer-wait-time": 5,
				"max-reclaim-leases": 12,
				"max-reclaim-time": 7
			},
			"hooks-libraries": [
				{
					"library": "/tmp/hooks-library.so"
				}
			]
		}
	}`, string(marshalled))

	require.NotNil(t, config1.GetAllocator())
	require.Equal(t, "flq", *config1.GetAllocator())

	require.NotNil(t, config1.GetBootFileName())
	require.Equal(t, "/tmp/filename", *config1.GetBootFileName())

	require.NotNil(t, config1.GetValidLifetimeParameters().ValidLifetime)
	require.EqualValues(t, 123, *config1.GetValidLifetimeParameters().ValidLifetime)
}

// Test merging a settable config into current config.
func TestMergeSettableConfig(t *testing.T) {
	source1 := `{
		"Dhcp4": {
			"allocator": "iterative",
			"valid-lifetime": 123,
			"host-reservation-identifiers": [ "hw-address", "circuit-id" ],
			"expired-leases-processing": {
				"reclaim-timer-wait-time": 5,
				"max-reclaim-leases": 0
			},
			"hooks-libraries": [
				{
					"library": "/tmp/hooks-library.so"
				}
			]
		},
		"hash": "01020304"
	}`
	config1, err := NewConfig(source1)
	require.NoError(t, err)
	require.NotNil(t, config1)

	config2 := NewSettableDHCPv4Config()
	require.NoError(t, err)

	config2.SetAllocator(storkutil.Ptr("flq"))
	config2.SetELPMaxReclaimLeases(storkutil.Ptr(int64(12)))
	config2.SetELPMaxReclaimTime(storkutil.Ptr(int64(7)))
	config2.SetDHCPDDNSEnableUpdates(storkutil.Ptr(true))
	config2.SetDHCPDDNSMaxQueueSize(nil)
	config2.SetValidLifetime(nil)

	config1.Merge(config2)
	require.NotNil(t, config1)

	marshalled, err := json.Marshal(config1)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"Dhcp4": {
			"allocator": "flq",
			"host-reservation-identifiers": [ "hw-address", "circuit-id" ],
			"dhcp-ddns": {
				"enable-updates": true
			},
			"expired-leases-processing": {
				"reclaim-timer-wait-time": 5,
				"max-reclaim-leases": 12,
				"max-reclaim-time": 7
			},
			"hooks-libraries": [
				{
					"library": "/tmp/hooks-library.so"
				}
			]
		}
	}`, string(marshalled))

	require.NotNil(t, config1.GetAllocator())
	require.Equal(t, "flq", *config1.GetAllocator())

	require.ElementsMatch(t, config1.GetGlobalReservationParameters().HostReservationIdentifiers, []string{"hw-address", "circuit-id"})

	d2 := config1.GetDHCPDDNSParameters()
	require.NotNil(t, d2)
	require.NotNil(t, d2.EnableUpdates)
	require.True(t, *d2.EnableUpdates)
	require.Nil(t, d2.MaxQueueSize)

	elp := config1.GetExpiredLeasesProcessingParameters()
	require.NotNil(t, elp)
	require.NotNil(t, elp.ReclaimTimerWaitTime)
	require.EqualValues(t, 5, *elp.ReclaimTimerWaitTime)
	require.NotNil(t, elp.MaxReclaimLeases)
	require.EqualValues(t, 12, *elp.MaxReclaimLeases)
	require.NotNil(t, elp.MaxReclaimTime)
	require.EqualValues(t, 7, *elp.MaxReclaimTime)

	require.Nil(t, config1.GetValidLifetimeParameters().ValidLifetime)
}

// Tests instantiating settable Kea Control Agent configuration.
func TestNewSettableCtrlAgentConfig(t *testing.T) {
	settableConfig := NewSettableCtrlAgentConfig()
	require.NotNil(t, settableConfig)
	require.NotNil(t, settableConfig.SettableCtrlAgentConfig)
}

// Tests instantiating settable D2 configuration.
func TestNewSettableD2Config(t *testing.T) {
	settableConfig := NewSettableD2Config()
	require.NotNil(t, settableConfig)
	require.NotNil(t, settableConfig.SettableD2Config)
}

// Tests instantiating settable DHCPv4 configuration.
func TestNewSettableDHCPv4Config(t *testing.T) {
	settableConfig := NewSettableDHCPv4Config()
	require.NotNil(t, settableConfig)
	require.NotNil(t, settableConfig.SettableDHCPv4Config)
}

// Tests instantiating settable DHCPv6 configuration.
func TestNewSettableDHCPv6Config(t *testing.T) {
	settableConfig := NewSettableDHCPv6Config()
	require.NotNil(t, settableConfig)
	require.NotNil(t, settableConfig.SettableDHCPv6Config)
}

// Test getting raw configuration.
func TestGetRawSettableConfig(t *testing.T) {
	// Set the original config.
	config := NewSettableDHCPv4Config()
	err := config.SetValidLifetime(storkutil.Ptr(int64(1111)))
	require.NoError(t, err)

	// Extract the raw config.
	rawConfig, err := config.GetRawConfig()
	require.NoError(t, err)
	require.NotNil(t, rawConfig)

	// Make sure it contains the top-level key.
	require.Contains(t, rawConfig, "Dhcp4")

	// Serialize it again and make sure it is equal to the original.
	marshalled, err := json.Marshal(config)
	require.NoError(t, err)

	require.JSONEq(t, `{
		"Dhcp4": {
			"valid-lifetime": 1111
		}
	}`, string(marshalled))
}

// Test setting various DHCPv4 global configuration parameters.
func TestSettingDHCPv4GlobalParameters(t *testing.T) {
	config := NewSettableDHCPv4Config()

	err := config.SetAllocator(storkutil.Ptr("flq"))
	require.NoError(t, err)

	err = config.SetCacheThreshold(storkutil.Ptr(float32(0.2)))
	require.NoError(t, err)

	err = config.SetDDNSSendUpdates(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSOverrideNoUpdate(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSOverrideClientUpdate(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSReplaceClientName(storkutil.Ptr("never"))
	require.NoError(t, err)

	err = config.SetDDNSGeneratedPrefix(storkutil.Ptr("myhost.example.org"))
	require.NoError(t, err)

	err = config.SetDDNSQualifyingSuffix(storkutil.Ptr("example.org"))
	require.NoError(t, err)

	err = config.SetDDNSUpdateOnRenew(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSUseConflictResolution(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSConflictResolutionMode(storkutil.Ptr("check-with-dhcid"))
	require.NoError(t, err)

	err = config.SetDDNSTTLPercent(storkutil.Ptr(float32(0.1)))
	require.NoError(t, err)

	err = config.SetDHCPDDNSEnableUpdates(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDHCPDDNSMaxQueueSize(storkutil.Ptr(int64(100)))
	require.NoError(t, err)

	err = config.SetDHCPDDNSNCRFormat(storkutil.Ptr("JSON"))
	require.NoError(t, err)

	err = config.SetDHCPDDNSNCRProtocol(storkutil.Ptr("UDP"))
	require.NoError(t, err)

	err = config.SetDHCPDDNSSenderIP(storkutil.Ptr("192.0.2.1"))
	require.NoError(t, err)

	err = config.SetDHCPDDNSSenderPort(storkutil.Ptr(int64(8080)))
	require.NoError(t, err)

	err = config.SetDHCPDDNSServerIP(storkutil.Ptr("192.0.2.2"))
	require.NoError(t, err)

	err = config.SetDHCPDDNSServerPort(storkutil.Ptr(int64(8081)))
	require.NoError(t, err)

	err = config.SetELPFlushReclaimedTimerWaitTime(storkutil.Ptr(int64(111)))
	require.NoError(t, err)

	err = config.SetELPHoldReclaimedTime(storkutil.Ptr(int64(111)))
	require.NoError(t, err)

	err = config.SetELPMaxReclaimLeases(storkutil.Ptr(int64(111)))
	require.NoError(t, err)

	err = config.SetELPMaxReclaimTime(storkutil.Ptr(int64(2)))
	require.NoError(t, err)

	err = config.SetELPReclaimTimerWaitTime(storkutil.Ptr(int64(3)))
	require.NoError(t, err)

	err = config.SetELPUnwarnedReclaimCycles(storkutil.Ptr(int64(10)))
	require.NoError(t, err)

	err = config.SetReservationsGlobal(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetReservationsInSubnet(storkutil.Ptr(false))
	require.NoError(t, err)

	err = config.SetReservationsOutOfPool(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetEarlyGlobalReservationsLookup(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetHostReservationIdentifiers([]string{"hw-address", "client-id"})
	require.NoError(t, err)

	err = config.SetAuthoritative(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetEchoClientID(storkutil.Ptr(false))
	require.NoError(t, err)

	err = config.SetDHCPOptions([]SingleOptionData{
		{
			AlwaysSend: true,
			Code:       42,
			CSVFormat:  true,
			Data:       "foo",
			Name:       "forty-two",
			Space:      "dhcp4",
		},
		{
			Code:  24,
			Data:  "bar",
			Name:  "twenty-four",
			Space: "dhcp4",
		},
	})
	require.NoError(t, err)

	serializedConfig, err := config.GetSerializedConfig()
	require.NoError(t, err)

	require.JSONEq(t, `{
		"Dhcp4": {
			"cache-threshold": 0.2,
			"ddns-generated-prefix": "myhost.example.org",
			"ddns-override-client-update": true,
			"ddns-override-no-update": true,
			"ddns-qualifying-suffix": "example.org",
			"ddns-replace-client-name": "never",
			"ddns-send-updates": true,
			"ddns-update-on-renew": true,
			"ddns-use-conflict-resolution": true,
			"ddns-conflict-resolution-mode": "check-with-dhcid",
			"ddns-ttl-percent": 0.1,
			"dhcp-ddns": {
				"enable-updates": true,
				"max-queue-size": 100,
				"ncr-format": "JSON",
				"ncr-protocol": "UDP",
				"sender-ip": "192.0.2.1",
				"sender-port": 8080,
				"server-ip": "192.0.2.2",
				"server-port": 8081
			},
			"early-global-reservations-lookup": true,
			"host-reservation-identifiers": [
				"hw-address",
				"client-id"
			],
			"reservations-global": true,
			"reservations-in-subnet": false,
			"reservations-out-of-pool": true,
			"allocator": "flq",
			"expired-leases-processing": {
				"flush-reclaimed-timer-wait-time": 111,
				"hold-reclaimed-time": 111,
				"max-reclaim-leases": 111,
				"max-reclaim-time": 2,
				"reclaim-timer-wait-time": 3,
				"unwarned-reclaim-cycles": 10
			},
			"authoritative": true,
			"echo-client-id": false,
			"option-data": [
				{
					"always-send": true,
					"code": 42,
					"csv-format": true,
					"data": "foo",
					"name": "forty-two",
					"space": "dhcp4"
				},
				{
					"code": 24,
					"csv-format": false,
					"data": "bar",
					"name": "twenty-four",
					"space": "dhcp4"
				}
			]
		}
	}`, serializedConfig)
}

// Test setting DHCP DDNS for the DHCPv4 server.
func TestSettingDHCPv4DHCPDDNS(t *testing.T) {
	config := NewSettableDHCPv4Config()

	dhcpDDNS := &SettableDHCPDDNS{
		EnableUpdates: storkutil.NewNullableFromValue(true),
		ServerIP:      storkutil.NewNullableFromValue("192.0.2.1"),
		ServerPort:    storkutil.NewNullableFromValue(int64(8080)),
		SenderIP:      storkutil.NewNullableFromValue("192.0.2.2"),
		SenderPort:    storkutil.NewNullableFromValue(int64(8081)),
		MaxQueueSize:  storkutil.NewNullableFromValue(int64(100)),
		NCRProtocol:   storkutil.NewNullableFromValue("UDP"),
		NCRFormat:     storkutil.NewNullableFromValue("JSON"),
	}
	err := config.SetDHCPDDNS(dhcpDDNS)
	require.NoError(t, err)

	serializedConfig, err := config.GetSerializedConfig()
	require.NoError(t, err)

	require.JSONEq(t, `{
		"Dhcp4": {
			"dhcp-ddns": {
				"enable-updates": true,
				"server-ip": "192.0.2.1",
				"server-port": 8080,
				"sender-ip": "192.0.2.2",
				"sender-port": 8081,
				"max-queue-size": 100,
				"ncr-format": "JSON",
				"ncr-protocol": "UDP"
			}
		}
	}`, serializedConfig)
}

// Test setting expired leases processing for DHCPv4 server.
func TestSettingDHCPv4ExpiredLeasesProcessing(t *testing.T) {
	config := NewSettableDHCPv4Config()

	expiredLeasesProcessing := &SettableExpiredLeasesProcessing{
		FlushReclaimedTimerWaitTime: storkutil.NewNullableFromValue(int64(1)),
		HoldReclaimedTime:           storkutil.NewNullableFromValue(int64(2)),
		MaxReclaimLeases:            storkutil.NewNullableFromValue(int64(3)),
		MaxReclaimTime:              storkutil.NewNullableFromValue(int64(4)),
		ReclaimTimerWaitTime:        storkutil.NewNullableFromValue(int64(5)),
		UnwarnedReclaimCycles:       storkutil.NewNullableFromValue(int64(6)),
	}
	err := config.SetExpiredLeasesProcessing(expiredLeasesProcessing)
	require.NoError(t, err)

	serializedConfig, err := config.GetSerializedConfig()
	require.NoError(t, err)

	require.JSONEq(t, `{
		"Dhcp4": {
			"expired-leases-processing": {
				"flush-reclaimed-timer-wait-time": 1,
				"hold-reclaimed-time": 2,
				"max-reclaim-leases": 3,
				"max-reclaim-time": 4,
				"reclaim-timer-wait-time": 5,
				"unwarned-reclaim-cycles": 6
			}
		}
	}`, serializedConfig)
}

// Test setting various DHCPv6 global configuration parameters.
func TestSettingDHCPv6GlobalParameters(t *testing.T) {
	config := NewSettableDHCPv6Config()

	err := config.SetAllocator(storkutil.Ptr("flq"))
	require.NoError(t, err)

	err = config.SetCacheThreshold(storkutil.Ptr(float32(0.2)))
	require.NoError(t, err)

	err = config.SetDDNSSendUpdates(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSOverrideNoUpdate(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSOverrideClientUpdate(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSReplaceClientName(storkutil.Ptr("never"))
	require.NoError(t, err)

	err = config.SetDDNSGeneratedPrefix(storkutil.Ptr("myhost.example.org"))
	require.NoError(t, err)

	err = config.SetDDNSQualifyingSuffix(storkutil.Ptr("example.org"))
	require.NoError(t, err)

	err = config.SetDDNSUpdateOnRenew(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSUseConflictResolution(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDDNSTTLPercent(storkutil.Ptr(float32(0.1)))
	require.NoError(t, err)

	err = config.SetDHCPDDNSEnableUpdates(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetDHCPDDNSMaxQueueSize(storkutil.Ptr(int64(100)))
	require.NoError(t, err)

	err = config.SetDHCPDDNSNCRFormat(storkutil.Ptr("JSON"))
	require.NoError(t, err)

	err = config.SetDHCPDDNSNCRProtocol(storkutil.Ptr("UDP"))
	require.NoError(t, err)

	err = config.SetDHCPDDNSSenderIP(storkutil.Ptr("2001:db8:1::1"))
	require.NoError(t, err)

	err = config.SetDHCPDDNSSenderPort(storkutil.Ptr(int64(8080)))
	require.NoError(t, err)

	err = config.SetDHCPDDNSServerIP(storkutil.Ptr("2001:db8:1::2"))
	require.NoError(t, err)

	err = config.SetDHCPDDNSServerPort(storkutil.Ptr(int64(8081)))
	require.NoError(t, err)

	err = config.SetELPFlushReclaimedTimerWaitTime(storkutil.Ptr(int64(111)))
	require.NoError(t, err)

	err = config.SetELPHoldReclaimedTime(storkutil.Ptr(int64(111)))
	require.NoError(t, err)

	err = config.SetELPMaxReclaimLeases(storkutil.Ptr(int64(111)))
	require.NoError(t, err)

	err = config.SetELPMaxReclaimTime(storkutil.Ptr(int64(2)))
	require.NoError(t, err)

	err = config.SetELPReclaimTimerWaitTime(storkutil.Ptr(int64(3)))
	require.NoError(t, err)

	err = config.SetELPUnwarnedReclaimCycles(storkutil.Ptr(int64(10)))
	require.NoError(t, err)

	err = config.SetReservationsGlobal(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetReservationsInSubnet(storkutil.Ptr(false))
	require.NoError(t, err)

	err = config.SetReservationsOutOfPool(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetEarlyGlobalReservationsLookup(storkutil.Ptr(true))
	require.NoError(t, err)

	err = config.SetHostReservationIdentifiers([]string{"hw-address", "client-id"})
	require.NoError(t, err)

	err = config.SetPDAllocator(storkutil.Ptr("random"))
	require.NoError(t, err)

	serializedConfig, err := config.GetSerializedConfig()
	require.NoError(t, err)

	require.JSONEq(t, `{
		"Dhcp6": {
			"cache-threshold": 0.2,
			"ddns-generated-prefix": "myhost.example.org",
			"ddns-override-client-update": true,
			"ddns-override-no-update": true,
			"ddns-qualifying-suffix": "example.org",
			"ddns-replace-client-name": "never",
			"ddns-send-updates": true,
			"ddns-update-on-renew": true,
			"ddns-use-conflict-resolution": true,
			"ddns-ttl-percent": 0.1,
			"dhcp-ddns": {
				"enable-updates": true,
				"max-queue-size": 100,
				"ncr-format": "JSON",
				"ncr-protocol": "UDP",
				"sender-ip": "2001:db8:1::1",
				"sender-port": 8080,
				"server-ip": "2001:db8:1::2",
				"server-port": 8081
			},
			"early-global-reservations-lookup": true,
			"host-reservation-identifiers": [
				"hw-address",
				"client-id"
			],
			"reservations-global": true,
			"reservations-in-subnet": false,
			"reservations-out-of-pool": true,
			"allocator": "flq",
			"expired-leases-processing": {
				"flush-reclaimed-timer-wait-time": 111,
				"hold-reclaimed-time": 111,
				"max-reclaim-leases": 111,
				"max-reclaim-time": 2,
				"reclaim-timer-wait-time": 3,
				"unwarned-reclaim-cycles": 10
			},
			"pd-allocator": "random"
		}
	}`, serializedConfig)
}

// Test setting DHCP DDNS for the DHCPv6 server.
func TestSettingDHCPv6DHCPDDNS(t *testing.T) {
	config := NewSettableDHCPv6Config()

	dhcpDDNS := &SettableDHCPDDNS{
		EnableUpdates: storkutil.NewNullableFromValue(true),
		ServerIP:      storkutil.NewNullableFromValue("2001:db8:1::1"),
		ServerPort:    storkutil.NewNullableFromValue(int64(8080)),
		SenderIP:      storkutil.NewNullableFromValue("2001:db8:1::2"),
		SenderPort:    storkutil.NewNullableFromValue(int64(8081)),
		MaxQueueSize:  storkutil.NewNullableFromValue(int64(100)),
		NCRProtocol:   storkutil.NewNullableFromValue("UDP"),
		NCRFormat:     storkutil.NewNullableFromValue("JSON"),
	}
	err := config.SetDHCPDDNS(dhcpDDNS)
	require.NoError(t, err)

	serializedConfig, err := config.GetSerializedConfig()
	require.NoError(t, err)

	require.JSONEq(t, `{
		"Dhcp6": {
			"dhcp-ddns": {
				"enable-updates": true,
				"server-ip": "2001:db8:1::1",
				"server-port": 8080,
				"sender-ip": "2001:db8:1::2",
				"sender-port": 8081,
				"max-queue-size": 100,
				"ncr-format": "JSON",
				"ncr-protocol": "UDP"
			}
		}
	}`, serializedConfig)
}

// Test setting expired leases processing for DHCPv6 server.
func TestSettingDHCPv6ExpiredLeasesProcessing(t *testing.T) {
	config := NewSettableDHCPv6Config()

	expiredLeasesProcessing := &SettableExpiredLeasesProcessing{
		FlushReclaimedTimerWaitTime: storkutil.NewNullableFromValue(int64(6)),
		HoldReclaimedTime:           storkutil.NewNullableFromValue(int64(5)),
		MaxReclaimLeases:            storkutil.NewNullableFromValue(int64(4)),
		MaxReclaimTime:              storkutil.NewNullableFromValue(int64(3)),
		ReclaimTimerWaitTime:        storkutil.NewNullableFromValue(int64(2)),
		UnwarnedReclaimCycles:       storkutil.NewNullableFromValue(int64(1)),
	}
	err := config.SetExpiredLeasesProcessing(expiredLeasesProcessing)
	require.NoError(t, err)

	serializedConfig, err := config.GetSerializedConfig()
	require.NoError(t, err)

	require.JSONEq(t, `{
		"Dhcp6": {
			"expired-leases-processing": {
				"flush-reclaimed-timer-wait-time": 6,
				"hold-reclaimed-time": 5,
				"max-reclaim-leases": 4,
				"max-reclaim-time": 3,
				"reclaim-timer-wait-time": 2,
				"unwarned-reclaim-cycles": 1
			}
		}
	}`, serializedConfig)
}
