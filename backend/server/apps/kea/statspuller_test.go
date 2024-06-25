package kea

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Prepares the Kea mock. It accepts list of serialized JSON responses in order:
// 1. DHCPv4
// 2. DHCPv4 RSP
// 3. DHCPv6
// 4. DHCPv6 RSP.
func createKeaMock(jsonFactory func(callNo int) (jsons []string)) func(callNo int, cmdResponses []interface{}) {
	return func(callNo int, cmdResponses []interface{}) {
		jsons := jsonFactory(callNo)
		// DHCPv4
		daemons := []keactrl.DaemonName{keactrl.DHCPv4}
		command := keactrl.NewCommandBase(keactrl.StatLease4Get, daemons...)
		keactrl.UnmarshalResponseList(command, []byte(jsons[0]), cmdResponses[0])

		// DHCPv4 RSP response
		rpsCmd := []*keactrl.Command{}
		_ = RpsAddCmd4(&rpsCmd, daemons)
		keactrl.UnmarshalResponseList(rpsCmd[0], []byte(jsons[1]), cmdResponses[1])

		if len(cmdResponses) < 4 {
			return
		}

		// DHCPv6
		daemons = []keactrl.DaemonName{keactrl.DHCPv6}
		command = keactrl.NewCommandBase(keactrl.StatLease6Get, daemons...)
		keactrl.UnmarshalResponseList(command, []byte(jsons[2]), cmdResponses[2])

		// DHCPv6 RSP response
		rpsCmd = []*keactrl.Command{}
		_ = RpsAddCmd6(&rpsCmd, daemons)
		keactrl.UnmarshalResponseList(rpsCmd[0], []byte(jsons[3]), cmdResponses[3])
	}
}

// Converts the list of int64s to list of the JSON-serializable big integers.
func convertInt64SeriesToBigInts(series [][]int64) (result [][]storkutil.BigIntJSON) {
	result = make([][]storkutil.BigIntJSON, len(series))
	for i, row := range series {
		result[i] = make([]storkutil.BigIntJSON, len(row))
		for j, item := range row {
			result[i][j] = storkutil.NewBigIntJSONFromInt64(item)
		}
	}
	return
}

// Prepares the Kea mock with the statistic values compatible with the
// configurations produced by the createDhcpConfigs function.
// It assigns different, predictable values for each application. Supports the
// applications with a single DHCPv4 daemon.
// Accepts a parameter that indicates if the old names of the statistics should
// be used (missing doubled "s" in "addresses" word).
func createStandardKeaMock(oldStatsFormat bool) func(callNo int, cmdResponses []interface{}) {
	statLeaseGetResponseDHCPv4Columns := []string{"subnet-id", "total-addresses", "assigned-addresses", "declined-addresses"}
	if oldStatsFormat {
		statLeaseGetResponseDHCPv4Columns = []string{"subnet-id", "total-addreses", "assigned-addreses", "declined-addreses"}
	}

	return createKeaMock(func(callNo int) []string {
		shift := int64(callNo * 100)
		totalShift := shift * 2
		data := []interface{}{
			[]StatLeaseGetResponse{
				{
					ResponseHeader: keactrl.ResponseHeader{
						Result: 0,
						Text:   "Everything is fine",
					},
					Arguments: &StatLeaseGetArgs{
						ResultSet: ResultSetInStatLeaseGet{
							Columns: statLeaseGetResponseDHCPv4Columns,
							Rows: convertInt64SeriesToBigInts([][]int64{
								{10, 256 + totalShift, 111 + shift, 0 + shift},
								{20, 4098 + totalShift, 2034 + shift, 4 + shift},
							}),
						},
						Timestamp: "2018-05-04 15:03:37.000000",
					},
				},
			},
			[]StatGetResponse4{
				{
					ResponseHeader: keactrl.ResponseHeader{
						Result: 0,
						Text:   "Everything is fine",
					},
					Arguments: &ResponseArguments4{
						Samples: []interface{}{
							[]interface{}{44, "2019-07-30 10:13:00.000000"},
						},
					},
				},
			},
			[]StatLeaseGetResponse{
				{
					ResponseHeader: keactrl.ResponseHeader{
						Result: 0,
						Text:   "Everything is fine",
					},
					Arguments: &StatLeaseGetArgs{
						ResultSet: ResultSetInStatLeaseGet{
							Columns: []string{"subnet-id", "total-nas", "assigned-nas", "declined-nas", "total-pds", "assigned-pds"},
							Rows: convertInt64SeriesToBigInts([][]int64{
								{30, 4096 + totalShift, 2400 + shift, 3 + shift, 0 + totalShift, 0 + shift},
								{40, 0 + totalShift, 0 + shift, 0 + shift, 1048 + totalShift, 233 + shift},
								{50, 256 + totalShift, 60 + shift, 0 + shift, 1048 + totalShift, 15 + shift},
								{60, -1, 9223372036854775807, 0, -2, -3},
							}),
						},
						Timestamp: "2018-05-04 15:03:37.000000",
					},
				},
			},
			[]StatGetResponse6{
				{
					ResponseHeader: keactrl.ResponseHeader{
						Result: 0,
						Text:   "Everything is fine",
					},
					Arguments: &ResponseArguments6{
						Samples: []interface{}{
							[]interface{}{66, "2019-07-30 10:13:00.000000"},
						},
					},
				},
			},
		}

		var jsons []string

		for _, item := range data {
			j, _ := json.Marshal(item)
			jsons = append(jsons, string(j))
		}

		return jsons
	})
}

// Creates test DHCPv4 and DHCPv6 configurations with some subnets and
// reservations.
func createDhcpConfigs() (string, string) {
	dhcp4 := `{
		"Dhcp4": {
			"hooks-libraries": [
				{
					"library": "/usr/lib/kea/libdhcp_stat_cmds.so"
				}
			],
			"reservations": [
				{
					"hw-address": "01:bb:cc:dd:ee:ff",
					"ip-address": "192.12.0.1"
				},
				{
					"hw-address": "02:bb:cc:dd:ee:ff",
					"ip-address": "192.12.0.2"
				}
			],
			"subnet4": [
				{
					"id": 10,
					"subnet": "192.0.2.0/24"
				},
				{
					"id": 20,
					"subnet": "192.0.3.0/24",
					// 1 in-pool, 2 out-of-pool host reservations
					"pools": [
						{
							"pool": "192.0.3.1 - 192.0.3.10"
						}
					],
					"reservations": [
						{
							"hw-address": "00:00:00:00:00:21",
							"ip-address": "192.0.3.2"
						},
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
	dhcp6 := `{
		"Dhcp6": {
			"hooks-libraries": [
				{
					"library": "/usr/lib/kea/libdhcp_stat_cmds.so"
				}
			],
			"reservations": [
				{
					"hw-address": "03:bb:cc:dd:ee:ff",
					"ip-address": "80:80::1"
				},
				{
					"hw-address": "04:bb:cc:dd:ee:ff",
					"ip-address": "80:90::/64"
				}
			],
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
					"pools": [
						{
							"pool": "2001:db8:3::100-2001:db8:3::ffff"
						}
					],
					"pd-pools": [
						{
							"prefix": "2001:db8:3:8000::",
							"prefix-len": 64,
							"delegated-len": 80
						}
					],
					// 2 out-of-pool, 1 in-pool host reservations
					// 1 out-of-pool, 1 in-pool prefix reservations
					"reservations": [
						{
							"hw-address": "00:00:00:00:01:21",
							"ip-address": "2001:db8:3::101",
							"prefixes": [ "2001:db8:3:8000::/80" ]
						},
						{
							"hw-address": "00:00:00:00:01:22",
							"ip-address": "2001:db8:3::21",
							"prefixes": [ "2001:db8:2:abcd::/80" ]
						},
						{
							"hw-address": "00:00:00:00:01:23",
							"ip-address": "2001:db8:3::23"
						}
					]
				},
				{
					"id": 60,
					"subnet": "2001:db8:4::/64"
				},
				{
					"id": 70,
					"subnet": "2001:db8:5::/64"
				}
			]
		}
	}`

	return dhcp4, dhcp6
}

// Checks if the puller correctly collected the local subnets statistics. It
// assumes that the DHCP configurations produced by the createDhcpConfigs
// function and statistics responses from the createStandardKeaMock function
// were used.
func verifyStandardLocalSubnetsStatistics(t *testing.T, db *pg.DB) {
	// Check collected stats in the local subnets. There is no meaning if they
	// are from the HA daemons.
	localSubnets := []*dbmodel.LocalSubnet{}
	q := db.Model(&localSubnets)
	q = q.Relation("Daemon")
	err := q.Select()
	require.NoError(t, err)
	snCnt := 0
	for _, sn := range localSubnets {
		shift := (sn.Daemon.AppID - 1) * 100
		totalShift := shift * 2
		switch sn.LocalSubnetID {
		case 10:
			require.Equal(t, uint64(111+shift), sn.Stats["assigned-addresses"])
			require.Equal(t, uint64(0+shift), sn.Stats["declined-addresses"])
			require.Equal(t, uint64(256+totalShift), sn.Stats["total-addresses"])
			snCnt++
		case 20:
			require.Equal(t, uint64(2034+shift), sn.Stats["assigned-addresses"])
			require.Equal(t, uint64(4+shift), sn.Stats["declined-addresses"])
			require.Equal(t, uint64(4098+totalShift), sn.Stats["total-addresses"])
			snCnt++
		case 30:
			require.Equal(t, uint64(2400+shift), sn.Stats["assigned-nas"])
			require.Equal(t, uint64(0+shift), sn.Stats["assigned-pds"])
			require.Equal(t, uint64(3+shift), sn.Stats["declined-nas"])
			require.Equal(t, uint64(4096+totalShift), sn.Stats["total-nas"])
			require.Equal(t, uint64(0+totalShift), sn.Stats["total-pds"])
			snCnt++
		case 40:
			require.Equal(t, uint64(0+shift), sn.Stats["assigned-nas"])
			require.Equal(t, uint64(233+shift), sn.Stats["assigned-pds"])
			require.Equal(t, uint64(0+shift), sn.Stats["declined-nas"])
			require.Equal(t, uint64(0+totalShift), sn.Stats["total-nas"])
			require.Equal(t, uint64(1048+totalShift), sn.Stats["total-pds"])
			snCnt++
		case 50:
			require.Equal(t, uint64(60+shift), sn.Stats["assigned-nas"])
			require.Equal(t, uint64(15+shift), sn.Stats["assigned-pds"])
			require.Equal(t, uint64(0+shift), sn.Stats["declined-nas"])
			require.Equal(t, uint64(256+totalShift), sn.Stats["total-nas"])
			require.Equal(t, uint64(1048+totalShift), sn.Stats["total-pds"])
			snCnt++
		case 60:
			require.Equal(t, uint64(math.MaxUint64), sn.Stats["total-nas"])
			require.Equal(t, uint64(math.MaxInt64), sn.Stats["assigned-nas"])
			require.Equal(t, uint64(0), sn.Stats["declined-nas"])
			require.Equal(t, uint64(math.MaxUint64)-1, sn.Stats["total-pds"])
			require.Equal(t, uint64(math.MaxUint64)-2, sn.Stats["assigned-pds"])
			snCnt++
		case 70:
			require.Nil(t, sn.Stats)
		}
	}

	daemons, _ := dbmodel.GetKeaDHCPDaemons(db)
	v4Daemons := 0
	v6Daemons := 0

	for _, daemon := range daemons {
		switch daemon.Name {
		case dbmodel.DaemonNameDHCPv4:
			v4Daemons++
		case dbmodel.DaemonNameDHCPv6:
			v6Daemons++
		default:
		}
	}

	require.Equal(t, 2*v4Daemons+4*v6Daemons, snCnt)
}

// Check creating and shutting down StatsPuller.
func TestStatsPullerBasic(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	fa := agentcommtest.NewFakeAgents(nil, nil)

	// Act
	sp, err := NewStatsPuller(db, fa)
	defer sp.Shutdown()

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, sp.RpsWorker)
}

// Check if Kea response to stat-leaseX-get command is handled correctly when it is
// empty or when stats hooks library is not loaded.  The RPS responses are valid,
// they have their own unit tests in rps_test.go.
func TestStatsPullerEmptyResponse(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	_ = createAppWithSubnets(t, db, 0, "", "")

	// prepare fake agents
	keaMock := createKeaMock(func(callNo int) (jsons []string) {
		return []string{
			// simulate empty response
			`[{
				"result": 0,
				"text": "Everything is fine",
				"arguments": {}
			}]`,
			`[{
				"result": 0, "text": "Everything is fine",
				"arguments": {
					"pkt4-ack-sent": [ [ 0, "2019-07-30 10:13:00.000000" ] ]
				}
			}]`,
			// simulate not loaded stat plugin in kea
			`[{
				"result": 2,
				"text": "'stat-lease6-get' command not supported."
			}]`,
			`[{
				"result": 0, "text": "Everything is fine",
				"arguments": {
					"pkt6-reply-sent": [ [ 0, "2019-07-30 10:13:00.000000" ] ]
				}
			}]`,
		}
	})

	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	// prepare stats puller
	sp, _ := NewStatsPuller(db, fa)
	defer sp.Shutdown()

	// Act
	// invoke pulling stats
	err := sp.pullStats()

	// Assert
	require.Error(t, err)
}

// Check if pulling stats works when RPS is included.
// RpsWorker has a thorough set of unit tests so in this
// we verify only that it has entries in its internal
// Map of statistics fetched.  This is enough to demonstrate
// that it is operational.
func checkStatsPullerPullStats(t *testing.T, statsFormat string) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	_ = dbmodel.InitializeStats(db)

	// prepare apps with subnets and local subnets
	v4Config, v6Config := createDhcpConfigs()
	app := createAppWithSubnets(t, db, 0, v4Config, v6Config)

	keaMock := createStandardKeaMock(statsFormat == "1.6")

	fa := agentcommtest.NewFakeAgents(keaMock, nil)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	for i := range app.Daemons {
		sharedNetworks, subnets, err := detectDaemonNetworks(db, app.Daemons[i], lookup)
		require.NoError(t, err)
		_, err = dbmodel.CommitNetworksIntoDB(db, sharedNetworks, subnets)
		require.NoError(t, err)
		hosts, err := detectGlobalHostsFromConfig(db, app.Daemons[i], lookup)
		require.NoError(t, err)
		err = dbmodel.CommitGlobalHostsIntoDB(db, hosts)
		require.NoError(t, err)
	}

	// prepare stats puller
	sp, _ := NewStatsPuller(db, fa)
	defer sp.Shutdown()

	// Act
	// invoke pulling stats
	err := sp.pullStats()

	// Assert
	require.NoError(t, err)

	// check collected stats
	verifyStandardLocalSubnetsStatistics(t, db)

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

	// Check out-of-pool addresses/NAs/PDs utilization
	subnets, _ := dbmodel.GetAllSubnets(db, 0)

	for _, sn := range subnets {
		switch sn.LocalSubnets[0].LocalSubnetID {
		case 10:
			require.InDelta(t, 111.0/256.0, float64(sn.AddrUtilization)/1000.0, 0.001)
			require.Zero(t, sn.PdUtilization)
		case 20:
			require.InDelta(t, 2034.0/(4098.0+2), float64(sn.AddrUtilization)/1000.0, 0.001)
			require.Zero(t, sn.PdUtilization)
		case 30:
			require.InDelta(t, 2400.0/4096.0, float64(sn.AddrUtilization)/1000.0, 0.001)
			require.Zero(t, sn.PdUtilization)
		case 40:
			require.Zero(t, sn.AddrUtilization)
			require.InDelta(t, 233.0/1048.0, float64(sn.PdUtilization)/1000.0, 0.001)
		case 50:
			require.InDelta(t, 60.0/(256.0+2), float64(sn.AddrUtilization)/1000.0, 0.001)
			require.InDelta(t, 15.0/(1048.0+1), float64(sn.PdUtilization)/1000.0, 0.001)
		}
	}

	// Check global statistics
	globals, err := dbmodel.GetAllStats(db)
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(4358), globals["total-addresses"])
	require.EqualValues(t, big.NewInt(2145), globals["assigned-addresses"])
	require.EqualValues(t, big.NewInt(4), globals["declined-addresses"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(4355), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["total-nas"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(2460), big.NewInt(math.MaxInt64),
	), globals["assigned-nas"])
	require.EqualValues(t, big.NewInt(3), globals["declined-nas"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(2097), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["total-pds"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(246), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["assigned-pds"])
}

func TestStatsPullerPullStatsKea16Format(t *testing.T) {
	checkStatsPullerPullStats(t, "1.6")
}

func TestStatsPullerPullStatsKea18Format(t *testing.T) {
	checkStatsPullerPullStats(t, "1.8")
}

// Stork should not attempt to get statistics from  the Kea application without the
// stat_cmds hook library.
func TestGetStatsFromAppWithoutStatCmd(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbmodel.InitializeSettings(db, 0)

	fa := agentcommtest.NewFakeAgents(nil, nil)

	app := &dbmodel.App{
		ID:   1,
		Type: dbmodel.AppTypeKea,
		Daemons: []*dbmodel.Daemon{
			{
				Active: true,
				Name:   "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: dbmodel.NewKeaConfig(&map[string]interface{}{
						"Dhcp4": map[string]interface{}{},
					}),
				},
			},
			{
				Active: true,
				Name:   "dhcp6",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: dbmodel.NewKeaConfig(&map[string]interface{}{
						"Dhcp6": map[string]interface{}{},
					}),
				},
			},
		},
	}

	sp, _ := NewStatsPuller(db, fa)

	// Act
	err := sp.getStatsFromApp(app)

	// Assert
	require.NoError(t, err)
	require.Zero(t, fa.CallNo)
}

// Prepares the Kea configuration file with HA hook and some subnets.
func getHATestConfigWithSubnets(rootName, thisServerName, mode string, peerNames ...string) *dbmodel.KeaConfig {
	// Creates standard HA config.
	haConfig := getHATestConfig(rootName, thisServerName, mode, peerNames...)

	// Creates Kea configs with expected subnets.
	dhcp4, dhcp6 := createDhcpConfigs()
	subnetsConfigRaw := dhcp4
	if rootName == "Dhcp6" {
		subnetsConfigRaw = dhcp6
	}
	subnetsConfig, _ := dbmodel.NewKeaConfigFromJSON(subnetsConfigRaw)

	// We are now going to insert hook libraries from one config into another config.
	// We insert the map of hook libraries into the Raw field of the subnetsConfig
	// because this is the structure that holds the entire configuration. The other
	// fields (e.g., DHCPv4Config) only hold partial configuration and are currently
	// used only for reading (rather than setting) the configuration. In the future,
	// we're going to add a mechanics to use the structures to update the entire
	// configuration, and it will also be a good time to refactor these tests. We should
	// have a cleaner way to generate various configs than merging two maps together.
	// This code comes from the old Stork days, though.
	haHooks := (haConfig.Raw)[rootName].(map[string]interface{})["hooks-libraries"].([]interface{})
	subnetHooks := (subnetsConfig.Raw)[rootName].(map[string]interface{})["hooks-libraries"].([]interface{})
	subnetHooks = append(subnetHooks, haHooks...)
	(subnetsConfig.Raw)[rootName].(map[string]interface{})["hooks-libraries"] = subnetHooks

	// Repack the new configuration, so the changes are populated to the parsed
	// data structures and not only reside in the Raw field.
	m, _ := json.Marshal(subnetsConfig)
	_ = json.Unmarshal(m, subnetsConfig)

	return subnetsConfig
}

// Prepares the HA service instances and loads them into database.
// First instance is composed from 3 DHCPv4 daemons and is configured in load
// balancing mode. Second instance is composed from 2 DHCPv6 daemons and is
// configured in hot-standby mode.
func prepareHAEnvironment(t *testing.T, db *pg.DB) (loadBalancing *dbmodel.Service, hotStandby *dbmodel.Service) {
	// Initialize database
	err := dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	err = dbmodel.InitializeStats(db)
	require.NoError(t, err)

	daemons := []*dbmodel.Daemon{}

	// Add machine and app for the primary server.
	m := &dbmodel.Machine{
		ID:        0,
		Address:   "primary",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	app := dbmodel.App{
		MachineID: m.ID,
		Type:      dbmodel.AppTypeKea,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:              dbmodel.AccessPointControl,
				Address:           "192.0.2.33",
				Port:              8000,
				Key:               "",
				UseSecureProtocol: true,
			},
		},
		Daemons: []*dbmodel.Daemon{
			{
				Active: true,
				Name:   "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getHATestConfigWithSubnets("Dhcp4", "server1", "load-balancing",
						"server1", "server2", "server4"),
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
			{
				Active: true,
				Name:   "dhcp6",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getHATestConfigWithSubnets("Dhcp6", "server1", "hot-standby",
						"server1", "server2"),
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
		},
	}

	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)

	daemons = append(daemons, app.Daemons...)

	// Add the secondary machine.
	m = &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	app = dbmodel.App{
		MachineID: m.ID,
		Type:      dbmodel.AppTypeKea,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:              dbmodel.AccessPointControl,
				Address:           "192.0.2.66",
				Key:               "",
				Port:              8000,
				UseSecureProtocol: false,
			},
		},
		Daemons: []*dbmodel.Daemon{
			{
				Active: true,
				Name:   "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getHATestConfigWithSubnets("Dhcp4", "server2", "load-balancing",
						"server1", "server2", "server4"),
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
			{
				Active: true,
				Name:   "dhcp6",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getHATestConfigWithSubnets("Dhcp6", "server2", "hot-standby",
						"server1", "server2"),
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)

	daemons = append(daemons, app.Daemons...)

	// Add machine and app for the DHCPv4 backup server.
	m = &dbmodel.Machine{
		ID:        0,
		Address:   "backup1",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	app = dbmodel.App{
		MachineID: m.ID,
		Type:      dbmodel.AppTypeKea,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:              dbmodel.AccessPointControl,
				Address:           "192.0.2.133",
				Key:               "",
				Port:              8000,
				UseSecureProtocol: false,
			},
		},
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getHATestConfigWithSubnets("Dhcp4", "server4", "load-balancing",
						"server1", "server2", "server4"),
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)

	daemons = append(daemons, app.Daemons...)

	// Detect HA services
	for _, daemon := range daemons {
		services, err := DetectHAServices(db, daemon)
		require.NoError(t, err)
		err = dbmodel.CommitServicesIntoDB(db, services, daemon)
		require.NoError(t, err)
	}

	// There should be two services returned, one for DHCPv4 and one for DHCPv6.
	services, err := dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)

	for _, service := range services {
		innerService := service
		switch service.HAService.HAMode {
		case "load-balancing":
			loadBalancing = &innerService
		case "hot-standby":
			hotStandby = &innerService
		default:
		}
	}

	require.NotNil(t, loadBalancing)
	require.NotNil(t, hotStandby)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	for _, daemon := range daemons {
		sharedNetworks, subnets, err := detectDaemonNetworks(db, daemon, lookup)
		require.NoError(t, err)
		_, err = dbmodel.CommitNetworksIntoDB(db, sharedNetworks, subnets)
		require.NoError(t, err)
		hosts, err := detectGlobalHostsFromConfig(db, daemon, lookup)
		require.NoError(t, err)
		err = dbmodel.CommitGlobalHostsIntoDB(db, hosts)
		require.NoError(t, err)
	}

	return loadBalancing, hotStandby
}

// Checks if the puller counts only the primary server statistics.
func verifyCountingStatisticsFromPrimary(t *testing.T, db *pg.DB) {
	// Check collected stats in the local subnets. There is no meaning if they
	// are from the HA daemons.
	verifyStandardLocalSubnetsStatistics(t, db)

	// Check the subnet utilizations.
	subnets, err := dbmodel.GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Len(t, subnets, 7)

	for _, sn := range subnets {
		switch sn.Prefix {
		case "192.0.2.0/24":
			require.InDelta(t, 111.0/256.0, float64(sn.AddrUtilization)/1000.0, 0.001)
			require.Zero(t, sn.PdUtilization)
		case "192.0.3.0/26":
			require.InDelta(t, 2034.0/(4098.0+2), float64(sn.AddrUtilization)/1000.0, 0.001)
			require.Zero(t, sn.PdUtilization)
		case "2001:db8:1::/64":
			require.InDelta(t, 2400.0/4096.0, float64(sn.AddrUtilization)/1000.0, 0.001)
			require.Zero(t, sn.PdUtilization)
		case "2001:db8:2::/64":
			require.Zero(t, sn.AddrUtilization)
			require.InDelta(t, 233.0/1048.0, float64(sn.PdUtilization)/1000.0, 0.001)
		case "2001:db8:3::/64":
			require.InDelta(t, 60.0/(256.0+2), float64(sn.AddrUtilization)/1000.0, 0.001)
			require.InDelta(t, 15.0/(1048.0+1), float64(sn.PdUtilization)/1000.0, 0.001)
		}
	}

	// Check global statistics
	globals, err := dbmodel.GetAllStats(db)
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(4358), globals["total-addresses"])
	require.EqualValues(t, big.NewInt(2145), globals["assigned-addresses"])
	require.EqualValues(t, big.NewInt(4), globals["declined-addresses"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(4355), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["total-nas"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(2460), big.NewInt(math.MaxInt64),
	), globals["assigned-nas"])
	require.EqualValues(t, big.NewInt(3), globals["declined-nas"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(2097), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["total-pds"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(246), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["assigned-pds"])
}

// Checks if the puller counts only the secondary server statistics.
func verifyCountingStatisticsFromSecondary(t *testing.T, db *pg.DB) {
	// Check collected stats in the local subnets. There is no meaning if they
	// are from the HA daemons.
	verifyStandardLocalSubnetsStatistics(t, db)

	// Check the subnet utilizations.
	subnets, err := dbmodel.GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Len(t, subnets, 7)

	for _, sn := range subnets {
		switch sn.Prefix {
		case "192.0.2.0/24":
			require.InDelta(t, 211.0/456.0, float64(sn.AddrUtilization)/1000.0, 0.001)
			require.Zero(t, sn.PdUtilization)
		case "192.0.3.0/26":
			require.InDelta(t, 2134.0/(4298.0+2), float64(sn.AddrUtilization)/1000.0, 0.001)
			require.Zero(t, sn.PdUtilization)
		case "2001:db8:1::/64":
			require.InDelta(t, 2500.0/4296.0, float64(sn.AddrUtilization)/1000.0, 0.001)
			require.EqualValues(t, 100.0/200.0, float64(sn.PdUtilization)/1000.0, 0.001)
		case "2001:db8:2::/64":
			require.EqualValues(t, 100.0/200.0, float64(sn.AddrUtilization)/1000.0, 0.001)
			require.InDelta(t, 333.0/1248.0, float64(sn.PdUtilization)/1000.0, 0.001)
		case "2001:db8:3::/64":
			require.InDelta(t, 160.0/(456.0+2), float64(sn.AddrUtilization)/1000.0, 0.001)
			require.InDelta(t, 115.0/(1248.0+1), float64(sn.PdUtilization)/1000.0, 0.001)
		}
	}

	// Check global statistics
	globals, err := dbmodel.GetAllStats(db)
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(4758), globals["total-addresses"])
	require.EqualValues(t, big.NewInt(2345), globals["assigned-addresses"])
	require.EqualValues(t, big.NewInt(204), globals["declined-addresses"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(4955), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["total-nas"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(2760), big.NewInt(math.MaxInt64),
	), globals["assigned-nas"])
	require.EqualValues(t, big.NewInt(303), globals["declined-nas"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(2697), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["total-pds"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(546), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["assigned-pds"])
}

func TestGetHATestConfigWithSubnets(t *testing.T) {
	// Act
	config := getHATestConfigWithSubnets("Dhcp4", "server1", "hot-standby", "server2", "server4")

	// Assert
	require.NotNil(t, config)
	path, params, ok := config.GetHookLibraries().GetHAHookLibrary()
	require.True(t, ok)
	require.NotEmpty(t, path)
	relationships := params.GetAllRelationships()
	require.Len(t, relationships, 1)
	require.Equal(t, "server1", *relationships[0].ThisServerName)
	subnets := config.GetSubnets()
	require.NotEmpty(t, subnets)
}

func TestPrepareHAEnvironment(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	loadBalancing, hotStandby := prepareHAEnvironment(t, db)
	keaMock := createKeaMock(func(callNo int) (jsons []string) { return []string{} })

	fa := agentcommtest.NewFakeAgents(keaMock, nil)
	sp, err := NewStatsPuller(db, fa)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, sp)
	require.NotNil(t, loadBalancing)
	require.NotNil(t, hotStandby)
}

// HA pair is detected but the states of the servers are unknown.
// The statistic puller should count only the primary server statistics.
func TestStatsPullerPullStatsHAPairNotInitializedYet(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	loadBalancing, hotStandby := prepareHAEnvironment(t, db)

	keaMock := createStandardKeaMock(false)

	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa)
	require.NoError(t, err)
	defer sp.Shutdown()

	// Act
	err = sp.pullStats()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, loadBalancing)
	require.NotNil(t, hotStandby)
	require.Empty(t, loadBalancing.HAService.PrimaryLastState)
	require.Empty(t, loadBalancing.HAService.SecondaryLastState)
	require.Empty(t, hotStandby.HAService.PrimaryLastState)
	require.Empty(t, hotStandby.HAService.SecondaryLastState)

	verifyCountingStatisticsFromPrimary(t, db)
}

// HA pair is healthy.
// The statistic puller should count only the primary server statistics.
func TestStatsPullerPullStatsHAPairHealthy(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	loadBalancing, hotStandby := prepareHAEnvironment(t, db)
	loadBalancing.HAService.PrimaryLastState = dbmodel.HAStateLoadBalancing
	loadBalancing.HAService.PrimaryReachable = true
	loadBalancing.HAService.SecondaryLastState = dbmodel.HAStateReady
	loadBalancing.HAService.SecondaryReachable = true
	hotStandby.HAService.PrimaryLastState = dbmodel.HAStateHotStandby
	hotStandby.HAService.PrimaryReachable = true
	hotStandby.HAService.SecondaryLastState = dbmodel.HAStateReady
	hotStandby.HAService.SecondaryReachable = true
	_ = dbmodel.UpdateService(db, loadBalancing)
	_ = dbmodel.UpdateService(db, hotStandby)

	keaMock := createStandardKeaMock(false)

	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa)
	require.NoError(t, err)
	defer sp.Shutdown()

	// Act
	err = sp.pullStats()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, loadBalancing)
	require.NotNil(t, hotStandby)

	verifyCountingStatisticsFromPrimary(t, db)
}

// The primary server is down, the secondary one is working.
// The statistic puller should count only the secondary server statistics.
func TestStatsPullerPullStatsHAPairPrimaryIsDownSecondaryIsReady(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	loadBalancing, hotStandby := prepareHAEnvironment(t, db)
	loadBalancing.HAService.PrimaryLastState = dbmodel.HAStateWaiting
	loadBalancing.HAService.PrimaryReachable = true
	loadBalancing.HAService.SecondaryLastState = dbmodel.HAStatePartnerDown
	loadBalancing.HAService.SecondaryReachable = true
	hotStandby.HAService.PrimaryLastState = dbmodel.HAStateSyncing
	hotStandby.HAService.PrimaryReachable = true
	hotStandby.HAService.SecondaryLastState = dbmodel.HAStatePartnerDown
	hotStandby.HAService.SecondaryReachable = true
	_ = dbmodel.UpdateService(db, loadBalancing)
	_ = dbmodel.UpdateService(db, hotStandby)

	keaMock := createStandardKeaMock(false)

	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa)
	require.NoError(t, err)
	defer sp.Shutdown()

	// Act
	err = sp.pullStats()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, loadBalancing)
	require.NotNil(t, hotStandby)

	verifyCountingStatisticsFromSecondary(t, db)
}

// HA pair doesn't work.
// The statistic puller should count only the primary server statistics.
func TestStatsPullerPullStatsHAPairPrimaryIsDownSecondaryIsDown(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	loadBalancing, hotStandby := prepareHAEnvironment(t, db)
	loadBalancing.HAService.PrimaryLastState = dbmodel.HAStateTerminated
	loadBalancing.HAService.PrimaryReachable = false
	loadBalancing.HAService.SecondaryLastState = dbmodel.HAStateTerminated
	loadBalancing.HAService.SecondaryReachable = false
	hotStandby.HAService.PrimaryLastState = dbmodel.HAStateTerminated
	hotStandby.HAService.PrimaryReachable = false
	hotStandby.HAService.SecondaryLastState = dbmodel.HAStateTerminated
	hotStandby.HAService.SecondaryReachable = false
	_ = dbmodel.UpdateService(db, loadBalancing)
	_ = dbmodel.UpdateService(db, hotStandby)

	keaMock := createStandardKeaMock(false)

	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa)
	require.NoError(t, err)
	defer sp.Shutdown()

	// Act
	err = sp.pullStats()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, loadBalancing)
	require.NotNil(t, hotStandby)

	verifyCountingStatisticsFromPrimary(t, db)
}

//go:embed testdata/kea-dhcp6_v2.5.5_stat-lease6-get_big-numbers.json
var statisticGetAllBigNumbersJSON []byte

// Test that unmarshalling of the Kea stat-lease6-get response does not lose
// precision when the values exceed the maximum value of int64.
func TestUnmarshalStatLeaseGetResponse(t *testing.T) {
	// Arrange
	var response []StatLeaseGetResponse
	expected0, _ := big.NewInt(0).SetString("844424930131968", 10)
	expected1, _ := big.NewInt(0).SetString("281474976710656", 10)
	expected2, _ := big.NewInt(0).SetString("2417851639229258349412352", 10)
	expectedValues := []*big.Int{expected0, expected1, expected2}

	// Act
	err := json.Unmarshal(statisticGetAllBigNumbersJSON, &response)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "subnet-id", response[0].Arguments.ResultSet.Columns[0])
	require.Equal(t, "total-nas", response[0].Arguments.ResultSet.Columns[1])

	for i, value := range expectedValues {
		require.EqualValues(t, i+1, response[0].Arguments.ResultSet.Rows[i][0].BigInt().Int64())
		require.Equal(t,
			value,
			response[0].Arguments.ResultSet.Rows[i][1].BigInt(),
		)
	}
}

// Test that the statistics with values exceeding the maximum value of int64
// are stored without loss of precision.
func TestProcessAppResponsesForResponseWithBigNumbers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)
	fa := agentcommtest.NewFakeAgents(nil, nil)
	puller, _ := NewStatsPuller(db, fa)

	var response []StatLeaseGetResponse
	_ = json.Unmarshal(statisticGetAllBigNumbersJSON, &response)

	// Seed database.
	machine := &dbmodel.Machine{Address: "localhost", AgentPort: 8080}
	_ = dbmodel.AddMachine(db, machine)

	app := &dbmodel.App{
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeKea,
		Active:    true,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp6", true),
		},
	}
	daemons, _ := dbmodel.AddApp(db, app)

	for i := 1; i <= 3; i++ {
		subnet := &dbmodel.Subnet{
			Prefix: fmt.Sprintf("3001:%d::/48", i),
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					DaemonID:      daemons[0].ID,
					LocalSubnetID: int64(i),
				},
			},
		}
		err := dbmodel.AddSubnet(db, subnet)
		require.NoError(t, err)
		err = dbmodel.AddLocalSubnets(db, subnet)
		require.NoError(t, err)
	}

	// Act
	err := puller.processAppResponses(app, []*keactrl.Command{keactrl.NewCommandBase(keactrl.StatLease6Get)}, daemons, []any{&response})

	// Assert
	require.NoError(t, err)
	subnets, err := dbmodel.GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Len(t, subnets, 3)

	subnet := subnets[0]
	require.Len(t, subnet.LocalSubnets, 1)
	require.EqualValues(t, 1, subnet.LocalSubnets[0].LocalSubnetID)
	stats := subnet.LocalSubnets[0].Stats
	require.Equal(t, uint64(844424930131968), stats["total-nas"])
	require.Equal(t, uint64(0), stats["cumulative-assigned-nas"])
	require.Equal(t, uint64(9), stats["assigned-nas"])
	require.Equal(t, uint64(10), stats["declined-addresses"])

	subnet = subnets[1]
	require.Len(t, subnet.LocalSubnets, 1)
	require.EqualValues(t, 2, subnet.LocalSubnets[0].LocalSubnetID)
	stats = subnet.LocalSubnets[0].Stats
	require.Equal(t, uint64(281474976710656), stats["total-nas"])
	require.Equal(t, uint64(0), stats["cumulative-assigned-nas"])
	require.Equal(t, uint64(0), stats["assigned-nas"])
	require.Equal(t, uint64(0), stats["declined-addresses"])

	subnet = subnets[2]
	require.Len(t, subnet.LocalSubnets, 1)
	require.EqualValues(t, 3, subnet.LocalSubnets[0].LocalSubnetID)
	stats = subnet.LocalSubnets[0].Stats
	expectedTotalNAs, _ := big.NewInt(0).SetString("2417851639229258349412352", 10)
	require.Equal(t, expectedTotalNAs, stats["total-nas"])
	require.Equal(t, uint64(0), stats["cumulative-assigned-nas"])
	require.Equal(t, uint64(0), stats["assigned-nas"])
	require.Equal(t, uint64(0), stats["declined-addresses"])
}
