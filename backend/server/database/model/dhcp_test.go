package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
	dbops "isc.org/stork/server/database"
)

// Returns test Kea configuration including multiple IPv4 subnets.
func getTestConfigWithIPv4Subnets(t *testing.T) *KeaConfig {
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

	config, err := keaconfig.NewConfig([]byte(configStr))
	require.NoError(t, err)

	return newKeaConfig(config)
}

// Returns test Kea configuration including multiple IPv6 subnets.
func getTestConfigWithIPv6Subnets(t *testing.T) *KeaConfig {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet6": [
                        {
                            "id": 567,
                            "subnet": "3001::/16"
                        },
                        {
                            "id": 678,
                            "subnet": "3002::/16"
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

	config, err := keaconfig.NewConfig([]byte(configStr))
	require.NoError(t, err)

	return newKeaConfig(config)
}

// Adds daemons to be used with subnet tests.
func addTestSubnetDaemons(t *testing.T, db *dbops.PgDB) (daemons []*Daemon) {
	// Add two machines with daemons.
	for i := 0; i < 2; i++ {
		m := &Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		accessPoints := []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "cool.example.org",
				Port:    int64(1234 + i),
				Key:     "",
			},
		}

		// Create DHCPv4 daemon
		daemon4 := NewDaemon(m, daemonname.DHCPv4, true, accessPoints)
		daemon4.KeaDaemon.Config = getTestConfigWithIPv4Subnets(t)
		err = AddDaemon(db, daemon4)
		require.NoError(t, err)
		daemons = append(daemons, daemon4)

		// Create DHCPv6 daemon
		daemon6 := NewDaemon(m, daemonname.DHCPv6, true, accessPoints)
		daemon6.KeaDaemon.Config = getTestConfigWithIPv6Subnets(t)
		err = AddDaemon(db, daemon6)
		require.NoError(t, err)
		daemons = append(daemons, daemon6)
	}
	return daemons
}
