package kea

import (
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

// Check creating and shutting down StatsPuller.
func TestStatsPullerBasic(t *testing.T) {
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

	sp, _ := NewStatsPuller(db, fa, true)
	require.NotEmpty(t, sp.RpsWorker)

	sp.Shutdown()
}

// Check if pulling stats works.
func TestStatsPullerPullStats(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("stat-lease4-get", daemons, nil)
		json := `[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "result-set": {
                                    "columns": [ "subnet-id", "total-addresses", "assigned-addresses", "declined-addresses" ],
                                    "rows": [
                                        [ 10, 256, 111, 0 ],
                                        [ 20, 4098, 2034, 4 ]
                                    ],
                                    "timestamp": "2018-05-04 15:03:37.000000"
                                }
                            }
                         }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])

		// DHCPv6
		daemons, _ = agentcomm.NewKeaDaemons("dhcp6")
		command, _ = agentcomm.NewKeaCommand("stat-lease6-get", daemons, nil)
		json = `[{
                           "result": 0,
                           "text": "Everything is fine",
                           "arguments": {
                               "result-set": {
                                   "columns": [ "subnet-id", "total-nas", "assigned-nas", "declined-nas", "total-pds", "assigned-pds" ],
                                   "rows": [
                                       [ 30, 4096, 2400, 3, 0, 0],
                                       [ 40, 0, 0, 0, 1048, 233 ],
                                       [ 50, 256, 60, 0, 1048, 15 ]
                                   ],
                                   "timestamp": "2018-05-04 15:03:37.000000"
                               }
                           }
                        }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[1])
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

	// prepare apps with subnets and local subnets
	v4Config := `
        {
            "Dhcp4": {
                "subnet4": [{"id": 10, "subnet": "192.0.2.0/24"},
                            {"id": 20, "subnet": "192.0.3.0/24"}]
            }
        }`
	v6Config := `
        {
            "Dhcp6": {
                "subnet6": [{"id": 30, "subnet": "2001:db8:1::/64"},
                            {"id": 40, "subnet": "2001:db8:2::/64"},
                            {"id": 50, "subnet": "2001:db8:3::/64"}]
            }
        }`
	app := createAppWithSubnets(t, db, 0, v4Config, v6Config)
	nets, snets, err := DetectNetworks(db, app)
	require.NoError(t, err)
	_, err = dbmodel.CommitNetworksIntoDB(db, nets, snets, app, 1)
	require.NoError(t, err)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "kea_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err = db.Insert(&setting)
	require.NoError(t, err)

	// prepare stats puller without RpsWorker
	sp, err := NewStatsPuller(db, fa, false)
	require.NoError(t, err)

	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// check collected stats
	subnets := []*dbmodel.LocalSubnet{}
	q := db.Model(&subnets)
	q = q.Column("local_subnet_id", "stats", "stats_collected_at")
	q = q.Where("local_subnet.app_id = ?", app.ID)
	err = q.Select()
	require.NoError(t, err)
	snCnt := 0
	for _, sn := range subnets {
		switch sn.LocalSubnetID {
		case 10:
			require.Equal(t, 111.0, sn.Stats["assigned-addresses"])
			require.Equal(t, 0.0, sn.Stats["declined-addresses"])
			require.Equal(t, 256.0, sn.Stats["total-addresses"])
			snCnt++
		case 20:
			require.Equal(t, 2034.0, sn.Stats["assigned-addresses"])
			require.Equal(t, 4.0, sn.Stats["declined-addresses"])
			require.Equal(t, 4098.0, sn.Stats["total-addresses"])
			snCnt++
		case 30:
			require.Equal(t, 2400.0, sn.Stats["assigned-nas"])
			require.Equal(t, 0.0, sn.Stats["assigned-pds"])
			require.Equal(t, 3.0, sn.Stats["declined-nas"])
			require.Equal(t, 0.0, sn.Stats["total-pds"])
			snCnt++
		case 40:
			require.Equal(t, 0.0, sn.Stats["assigned-nas"])
			require.Equal(t, 233.0, sn.Stats["assigned-pds"])
			require.Equal(t, 0.0, sn.Stats["declined-nas"])
			require.Equal(t, 0.0, sn.Stats["total-nas"])
			require.Equal(t, 1048.0, sn.Stats["total-pds"])
			snCnt++
		case 50:
			require.Equal(t, 60.0, sn.Stats["assigned-nas"])
			require.Equal(t, 15.0, sn.Stats["assigned-pds"])
			require.Equal(t, 0.0, sn.Stats["declined-nas"])
			require.Equal(t, 256.0, sn.Stats["total-nas"])
			require.Equal(t, 1048.0, sn.Stats["total-pds"])
			snCnt++
		}
	}
	require.Equal(t, 5, snCnt)
}

// Check if Kea response to stat-leaseX-get command is handled correctly when it is
// empty or when stats hooks library is not loaded.
func TestStatsPullerEmptyResponse(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("stat-lease4-get", daemons, nil)
		// simulate empty response
		json := `[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {}
                         }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])

		// DHCPv6
		daemons, _ = agentcomm.NewKeaDaemons("dhcp6")
		command, _ = agentcomm.NewKeaCommand("stat-lease6-get", daemons, nil)
		// simulate not loaded stat plugin in kea
		json = `[{
                           "result": 2,
                           "text": "'stat-lease6-get' command not supported."
                        }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[1])
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

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

	// prepare stats puller without RpsWorker
	sp, err := NewStatsPuller(db, fa, false)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.Error(t, err)
	require.Equal(t, 0, appsOkCnt)
}

// Check if pulling stats works when RPS is included.
// RpsWorker has a thorough set of unit tests so in this
// we verify only that it has entries in its internal
// Map of statistics fetched.  This is enough to demonstrate
// that it is operational.  We repeat the lease stats
// checks to make sure they have not been interfered with.
func TestStatsPullerPullStatsWithRps(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("stat-lease4-get", daemons, nil)
		json := `[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "result-set": {
                                    "columns": [ "subnet-id", "total-addresses", "assigned-addresses", "declined-addresses" ],
                                    "rows": [
                                        [ 10, 256, 111, 0 ],
                                        [ 20, 4098, 2034, 4 ]
                                    ],
                                    "timestamp": "2018-05-04 15:03:37.000000"
                                }
                            }
                         }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])

		// Command and response for DHCP4 RPS statistic pull
		command, _ = agentcomm.NewKeaCommand("statistic-get", daemons, nil)
		json = `[{
                    "result": 0,
                    "text": "Everything is fine",
                    "arguments": {
                        "pkt4-ack-sent": [ [ 44, "2019-07-30 10:13:00.000000" ] ]
                    }
                }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[1])

		// DHCPv6
		daemons, _ = agentcomm.NewKeaDaemons("dhcp6")
		command, _ = agentcomm.NewKeaCommand("stat-lease6-get", daemons, nil)
		json = `[{
                           "result": 0,
                           "text": "Everything is fine",
                           "arguments": {
                               "result-set": {
                                   "columns": [ "subnet-id", "total-nas", "assigned-nas", "declined-nas", "total-pds", "assigned-pds" ],
                                   "rows": [
                                       [ 30, 4096, 2400, 3, 0, 0],
                                       [ 40, 0, 0, 0, 1048, 233 ],
                                       [ 50, 256, 60, 0, 1048, 15 ]
                                   ],
                                   "timestamp": "2018-05-04 15:03:37.000000"
                               }
                           }
                        }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[2])

		// Command and response for DHCP6 RPS statistic pull
		command, _ = agentcomm.NewKeaCommand("statistic-get", daemons, nil)
		json = `[{
                    "result": 0,
                    "text": "Everything is fine",
                    "arguments": {
                        "pkt6-reply-sent": [ [ 66, "2019-07-30 10:13:00.000000" ] ]
                    }
                }]`

		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[3])
	}
	fa := storktest.NewFakeAgents(keaMock, nil)

	// prepare apps with subnets and local subnets
	v4Config := `
        {
            "Dhcp4": {
                "subnet4": [{"id": 10, "subnet": "192.0.2.0/24"},
                            {"id": 20, "subnet": "192.0.3.0/24"}]
            }
        }`
	v6Config := `
        {
            "Dhcp6": {
                "subnet6": [{"id": 30, "subnet": "2001:db8:1::/64"},
                            {"id": 40, "subnet": "2001:db8:2::/64"},
                            {"id": 50, "subnet": "2001:db8:3::/64"}]
            }
        }`
	app := createAppWithSubnets(t, db, 0, v4Config, v6Config)
	nets, snets, err := DetectNetworks(db, app)
	require.NoError(t, err)
	_, err = dbmodel.CommitNetworksIntoDB(db, nets, snets, app, 1)
	require.NoError(t, err)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "kea_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err = db.Insert(&setting)
	require.NoError(t, err)

	// prepare stats puller with RpsWorker
	sp, err := NewStatsPuller(db, fa, true)
	require.NoError(t, err)

	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// check collected stats
	subnets := []*dbmodel.LocalSubnet{}
	q := db.Model(&subnets)
	q = q.Column("local_subnet_id", "stats", "stats_collected_at")
	q = q.Where("local_subnet.app_id = ?", app.ID)
	err = q.Select()
	require.NoError(t, err)
	snCnt := 0
	for _, sn := range subnets {
		switch sn.LocalSubnetID {
		case 10:
			require.Equal(t, 111.0, sn.Stats["assigned-addresses"])
			require.Equal(t, 0.0, sn.Stats["declined-addresses"])
			require.Equal(t, 256.0, sn.Stats["total-addresses"])
			snCnt++
		case 20:
			require.Equal(t, 2034.0, sn.Stats["assigned-addresses"])
			require.Equal(t, 4.0, sn.Stats["declined-addresses"])
			require.Equal(t, 4098.0, sn.Stats["total-addresses"])
			snCnt++
		case 30:
			require.Equal(t, 2400.0, sn.Stats["assigned-nas"])
			require.Equal(t, 0.0, sn.Stats["assigned-pds"])
			require.Equal(t, 3.0, sn.Stats["declined-nas"])
			require.Equal(t, 0.0, sn.Stats["total-pds"])
			snCnt++
		case 40:
			require.Equal(t, 0.0, sn.Stats["assigned-nas"])
			require.Equal(t, 233.0, sn.Stats["assigned-pds"])
			require.Equal(t, 0.0, sn.Stats["declined-nas"])
			require.Equal(t, 0.0, sn.Stats["total-nas"])
			require.Equal(t, 1048.0, sn.Stats["total-pds"])
			snCnt++
		case 50:
			require.Equal(t, 60.0, sn.Stats["assigned-nas"])
			require.Equal(t, 15.0, sn.Stats["assigned-pds"])
			require.Equal(t, 0.0, sn.Stats["declined-nas"])
			require.Equal(t, 256.0, sn.Stats["total-nas"])
			require.Equal(t, 1048.0, sn.Stats["total-pds"])
			snCnt++
		}
	}
	require.Equal(t, 5, snCnt)

	// We should have two rows in RpsWorker.PreviousRps map one for each daemon
	require.Equal(t, 2, len(sp.RpsWorker.PreviousRps))

	// Entry for ID 1 should be dhcp4 daemon, it should have an RPS value of 44
	previous := sp.RpsWorker.PreviousRps[1]
	require.NotEqual(t, nil, previous)
	require.EqualValues(t, 44, previous.Value)

	// Entry for ID 2 should be dhcp6 daemon, it should have an RPS value of 66
	previous = sp.RpsWorker.PreviousRps[2]
	require.NotEqual(t, nil, previous)
	require.EqualValues(t, 66, previous.Value)
}
