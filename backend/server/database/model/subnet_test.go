package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

func TestAddSubnetWithAddressPools(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	subnet := &Subnet{
		Prefix: "192.0.2.0/24",
		AddressPools: []AddressPool{
			{
				LowerBound: "192.0.2.1",
				UpperBound: "192.0.2.10",
			},
			{
				LowerBound: "192.0.2.11",
				UpperBound: "192.0.2.20",
			},
			{
				LowerBound: "192.0.2.21",
				UpperBound: "192.0.2.30",
			},
		},
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	returned, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	require.Equal(t, subnet.Prefix, returned.Prefix)
	require.Len(t, subnet.AddressPools, 3)
	require.Empty(t, subnet.PrefixPools)
}

func TestAddSubnetWithPrefixPools(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	subnet := &Subnet{
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
			{
				LowerBound: "2001:db8:1::21",
				UpperBound: "2001:db8:1::30",
			},
		},
		PrefixPools: []PrefixPool{
			{
				Prefix:       "3001::/64",
				DelegatedLen: 80,
			},
			{
				Prefix:       "3000::/32",
				DelegatedLen: 120,
			},
			{
				Prefix:       "2001:db8:2::/64",
				DelegatedLen: 96,
			},
		},
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	returned, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.NotZero(t, returned.Created)
	require.Equal(t, subnet.Prefix, returned.Prefix)
	require.Zero(t, returned.SharedNetworkID)
	require.Nil(t, returned.SharedNetwork)

	require.Len(t, returned.AddressPools, 3)
	require.Len(t, returned.PrefixPools, 3)

	for i, p := range returned.AddressPools {
		require.NotZero(t, p.Created)
		require.Equal(t, subnet.AddressPools[i].LowerBound, p.LowerBound)
		require.Equal(t, subnet.AddressPools[i].UpperBound, p.UpperBound)
	}

	for i, p := range returned.PrefixPools {
		require.NotZero(t, p.Created)
		require.Equal(t, subnet.PrefixPools[i].Prefix, p.Prefix)
		require.Equal(t, subnet.PrefixPools[i].DelegatedLen, p.DelegatedLen)
	}
}

func TestAddSubnetWithExistingSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	sharedNetwork := &SharedNetwork{
		Name: "test",
	}
	err := AddSharedNetwork(db, sharedNetwork)
	require.NoError(t, err)
	require.NotZero(t, sharedNetwork.ID)

	subnet := &Subnet{
		Prefix:          "2001:db8:1::/64",
		SharedNetworkID: sharedNetwork.ID,
	}
	err = AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	returnedSubnet, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.NotNil(t, returnedSubnet.SharedNetwork)
	require.Equal(t, returnedSubnet.SharedNetwork.ID, returnedSubnet.SharedNetworkID)
}
