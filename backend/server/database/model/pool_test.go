package dbmodel

import (
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"

	"testing"
)

func TestAddDeleteAddressPool(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	subnet := Subnet{
		Prefix: "192.0.2.0/24",
	}
	err := AddSubnet(db, &subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	pool := AddressPool{
		LowerBound: "192.0.2.10",
		UpperBound: "192.0.2.20",
		Subnet: &Subnet{
			ID: subnet.ID,
		},
	}
	err = AddAddressPool(db, &pool)
	require.NoError(t, err)

	returnedSubnets, err := GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.NotEmpty(t, returnedSubnets)
	returnedSubnet := returnedSubnets[0]
	require.Len(t, returnedSubnet.AddressPools, 1)
	require.NotZero(t, returnedSubnet.AddressPools[0].ID)
	require.NotZero(t, returnedSubnet.AddressPools[0].Created)
	require.Equal(t, "192.0.2.10", returnedSubnet.AddressPools[0].LowerBound)
	require.Equal(t, "192.0.2.20", returnedSubnet.AddressPools[0].UpperBound)

	err = DeleteAddressPool(db, pool.ID)
	require.NoError(t, err)
	returnedSubnets, err = GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.NotEmpty(t, returnedSubnets)
	returnedSubnet = returnedSubnets[0]
	require.Empty(t, returnedSubnet.AddressPools)
}

func TestAddDeletePrefixPool(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	subnet := Subnet{
		Prefix: "2001:db8:1::/64",
	}
	err := AddSubnet(db, &subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	pool := PrefixPool{
		Prefix:       "2001:db8:1:1::/80",
		DelegatedLen: 96,
		Subnet: &Subnet{
			ID: subnet.ID,
		},
	}
	err = AddPrefixPool(db, &pool)
	require.NoError(t, err)

	returnedSubnets, err := GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.NotEmpty(t, returnedSubnets)
	returnedSubnet := returnedSubnets[0]
	require.Len(t, returnedSubnet.PrefixPools, 1)
	require.NotZero(t, returnedSubnet.PrefixPools[0].ID)
	require.NotZero(t, returnedSubnet.PrefixPools[0].Created)
	require.Equal(t, "2001:db8:1:1::/80", returnedSubnet.PrefixPools[0].Prefix)
	require.Equal(t, 96, returnedSubnet.PrefixPools[0].DelegatedLen)

	err = DeletePrefixPool(db, pool.ID)
	require.NoError(t, err)
	returnedSubnets, err = GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.NotEmpty(t, returnedSubnets)
	returnedSubnet = returnedSubnets[0]
	require.Empty(t, returnedSubnet.PrefixPools)
}
