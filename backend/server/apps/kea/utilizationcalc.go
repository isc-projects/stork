package kea

import (
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// The invalid statistic value.
// It is returned by Kea when the value exceed the int64/float64 range.
const invalidStatValue = -1

// The sum of statistics from all subnets.
type globalStats struct {
	totalAddresses         *storkutil.BigNumber
	totalAssignedAddresses *storkutil.BigNumber
	totalDeclinedAddresses *storkutil.BigNumber
	totalNAs               *storkutil.BigNumber
	totalAssignedNAs       *storkutil.BigNumber
	totalDeclinedNAs       *storkutil.BigNumber
	totalPDs               *storkutil.BigNumber
	totalAssignedPDs       *storkutil.BigNumber
}

func newGlobalStats() *globalStats {
	return &globalStats{
		totalAddresses:         storkutil.NewBigNumber(0),
		totalAssignedAddresses: storkutil.NewBigNumber(0),
		totalDeclinedAddresses: storkutil.NewBigNumber(0),
		totalNAs:               storkutil.NewBigNumber(0),
		totalAssignedNAs:       storkutil.NewBigNumber(0),
		totalDeclinedNAs:       storkutil.NewBigNumber(0),
		totalPDs:               storkutil.NewBigNumber(0),
		totalAssignedPDs:       storkutil.NewBigNumber(0),
	}
}

// Add the IPv4 subnet statistics to the global state.
func (g *globalStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	g.totalAddresses.AddInPlace(subnet.totalAddresses)
	g.totalAssignedAddresses.AddInPlace(subnet.totalAssignedAddresses)
	g.totalDeclinedAddresses.AddInPlace(subnet.totalDeclinedAddresses)
}

// Add the IPv6 subnet statistics to the global state.
func (g *globalStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	g.totalNAs.AddInPlace(subnet.totalNAs)
	g.totalAssignedNAs.AddInPlace(subnet.totalAssignedNAs)
	g.totalDeclinedNAs.AddInPlace(subnet.totalDeclinedNAs)
	g.totalPDs.AddInPlace(subnet.totalPDs)
	g.totalAssignedPDs.AddInPlace(subnet.totalAssignedPDs)
}

// General subnet lease statistics.
// It unifies the IPv4 and IPv6 subnet data.
type leaseStats interface {
	getAddressUtilization() float64
	getPDUtilization() float64
}

// Sum of the subnet statistics from the single shared network.
type sharedNetworkStats struct {
	totalAddresses         *storkutil.BigNumber
	totalAssignedAddresses *storkutil.BigNumber
	totalPDs               *storkutil.BigNumber
	totalAssignedPDs       *storkutil.BigNumber
}

func newSharedNetworkStats() *sharedNetworkStats {
	return &sharedNetworkStats{
		totalAddresses:         storkutil.NewBigNumber(0),
		totalAssignedAddresses: storkutil.NewBigNumber(0),
		totalPDs:               storkutil.NewBigNumber(0),
		totalAssignedPDs:       storkutil.NewBigNumber(0),
	}
}

// Address utilization of the shared network.
func (s *sharedNetworkStats) getAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return s.totalAssignedAddresses.DivideSafe(s.totalAddresses)
}

// Delegated prefix utilization of the shared network.
func (s *sharedNetworkStats) getPDUtilization() float64 {
	return s.totalAssignedPDs.DivideSafe(s.totalPDs)
}

// Add the IPv4 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	s.totalAddresses.AddInPlace(subnet.totalAddresses)
	s.totalAssignedAddresses.AddInPlace(subnet.totalAssignedAddresses)
}

// Add the IPv6 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	s.totalAddresses.AddInPlace(subnet.totalNAs)
	s.totalAssignedAddresses.AddInPlace(subnet.totalAssignedNAs)
	s.totalPDs.AddInPlace(subnet.totalPDs)
	s.totalAssignedPDs.AddInPlace(subnet.totalAssignedPDs)
}

// IPv4 statistics retrieved from the single subnet.
type subnetIPv4Stats struct {
	totalAddresses         *storkutil.BigNumber
	totalAssignedAddresses *storkutil.BigNumber
	totalDeclinedAddresses *storkutil.BigNumber
}

func newSubnetIPv4Stats() *subnetIPv4Stats {
	return &subnetIPv4Stats{
		totalAddresses:         storkutil.NewBigNumber(0),
		totalAssignedAddresses: storkutil.NewBigNumber(0),
		totalDeclinedAddresses: storkutil.NewBigNumber(0),
	}
}

// Return the address utilization for a single IPv4 subnet.
func (s *subnetIPv4Stats) getAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return s.totalAssignedAddresses.DivideSafe(s.totalAddresses)
}

// Return the delegated prefix utilization for a single IPv4 subnet.
// It's always zero because the PD doesn't apply to IPv4.
func (s *subnetIPv4Stats) getPDUtilization() float64 {
	return 0.0
}

// IPv6 statistics retrieved from the single subnet.
type subnetIPv6Stats struct {
	totalNAs         *storkutil.BigNumber
	totalAssignedNAs *storkutil.BigNumber
	totalDeclinedNAs *storkutil.BigNumber
	totalPDs         *storkutil.BigNumber
	totalAssignedPDs *storkutil.BigNumber
}

func newSubnetIPv6Stats() *subnetIPv6Stats {
	return &subnetIPv6Stats{
		totalNAs:         storkutil.NewBigNumber(0),
		totalAssignedNAs: storkutil.NewBigNumber(0),
		totalDeclinedNAs: storkutil.NewBigNumber(0),
		totalPDs:         storkutil.NewBigNumber(0),
		totalAssignedPDs: storkutil.NewBigNumber(0),
	}
}

// Return the IPv6 address utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) getAddressUtilization() float64 {
	// The assigned addresses include the declined ones that aren't reclaimed yet.
	return s.totalAssignedNAs.DivideSafe(s.totalNAs)
}

// Return the delegated prefix utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) getPDUtilization() float64 {
	return s.totalAssignedPDs.DivideSafe(s.totalPDs)
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
func sumStatLocalSubnets(subnet *dbmodel.Subnet, statName string) *storkutil.BigNumber {
	sum := storkutil.NewBigNumber(0)
	for _, localSubnet := range subnet.LocalSubnets {
		stat := getLocalSubnetStatValueIntOrDefault(localSubnet, statName)

		if stat == invalidStatValue {
			return storkutil.NewBigNumberNaN()
		}

		sum = sum.AddInt64(stat)
	}
	return sum
}

// Retrieve the statistic value from the provided local subnet or return zero value.
func getLocalSubnetStatValueIntOrDefault(localSubnet *dbmodel.LocalSubnet, name string) int64 {
	value, ok := localSubnet.Stats[name]
	if !ok {
		return 0
	}

	valueFloat, ok := value.(float64)
	if !ok {
		return 0
	}

	return int64(valueFloat)
}
