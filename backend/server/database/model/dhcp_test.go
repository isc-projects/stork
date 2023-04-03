package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
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

	cfg, err := NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
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

	cfg, err := NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Adds apps to be used with subnet tests.
func addTestSubnetApps(t *testing.T, db *dbops.PgDB) (apps []*App) {
	// Add two apps.
	for i := 0; i < 2; i++ {
		m := &Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		accessPoints := []*AccessPoint{}
		accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "cool.example.org", "", int64(1234+i), true)

		a := &App{
			ID:           0,
			MachineID:    m.ID,
			Type:         AppTypeKea,
			Active:       true,
			AccessPoints: accessPoints,
			Daemons: []*Daemon{
				{
					Name: DaemonNameDHCPv4,
					KeaDaemon: &KeaDaemon{
						Config: getTestConfigWithIPv4Subnets(t),
					},
				},
				{
					Name: DaemonNameDHCPv6,
					KeaDaemon: &KeaDaemon{
						Config: getTestConfigWithIPv6Subnets(t),
					},
				},
			},
		}

		apps = append(apps, a)
	}

	// Add the apps to be database.
	for _, app := range apps {
		_, err := AddApp(db, app)
		require.NoError(t, err)
	}
	return apps
}
