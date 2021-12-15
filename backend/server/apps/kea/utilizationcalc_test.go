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
				Stats: map[string]interface{}{
					"total-addresses":    int64(100),
					"assigned-addresses": int64(10),
					"declined-addresses": int64(20),
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
				Stats: map[string]interface{}{
					"total-nas":    int64(100),
					"assigned-nas": int64(40),
					"declined-nas": int64(30),
					"total-pds":    int64(20),
					"assigned-pds": int64(10),
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

// Test that the calculator returns the proper utilization for multiple IPv4 local subnets.
func TestCalculatorAddMultipleIPv4LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "192.0.2.0/24",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: map[string]interface{}{
					"total-addresses":    int64(100),
					"assigned-addresses": int64(10),
					"declined-addresses": int64(20),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-addresses":    int64(200),
					"assigned-addresses": int64(20),
					"declined-addresses": int64(40),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-addresses":    int64(5),
					"assigned-addresses": int64(3),
					"declined-addresses": int64(1),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-addresses":    int64(50),
					"assigned-addresses": int64(1),
					"declined-addresses": int64(2),
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
				Stats: map[string]interface{}{
					"total-nas":    int64(100),
					"assigned-nas": int64(10),
					"declined-nas": int64(20),
					"total-pds":    int64(40),
					"assigned-pds": int64(30),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-nas":    int64(200),
					"assigned-nas": int64(20),
					"declined-nas": int64(40),
					"total-pds":    int64(100),
					"assigned-pds": int64(10),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-nas":    int64(5),
					"assigned-nas": int64(3),
					"declined-nas": int64(1),
					"total-pds":    int64(3),
					"assigned-pds": int64(1),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-nas":    int64(50),
					"assigned-nas": int64(1),
					"declined-nas": int64(2),
					"total-pds":    int64(100),
					"assigned-pds": int64(3),
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
					Stats: map[string]interface{}{
						"total-nas":    int64(100),
						"assigned-nas": int64(10),
						"declined-nas": int64(20),
						"total-pds":    int64(40),
						"assigned-pds": int64(30),
					},
				},
			},
		},
		{
			SharedNetworkID: 1,
			Prefix:          "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: map[string]interface{}{
						"total-nas":    int64(200),
						"assigned-nas": int64(40),
						"declined-nas": int64(50),
						"total-pds":    int64(80),
						"assigned-pds": int64(70),
					},
				},
			},
		},
		{
			SharedNetworkID: 1,
			Prefix:          "192.0.2.0/24",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: map[string]interface{}{
						"total-addresses":    int64(300),
						"assigned-addresses": int64(90),
						"declined-addresses": int64(100),
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
					Stats: map[string]interface{}{
						"total-nas":    int64(100),
						"assigned-nas": int64(10),
						"declined-nas": int64(20),
						"total-pds":    int64(40),
						"assigned-pds": int64(30),
					},
				},
			},
		},
		{
			SharedNetworkID: 4,
			Prefix:          "20::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: map[string]interface{}{
						"total-nas":    int64(200),
						"assigned-nas": int64(40),
						"declined-nas": int64(50),
						"total-pds":    int64(80),
						"assigned-pds": int64(70),
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

// Test that int64 is cast to uint64
func TestCalculatorAddNegativeStatistics(t *testing.T) {
	// Act
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 13,
		Prefix:          "20::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: map[string]interface{}{
					"total-nas":    int64(-1),
					"assigned-nas": int64(-2),
					"declined-nas": int64(-3),
					"total-pds":    int64(-4),
					"assigned-pds": int64(-5),
				},
			},
		},
	}

	// Act
	calculator := newUtilizationCalculator()
	_ = calculator.add(subnet)

	// Assert
	expected := big.NewInt(0).SetUint64(math.MaxUint64)
	one := big.NewInt(1)
	require.EqualValues(t, expected, calculator.global.totalNAs.ToBigInt())
	require.EqualValues(t, expected.Sub(expected, one), calculator.global.totalAssignedNAs.ToBigInt())
	require.EqualValues(t, expected.Sub(expected, one), calculator.global.totalDeclinedNAs.ToBigInt())
	require.EqualValues(t, expected.Sub(expected, one), calculator.global.totalPDs.ToBigInt())
	require.EqualValues(t, expected.Sub(expected, one), calculator.global.totalAssignedPDs.ToBigInt())
}

// Test the calculator using real Kea response
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
				stats[name] = val
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
		if subnet.ID == 0 {
			require.InDelta(t, float64((111.0+2034.0)/(256.0+4098.0)), utilization.getAddressUtilization(), float64(0.001))
		} else if subnet.ID == 1 {
			require.InDelta(t, float64((2400.0+60.0)/(4096.0+256.0)), utilization.getAddressUtilization(), float64(0.001))
			require.InDelta(t, float64((15.0)/(500.0+1048.0)), utilization.getPDUtilization(), float64(0.001))

			require.EqualValues(t, int64(4096+256), calculator.global.totalNAs.ToInt64())
		} else if subnet.ID == 2 {
			expected := big.NewInt(4096 + 256)
			expected = expected.Add(expected, big.NewInt(0).SetUint64(math.MaxUint64))
			require.EqualValues(t, expected, calculator.global.totalNAs.ToBigInt())
			require.InDelta(t, float64(0.5), utilization.getAddressUtilization(), float64(0.001))
		}

	}

}
