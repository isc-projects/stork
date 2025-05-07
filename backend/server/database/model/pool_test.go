package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the pool instance can be created from two addresses
// or from a prefix.
func TestNewAddressPoolFromRange(t *testing.T) {
	// IPv4 case.
	pool, err := NewAddressPoolFromRange("192.0.2.10 - 192.0.2.55")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.10", pool.LowerBound)
	require.Equal(t, "192.0.2.55", pool.UpperBound)

	// IPv6 case with some odd spacing.
	pool, err = NewAddressPoolFromRange("2001:db8:1:1::1000 -2001:db8:1:2::EEEE")
	require.NoError(t, err)
	require.Equal(t, "2001:db8:1:1::1000", pool.LowerBound)
	require.Equal(t, "2001:db8:1:2::eeee", pool.UpperBound)

	// Check that the pool can be specified as prefix.
	pool, err = NewAddressPoolFromRange("3000:1::/32")
	require.NoError(t, err)
	require.Equal(t, "3000:1::", pool.LowerBound)
	require.Equal(t, "3000:1:ffff:ffff:ffff:ffff:ffff:ffff", pool.UpperBound)

	// Two hyphens and 3 addresses is wrong.
	_, err = NewAddressPoolFromRange("192.0.2.0-192.0.2.100-192.0.3.100")
	require.Error(t, err)

	// No upper bound.
	_, err = NewAddressPoolFromRange("192.0.2.0- ")
	require.Error(t, err)

	// Mix of IPv4 and IPv6 is wrong.
	_, err = NewAddressPoolFromRange("192.0.2.0-2001:db8:1::100")
	require.Error(t, err)
}

// Test that the prefix pool instance can be created from the prefix
// and the delegated length.
func TestNewPrefixPool(t *testing.T) {
	pool, err := NewPrefixPool("2001:db8:1::/64", 96, "")
	require.NoError(t, err)
	require.NotNil(t, pool)

	require.Equal(t, "2001:db8:1::/64", pool.Prefix)
	require.EqualValues(t, 96, pool.DelegatedLen)
	require.Empty(t, pool.ExcludedPrefix)
	require.Zero(t, pool.LocalSubnetID)

	// IPv4 is not accepted.
	_, err = NewPrefixPool("192.0.2.0/24", 24, "")
	require.Error(t, err)

	// Non-empty excluded prefix
	pool, err = NewPrefixPool("2001:db8:1::/64", 96, "2001:db8:1:42::/80")
	require.NoError(t, err)
	require.EqualValues(t, "2001:db8:1:42::/80", pool.ExcludedPrefix)
	require.Zero(t, pool.LocalSubnetID)
}

func TestAddDeleteAddressPool(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestSubnetApps(t, db)

	subnet := Subnet{
		Prefix: "192.0.2.0/24",
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: apps[0].Daemons[0].ID,
			},
		},
	}
	err := AddSubnet(db, &subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	err = AddLocalSubnets(db, &subnet)
	require.NoError(t, err)

	pool := AddressPool{
		LowerBound: "192.0.2.10",
		UpperBound: "192.0.2.20",
		LocalSubnet: &LocalSubnet{
			ID: subnet.LocalSubnets[0].ID,
		},
	}
	err = AddAddressPool(db, &pool)
	require.NoError(t, err)

	returnedSubnets, err := GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.NotEmpty(t, returnedSubnets)
	returnedSubnet := returnedSubnets[0]
	require.Len(t, returnedSubnet.LocalSubnets, 1)
	require.Len(t, returnedSubnet.LocalSubnets[0].AddressPools, 1)
	require.NotZero(t, returnedSubnet.LocalSubnets[0].AddressPools[0].ID)
	require.NotZero(t, returnedSubnet.LocalSubnets[0].AddressPools[0].CreatedAt)
	require.Equal(t, "192.0.2.10", returnedSubnet.LocalSubnets[0].AddressPools[0].LowerBound)
	require.Equal(t, "192.0.2.20", returnedSubnet.LocalSubnets[0].AddressPools[0].UpperBound)

	err = DeleteAddressPool(db, pool.ID)
	require.NoError(t, err)
	returnedSubnets, err = GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.NotEmpty(t, returnedSubnets)
	returnedSubnet = returnedSubnets[0]
	require.Len(t, returnedSubnet.LocalSubnets, 1)
	require.Empty(t, returnedSubnet.LocalSubnets[0].AddressPools)
}

func TestAddDeletePrefixPool(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestSubnetApps(t, db)

	subnet := Subnet{
		Prefix: "2001:db8:1::/64",
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: apps[0].Daemons[0].ID,
			},
		},
	}
	err := AddSubnet(db, &subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	err = AddLocalSubnets(db, &subnet)
	require.NoError(t, err)

	pool := PrefixPool{
		Prefix:       "2001:db8:1:1::/80",
		DelegatedLen: 96,
		LocalSubnet: &LocalSubnet{
			ID: subnet.LocalSubnets[0].ID,
		},
	}
	err = AddPrefixPool(db, &pool)
	require.NoError(t, err)

	returnedSubnets, err := GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.NotEmpty(t, returnedSubnets)
	returnedSubnet := returnedSubnets[0]
	require.Len(t, returnedSubnet.LocalSubnets, 1)
	require.Len(t, returnedSubnet.LocalSubnets[0].PrefixPools, 1)
	require.NotZero(t, returnedSubnet.LocalSubnets[0].PrefixPools[0].ID)
	require.NotZero(t, returnedSubnet.LocalSubnets[0].PrefixPools[0].CreatedAt)
	require.Equal(t, "2001:db8:1:1::/80", returnedSubnet.LocalSubnets[0].PrefixPools[0].Prefix)
	require.Equal(t, 96, returnedSubnet.LocalSubnets[0].PrefixPools[0].DelegatedLen)

	err = DeletePrefixPool(db, pool.ID)
	require.NoError(t, err)
	returnedSubnets, err = GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.NotEmpty(t, returnedSubnets)
	returnedSubnet = returnedSubnets[0]
	require.Len(t, returnedSubnet.LocalSubnets, 1)
	require.Empty(t, returnedSubnet.LocalSubnets[0].PrefixPools)
}

// Test the implementation of the dhcpmodel.PrefixPoolAccessor interface
// (GetModel() function).
func TestPrefixPoolGetModel(t *testing.T) {
	pool := PrefixPool{
		Prefix:         "3001::/80",
		DelegatedLen:   88,
		ExcludedPrefix: "3001::/96",
	}
	model := pool.GetModel()
	require.Equal(t, "3001::/80", model.Prefix)
	require.EqualValues(t, 88, model.DelegatedLen)
	require.Equal(t, "3001::/96", model.ExcludedPrefix)
}

// Test retrieving Kea parameters from a prefix pool.
func TestPrefixPoolGetKeaParameters(t *testing.T) {
	pool := PrefixPool{
		Prefix:        "3001::/80",
		KeaParameters: &keaconfig.PoolParameters{},
	}
	require.Equal(t, pool.GetKeaParameters(), pool.KeaParameters)
}

// Test retrieving Kea parameters from a prefix pool when the parameters are nil.
func TestPrefixPoolGetNilKeaParameters(t *testing.T) {
	pool := PrefixPool{
		Prefix: "3001::/80",
	}
	require.Nil(t, pool.GetKeaParameters())
}

// Test the implementation of the dhcpmodel.PrefixPoolAccessor interface
// (GetDHCPOptions() function).
func TestPrefixPoolGetDHCPOptions(t *testing.T) {
	pool := PrefixPool{
		Prefix: "3001::/80",
		DHCPOptionSet: []DHCPOption{
			{
				Code:  7,
				Space: dhcpmodel.DHCPv4OptionSpace,
			},
		},
	}
	options := pool.GetDHCPOptions()
	require.Len(t, options, 1)
	require.EqualValues(t, 7, options[0].GetCode())
	require.Equal(t, dhcpmodel.DHCPv4OptionSpace, options[0].GetSpace())
}

// Test the implementation of the dhcpmodel.AddressPoolAccessor interface
// (GetLowerBound() and GetUpperBound() functions).
func TestAddressPoolGetBounds(t *testing.T) {
	pool := AddressPool{
		LowerBound: "192.0.2.1",
		UpperBound: "192.0.2.10",
	}
	require.Equal(t, "192.0.2.1", pool.GetLowerBound())
	require.Equal(t, "192.0.2.10", pool.GetUpperBound())
}

// Test the implementation of the keaconfig.AddressPool interface
// (GetKeaParameters() function).
func TestAddressPoolGetKeaParameters(t *testing.T) {
	clientClass := "foo"
	pool := AddressPool{
		LowerBound: "2001:db8:1::cafe",
		UpperBound: "2001:db8:1::ffff",
		KeaParameters: &keaconfig.PoolParameters{
			ClientClassParameters: keaconfig.ClientClassParameters{
				ClientClass: &clientClass,
			},
		},
	}
	params := pool.GetKeaParameters()
	require.NotNil(t, params)
	require.Equal(t, "foo", *params.ClientClass)
}

// Test the implementation of the dhcpmodel.AddressPoolAccessor interface
// (GetDHCPOptions() function).
func TestAddressPoolGetDHCPOptions(t *testing.T) {
	pool := AddressPool{
		LowerBound: "2001:db8:1::cafe",
		UpperBound: "2001:db8:1::ffff",
		DHCPOptionSet: NewDHCPOptionSet([]DHCPOption{
			{
				Code:  10,
				Space: dhcpmodel.DHCPv6OptionSpace,
			},
		}, keaconfig.NewHasher()),
	}
	options := pool.GetDHCPOptions()
	require.Len(t, options, 1)
	require.EqualValues(t, 10, options[0].GetCode())
	require.Equal(t, dhcpmodel.DHCPv6OptionSpace, options[0].GetSpace())
}

// Test the IsIPv6 function recognizes IPv6 address pools.
func TestAddressPoolIsIPv6(t *testing.T) {
	t.Run("IPv4", func(t *testing.T) {
		v4 := AddressPool{
			LowerBound: "192.0.2.1",
			UpperBound: "192.0.2.10",
		}
		require.False(t, v4.IsIPv6())
	})

	t.Run("IPv6", func(t *testing.T) {
		v6 := AddressPool{
			LowerBound: "2001:db8:1::cafe",
			UpperBound: "2001:db8:1::ffff",
		}
		require.True(t, v6.IsIPv6())
	})
}

// Test that the address pool statistics are updated correctly.
func TestAddressPoolUpdateStats(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestSubnetApps(t, db)
	subnet := Subnet{
		Prefix: "192.0.2.0/24",
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: apps[0].Daemons[0].ID,
			},
		},
	}
	err := AddSubnet(db, &subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	err = AddLocalSubnets(db, &subnet)
	require.NoError(t, err)

	pool := AddressPool{
		LowerBound: "192.0.2.10",
		UpperBound: "192.0.2.20",
		LocalSubnet: &LocalSubnet{
			ID: subnet.LocalSubnets[0].ID,
		},
	}
	err = AddAddressPool(db, &pool)
	require.NoError(t, err)

	// Act
	err = pool.UpdateStats(db, SubnetStats{
		"foo":                uint64(42),
		"total-addresses":    uint64(100),
		"assigned-addresses": uint64(33),
	})

	// Assert
	require.NoError(t, err)
	subnetDB, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	poolDB := subnetDB.LocalSubnets[0].AddressPools[0]

	require.Len(t, poolDB.Stats, 3)
	require.Equal(t, uint64(42), poolDB.Stats["foo"])
	require.NotZero(t, poolDB.StatsCollectedAt)
	require.Equal(t, Utilization(0.33), poolDB.Utilization)
}

// Test that the prefix pool statistics are updated correctly.
func TestPrefixPoolUpdateStats(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestSubnetApps(t, db)
	subnet := Subnet{
		Prefix: "fe80::/64",
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: apps[0].Daemons[0].ID,
			},
		},
	}
	err := AddSubnet(db, &subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	err = AddLocalSubnets(db, &subnet)
	require.NoError(t, err)

	pdPool := PrefixPool{
		Prefix:       "fe80::/80",
		DelegatedLen: 96,
		LocalSubnet: &LocalSubnet{
			ID: subnet.LocalSubnets[0].ID,
		},
	}
	err = AddPrefixPool(db, &pdPool)
	require.NoError(t, err)

	addressPool := AddressPool{
		LowerBound: "fe80::1",
		UpperBound: "fe80::ffff",
		LocalSubnet: &LocalSubnet{
			ID: subnet.LocalSubnets[0].ID,
		},
	}
	err = AddAddressPool(db, &addressPool)
	require.NoError(t, err)

	// Act
	errPD := pdPool.UpdateStats(db, SubnetStats{
		"foo":          uint64(42),
		"total-pds":    uint64(100),
		"assigned-pds": uint64(33),
	})

	errAddress := addressPool.UpdateStats(db, SubnetStats{
		"foo":          uint64(42),
		"total-nas":    uint64(200),
		"assigned-nas": uint64(50),
	})

	// Assert
	require.NoError(t, errPD)
	require.NoError(t, errAddress)
	subnetDB, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	pdPoolDB := subnetDB.LocalSubnets[0].PrefixPools[0]
	addressPoolDB := subnetDB.LocalSubnets[0].AddressPools[0]

	require.Len(t, pdPoolDB.Stats, 3)
	require.Equal(t, uint64(42), pdPoolDB.Stats["foo"])
	require.NotZero(t, pdPoolDB.StatsCollectedAt)
	require.Equal(t, Utilization(0.33), pdPoolDB.Utilization)

	require.Len(t, addressPoolDB.Stats, 3)
	require.Equal(t, uint64(42), addressPoolDB.Stats["foo"])
	require.NotZero(t, addressPoolDB.StatsCollectedAt)
	require.Equal(t, Utilization(0.25), addressPoolDB.Utilization)
}
