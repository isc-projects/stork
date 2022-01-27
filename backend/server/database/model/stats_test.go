package dbmodel

import (
	"math/big"
	"testing"

	require "github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

func TestStats(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// initialize stats to 0
	err := InitializeStats(db)
	require.NoError(t, err)

	// get all stats and check some values
	stats, err := GetAllStats(db)
	require.NoError(t, err)
	require.Len(t, stats, 8)
	require.Contains(t, stats, "assigned-addresses")
	require.EqualValues(t, big.NewInt(0), stats["assigned-addresses"])

	// modify one stats and store it in db
	stats["assigned-addresses"] = big.NewInt(10)
	err = SetStats(db, stats)
	require.NoError(t, err)

	// get stats again and check if they have been modified
	stats, err = GetAllStats(db)
	require.NoError(t, err)
	require.Len(t, stats, 8)
	require.Contains(t, stats, "assigned-addresses")
	require.EqualValues(t, big.NewInt(10), stats["assigned-addresses"])

	// add very large stat value - 40 digits
	largeValue, ok := big.NewInt(0).SetString("123456789012345678901234567890123456789012345678901234567890", 10)
	require.True(t, ok)
	stats["assigned-addresses"] = largeValue
	err = SetStats(db, stats)
	require.NoError(t, err)
	stats, err = GetAllStats(db)
	require.NoError(t, err)
	require.Len(t, stats, 8)
	require.Contains(t, stats, "assigned-addresses")
	require.EqualValues(t, largeValue, stats["assigned-addresses"])
}

// The statistic value cannot be nil.
func TestStatisticNilValueIsNotError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	stats := map[string]*big.Int{
		"foo": nil,
	}

	// Act
	errSet := SetStats(db, stats)
	stats, errGet := GetAllStats(db)

	// Assert
	require.NoError(t, errSet)
	require.NoError(t, errGet)

	require.Nil(t, stats["foo"])
}
