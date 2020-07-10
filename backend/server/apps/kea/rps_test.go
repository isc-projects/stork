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

// Check creating and shutting down RpsPuller.
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

	sp, _ := NewRpsPuller(db, fa)
	sp.Shutdown()
}

// Convenience function that creates a machine with one Kea app and two daemons.
func addMachine(t *testing.T, db *dbops.PgDB) {
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
				Active:    true,
				Name:      "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{},
			},
			{
				Active:    true,
				Name:      "dhcp6",
				KeaDaemon: &dbmodel.KeaDaemon{},
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

// Check if Kea response to statistic-get command is handled correctly when it is empty
func TestRpsPullerEmptyResponse(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("statistic-get", daemons, &map[string]interface{}{"name": "pkt4-ack-sent"})
		// simulate empty response
		json := `[{
                            "result": 0,
                            "text": "Empty arguments",
                            "arguments": {}
                         }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])

		// DHCPv6
		daemons, _ = agentcomm.NewKeaDaemons("dhcp6")
		command, _ = agentcomm.NewKeaCommand("statistic-get", daemons, &map[string]interface{}{"name": "pkt6-reply-sent"})
		// missing arguments
		json = `[{
                            "result": 0,
                            "text": "No Arguments",
                        }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[1])
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

	// Create a machine with one app and two kea daemons
	addMachine(t, db)

	// prepare stats puller
	sp, err := NewRpsPuller(db, fa)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)
}

// Check if pulling stats works.
func TestRpsPullerPullRps(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Prepare fake agents. They will return an incremented statisic value with
	// each subsequent call.  We don't bother about the timestamps, they are not used.
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("statistic-get", daemons, &map[string]interface{}{"name": "pkt4-ack-sent"})

		json := fmt.Sprintf(`[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "pkt4-ack-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                            }}]`, ((callNo + 1) * 5))

		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])

		// DHCPv6
		daemons, _ = agentcomm.NewKeaDaemons("dhcp6")
		command, _ = agentcomm.NewKeaCommand("statistic-get", daemons, &map[string]interface{}{"name": "pkt6-reply-sent"})

		json = fmt.Sprintf(`[{
                           "result": 0,
                           "text": "Everything is fine",
                           "arguments": {
                                "pkt6-reply-sent": [ [ %d, "2019-07-30 10:13:00.000000" ] ]
                           }}]`, ((callNo + 1) * 7))

		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[1])
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

	// Create a machine with one app and two kea daemons
	addMachine(t, db)

	// prepare stats puller
	sp, err := NewRpsPuller(db, fa)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

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

	// sleep two seconds so we will have a later recorded time
	time.Sleep(2 * time.Second)

	// Pull the stats again
	appsOkCnt, err = sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

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
	require.EqualValues(t, 2, (current6.SampledAt.Unix() - previous6.SampledAt.Unix()))

	// Now let's verify the intervals.
	rpsIntervals, err := dbmodel.GetAllRpsIntervals(db)
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
}
