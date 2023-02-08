package keaconfig

import (
	"fmt"
	"testing"

	require "github.com/stretchr/testify/require"
)

// Returns test Kea configuration lacking hooks libraries configurations.
func getTestConfigWithoutHooks(t *testing.T) *Map {
	configStr := `{
        "Dhcp4": {
            "valid-lifetime": 1000
        }
    }`

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration with empty list of hooks libraries.
func getTestConfigEmptyHooks(t *testing.T) *Map {
	configStr := `{
        "Dhcp4": {
            "valid-lifetime": 1000,
            "high-availability": [ ]
        }
    }`

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration including two hooks libraries.
func getTestConfigWithHooks(t *testing.T) *Map {
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

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test HA configuration which lacks some parameters.
func getTestMinimalHAConfig(t *testing.T) *Map {
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

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

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

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Tests that Logging key is ignored as non-root key.
func TestGetRootNameNoRoot(t *testing.T) {
	cfg, err := NewFromJSON(`
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

// Tests that new configuration map can be created.
func TestNewMap(t *testing.T) {
	cfg := map[string]interface{}{
		"key": "value",
	}
	c := New(&cfg)
	require.NotNil(t, c)
	require.Contains(t, *c, "key")
}

// Tests that the configuration root key can be found.
func TestGetRootName(t *testing.T) {
	cfg, err := NewFromJSON(`
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
	cfg := HA{}

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

// Verifies that a list of loggers is parsed correctly for a daemon.
func TestGetLoggers(t *testing.T) {
	configStr := `{
        "Dhcp4": {
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

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

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

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

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

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	sockets := cfg.GetControlSockets()

	names := sockets.ConfiguredDaemonNames()
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

	cfg, err = NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	sockets = cfg.GetControlSockets()

	// This time only two sockets have been configured.
	names = sockets.ConfiguredDaemonNames()
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
		cfg, err := NewFromJSON(configStr)
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
		cfg, err := NewFromJSON(configStr)
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
		cfg, err := NewFromJSON(configStr)
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
		cfg, err := NewFromJSON(configStr)
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
		cfg, err := NewFromJSON(configStr)
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
		cfg, err := NewFromJSON(configStr)
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
		cfg, err := NewFromJSON(configStr)
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
	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationModes()
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
	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationModes()
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
	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationModes()
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
	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationModes()
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
	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationModes()
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
	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationModes()
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
	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	modes := cfg.GetGlobalReservationModes()
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
	modes := []ReservationModes{
		{
			OutOfPool: nil,
		},
		{
			OutOfPool: new(bool),
		},
		{
			OutOfPool: new(bool),
		},
	}
	*modes[2].OutOfPool = true

	require.True(t, IsInAnyReservationModes(func(modes ReservationModes) (bool, bool) {
		return modes.IsOutOfPool()
	}, modes[0], modes[1], modes[2]))

	require.True(t, IsInAnyReservationModes(func(modes ReservationModes) (bool, bool) {
		return modes.IsOutOfPool()
	}, modes[0], modes[2], modes[1]))

	require.False(t, IsInAnyReservationModes(func(modes ReservationModes) (bool, bool) {
		return modes.IsOutOfPool()
	}, modes[1], modes[0]))

	require.False(t, IsInAnyReservationModes(func(modes ReservationModes) (bool, bool) {
		return modes.IsOutOfPool()
	}, modes[0], modes[0]))
}

// Test that the sensitive data are hidden.
func TestHideSensitiveData(t *testing.T) {
	// Arrange
	input := map[string]interface{}{
		"foo":      "bar",
		"password": "xxx",
		"token":    "",
		"secret":   "aaa",
		"first": map[string]interface{}{
			"foo":      "baz",
			"Password": 42,
			"Token":    nil,
			"Secret":   "bbb",
			"second": map[string]interface{}{
				"foo":      "biz",
				"passworD": true,
				"tokeN":    "yyy",
				"secreT":   "ccc",
			},
		},
	}

	keaMap := New(&input)

	// Act
	keaMap.HideSensitiveData()
	data := *keaMap

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
	cfg, err := NewFromJSON(configStr)
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
	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	clientClasses := cfg.GetClientClasses()
	require.Empty(t, clientClasses)
}

// Test that client classes can be deleted from the configuration.
func TestDeleteClientClasses(t *testing.T) {
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
	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Delete client classes multiple times and make sure they
	// are gone and there is no panic.
	for i := 0; i < 2; i++ {
		cfg.DeleteClientClasses()
		clientClasses := cfg.GetClientClasses()
		require.Empty(t, clientClasses)
	}
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

// Test that it is possible to parse subnet4 list into a custom
// structure.
func TestDecodeIPv4TopLevelSubnets(t *testing.T) {
	cfg := getTestConfigWithIPv4Subnets(t)

	subnets := []struct {
		Subnet string
	}{}
	err := cfg.DecodeTopLevelSubnets(&subnets)
	require.NoError(t, err)
	require.Len(t, subnets, 3)
	require.Equal(t, "192.0.2.0/24", subnets[0].Subnet)
	require.Equal(t, "192.0.3.0/24", subnets[1].Subnet)
	require.Equal(t, "10.0.0.0/8", subnets[2].Subnet)
}

// Test that it is possible to parse subnet6 list into a custom
// structure.
func TestDecodeIPv6TopLevelSubnets(t *testing.T) {
	cfg := getTestConfigWithIPv6Subnets(t)

	subnets := []struct {
		Subnet  string
		PdPools []struct {
			Prefix       string
			PrefixLen    int
			DelegatedLen int
		}
	}{}
	err := cfg.DecodeTopLevelSubnets(&subnets)
	require.NoError(t, err)
	require.Len(t, subnets, 3)
	require.Equal(t, "2001:db8:1::/64", subnets[0].Subnet)
	require.Equal(t, "2001:db8:2::/64", subnets[1].Subnet)
	require.Equal(t, "2001:db8:3::/64", subnets[2].Subnet)

	require.Len(t, subnets[1].PdPools, 1)
}

// Test that an error is returned when the subnet6 list
// is malformed, i.e., does not match the specified structure.
func TestDecodeMalformedSubnets(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": 1234
                }
            ]
        }
    }`

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	subnets := []struct {
		Subnet string
	}{}
	err = cfg.DecodeTopLevelSubnets(&subnets)
	require.Error(t, err)
}

// Test that reservation modes can be parsed at various configuration
// levels, i.e., top-level subnet, shared network and subnet embedded
// in a shared network.
func TestDecodeIPv4SubnetsWithHostReservationModes(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet4": [
                        {
                            "id": 567,
                            "subnet": "10.1.0.0/16",
                            "reservation-mode": "global"
                        }
                    ],
                    "reservations-in-subnet": true,
                    "reservations-out-of-pool": true
                }
            ],
            "subnet4": [
                {
                    "id": 123,
                    "subnet": "192.0.2.0/24",
                    "reservations-in-subnet": true
                }
            ]
        }
    }`

	cfg, err := NewFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Parse the top-level subnet.
	subnets := []struct {
		Subnet string
		ReservationModes
	}{}
	err = cfg.DecodeTopLevelSubnets(&subnets)
	require.NoError(t, err)

	require.Len(t, subnets, 1)
	val, set := subnets[0].ReservationModes.IsInSubnet()
	require.True(t, val)
	require.True(t, set)
	val, set = subnets[0].ReservationModes.IsOutOfPool()
	require.False(t, val)
	require.False(t, set)
	val, set = subnets[0].ReservationModes.IsGlobal()
	require.False(t, val)
	require.False(t, set)

	// Parse the shared network.
	networks := []struct {
		Name    string
		Subnet4 []struct {
			Subnet string
			ReservationModes
		}
		ReservationModes
	}{}
	err = cfg.DecodeSharedNetworks(&networks)
	require.NoError(t, err)
	require.Len(t, networks, 1)
	val, set = networks[0].ReservationModes.IsInSubnet()
	require.True(t, val)
	require.True(t, set)
	val, set = networks[0].ReservationModes.IsOutOfPool()
	require.True(t, val)
	require.True(t, set)
	val, set = networks[0].ReservationModes.IsGlobal()
	require.False(t, val)
	require.False(t, set)

	// Validate the reservation modes specified for the subnet within
	// the shared network.
	require.Len(t, networks[0].Subnet4, 1)
	val, set = networks[0].Subnet4[0].ReservationModes.IsGlobal()
	require.True(t, val)
	require.True(t, set)
	val, set = networks[0].Subnet4[0].ReservationModes.IsInSubnet()
	require.False(t, val)
	require.True(t, set)
	val, set = networks[0].Subnet4[0].ReservationModes.IsOutOfPool()
	require.False(t, val)
	require.True(t, set)
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

	config, _ := NewFromJSON(configStr)

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
	config, _ := NewFromJSON(configStr)

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
	config, _ := NewFromJSON(configStr)

	// Act
	multiThreading := config.GetMultiThreading()

	// Assert
	require.Nil(t, multiThreading)
}
