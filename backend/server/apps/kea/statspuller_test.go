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

	sp, _ := NewStatsPuller(db, fa)
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
	err = dbmodel.CommitNetworksIntoDB(db, nets, snets, app, 1)
	require.NoError(t, err)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "kea_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err = db.Insert(&setting)
	require.NoError(t, err)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullLeaseStats()
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
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemonJSON{
				{
					Active: true,
					Name:   "dhcp4",
				},
				{
					Active: true,
					Name:   "dhcp6",
				},
			},
		},
	}
	err = dbmodel.AddApp(db, a)
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

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullLeaseStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)
}
