package kea

import (
	"math/big"

	log "github.com/sirupsen/logrus"
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

// Constructor of the global statistic struct with all
// counters set to zero.
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
	g.totalAddresses.AddUint64(subnet.totalAddresses)
	g.totalAssignedAddresses.AddUint64(subnet.totalAssignedAddresses)
	g.totalDeclinedAddresses.AddUint64(subnet.totalDeclinedAddresses)
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

// Constructor of the sharedNetworkStats struct with
// all counters set to zero.
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
	s.totalAddresses.AddUint64(subnet.totalAddresses)
	s.totalAssignedAddresses.AddUint64(subnet.totalAssignedAddresses)
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
	totalAddresses         uint64
	totalAssignedAddresses uint64
	totalDeclinedAddresses uint64
}

// Return the address utilization for a single IPv4 subnet.
func (s *subnetIPv4Stats) getAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	if s.totalAddresses == 0 {
		return 0
	}
	return float64(s.totalAssignedAddresses) / float64(s.totalAddresses)
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
// The total counter (total addresses or NAs) will be increased by
// extraTotal value.
// It returns the utilization of this subnet.
func (c *utilizationCalculator) add(subnet *dbmodel.Subnet, extraTotal uint64) leaseStats {
	if subnet.SharedNetworkID != 0 {
		_, ok := c.sharedNetworks[subnet.SharedNetworkID]
		if !ok {
			c.sharedNetworks[subnet.SharedNetworkID] = newSharedNetworkStats()
		}
	}

	if subnet.GetFamily() == 6 {
		return c.addIPv6Subnet(subnet, extraTotal)
	}
	return c.addIPv4Subnet(subnet, extraTotal)
}

// Add the IPv4 subnet statistics for the current calculator state.
// Total addresses counter will be increased by the extraTotal value.
// It shouldn't be called outside the calculator.
func (c *utilizationCalculator) addIPv4Subnet(subnet *dbmodel.Subnet, extraTotal uint64) *subnetIPv4Stats {
	stats := &subnetIPv4Stats{
		totalAddresses:         sumStatLocalSubnetsIPv4(subnet, "total-addresses") + extraTotal,
		totalAssignedAddresses: sumStatLocalSubnetsIPv4(subnet, "assigned-addresses"),
		totalDeclinedAddresses: sumStatLocalSubnetsIPv4(subnet, "declined-addresses"),
	}

	if subnet.SharedNetworkID != 0 {
		c.sharedNetworks[subnet.SharedNetworkID].addIPv4Subnet(stats)
	}

	c.global.addIPv4Subnet(stats)

	return stats
}

// Add the IPv6 subnet statistics for the current calculator state.
// The total NAS counter will be increased by the extraTotal value.
// It shouldn't be called outside the calculator.
func (c *utilizationCalculator) addIPv6Subnet(subnet *dbmodel.Subnet, extraTotal uint64) *subnetIPv6Stats {
	stats := &subnetIPv6Stats{
		totalNAs:         sumStatLocalSubnetsIPv6(subnet, "total-nas").AddUint64(extraTotal),
		totalAssignedNAs: sumStatLocalSubnetsIPv6(subnet, "assigned-nas"),
		totalDeclinedNAs: sumStatLocalSubnetsIPv6(subnet, "declined-nas"),
		totalPDs:         sumStatLocalSubnetsIPv6(subnet, "total-pds"),
		totalAssignedPDs: sumStatLocalSubnetsIPv6(subnet, "assigned-pds"),
	}

	if subnet.SharedNetworkID != 0 {
		c.sharedNetworks[subnet.SharedNetworkID].addIPv6Subnet(stats)
	}

	c.global.addIPv6Subnet(stats)

	return stats
}

// Return the sum of specific statistics for each local subnet in the provided subnet.
// It expects that the counting value may exceed uint64 range.
func sumStatLocalSubnetsIPv6(subnet *dbmodel.Subnet, statName string) *storkutil.BigCounter {
	sum := storkutil.NewBigCounter(0)
	hasNegativeStatistic := false
	for _, localSubnet := range subnet.LocalSubnets {
		value, ok := localSubnet.Stats[statName]
		if !ok {
			continue
		}

		valueUint, ok := value.(uint64)
		if ok {
			sum.AddUint64(valueUint)
			continue
		}

		valueBigInt, ok := value.(*big.Int)
		if ok {
			_, ok = sum.AddBigInt(valueBigInt)
			hasNegativeStatistic = hasNegativeStatistic || ok
		}
	}

	if hasNegativeStatistic {
		log.Warnf("Subnet %d contains negative value for statistic: %s", subnet.ID, statName)
	}
	return sum
}

// Return the sum of specific statistics for each local subnet in the provided subnet.
// It assumes that the counting value does not exceed uint64 range.
func sumStatLocalSubnetsIPv4(subnet *dbmodel.Subnet, statName string) uint64 {
	sum := uint64(0)
	for _, localSubnet := range subnet.LocalSubnets {
		value, ok := localSubnet.Stats[statName]
		if !ok {
			continue
		}

		valueUint, ok := value.(uint64)
		if !ok {
			continue
		}

		sum += valueUint
	}
	return sum
}
