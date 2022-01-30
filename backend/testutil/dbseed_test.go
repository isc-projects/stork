package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Test that multiple indexes can be combinet into one.
func TestIndexCombination(t *testing.T) {
	// Picture
	// Layer 0
	//      Columns
	//       0  1  2  3  4
	// R 0   0  1  2  3  4
	// O 1   5  6  7  8  9
	// W 2  10 11 12 13 14
	// S 3  15 16 17 18 19
	//
	// Layer 1
	//      Columns
	//       0  1  2  3  4
	// R 0  20 21 22 23 24
	// O 1  25 26 27 28 29
	// W 2  30 31 32 33 34
	// S 3  35 36 37 38 39

	// Arrange
	columns := 5
	rows := 4
	layers := 2

	column := 2
	row := 3
	layer := 1

	columnIdx := newIndex(column, columns)
	rowIdx := newIndex(row, rows)
	layerIdx := newIndex(layer, layers)

	// Act
	planeIdx := rowIdx.combine(columnIdx)
	cubicIdx := layerIdx.combine(planeIdx)

	// Assert
	require.EqualValues(t, 17, planeIdx.item)
	require.EqualValues(t, 20, planeIdx.total)

	require.EqualValues(t, 37, cubicIdx.item)
	require.EqualValues(t, 40, cubicIdx.total)
}

// Checks if the seed function creates expected number of entries.
func TestSeedExecute(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	config := &SeedConfig{
		Machines:                  2,
		Apps:                      3,
		SubnetsV4:                 4,
		SubnetsV6:                 5,
		Daemons:                   6,
		HostReservationsInPool:    7,
		HostReservationsOutOfPool: 8,
		// HostReservationsGlobalInPool:      9,
		// HostReservationsGlobalOutOfPool:   10,
		PrefixReservationsInPool:    11,
		PrefixReservationsOutOfPool: 12,
		// PrefixReservationsGlobalInPool:    13,
		// PrefixReservationsGlobalOutOfPool: 14,
	}

	// Act
	err := Seed(db, config)

	// Assert
	require.NoError(t, err)

	machines, _ := dbmodel.GetAllMachines(db, nil)
	require.Len(t, machines, 2)
	apps, _ := dbmodel.GetAllApps(db, true)
	require.Len(t, apps, 2*3)
	subnetsV4, _ := dbmodel.GetAllSubnets(db, 4)
	require.Len(t, subnetsV4, 2*3*4)
	subnetsV6, _ := dbmodel.GetAllSubnets(db, 6)
	require.Len(t, subnetsV6, 2*3*5)
	hosts, _ := dbmodel.GetAllHosts(db, 0)
	require.Len(t, hosts, 2*3*4+2*3*5)

	for _, app := range apps {
		require.Len(t, app.Daemons, 6)
	}

	totalHostReservations := 0
	totalPrefixReservations := 0

	for _, host := range hosts {
		for _, reservation := range host.IPReservations {
			if reservation.IsPrefixReservation() {
				totalPrefixReservations++
			} else {
				totalHostReservations++
			}
		}
	}

	require.EqualValues(t, 2*3*(4+5)*(7+8), totalHostReservations)
	require.EqualValues(t, 2*3*5*(11+12), totalPrefixReservations)

	outOfPoolAddressCounts, _ := dbmodel.CountOutOfPoolAddressReservations(db)
	totalOutOfPoolAddressReservations := uint64(0)
	for _, count := range outOfPoolAddressCounts {
		totalOutOfPoolAddressReservations += count
	}
	require.EqualValues(t, 2*3*(4+5)*8, totalOutOfPoolAddressReservations)

	outOfPoolPrefixCounts, _ := dbmodel.CountOutOfPoolPrefixReservations(db)
	totalOutOfPoolPrefixReservations := uint64(0)
	for _, count := range outOfPoolPrefixCounts {
		totalOutOfPoolPrefixReservations += count
	}
	require.EqualValues(t, 2*3*(5)*12, totalOutOfPoolPrefixReservations)
}

// Test that the seed function generates expected number of out-of-pool reservations.
func TestSeedGenerateHostReservations(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	config := &SeedConfig{
		Machines:                  1,
		Apps:                      1,
		SubnetsV4:                 1,
		SubnetsV6:                 1,
		Daemons:                   0,
		HostReservationsInPool:    10,
		HostReservationsOutOfPool: 20,
		// HostReservationsGlobalInPool:      0,
		// HostReservationsGlobalOutOfPool:   0,
		PrefixReservationsInPool:    0,
		PrefixReservationsOutOfPool: 0,
		// PrefixReservationsGlobalInPool:    0,
		// PrefixReservationsGlobalOutOfPool: 0,
	}

	// Act
	err := Seed(db, config)

	// Assert
	require.NoError(t, err)

	outOfPoolCounts, _ := dbmodel.CountOutOfPoolAddressReservations(db)
	totalOutOfPoolHostReservations := uint64(0)
	for _, count := range outOfPoolCounts {
		totalOutOfPoolHostReservations += count
	}
	require.EqualValues(t, 2*20, totalOutOfPoolHostReservations)
}

// Benchmark the queries with huge data.
func BenchmarkOutOfPoolReservations(b *testing.B) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	config := &SeedConfig{
		Machines:                  1,
		Apps:                      10,
		SubnetsV4:                 1000,
		SubnetsV6:                 1000,
		Daemons:                   0,
		HostReservationsInPool:    50,
		HostReservationsOutOfPool: 80,
		// HostReservationsGlobalInPool:      0,
		// HostReservationsGlobalOutOfPool:   0,
		PrefixReservationsInPool:    50,
		PrefixReservationsOutOfPool: 80,
		// PrefixReservationsGlobalInPool:    0,
		// PrefixReservationsGlobalOutOfPool: 0,
	}

	// Act
	err := Seed(db, config)
	counts, err2 := dbmodel.CountOutOfPoolAddressReservations(db)

	// Assert
	require.NoError(b, err)
	require.NoError(b, err2)
	require.NotNil(b, counts)
}
