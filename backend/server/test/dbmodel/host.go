package storktestdbmodel

import (
	"fmt"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// This function creates multiple hosts used in tests which fetch and
// filter hosts.
func AddTestHosts(t *testing.T, db *pg.DB) (hosts []dbmodel.Host, apps []dbmodel.App) {
	// Add two apps.
	for i := 0; i < 2; i++ {
		m := &dbmodel.Machine{
			ID:        0,
			Address:   "cool.example.org",
			AgentPort: int64(8080 + i),
		}
		err := dbmodel.AddMachine(db, m)
		require.NoError(t, err)

		accessPoints := []*dbmodel.AccessPoint{}
		accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", int64(1234+i), true)

		a := dbmodel.App{
			ID:           0,
			MachineID:    m.ID,
			Type:         dbmodel.AppTypeKea,
			Name:         fmt.Sprintf("dhcp-server%d", i),
			Active:       true,
			AccessPoints: accessPoints,
			Daemons: []*dbmodel.Daemon{
				dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true),
				dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv6, true),
			},
		}
		err = a.Daemons[0].SetConfigFromJSON(`{
            "Dhcp4": {
				"client-classes": [
					{
						"name": "class2"
					},
					{
						"name": "class1"
					}
				],
                "subnet4": [
                    {
                        "id": 111,
                        "subnet": "192.0.2.0/24"
                    }
                ],
                "hooks-libraries": [
                    {
                        "library": "libdhcp_host_cmds.so"
                    }
                ]
            }
        }`)
		require.NoError(t, err)

		err = a.Daemons[1].SetConfigFromJSON(`{
            "Dhcp6": {
				"client-classes": [
					{
						"name": "class2"
					},
					{
						"name": "class3"
					}
				],
                "subnet6": [
                    {
                        "id": 222,
                        "subnet": "2001:db8:1::/64"
                    }
                ],
                "hooks-libraries": [
                    {
                        "library": "libdhcp_host_cmds.so"
                    }
                ]
            }
        }`)
		require.NoError(t, err)
		apps = append(apps, a)
	}

	subnets := []dbmodel.Subnet{
		{
			ID:     1,
			Prefix: "192.0.2.0/24",
		},
		{
			ID:     2,
			Prefix: "2001:db8:1::/64",
		},
	}
	for i, s := range subnets {
		subnet := s
		err := dbmodel.AddSubnet(db, &subnet)
		require.NoError(t, err)
		require.NotZero(t, subnet.ID)
		subnets[i] = subnet
	}

	// Add apps to the database.
	for i, a := range apps {
		app := a
		_, err := dbmodel.AddApp(db, &app)
		require.NoError(t, err)
		require.NotZero(t, app.ID)
		// Associate the daemons with the subnets.
		for j := range apps[i].Daemons {
			err = dbmodel.AddDaemonToSubnet(db, &subnets[j], apps[i].Daemons[j])
			require.NoError(t, err)
		}
		apps[i] = app
	}

	hasher := keaconfig.NewHasher()
	hosts = []dbmodel.Host{
		{
			SubnetID: 1,
			Hostname: "first.example.org",
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
				{
					Type:  "circuit-id",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "192.0.2.4",
				},
				{
					Address: "192.0.2.5",
				},
			},
			LocalHosts: []dbmodel.LocalHost{
				{
					DaemonID:       apps[0].Daemons[0].ID,
					DataSource:     dbmodel.HostDataSourceAPI,
					NextServer:     "192.2.2.2",
					ServerHostname: "stork.example.org",
					BootFileName:   "/tmp/boot.xyz",
				},
				{
					DaemonID:       apps[1].Daemons[0].ID,
					DataSource:     dbmodel.HostDataSourceAPI,
					NextServer:     "192.2.2.2",
					ServerHostname: "stork.example.org",
					BootFileName:   "/tmp/boot.xyz",
				},
			},
		},
		{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{2, 3, 4, 5, 6, 7},
				},
				{
					Type:  "circuit-id",
					Value: []byte{2, 3, 4, 5},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "192.0.2.6",
				},
				{
					Address: "192.0.2.7",
				},
			},
			LocalHosts: []dbmodel.LocalHost{
				{
					DaemonID:   apps[0].Daemons[0].ID,
					DataSource: dbmodel.HostDataSourceAPI,
				},
				{
					DaemonID:   apps[1].Daemons[0].ID,
					DataSource: dbmodel.HostDataSourceAPI,
				},
			},
		},
		{
			SubnetID: 2,
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "2001:db8:1::1",
				},
			},
			LocalHosts: []dbmodel.LocalHost{
				{
					DaemonID:   apps[0].Daemons[1].ID,
					DataSource: dbmodel.HostDataSourceAPI,
				},
				{
					DaemonID:   apps[1].Daemons[1].ID,
					DataSource: dbmodel.HostDataSourceAPI,
				},
			},
		},
		{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "duid",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "2001:db8:1::2",
				},
			},
			LocalHosts: []dbmodel.LocalHost{
				{
					DaemonID:   apps[0].Daemons[1].ID,
					DataSource: dbmodel.HostDataSourceAPI,
				},
				{
					DaemonID:   apps[1].Daemons[1].ID,
					DataSource: dbmodel.HostDataSourceAPI,
				},
			},
		},
		{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "duid",
					Value: []byte{2, 2, 2, 2},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "3000::/48",
				},
			},
			LocalHosts: []dbmodel.LocalHost{
				{
					DaemonID:   apps[0].Daemons[1].ID,
					DataSource: dbmodel.HostDataSourceAPI,
					ClientClasses: []string{
						"foo",
						"bar",
					},
					DHCPOptionSet: dbmodel.NewDHCPOptionSet([]dbmodel.DHCPOption{
						{
							Code: 23,
							Fields: []dbmodel.DHCPOptionField{
								{
									FieldType: dhcpmodel.IPv6AddressField,
									Values:    []any{"3001:dbef:1e5::"},
								},
								{
									FieldType: dhcpmodel.IPv6AddressField,
									Values:    []any{"3002:abc::"},
								},
							},
							Name:     "dns-servers",
							Space:    dhcpmodel.DHCPv6OptionSpace,
							Universe: storkutil.IPv6,
						},
					}, hasher),
				},
				{
					DaemonID:   apps[1].Daemons[1].ID,
					DataSource: dbmodel.HostDataSourceAPI,
					ClientClasses: []string{
						"foo",
						"bar",
					},
					DHCPOptionSet: dbmodel.NewDHCPOptionSet([]dbmodel.DHCPOption{
						{
							Code: 23,
							Fields: []dbmodel.DHCPOptionField{
								{
									FieldType: dhcpmodel.IPv6AddressField,
									Values:    []any{"3001:dbef:1e5::"},
								},
								{
									FieldType: dhcpmodel.IPv6AddressField,
									Values:    []any{"3002:abc::"},
								},
							},
							Name:     "dns-servers",
							Space:    dhcpmodel.DHCPv6OptionSpace,
							Universe: storkutil.IPv6,
						},
					}, hasher),
				},
			},
		},
	}

	// Add hosts to the database.
	for i, h := range hosts {
		host := h
		err := dbmodel.AddHostWithLocalHosts(db, &host)
		require.NoError(t, err)
		require.NotZero(t, host.ID)
		hosts[i] = host
	}
	return hosts, apps
}
