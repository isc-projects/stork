package kea

import (
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// The sum of statistics from all subnets.
type globalStats struct {
	totalIPv4Addresses             *storkutil.BigCounter
	totalAssignedIPv4Addresses     *storkutil.BigCounter
	totalDeclinedIPv4Addresses     *storkutil.BigCounter
	totalIPv6Addresses             *storkutil.BigCounter
	totalAssignedIPv6Addresses     *storkutil.BigCounter
	totalDeclinedIPv6Addresses     *storkutil.BigCounter
	totalDelegatedPrefixes         *storkutil.BigCounter
	totalAssignedDelegatedPrefixes *storkutil.BigCounter
}

// Constructor of the global statistic struct with all
// counters set to zero.
func newGlobalStats() *globalStats {
	return &globalStats{
		totalIPv4Addresses:             storkutil.NewBigCounter(0),
		totalAssignedIPv4Addresses:     storkutil.NewBigCounter(0),
		totalDeclinedIPv4Addresses:     storkutil.NewBigCounter(0),
		totalIPv6Addresses:             storkutil.NewBigCounter(0),
		totalAssignedIPv6Addresses:     storkutil.NewBigCounter(0),
		totalDeclinedIPv6Addresses:     storkutil.NewBigCounter(0),
		totalDelegatedPrefixes:         storkutil.NewBigCounter(0),
		totalAssignedDelegatedPrefixes: storkutil.NewBigCounter(0),
	}
}

// Add the IPv4 subnet statistics to the global state.
func (g *globalStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	g.totalIPv4Addresses.AddUint64(subnet.totalAddresses)
	g.totalAssignedIPv4Addresses.AddUint64(subnet.totalAssignedAddresses)
	g.totalDeclinedIPv4Addresses.AddUint64(subnet.totalDeclinedAddresses)
}

// Add the IPv6 subnet statistics to the global state.
func (g *globalStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	g.totalIPv6Addresses.Add(subnet.totalAddresses)
	g.totalAssignedIPv6Addresses.Add(subnet.totalAssignedAddresses)
	g.totalDeclinedIPv6Addresses.Add(subnet.totalDeclinedAddresses)
	g.totalDelegatedPrefixes.Add(subnet.totalDelegatedPrefixes)
	g.totalAssignedDelegatedPrefixes.Add(subnet.totalAssignedDelegatedPrefixes)
}

// General subnet lease statistics.
// It unifies the IPv4 and IPv6 subnet data.
type subnetStats interface {
	GetAddressUtilization() float64
	GetDelegatedPrefixUtilization() float64
	GetStatistics() dbmodel.SubnetStats
}

// Sum of the subnet statistics from the single shared network.
type sharedNetworkStats struct {
	totalAddresses                 *storkutil.BigCounter
	totalAssignedAddresses         *storkutil.BigCounter
	totalDelegatedPrefixes         *storkutil.BigCounter
	totalAssignedDelegatedPrefixes *storkutil.BigCounter
}

// Constructor of the sharedNetworkStats struct with
// all counters set to zero.
func newSharedNetworkStats() *sharedNetworkStats {
	return &sharedNetworkStats{
		totalAddresses:                 storkutil.NewBigCounter(0),
		totalAssignedAddresses:         storkutil.NewBigCounter(0),
		totalDelegatedPrefixes:         storkutil.NewBigCounter(0),
		totalAssignedDelegatedPrefixes: storkutil.NewBigCounter(0),
	}
}

// Address utilization of the shared network.
func (s *sharedNetworkStats) GetAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return s.totalAssignedAddresses.DivideSafeBy(s.totalAddresses)
}

// Delegated prefix utilization of the shared network.
func (s *sharedNetworkStats) GetDelegatedPrefixUtilization() float64 {
	return s.totalAssignedDelegatedPrefixes.DivideSafeBy(s.totalDelegatedPrefixes)
}

// Returns set of accumulated statistics from all local subnets belonging to
// a given shared network.
func (s *sharedNetworkStats) GetStatistics() dbmodel.SubnetStats {
	return dbmodel.SubnetStats{
		dbmodel.SubnetStatsNameTotalNAs:    s.totalAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedNAs: s.totalAssignedAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameTotalPDs:    s.totalDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedPDs: s.totalAssignedDelegatedPrefixes.ConvertToNativeType(),
	}
}

// Add the IPv4 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	s.totalAddresses.AddUint64(subnet.totalAddresses)
	s.totalAssignedAddresses.AddUint64(subnet.totalAssignedAddresses)
}

// Add the IPv6 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	s.totalAddresses.Add(subnet.totalAddresses)
	s.totalAssignedAddresses.Add(subnet.totalAssignedAddresses)
	s.totalDelegatedPrefixes.Add(subnet.totalDelegatedPrefixes)
	s.totalAssignedDelegatedPrefixes.Add(subnet.totalAssignedDelegatedPrefixes)
}

// IPv4 statistics retrieved from the single subnet.
type subnetIPv4Stats struct {
	totalAddresses         uint64
	totalAssignedAddresses uint64
	totalDeclinedAddresses uint64
}

// Return the address utilization for a single IPv4 subnet.
func (s *subnetIPv4Stats) GetAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	if s.totalAddresses == 0 {
		return 0
	}
	return float64(s.totalAssignedAddresses) / float64(s.totalAddresses)
}

// Return the delegated prefix utilization for a single IPv4 subnet.
// It's always zero because the delegated prefix doesn't apply to IPv4.
func (s *subnetIPv4Stats) GetDelegatedPrefixUtilization() float64 {
	return 0.0
}

// Returns set of accumulated statistics from all local subnets belonging to
// a given IPv4 subnet.
func (s *subnetIPv4Stats) GetStatistics() dbmodel.SubnetStats {
	return dbmodel.SubnetStats{
		dbmodel.SubnetStatsNameTotalAddresses:    s.totalAddresses,
		dbmodel.SubnetStatsNameAssignedAddresses: s.totalAssignedAddresses,
		dbmodel.SubnetStatsNameDeclinedAddresses: s.totalDeclinedAddresses,
	}
}

// IPv6 statistics retrieved from the single subnet.
type subnetIPv6Stats struct {
	totalAddresses                 *storkutil.BigCounter
	totalAssignedAddresses         *storkutil.BigCounter
	totalDeclinedAddresses         *storkutil.BigCounter
	totalDelegatedPrefixes         *storkutil.BigCounter
	totalAssignedDelegatedPrefixes *storkutil.BigCounter
}

// Return the IPv6 address utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) GetAddressUtilization() float64 {
	// The assigned addresses include the declined ones that aren't reclaimed yet.
	return s.totalAssignedAddresses.DivideSafeBy(s.totalAddresses)
}

// Return the delegated prefix utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) GetDelegatedPrefixUtilization() float64 {
	return s.totalAssignedDelegatedPrefixes.DivideSafeBy(s.totalDelegatedPrefixes)
}

// Returns set of accumulated statistics from all local subnets belonging to
// a given IPv6 network.
func (s *subnetIPv6Stats) GetStatistics() dbmodel.SubnetStats {
	return dbmodel.SubnetStats{
		dbmodel.SubnetStatsNameTotalNAs:    s.totalAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedNAs: s.totalAssignedAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameDeclinedNAs: s.totalDeclinedAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameTotalPDs:    s.totalDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedPDs: s.totalAssignedDelegatedPrefixes.ConvertToNativeType(),
	}
}

// Statistics Counter is a helper for calculating the global IPv4 and IPv6
// address, and delegated prefix statistics per subnet and shared network.
type statisticsCounter struct {
	global             *globalStats
	sharedNetworks     map[int64]*sharedNetworkStats
	outOfPoolAddresses map[int64]uint64
	outOfPoolPrefixes  map[int64]uint64
	excludedDaemons    map[int64]bool
}

// Constructor of the statistics counter.
func newStatisticsCounter() *statisticsCounter {
	return &statisticsCounter{
		sharedNetworks:     make(map[int64]*sharedNetworkStats),
		global:             newGlobalStats(),
		outOfPoolAddresses: make(map[int64]uint64),
		outOfPoolPrefixes:  make(map[int64]uint64),
	}
}

// The total IPv4 and IPv6 addresses statistics returned by Kea exclude
// out-of-pool reservations, yielding possibly incorrect calculations.
// The values can be corrected by including the out-of-pool
// reservation counts from the Stork database. The argument is a subnet
// ID mapping to the total out-of-pool addresses for the subnet.
func (c *statisticsCounter) setOutOfPoolAddresses(outOfPoolAddressesPerSubnet map[int64]uint64) {
	c.outOfPoolAddresses = outOfPoolAddressesPerSubnet
}

// The total delegated prefixes statistics returned by Kea exclude
// out-of-pool reservations, yielding possibly incorrect calculations.
// The values can be corrected by including the out-of-pool
// reservation counts from the Stork database. The argument is a subnet
// ID mapping to the total out-of-pool prefixes for the subnet.
func (c *statisticsCounter) setOutOfPoolPrefixes(outOfPoolPrefixesPerSubnet map[int64]uint64) {
	c.outOfPoolPrefixes = outOfPoolPrefixesPerSubnet
}

// The subnet statistics from the specific daemons can be excluded from the
// calculations. It allows for avoiding duplicating values from the HA servers.
func (c *statisticsCounter) setExcludedDaemons(daemons []int64) {
	c.excludedDaemons = make(map[int64]bool, len(daemons))
	for _, daemon := range daemons {
		c.excludedDaemons[daemon] = true
	}
}

// Add the subnet statistics for the current counter state.
// The total counter (total addresses or NAs) will be increased by
// outOfPool value.
// It returns the statistics of this subnet.
func (c *statisticsCounter) add(subnet *dbmodel.Subnet) subnetStats {
	if subnet.SharedNetworkID != 0 {
		_, ok := c.sharedNetworks[subnet.SharedNetworkID]
		if !ok {
			c.sharedNetworks[subnet.SharedNetworkID] = newSharedNetworkStats()
		}
	}

	outOfPoolAddresses, ok := c.outOfPoolAddresses[subnet.ID]
	if !ok {
		outOfPoolAddresses = 0
	}

	if subnet.GetFamily() == 4 {
		return c.addIPv4Subnet(subnet, outOfPoolAddresses)
	}

	outOfPoolPrefixes, ok := c.outOfPoolPrefixes[subnet.ID]
	if !ok {
		outOfPoolPrefixes = 0
	}

	return c.addIPv6Subnet(subnet, outOfPoolAddresses, outOfPoolPrefixes)
}

// The resulting addresses counter will be a sum of the addresses returned by Kea for this
// subnet and the outOfPool counter holding the number of the out-of-pool reservations
// that Kea does not include in its statistics.
func (c *statisticsCounter) addIPv4Subnet(subnet *dbmodel.Subnet, outOfPool uint64) *subnetIPv4Stats {
	stats := &subnetIPv4Stats{
		totalAddresses:         sumStatLocalSubnetsIPv4(subnet, dbmodel.SubnetStatsNameTotalAddresses, c.excludedDaemons) + outOfPool,
		totalAssignedAddresses: sumStatLocalSubnetsIPv4(subnet, dbmodel.SubnetStatsNameAssignedAddresses, c.excludedDaemons),
		totalDeclinedAddresses: sumStatLocalSubnetsIPv4(subnet, dbmodel.SubnetStatsNameDeclinedAddresses, c.excludedDaemons),
	}

	if subnet.SharedNetworkID != 0 {
		c.sharedNetworks[subnet.SharedNetworkID].addIPv4Subnet(stats)
	}

	c.global.addIPv4Subnet(stats)

	return stats
}

// The resulting addresses counter will be a sum of the addresses returned by Kea for this
// subnet and the outOfPool counter holding the number of the out-of-pool reservations
// that Kea does not include in its statistics. The delegated prefixes counter will be
// calculated similarly.
func (c *statisticsCounter) addIPv6Subnet(subnet *dbmodel.Subnet, outOfPoolTotalAddresses, outOfPoolDelegatedPrefixes uint64) *subnetIPv6Stats {
	stats := &subnetIPv6Stats{
		totalAddresses:                 sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameTotalNAs, c.excludedDaemons).AddUint64(outOfPoolTotalAddresses),
		totalAssignedAddresses:         sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameAssignedNAs, c.excludedDaemons),
		totalDeclinedAddresses:         sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameDeclinedNAs, c.excludedDaemons),
		totalDelegatedPrefixes:         sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameTotalPDs, c.excludedDaemons).AddUint64(outOfPoolDelegatedPrefixes),
		totalAssignedDelegatedPrefixes: sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameAssignedPDs, c.excludedDaemons),
	}

	if subnet.SharedNetworkID != 0 {
		c.sharedNetworks[subnet.SharedNetworkID].addIPv6Subnet(stats)
	}

	c.global.addIPv6Subnet(stats)

	return stats
}

// Return the sum of specific statistics for each local subnet in the provided subnet.
// It expects that the counting value may exceed uint64 range.
// The local subnets that belong to excluded daemons will not be processed.
func sumStatLocalSubnetsIPv6(subnet *dbmodel.Subnet, statName string, excludedDaemons map[int64]bool) *storkutil.BigCounter {
	sum := storkutil.NewBigCounter(0)
	for _, localSubnet := range subnet.LocalSubnets {
		if _, ok := excludedDaemons[localSubnet.DaemonID]; ok {
			continue
		}

		value := localSubnet.Stats.GetBigCounter(statName)
		if value == nil {
			continue
		}

		sum.Add(value)
	}
	return sum
}

// Return the sum of specific statistics for each local subnet in the provided subnet.
// It assumes that the counting value does not exceed uint64 range.
// The local subnets that belong to excluded daemons will not be processed.
func sumStatLocalSubnetsIPv4(subnet *dbmodel.Subnet, statName string, excludedDaemons map[int64]bool) uint64 {
	sum := uint64(0)
	for _, localSubnet := range subnet.LocalSubnets {
		if _, ok := excludedDaemons[localSubnet.DaemonID]; ok {
			continue
		}

		value := localSubnet.Stats.GetBigCounter(statName)
		if value == nil {
			continue
		}

		valueUint, ok := value.ToUint64()
		if !ok {
			continue
		}

		sum += valueUint
	}
	return sum
}
