package kea_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/apps/kea"
	dbmodel "isc.org/stork/server/database/model"
)

// Test that the utilization calculator is properly constructed.
func TestCalculatorConstruction(t *testing.T) {
	// Act
	calculator := kea.NewUtilizationCalculator()

	// Assert
	require.EqualValues(t, 0, calculator.Global.TotalAddresses)
	require.EqualValues(t, 0, calculator.Global.TotalAssignedAddresses)
	require.EqualValues(t, 0, calculator.Global.TotalDeclinedAddresses)
	require.EqualValues(t, 0, calculator.Global.TotalNas)
	require.EqualValues(t, 0, calculator.Global.TotalAssignedNas)
	require.EqualValues(t, 0, calculator.Global.TotalDeclinedNas)
	require.EqualValues(t, 0, calculator.Global.TotalPds)
	require.EqualValues(t, 0, calculator.Global.TotalAssignedPds)
	require.Len(t, calculator.SharedNetworks, 0)
}

// Test that the calculator return utilization for IPv4 subnet with single local network.
func TestCalculatorAddSingleIPv4LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "127.0.0.1",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: map[string]interface{}{
					"total-addresses":    float64(100),
					"assigned-addresses": float64(10),
					"declined-addresses": float64(20),
				},
			},
		},
	}

	calculator := kea.NewUtilizationCalculator()

	// Act
	utilization := calculator.Add(subnet)

	// Assert
	require.InDelta(t, float64(0.1), utilization.AddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), utilization.PdUtilization(), float64(0.001))

	require.EqualValues(t, 100, calculator.Global.TotalAddresses)
	require.EqualValues(t, 10, calculator.Global.TotalAssignedAddresses)
	require.EqualValues(t, 20, calculator.Global.TotalDeclinedAddresses)
	require.EqualValues(t, 0, calculator.Global.TotalNas)
	require.EqualValues(t, 0, calculator.Global.TotalAssignedNas)
	require.EqualValues(t, 0, calculator.Global.TotalDeclinedNas)
	require.EqualValues(t, 0, calculator.Global.TotalPds)
	require.EqualValues(t, 0, calculator.Global.TotalAssignedPds)

	require.Len(t, calculator.SharedNetworks, 0)
}

// Test that the calculator return utilization for IPv6 subnet with single local network.
func TestCalculatorAddSingleIPv6LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: map[string]interface{}{
					"total-nas":    float64(100),
					"assigned-nas": float64(40),
					"declined-nas": float64(30),
					"total-pds":    float64(20),
					"assigned-pds": float64(10),
				},
			},
		},
	}

	calculator := kea.NewUtilizationCalculator()

	// Act
	utilization := calculator.Add(subnet)

	// Assert
	require.InDelta(t, float64(0.4), utilization.AddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.5), utilization.PdUtilization(), float64(0.001))

	require.EqualValues(t, 0, calculator.Global.TotalAddresses)
	require.EqualValues(t, 0, calculator.Global.TotalAssignedAddresses)
	require.EqualValues(t, 0, calculator.Global.TotalDeclinedAddresses)
	require.EqualValues(t, 100, calculator.Global.TotalNas)
	require.EqualValues(t, 40, calculator.Global.TotalAssignedNas)
	require.EqualValues(t, 30, calculator.Global.TotalDeclinedNas)
	require.EqualValues(t, 20, calculator.Global.TotalPds)
	require.EqualValues(t, 10, calculator.Global.TotalAssignedPds)

	require.Len(t, calculator.SharedNetworks, 0)
}

// Test that the calculator returns the proper utilization for multiple IPv4 local subnets.
func TestCalculatorAddMultipleIPv4LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "127.0.0.1",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: map[string]interface{}{
					"total-addresses":    float64(100),
					"assigned-addresses": float64(10),
					"declined-addresses": float64(20),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-addresses":    float64(200),
					"assigned-addresses": float64(20),
					"declined-addresses": float64(40),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-addresses":    float64(5),
					"assigned-addresses": float64(3),
					"declined-addresses": float64(1),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-addresses":    float64(50),
					"assigned-addresses": float64(1),
					"declined-addresses": float64(2),
				},
			},
		},
	}

	calculator := kea.NewUtilizationCalculator()

	// Act
	utilization := calculator.Add(subnet)

	// Assert
	require.InDelta(t, float64(34.0/355.0), utilization.AddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), utilization.PdUtilization(), float64(0.001))

	require.EqualValues(t, 355, calculator.Global.TotalAddresses)
	require.EqualValues(t, 34, calculator.Global.TotalAssignedAddresses)
	require.EqualValues(t, 63, calculator.Global.TotalDeclinedAddresses)
	require.EqualValues(t, 0, calculator.Global.TotalNas)
	require.EqualValues(t, 0, calculator.Global.TotalAssignedNas)
	require.EqualValues(t, 0, calculator.Global.TotalDeclinedNas)
	require.EqualValues(t, 0, calculator.Global.TotalPds)
	require.EqualValues(t, 0, calculator.Global.TotalAssignedPds)
}

// Test that the calculator returns the proper utilization for multiple IPv6 local subnets.
func TestCalculatorAddMultipleIPv6LocalSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 0,
		Prefix:          "20::",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				Stats: map[string]interface{}{
					"total-nas":    float64(100),
					"assigned-nas": float64(10),
					"declined-nas": float64(20),
					"total-pds":    float64(40),
					"assigned-pds": float64(30),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-nas":    float64(200),
					"assigned-nas": float64(20),
					"declined-nas": float64(40),
					"total-pds":    float64(100),
					"assigned-pds": float64(10),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-nas":    float64(5),
					"assigned-nas": float64(3),
					"declined-nas": float64(1),
					"total-pds":    float64(3),
					"assigned-pds": float64(1),
				},
			},
			{
				Stats: map[string]interface{}{
					"total-nas":    float64(50),
					"assigned-nas": float64(1),
					"declined-nas": float64(2),
					"total-pds":    float64(100),
					"assigned-pds": float64(3),
				},
			},
		},
	}

	calculator := kea.NewUtilizationCalculator()

	// Act
	utilization := calculator.Add(subnet)

	// Assert
	require.InDelta(t, float64(34.0/355.0), utilization.AddressUtilization(), float64(0.001))
	require.InDelta(t, float64(44.0/243.0), utilization.PdUtilization(), float64(0.001))

	require.EqualValues(t, 0, calculator.Global.TotalAddresses)
	require.EqualValues(t, 0, calculator.Global.TotalAssignedAddresses)
	require.EqualValues(t, 0, calculator.Global.TotalDeclinedAddresses)
	require.EqualValues(t, 355, calculator.Global.TotalNas)
	require.EqualValues(t, 34, calculator.Global.TotalAssignedNas)
	require.EqualValues(t, 63, calculator.Global.TotalDeclinedNas)
	require.EqualValues(t, 243, calculator.Global.TotalPds)
	require.EqualValues(t, 44, calculator.Global.TotalAssignedPds)
}

// Test that the calculator returns the proper utilization for the shared network.
func TestCalculatorAddSharedNetworkSubnets(t *testing.T) {
	// Arrange
	subnets := []*dbmodel.Subnet{
		{
			SharedNetworkID: 1,
			Prefix:          "20::",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: map[string]interface{}{
						"total-nas":    float64(100),
						"assigned-nas": float64(10),
						"declined-nas": float64(20),
						"total-pds":    float64(40),
						"assigned-pds": float64(30),
					},
				},
			},
		},
		{
			SharedNetworkID: 1,
			Prefix:          "20::",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: map[string]interface{}{
						"total-nas":    float64(200),
						"assigned-nas": float64(40),
						"declined-nas": float64(50),
						"total-pds":    float64(80),
						"assigned-pds": float64(70),
					},
				},
			},
		},
		{
			SharedNetworkID: 1,
			Prefix:          "127.0.0.1",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: map[string]interface{}{
						"total-addresses":    float64(300),
						"assigned-addresses": float64(90),
						"declined-addresses": float64(100),
					},
				},
			},
		},
	}

	// Act
	calculator := kea.NewUtilizationCalculator()
	for _, subnet := range subnets {
		_ = calculator.Add(subnet)
	}

	// Assert
	require.Len(t, calculator.SharedNetworks, 1)
	utilization := calculator.SharedNetworks[1]
	require.InDelta(t, float64(140.0/600.0), utilization.AddressUtilization(), float64(0.001))
	require.InDelta(t, float64(100.0/120.0), utilization.PdUtilization(), float64(0.001))
}

// Test that the calculator separates the shared networks during the calculations.
func TestCalculatorAddMultipleSharedNetworkSubnets(t *testing.T) {
	// Arrange
	subnets := []*dbmodel.Subnet{
		{
			SharedNetworkID: 13,
			Prefix:          "20::",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: map[string]interface{}{
						"total-nas":    float64(100),
						"assigned-nas": float64(10),
						"declined-nas": float64(20),
						"total-pds":    float64(40),
						"assigned-pds": float64(30),
					},
				},
			},
		},
		{
			SharedNetworkID: 4,
			Prefix:          "20::",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					Stats: map[string]interface{}{
						"total-nas":    float64(200),
						"assigned-nas": float64(40),
						"declined-nas": float64(50),
						"total-pds":    float64(80),
						"assigned-pds": float64(70),
					},
				},
			},
		},
	}

	// Act
	calculator := kea.NewUtilizationCalculator()
	for _, subnet := range subnets {
		_ = calculator.Add(subnet)
	}

	// Assert
	require.Len(t, calculator.SharedNetworks, 2)
	utilization := calculator.SharedNetworks[13]
	require.InDelta(t, float64(10.0/100.0), utilization.AddressUtilization(), float64(0.001))
	require.InDelta(t, float64(30.0/40.0), utilization.PdUtilization(), float64(0.001))
	utilization = calculator.SharedNetworks[4]
	require.InDelta(t, float64(40.0/200.0), utilization.AddressUtilization(), float64(0.001))
	require.InDelta(t, float64(70.0/80.0), utilization.PdUtilization(), float64(0.001))
}

// Test that the calculator works for a subnet without the local subnets.
func TestCalculatorAddEmptySubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		SharedNetworkID: 42,
		Prefix:          "20::",
		LocalSubnets:    []*dbmodel.LocalSubnet{},
	}

	// Act
	calculator := kea.NewUtilizationCalculator()
	utilization := calculator.Add(subnet)

	// Assert
	require.InDelta(t, float64(0.0), utilization.AddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), utilization.PdUtilization(), float64(0.001))
	utilization = calculator.SharedNetworks[42]
	require.InDelta(t, float64(0.0), utilization.AddressUtilization(), float64(0.001))
	require.InDelta(t, float64(0.0), utilization.PdUtilization(), float64(0.001))
}
