package kea

import (
	"encoding/json"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Test that the utilization calculator is properly constructed.
func TestCalculatorConstruction(t *testing.T) {
	// Act
	calculator := newUtilizationCalculator()

	// Assert
	require.EqualValues(t, 0, calculator.global.totalAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalDeclinedAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalDeclinedNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalPDs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedPDs.ToInt64())
	require.Len(t, calculator.sharedNetworks, 0)
}

// Test that the calculator returns utilization for IPv4 subnet with single local subnet.
func TestCalculatorAddSingleIPv4LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "192.0.2.0/24",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-addresses":    uint64(100),
					"assigned-addresses": uint64(10),
					"declined-addresses": uint64(20),
				},
			},
		},
	}

	calculator := newUtilizationCalculator()

	// Act
	utilization := calculator.add(subnet)

	// Assert
	require.InDelta(t, float64(0.1), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), utilization.getPDUtilization(), float64(0.001))

	require.EqualValues(t, 100, calculator.global.totalAddresses.ToInt64())
	require.EqualValues(t, 10, calculator.global.totalAssignedAddresses.ToInt64())
	require.EqualValues(t, 20, calculator.global.totalDeclinedAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalDeclinedNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalPDs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedPDs.ToInt64())

	require.Len(t, calculator.sharedNetworks, 0)
}

// Test that the calculator returns utilization for IPv6 subnet with single local subnet.
func TestCalculatorAddSingleIPv6LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-nas":    uint64(100),
					"assigned-nas": uint64(40),
					"declined-nas": uint64(30),
					"total-pds":    uint64(20),
					"assigned-pds": uint64(10),
				},
			},
		},
	}

	calculator := newUtilizationCalculator()

	// Act
	utilization := calculator.add(subnet)

	// Assert
	require.InDelta(t, float64(0.4), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.5), utilization.getPDUtilization(), float64(0.001))

	require.EqualValues(t, 0, calculator.global.totalAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalDeclinedAddresses.ToInt64())
	require.EqualValues(t, 100, calculator.global.totalNAs.ToInt64())
	require.EqualValues(t, 40, calculator.global.totalAssignedNAs.ToInt64())
	require.EqualValues(t, 30, calculator.global.totalDeclinedNAs.ToInt64())
	require.EqualValues(t, 20, calculator.global.totalPDs.ToInt64())
	require.EqualValues(t, 10, calculator.global.totalAssignedPDs.ToInt64())

	require.Len(t, calculator.sharedNetworks, 0)
}

// Test that the non-uint64 statistics aren't supported.
func TestCalculatorAddSubnetUsingNonUint64(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-nas":    int64(100),
					"assigned-nas": int32(40),
					"declined-nas": int16(30),
					"total-pds":    int(20),
					"assigned-pds": uint32(10),
				},
			},
		},
	}

	calculator := newUtilizationCalculator()

	// Act
	utilization := calculator.add(subnet)

	// Assert
	require.InDelta(t, float64(0.0), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), utilization.getPDUtilization(), float64(0.001))

	require.EqualValues(t, 0, calculator.global.totalAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalDeclinedAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalDeclinedNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalPDs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedPDs.ToInt64())

	require.Len(t, calculator.sharedNetworks, 0)
}

// Test that the calculator returns the proper utilization for multiple IPv4 local subnets.
func TestCalculatorAddMultipleIPv4LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "192.0.2.0/24",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-addresses":    uint64(100),
					"assigned-addresses": uint64(10),
					"declined-addresses": uint64(20),
				},
			},
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-addresses":    uint64(200),
					"assigned-addresses": uint64(20),
					"declined-addresses": uint64(40),
				},
			},
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-addresses":    uint64(5),
					"assigned-addresses": uint64(3),
					"declined-addresses": uint64(1),
				},
			},
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-addresses":    uint64(50),
					"assigned-addresses": uint64(1),
					"declined-addresses": uint64(2),
				},
			},
		},
	}

	calculator := newUtilizationCalculator()

	// Act
	utilization := calculator.add(subnet)

	// Assert
	require.InDelta(t, float64(34.0/355.0), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), utilization.getPDUtilization(), float64(0.001))

	require.EqualValues(t, 355, calculator.global.totalAddresses.ToInt64())
	require.EqualValues(t, 34, calculator.global.totalAssignedAddresses.ToInt64())
	require.EqualValues(t, 63, calculator.global.totalDeclinedAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalDeclinedNAs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalPDs.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedPDs.ToInt64())
}

// Test that the calculator returns the proper utilization for multiple IPv6 local subnets.
func TestCalculatorAddMultipleIPv6LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-nas":    uint64(100),
					"assigned-nas": uint64(10),
					"declined-nas": uint64(20),
					"total-pds":    uint64(40),
					"assigned-pds": uint64(30),
				},
			},
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-nas":    uint64(200),
					"assigned-nas": uint64(20),
					"declined-nas": uint64(40),
					"total-pds":    uint64(100),
					"assigned-pds": uint64(10),
				},
			},
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-nas":    uint64(5),
					"assigned-nas": uint64(3),
					"declined-nas": uint64(1),
					"total-pds":    uint64(3),
					"assigned-pds": uint64(1),
				},
			},
			{
				Stats: dbmodel.LocalSubnetStats{
					"total-nas":    uint64(50),
					"assigned-nas": uint64(1),
					"declined-nas": uint64(2),
					"total-pds":    uint64(100),
					"assigned-pds": uint64(3),
				},
			},
		},
	}

	calculator := newUtilizationCalculator()

	// Act
	utilization := calculator.add(subnet)

	// Assert
	require.InDelta(t, float64(34.0/355.0), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(44.0/243.0), utilization.getPDUtilization(), float64(0.001))

	require.EqualValues(t, 0, calculator.global.totalAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalAssignedAddresses.ToInt64())
	require.EqualValues(t, 0, calculator.global.totalDeclinedAddresses.ToInt64())
	require.EqualValues(t, 355, calculator.global.totalNAs.ToInt64())
	require.EqualValues(t, 34, calculator.global.totalAssignedNAs.ToInt64())
	require.EqualValues(t, 63, calculator.global.totalDeclinedNAs.ToInt64())
	require.EqualValues(t, 243, calculator.global.totalPDs.ToInt64())
	require.EqualValues(t, 44, calculator.global.totalAssignedPDs.ToInt64())
}

// Test that the calculator returns the proper utilization for the shared network.
func TestCalculatorAddSharedNetworkSubnets(t *testing.T) {
	// Arrange
	subnets := []*dbmodel.Subnet{
		{
			SharedNetworkID: 1,
			Prefix:          "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: dbmodel.LocalSubnetStats{
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
					Stats: dbmodel.LocalSubnetStats{
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
					Stats: dbmodel.LocalSubnetStats{
						"total-addresses":    uint64(300),
						"assigned-addresses": uint64(90),
						"declined-addresses": uint64(100),
					},
				},
			},
		},
	}

	// Act
	calculator := newUtilizationCalculator()
	for _, subnet := range subnets {
		_ = calculator.add(subnet)
	}

	// Assert
	require.Len(t, calculator.sharedNetworks, 1)
	utilization := calculator.sharedNetworks[1]
	require.InDelta(t, float64(140.0/600.0), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(100.0/120.0), utilization.getPDUtilization(), float64(0.001))
}

// Test that the calculator separates the shared networks during the calculations.
func TestCalculatorAddMultipleSharedNetworkSubnets(t *testing.T) {
	// Arrange
	subnets := []*dbmodel.Subnet{
		{
			SharedNetworkID: 13,
			Prefix:          "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: dbmodel.LocalSubnetStats{
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
					Stats: dbmodel.LocalSubnetStats{
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
	calculator := newUtilizationCalculator()
	for _, subnet := range subnets {
		_ = calculator.add(subnet)
	}

	// Assert
	require.Len(t, calculator.sharedNetworks, 2)
	utilization := calculator.sharedNetworks[13]
	require.InDelta(t, float64(10.0/100.0), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(30.0/40.0), utilization.getPDUtilization(), float64(0.001))
	utilization = calculator.sharedNetworks[4]
	require.InDelta(t, float64(40.0/200.0), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(70.0/80.0), utilization.getPDUtilization(), float64(0.001))
}

// Test that the calculator works for a subnet without the local subnets.
func TestCalculatorAddEmptySubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 42,
		Prefix:          "20::/64",
		LocalSubnets:    []*dbmodel.LocalSubnet{},
	}

	// Act
	calculator := newUtilizationCalculator()
	utilization := calculator.add(subnet)

	// Assert
	require.InDelta(t, float64(0.0), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), utilization.getPDUtilization(), float64(0.001))
	utilization = calculator.sharedNetworks[42]
	require.InDelta(t, float64(0.0), utilization.getAddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), utilization.getPDUtilization(), float64(0.001))
}

// Test the calculator using real Kea response.
func TestCalculatorRealKeaResponse(t *testing.T) {
	// Arrange
	statLease4GetResponseRaw := `{
		"result": 0,
		"text": "stat-lease4-get: 2 rows found",
		"arguments": {
			"result-set": {
				"columns": [ "subnet-id",
							"total-addresses",
							"cumulative-assigned-addresses",
							"assigned-addresses",
							"declined-addresses" ],
				"rows": [
					[ 10, 256, 200, 111, 0 ],
					[ 20, 4098, 5000, 2034, 4 ]
				],
				"timestamp": "2018-05-04 15:03:37.000000"
			}
		}
	}`

	statLease6GetResponseRaw := `{
		"result": 0,
		"text": "stat-lease6-get: 2 rows found",
		"arguments": {
			"result-set": {
				"columns": [ "subnet-id", "total-nas", "cumulative-assigned-nas", "assigned-nas", "declined-nas", "total-pds",  "cumulative-assigned-pds", "assigned-pds" ],
				"rows": [
					[ 30, 4096, 3000, 2400, 3, 0, 0],
					[ 40, 0, 0, 0, 1048, 500, 233 ],
					[ 50, 256, 300, 60, 0, 1048, 15, 15 ]
				],
				"timestamp": "2018-05-04 15:03:37.000000"
			}
		}
	}`

	statLease6GetResponseMaxRaw := `{
		"result": 0,
		"text": "stat-lease6-get: 2 rows found",
		"arguments": {
			"result-set": {
				"columns": [ "subnet-id", "total-nas", "cumulative-assigned-nas", "assigned-nas", "declined-nas", "total-pds",  "cumulative-assigned-pds", "assigned-pds" ],
				"rows": [
					[ 60, -1, -1, 9223372036854775807, 0, -1, -1, -1]
				],
				"timestamp": "2018-05-04 15:03:37.000000"
			}
		}
	}`

	statResponses := []string{
		statLease4GetResponseRaw,
		statLease6GetResponseRaw,
		statLease6GetResponseMaxRaw,
	}

	subnets := make([]*dbmodel.Subnet, 0)

	for subnetIdx, statResponseRaw := range statResponses {
		var statResponse StatLeaseGetResponse
		_ = json.Unmarshal([]byte(statResponseRaw), &statResponse)

		prefix := "10.0.0.0/24"
		if strings.HasPrefix(statResponse.Text, "stat-lease6-get") {
			prefix = "88::"
		}

		localSubnets := make([]*dbmodel.LocalSubnet, 0)
		resultSet := statResponse.Arguments.ResultSet
		for _, row := range resultSet.Rows {
			stats := make(map[string]interface{})
			for colIdx, val := range row {
				name := resultSet.Columns[colIdx]
				stats[name] = uint64(val)
			}
			sn := &dbmodel.LocalSubnet{
				Stats: stats,
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

	calculator := newUtilizationCalculator()

	for _, subnet := range subnets {
		// Act
		utilization := calculator.add(subnet)

		// Assert
		switch subnet.ID {
		case 0:
			require.InDelta(t, float64((111.0+2034.0)/(256.0+4098.0)), utilization.getAddressUtilization(), float64(0.001))
		case 1:
			require.InDelta(t, float64((2400.0+60.0)/(4096.0+256.0)), utilization.getAddressUtilization(), float64(0.001))
			require.InDelta(t, float64((15.0)/(500.0+1048.0)), utilization.getPDUtilization(), float64(0.001))

			require.EqualValues(t, int64(4096+256), calculator.global.totalNAs.ToInt64())
		case 2:
			expected := big.NewInt(4096 + 256)
			expected = expected.Add(expected, big.NewInt(0).SetUint64(math.MaxUint64))
			require.EqualValues(t, expected, calculator.global.totalNAs.ToBigInt())
			require.InDelta(t, float64(0.5), utilization.getAddressUtilization(), float64(0.001))
		}
	}
}

// Test that the negative statistic value is ignored.
func TestCalculatorAddIgnoreNegativeNumbers(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 13,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: dbmodel.LocalSubnetStats{
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
	calculator := newUtilizationCalculator()
	utilization := calculator.add(subnet)

	// Assert
	require.Zero(t, utilization.getAddressUtilization())
	require.Zero(t, utilization.getPDUtilization())
	require.Zero(t, calculator.global.totalAddresses.ToInt64())
	require.Zero(t, calculator.global.totalAssignedAddresses.ToInt64())
	require.Zero(t, calculator.global.totalDeclinedAddresses.ToInt64())
	require.Zero(t, calculator.global.totalNAs.ToInt64())
	require.Zero(t, calculator.global.totalAssignedNAs.ToInt64())
	require.Zero(t, calculator.global.totalDeclinedNAs.ToInt64())
	require.Zero(t, calculator.global.totalPDs.ToInt64())
	require.Zero(t, calculator.global.totalAssignedPDs.ToInt64())
}

// Test that the calculator add extra addresses, NAs and prefixes.
func TestCalculatorAddExtraToTotalCounters(t *testing.T) {
	// Arrange
	subnets := []dbmodel.Subnet{
		{
			ID:     1,
			Prefix: "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: map[string]interface{}{
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
					Stats: map[string]interface{}{
						"total-addresses":    uint64(60),
						"assigned-addresses": uint64(20),
						"declined-addresses": uint64(30),
					},
				},
			},
			SharedNetworkID: 42,
		},
	}

	extraAddresses := map[int64]uint64{
		1: 10,
		2: 20,
	}

	extraPrefixes := map[int64]uint64{
		1: 30,
		// Bug - IPv4 has no prefixes, but the calculator should keep working correctly.
		2: 40,
	}

	// Act
	calculator := newUtilizationCalculator()
	calculator.setExtraTotalAddresses(extraAddresses)
	calculator.setExtraTotalPrefixes(extraPrefixes)

	utilization1 := calculator.add(&subnets[0])
	utilization2 := calculator.add(&subnets[1])

	// Assert
	require.EqualValues(t, uint64(80), calculator.global.totalAddresses.ToUint64())
	require.EqualValues(t, uint64(20), calculator.global.totalAssignedAddresses.ToUint64())
	require.EqualValues(t, uint64(30), calculator.global.totalDeclinedAddresses.ToUint64())
	require.EqualValues(t, uint64(100), calculator.global.totalNAs.ToUint64())
	require.EqualValues(t, uint64(50), calculator.global.totalAssignedNAs.ToUint64())
	require.EqualValues(t, uint64(40), calculator.global.totalDeclinedNAs.ToUint64())
	require.EqualValues(t, uint64(90), calculator.global.totalPDs.ToUint64())
	require.EqualValues(t, uint64(9), calculator.global.totalAssignedPDs.ToUint64())
	require.Len(t, calculator.sharedNetworks, 1)

	require.EqualValues(t, 0.5, utilization1.getAddressUtilization())
	require.EqualValues(t, 0.1, utilization1.getPDUtilization())
	require.EqualValues(t, 0.25, utilization2.getAddressUtilization())
	require.EqualValues(t, 0.0, utilization2.getPDUtilization())
}
