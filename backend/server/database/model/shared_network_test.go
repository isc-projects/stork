package dbmodel

import (
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"

	"testing"
)

// Tests that the shared network can be added and retrieved.
func TestAddSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name:   "funny name",
		Family: 6,
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, network.Name, returned.Name)
	require.NotZero(t, returned.CreatedAt)
}

func TestAddSharedNetworkWithSubnetsPools(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := &SharedNetwork{
		Name:   "funny name",
		Family: 6,
		Subnets: []Subnet{
			{
				Prefix: "2001:db8:1::/64",
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
			{
				Prefix: "2001:db8:2::/64",
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
	}
	err := AddSharedNetwork(db, network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	returnedNetwork, err := GetSharedNetworkWithSubnets(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedNetwork)

	require.Len(t, returnedNetwork.Subnets, 2)

	for i, s := range returnedNetwork.Subnets {
		require.NotZero(t, s.ID)
		require.NotZero(t, s.CreatedAt)
		require.Equal(t, network.Subnets[i].Prefix, s.Prefix)
		require.Equal(t, returnedNetwork.ID, s.SharedNetworkID)

		require.Len(t, s.AddressPools, len(network.Subnets[i].AddressPools))
		require.Len(t, s.PrefixPools, len(network.Subnets[i].PrefixPools))

		for j, p := range s.AddressPools {
			require.NotZero(t, p.ID)
			require.NotZero(t, p.CreatedAt)
			require.Equal(t, returnedNetwork.Subnets[i].AddressPools[j].LowerBound, p.LowerBound)
			require.Equal(t, returnedNetwork.Subnets[i].AddressPools[j].UpperBound, p.UpperBound)
		}

		for j, p := range s.PrefixPools {
			require.NotZero(t, p.ID)
			require.NotZero(t, p.CreatedAt)
			require.Equal(t, returnedNetwork.Subnets[i].PrefixPools[j].Prefix, p.Prefix)
		}
	}

	baseNetworks, err := GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, baseNetworks, 1)
	require.Empty(t, baseNetworks[0].Subnets)
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

	// update name and check if it was stored
	network.Name = "different name"
	err = UpdateSharedNetwork(db, &network)
	require.NoError(t, err)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, network.Name, returned.Name)

	// update utilization
	err = UpdateUtilizationInSharedNetwork(db, network.ID, 10, 20)
	require.NoError(t, err)

	returned, err = GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.EqualValues(t, 10, returned.AddrUtilization)
	require.EqualValues(t, 20, returned.PdUtilization)
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
