package dbmodel

import (
	"testing"

	require "github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

func TestStats(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// initialize stats to 0
	InitializeStats(db)

	// get all stats and check some values
	stats, err := GetAllStats(db)
	require.NoError(t, err)
	require.Len(t, stats, 8)
	require.Contains(t, stats, "assigned-addresses")
	require.Zero(t, stats["assigned-addresses"])

	// modify one stats and store it in db
	stats["assigned-addresses"] = 10
	err = SetStats(db, stats)
	require.NoError(t, err)

	// get stats again and check if they have been modified
	stats, err = GetAllStats(db)
	require.NoError(t, err)
	require.Len(t, stats, 8)
	require.Contains(t, stats, "assigned-addresses")
	require.EqualValues(t, 10, stats["assigned-addresses"])
}
