package dbmodel

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Test implementation of the keaconfig.SharedNetwork interface (GetName() function).
func TestSharedNetworkGetName(t *testing.T) {
	network := SharedNetwork{
		Name: "my-secret-network",
	}
	require.Equal(t, "my-secret-network", network.GetName())
}

// Test retrieving a local shared network instance by daemon ID.
func TestSharedNetworkGetLocalSharedNetwork(t *testing.T) {
	network := SharedNetwork{
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: 110,
			},
			{
				DaemonID: 111,
			},
		},
	}
	lsn := network.GetLocalSharedNetwork(110)
	require.NotNil(t, lsn)
	require.EqualValues(t, 110, lsn.DaemonID)

	lsn = network.GetLocalSharedNetwork(111)
	require.NotNil(t, lsn)
	require.EqualValues(t, 111, lsn.DaemonID)

	require.Nil(t, network.GetLocalSharedNetwork(112))
}

// Test implementation of the keaconfig.SharedNetwork interface (GetKeaParameters()
// function).
func TestSharedNetworkGetKeaParameters(t *testing.T) {
	network := SharedNetwork{
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: 110,
				KeaParameters: &keaconfig.SharedNetworkParameters{
					Allocator: storkutil.Ptr("random"),
				},
			},
			{
				DaemonID: 111,
				KeaParameters: &keaconfig.SharedNetworkParameters{
					Allocator: storkutil.Ptr("iterative"),
				},
			},
		},
	}
	params0 := network.GetKeaParameters(110)
	require.NotNil(t, params0)
	require.Equal(t, "random", *params0.Allocator)
	params1 := network.GetKeaParameters(111)
	require.NotNil(t, params1)
	require.Equal(t, "iterative", *params1.Allocator)

	require.Nil(t, network.GetKeaParameters(1000))
}

// Test implementation of the dhcpmodel.SharedNetworkAccessor interface
// (GetDHCPOptions() function).
func TestSharedNetworkGetDHCPOptions(t *testing.T) {
	hasher := keaconfig.NewHasher()
	network := SharedNetwork{
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: 110,
				DHCPOptionSet: NewDHCPOptionSet([]DHCPOption{
					{
						Code:  7,
						Space: dhcpmodel.DHCPv4OptionSpace,
					},
				}, hasher),
			},
			{
				DaemonID: 111,
				DHCPOptionSet: NewDHCPOptionSet([]DHCPOption{
					{
						Code:  8,
						Space: dhcpmodel.DHCPv4OptionSpace,
					},
				}, hasher),
			},
		},
	}
	options0 := network.GetDHCPOptions(110)
	require.Len(t, options0, 1)
	require.EqualValues(t, 7, options0[0].GetCode())

	options1 := network.GetDHCPOptions(111)
	require.Len(t, options1, 1)
	require.EqualValues(t, 8, options1[0].GetCode())

	require.Nil(t, network.GetDHCPOptions(1000))
}

// Test getting a list of subnets for a shared network by daemon ID.
func TestSharedNetworkGetSubnets(t *testing.T) {
	network := SharedNetwork{
		Subnets: []Subnet{
			{
				Prefix: "192.0.2.0/24",
				LocalSubnets: []*LocalSubnet{
					{
						DaemonID: 110,
					},
				},
			},
			{
				Prefix: "192.0.3.0/24",
				LocalSubnets: []*LocalSubnet{
					{
						DaemonID: 110,
					},
					{
						DaemonID: 111,
					},
				},
			},
		},
	}
	subnets := network.GetSubnets(110)
	require.Len(t, subnets, 2)
	require.Equal(t, "192.0.2.0/24", subnets[0].GetPrefix())
	require.Equal(t, "192.0.3.0/24", subnets[1].GetPrefix())

	subnets = network.GetSubnets(111)
	require.Len(t, subnets, 1)
	require.Equal(t, "192.0.3.0/24", subnets[0].GetPrefix())
}

// Test that daemon information can be populated to the existing
// shared network instance when LocalSharedNetwork instances merely
// contain DaemonID values.
func TestPopulateSharedNetworkDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps to the database.
	apps := addTestSubnetApps(t, db)

	// Create bare shared network that lacks Daemon instances but has valid
	// DaemonID values.
	sharedNetwork := &SharedNetwork{
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: apps[0].Daemons[0].ID,
			},
			{
				DaemonID: apps[1].Daemons[0].ID,
			},
		},
	}
	err := sharedNetwork.PopulateDaemons(db)
	require.NoError(t, err)

	// Make sure that the daemon information was assigned to the shared network.
	require.Len(t, sharedNetwork.LocalSharedNetworks, 2)
	require.NotNil(t, sharedNetwork.LocalSharedNetworks[0].Daemon)
	require.EqualValues(t, apps[0].Daemons[0].ID, sharedNetwork.LocalSharedNetworks[0].Daemon.ID)
	require.NotNil(t, sharedNetwork.LocalSharedNetworks[1].Daemon)
	require.EqualValues(t, apps[1].Daemons[0].ID, sharedNetwork.LocalSharedNetworks[1].Daemon.ID)
}

// Tests that the shared network can be added and retrieved.
func TestAddAndGetSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{Address: "127.0.0.1", AgentPort: 8080}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	app := &App{
		Type: AppTypeKea,
		Daemons: []*Daemon{{
			Name: DaemonNameDHCPv4,
			KeaDaemon: &KeaDaemon{
				ConfigHash: "hash",
			},
		}},
		MachineID: machine.ID,
	}
	daemons, err := AddApp(db, app)
	require.NoError(t, err)
	daemon := daemons[0]

	network := SharedNetwork{
		Name:   "funny name",
		Family: 6,
		Subnets: []Subnet{{
			Prefix: "2001:db8::/32",
		}},
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: daemon.ID,
			},
		},
	}

	err = AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	err = AddDaemonToSubnet(db, &network.Subnets[0], daemon)
	require.NoError(t, err)

	subnet, err := GetSubnet(db, network.Subnets[0].ID)
	require.NoError(t, err)

	err = AddAddressPool(db, &AddressPool{
		LowerBound:    "2001:db8::1",
		UpperBound:    "2001:db8::10",
		LocalSubnetID: subnet.LocalSubnets[0].ID,
	})
	require.NoError(t, err)

	err = AddPrefixPool(db, &PrefixPool{
		Prefix:        "2001:db8:1::/96",
		DelegatedLen:  120,
		LocalSubnetID: subnet.LocalSubnets[0].ID,
	})
	require.NoError(t, err)

	err = AddLocalSharedNetworks(db, &network)
	require.NoError(t, err)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, network.Name, returned.Name)
	require.NotZero(t, returned.CreatedAt)

	// Check the relations.
	require.NotEmpty(t, returned.LocalSharedNetworks)
	require.NotEmpty(t, returned.LocalSharedNetworks[0].Daemon)
	require.NotEmpty(t, returned.LocalSharedNetworks[0].Daemon.KeaDaemon)
	require.NotEmpty(t, returned.Subnets)
	require.NotEmpty(t, returned.Subnets[0].LocalSubnets)
	require.NotEmpty(t, returned.Subnets[0].LocalSubnets[0].Daemon)
	require.NotEmpty(t, returned.Subnets[0].LocalSubnets[0].Daemon.KeaDaemon)
	require.NotEmpty(t, returned.Subnets[0].LocalSubnets[0].AddressPools)
	require.NotEmpty(t, returned.Subnets[0].LocalSubnets[0].PrefixPools)
}

// Tests that the caller can specify list of shared network relations.
func TestGetSharedNetworkWithRelations(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{Address: "127.0.0.1", AgentPort: 8080}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	app := &App{
		Type: AppTypeKea,
		Daemons: []*Daemon{{
			Name: DaemonNameDHCPv4,
			KeaDaemon: &KeaDaemon{
				ConfigHash: "hash",
			},
		}},
		MachineID: machine.ID,
	}
	daemons, err := AddApp(db, app)
	require.NoError(t, err)
	daemon := daemons[0]

	network := SharedNetwork{
		Name:   "funny name",
		Family: 6,
		Subnets: []Subnet{{
			Prefix: "2001:db8::/32",
		}},
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: daemon.ID,
			},
		},
	}

	err = AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	err = AddDaemonToSubnet(db, &network.Subnets[0], daemon)
	require.NoError(t, err)

	subnet, err := GetSubnet(db, network.Subnets[0].ID)
	require.NoError(t, err)

	err = AddAddressPool(db, &AddressPool{
		LowerBound:    "2001:db8::1",
		UpperBound:    "2001:db8::10",
		LocalSubnetID: subnet.LocalSubnets[0].ID,
	})
	require.NoError(t, err)

	err = AddPrefixPool(db, &PrefixPool{
		Prefix:        "2001:db8:1::/96",
		DelegatedLen:  120,
		LocalSubnetID: subnet.LocalSubnets[0].ID,
	})
	require.NoError(t, err)

	err = AddLocalSharedNetworks(db, &network)
	require.NoError(t, err)

	returned, err := GetSharedNetworkWithRelations(
		db, network.ID,
		SharedNetworkRelationSubnetsAddressPools,
		SharedNetworkRelationSubnetsPrefixPools,
	)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, network.Name, returned.Name)
	require.NotZero(t, returned.CreatedAt)

	// Check the references.
	require.Empty(t, returned.LocalSharedNetworks)
	require.NotEmpty(t, returned.Subnets)
	require.NotEmpty(t, returned.Subnets[0].LocalSubnets)
	require.Empty(t, returned.Subnets[0].LocalSubnets[0].Daemon)
	require.NotEmpty(t, returned.Subnets[0].LocalSubnets[0].AddressPools)
	require.NotEmpty(t, returned.Subnets[0].LocalSubnets[0].PrefixPools)
}

// Test that a shared network with subnets and pools can be added and
// retrieved from the database.
func TestAddSharedNetworkWithSubnetsPools(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestApps(t, db)

	network := &SharedNetwork{
		Name:   "funny name",
		Family: 6,
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: apps[1].Daemons[1].ID,
			},
		},
		Subnets: []Subnet{
			{
				Prefix: "2001:db8:1::/64",
				LocalSubnets: []*LocalSubnet{
					{
						DaemonID: apps[1].Daemons[1].ID,
						AddressPools: []AddressPool{
							{
								LowerBound: "2001:db8:1::1",
								UpperBound: "2001:db8:1::10",
							},
							{
								LowerBound: "2001:db8:1::11",
								UpperBound: "2001:db8:1::20",
							},
						},
						PrefixPools: []PrefixPool{
							{
								Prefix:       "2001:db8:1:1::/96",
								DelegatedLen: 120,
							},
						},
					},
				},
			},
			{
				Prefix: "2001:db8:2::/64",
				LocalSubnets: []*LocalSubnet{
					{
						DaemonID: apps[1].Daemons[1].ID,
						AddressPools: []AddressPool{
							{
								LowerBound: "2001:db8:2::1",
								UpperBound: "2001:db8:2::10",
							},
						},
						PrefixPools: []PrefixPool{
							{
								Prefix:       "2001:db8:2:1::/96",
								DelegatedLen: 120,
							},
							{
								Prefix:       "3000::/64",
								DelegatedLen: 80,
							},
						},
					},
				},
			},
		},
	}
	err := AddSharedNetwork(db, network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	_, err = CommitNetworksIntoDB(db, []SharedNetwork{*network}, []Subnet{})
	require.NoError(t, err)

	// Create a common function verifying the contents of a shared network and its subnets.
	// It accepts the boolean flag indicating whether the shared network was
	// returned by a function fetching a list of shared networks (true) or
	// by a function fetching a single shared network (false).
	verifySharedNetworkFn := func(t *testing.T, returnedNetwork *SharedNetwork, listing bool, includeMachine bool) {
		require.Len(t, returnedNetwork.LocalSharedNetworks, 1)
		require.NotNil(t, returnedNetwork.LocalSharedNetworks[0].Daemon)

		if listing {
			// It must be nil to limit memory usage.
			require.Nil(t, returnedNetwork.LocalSharedNetworks[0].Daemon.KeaDaemon)
		} else {
			require.NotNil(t, returnedNetwork.LocalSharedNetworks[0].Daemon.KeaDaemon)
		}
		require.NotNil(t, returnedNetwork.LocalSharedNetworks[0].Daemon.App)

		if includeMachine {
			require.NotNil(t, returnedNetwork.LocalSharedNetworks[0].Daemon.App.Machine)
		} else {
			require.Nil(t, returnedNetwork.LocalSharedNetworks[0].Daemon.App.Machine)
		}
		require.Len(t, returnedNetwork.LocalSharedNetworks[0].Daemon.App.AccessPoints, 1)

		require.Len(t, returnedNetwork.Subnets, 2)

		for i, s := range returnedNetwork.Subnets {
			require.NotZero(t, s.ID)
			require.NotZero(t, s.CreatedAt)
			require.Equal(t, network.Subnets[i].Prefix, s.Prefix)
			require.Equal(t, returnedNetwork.ID, s.SharedNetworkID)

			require.Len(t, s.LocalSubnets, 1)
			require.NotNil(t, s.LocalSubnets[0].Daemon)
			require.Len(t, s.LocalSubnets[0].Daemon.App.AccessPoints, 1)
			require.NotNil(t, s.LocalSubnets[0].Daemon.App.Machine)

			if listing {
				// It must be nil to limit memory usage.
				require.Nil(t, s.LocalSubnets[0].Daemon.KeaDaemon)
				continue
			}

			// The below fields are not included in the list of entities.
			require.NotNil(t, s.LocalSubnets[0].Daemon.KeaDaemon)
			require.NotNil(t, s.LocalSubnets[0].Daemon.App.Machine)

			require.Len(t, s.LocalSubnets[0].AddressPools, len(network.Subnets[i].LocalSubnets[0].AddressPools))
			require.Len(t, s.LocalSubnets[0].PrefixPools, len(network.Subnets[i].LocalSubnets[0].PrefixPools))

			for j, p := range s.LocalSubnets[0].AddressPools {
				require.NotZero(t, p.ID)
				require.NotZero(t, p.CreatedAt)
				require.Equal(t, returnedNetwork.Subnets[i].LocalSubnets[0].AddressPools[j].LowerBound, p.LowerBound)
				require.Equal(t, returnedNetwork.Subnets[i].LocalSubnets[0].AddressPools[j].UpperBound, p.UpperBound)
			}

			for j, p := range s.LocalSubnets[0].PrefixPools {
				require.NotZero(t, p.ID)
				require.NotZero(t, p.CreatedAt)
				require.Equal(t, returnedNetwork.Subnets[i].LocalSubnets[0].PrefixPools[j].Prefix, p.Prefix)
			}
		}
	}

	t.Run("GetSharedNetwork", func(t *testing.T) {
		returnedNetwork, err := GetSharedNetwork(db, network.ID)
		require.NoError(t, err)
		require.NotNil(t, returnedNetwork)
		verifySharedNetworkFn(t, returnedNetwork, false, true)
	})

	t.Run("GetSharedNetworksByPage", func(t *testing.T) {
		returnedNetworks, total, err := GetSharedNetworksByPage(db, 0, 10, apps[1].ID, 6, nil, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, returnedNetworks, 1)
		verifySharedNetworkFn(t, &returnedNetworks[0], true, false)
	})

	t.Run("GetAllSharedNetworks", func(t *testing.T) {
		baseNetworks, err := GetAllSharedNetworks(db, 0)
		require.NoError(t, err)
		require.Len(t, baseNetworks, 1)
		require.Empty(t, baseNetworks[0].Subnets)
	})
}

// Test that family of the subnets being added within the shared network
// must match the shared network family.
func TestAddSharedNetworkSubnetsFamilyClash(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := &SharedNetwork{
		Name:   "funny name",
		Family: 6,
		Subnets: []Subnet{
			{
				Prefix: "2001:db8:1::/64",
			},
			{
				Prefix: "192.0.2.0/24",
			},
		},
	}
	err := AddSharedNetwork(db, network)
	require.Error(t, err)
}

func TestAddLocalSharedNetworks(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestSubnetApps(t, db)
	network := &SharedNetwork{
		Name:   "my name",
		Family: 4,
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: apps[0].Daemons[0].ID,
				KeaParameters: &keaconfig.SharedNetworkParameters{
					Authoritative: storkutil.Ptr(true),
					Allocator:     storkutil.Ptr("iterative"),
					Interface:     storkutil.Ptr("eth0"),
				},
			},
		},
	}
	err := AddSharedNetwork(db, network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	err = AddLocalSharedNetworks(db, network)
	require.NoError(t, err)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, "my name", returned.Name)

	require.Len(t, returned.LocalSharedNetworks, 1)
	require.EqualValues(t, returned.LocalSharedNetworks[0].DaemonID, apps[0].Daemons[0].ID)
	require.EqualValues(t, returned.LocalSharedNetworks[0].SharedNetworkID, network.ID)

	require.NotNil(t, returned.LocalSharedNetworks[0].KeaParameters)
	require.True(t, *returned.LocalSharedNetworks[0].KeaParameters.Authoritative)
	require.Equal(t, "iterative", *returned.LocalSharedNetworks[0].KeaParameters.Allocator)
	require.Equal(t, "eth0", *returned.LocalSharedNetworks[0].KeaParameters.Interface)
}

// Tests that shared networks can be fetched by family.
func TestGetSharedNetworksByFamily(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	networks := []SharedNetwork{
		{
			Name:   "fox",
			Family: 6,
		},
		{
			Name:   "frog",
			Family: 4,
		},
		{
			Name:   "fox",
			Family: 4,
		},
		{
			Name:   "snake",
			Family: 6,
		},
	}
	// Add two IPv4 and two IPv6 shared networks.
	for _, n := range networks {
		network := n
		err := AddSharedNetwork(db, &network)
		require.NoError(t, err)
		require.NotZero(t, network.ID)
	}

	// Get all shared networks without specifying family.
	returned, err := GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, returned, 4)
	require.Equal(t, "fox", returned[0].Name)
	require.Equal(t, "frog", returned[1].Name)
	require.Equal(t, "fox", returned[2].Name)
	require.Equal(t, "snake", returned[3].Name)

	// Get only IPv4 networks.
	returned, err = GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, returned, 2)
	require.Equal(t, "frog", returned[0].Name)
	require.Equal(t, "fox", returned[1].Name)

	// Get only IPv6 networks.
	returned, err = GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, returned, 2)
	require.Equal(t, "fox", returned[0].Name)
	require.Equal(t, "snake", returned[1].Name)
}

// Tests that the shared network information can be updated.
func TestUpdateSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name:   "funny name",
		Family: 4,
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	// Remember the creation time so it can be compared after the update.
	createdAt := network.CreatedAt
	require.NotZero(t, createdAt)

	// Reset creation time to ensure it is not modified during the update.
	network.CreatedAt = time.Time{}

	// update name and check if it was stored
	network.Name = "different name"
	err = UpdateSharedNetwork(db, &network)
	require.NoError(t, err)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, network.Name, returned.Name)

	// update utilization
	err = UpdateStatisticsInSharedNetwork(db, network.ID, newUtilizationStatsMock(0.01, 0.02, SubnetStats{
		"assigned-nas": uint64(1),
		"total-nas":    uint64(100),
		"assigned-pds": uint64(4),
		"total-pds":    uint64(200),
	}))
	require.NoError(t, err)

	returned, err = GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.EqualValues(t, 10, returned.AddrUtilization)
	require.EqualValues(t, 20, returned.PdUtilization)
	require.EqualValues(t, 100, returned.Stats["total-nas"])
	require.EqualValues(t, 200, returned.Stats["total-pds"])
	require.InDelta(t, time.Now().Unix(), returned.StatsCollectedAt.Unix(), 10.0)
	require.Equal(t, createdAt, returned.CreatedAt)
}

// Tests that the shared network can be deleted.
func TestDeleteSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name:   "funny name",
		Family: 4,
		Subnets: []Subnet{
			{
				Prefix: "192.0.2.0/24",
			},
		},
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	err = DeleteSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.Nil(t, returned)

	returnedSubnets, err := GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.NotEmpty(t, returnedSubnets)
	returnedSubnet := returnedSubnets[0]
	require.Equal(t, "192.0.2.0/24", returnedSubnet.Prefix)
	require.Nil(t, returnedSubnet.SharedNetwork)
	require.Zero(t, returnedSubnet.SharedNetworkID)
}

// Tests that deleting a shared network also deletes its subnets.
func TestDeleteSharedNetworkWithSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name:   "funny name",
		Family: 4,
		Subnets: []Subnet{
			{
				Prefix: "192.0.2.0/24",
			},
		},
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	err = DeleteSharedNetworkWithSubnets(db, network.ID)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.Nil(t, returned)

	returnedSubnets, err := GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Empty(t, returnedSubnets)
}

// Test deleting all empty shared networks.
func TestDeleteStaleSharedNetworks(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create three shared networks. Two of them have no subnets.
	networks := []SharedNetwork{
		{
			Name:   "with subnets",
			Family: 4,
			Subnets: []Subnet{
				{
					Prefix: "192.0.2.0/24",
				},
			},
		},
		{
			Name:   "without subnets",
			Family: 4,
		},
		{
			Name:   "again without subnets",
			Family: 4,
		},
	}
	for i := range networks {
		err := AddSharedNetwork(db, &networks[i])
		require.NoError(t, err)
		require.NotZero(t, networks[i].ID)
	}

	// Delete shared networks having no subnets.
	count, err := DeleteEmptySharedNetworks(db)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	// The first shared network should still exist.
	returned, err := GetSharedNetwork(db, networks[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	// The other two should have been deleted.
	returned, err = GetSharedNetwork(db, networks[1].ID)
	require.NoError(t, err)
	require.Nil(t, returned)

	returned, err = GetSharedNetwork(db, networks[2].ID)
	require.NoError(t, err)
	require.Nil(t, returned)

	// Deleting shared networks again should affect no shared networks.
	count, err = DeleteEmptySharedNetworks(db)
	require.NoError(t, err)
	require.Zero(t, count)
}

// Test that an association of a daemon with a shared network can be
// deleted, and other associations remain.
func TestDeleteDaemonFromSharedNetworks(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestApps(t, db)

	// Add a shared network.
	network := SharedNetwork{
		Name:   "my name",
		Family: 4,
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: apps[0].Daemons[0].ID,
			},
			{
				DaemonID: apps[1].Daemons[0].ID,
			},
		},
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)

	// Associate two daemons with the shared network.
	err = AddLocalSharedNetworks(db, &network)
	require.NoError(t, err)

	// Delete the first daemon's association with the shared network.
	n, err := DeleteDaemonFromSharedNetworks(db, apps[0].Daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	// Get the shared network from the database.
	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	// Ensure that the second association remains.
	require.Len(t, returned.LocalSharedNetworks, 1)
	require.EqualValues(t, apps[1].Daemons[0].ID, returned.LocalSharedNetworks[0].DaemonID)
}

// Test deleting a shared network associated with no daemons.
func TestDeleteOrphanedSharedNetworks(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestApps(t, db)

	// Add a shared network.
	network := SharedNetwork{
		Name:   "my name",
		Family: 4,
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: apps[0].Daemons[0].ID,
			},
			{
				DaemonID: apps[1].Daemons[0].ID,
			},
		},
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)

	// Associate two daemons with the shared network.
	err = AddLocalSharedNetworks(db, &network)
	require.NoError(t, err)

	// No shared networks should be deleted because the sole shared network
	// is associated with two daemons.
	n, err := DeleteOrphanedSharedNetworks(db)
	require.NoError(t, err)
	require.Zero(t, n)

	// Delete one of the associations.
	n, err = DeleteDaemonFromSharedNetworks(db, apps[0].Daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	// The shared network should not be deleted yet.
	n, err = DeleteOrphanedSharedNetworks(db)
	require.NoError(t, err)
	require.Zero(t, n)

	// Delete the second association.
	n, err = DeleteDaemonFromSharedNetworks(db, apps[1].Daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	// The shared network has no associations (is orphaned) and should be deleted.
	n, err = DeleteOrphanedSharedNetworks(db)
	require.NoError(t, err)
	require.EqualValues(t, n, 1)

	// Make sure that the shared network has been deleted.
	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.Nil(t, returned)
}

// Test that LocalSharedNetwork instance is appended to the SharedNetwork
// when there is no corresponding LocalSharedNetwork, and it is replaced
// when the corresponding LocalSharedNetwork exists.
func TestLocalSharedNetwork(t *testing.T) {
	// Create a shared network with one local shared network.
	sharedNetwork := SharedNetwork{
		Name: "foo",
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: 1,
				KeaParameters: &keaconfig.SharedNetworkParameters{
					Allocator: storkutil.Ptr("random"),
				},
			},
		},
	}
	// Create another local shared network and ensure there are now two.
	sharedNetwork.SetLocalSharedNetwork(&LocalSharedNetwork{
		DaemonID: 2,
	})
	require.Len(t, sharedNetwork.LocalSharedNetworks, 2)
	require.EqualValues(t, 1, sharedNetwork.LocalSharedNetworks[0].DaemonID)
	require.EqualValues(t, 2, sharedNetwork.LocalSharedNetworks[1].DaemonID)

	// Replace the first instance with a new one.
	sharedNetwork.SetLocalSharedNetwork(&LocalSharedNetwork{
		DaemonID: 1,
		KeaParameters: &keaconfig.SharedNetworkParameters{
			Allocator: storkutil.Ptr("iterative"),
		},
	})
	require.Len(t, sharedNetwork.LocalSharedNetworks, 2)
	require.EqualValues(t, 1, sharedNetwork.LocalSharedNetworks[0].DaemonID)
	require.EqualValues(t, 2, sharedNetwork.LocalSharedNetworks[1].DaemonID)
	require.NotNil(t, sharedNetwork.LocalSharedNetworks[0].KeaParameters)
	require.Equal(t, "iterative", *sharedNetwork.LocalSharedNetworks[0].KeaParameters.Allocator)
}

// Test that LocalSharedNetworks between two SharedNetwork instances can be
// combined in a single instance.
func TestJoinSharedNetworks(t *testing.T) {
	sharedNetwork0 := SharedNetwork{
		Name: "foo",
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: 1,
			},
			{
				DaemonID: 2,
			},
		},
	}
	sharedNetwork1 := SharedNetwork{
		Name: "foo",
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID: 2,
			},
			{
				DaemonID: 3,
			},
		},
	}
	sharedNetwork0.Join(&sharedNetwork1)
	require.Len(t, sharedNetwork0.LocalSharedNetworks, 3)
	require.EqualValues(t, 1, sharedNetwork0.LocalSharedNetworks[0].DaemonID)
	require.EqualValues(t, 2, sharedNetwork0.LocalSharedNetworks[1].DaemonID)
	require.EqualValues(t, 3, sharedNetwork0.LocalSharedNetworks[2].DaemonID)
}

// Test that deleting daemons from a shared network removes the associations of
// the shared network with daemons but also the associations of the subnets
// with daemons.
func TestDeleteDaemonsFromSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestSubnetApps(t, db)

	// Add some shared networks with subnets. Each of them is associated
	// with multiple daemons.
	var networks []*SharedNetwork
	for i := 0; i < 2; i++ {
		network := &SharedNetwork{
			Name:   fmt.Sprintf("network%d", i),
			Family: 4,
			LocalSharedNetworks: []*LocalSharedNetwork{
				{
					DaemonID: apps[0].Daemons[0].ID,
				},
				{
					DaemonID: apps[1].Daemons[0].ID,
				},
			},
			Subnets: []Subnet{
				{
					Prefix: fmt.Sprintf("192.0.%d.0/24", i),
					LocalSubnets: []*LocalSubnet{
						{
							DaemonID: apps[0].Daemons[0].ID,
						},
						{
							DaemonID: apps[1].Daemons[0].ID,
						},
					},
				},
			},
		}
		err := AddSharedNetwork(db, network)
		require.NoError(t, err)
		require.NotZero(t, network.ID)

		err = AddLocalSharedNetworks(db, network)
		require.NoError(t, err)

		err = AddLocalSubnets(db, &network.Subnets[0])
		require.NoError(t, err)

		networks = append(networks, network)
	}

	// Ensure that the shared network is associated with multiple daemons and
	// the subnets are also associated with multiple daemons.
	network0, err := GetSharedNetwork(db, networks[0].ID)
	require.NoError(t, err)
	require.Len(t, network0.LocalSharedNetworks, 2)
	require.Len(t, network0.Subnets, 1)
	require.Len(t, network0.Subnets[0].LocalSubnets, 2)

	// Delete daemons from the first shared network.
	err = DeleteDaemonsFromSharedNetwork(db, network0.ID)
	require.NoError(t, err)

	// Ensure that neither the shared network nor the subnets are
	// associated with the daemons.
	network0, err = GetSharedNetwork(db, networks[0].ID)
	require.NoError(t, err)
	require.NotNil(t, network0)
	require.Empty(t, network0.LocalSharedNetworks, 0)
	require.Len(t, network0.Subnets, 1)
	require.Empty(t, network0.Subnets[0].LocalSubnets, 0)

	// It should not affect the second shared network.
	network1, err := GetSharedNetwork(db, networks[1].ID)
	require.NoError(t, err)
	require.NotNil(t, network1)
	require.Len(t, network1.LocalSharedNetworks, 2)
	require.Len(t, network1.Subnets, 1)
	require.Len(t, network1.Subnets[0].LocalSubnets, 2)
}
