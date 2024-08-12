package dbmodel

import (
	"testing"
	"time"

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

// Test that two prefix pools have unequal data if their prefixes differ.
func TestPrefixPoolHasEqualDataPrefix(t *testing.T) {
	// Arrange
	first := &PrefixPool{Prefix: "fe80::/64"}
	second := &PrefixPool{Prefix: "3001::/64"}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.False(t, equality)
}

// Test that two prefix pools have unequal data if their delegated lengths differ.
func TestPrefixPoolHasEqualDataDelegatedLength(t *testing.T) {
	// Arrange
	first := &PrefixPool{DelegatedLen: 64}
	second := &PrefixPool{DelegatedLen: 80}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.False(t, equality)
}

// Test that two prefix pools have unequal data if their excluded prefixes differ.
func TestPrefixPoolHasEqualDataExcludedPrefix(t *testing.T) {
	// Arrange
	first := &PrefixPool{ExcludedPrefix: "fe80::/80"}
	second := &PrefixPool{ExcludedPrefix: "3001::/80"}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.False(t, equality)
}

// Test that two prefix pools have equal data if their IDs differ.
func TestPrefixPoolHasEqualDataID(t *testing.T) {
	// Arrange
	first := &PrefixPool{ID: 1}
	second := &PrefixPool{ID: 2}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.True(t, equality)
}

// Test that two prefix pools have equal data if their create timestamps differ.
func TestPrefixPoolHasEqualDataCreateTimestamp(t *testing.T) {
	// Arrange
	first := &PrefixPool{CreatedAt: time.Date(1980, 1, 1, 12, 0o0, 0o0, 0, time.UTC)}
	second := &PrefixPool{CreatedAt: time.Date(2023, 1, 4, 16, 49, 0o0, 0, time.Local)}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.True(t, equality)
}

// Test that two prefix pools have equal data if their subnet IDs differ.
func TestPrefixPoolHasEqualDataSubnetID(t *testing.T) {
	// Arrange
	first := &PrefixPool{LocalSubnetID: 1}
	second := &PrefixPool{LocalSubnetID: 2}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.True(t, equality)
}

// Test that two prefix pools have equal data if their local subnet IDs differ.
func TestPrefixPoolHasEqualDataSubnet(t *testing.T) {
	// Arrange
	first := &PrefixPool{LocalSubnet: &LocalSubnet{LocalSubnetID: 14}}
	second := &PrefixPool{LocalSubnet: &LocalSubnet{LocalSubnetID: 15}}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.True(t, equality)
}

// Test that two prefix pools have equal data if they are the same.
func TestPrefixPoolHasEqualDataTheSame(t *testing.T) {
	// Arrange
	first := &PrefixPool{
		ID:             42,
		CreatedAt:      time.Time{},
		Prefix:         "fe80::/64",
		DelegatedLen:   80,
		LocalSubnetID:  24,
		LocalSubnet:    &LocalSubnet{},
		ExcludedPrefix: "fe80::/80",
	}

	second := &PrefixPool{
		ID:             42,
		CreatedAt:      time.Time{},
		Prefix:         "fe80::/64",
		DelegatedLen:   80,
		LocalSubnetID:  24,
		LocalSubnet:    &LocalSubnet{},
		ExcludedPrefix: "fe80::/80",
	}

	// Act
	equalityFirstSecond := first.HasEqualData(second)
	equalitySecondFirst := second.HasEqualData(first)

	// Assert
	require.True(t, equalityFirstSecond)
	require.True(t, equalitySecondFirst)
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

// Test that two address pools have equal data if their IDs differ.
func TestAddressPoolHasEqualDataID(t *testing.T) {
	// Arrange
	first := &AddressPool{ID: 1}
	second := &AddressPool{ID: 2}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.True(t, equality)
}

// Test that two address pools have equal data if their create timestamps differ.
func TestAddressPoolHasEqualDataCreateTimestamp(t *testing.T) {
	// Arrange
	first := &AddressPool{CreatedAt: time.Date(1980, 1, 1, 12, 0o0, 0o0, 0, time.UTC)}
	second := &AddressPool{CreatedAt: time.Date(2023, 1, 4, 16, 49, 0o0, 0, time.Local)}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.True(t, equality)
}

// Test that two address pools have unequal data if their lower bounds differ.
func TestAddressPoolHasEqualDataLowerBound(t *testing.T) {
	// Arrange
	first := &AddressPool{LowerBound: "10.0.0.1"}
	second := &AddressPool{LowerBound: "192.168.0.1"}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.False(t, equality)
}

// Test that two address pools have equal data if their upper bounds differ.
func TestAddressPoolHasEqualDataUpperBound(t *testing.T) {
	// Arrange
	first := &AddressPool{UpperBound: "10.0.0.42"}
	second := &AddressPool{UpperBound: "192.168.0.42"}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.False(t, equality)
}

// Test that two address pools have equal data if their subnet IDs differ.
func TestAddressPoolHasEqualDataSubnetID(t *testing.T) {
	// Arrange
	first := &AddressPool{LocalSubnetID: 1}
	second := &AddressPool{LocalSubnetID: 2}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.True(t, equality)
}

// Test that two address pools have equal data if their IDs differ.
func TestAddressPoolHasEqualDataSubnetIDs(t *testing.T) {
	// Arrange
	first := &AddressPool{LocalSubnet: &LocalSubnet{ID: 14}}
	second := &AddressPool{LocalSubnet: &LocalSubnet{ID: 15}}

	// Act
	equality := first.HasEqualData(second)

	// Assert
	require.True(t, equality)
}

// Test that two address pools have equal data of they are the same.
func TestAddressPoolHasEqualDataTheSame(t *testing.T) {
	// Arrange
	first := &AddressPool{
		ID:            42,
		CreatedAt:     time.Time{},
		LowerBound:    "10.0.0.1",
		UpperBound:    "10.0.0.10",
		LocalSubnetID: 24,
		LocalSubnet:   &LocalSubnet{},
	}

	second := &AddressPool{
		ID:            42,
		CreatedAt:     time.Time{},
		LowerBound:    "10.0.0.1",
		UpperBound:    "10.0.0.10",
		LocalSubnetID: 24,
		LocalSubnet:   &LocalSubnet{},
	}

	// Act
	equalityFirstSecond := first.HasEqualData(second)
	equalitySecondFirst := second.HasEqualData(first)

	// Assert
	require.True(t, equalityFirstSecond)
	require.True(t, equalitySecondFirst)
}
