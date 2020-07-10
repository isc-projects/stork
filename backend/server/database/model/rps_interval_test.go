package dbmodel

import (
	"testing"

	require "github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

func TestRpsIntervals(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Table should be empty.
	rpsIntervals, err := GetAllRpsIntervals(db)
	require.NoError(t, err)
	require.Len(t, rpsIntervals, 0)

	// Create one and store it in db
	interval1 := &RpsInterval{
		KeaDaemonID: 99,
		StartTime:   storkutil.UTCNow(),
		Duration:    int64(100),
		Responses:   int64(5),
	}

	err = AddRpsInterval(db, interval1)
	require.NoError(t, err)

	// We should have one row.
	rpsIntervals, err = GetAllRpsIntervals(db)
	require.NoError(t, err)
	require.Len(t, rpsIntervals, 1)

	interval2 := rpsIntervals[0]
	require.Equal(t, interval1.KeaDaemonID, interval2.KeaDaemonID)
	require.Equal(t, interval2.StartTime.Unix(), interval1.StartTime.Unix())
	require.Equal(t, interval1.Duration, interval2.Duration)
	require.Equal(t, interval1.Responses, interval2.Responses)
}
