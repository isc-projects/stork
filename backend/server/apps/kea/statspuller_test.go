package kea

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Prepares the Kea mock. It accepts list of serialized JSON responses in order:
// 1. Statistic-get-all DHCPv4
// 3. Statistic-get-all DHCPv6.
func createKeaMock(t *testing.T, jsonFactory func(callNo int) (jsons []string)) func(callNo int, cmdResponses []interface{}) {
	return func(callNo int, cmdResponses []interface{}) {
		jsons := jsonFactory(callNo)
		// DHCPv4
		daemons := []keactrl.DaemonName{keactrl.DHCPv4}
		command := keactrl.NewCommandBase(keactrl.StatisticGetAll, daemons...)
		err := keactrl.UnmarshalResponseList(command, []byte(jsons[0]), cmdResponses[0])
		require.NoError(t, err)

		if len(cmdResponses) < 2 {
			return
		}

		// DHCPv6
		daemons = []keactrl.DaemonName{keactrl.DHCPv6}
		command = keactrl.NewCommandBase(keactrl.StatisticGetAll, daemons...)
		err = keactrl.UnmarshalResponseList(command, []byte(jsons[1]), cmdResponses[1])
		require.NoError(t, err)
	}
}

// Prepares the Kea mock with the statistic values compatible with the
// configurations produced by the createDhcpConfigs function.
// It assigns different, predictable values for each application. Supports the
// applications with a single DHCPv4 daemon.
// Accepts a parameter that indicates if the old names of the statistics should
// be used (missing doubled "s" in "addresses" word).
func createStandardKeaMock(t *testing.T, oldStatsFormat bool) func(callNo int, cmdResponses []any) {
	statistic4Names := []string{"total-addresses", "assigned-addresses", "declined-addresses"}
	if oldStatsFormat {
		statistic4Names = []string{"total-addreses", "assigned-addreses", "declined-addreses"}
	}

	return createKeaMock(t, func(callNo int) (jsons []string) {
		shift := int64(callNo * 100)
		totalShift := shift * 2

		return []string{
			fmt.Sprintf(`[{
				"result": 0,
				"arguments": {
					"subnet[10].%[1]s": [ [ %[4]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[10].%[2]s": [ [ %[5]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[10].%[3]s": [ [ %[6]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[10].pool[0].%[1]s": [ [ %[4]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[10].pool[0].%[2]s": [ [ %[5]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[10].pool[0].%[3]s": [ [ %[6]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[20].%[1]s": [ [ %[7]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[20].%[2]s": [ [ %[8]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[20].%[3]s": [ [ %[9]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[20].pool[0].%[1]s": [ [ %[10]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[20].pool[0].%[2]s": [ [ %[11]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[20].pool[0].%[3]s": [ [ %[12]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[20].pool[1].%[1]s": [ [ %[13]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[20].pool[1].%[2]s": [ [ %[14]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[20].pool[1].%[3]s": [ [ %[15]d, "2019-07-30 10:13:00.000000" ] ],
					"pkt4-ack-sent": [ [ 44, "2019-07-30 10:13:00.000000" ] ]
				}
			}]`,
				// Labels.
				// 1.               2.                  3.
				statistic4Names[0], statistic4Names[1], statistic4Names[2],
				// Subnet 10 (and pool 0).
				// 4.           5.         6.
				256+totalShift, 111+shift, 0+shift,
				// Subnet 20.
				// 7.            8.          9.
				4098+totalShift, 2034+shift, 4+shift,
				// Pool 0 in subnet 20.
				// 10.         11.      12.
				10+totalShift, 4+shift, 2+shift,
				// Pool 1 in subnet 20.
				// 13. 14.  15.
				4088, 2030, 2,
			),
			fmt.Sprintf(`[{
				"result": 0,
				"arguments": {
					"subnet[30].total-nas": [ [ %[1]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[30].assigned-nas": [ [ %[2]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[30].declined-addresses": [ [ %[3]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[30].total-pds": [ [ %[4]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[30].assigned-pds": [ [ %[5]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[30].pool[0].total-nas": [ [ %[1]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[30].pool[0].assigned-nas": [ [ %[2]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[30].pool[0].declined-addresses": [ [ %[3]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[30].pd-pool[0].total-pds": [ [ %[4]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[30].pd-pool[0].assigned-pds": [ [ %[5]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].total-nas": [ [ %[6]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].assigned-nas": [ [ %[7]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].declined-addresses": [ [ %[8]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].total-pds": [ [ %[9]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].assigned-pds": [ [ %[10]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].pool[0].total-nas": [ [ %[6]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].pool[0].assigned-nas": [ [ %[7]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].pool[0].declined-addresses": [ [ %[8]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].pd-pool[0].total-pds": [ [ %[9]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[40].pd-pool[0].assigned-pds": [ [ %[10]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].total-nas": [ [ %[11]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].assigned-nas": [ [ %[12]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].declined-addresses": [ [ %[13]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].total-pds": [ [ %[14]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].assigned-pds": [ [ %[15]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].total-nas": [ [ %[16]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].assigned-nas": [ [ %[17]s, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].declined-addresses": [ [ %[18]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].total-pds": [ [ %[19]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].assigned-pds": [ [ %[20]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].pool[0].total-nas": [ [ %[16]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].pool[0].assigned-nas": [ [ %[17]s, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].pool[0].declined-addresses": [ [ %[18]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].pd-pool[0].total-pds": [ [ %[19]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[60].pd-pool[0].assigned-pds": [ [ %[20]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pool[0].total-nas": [ [ %[21]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pool[0].assigned-nas": [ [ %[22]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pool[0].declined-addresses": [ [ %[23]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pd-pool[0].total-pds": [ [ %[24]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pd-pool[0].assigned-pds": [ [ %[25]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pool[1].total-nas": [ [ %[26]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pool[1].assigned-nas": [ [ %[27]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pool[1].declined-addresses": [ [ %[28]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pd-pool[1].total-pds": [ [ %[29]d, "2019-07-30 10:13:00.000000" ] ],
					"subnet[50].pd-pool[1].assigned-pds": [ [ %[30]d, "2019-07-30 10:13:00.000000" ] ],
					"pkt6-reply-sent": [ [ 66, "2019-07-30 10:13:00.000000" ] ]
				}
			}]`,
				// Subnet 30.
				// 1.            2.          3.       4.            5.
				4096+totalShift, 2400+shift, 3+shift, 0+totalShift, 0+shift,
				// Subnet 40.
				// 6.         7.       8.       9.               10.
				0+totalShift, 0+shift, 0+shift, 1048+totalShift, 233+shift,
				// Subnet 50.
				// 11.          12.       13.       14.              15.
				256+totalShift, 60+shift, 10+shift, 1048+totalShift, 35+shift,
				// Subnet 60.
				// 16. 17.                 18. 19. 20.
				-1, "9223372036854775807", 0, -2, -3,
				// Address pool 0 and PD pool 0 in subnet 50.
				// 21.         22.      23.      24.            25.
				20+totalShift, 8+shift, 4+shift, 40+totalShift, 20+shift,
				// Address pool 1 and PD pool 1 in subnet 50.
				// 26.         27.      28.      29.            30.
				234, 50, 0, 1008, 5,
			),
		}
	})
}

// Creates test DHCPv4 and DHCPv6 configurations with some subnets and
// reservations.
func createDhcpConfigs() (string, string) {
	dhcp4 := `{
		"Dhcp4": {
			"hooks-libraries": [],
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
					"subnet": "192.0.2.0/24",
					"pools": [
						{
							"pool": "192.0.2.1 - 192.0.2.10"
						}
					]
				},
				{
					"id": 20,
					"subnet": "192.0.3.0/24",
					"pools": [
						{
							"pool": "192.0.3.1 - 192.0.3.10"
						},
						{
							"pool": "192.0.3.11 - 192.0.3.20"
						},
						{
							"pool": "192.0.3.21 - 192.0.3.30",
							"pool-id": 1
						}
					],
					// 1 in-pool, 2 out-of-pool host reservations
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
			"hooks-libraries": [],
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
					"subnet": "2001:db8:1::/64",
					"pools": [{
						"pool": "2001:db8:1::100-2001:db8:1::ffff"
					}],
					"pd-pools": [{
						"prefix": "2001:db8:1:8000::",
						"prefix-len": 64,
						"delegated-len": 80
					}]
				},
				{
					"id": 40,
					"subnet": "2001:db8:2::/64",
					"pools": [{
						"pool": "2001:db8:2::100-2001:db8:2::ffff"
					}],
					"pd-pools": [{
						"prefix": "2001:db8:2:8000::",
						"prefix-len": 64,
						"delegated-len": 80
					}]
				},
				{
					"id": 50,
					"subnet": "2001:db8:3::/64",
					"pools": [
						{
							"pool": "2001:db8:3::100-2001:db8:3::ffff"
						},
						{
							"pool": "2001:db8:3::1:100-2001:db8:3::1:ffff"
						},
						{
							"pool": "2001:db8:3::2:100-2001:db8:3::2:ffff",
							"pool-id": 1
						}
					],
					"pd-pools": [
						{
							"prefix": "2001:db8:3:1:8000::",
							"prefix-len": 64,
							"delegated-len": 80
						},
						{
							"prefix": "2001:db8:2:8000::",
							"prefix-len": 64,
							"delegated-len": 80
						},
						{
							"prefix": "2001:db8:3:8000::",
							"prefix-len": 64,
							"delegated-len": 80,
							"pool-id": 1
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
	q = q.Relation("AddressPools")
	q = q.Relation("PrefixPools")
	err := q.Select()
	require.NoError(t, err)
	snCnt := 0
	for _, sn := range localSubnets {
		sort.Slice(sn.AddressPools, func(i, j int) bool {
			return sn.AddressPools[i].KeaParameters.PoolID < sn.AddressPools[j].KeaParameters.PoolID
		})
		sort.Slice(sn.PrefixPools, func(i, j int) bool {
			return sn.PrefixPools[i].KeaParameters.PoolID < sn.PrefixPools[j].KeaParameters.PoolID
		})

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

			// Check pools.
			require.Len(t, sn.AddressPools, 3)

			require.Zero(t, sn.AddressPools[0].KeaParameters.PoolID)
			require.Equal(t, uint64(10+totalShift), sn.AddressPools[0].Stats["total-addresses"])
			require.Equal(t, uint64(4+shift), sn.AddressPools[0].Stats["assigned-addresses"])
			require.Equal(t, uint64(2+shift), sn.AddressPools[0].Stats["declined-addresses"])
			require.Zero(t, sn.AddressPools[1].KeaParameters.PoolID)
			require.Equal(t, uint64(10+totalShift), sn.AddressPools[1].Stats["total-addresses"])
			require.Equal(t, uint64(4+shift), sn.AddressPools[1].Stats["assigned-addresses"])
			require.Equal(t, uint64(2+shift), sn.AddressPools[1].Stats["declined-addresses"])
			require.EqualValues(t, 1, sn.AddressPools[2].KeaParameters.PoolID)
			require.Equal(t, uint64(4088), sn.AddressPools[2].Stats["total-addresses"])
			require.Equal(t, uint64(2030), sn.AddressPools[2].Stats["assigned-addresses"])
			require.Equal(t, uint64(2), sn.AddressPools[2].Stats["declined-addresses"])
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
			require.Equal(t, uint64(35+shift), sn.Stats["assigned-pds"])
			require.Equal(t, uint64(10+shift), sn.Stats["declined-nas"])
			require.Equal(t, uint64(256+totalShift), sn.Stats["total-nas"])
			require.Equal(t, uint64(1048+totalShift), sn.Stats["total-pds"])
			snCnt++

			// Check pools.
			require.Len(t, sn.AddressPools, 3)

			require.Zero(t, sn.AddressPools[0].KeaParameters.PoolID)
			require.Equal(t, uint64(20+totalShift), sn.AddressPools[0].Stats["total-nas"])
			require.Equal(t, uint64(8+shift), sn.AddressPools[0].Stats["assigned-nas"])
			require.Equal(t, uint64(4+shift), sn.AddressPools[0].Stats["declined-nas"])

			require.Zero(t, sn.AddressPools[1].KeaParameters.PoolID)
			require.Equal(t, uint64(20+totalShift), sn.AddressPools[1].Stats["total-nas"])
			require.Equal(t, uint64(8+shift), sn.AddressPools[1].Stats["assigned-nas"])
			require.Equal(t, uint64(4+shift), sn.AddressPools[1].Stats["declined-nas"])

			require.EqualValues(t, 1, sn.AddressPools[2].KeaParameters.PoolID)
			require.Equal(t, uint64(234), sn.AddressPools[2].Stats["total-nas"])
			require.Equal(t, uint64(50), sn.AddressPools[2].Stats["assigned-nas"])
			require.Equal(t, uint64(0), sn.AddressPools[2].Stats["declined-nas"])

			require.Len(t, sn.PrefixPools, 3)

			require.Zero(t, sn.PrefixPools[0].KeaParameters.PoolID)
			require.Equal(t, uint64(40+totalShift), sn.PrefixPools[0].Stats["total-pds"])
			require.Equal(t, uint64(20+shift), sn.PrefixPools[0].Stats["assigned-pds"])

			require.Zero(t, sn.PrefixPools[1].KeaParameters.PoolID)
			require.Equal(t, uint64(40+totalShift), sn.PrefixPools[1].Stats["total-pds"])
			require.Equal(t, uint64(20+shift), sn.PrefixPools[1].Stats["assigned-pds"])

			require.EqualValues(t, 1, sn.PrefixPools[2].KeaParameters.PoolID)
			require.Equal(t, uint64(1008), sn.PrefixPools[2].Stats["total-pds"])
			require.Equal(t, uint64(5), sn.PrefixPools[2].Stats["assigned-pds"])
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
// empty or when Kea returns an error.  The RPS responses are valid,
// they have their own unit tests in rps_test.go.
func TestStatsPullerEmptyResponse(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	_ = createAppWithSubnets(t, db, 0, "", "")

	// prepare fake agents
	keaMock := createKeaMock(t, func(callNo int) (jsons []string) {
		return []string{
			// simulate empty response
			`[{
				"result": 0, "text": "Everything is fine",
				"arguments": {
					"pkt4-ack-sent": [ [ 0, "2019-07-30 10:13:00.000000" ] ]
				}
			}]`,
			// simulate an error is returned from Kea
			`[{
				"result": 1, "text": "error occurred"
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

	keaMock := createStandardKeaMock(t, statsFormat == "1.6")

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
	require.Equal(t, 2, len(sp.PreviousRps))

	// Entry for ID 1 should be dhcp4 daemon, it should have an RPS value of 44
	previous := sp.PreviousRps[1]
	require.NotEqual(t, nil, previous)
	require.EqualValues(t, 44, previous.Value)

	// Entry for ID 2 should be dhcp6 daemon, it should have an RPS value of 66
	previous = sp.PreviousRps[2]
	require.NotEqual(t, nil, previous)
	require.EqualValues(t, 66, previous.Value)

	// Check out-of-pool addresses/NAs/PDs utilization
	subnets, _ := dbmodel.GetAllSubnets(db, 0)

	for _, sn := range subnets {
		switch sn.LocalSubnets[0].LocalSubnetID {
		case 10:
			require.InDelta(t, 111.0/256.0, float64(sn.AddrUtilization), 0.001)
			require.Zero(t, sn.PdUtilization)
			require.EqualValues(t, 256, sn.Stats.GetBigCounter(dbmodel.StatNameTotalAddresses).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameTotalOutOfPoolAddresses).ToInt64())
			require.EqualValues(t, 111, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedAddresses).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedOutOfPoolAddresses).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedAddresses).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedOutOfPoolAddresses).ToInt64())
		case 20:
			require.InDelta(t, 2034.0/(4098.0+2), float64(sn.AddrUtilization), 0.001)
			require.Zero(t, sn.PdUtilization)
			require.EqualValues(t, 4100, sn.Stats.GetBigCounter(dbmodel.StatNameTotalAddresses).ToInt64())
			require.EqualValues(t, 2, sn.Stats.GetBigCounter(dbmodel.StatNameTotalOutOfPoolAddresses).ToInt64())
			require.EqualValues(t, 2034, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedAddresses).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedOutOfPoolAddresses).ToInt64())
			require.EqualValues(t, 4, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedAddresses).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedOutOfPoolAddresses).ToInt64())
		case 30:
			require.InDelta(t, 2400.0/4096.0, float64(sn.AddrUtilization), 0.001)
			require.Zero(t, sn.PdUtilization)
			require.EqualValues(t, 4096, sn.Stats.GetBigCounter(dbmodel.StatNameTotalNAs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameTotalOutOfPoolNAs).ToInt64())
			require.EqualValues(t, 2400, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedNAs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedOutOfPoolNAs).ToInt64())
			require.EqualValues(t, 3, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedNAs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedOutOfPoolNAs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameTotalPDs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameTotalOutOfPoolPDs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedPDs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedOutOfPoolPDs).ToInt64())
		case 40:
			require.Zero(t, sn.AddrUtilization)
			require.InDelta(t, 233.0/1048.0, float64(sn.PdUtilization), 0.001)
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameTotalNAs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameTotalOutOfPoolNAs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedNAs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedOutOfPoolNAs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedNAs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedOutOfPoolNAs).ToInt64())
			require.EqualValues(t, 1048, sn.Stats.GetBigCounter(dbmodel.StatNameTotalPDs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameTotalOutOfPoolPDs).ToInt64())
			require.EqualValues(t, 233, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedPDs).ToInt64())
			require.Zero(t, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedOutOfPoolPDs).ToInt64())
		case 50:
			require.InDelta(t, 60.0/(254.0+4), float64(sn.AddrUtilization), 0.001)
			require.InDelta(t, 35.0/(1048.0+1), float64(sn.PdUtilization), 0.001)
			require.EqualValues(t, 254+4, sn.Stats.GetBigCounter(dbmodel.StatNameTotalNAs).ToInt64())
			require.EqualValues(t, 4, sn.Stats.GetBigCounter(dbmodel.StatNameTotalOutOfPoolNAs).ToInt64())
			require.EqualValues(t, 60, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedNAs).ToInt64())
			require.EqualValues(t, 2, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedOutOfPoolNAs).ToInt64())
			require.EqualValues(t, 10, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedNAs).ToInt64())
			require.EqualValues(t, 6, sn.Stats.GetBigCounter(dbmodel.StatNameDeclinedOutOfPoolNAs).ToInt64())
			require.EqualValues(t, 1048+1, sn.Stats.GetBigCounter(dbmodel.StatNameTotalPDs).ToInt64())
			require.EqualValues(t, 1, sn.Stats.GetBigCounter(dbmodel.StatNameTotalOutOfPoolPDs).ToInt64())
			require.EqualValues(t, 35, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedPDs).ToInt64())
			require.EqualValues(t, 10, sn.Stats.GetBigCounter(dbmodel.StatNameAssignedOutOfPoolPDs).ToInt64())
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
	require.EqualValues(t, big.NewInt(13), globals["declined-nas"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(2097), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["total-pds"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(266), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["assigned-pds"])
}

func TestStatsPullerPullStatsKea16Format(t *testing.T) {
	checkStatsPullerPullStats(t, "1.6")
}

func TestStatsPullerPullStatsKea18Format(t *testing.T) {
	checkStatsPullerPullStats(t, "1.8")
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
			require.InDelta(t, 111.0/256.0, float64(sn.AddrUtilization), 0.001)
			require.Zero(t, sn.PdUtilization)
		case "192.0.3.0/26":
			require.InDelta(t, 2034.0/(4098.0+2), float64(sn.AddrUtilization), 0.001)
			require.Zero(t, sn.PdUtilization)
		case "2001:db8:1::/64":
			require.InDelta(t, 2400.0/4096.0, float64(sn.AddrUtilization), 0.001)
			require.Zero(t, sn.PdUtilization)
		case "2001:db8:2::/64":
			require.Zero(t, sn.AddrUtilization)
			require.InDelta(t, 233.0/1048.0, float64(sn.PdUtilization), 0.001)
		case "2001:db8:3::/64":
			require.InDelta(t, 60.0/(256.0+2), float64(sn.AddrUtilization), 0.001)
			require.InDelta(t, 35.0/(1048.0+1), float64(sn.PdUtilization), 0.001)
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
	require.EqualValues(t, big.NewInt(13), globals["declined-nas"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(2097), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["total-pds"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(266), big.NewInt(0).SetUint64(math.MaxUint64),
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
			require.InDelta(t, 211.0/456.0, float64(sn.AddrUtilization), 0.001)
			require.Zero(t, sn.PdUtilization)
		case "192.0.3.0/26":
			require.InDelta(t, 2134.0/(4298.0+2), float64(sn.AddrUtilization), 0.001)
			require.Zero(t, sn.PdUtilization)
		case "2001:db8:1::/64":
			require.InDelta(t, 2500.0/4296.0, float64(sn.AddrUtilization), 0.001)
			require.EqualValues(t, 100.0/200.0, float64(sn.PdUtilization), 0.001)
		case "2001:db8:2::/64":
			require.EqualValues(t, 100.0/200.0, float64(sn.AddrUtilization), 0.001)
			require.InDelta(t, 333.0/1248.0, float64(sn.PdUtilization), 0.001)
		case "2001:db8:3::/64":
			require.InDelta(t, 160.0/(456.0+2), float64(sn.AddrUtilization), 0.001)
			require.InDelta(t, 135.0/(1248.0+1), float64(sn.PdUtilization), 0.001)
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
	require.EqualValues(t, big.NewInt(313), globals["declined-nas"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(2697), big.NewInt(0).SetUint64(math.MaxUint64),
	), globals["total-pds"])
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(566), big.NewInt(0).SetUint64(math.MaxUint64),
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
	keaMock := createKeaMock(t, func(callNo int) (jsons []string) { return []string{} })

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

	keaMock := createStandardKeaMock(t, false)

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

	require.Len(t, fa.RecordedCommands, 5)
	for _, command := range fa.RecordedCommands {
		daemons := command.GetDaemonsList()
		require.Len(t, daemons, 1)
		require.Contains(t, []string{dbmodel.DaemonNameDHCPv4, dbmodel.DaemonNameDHCPv6}, daemons[0])
	}

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

	keaMock := createStandardKeaMock(t, false)

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

	keaMock := createStandardKeaMock(t, false)

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

	keaMock := createStandardKeaMock(t, false)

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

//go:embed testdata/kea-dhcp6_v2.5.5_statistic-get-all_big-numbers.json
var statisticGetAllBigNumbersJSON []byte

// Test that the statistics with values exceeding the maximum value of int64
// are stored without loss of precision.
func TestProcessAppResponsesForResponseWithBigNumbers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)
	fa := agentcommtest.NewFakeAgents(nil, nil)
	puller, _ := NewStatsPuller(db, fa)

	var response keactrl.StatisticGetAllResponse
	err := json.Unmarshal(statisticGetAllBigNumbersJSON, &response)
	require.NoError(t, err)

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

	for i := 1; i <= 10; i++ {
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
		for _, ls := range subnet.LocalSubnets {
			for _, pool := range ls.AddressPools {
				err = dbmodel.AddAddressPool(db, &pool)
				require.NoError(t, err)
			}
		}
		for _, ls := range subnet.LocalSubnets {
			for _, pool := range ls.PrefixPools {
				err = dbmodel.AddPrefixPool(db, &pool)
				require.NoError(t, err)
			}
		}
	}

	// Act
	err = puller.processAppResponses(
		app, []*keactrl.Command{keactrl.NewCommandBase(keactrl.StatisticGetAll)},
		daemons, []keactrl.StatisticGetAllResponseItem{response[0]},
	)

	// Assert
	require.NoError(t, err)
	subnets, err := dbmodel.GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Len(t, subnets, 10)

	subnet := subnets[0]
	require.Len(t, subnet.LocalSubnets, 1)
	require.EqualValues(t, 1, subnet.LocalSubnets[0].LocalSubnetID)
	stats := subnet.LocalSubnets[0].Stats
	require.Equal(t, uint64(844424930131968), stats["total-nas"])
	require.Equal(t, uint64(0), stats["cumulative-assigned-nas"])
	require.Equal(t, uint64(2), stats["assigned-nas"])
	require.Equal(t, uint64(1), stats["declined-addresses"])

	subnet = subnets[1]
	require.Len(t, subnet.LocalSubnets, 1)
	require.EqualValues(t, 2, subnet.LocalSubnets[0].LocalSubnetID)
	stats = subnet.LocalSubnets[0].Stats
	require.Equal(t, uint64(281474976710656), stats["total-nas"])
	require.Equal(t, uint64(0), stats["cumulative-assigned-nas"])
	require.Equal(t, uint64(0), stats["assigned-nas"])
	require.Equal(t, uint64(0), stats["declined-addresses"])

	subnet = subnets[4]
	require.Len(t, subnet.LocalSubnets, 1)
	require.EqualValues(t, 5, subnet.LocalSubnets[0].LocalSubnetID)
	stats = subnet.LocalSubnets[0].Stats
	expectedTotalNAs, _ := big.NewInt(0).SetString("36893488147419103232", 10)
	require.Equal(t, expectedTotalNAs, stats["total-nas"])
	require.Equal(t, uint64(0), stats["cumulative-assigned-nas"])
	require.Equal(t, uint64(0), stats["assigned-nas"])
	require.Equal(t, uint64(0), stats["declined-addresses"])
}

// Test that an error is returned when the number of responses does not match
// the number of commands.
func TestProcessAppResponsesWithDifferentNumberOfResponses(t *testing.T) {
	// Arrange
	puller := &StatsPuller{}

	responses := []keactrl.StatisticGetAllResponseItem{{}, {}}
	commandsCases := [][]*keactrl.Command{
		// More responses than commands.
		{keactrl.NewCommandBase(keactrl.StatisticGetAll)},
		// More commands than responses.
		{
			keactrl.NewCommandBase(keactrl.StatisticGetAll),
			keactrl.NewCommandBase(keactrl.StatisticGetAll),
			keactrl.NewCommandBase(keactrl.StatisticGetAll),
		},
	}
	caseLabels := []string{
		"More responses than commands", "More commands than responses",
	}

	for i, label := range caseLabels {
		t.Run(label, func(t *testing.T) {
			commands := commandsCases[i]

			// Act
			err := puller.processAppResponses(
				nil, commands, nil, responses,
			)

			// Assert
			require.ErrorContains(
				t, err,
				fmt.Sprintf(
					"number of commands (%d) does not match number of responses (%d)",
					len(commands), len(responses),
				),
			)
		})
	}
}
