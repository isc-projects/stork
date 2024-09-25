package kea

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Check if Kea response to statistic-get command is handled correctly
// when it is empty or malformed.
func TestRpsWorkerEmptyOrInvalidResponses(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// JSON response to send per call number
	jsonResponses := []string{
		`[{ "result": 0, "text": "Samples missing", "arguments": {} }]`,
		`[{ "result": 0, "text": "No Arguments", }]`,
		`[{ "result": 1, "text": "Error response", }]`,
	}

	// Create a machine with one app and two kea daemons
	dhcp4Daemon, dhcp6Daemon := rpsTestAddMachine(t, db, true, true)

	// prepare stats puller
	rps, err := NewRpsWorker(db)
	require.NoError(t, err)

	for call := 0; call < len(jsonResponses); call++ {
		err := rpsTestInvokeResponse4Handler(rps, dhcp4Daemon, jsonResponses[call])
		require.Error(t, err)

		err = rpsTestInvokeResponse6Handler(rps, dhcp6Daemon, jsonResponses[call])
		require.Error(t, err)
	}
}

// Check if pulling and calculating stats for both servers works correctly.
// This test includes verification of RPS_INTERVAL table contents.
func TestRpsWorkerPullRps(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	makeJSON4 := func(callNo int) string {
		return (fmt.Sprintf(`[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "pkt4-ack-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                            }}]`, (callNo * 5)))
	}

	makeJSON6 := func(callNo int) string {
		return (fmt.Sprintf(`[{
                           "result": 0,
                           "text": "Everything is fine",
                           "arguments": {
                                "pkt6-reply-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                           }}]`, (callNo * 7)))
	}

	// Create a machine with one app and two kea daemons
	dhcp4Daemon, dhcp6Daemon := rpsTestAddMachine(t, db, true, true)

	// prepare stats puller
	rps, err := NewRpsWorker(db)
	require.NoError(t, err)

	// Process a round of statistics for both daemons (equates to a single pull cycle)
	callNo := 1
	err = rpsTestInvokeResponse4Handler(rps, dhcp4Daemon, makeJSON4(callNo))
	require.NoError(t, err)

	err = rpsTestInvokeResponse6Handler(rps, dhcp6Daemon, makeJSON6(callNo))
	require.NoError(t, err)

	// We should have two rows in PreviousRps map, one for each daemon
	require.Equal(t, 2, len(rps.PreviousRps))

	// Row 1 should be dhcp4 daemon, it should have an RPS value of 5
	previous4 := rps.PreviousRps[1]
	require.NotEqual(t, nil, previous4)
	require.EqualValues(t, 5, previous4.Value)

	// Row 2 should be dhcp6 daemon, it should have an RPS value of 7
	previous6 := rps.PreviousRps[2]
	require.NotEqual(t, nil, previous6)
	require.EqualValues(t, 7, previous6.Value)

	// Now let's verify that there are no intervals yet.
	rpsIntervals, err := dbmodel.GetAllRpsIntervals(db)
	require.NoError(t, err)
	require.Len(t, rpsIntervals, 0)

	// Verify daemon RPS stat values are all 0.
	checkDaemonRpsStats(t, db, 1, 0, 0)
	checkDaemonRpsStats(t, db, 2, 0, 0)

	// sleep two seconds so we will have a later recorded time
	time.Sleep(2 * time.Second)

	// Do another "pull"
	callNo++
	err = rpsTestInvokeResponse4Handler(rps, dhcp4Daemon, makeJSON4(callNo))
	require.NoError(t, err)

	err = rpsTestInvokeResponse6Handler(rps, dhcp6Daemon, makeJSON6(callNo))
	require.NoError(t, err)

	// We should still only have two rows in PreviousRps map, one for each daemon
	require.Equal(t, 2, len(rps.PreviousRps))

	// Row 1 should be dhcp4 daemon, it should have an RPS value of 10
	current4 := rps.PreviousRps[1]
	require.NotEqual(t, nil, current4)
	require.Equal(t, int64(10), current4.Value)
	// The current recorded time should be two seconds later than the previous time.
	require.GreaterOrEqual(t, (current4.SampledAt.Unix() - previous4.SampledAt.Unix()), int64(2))

	// Row 2 should be dhcp6 daemon, it should have an RPS value of 14
	current6 := rps.PreviousRps[2]
	require.NotEqual(t, nil, current6)
	require.EqualValues(t, 14, current6.Value)
	// The current recorded time should be two seconds later than the previous time.
	require.GreaterOrEqual(t, (current6.SampledAt.Unix() - previous6.SampledAt.Unix()), int64(2))

	// Now let's verify the intervals.
	rpsIntervals, err = dbmodel.GetAllRpsIntervals(db)
	require.NoError(t, err)
	require.Len(t, rpsIntervals, 2)

	// First row should be for dhcp4
	interval := rpsIntervals[0]
	require.EqualValues(t, 1, interval.KeaDaemonID)
	require.Equal(t, previous4.SampledAt.Unix(), interval.StartTime.Unix())
	require.GreaterOrEqual(t, (current4.SampledAt.Unix() - previous4.SampledAt.Unix()), interval.Duration)
	require.EqualValues(t, 5, interval.Responses)

	// Second row should be for dhcp6
	interval = rpsIntervals[1]
	require.EqualValues(t, 2, interval.KeaDaemonID)
	require.Equal(t, previous6.SampledAt.Unix(), interval.StartTime.Unix())
	require.GreaterOrEqual(t, (current6.SampledAt.Unix() - previous6.SampledAt.Unix()), interval.Duration)
	require.EqualValues(t, 7, interval.Responses)

	// Verify daemon RPS stat values are as expected.
	checkDaemonRpsStats(t, db, 1, 2, 2)
	checkDaemonRpsStats(t, db, 2, 3, 3)
}

// Verifies that getting stat values that are less than or equal to the previous
// value are handled correctly.  This is only tested for dhcp4, as this is primarily
// a verification of RpsWorker.updateDaemonRps() logic, which is common to both.
func TestRpsWorkerValuePermutations(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Array of statistic values "returned" in statistic-get
	// between 200 and 35, this simulates kea-restart, rollover, or stat reset
	// between 35 and 35, this simulates no packets sent since last pull
	// between 50 and 0, this simulates kea-restart, rollover, or stat reset
	// value of -1 is an invalid value which should be ignored, treated as 0
	statValues := []int64{100, 200, 35, 35, 50, 0, 10, -1, 17}

	// Array of expected values of RpsPrevious map row
	expectedPrevious := []int64{100, 200, 35, 35, 50, 0, 10, 0, 17}

	// Array of expected RpsInterval.Responses for each interval row added
	expectedResponses := []int64{100, 35, 0, 15, 0, 10, 0, 17}

	makeJSON4 := func(value int64) string {
		resp := fmt.Sprintf(`[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "pkt4-ack-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                            }}]`, value)
		return (resp)
	}

	// Create a machine with one app and two daemons: dhcp4 active, dhcp6 false
	dhcp4Daemon, _ := rpsTestAddMachine(t, db, true, false)

	// prepare stats puller
	rps, err := NewRpsWorker(db)
	require.NoError(t, err)

	for pass := 0; pass < len(statValues); pass++ {
		// Process the next command response
		err = rpsTestInvokeResponse4Handler(rps, dhcp4Daemon, makeJSON4(statValues[pass]))
		require.NoError(t, err)

		// Verify the contents of PreviousRps map
		require.Equal(t, 1, len(rps.PreviousRps))
		previous := rps.PreviousRps[1]
		require.NotEqual(t, nil, previous)
		require.EqualValues(t, expectedPrevious[pass], previous.Value)

		// Verify the number of interval rows.
		rpsIntervals, err := dbmodel.GetAllRpsIntervals(db)
		require.NoError(t, err)
		require.Len(t, rpsIntervals, pass)

		// After the first pass, verify the content of the newest interval row
		// and the kea_dhcp_daemon table.
		if pass > 0 {
			require.Equal(t, expectedResponses[pass-1], rpsIntervals[pass-1].Responses)

			// Verify daemon RPS stats are as expected.  We calculate them from
			// the recorded intervals to avoid sporadic timing differences in duration
			// which can cause the test to fail.
			expectedRps := getExpectedRps(rpsIntervals, pass)
			checkDaemonRpsStats(t, db, 1, expectedRps, expectedRps)
		}

		// Sleep for 1 second to ensure durations are at least that long.
		time.Sleep(1 * time.Second)
	}
}

// Convenience function that creates a machine with one Kea app and two daemons.
func rpsTestAddMachine(t *testing.T, db *dbops.PgDB, dhcp4Active bool, dhcp6Active bool) (*dbmodel.Daemon, *dbmodel.Daemon) {
	// add one machine with one kea app
	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "cool.example.org", "", 1234, true)
	a := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Active: dhcp4Active,
				Name:   dhcp4,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
			{
				Active: dhcp6Active,
				Name:   dhcp6,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, a)
	require.NoError(t, err)
	require.NotEqual(t, 0, a.ID)

	return a.Daemons[0], a.Daemons[1]
}

// Verifies RPS values for both intervals for a given daemon.
func checkDaemonRpsStats(t *testing.T, db *dbops.PgDB, keaDaemonID int64, interval1 float32, interval2 float32) {
	daemon := &dbmodel.KeaDHCPDaemon{}
	err := db.Model(daemon).
		Where("kea_daemon_id = ?", keaDaemonID).
		Select()

	require.NoError(t, err)

	// Since we use Sleep() in our tests, it is possible that the actual duration between
	// samples is a bit longer than expected. The corresponding RPS values can therefore
	// be rounded down. Let's add a margin of 1 to these checks. Without it, the test
	// results were unstable.
	require.Condition(t, func() bool {
		return daemon.Stats.RPS1 <= interval1 || daemon.Stats.RPS1 >= interval1-1
	}, "RPS1: %d, interval1: %d", daemon.Stats.RPS1, interval1)
	require.Condition(t, func() bool {
		return daemon.Stats.RPS2 <= interval2 || daemon.Stats.RPS2 >= interval2-1
	}, "RPS2: %d, interval2: %d", daemon.Stats.RPS2, interval2)
}

// Calculate the RPS from an array of RpsIntervals.
func getExpectedRps(rpsIntervals []*dbmodel.RpsInterval, endIdx int) float32 {
	var responses int64
	var duration int64

	for idx := 0; idx < endIdx; idx++ {
		responses += rpsIntervals[idx].Responses
		duration += rpsIntervals[idx].Duration
	}

	if duration <= 0 {
		return 0
	}

	return float32(responses) / float32(duration)
}

// Marshall a given json response to a DHCP4 command and pass that into Response4Handler.
func rpsTestInvokeResponse4Handler(rps *RpsWorker, daemon *dbmodel.Daemon, jsonResponse string) error {
	cmds := []*keactrl.Command{}
	responses := []interface{}{}

	responses = append(responses, RpsAddCmd4(&cmds, []string{dhcp4}))
	keactrl.UnmarshalResponseList(cmds[0], []byte(jsonResponse), responses[0])

	err := rps.Response4Handler(daemon, responses[0])
	return err
}

// Marshall a given json response to a DHCP6 command and pass that into Response6Handler.
func rpsTestInvokeResponse6Handler(rps *RpsWorker, daemon *dbmodel.Daemon, jsonResponse string) error {
	cmds := []*keactrl.Command{}
	responses := []interface{}{}

	responses = append(responses, RpsAddCmd6(&cmds, []string{dhcp6}))
	keactrl.UnmarshalResponseList(cmds[0], []byte(jsonResponse), responses[0])

	err := rps.Response6Handler(daemon, responses[0])
	return err
}
