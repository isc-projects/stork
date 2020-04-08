package dbmodel

import (
	"testing"

	require "github.com/stretchr/testify/require"
	//log "github.com/sirupsen/logrus"

	dbtest "isc.org/stork/server/database/test"
)

func TestStats(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// initialize stats to 0
	InitializeStats(db)

	// get all stats and checks some values
	stats, err := GetAllStats(db)
	require.NoError(t, err)
	require.Len(t, stats, 8)
	require.Contains(t, stats, "assigned-addreses")
	require.EqualValues(t, 0, stats["assigned-addreses"])

	// modify one stats and store it in db
	stats["assigned-addreses"] = 10
	err = SetStats(db, stats)
	require.NoError(t, err)

	// get again stats and check if the modification is there
	stats, err = GetAllStats(db)
	require.NoError(t, err)
	require.Len(t, stats, 8)
	require.Contains(t, stats, "assigned-addreses")
	require.EqualValues(t, 10, stats["assigned-addreses"])
}
