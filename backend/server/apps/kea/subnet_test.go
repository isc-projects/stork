package kea

import (
	"testing"

	require "github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Creates an app instance used in the tests. The index value should be incremented
// for each new app to make sure that the address/address port tuple inserted to
// the database is unique. The DHCPv4 and DHCPv6 configurations provided as text.
// If any of them is empty, it is ignored. The created app instance is inserted
// to the database and then returned to the unit test.
func createAppWithSubnets(t *testing.T, db *dbops.PgDB, index int64, v4Config, v6Config string) *dbmodel.App {
	// Add the machine.
	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080 + index,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// DHCPv4 configuration.
	var kea4Config *dbmodel.KeaConfig
	if len(v4Config) > 0 {
		kea4Config, err = dbmodel.NewKeaConfigFromJSON(v4Config)
		require.NoError(t, err)
	}

	// DHCPv6 configuration.
	var kea6Config *dbmodel.KeaConfig
	if len(v6Config) > 0 {
		kea6Config, err = dbmodel.NewKeaConfigFromJSON(v6Config)
		require.NoError(t, err)
	}

	// Creates new app with provided configurations.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{
				{
					Name:   "dhcp4",
					Config: kea4Config,
				},
				{
					Name:   "dhcp6",
					Config: kea6Config,
				},
			},
		},
	}
	// Add the app to the database.
	err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	return &app
}

// Multi step test which verifies that the subnets and shared networks can be
// created and matched when new Kea app instances are being added.
func TestDetectNetworks(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	v4Config := `
        {
            "Dhcp4": {
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24",
                        "reservations": [
                            {
                                "hw-address": "01:02:03:04:05:06",
                                "ip-address": "192.0.2.55"
                            },
                            {
                                "duid": "0A:0A:0A:0A:0A:0A",
                                "ip-address": "192.0.2.55"
                            }
                        ]
                    },
                    {
                        "subnet": "192.0.3.0/24"
                    }
                ]
            }
        }`

	v6Config := `
        {
            "Dhcp6": {
                "subnet6": [
                    {
                        "subnet": "2001:db8:1::/64",
                        "reservations": [
                            {
                                "duid": "01:02:03:04",
                                "ip-addresses": [ "2001:db8:1::55" ]
                            }
                        ]
                    },
                    {
                        "subnet": "2001:db8:2::/64"
                    }
                ]
            }
        }`
	app := createAppWithSubnets(t, db, 0, v4Config, v6Config)

	// First case: there are no subnets nor shared networks in the database.
	networks, subnets, err := DetectNetworks(db, app)
	require.NoError(t, err)
	// The configuration lacks shared networks so they should not be
	// returned.
	require.Empty(t, networks)
	// There should be 4 new subnets returned as a result of processing
	// the configurations of this app.
	require.Len(t, subnets, 4)

	// Verify that the subnets are correct.
	for i, s := range subnets {
		// The app is not associated automatically with the subnets. This is
		// to indicate that the subnet is not associated with the app until
		// explicitly requested.
		require.Empty(t, s.LocalSubnets)
		// The subnets haven't been added to the database yet. Their IDs should
		// be 0. That also allows to determine that this is a new subnet.
		require.Zero(t, s.ID)

		// Even subnets should include reservations and odd subnets should not.
		if i%2 == 0 {
			require.Len(t, subnets[i].Hosts, 1)
			require.Empty(t, subnets[i].Hosts[0].LocalHosts)
		} else {
			require.Empty(t, subnets[i].Hosts)
		}

		// Ok let's add the subnet to the db and associate this subnet with
		// our app.
		subnet := s
		err = dbmodel.AddSubnet(db, &subnet)
		require.NoError(t, err)
		err = dbmodel.AddAppToSubnet(db, &subnet, app)
		require.NoError(t, err)

		// Add the host to the database
		for _, h := range subnets[i].Hosts {
			host := h
			err = dbmodel.AddHost(db, &host)
			require.NoError(t, err)
			require.NotZero(t, host.ID)
			err = dbmodel.AddAppToHost(db, &host, app, "config")
			require.NoError(t, err)
		}
	}

	// Second case: introducing a shared network. Note that the top level subnet
	// already exists in the database.
	v4Config = `
        {
            "Dhcp4": {
                "shared-networks": [
                    {
                        "name": "foo",
                        "subnet4": [
                            {
                                "subnet": "10.0.0.0/8",
                                "reservations": [
                                    {
                                        "hw-address": "02:02:02:02:02:02",
                                        "ip-address": "10.1.1.1"
                                    }
                                ]
                            }
                        ]
                    }
                ],
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24",
                        "reservations": [
                            {
                                "hw-address": "01:02:03:04:05:06",
                                "ip-address": "192.0.2.55"
                            },
                            {
                                "duid": "0A:0A:0A:0A:0A:0A",
                                "ip-address": "192.0.2.55"
                            },
                            {
                                "hw-address": "09:09:09:09:09:09",
                                "ip-address": "192.0.2.66"
                            }
                        ]
                    }
                ]
            }
        }`
	app = createAppWithSubnets(t, db, 1, v4Config, "")

	networks, subnets, err = DetectNetworks(db, app)
	require.NoError(t, err)
	// This time we should get one shared network in return.
	require.Len(t, networks, 1)
	newNetwork := networks[0]

	// This is new shared network so its ID should be 0 until explicitly
	// added to the database.
	require.Zero(t, newNetwork.ID)
	require.Equal(t, "foo", newNetwork.Name)
	// The shared network contains one subnet.
	require.Len(t, newNetwork.Subnets, 1)
	// This is new subnet so the ID should be unset.
	require.Zero(t, newNetwork.Subnets[0].ID)
	require.Equal(t, "10.0.0.0/8", newNetwork.Subnets[0].Prefix)
	// The reservation should exist for this subnet.
	require.Len(t, newNetwork.Subnets[0].Hosts, 1)
	require.Empty(t, newNetwork.Subnets[0].Hosts[0].LocalHosts)
	// The subnet is not associated with any apps until such association
	// is explicitly made.
	require.Empty(t, newNetwork.Subnets[0].LocalSubnets)

	// Add the shared network and the subnet it contains to the database.
	err = dbmodel.AddSharedNetwork(db, &newNetwork)
	require.NoError(t, err)

	// The subnet ID should have been set after being added to the database.
	newSubnet := newNetwork.Subnets[0]
	require.NotZero(t, newSubnet.ID)

	// The association of the subnet and the app must be done explicitly.
	err = dbmodel.AddAppToSubnet(db, &newSubnet, app)
	require.NoError(t, err)

	newHost := newSubnet.Hosts[0]
	err = dbmodel.AddHost(db, &newHost)
	require.NoError(t, err)
	require.NotZero(t, newHost.ID)
	err = dbmodel.AddAppToHost(db, &newHost, app, "config")
	require.NoError(t, err)

	// Verify that we have one top level subnet.
	require.Len(t, subnets, 1)
	newSubnet = subnets[0]

	// This subnet was already present in the database, therefore it should
	// already have an ID.
	require.NotZero(t, newSubnet.ID)
	require.Equal(t, "192.0.2.0/24", newSubnet.Prefix)
	// Also this subnet should be already associated with the previous app.
	require.Len(t, newSubnet.LocalSubnets, 1)

	// This subnet should now have two reservations.
	require.Len(t, newSubnet.Hosts, 2)
	require.NotZero(t, newSubnet.Hosts[0].ID)
	require.Len(t, newSubnet.Hosts[0].LocalHosts, 1)
	require.Zero(t, newSubnet.Hosts[1].ID)
	require.Empty(t, newSubnet.Hosts[1].LocalHosts)

	// Add association of our new app with that subnet.
	err = dbmodel.AddAppToSubnet(db, &newSubnet, app)
	require.NoError(t, err)

	// Third case: two shared networks of which one already exists.
	v4Config = `
        {
            "Dhcp4": {
                "shared-networks": [
                    {
                        "name": "foo",
                        "subnet4": [
                            {
                                "subnet": "10.0.0.0/8",
                                "reservations": [
                                    {
                                        "hw-address": "02:02:02:02:02:02",
                                        "ip-address": "10.1.1.1"
                                    },
                                    {
                                        "hw-address": "03:03:03:03:03:03",
                                        "ip-address": "10.2.2.2"
                                    }
                                ]
                            },
                            {
                                "subnet": "10.1.0.0/16"
                            }
                        ]
                    },
                    {
                        "name": "bar",
                        "subnet4": [
                            {
                                "subnet": "192.0.3.0/24"
                            },
                            {
                                "subnet": "192.0.4.0/24"
                            }
                        ]
                    }
                ]
            }
        }`

	// Add IPv6 configuration with a shared network having the same name
	// as the IPv4 shared network.
	v6Config = `
        {
            "Dhcp6": {
                "shared-networks": [
                    {
                        "name": "foo",
                        "subnet6": [
                            {
                                "subnet": "3000:1::/32"
                            }
                        ]
                    }
                ]
            }
        }`
	app = createAppWithSubnets(t, db, 2, v4Config, v6Config)

	networks, subnets, err = DetectNetworks(db, app)
	require.NoError(t, err)
	require.Len(t, networks, 3)
	// This time there should be no top level subnets.
	require.Empty(t, subnets)

	// First shared network was already in the database so it should have
	// non zero id.
	require.NotZero(t, networks[0].ID)
	require.Equal(t, "foo", networks[0].Name)
	// It should now include two subnets: one old and one new.
	require.Len(t, networks[0].Subnets, 2)

	// The first subnet already existed in the database so it should
	// have non zero id.
	require.NotZero(t, networks[0].Subnets[0].ID)
	require.Equal(t, "10.0.0.0/8", networks[0].Subnets[0].Prefix)
	// Also, this subnet already had an association with one of the apps.
	require.Len(t, networks[0].Subnets[0].LocalSubnets, 1)

	// There should now be two hosts for this subnet. One already existed
	// and the other one is new.
	require.Len(t, networks[0].Subnets[0].Hosts, 2)

	// The second subnet is new and therefore has id of 0.
	require.Zero(t, networks[0].Subnets[1].ID)
	require.Equal(t, "10.1.0.0/16", networks[0].Subnets[1].Prefix)
	// Also, it is not associated with any apps yet.
	require.Empty(t, networks[0].Subnets[1].LocalSubnets)

	// The second shared network is brand new. The subnets in it are
	// also brand new.
	require.Equal(t, "bar", networks[1].Name)
	require.Len(t, networks[1].Subnets, 2)
	require.Zero(t, networks[1].Subnets[0].ID)
	require.Equal(t, "192.0.3.0/24", networks[1].Subnets[0].Prefix)
	require.Empty(t, networks[1].Subnets[0].LocalSubnets)
	require.Zero(t, networks[1].Subnets[1].ID)
	require.Equal(t, "192.0.4.0/24", networks[1].Subnets[1].Prefix)
	require.Empty(t, networks[1].Subnets[1].LocalSubnets)

	// The third shared network has the same name as the first one,
	// but it should be separate from the first one because it
	// contains IPv6 subnet.
	require.Equal(t, "foo", networks[2].Name)
	require.Zero(t, networks[2].ID)
	require.Len(t, networks[2].Subnets, 1)
	require.Zero(t, networks[2].Subnets[0].ID)
	require.Equal(t, "3000:1::/32", networks[2].Subnets[0].Prefix)
}

// Multi step test which verifies that the subnets and shared networks can be
// updated in the database for each newly added or updated app.
func TestDetectNetworksWhenAppCommitted(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create the Kea app supporting DHCPv4 and DHCPv6.
	v4Config := `
        {
            "Dhcp4": {
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24",
                        "reservations": [
                            {
                                "hw-address": "01:02:03:04:05:06",
                                "ip-address": "192.0.2.100"
                            }
                        ]
                    },
                    {
                        "subnet": "192.0.3.0/24"
                    }
                ]
            }
        }`

	v6Config := `
        {
            "Dhcp6": {
                "subnet6": [
                    {
                        "subnet": "2001:db8:1::/64"
                    },
                    {
                        "subnet": "2001:db8:2::/64",
                        "reservations": [
                            {
                                "duid": "01:02:03:04",
                                "ip-addresses": [
                                    "2001:db8:2::100", "2001:db8:2::101"
                                ],
                                "prefixes": [
                                    "3000:1::/96", "3000:2::/96"
                                ]
                            }
                        ]
                    }
                ]
            }
        }`
	app := createAppWithSubnets(t, db, 0, v4Config, v6Config)
	err := CommitAppIntoDB(db, app)
	require.NoError(t, err)

	// The configuration didn't include any shared network, so it should
	// be empty initially.
	networks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Empty(t, networks)

	// There should be 2 IPv4 subnets created.
	subnets, err := dbmodel.GetAllSubnets(db, 4)
	require.NoError(t, err)
	require.Len(t, subnets, 2)

	// The first subnet should have one reservation.
	reservations, err := dbmodel.GetHostsBySubnetID(db, subnets[0].ID)
	require.NoError(t, err)
	require.Len(t, reservations, 1)
	require.EqualValues(t, subnets[0].ID, reservations[0].SubnetID)
	require.Len(t, reservations[0].LocalHosts, 1)
	require.EqualValues(t, app.ID, reservations[0].LocalHosts[0].AppID)
	// The second subnet should have no reservations.
	reservations, err = dbmodel.GetHostsBySubnetID(db, subnets[1].ID)
	require.NoError(t, err)
	require.Empty(t, reservations)

	// There should be 2 IPv6 subnets created.
	subnets, err = dbmodel.GetAllSubnets(db, 6)
	require.NoError(t, err)
	require.Len(t, subnets, 2)

	// The first subnet should have no reservations.
	reservations, err = dbmodel.GetHostsBySubnetID(db, subnets[0].ID)
	require.NoError(t, err)
	require.Empty(t, reservations)
	// The second subnet should have one reservation.
	reservations, err = dbmodel.GetHostsBySubnetID(db, subnets[1].ID)
	require.NoError(t, err)
	require.Len(t, reservations, 1)
	require.EqualValues(t, subnets[1].ID, reservations[0].SubnetID)
	require.Len(t, reservations[0].LocalHosts, 1)
	require.EqualValues(t, app.ID, reservations[0].LocalHosts[0].AppID)

	// Create another Kea app which introduces a shared network and for
	// which the subnets partially overlaps.
	v4Config = `
        {
            "Dhcp4": {
                "shared-networks": [
                    {
                        "name": "foo",
                        "subnet4": [
                            {
                                "subnet": "10.0.0.0/8"
                            },
                            {
                                "subnet": "10.1.0.0/16"
                            }
                        ]
                    }
                ],
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24",
                        "reservations": [
                            {
                                "hw-address": "01:02:03:04:05:06",
                                "ip-address": "192.0.2.100"
                            },
                            {
                                "hw-address": "02:02:02:02:02:02",
                                "ip-address": "192.0.2.111"
                            }
                        ]
                    },
                    {
                        "subnet": "192.0.4.0/24"
                    }
                ]
            }
        }`
	app = createAppWithSubnets(t, db, 1, v4Config, "")
	err = CommitAppIntoDB(db, app)
	require.NoError(t, err)

	// There should be one shared network in the database.
	networks, err = dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, networks, 1)

	// Make sure that the number of subnets stored for the shared network is 2.
	network, err := dbmodel.GetSharedNetworkWithSubnets(db, networks[0].ID)
	require.NoError(t, err)
	require.Len(t, network.Subnets, 2)

	// The total number of subnets should be 7.
	subnets, err = dbmodel.GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Len(t, subnets, 7)

	subnets, err = dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	// Verify that the hosts within the 192.0.2.0/24 subnet have
	// been updated.
	hosts, err := dbmodel.GetHostsBySubnetID(db, subnets[0].ID)
	require.NoError(t, err)
	require.Len(t, hosts, 2)
	// Both hosts should have non zero ID and should belong to the same subnet.
	for _, h := range hosts {
		require.NotZero(t, h.ID)
		require.EqualValues(t, subnets[0].ID, h.SubnetID)
	}
	// The first host belongs to two apps.
	require.Len(t, hosts[0].LocalHosts, 2)
	require.NotEqual(t, hosts[0].LocalHosts[0].AppID, app.ID)
	require.Equal(t, app.ID, hosts[0].LocalHosts[1].AppID)
	// The second host belongs to one app.
	require.Len(t, hosts[1].LocalHosts, 1)
	require.Equal(t, app.ID, hosts[1].LocalHosts[0].AppID)

	// Let's add another app with the same shared network and new subnet in it.
	v4Config = `
        {
            "Dhcp4": {
                "shared-networks": [
                    {
                        "name": "foo",
                        "subnet4": [
                            {
                                "subnet": "10.2.0.0/16"
                            }
                        ]
                    }
                ],
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24"
                    },
                    {
                        "subnet": "192.0.5.0/24"
                    }
                ]
            }
        }`
	app = createAppWithSubnets(t, db, 2, v4Config, "")
	err = CommitAppIntoDB(db, app)
	require.NoError(t, err)

	// There should still be just one shared network.
	networks, err = dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, networks, 1)

	// It should now contain 3 subnets.
	network, err = dbmodel.GetSharedNetworkWithSubnets(db, networks[0].ID)
	require.NoError(t, err)
	require.Len(t, network.Subnets, 3)

	// Adding the same subnet again should be fine and should not result in
	// any conflicts.
	err = CommitAppIntoDB(db, app)
	require.NoError(t, err)

	networks, err = dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, networks, 1)

	network, err = dbmodel.GetSharedNetworkWithSubnets(db, networks[0].ID)
	require.NoError(t, err)
	require.Len(t, network.Subnets, 3)
}
