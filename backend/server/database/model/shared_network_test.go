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
		Name: "funny name",
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, network.Name, returned.Name)
	require.NotZero(t, returned.Created)
}

func TestAddSharedNetworkWithSubnetsPools(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := &SharedNetwork{
		Name: "funny name",
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
		require.NotZero(t, s.Created)
		require.Equal(t, network.Subnets[i].Prefix, s.Prefix)
		require.Equal(t, returnedNetwork.ID, s.SharedNetworkID)

		require.Len(t, s.AddressPools, len(network.Subnets[i].AddressPools))
		require.Len(t, s.PrefixPools, len(network.Subnets[i].PrefixPools))

		for j, p := range s.AddressPools {
			require.NotZero(t, p.ID)
			require.NotZero(t, p.Created)
			require.Equal(t, returnedNetwork.Subnets[i].AddressPools[j].LowerBound, p.LowerBound)
			require.Equal(t, returnedNetwork.Subnets[i].AddressPools[j].UpperBound, p.UpperBound)
		}

		for j, p := range s.PrefixPools {
			require.NotZero(t, p.ID)
			require.NotZero(t, p.Created)
			require.Equal(t, returnedNetwork.Subnets[i].PrefixPools[j].Prefix, p.Prefix)
		}
	}
}

// Tests that the shared network information can be updated.
func TestUpdateSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name: "funny name",
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	network.Name = "different name"
	err = UpdateSharedNetwork(db, &network)
	require.NoError(t, err)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, network.Name, returned.Name)
}

// Tests that the shared network can be deleted.
func TestDeleteSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name: "funny name",
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

	returnedSubnet, err := GetSubnetByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Equal(t, "192.0.2.0/24", returnedSubnet.Prefix)
	require.Nil(t, returnedSubnet.SharedNetwork)
	require.Zero(t, returnedSubnet.SharedNetworkID)
}

func TestDeleteSharedNetworkWithSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name: "funny name",
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

	returnedSubnet, err := GetSubnetByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Nil(t, returnedSubnet)
}
