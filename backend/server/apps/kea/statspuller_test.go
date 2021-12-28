package kea

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
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
	_, err := db.Model(&setting).Insert()
	require.NoError(t, err)

	// prepare fake agents
	fa := agentcommtest.NewFakeAgents(nil, nil)

	sp, _ := NewStatsPuller(db, fa)
	require.NotEmpty(t, sp.RpsWorker)

	sp.Shutdown()
}

// Check if Kea response to stat-leaseX-get command is handled correctly when it is
// empty or when stats hooks library is not loaded.  The RPS responses are valid,
// they have their own unit tests in rps_test.go.
func TestStatsPullerEmptyResponse(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := keactrl.NewDaemons("dhcp4")
		command, _ := keactrl.NewCommand("stat-lease4-get", daemons, nil)
		// simulate empty response
		json := `[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {}
                         }]`
		keactrl.UnmarshalResponseList(command, []byte(json), cmdResponses[0])

		// DHCPv4 RSP response
		json = `[{ "result": 0, "text": "Everything is fine",
                    "arguments": {
                                "pkt4-ack-sent": [ [ 0, "2019-07-30 10:13:00.000000" ] ]
                }}]`

		rpsCmd := []*keactrl.Command{}
		_ = RpsAddCmd4(&rpsCmd, daemons)
		keactrl.UnmarshalResponseList(rpsCmd[0], []byte(json), cmdResponses[1])

		// DHCPv6
		daemons, _ = keactrl.NewDaemons("dhcp6")
		command, _ = keactrl.NewCommand("stat-lease6-get", daemons, nil)
		// simulate not loaded stat plugin in kea
		json = `[{
                           "result": 2,
                           "text": "'stat-lease6-get' command not supported."
                        }]`
		keactrl.UnmarshalResponseList(command, []byte(json), cmdResponses[2])

		// DHCPv6 RSP response
		json = `[{ "result": 0, "text": "Everything is fine",
                    "arguments": {
                                "pkt6-reply-sent": [ [ 0, "2019-07-30 10:13:00.000000" ] ]
                }}]`

		rpsCmd = []*keactrl.Command{}
		_ = RpsAddCmd6(&rpsCmd, daemons)
		keactrl.UnmarshalResponseList(rpsCmd[0], []byte(json), cmdResponses[3])
	}
	fa := agentcommtest.NewFakeAgents(keaMock, nil)

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
				Active: true,
				Name:   "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
			{
				Active: true,
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
	_, err = db.Model(&setting).Insert()
	require.NoError(t, err)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	err = sp.pullStats()
	require.Error(t, err)
}

// Check if pulling stats works when RPS is included.
// RpsWorker has a thorough set of unit tests so in this
// we verify only that it has entries in its internal
// Map of statistics fetched.  This is enough to demonstrate
// that it is operational.
func checkStatsPullerPullStats(t *testing.T, statsFormat string) {
	// 1.6 format
	totalAddrs := "total-addreses"
	assignedAddrs := "assigned-addreses"
	declinedAddrs := "declined-addreses"
	if statsFormat == "1.8" {
		totalAddrs = "total-addresses"
		assignedAddrs = "assigned-addresses"
		declinedAddrs = "declined-addresses"
	}

	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := keactrl.NewDaemons("dhcp4")
		command, _ := keactrl.NewCommand("stat-lease4-get", daemons, nil)
		json := fmt.Sprintf(`[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "result-set": {
                                    "columns": [ "subnet-id", "%s", "%s", "%s" ],
                                    "rows": [
                                        [ 10, 256, 111, 0 ],
                                        [ 20, 4098, 2034, 4 ]
                                    ],
                                    "timestamp": "2018-05-04 15:03:37.000000"
                                }
                            }
                         }]`, totalAddrs, assignedAddrs, declinedAddrs)
		keactrl.UnmarshalResponseList(command, []byte(json), cmdResponses[0])

		// Command and response for DHCP4 RPS statistic pull
		rpsCmd := []*keactrl.Command{}
		_ = RpsAddCmd4(&rpsCmd, daemons)
		json = `[{
                    "result": 0,
                    "text": "Everything is fine",
                    "arguments": {
                        "pkt4-ack-sent": [ [ 44, "2019-07-30 10:13:00.000000" ] ]
                    }
                }]`
		keactrl.UnmarshalResponseList(rpsCmd[0], []byte(json), cmdResponses[1])

		// DHCPv6
		daemons, _ = keactrl.NewDaemons("dhcp6")
		command, _ = keactrl.NewCommand("stat-lease6-get", daemons, nil)
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
		keactrl.UnmarshalResponseList(command, []byte(json), cmdResponses[2])

		// Command and response for DHCP6 RPS statistic pull
		rpsCmd = []*keactrl.Command{}
		_ = RpsAddCmd4(&rpsCmd, daemons)
		json = `[{
                    "result": 0,
                    "text": "Everything is fine",
                    "arguments": {
                        "pkt6-reply-sent": [ [ 66, "2019-07-30 10:13:00.000000" ] ]
                    }
                }]`
		keactrl.UnmarshalResponseList(rpsCmd[0], []byte(json), cmdResponses[3])
	}
	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	// prepare apps with subnets and local subnets
	// the host reservations shouldn't affect the statistics
	v4Config := `{
					"Dhcp4": {
						"subnet4": [
							{
								"id": 10,
								"subnet": "192.0.2.0/24"
							},
							{
								"id": 20,
								"subnet": "192.0.3.0/24",
								"reservations": [
									{
										"hw-address": "00:00:00:00:00:22",
										"ip-address": "192.0.2.22"
									},
									{
										"hw-address": "00:00:00:00:00:23",
										"ip-address": "192.0.2.23"
									}
								]
							}
						]
					}
				}`
	v6Config := `{
					"Dhcp6": {
						"subnet6": [
							{
								"id": 30,
								"subnet": "2001:db8:1::/64"
							},
							{
								"id": 40,
								"subnet": "2001:db8:2::/64"
							},
							{
								"id": 50,
								"subnet": "2001:db8:3::/64",
								"reservations": [
									{
										"hw-address": "00:00:00:00:01:22",
										"ip-address": "2001:db8:3::21"
									},
									{
										"hw-address": "00:00:00:00:01:23",
										"ip-address": "2001:db8:3::23"
									}
								]
							}
						]
					}
				}`
	app := createAppWithSubnets(t, db, 0, v4Config, v6Config)

	for i := range app.Daemons {
		nets, snets, err := detectDaemonNetworks(db, app.Daemons[i])
		require.NoError(t, err)
		_, err = dbmodel.CommitNetworksIntoDB(db, nets, snets, app.Daemons[i], 1)
		require.NoError(t, err)
	}

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "kea_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	_, err := db.Model(&setting).Insert()
	require.NoError(t, err)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa)
	require.NoError(t, err)

	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	err = sp.pullStats()
	require.NoError(t, err)

	// check collected stats
	subnets := []*dbmodel.LocalSubnet{}
	q := db.Model(&subnets)
	q = q.Column("local_subnet_id", "stats", "stats_collected_at")
	q = q.Join("INNER JOIN daemon ON local_subnet.daemon_id = daemon.id")
	q = q.Where("daemon.app_id = ?", app.ID)
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

func TestStatsPullerPullStatsKea16Format(t *testing.T) {
	checkStatsPullerPullStats(t, "1.6")
}

func TestStatsPullerPullStatsKea18Format(t *testing.T) {
	checkStatsPullerPullStats(t, "1.8")
}
