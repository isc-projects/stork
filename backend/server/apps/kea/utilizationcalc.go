package kea

import (
	"math"
	"math/big"

	dbmodel "isc.org/stork/server/database/model"
)

// The sum of statistics from all subnets.
type globalStats struct {
	totalAddresses         *big.Float
	totalAssignedAddresses *big.Float
	totalDeclinedAddresses *big.Float
	totalNAs               *big.Float
	totalAssignedNAs       *big.Float
	totalDeclinedNAs       *big.Float
	totalPDs               *big.Float
	totalAssignedPDs       *big.Float
}

func newGlobalStats() *globalStats {
	return &globalStats{
		totalAddresses:         big.NewFloat(0),
		totalAssignedAddresses: big.NewFloat(0),
		totalDeclinedAddresses: big.NewFloat(0),
		totalNAs:               big.NewFloat(0),
		totalAssignedNAs:       big.NewFloat(0),
		totalDeclinedNAs:       big.NewFloat(0),
		totalPDs:               big.NewFloat(0),
		totalAssignedPDs:       big.NewFloat(0),
	}
}

// Add the IPv4 subnet statistics to the global state.
func (g *globalStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	g.totalAddresses.Add(g.totalAddresses, big.NewFloat(subnet.totalAddresses))
	g.totalAssignedAddresses.Add(g.totalAssignedAddresses, big.NewFloat(subnet.totalAssignedAddresses))
	g.totalDeclinedAddresses.Add(g.totalDeclinedAddresses, big.NewFloat(subnet.totalDeclinedAddresses))
}

// Add the IPv6 subnet statistics to the global state.
func (g *globalStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	g.totalNAs.Add(g.totalNAs, big.NewFloat(subnet.totalNAs))
	g.totalAssignedNAs.Add(g.totalAssignedNAs, big.NewFloat(subnet.totalAssignedNAs))
	g.totalDeclinedNAs.Add(g.totalDeclinedNAs, big.NewFloat(subnet.totalDeclinedNAs))
	g.totalPDs.Add(g.totalPDs, big.NewFloat(subnet.totalPDs))
	g.totalAssignedPDs.Add(g.totalAssignedPDs, big.NewFloat(subnet.totalAssignedPDs))
}

// General subnet lease statistics.
// It unifies the IPv4 and IPv6 subnet data.
type leaseStats interface {
	getAddressUtilization() float64
	getPDUtilization() float64
}

// Sum of the subnet statistics from the single shared network.
type sharedNetworkStats struct {
	totalAddresses         *big.Float
	totalAssignedAddresses *big.Float
	totalPDs               *big.Float
	totalAssignedPDs       *big.Float
}

func newSharedNetworkStats() *sharedNetworkStats {
	return &sharedNetworkStats{
		totalAddresses:         big.NewFloat(0),
		totalAssignedAddresses: big.NewFloat(0),
		totalPDs:               big.NewFloat(0),
		totalAssignedPDs:       big.NewFloat(0),
	}
}

// Address utilization of the shared network.
func (s *sharedNetworkStats) getAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return safeDivBigFloat(s.totalAssignedAddresses, s.totalAddresses)
}

// Delegated prefix utilization of the shared network.
func (s *sharedNetworkStats) getPDUtilization() float64 {
	return safeDivBigFloat(s.totalAssignedPDs, s.totalPDs)
}

// Add the IPv4 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	s.totalAddresses.Add(s.totalAddresses, big.NewFloat(subnet.totalAddresses))
	s.totalAssignedAddresses.Add(s.totalAssignedAddresses, big.NewFloat(subnet.totalAssignedAddresses))
}

// Add the IPv6 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	s.totalAddresses.Add(s.totalAddresses, big.NewFloat(subnet.totalNAs))
	s.totalAssignedAddresses.Add(s.totalAssignedAddresses, big.NewFloat(subnet.totalAssignedNAs))
	s.totalPDs.Add(s.totalPDs, big.NewFloat(subnet.totalPDs))
	s.totalAssignedPDs.Add(s.totalAssignedPDs, big.NewFloat(subnet.totalAssignedPDs))
}

// IPv4 statistics retrieved from the single subnet.
type subnetIPv4Stats struct {
	totalAddresses         float64
	totalAssignedAddresses float64
	totalDeclinedAddresses float64
}

// Return the address utilization for a single IPv4 subnet.
func (s *subnetIPv4Stats) getAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return safeDiv(s.totalAssignedAddresses, s.totalAddresses)
}

// Return the delegated prefix utilization for a single IPv4 subnet.
// It's always zero because the PD doesn't apply to IPv4.
func (s *subnetIPv4Stats) getPDUtilization() float64 {
	return 0.0
}

// IPv6 statistics retrieved from the single subnet.
type subnetIPv6Stats struct {
	totalNAs         float64
	totalAssignedNAs float64
	totalDeclinedNAs float64
	totalPDs         float64
	totalAssignedPDs float64
}

// Return the IPv6 address utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) getAddressUtilization() float64 {
	// The assigned addresses include the declined ones that aren't reclaimed yet.
	return safeDiv(s.totalAssignedNAs, s.totalNAs)
}

// Return the delegated prefix utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) getPDUtilization() float64 {
	return safeDiv(s.totalAssignedPDs, s.totalPDs)
}

// Utilization calculator is a helper for calculating the global
// IPv4 address/NA/PD statistic and utilization per subnet and
// shared network.
type utilizationCalculator struct {
	global         *globalStats
	sharedNetworks map[int64]*sharedNetworkStats
}

// Constructor of the utilization calculator.
func newUtilizationCalculator() *utilizationCalculator {
	return &utilizationCalculator{
		sharedNetworks: make(map[int64]*sharedNetworkStats),
		global:         newGlobalStats(),
	}
}

// Add the subnet statistics for the current calculator state.
// It returns the utilization of this subnet.
func (c *utilizationCalculator) add(subnet *dbmodel.Subnet) leaseStats {
	if subnet.SharedNetworkID != 0 {
		_, ok := c.sharedNetworks[subnet.SharedNetworkID]
		if !ok {
			c.sharedNetworks[subnet.SharedNetworkID] = newSharedNetworkStats()
		}
	}

	if subnet.GetFamily() == 6 {
		return c.addIPv6Subnet(subnet)
	}
	return c.addIPv4Subnet(subnet)
}

// Add the IPv4 subnet statistics for the current calculator state.
// It shouldn't be called outside the calculator.
func (c *utilizationCalculator) addIPv4Subnet(subnet *dbmodel.Subnet) *subnetIPv4Stats {
	stats := &subnetIPv4Stats{
		totalAddresses:         sumStatLocalSubnets(subnet, "total-addresses"),
		totalAssignedAddresses: sumStatLocalSubnets(subnet, "assigned-addresses"),
		totalDeclinedAddresses: sumStatLocalSubnets(subnet, "declined-addresses"),
	}

	if subnet.SharedNetworkID != 0 {
		c.sharedNetworks[subnet.SharedNetworkID].addIPv4Subnet(stats)
	}

	c.global.addIPv4Subnet(stats)

	return stats
}

// Add the IPv6 subnet statistics for the current calculator state.
// It shouldn't be called outside the calculator.
func (c *utilizationCalculator) addIPv6Subnet(subnet *dbmodel.Subnet) *subnetIPv6Stats {
	stats := &subnetIPv6Stats{
		totalNAs:         sumStatLocalSubnets(subnet, "total-nas"),
		totalAssignedNAs: sumStatLocalSubnets(subnet, "assigned-nas"),
		totalDeclinedNAs: sumStatLocalSubnets(subnet, "declined-nas"),
		totalPDs:         sumStatLocalSubnets(subnet, "total-pds"),
		totalAssignedPDs: sumStatLocalSubnets(subnet, "assigned-pds"),
	}

	if subnet.SharedNetworkID != 0 {
		c.sharedNetworks[subnet.SharedNetworkID].addIPv6Subnet(stats)
	}

	c.global.addIPv6Subnet(stats)

	return stats
}

// Return the sum of specific statistics for each local subnet in the provided subnet.
func sumStatLocalSubnets(subnet *dbmodel.Subnet, statName string) float64 {
	sum := 0.0
	for _, localSubnet := range subnet.LocalSubnets {
		stat := getLocalSubnetStatValueIntOrDefault(localSubnet, statName)

		// The invalid statistic value.
		// It is returned by Kea when the value exceed the int64/float64 range.
		if stat == -1 {
			return math.NaN()
		}

		sum += stat
	}
	return sum
}

// Retrieve the statistic value from the provided local subnet or return zero value.
func getLocalSubnetStatValueIntOrDefault(localSubnet *dbmodel.LocalSubnet, name string) float64 {
	value, ok := localSubnet.Stats[name]
	if !ok {
		return 0
	}

	valueFloat, ok := value.(float64)
	if !ok {
		return 0
	}

	return valueFloat
}

// Division that doesn't panic when divisive is 0.
func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// Division big float that doesn't panic when divisive is 0.
func safeDivBigFloat(a, b *big.Float) float64 {
	if b.Cmp(big.NewFloat(0)) == 0 {
		return 0
	}
	res, _ := new(big.Float).Quo(a, b).Float64()
	return res
}
