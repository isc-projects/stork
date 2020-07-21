package dbmodel

import (
	"math"
	"testing"
	"time"

	"github.com/go-pg/pg/v9"
	require "github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Test the basics of inserting and fetching.
func TestRpsIntervalBasics(t *testing.T) {
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

	// Delete all rows at least a minute old (i.e. so none)
	startTime := interval2.StartTime.Add(time.Duration(-60) * time.Second)
	err = AgeOffRpsInterval(db, startTime)
	require.NoError(t, err)

	// We should still have one row.
	rpsIntervals, err = GetAllRpsIntervals(db)
	require.NoError(t, err)
	require.Len(t, rpsIntervals, 1)

	// Delete all rows at least interval2.StarTime old
	startTime = interval2.StartTime.Add(time.Duration(1) * time.Second)
	err = AgeOffRpsInterval(db, startTime)
	require.NoError(t, err)

	// We should still have no rows.
	rpsIntervals, err = GetAllRpsIntervals(db)
	require.NoError(t, err)
	require.Len(t, rpsIntervals, 0)
}

// Verifies operation of rps_interval.GetTotalRpsOverInterval()
// It populates the RpsInterval and then tests four invocations
// with varying time frames.
func TestRpsIntervalTotals(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Table should be empty.
	rpsIntervals, err := GetAllRpsIntervals(db)
	require.NoError(t, err)
	require.Len(t, rpsIntervals, 0)

	// Get a start time and round it off to seconds.
	// then back it up 60 seconds
	timeZero := storkutil.UTCNow().Round(time.Second)
	timeZero = timeZero.Add(time.Duration(-60) * time.Second)

	// Getting totals with no rows should return no rows.
	rpsTotals, err := GetTotalRpsOverInterval(db, timeZero, storkutil.UTCNow())
	require.NoError(t, err)
	require.Len(t, rpsTotals, 0)

	startTime := timeZero
	intervals := 5
	daemons := 3
	for interval := 1; interval <= intervals; interval++ {
		for daemon := 1; daemon <= daemons; daemon++ {
			// Create an interval and store it
			interval := &RpsInterval{
				KeaDaemonID: int64(daemon),
				StartTime:   startTime,
				Duration:    int64(5),
				Responses:   int64(interval) * int64(math.Pow10(daemon)),
			}

			err = AddRpsInterval(db, interval)
			require.NoError(t, err)
		}

		startTime = startTime.Add(time.Duration(5) * time.Second)
	}

	// Get totals that span the whole table
	startTime = timeZero
	endTime := storkutil.UTCNow()
	rpsTotals, err = GetTotalRpsOverInterval(db, startTime, endTime)
	require.NoError(t, err)
	require.Len(t, rpsTotals, 3)
	expDuration := (intervals * 5)

	// Verify the totals.
	for row := 0; row < daemons; row++ {
		interval := rpsTotals[row]
		daemon := row + 1
		require.EqualValues(t, daemon, interval.KeaDaemonID)
		require.EqualValues(t, expDuration, interval.Duration)

		var expResponses int64
		for interval := 1; interval <= intervals; interval++ {
			expResponses += int64(interval) * int64(math.Pow10(daemon))
		}

		require.EqualValues(t, expResponses, interval.Responses)

		// Now check the RPS values when pulled for a single daemon.
		checkIntervalPerDaemon(t, db, startTime, endTime,
			interval.KeaDaemonID, interval.Responses, interval.Duration)
	}

	// Fetch totals for a time frame containing only the first two intervals
	startTime = timeZero
	endTime = timeZero.Add(time.Duration(7) * time.Second)
	rpsTotals, err = GetTotalRpsOverInterval(db, startTime, endTime)
	require.NoError(t, err)
	require.Len(t, rpsTotals, 3)

	// Verify the totals.
	for row := 0; row < daemons; row++ {
		interval := rpsTotals[row]
		daemon := row + 1
		require.EqualValues(t, daemon, interval.KeaDaemonID)
		require.EqualValues(t, 10, interval.Duration)

		var expResponses int64
		for interval := 1; interval <= 2; interval++ {
			expResponses += int64(interval) * int64(math.Pow10(daemon))
		}

		require.EqualValues(t, expResponses, interval.Responses)

		// Now check the RPS values when pulled for a single daemon.
		checkIntervalPerDaemon(t, db, startTime, endTime,
			interval.KeaDaemonID, interval.Responses, interval.Duration)
	}

	// Fetch totals for a time frame containing only the middle three intervals
	startTime = timeZero.Add(time.Duration(5) * time.Second)
	endTime = startTime.Add(time.Duration(10) * time.Second)
	rpsTotals, err = GetTotalRpsOverInterval(db, startTime, endTime)
	require.NoError(t, err)
	require.Len(t, rpsTotals, 3)

	// Verify the totals.
	for row := 0; row < daemons; row++ {
		interval := rpsTotals[row]
		daemon := row + 1
		require.EqualValues(t, daemon, interval.KeaDaemonID)
		require.EqualValues(t, 15, interval.Duration)

		var expResponses int64
		for interval := 2; interval <= 4; interval++ {
			expResponses += int64(interval) * int64(math.Pow10(daemon))
		}

		require.EqualValues(t, expResponses, interval.Responses)

		// Now check the RPS values when pulled for a single daemon.
		checkIntervalPerDaemon(t, db, startTime, endTime,
			interval.KeaDaemonID, interval.Responses, interval.Duration)
	}

	// Fetch totals for a time frame containing only the last interval
	startTime = timeZero.Add(time.Duration(20) * time.Second)
	endTime = storkutil.UTCNow().Round(time.Second)
	rpsTotals, err = GetTotalRpsOverInterval(db, startTime, endTime)
	require.NoError(t, err)
	require.Len(t, rpsTotals, 3)

	// Verify the totals.
	for row := 0; row < daemons; row++ {
		interval := rpsTotals[row]
		daemon := row + 1
		require.EqualValues(t, daemon, interval.KeaDaemonID)
		require.EqualValues(t, 5, interval.Duration)
		expResponses := int64(5) * int64(math.Pow10(daemon))
		require.EqualValues(t, expResponses, interval.Responses)

		// Now check the RPS values when pulled for a single daemon.
		checkIntervalPerDaemon(t, db, startTime, endTime,
			interval.KeaDaemonID, interval.Responses, interval.Duration)
	}
}

func checkIntervalPerDaemon(t *testing.T, db *pg.DB, startTime time.Time, endTime time.Time, daemonID int64, expResponses int64, expDuration int64) {
	rpsTotals, err := GetTotalRpsOverIntervalForDaemon(db, startTime, endTime, daemonID)
	require.NoError(t, err)
	require.Len(t, rpsTotals, 1)
	interval := rpsTotals[0]
	require.EqualValues(t, daemonID, interval.KeaDaemonID)
	require.EqualValues(t, expDuration, interval.Duration)
	require.EqualValues(t, expResponses, interval.Responses)
}
