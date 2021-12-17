package kea

import (
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// The sum of statistics from all subnets.
type globalStats struct {
	totalAddresses         *storkutil.BigCounter
	totalAssignedAddresses *storkutil.BigCounter
	totalDeclinedAddresses *storkutil.BigCounter
	totalNAs               *storkutil.BigCounter
	totalAssignedNAs       *storkutil.BigCounter
	totalDeclinedNAs       *storkutil.BigCounter
	totalPDs               *storkutil.BigCounter
	totalAssignedPDs       *storkutil.BigCounter
}

func newGlobalStats() *globalStats {
	return &globalStats{
		totalAddresses:         storkutil.NewBigCounter(0),
		totalAssignedAddresses: storkutil.NewBigCounter(0),
		totalDeclinedAddresses: storkutil.NewBigCounter(0),
		totalNAs:               storkutil.NewBigCounter(0),
		totalAssignedNAs:       storkutil.NewBigCounter(0),
		totalDeclinedNAs:       storkutil.NewBigCounter(0),
		totalPDs:               storkutil.NewBigCounter(0),
		totalAssignedPDs:       storkutil.NewBigCounter(0),
	}
}

// Add the IPv4 subnet statistics to the global state.
func (g *globalStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	g.totalAddresses.Add(subnet.totalAddresses)
	g.totalAssignedAddresses.Add(subnet.totalAssignedAddresses)
	g.totalDeclinedAddresses.Add(subnet.totalDeclinedAddresses)
}

// Add the IPv6 subnet statistics to the global state.
func (g *globalStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	g.totalNAs.Add(subnet.totalNAs)
	g.totalAssignedNAs.Add(subnet.totalAssignedNAs)
	g.totalDeclinedNAs.Add(subnet.totalDeclinedNAs)
	g.totalPDs.Add(subnet.totalPDs)
	g.totalAssignedPDs.Add(subnet.totalAssignedPDs)
}

// General subnet lease statistics.
// It unifies the IPv4 and IPv6 subnet data.
type leaseStats interface {
	getAddressUtilization() float64
	getPDUtilization() float64
}

// Sum of the subnet statistics from the single shared network.
type sharedNetworkStats struct {
	totalAddresses         *storkutil.BigCounter
	totalAssignedAddresses *storkutil.BigCounter
	totalPDs               *storkutil.BigCounter
	totalAssignedPDs       *storkutil.BigCounter
}

func newSharedNetworkStats() *sharedNetworkStats {
	return &sharedNetworkStats{
		totalAddresses:         storkutil.NewBigCounter(0),
		totalAssignedAddresses: storkutil.NewBigCounter(0),
		totalPDs:               storkutil.NewBigCounter(0),
		totalAssignedPDs:       storkutil.NewBigCounter(0),
	}
}

// Address utilization of the shared network.
func (s *sharedNetworkStats) getAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return s.totalAssignedAddresses.DivideSafeBy(s.totalAddresses)
}

// Delegated prefix utilization of the shared network.
func (s *sharedNetworkStats) getPDUtilization() float64 {
	return s.totalAssignedPDs.DivideSafeBy(s.totalPDs)
}

// Add the IPv4 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	s.totalAddresses.Add(subnet.totalAddresses)
	s.totalAssignedAddresses.Add(subnet.totalAssignedAddresses)
}

// Add the IPv6 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	s.totalAddresses.Add(subnet.totalNAs)
	s.totalAssignedAddresses.Add(subnet.totalAssignedNAs)
	s.totalPDs.Add(subnet.totalPDs)
	s.totalAssignedPDs.Add(subnet.totalAssignedPDs)
}

// IPv4 statistics retrieved from the single subnet.
type subnetIPv4Stats struct {
	totalAddresses         *storkutil.BigCounter
	totalAssignedAddresses *storkutil.BigCounter
	totalDeclinedAddresses *storkutil.BigCounter
}

// Return the address utilization for a single IPv4 subnet.
func (s *subnetIPv4Stats) getAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return s.totalAssignedAddresses.DivideSafeBy(s.totalAddresses)
}

// Return the delegated prefix utilization for a single IPv4 subnet.
// It's always zero because the PD doesn't apply to IPv4.
func (s *subnetIPv4Stats) getPDUtilization() float64 {
	return 0.0
}

// IPv6 statistics retrieved from the single subnet.
type subnetIPv6Stats struct {
	totalNAs         *storkutil.BigCounter
	totalAssignedNAs *storkutil.BigCounter
	totalDeclinedNAs *storkutil.BigCounter
	totalPDs         *storkutil.BigCounter
	totalAssignedPDs *storkutil.BigCounter
}

// Return the IPv6 address utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) getAddressUtilization() float64 {
	// The assigned addresses include the declined ones that aren't reclaimed yet.
	return s.totalAssignedNAs.DivideSafeBy(s.totalNAs)
}

// Return the delegated prefix utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) getPDUtilization() float64 {
	return s.totalAssignedPDs.DivideSafeBy(s.totalPDs)
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
func sumStatLocalSubnets(subnet *dbmodel.Subnet, statName string) *storkutil.BigCounter {
	sum := storkutil.NewBigCounter(0)
	for _, localSubnet := range subnet.LocalSubnets {
		stat := getLocalSubnetStatValueIntOrDefault(localSubnet, statName)
		sum.AddUInt64(stat)
	}
	return sum
}

// Retrieve the statistic value from the provided local subnet or return zero value.
func getLocalSubnetStatValueIntOrDefault(localSubnet *dbmodel.LocalSubnet, name string) uint64 {
	value, ok := localSubnet.Stats[name]
	if !ok {
		return 0
	}

	valueInt, ok := value.(uint64)
	if !ok {
		return 0
	}

	// Kea casts the value to int64 before serializing it to JSON.
	return valueInt
}
