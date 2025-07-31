package kea

import (
	"encoding/json"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Test that the statistics counter is properly constructed.
func TestCounterConstruction(t *testing.T) {
	// Act
	counter := newStatisticsCounter()

	// Assert
	require.Zero(t, counter.global.totalIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDelegatedPrefixes.ToInt64())
	require.Zero(t, counter.global.totalAssignedDelegatedPrefixes.ToInt64())
	require.Len(t, counter.sharedNetworks, 0)
}

// Test that the counter returns statistics for IPv4 subnet with single local subnet.
func TestCounterAddSingleIPv4LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "192.0.2.0/24",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(100),
					"assigned-addresses": uint64(10),
					"declined-addresses": uint64(20),
				},
			},
		},
	}

	counter := newStatisticsCounter()

	// Act
	statistics := counter.add(subnet)

	// Assert
	require.InDelta(t, float64(0.1), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))

	require.EqualValues(t, 100, counter.global.totalIPv4Addresses.ToInt64())
	require.EqualValues(t, 10, counter.global.totalAssignedIPv4Addresses.ToInt64())
	require.EqualValues(t, 20, counter.global.totalDeclinedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDelegatedPrefixes.ToInt64())
	require.Zero(t, counter.global.totalAssignedDelegatedPrefixes.ToInt64())

	require.Len(t, counter.sharedNetworks, 0)
}

// Test that the counter returns utilization for IPv6 subnet with single local subnet.
func TestCounterAddSingleIPv6LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(100),
					"assigned-nas": uint64(40),
					"declined-nas": uint64(30),
					"total-pds":    uint64(20),
					"assigned-pds": uint64(10),
				},
			},
		},
	}

	counter := newStatisticsCounter()

	// Act
	statistics := counter.add(subnet)

	// Assert
	require.InDelta(t, float64(0.4), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.5), statistics.GetDelegatedPrefixUtilization(), float64(0.001))

	require.Zero(t, counter.global.totalIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv4Addresses.ToInt64())
	require.EqualValues(t, 100, counter.global.totalIPv6Addresses.ToInt64())
	require.EqualValues(t, 40, counter.global.totalAssignedIPv6Addresses.ToInt64())
	require.EqualValues(t, 30, counter.global.totalDeclinedIPv6Addresses.ToInt64())
	require.EqualValues(t, 20, counter.global.totalDelegatedPrefixes.ToInt64())
	require.EqualValues(t, 10, counter.global.totalAssignedDelegatedPrefixes.ToInt64())

	require.Len(t, counter.sharedNetworks, 0)
}

// Test that the except positive int64 statistics data types other than uint64 and int64 aren't
// supported.
func TestCounterAddSubnetUsingNonUint64OrInt64(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-nas":    int64(100),
					"assigned-nas": int32(40),
					"declined-nas": int16(30),
					"total-pds":    int(20),
					"assigned-pds": uint32(10),
				},
			},
		},
	}

	counter := newStatisticsCounter()

	// Act
	statistics := counter.add(subnet)

	// Assert
	require.InDelta(t, float64(0.0), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))

	require.Zero(t, counter.global.totalIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv4Addresses.ToInt64())
	// The positive int64 values are accepted.
	require.Equal(t, int64(100), counter.global.totalIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDelegatedPrefixes.ToInt64())
	require.Zero(t, counter.global.totalAssignedDelegatedPrefixes.ToInt64())

	require.Len(t, counter.sharedNetworks, 0)
}

// Test that the counter returns the proper utilization for multiple IPv4 local subnets.
func TestCounterAddMultipleIPv4LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "192.0.2.0/24",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(100),
					"assigned-addresses": uint64(10),
					"declined-addresses": uint64(20),
				},
			},
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(200),
					"assigned-addresses": uint64(20),
					"declined-addresses": uint64(40),
				},
			},
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(5),
					"assigned-addresses": uint64(3),
					"declined-addresses": uint64(1),
				},
			},
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(50),
					"assigned-addresses": uint64(1),
					"declined-addresses": uint64(2),
				},
			},
		},
	}

	counter := newStatisticsCounter()

	// Act
	statistics := counter.add(subnet)

	// Assert
	require.InDelta(t, float64(34.0/355.0), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))

	require.EqualValues(t, 355, counter.global.totalIPv4Addresses.ToInt64())
	require.EqualValues(t, 34, counter.global.totalAssignedIPv4Addresses.ToInt64())
	require.EqualValues(t, 63, counter.global.totalDeclinedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDelegatedPrefixes.ToInt64())
	require.Zero(t, counter.global.totalAssignedDelegatedPrefixes.ToInt64())
}

// Test that the counter returns the proper utilization for multiple IPv6 local subnets.
func TestCounterAddMultipleIPv6LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(100),
					"assigned-nas": uint64(10),
					"declined-nas": uint64(20),
					"total-pds":    uint64(40),
					"assigned-pds": uint64(30),
				},
			},
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(200),
					"assigned-nas": uint64(20),
					"declined-nas": uint64(40),
					"total-pds":    uint64(100),
					"assigned-pds": uint64(10),
				},
			},
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(5),
					"assigned-nas": uint64(3),
					"declined-nas": uint64(1),
					"total-pds":    uint64(3),
					"assigned-pds": uint64(1),
				},
			},
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(50),
					"assigned-nas": uint64(1),
					"declined-nas": uint64(2),
					"total-pds":    uint64(100),
					"assigned-pds": uint64(3),
				},
			},
		},
	}

	counter := newStatisticsCounter()

	// Act
	statistics := counter.add(subnet)

	// Assert
	require.InDelta(t, float64(34.0/355.0), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(44.0/243.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))

	require.Zero(t, counter.global.totalIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv4Addresses.ToInt64())
	require.EqualValues(t, 355, counter.global.totalIPv6Addresses.ToInt64())
	require.EqualValues(t, 34, counter.global.totalAssignedIPv6Addresses.ToInt64())
	require.EqualValues(t, 63, counter.global.totalDeclinedIPv6Addresses.ToInt64())
	require.EqualValues(t, 243, counter.global.totalDelegatedPrefixes.ToInt64())
	require.EqualValues(t, 44, counter.global.totalAssignedDelegatedPrefixes.ToInt64())
}

// Test that the counter returns the proper utilization for the shared network.
func TestCounterAddSharedNetworkSubnets(t *testing.T) {
	// Arrange
	subnets := []*dbmodel.Subnet{
		{
			SharedNetworkID: 1,
			Prefix:          "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: dbmodel.Stats{
						"total-nas":    uint64(100),
						"assigned-nas": uint64(10),
						"declined-nas": uint64(20),
						"total-pds":    uint64(40),
						"assigned-pds": uint64(30),
					},
				},
			},
		},
		{
			SharedNetworkID: 1,
			Prefix:          "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: dbmodel.Stats{
						"total-nas":    uint64(200),
						"assigned-nas": uint64(40),
						"declined-nas": uint64(50),
						"total-pds":    uint64(80),
						"assigned-pds": uint64(70),
					},
				},
			},
		},
		{
			SharedNetworkID: 1,
			Prefix:          "192.0.2.0/24",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: dbmodel.Stats{
						"total-addresses":    uint64(300),
						"assigned-addresses": uint64(90),
						"declined-addresses": uint64(100),
					},
				},
			},
		},
	}

	// Act
	counter := newStatisticsCounter()
	for _, subnet := range subnets {
		_ = counter.add(subnet)
	}

	// Assert
	require.Len(t, counter.sharedNetworks, 1)
	statistics := counter.sharedNetworks[1]
	require.InDelta(t, float64(140.0/600.0), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(100.0/120.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))
}

// Test that the counter separates the shared networks during the calculations.
func TestCounterAddMultipleSharedNetworkSubnets(t *testing.T) {
	// Arrange
	subnets := []*dbmodel.Subnet{
		{
			SharedNetworkID: 13,
			Prefix:          "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: dbmodel.Stats{
						"total-nas":    uint64(100),
						"assigned-nas": uint64(10),
						"declined-nas": uint64(20),
						"total-pds":    uint64(40),
						"assigned-pds": uint64(30),
					},
				},
			},
		},
		{
			SharedNetworkID: 4,
			Prefix:          "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: dbmodel.Stats{
						"total-nas":    uint64(200),
						"assigned-nas": uint64(40),
						"declined-nas": uint64(50),
						"total-pds":    uint64(80),
						"assigned-pds": uint64(70),
					},
				},
			},
		},
	}

	// Act
	counter := newStatisticsCounter()
	for _, subnet := range subnets {
		_ = counter.add(subnet)
	}

	// Assert
	require.Len(t, counter.sharedNetworks, 2)
	statistics := counter.sharedNetworks[13]
	require.InDelta(t, float64(10.0/100.0), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(30.0/40.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))
	statistics = counter.sharedNetworks[4]
	require.InDelta(t, float64(40.0/200.0), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(70.0/80.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))
}

// Test that the counter works for a subnet without the local subnets.
func TestCounterAddEmptySubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 42,
		Prefix:          "20::/64",
		LocalSubnets:    []*dbmodel.LocalSubnet{},
	}

	// Act
	counter := newStatisticsCounter()
	statistics := counter.add(subnet)

	// Assert
	require.InDelta(t, float64(0.0), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))
	statistics = counter.sharedNetworks[42]
	require.InDelta(t, float64(0.0), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))
}

// Test that the counter add extra IPv4 and IPv6 addresses, and delegated prefixes.
func TestCounterRealKeaResponse(t *testing.T) {
	// Arrange
	statisticGetAll4ResponseRaw := `[{
		"result": 0,
		"arguments": {
			"subnet[10].total-addresses": [[256, "2025-04-22 17:59:15.328731"]],
			"subnet[10].cumulative-assigned-addresses": [[200, "2025-04-22 17:59:15.328731"]],
			"subnet[10].assigned-addresses": [[111, "2025-04-22 17:59:15.328731"]],
			"subnet[10].declined-addresses": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[20].total-addresses": [[4098, "2025-04-22 17:59:15.328731"]],
			"subnet[20].cumulative-assigned-addresses": [[5000, "2025-04-22 17:59:15.328731"]],
			"subnet[20].assigned-addresses": [[2038, "2025-04-22 17:59:15.328731"]],
			"subnet[20].declined-addresses": [[4, "2025-04-22 17:59:15.328731"]]
		}
	}]`

	statisticGetAll6ResponseRaw := `[{
		"result": 0,
		"arguments": {
			"subnet[30].total-nas": [[4096, "2025-04-22 17:59:15.328731"]],
			"subnet[30].cumulative-assigned-nas": [[3000, "2025-04-22 17:59:15.328731"]],
			"subnet[30].assigned-nas": [[2400, "2025-04-22 17:59:15.328731"]],
			"subnet[30].declined-addresses": [[3, "2025-04-22 17:59:15.328731"]],
			"subnet[30].total-pds": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[30].cumulative-assigned-pds": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[30].assigned-pds": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[40].total-nas": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[40].cumulative-assigned-nas": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[40].assigned-nas": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[40].declined-addresses": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[40].total-pds": [[500, "2025-04-22 17:59:15.328731"]],
			"subnet[40].cumulative-assigned-pds": [[233, "2025-04-22 17:59:15.328731"]],
			"subnet[40].assigned-pds": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[50].total-nas": [[256, "2025-04-22 17:59:15.328731"]],
			"subnet[50].cumulative-assigned-nas": [[300, "2025-04-22 17:59:15.328731"]],
			"subnet[50].assigned-nas": [[60, "2025-04-22 17:59:15.328731"]],
			"subnet[50].declined-addresses": [[0, "2025-04-22 17:59:15.328731"]],
			"subnet[50].total-pds": [[1048, "2025-04-22 17:59:15.328731"]],
			"subnet[50].cumulative-assigned-pds": [[15, "2025-04-22 17:59:15.328731"]],
			"subnet[50].assigned-pds": [[15, "2025-04-22 17:59:15.328731"]]
		}
	}]`

	statisticGetAll6MaxResponseRaw := `[{
		"result": 0,
		"arguments": {
			"subnet[60].total-nas": [[-1, "2018-05-04 15:03:37.000000"]],
			"subnet[60].cumulative-assigned-nas": [[-1, "2018-05-04 15:03:37.000000"]],
			"subnet[60].assigned-nas": [[9223372036854775807, "2018-05-04 15:03:37.000000"]],
			"subnet[60].declined-addresses": [[0, "2018-05-04 15:03:37.000000"]],
			"subnet[60].total-pds": [[-1, "2018-05-04 15:03:37.000000"]],
			"subnet[60].cumulative-assigned-pds": [[-1, "2018-05-04 15:03:37.000000"]],
			"subnet[60].assigned-pds": [[-1, "2018-05-04 15:03:37.000000"]]
		}
	}]`

	statResponses := []string{
		statisticGetAll4ResponseRaw,
		statisticGetAll6ResponseRaw,
		statisticGetAll6MaxResponseRaw,
	}

	subnets := make([]*dbmodel.Subnet, 0)

	for subnetIdx, statResponseRaw := range statResponses {
		var statResponse keactrl.StatisticGetAllResponse
		_ = json.Unmarshal([]byte(statResponseRaw), &statResponse)

		prefix := "10.0.0.0/24"
		if subnetIdx != 0 {
			prefix = "88::"
		}

		require.Len(t, statResponse, 1)
		statResponseItems := statResponse[0]
		require.NotNil(t, statResponseItems.Arguments)

		localSubnets := make([]*dbmodel.LocalSubnet, 0)

		statSamplesBySubnet := make(map[int64][]*keactrl.StatisticGetAllResponseSample)
		for _, statSample := range statResponseItems.Arguments {
			statSamplesBySubnet[statSample.SubnetID] = append(
				statSamplesBySubnet[statSample.SubnetID],
				statSample,
			)
		}

		for localSubnetID, statSamples := range statSamplesBySubnet {
			stats := dbmodel.Stats{}
			for _, statSample := range statSamples {
				stats.SetBigCounter(
					statSample.Name,
					storkutil.NewBigCounterFromBigInt(statSample.Value),
				)
			}
			sn := &dbmodel.LocalSubnet{
				Stats: stats,
				ID:    localSubnetID,
			}
			localSubnets = append(localSubnets, sn)
		}

		subnet := &dbmodel.Subnet{
			ID:              int64(subnetIdx),
			SharedNetworkID: 0,
			Prefix:          prefix,
			LocalSubnets:    localSubnets,
		}

		subnets = append(subnets, subnet)
	}

	counter := newStatisticsCounter()

	for _, subnet := range subnets {
		// Act
		statistics := counter.add(subnet)

		// Assert
		switch subnet.ID {
		case 0:
			require.InDelta(t, float64((111.0+2034.0)/(256.0+4098.0)), statistics.GetAddressUtilization(), float64(0.001))
		case 1:
			require.InDelta(t, float64((2400.0+60.0)/(4096.0+256.0)), statistics.GetAddressUtilization(), float64(0.001))
			require.InDelta(t, float64((15.0)/(500.0+1048.0)), statistics.GetDelegatedPrefixUtilization(), float64(0.001))

			require.EqualValues(t, int64(4096+256), counter.global.totalIPv6Addresses.ToInt64())
		case 2:
			expected := big.NewInt(4096 + 256)
			expected = expected.Add(expected, big.NewInt(0).SetUint64(math.MaxUint64))
			require.EqualValues(t, expected, counter.global.totalIPv6Addresses.ToBigInt())
			require.InDelta(t, float64(0.5), statistics.GetAddressUtilization(), float64(0.001))
		}
	}
}

// Test that the negative statistic value is ignored.
func TestCounterAddIgnoreNegativeNumbers(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 13,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-nas":    big.NewInt(-1),
					"assigned-nas": big.NewInt(math.MinInt64),
					"declined-nas": big.NewInt(0).Mul(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(-1)),
					"total-pds":    big.NewInt(-2),
					"assigned-pds": big.NewInt(-3),
				},
			},
		},
	}
	// Act
	counter := newStatisticsCounter()
	statistics := counter.add(subnet)

	// Assert
	require.Zero(t, statistics.GetAddressUtilization())
	require.Zero(t, statistics.GetDelegatedPrefixUtilization())
	require.Zero(t, counter.global.totalIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDelegatedPrefixes.ToInt64())
	require.Zero(t, counter.global.totalAssignedDelegatedPrefixes.ToInt64())
}

// Checks if the out-of-pool values are added to the total counters.
func TestCounterAddExtraToTotalCounters(t *testing.T) {
	// Arrange
	subnets := []dbmodel.Subnet{
		{
			ID:     1,
			Prefix: "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: dbmodel.Stats{
						"total-nas":    uint64(90),
						"assigned-nas": uint64(50),
						"declined-nas": uint64(40),
						"total-pds":    uint64(60),
						"assigned-pds": uint64(9),
					},
				},
			},
			SharedNetworkID: 42,
		},
		{
			ID:     2,
			Prefix: "10.0.0.0/16",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: dbmodel.Stats{
						"total-addresses":    uint64(60),
						"assigned-addresses": uint64(20),
						"declined-addresses": uint64(30),
					},
				},
			},
			SharedNetworkID: 42,
		},
	}

	outOfPoolAddresses := map[int64]uint64{
		1: 10,
		2: 20,
	}

	outOfPoolPrefixes := map[int64]uint64{
		1: 30,
		// Bug - IPv4 has no prefixes, but the counter should keep working correctly.
		2: 40,
	}

	// Act
	counter := newStatisticsCounter()
	counter.setOutOfPoolShifts(outOfPoolShifts{
		outOfPoolAddresses: outOfPoolAddresses,
		outOfPoolPrefixes:  outOfPoolPrefixes,
	})

	utilization1 := counter.add(&subnets[0])
	utilization2 := counter.add(&subnets[1])

	// Assert
	value, _ := counter.global.totalIPv4Addresses.ToUint64()
	require.EqualValues(t, uint64(80), value)
	value, _ = counter.global.totalAssignedIPv4Addresses.ToUint64()
	require.EqualValues(t, uint64(20), value)
	value, _ = counter.global.totalDeclinedIPv4Addresses.ToUint64()
	require.EqualValues(t, uint64(30), value)
	value, _ = counter.global.totalIPv6Addresses.ToUint64()
	require.EqualValues(t, uint64(100), value)
	value, _ = counter.global.totalAssignedIPv6Addresses.ToUint64()
	require.EqualValues(t, uint64(50), value)
	value, _ = counter.global.totalDeclinedIPv6Addresses.ToUint64()
	require.EqualValues(t, uint64(40), value)
	value, _ = counter.global.totalDelegatedPrefixes.ToUint64()
	require.EqualValues(t, uint64(90), value)
	value, _ = counter.global.totalAssignedDelegatedPrefixes.ToUint64()
	require.EqualValues(t, uint64(9), value)
	require.Len(t, counter.sharedNetworks, 1)

	require.EqualValues(t, 0.5, utilization1.GetAddressUtilization())
	require.EqualValues(t, 0.1, utilization1.GetDelegatedPrefixUtilization())
	require.EqualValues(t, 0.25, utilization2.GetAddressUtilization())
	require.EqualValues(t, 0.0, utilization2.GetDelegatedPrefixUtilization())

	sharedNetwork := counter.sharedNetworks[42]
	value, _ = sharedNetwork.totalAddresses.ToUint64()
	require.EqualValues(t, 180, value)
	value, _ = sharedNetwork.totalAssignedAddresses.ToUint64()
	require.EqualValues(t, 70, value)
	value, _ = sharedNetwork.totalAssignedDelegatedPrefixes.ToUint64()
	require.EqualValues(t, 9, value)
	value, _ = sharedNetwork.totalDelegatedPrefixes.ToUint64()
	require.EqualValues(t, 90, value)

	require.InDelta(t, 7.0/18.0, sharedNetwork.GetAddressUtilization(), 0.001)
	require.EqualValues(t, 0.1, sharedNetwork.GetDelegatedPrefixUtilization())
}

// Checks if the excluded daemons are respected for IPv4 subnets.
func TestCounterSkipExcludedDaemonsIPv4(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "192.0.2.0/24",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(100),
					"assigned-addresses": uint64(10),
					"declined-addresses": uint64(20),
				},
				DaemonID: 1,
			},
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(200),
					"assigned-addresses": uint64(20),
					"declined-addresses": uint64(40),
				},
				DaemonID: 1,
			},
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(5),
					"assigned-addresses": uint64(3),
					"declined-addresses": uint64(1),
				},
				DaemonID: 2,
			},
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(50),
					"assigned-addresses": uint64(1),
					"declined-addresses": uint64(2),
				},
				DaemonID: 3,
			},
		},
	}

	counter := newStatisticsCounter()
	counter.setExcludedDaemons([]int64{2, 3})

	// Act
	statistics := counter.add(subnet)

	// Assert
	require.InDelta(t, float64(0.1), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))

	require.EqualValues(t, 300, counter.global.totalIPv4Addresses.ToInt64())
	require.EqualValues(t, 30, counter.global.totalAssignedIPv4Addresses.ToInt64())
	require.EqualValues(t, 60, counter.global.totalDeclinedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv6Addresses.ToInt64())
	require.Zero(t, counter.global.totalDelegatedPrefixes.ToInt64())
	require.Zero(t, counter.global.totalAssignedDelegatedPrefixes.ToInt64())
}

// Checks if the excluded daemons are respected for IPv6 subnets.
func TestCounterSkipExcludedDaemonsIPv6(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(100),
					"assigned-nas": uint64(10),
					"declined-nas": uint64(20),
					"total-pds":    uint64(40),
					"assigned-pds": uint64(30),
				},
				DaemonID: 1,
			},
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(200),
					"assigned-nas": uint64(20),
					"declined-nas": uint64(40),
					"total-pds":    uint64(100),
					"assigned-pds": uint64(10),
				},
				DaemonID: 2,
			},
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(5),
					"assigned-nas": uint64(3),
					"declined-nas": uint64(1),
					"total-pds":    uint64(3),
					"assigned-pds": uint64(1),
				},
				DaemonID: 3,
			},
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(50),
					"assigned-nas": uint64(1),
					"declined-nas": uint64(2),
					"total-pds":    uint64(100),
					"assigned-pds": uint64(3),
				},
				DaemonID: 4,
			},
		},
	}

	counter := newStatisticsCounter()
	counter.setExcludedDaemons([]int64{3, 4})

	// Act
	statistics := counter.add(subnet)

	// Assert
	require.InDelta(t, float64(0.1), statistics.GetAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(40.0/140.0), statistics.GetDelegatedPrefixUtilization(), float64(0.001))

	require.Zero(t, counter.global.totalIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalAssignedIPv4Addresses.ToInt64())
	require.Zero(t, counter.global.totalDeclinedIPv4Addresses.ToInt64())
	require.EqualValues(t, 300, counter.global.totalIPv6Addresses.ToInt64())
	require.EqualValues(t, 30, counter.global.totalAssignedIPv6Addresses.ToInt64())
	require.EqualValues(t, 60, counter.global.totalDeclinedIPv6Addresses.ToInt64())
	require.EqualValues(t, 140, counter.global.totalDelegatedPrefixes.ToInt64())
	require.EqualValues(t, 40, counter.global.totalAssignedDelegatedPrefixes.ToInt64())
}

// Checks if the subnet statistics contain proper values for IPv4 subnet.
func TestCounterGetStatisticsForIPv4Subnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "10.0.0.0/16",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(100),
					"assigned-addresses": uint64(10),
					"declined-addresses": uint64(20),
				},
			},
			{
				Stats: dbmodel.Stats{
					"total-addresses":    uint64(200),
					"assigned-addresses": uint64(20),
					"declined-addresses": uint64(40),
				},
			},
		},
	}

	counter := newStatisticsCounter()
	sn := counter.add(subnet)

	// Act
	stats := sn.GetStatistics()

	// Assert
	require.EqualValues(t, 300, stats["total-addresses"])
	require.EqualValues(t, 30, stats["assigned-addresses"])
	require.EqualValues(t, 60, stats["declined-addresses"])
}

// Checks if the subnet statistics contain proper values for IPv6 subnet.
func TestCounterGetStatisticsForIPv6Subnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(100),
					"assigned-nas": uint64(10),
					"declined-nas": uint64(20),
					"total-pds":    uint64(40),
					"assigned-pds": uint64(30),
				},
			},
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(200),
					"assigned-nas": uint64(20),
					"declined-nas": uint64(40),
					"total-pds":    uint64(100),
					"assigned-pds": uint64(10),
				},
			},
		},
	}

	counter := newStatisticsCounter()
	sn := counter.add(subnet)

	// Act
	stats := sn.GetStatistics()

	// Assert
	require.EqualValues(t, 300, stats["total-nas"])
	require.EqualValues(t, 30, stats["assigned-nas"])
	require.EqualValues(t, 60, stats["declined-nas"])
	require.EqualValues(t, 140, stats["total-pds"])
	require.EqualValues(t, 40, stats["assigned-pds"])
}

// Checks if the subnet statistics contain proper values for a shared network.
func TestCounterGetStatisticsForSharedNetwork(t *testing.T) {
	// Arrange
	subnet1 := &dbmodel.Subnet{
		SharedNetworkID: 1,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(100),
					"assigned-nas": uint64(10),
					"declined-nas": uint64(20),
					"total-pds":    uint64(40),
					"assigned-pds": uint64(30),
				},
			},
		},
	}

	subnet2 := &dbmodel.Subnet{
		SharedNetworkID: 1,
		Prefix:          "30::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.Stats{
					"total-nas":    uint64(200),
					"assigned-nas": uint64(20),
					"declined-nas": uint64(40),
					"total-pds":    uint64(100),
					"assigned-pds": uint64(10),
				},
			},
		},
	}

	counter := newStatisticsCounter()
	_ = counter.add(subnet1)
	_ = counter.add(subnet2)
	sn := counter.sharedNetworks[1]

	// Act
	stats := sn.GetStatistics()

	// Assert
	require.EqualValues(t, 300, stats["total-nas"])
	require.EqualValues(t, 30, stats["assigned-nas"])
	require.EqualValues(t, 140, stats["total-pds"])
	require.EqualValues(t, 40, stats["assigned-pds"])
}
