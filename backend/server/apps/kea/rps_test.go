package kea

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

// Check creating and shutting down RpsPuller with PeriodicPuller.
func TestRpsPullerBasic(t *testing.T) {
	// prepare db
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "kea_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err := db.Insert(&setting)
	require.NoError(t, err)

	// prepare fake agents
	fa := storktest.NewFakeAgents(nil, nil)

	// Create the puller.
	sp, _ := NewRpsPuller(db, fa, true)

	// Should have a periodic puller.
	require.NotEmpty(t, sp.PeriodicPuller)
	sp.Shutdown()
}

// Check creating and shutting down RpsPuller without PeriodicPuller.
func TestRpsPullerBasicWithoutPuller(t *testing.T) {
	// prepare db
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "kea_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err := db.Insert(&setting)
	require.NoError(t, err)

	// prepare fake agents
	fa := storktest.NewFakeAgents(nil, nil)

	// Create the puller.
	sp, _ := NewRpsPuller(db, fa, false)

	// Should not have a periodic puller.
	require.Empty(t, sp.PeriodicPuller)

	// Shutdown should be harmless.
	sp.Shutdown()
}

// Convenience function that creates a machine with one Kea app and two daemons.
func addMachine(t *testing.T, db *dbops.PgDB, dhcp4Active bool, dhcp6Active bool) {
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
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "cool.example.org", "", 1234)
	a := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Active: dhcp4Active,
				Name:   "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
			{
				Active: dhcp6Active,
				Name:   "dhcp6",
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, a)
	require.NoError(t, err)
	require.NotEqual(t, 0, a.ID)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "kea_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err = db.Insert(&setting)
	require.NoError(t, err)
}

func checkDaemonRpsStats(t *testing.T, db *dbops.PgDB, keaDaemonID int64, interval1 int, interval2 int) {
	daemon := &dbmodel.KeaDHCPDaemon{}
	err := db.Model(daemon).
		Where("kea_daemon_id = ?", keaDaemonID).
		Select()

	require.NoError(t, err)
	require.Equal(t, interval1, daemon.Stats.RPS1)
	require.Equal(t, interval2, daemon.Stats.RPS2)
}

// Check if Kea response to statistic-get command is handled correctly
// when it is empty or malformed.
func TestRpsPullerEmptyOrInvalidResponses(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dhcp4Arguments := RpsGetDhcp4Arguments()
	dhcp6Arguments := RpsGetDhcp6Arguments()

	// JSON response to send per call number
	jsonResponses := []string{
		`[{ "result": 0, "text": "Samples missing", "arguments": {} }]`,
		`[{ "result": 0, "text": "No Arguments", }]`,
		`[{ "result": 1, "text": "Error response", }]`,
	}

	// Prepare fake agent which returns a response based on callNo
	keaMock := func(callNo int, cmdResponses []interface{}) {
		require.Less(t, callNo, len(jsonResponses))

		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("statistic-get", daemons, &dhcp4Arguments)
		agentcomm.UnmarshalKeaResponseList(command, jsonResponses[callNo], cmdResponses[0])

		// DHCPv6
		daemons, _ = agentcomm.NewKeaDaemons("dhcp6")
		command, _ = agentcomm.NewKeaCommand("statistic-get", daemons, &dhcp6Arguments)
		agentcomm.UnmarshalKeaResponseList(command, jsonResponses[callNo], cmdResponses[1])
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

	// Create a machine with one app and two kea daemons
	addMachine(t, db, true, true)

	// prepare stats puller
	sp, err := NewRpsPuller(db, fa, true)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	cmdIdx := 0
	for call := 0; call < len(jsonResponses); call++ {
		// invoke pulling stats
		appsOkCnt, err := sp.pullStats()
		require.NoError(t, err)
		require.Equal(t, 1, appsOkCnt)

		// Make sure we recorded 1 command per daemon per iteration
		require.Equal(t, (cmdIdx + 2), len(fa.RecordedCommands))
		require.Equal(t, &dhcp4Arguments, fa.RecordedCommands[cmdIdx].Arguments)
		cmdIdx++
		require.Equal(t, &dhcp6Arguments, fa.RecordedCommands[cmdIdx].Arguments)
		cmdIdx++
	}
}

// Verifies that no commands are sent or processed when neither
// dhcp4 or dhcp6 daeamons in an App are active.
func TestRpsPullerNoActiveDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Prepare a fake agent that will bomb if it gets called.
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// We should not get here at all.
		require.Equal(t, -1, callNo)
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

	// Create a machine with one app and two daemons: dhcp4 inactive, dhcp6 inactive
	addMachine(t, db, false, false)

	// prepare stats puller
	sp, err := NewRpsPuller(db, fa, true)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// We should have no recorded commands in the agent.
	require.Equal(t, 0, len(fa.RecordedCommands))

	// We should have no rows in PreviousRps map for dhcp4 daemon
	require.Equal(t, 0, len(sp.PreviousRps))
}

// Verifies that a command is only issued and response processed
// for the dhcp4 daemon, when the dhcp6 daemon is inactive.
func TestRpsPullerDhcp4Only(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dhcp4Arguments := RpsGetDhcp4Arguments()

	// Prepare fake agents. They will return an incremented statisic value with
	// each subsequent call.  We don't bother about the timestamps, they are not used.
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("statistic-get", daemons, &dhcp4Arguments)

		json := fmt.Sprintf(`[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "pkt4-ack-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                            }}]`, ((callNo + 1) * 5))

		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

	// Create a machine with one app and two daemons: dhcp4 active, dhcp6 inactive
	addMachine(t, db, true, false)

	// prepare stats puller
	sp, err := NewRpsPuller(db, fa, true)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// We should have one dhcp4 command recorded in the agent.
	require.Equal(t, 1, len(fa.RecordedCommands))
	require.Equal(t, &dhcp4Arguments, fa.RecordedCommands[0].Arguments)

	// We should have one row in PreviousRps map for dhcp4 daemon
	require.Equal(t, 1, len(sp.PreviousRps))

	// Row 1 should be dhcp4 daemon, it should have an RPS value of 5
	previous := sp.PreviousRps[1]
	require.NotEqual(t, nil, previous)
	require.EqualValues(t, 5, previous.Value)
}

// Verifies that a command is only issued and response processed
// for the dhcp6 daemon, when the dhcp4 daemon is inactive.
func TestRpsPullerDhcp6Only(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dhcp6Arguments := RpsGetDhcp6Arguments()

	// Prepare fake agents. They will return an incremented statisic value with
	// each subsequent call.  We don't bother about the timestamps, they are not used.
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv6
		daemons, _ := agentcomm.NewKeaDaemons("dhcp6")
		command, _ := agentcomm.NewKeaCommand("statistic-get", daemons, &dhcp6Arguments)

		json := fmt.Sprintf(`[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "pkt6-reply-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                            }}]`, ((callNo + 1) * 7))

		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

	// Create a machine with one app and two daemons: dhcp4 inactive, dhcp6 active
	addMachine(t, db, false, true)

	// prepare stats puller
	sp, err := NewRpsPuller(db, fa, true)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// We should have one dhcp6 command recorded in the agent.
	require.Equal(t, 1, len(fa.RecordedCommands))
	require.Equal(t, &dhcp6Arguments, fa.RecordedCommands[0].Arguments)

	// We should have one row in PreviousRps map for dhcp6 daemon
	require.Equal(t, 1, len(sp.PreviousRps))

	// Row 1 should be dhcp6 daemon, it should have an RPS value of 7
	previous := sp.PreviousRps[2]
	require.NotEqual(t, nil, previous)
	require.EqualValues(t, 7, previous.Value)
}

// Check if pulling and calculating stats for both servers works correctly.
// This test includes verification of RPS_INTERVAL table contents.
func TestRpsPullerPullRps(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dhcp4Arguments := RpsGetDhcp4Arguments()
	dhcp6Arguments := RpsGetDhcp6Arguments()

	// Prepare a fake agent to return an incremented statisic value with  each subsequent call.
	// We don't bother about the timestamp values, they are not used.
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("statistic-get", daemons, &dhcp4Arguments)

		json := fmt.Sprintf(`[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "pkt4-ack-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                            }}]`, ((callNo + 1) * 5))

		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])

		// DHCPv6
		daemons, _ = agentcomm.NewKeaDaemons("dhcp6")
		command, _ = agentcomm.NewKeaCommand("statistic-get", daemons, &dhcp6Arguments)

		json = fmt.Sprintf(`[{
                           "result": 0,
                           "text": "Everything is fine",
                           "arguments": {
                                "pkt6-reply-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                           }}]`, ((callNo + 1) * 7))

		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[1])
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

	// Create a machine with one app and two daemons: dhcp4 active, dhcp6 active
	addMachine(t, db, true, true)

	// prepare stats puller
	sp, err := NewRpsPuller(db, fa, true)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// We should have two commands, one for each daemon recorded in the agent.
	require.Equal(t, 2, len(fa.RecordedCommands))
	require.Equal(t, &dhcp4Arguments, fa.RecordedCommands[0].Arguments)
	require.Equal(t, &dhcp6Arguments, fa.RecordedCommands[1].Arguments)

	// We should have two rows in PreviousRps map, one for each daemon
	require.Equal(t, 2, len(sp.PreviousRps))

	// Row 1 should be dhcp4 daemon, it should have an RPS value of 5
	previous4 := sp.PreviousRps[1]
	require.NotEqual(t, nil, previous4)
	require.EqualValues(t, 5, previous4.Value)

	// Row 2 should be dhcp6 daemon, it should have an RPS value of 7
	previous6 := sp.PreviousRps[2]
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

	// Pull the stats again
	appsOkCnt, err = sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// We should have two more commands, one for each daemon recorded in the agent.
	require.Equal(t, 4, len(fa.RecordedCommands))
	require.Equal(t, &dhcp4Arguments, fa.RecordedCommands[2].Arguments)
	require.Equal(t, &dhcp6Arguments, fa.RecordedCommands[3].Arguments)

	// We should still only have two rows in PreviousRps map, one for each daemon
	require.Equal(t, 2, len(sp.PreviousRps))

	// Row 1 should be dhcp4 daemon, it should have an RPS value of 10
	current4 := sp.PreviousRps[1]
	require.NotEqual(t, nil, current4)
	require.Equal(t, int64(10), current4.Value)
	// The current recorded time should be two seconds later than the previous time.
	require.GreaterOrEqual(t, (current4.SampledAt.Unix() - previous4.SampledAt.Unix()), int64(2))

	// Row 2 should be dhcp6 daemon, it should have an RPS value of 14
	current6 := sp.PreviousRps[2]
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
// a verification of RpsPuller.updateDaemonRps() logic, which is common to both.
func TestRpsPullerValuePermutations(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dhcp4Arguments := RpsGetDhcp4Arguments()

	// Array of statistic values "returned" in statistic-get
	// between 200 and 35, this simulates kea-restart, rollover, or stat reset
	// between 35 and 35, this simulates no packets sent since last pull
	// between 50 and 0, this simulates kea-restart, rollover, or stat reset
	// valud of -1 is an invalid value which should be ignored, treated as 0
	statValues := []int64{100, 200, 35, 35, 50, 0, 10, -1, 17}

	// Array of expected values of RpsPreivous map row
	expectedPrevious := []int64{100, 200, 35, 35, 50, 0, 10, 0, 17}

	// Array of expected RpsInterval.Responses for each interval row added
	expectedResponses := []int64{100, 35, 0, 15, 0, 10, 0, 17}

	// Createa a fake agent that returns a statistic value from statValues[] using
	// callNo as an index.
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("statistic-get", daemons, &dhcp4Arguments)

		require.Less(t, callNo, len(statValues))

		json := fmt.Sprintf(`[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "pkt4-ack-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                            }}]`, statValues[callNo])

		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
	}

	fa := storktest.NewFakeAgents(keaMock, nil)

	// Create a machine with one app and two daemons: dhcp4 active, dhcp6 false
	addMachine(t, db, true, false)

	// prepare stats puller
	sp, err := NewRpsPuller(db, fa, true)
	require.NoError(t, err) // shutdown stats puller at the end
	defer sp.Shutdown()

	for pass := 0; pass < len(statValues); pass++ {
		// invoke pulling stats
		appsOkCnt, err := sp.pullStats()
		require.NoError(t, err)
		require.Equal(t, 1, appsOkCnt)

		// Verify the contents of PreviousRps map
		require.Equal(t, 1, len(sp.PreviousRps))
		previous := sp.PreviousRps[1]
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

// Calculte the RPS from an array of RpsIntervals.
func getExpectedRps(rpsIntervals []*dbmodel.RpsInterval, endIdx int) int {
	var responses int64
	var duration int64

	for idx := 0; idx < endIdx; idx++ {
		responses += rpsIntervals[idx].Responses
		duration += rpsIntervals[idx].Duration
	}

	if duration <= 0 {
		return 0
	}

	if responses < duration {
		return 1
	}

	return (int(responses / duration))
}
