package kea

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	require "github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
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
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := dbmodel.App{
		MachineID:    m.ID,
		Machine:      m,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config:        kea4Config,
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
			{
				Name:   "dhcp6",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config:        kea6Config,
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
		},
	}
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	return &app
}

// Multi step test which verifies that the subnets and shared networks can be
// updated in the database for each newly added or updated app.
func TestDetectNetworksWhenAppCommitted(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

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
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err := CommitAppIntoDB(db, app, fec, nil, lookup)
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
	require.EqualValues(t, app.Daemons[0].ID, reservations[0].LocalHosts[0].DaemonID)
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
	require.EqualValues(t, app.Daemons[1].ID, reservations[0].LocalHosts[0].DaemonID)

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
	err = CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	// There should be one shared network in the database.
	networks, err = dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, networks, 1)

	// Make sure that the number of subnets stored for the shared network is 2.
	network, err := dbmodel.GetSharedNetwork(db, networks[0].ID)
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
	// The first host belongs to two daemons.
	require.Len(t, hosts[0].LocalHosts, 2)
	require.NotEqual(t, hosts[0].LocalHosts[0].DaemonID, app.Daemons[0].ID)
	require.Equal(t, app.Daemons[0].ID, hosts[0].LocalHosts[1].DaemonID)
	// The second host belongs to one daemon.
	require.Len(t, hosts[1].LocalHosts, 1)
	require.Equal(t, app.Daemons[0].ID, hosts[1].LocalHosts[0].DaemonID)

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
	err = CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	// There should still be just one shared network.
	networks, err = dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, networks, 1)

	// It should now contain 3 subnets.
	network, err = dbmodel.GetSharedNetwork(db, networks[0].ID)
	require.NoError(t, err)
	require.Len(t, network.Subnets, 3)

	// Adding the same subnet again should be fine and should not result in
	// any conflicts.
	err = CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	networks, err = dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, networks, 1)

	network, err = dbmodel.GetSharedNetwork(db, networks[0].ID)
	require.NoError(t, err)
	require.Len(t, network.Subnets, 3)
}

// Test that subnets are not committed to the database for a daemon which
// have been marked as having the same config since last update.
func TestCommitAppSameConfigs(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	// Create the Kea app supporting DHCPv4 and DHCPv6.
	v4Config := `
        {
            "Dhcp4": {
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24"
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
                        "subnet": "2001:db8:2::/64"
                    }
                ]
            }
        }`

	// Indicate that the configuration for a DHCPv4 daemon hasn't changed.
	state := &AppStateMeta{
		SameConfigDaemons: map[string]bool{
			dbmodel.DaemonNameDHCPv4: true,
		},
	}
	app := createAppWithSubnets(t, db, 0, v4Config, v6Config)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err := CommitAppIntoDB(db, app, fec, state, lookup)
	require.NoError(t, err)

	// There should be no IPv4 subnets because they should have been skipped.
	subnets, err := dbmodel.GetAllSubnets(db, 4)
	require.NoError(t, err)
	require.Empty(t, subnets)

	// There should be 2 IPv6 subnets created.
	subnets, err = dbmodel.GetAllSubnets(db, 6)
	require.NoError(t, err)
	require.Len(t, subnets, 2)
}

// Test that moving a subnet outside of the shared network does not leave
// old configurations in the database.
func TestDetectNetworksMoveSubnetsAround(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	v4Config := `
        {
            "Dhcp4": {
                "shared-networks": [
                    {
                        "name": "foo",
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
                ]
            }
        }`
	app := createAppWithSubnets(t, db, 0, v4Config, "")
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err := CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	// Shared network should have been created along with the subnets.
	networks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, networks, 1)

	// Ensure that the shared network holds two subnets.
	network, err := dbmodel.GetSharedNetwork(db, networks[0].ID)
	require.NoError(t, err)
	require.Len(t, network.Subnets, 2)

	// Create new configuration which moves one of the subnets outside of
	// the shared network.
	v4Config0 := `
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
                    }
                ],
                "shared-networks": [
                    {
                        "name": "foo",
                        "subnet4": [
                            {
                                "subnet": "192.0.3.0/24"
                            }
                        ]
                    }
                ]
            }
        }`
	kea4Config0, err := dbmodel.NewKeaConfigFromJSON(v4Config0)
	require.NoError(t, err)

	app.Daemons[0].KeaDaemon.Config = kea4Config0
	err = CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	// The shared network should still be there.
	networks, err = dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, networks, 1)

	// Both subnets should exist in the database.
	subnets, err := dbmodel.GetAllSubnets(db, 4)
	require.NoError(t, err)
	require.Len(t, subnets, 2)

	// Ensure that only one subnet is now in our shared network.
	network, err = dbmodel.GetSharedNetwork(db, networks[0].ID)
	require.NoError(t, err)
	require.Len(t, network.Subnets, 1)
	require.Equal(t, "192.0.3.0/24", network.Subnets[0].Prefix)

	// Move the second subnet outside of the shared network.
	v4Config1 := `
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
	kea4Config1, err := dbmodel.NewKeaConfigFromJSON(v4Config1)
	require.NoError(t, err)

	app.Daemons[0].KeaDaemon.Config = kea4Config1
	err = CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	// The shared network should now be gone.
	networks, err = dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Empty(t, networks)

	// Revert to the original config.
	app.Daemons[0].KeaDaemon.Config = kea4Config0
	err = CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	// The shared network should be there.
	networks, err = dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, networks, 1)

	// Verify the subnet prefix within the shared network.
	network, err = dbmodel.GetSharedNetwork(db, networks[0].ID)
	require.NoError(t, err)
	require.Len(t, network.Subnets, 1)
	require.Equal(t, "192.0.3.0/24", network.Subnets[0].Prefix)
}

// Test that the subnets not assigned to any apps are removed as a
// result of an app configuration update.
func TestDetectNetworksRemoveOrphanedSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	apps := make([]*dbmodel.App, 2)

	// Create a configuration with a single subnet.
	v4Config := `
        {
            "Dhcp4": {
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24"
                    }
                ]
            }
        }`
	// Assign the same configuration to two apps.
	for i := 0; i < len(apps); i++ {
		apps[i] = createAppWithSubnets(t, db, int64(i), v4Config, "")
		err := CommitAppIntoDB(db, apps[i], fec, nil, lookup)
		require.NoError(t, err)
	}

	// Ensure there is a single subnet instance in the database.
	subnets, err := dbmodel.GetAllSubnets(db, 4)
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	// Update the first app to use a different subnet.
	v4Config = `
        {
            "Dhcp4": {
                "subnet4": [
                    {
                        "subnet": "192.0.3.0/24"
                    }
                ]
            }
        }`
	kea4Config, err := dbmodel.NewKeaConfigFromJSON(v4Config)
	require.NoError(t, err)

	apps[0].Daemons[0].KeaDaemon.Config = kea4Config
	err = CommitAppIntoDB(db, apps[0], fec, nil, lookup)
	require.NoError(t, err)

	// There should still be two subnets in the database, each owned
	// by a different app.
	subnets, err = dbmodel.GetAllSubnets(db, 4)
	require.NoError(t, err)
	require.Len(t, subnets, 2)
	require.Len(t, subnets[0].LocalSubnets, 1)
	require.Len(t, subnets[1].LocalSubnets, 1)

	// Update the second app to use the second subnet.
	apps[1].Daemons[0].KeaDaemon.Config = kea4Config
	err = CommitAppIntoDB(db, apps[1], fec, nil, lookup)
	require.NoError(t, err)

	// The first subnet should have been removed because the second
	// subnet is now associated with both apps.
	subnets, err = dbmodel.GetAllSubnets(db, 4)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.Len(t, subnets[0].LocalSubnets, 2)
	require.Equal(t, "192.0.3.0/24", subnets[0].Prefix)
}

// Test that the hosts not assigned to any apps are removed as a
// result of an app configuration update.
func TestDetectNetworksRemoveOrphanedHosts(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	apps := make([]*dbmodel.App, 2)

	// Create a configuration with a single host reservation.
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
                            }
                        ]
                    }
                ]
            }
        }`
	// Assign the same configuration to two apps.
	for i := 0; i < len(apps); i++ {
		apps[i] = createAppWithSubnets(t, db, int64(i), v4Config, "")
		err := CommitAppIntoDB(db, apps[i], fec, nil, lookup)
		require.NoError(t, err)
	}

	// Ensure there is a single host reservation instance in the database.
	subnets, err := dbmodel.GetAllSubnets(db, 4)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	hosts, err := dbmodel.GetHostsBySubnetID(db, subnets[0].ID)
	require.NoError(t, err)
	require.Len(t, hosts, 1)

	// Update the first app to use a different reservation.
	v4Config = `
        {
            "Dhcp4": {
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24",
                        "reservations": [
                            {
                                "hw-address": "01:02:03:04:05:07",
                                "ip-address": "192.0.2.66"
                            }
                        ]
                    }
                ]
            }
        }`
	kea4Config, err := dbmodel.NewKeaConfigFromJSON(v4Config)
	require.NoError(t, err)

	apps[0].Daemons[0].KeaDaemon.Config = kea4Config
	err = CommitAppIntoDB(db, apps[0], fec, nil, lookup)
	require.NoError(t, err)

	// There should be two hosts in the database, each owned by a
	// different app.
	hosts, err = dbmodel.GetHostsBySubnetID(db, subnets[0].ID)
	require.NoError(t, err)
	require.Len(t, hosts, 2)

	// Update the second app to use the second reservation.
	apps[1].Daemons[0].KeaDaemon.Config = kea4Config
	err = CommitAppIntoDB(db, apps[1], fec, nil, lookup)
	require.NoError(t, err)

	// The first host should have been removed because the second
	// host is now associated with both apps.
	subnets, err = dbmodel.GetAllSubnets(db, 4)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	hosts, err = dbmodel.GetHostsBySubnetID(db, subnets[0].ID)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Len(t, hosts[0].LocalHosts, 2)

	// Ensure that the correct host is in the database.
	require.Len(t, hosts[0].LocalHosts[0].IPReservations, 1)
	require.Equal(t, "192.0.2.66/32", hosts[0].LocalHosts[0].IPReservations[0].Address)
	require.Len(t, hosts[0].LocalHosts[1].IPReservations, 1)
	require.Equal(t, "192.0.2.66/32", hosts[0].LocalHosts[1].IPReservations[0].Address)
}

// Utility shorthand alias.
type m = map[string]any

// Test that the address pools are updated.
func TestDetectNetworkUpdateAddressPool(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	v4Config := m{
		"Dhcp4": m{
			"subnet4": []m{
				{
					"subnet": "192.0.2.0/24",
					"pools": []m{
						{"pool": "192.0.2.1 - 192.0.2.10"},
					},
				},
			},
		},
	}

	v4ConfigJSON, _ := json.Marshal(v4Config)
	app := createAppWithSubnets(t, db, 0, string(v4ConfigJSON), "")
	_ = CommitAppIntoDB(db, app, fec, nil, lookup)

	// Act
	// Update the config.
	v4Config["Dhcp4"].(m)["subnet4"].([]m)[0]["pools"].([]m)[0]["pool"] = "192.0.2.1 - 192.0.2.42"
	v4ConfigJSON, _ = json.Marshal(v4Config)
	kea4Config, _ := dbmodel.NewKeaConfigFromJSON(string(v4ConfigJSON))
	app.Daemons[0].KeaDaemon.Config = kea4Config
	err := CommitAppIntoDB(db, app, fec, nil, lookup)

	// Assert
	require.NoError(t, err)
	subnets, _ := dbmodel.GetAllSubnets(db, 4)
	require.Len(t, subnets, 1)
	require.Len(t, subnets[0].LocalSubnets, 1)
	require.Len(t, subnets[0].LocalSubnets[0].AddressPools, 1)
	require.EqualValues(t, "192.0.2.1", subnets[0].LocalSubnets[0].AddressPools[0].LowerBound)
	require.EqualValues(t, "192.0.2.42", subnets[0].LocalSubnets[0].AddressPools[0].UpperBound)
}

// Test that the client class is updated.
func TestDetectNetworkUpdateClientClass(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	v4Config := m{
		"Dhcp4": m{
			"subnet4": []m{
				{
					"subnet":       "192.0.2.0/24",
					"client-class": "foo",
				},
			},
		},
	}

	v4ConfigJSON, _ := json.Marshal(v4Config)
	app := createAppWithSubnets(t, db, 0, string(v4ConfigJSON), "")
	err := CommitAppIntoDB(db, app, fec, nil, lookup)
	require.NoError(t, err)

	// Act
	// Update the config.
	v4Config["Dhcp4"].(m)["subnet4"].([]m)[0]["client-class"] = "bar"
	v4ConfigJSON, _ = json.Marshal(v4Config)
	kea4Config, _ := dbmodel.NewKeaConfigFromJSON(string(v4ConfigJSON))
	app.Daemons[0].KeaDaemon.Config = kea4Config
	err = CommitAppIntoDB(db, app, fec, nil, lookup)

	// Assert
	require.NoError(t, err)
	subnets, _ := dbmodel.GetAllSubnets(db, 4)
	require.Len(t, subnets, 1)
	require.EqualValues(t, "bar", subnets[0].ClientClass)
}

// Test that the delegated prefix pools are updated.
func TestDetectNetworkUpdateDelegatedPrefixPool(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	config := m{
		"Dhcp6": m{
			"subnet6": []m{
				{
					"subnet": "fe80::/32",
					"pd-pools": []m{
						{
							"prefix":        "fe80:1::",
							"prefix-len":    64,
							"delegated-len": 80,
						},
					},
				},
			},
		},
	}

	configJSON, _ := json.Marshal(config)
	app := createAppWithSubnets(t, db, 0, "", string(configJSON))
	_ = CommitAppIntoDB(db, app, fec, nil, lookup)

	// Act
	// Update the config.
	config["Dhcp6"].(m)["subnet6"].([]m)[0]["pd-pools"].([]m)[0]["prefix"] = "fe80:42::"
	config["Dhcp6"].(m)["subnet6"].([]m)[0]["pd-pools"].([]m)[0]["prefix-len"] = 72
	config["Dhcp6"].(m)["subnet6"].([]m)[0]["pd-pools"].([]m)[0]["delegated-len"] = 92
	configJSON, _ = json.Marshal(config)
	keaConfig, _ := dbmodel.NewKeaConfigFromJSON(string(configJSON))
	app.Daemons[1].KeaDaemon.Config = keaConfig
	err := CommitAppIntoDB(db, app, fec, nil, lookup)

	// Assert
	require.NoError(t, err)
	subnets, _ := dbmodel.GetAllSubnets(db, 6)
	require.Len(t, subnets, 1)
	require.Len(t, subnets[0].LocalSubnets, 1)
	require.Len(t, subnets[0].LocalSubnets[0].PrefixPools, 1)
	require.EqualValues(t, "fe80:42::/72", subnets[0].LocalSubnets[0].PrefixPools[0].Prefix)
	require.EqualValues(t, 92, subnets[0].LocalSubnets[0].PrefixPools[0].DelegatedLen)
}

// Benchmark measuring performance of the findMatchingSubnet function. This
// function checks if the given subnet belongs to the set of existing subnets.
// It uses indexing by prefix to lookup an existing subnet.
func BenchmarkFindMatchingSubnet(b *testing.B) {
	// Create many subnets.
	subnets := []dbmodel.Subnet{}
	for i := 0; i < 10000; i++ {
		subnet := dbmodel.Subnet{
			Prefix: fmt.Sprintf("%d.%d.%d.%d/24", byte(i>>24), byte(i>>16), byte(i>>8), byte(i)),
		}
		subnets = append(subnets, subnet)
	}
	// Index the subnets.
	existingSubnets := dbmodel.NewIndexedSubnets(subnets)
	existingSubnets.Populate()

	// The actual benchmark starts here.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Randomize the subnet to be looked up.
		subnetIndex := rand.Intn(len(subnets))
		// Find the subnet using indexes.
		findMatchingSubnet(&subnets[subnetIndex], existingSubnets)
	}
}
