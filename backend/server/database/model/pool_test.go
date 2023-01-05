package dbmodel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

	// IPv4 is not accepted.
	_, err = NewPrefixPool("192.0.2.0/24", 24, "")
	require.Error(t, err)

	// Non-empty excluded prefix
	pool, err = NewPrefixPool("2001:db8:1::/64", 96, "2001:db8:1:42::/80")
	require.NoError(t, err)
	require.EqualValues(t, "2001:db8:1:42::/80", pool.ExcludedPrefix)
}

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
	require.NotZero(t, returnedSubnet.AddressPools[0].CreatedAt)
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
	require.NotZero(t, returnedSubnet.PrefixPools[0].CreatedAt)
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

// Test that two prefix pools have not equal data if their prefixes differ.
func TestPrefixPoolEqualsDataPrefix(t *testing.T) {
	// Arrange
	first := &PrefixPool{Prefix: "fe80::/64"}
	second := &PrefixPool{Prefix: "3001::/64"}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.False(t, equity)
}

// Test that two prefix pools have not equal data if their delegated lengths differ.
func TestPrefixPoolEqualsDataDelegatedLength(t *testing.T) {
	// Arrange
	first := &PrefixPool{DelegatedLen: 64}
	second := &PrefixPool{DelegatedLen: 80}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.False(t, equity)
}

// Test that two prefix pools have equal data if their IDs differ.
func TestPrefixPoolEqualsDataID(t *testing.T) {
	// Arrange
	first := &PrefixPool{ID: 1}
	second := &PrefixPool{ID: 2}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.True(t, equity)
}

// Test that two prefix pools have equal data if their create timestamps differ.
func TestPrefixPoolEqualsDataCreateTimestamp(t *testing.T) {
	// Arrange
	first := &PrefixPool{CreatedAt: time.Date(1980, 1, 1, 12, 00, 00, 0, time.UTC)}
	second := &PrefixPool{CreatedAt: time.Date(2023, 1, 4, 16, 49, 00, 0, time.Local)}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.True(t, equity)
}

// Test that two prefix pools have equal data if their subnet IDs differ.
func TestPrefixPoolEqualsDataSubnetID(t *testing.T) {
	// Arrange
	first := &PrefixPool{SubnetID: 1}
	second := &PrefixPool{SubnetID: 2}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.True(t, equity)
}

// Test that two prefix pools have equal data if their subnets differ.
func TestPrefixPoolEqualsDataSubnet(t *testing.T) {
	// Arrange
	first := &PrefixPool{Subnet: &Subnet{Prefix: "fe80::/64"}}
	second := &PrefixPool{Subnet: &Subnet{Prefix: "10.0.0.0/8"}}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.True(t, equity)
}

// Test that two prefix pools have equal data if they are the same.
func TestPrefixPoolEqualsDataTheSame(t *testing.T) {
	// Arrange
	first := &PrefixPool{
		ID:           42,
		CreatedAt:    time.Time{},
		Prefix:       "fe80::/64",
		DelegatedLen: 80,
		SubnetID:     24,
		Subnet:       &Subnet{},
	}

	second := &PrefixPool{
		ID:           42,
		CreatedAt:    time.Time{},
		Prefix:       "fe80::/64",
		DelegatedLen: 80,
		SubnetID:     24,
		Subnet:       &Subnet{},
	}

	// Act
	equityFirstSecond := first.EqualsData(second)
	equitySecondFirst := second.EqualsData(first)

	// Assert
	require.True(t, equityFirstSecond)
	require.True(t, equitySecondFirst)
}

// Test that two address pools have equal data if their IDs differ.
func TestAddressPoolEqualsDataID(t *testing.T) {
	// Arrange
	first := &AddressPool{ID: 1}
	second := &AddressPool{ID: 2}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.True(t, equity)
}

// Test that two address pools have equal data if their create timestamp differ.
func TestAddressPoolEqualsDataCreateTimestamp(t *testing.T) {
	// Arrange
	first := &AddressPool{CreatedAt: time.Date(1980, 1, 1, 12, 00, 00, 0, time.UTC)}
	second := &AddressPool{CreatedAt: time.Date(2023, 1, 4, 16, 49, 00, 0, time.Local)}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.True(t, equity)
}

// Test that two address pools have not equal data if their lower bounds differ.
func TestAddressPoolEqualsDataLowerBound(t *testing.T) {
	// Arrange
	first := &AddressPool{LowerBound: "10.0.0.1"}
	second := &AddressPool{LowerBound: "192.168.0.1"}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.False(t, equity)
}

// Test that two address pools have equal data if their upper bounds differ.
func TestAddressPoolEqualsDataUpperBound(t *testing.T) {
	// Arrange
	first := &AddressPool{UpperBound: "10.0.0.42"}
	second := &AddressPool{UpperBound: "192.168.0.42"}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.False(t, equity)
}

// Test that two address pools have equal data if their subnet IDs differ.
func TestAddressPoolEqualsDataSubnetID(t *testing.T) {
	// Arrange
	first := &AddressPool{SubnetID: 1}
	second := &AddressPool{SubnetID: 2}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.True(t, equity)
}

// Test that two address pools have equal data if their subnets differ.
func TestAddressPoolEqualsDataSubnets(t *testing.T) {
	// Arrange
	first := &AddressPool{Subnet: &Subnet{Prefix: "fe80::/64"}}
	second := &AddressPool{Subnet: &Subnet{Prefix: "3001::/48"}}

	// Act
	equity := first.EqualsData(second)

	// Assert
	require.True(t, equity)
}

// Test that two address pools have equal data of they are the same.
func TestAddressPoolEqualsDataTheSame(t *testing.T) {
	// Arrange
	first := &AddressPool{
		ID:         42,
		CreatedAt:  time.Time{},
		LowerBound: "10.0.0.1",
		UpperBound: "10.0.0.10",
		SubnetID:   24,
		Subnet:     &Subnet{},
	}

	second := &AddressPool{
		ID:         42,
		CreatedAt:  time.Time{},
		LowerBound: "10.0.0.1",
		UpperBound: "10.0.0.10",
		SubnetID:   24,
		Subnet:     &Subnet{},
	}

	// Act
	equityFirstSecond := first.EqualsData(second)
	equitySecondFirst := second.EqualsData(first)

	// Assert
	require.True(t, equityFirstSecond)
	require.True(t, equitySecondFirst)
}
