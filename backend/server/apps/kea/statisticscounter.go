package kea

import (
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// The sum of statistics from all subnets.
type globalStats struct {
	totalIPv4Addresses                    *storkutil.BigCounter
	totalAssignedIPv4Addresses            *storkutil.BigCounter
	totalDeclinedIPv4Addresses            *storkutil.BigCounter
	totalIPv4AddressesInPools             *storkutil.BigCounter
	totalAssignedIPv4AddressesInPools     *storkutil.BigCounter
	totalDeclinedIPv4AddressesInPools     *storkutil.BigCounter
	totalIPv6Addresses                    *storkutil.BigCounter
	totalIPv6AddressesInPools             *storkutil.BigCounter
	totalAssignedIPv6Addresses            *storkutil.BigCounter
	totalAssignedIPv6AddressesInPools     *storkutil.BigCounter
	totalDeclinedIPv6Addresses            *storkutil.BigCounter
	totalDeclinedIPv6AddressesInPools     *storkutil.BigCounter
	totalDelegatedPrefixes                *storkutil.BigCounter
	totalDelegatedPrefixesInPools         *storkutil.BigCounter
	totalAssignedDelegatedPrefixes        *storkutil.BigCounter
	totalAssignedDelegatedPrefixesInPools *storkutil.BigCounter
}

// Constructor of the global statistic struct with all
// counters set to zero.
func newGlobalStats() *globalStats {
	return &globalStats{
		totalIPv4Addresses:                    storkutil.NewBigCounter(0),
		totalIPv4AddressesInPools:             storkutil.NewBigCounter(0),
		totalAssignedIPv4Addresses:            storkutil.NewBigCounter(0),
		totalAssignedIPv4AddressesInPools:     storkutil.NewBigCounter(0),
		totalDeclinedIPv4Addresses:            storkutil.NewBigCounter(0),
		totalDeclinedIPv4AddressesInPools:     storkutil.NewBigCounter(0),
		totalIPv6Addresses:                    storkutil.NewBigCounter(0),
		totalIPv6AddressesInPools:             storkutil.NewBigCounter(0),
		totalAssignedIPv6Addresses:            storkutil.NewBigCounter(0),
		totalAssignedIPv6AddressesInPools:     storkutil.NewBigCounter(0),
		totalDeclinedIPv6Addresses:            storkutil.NewBigCounter(0),
		totalDeclinedIPv6AddressesInPools:     storkutil.NewBigCounter(0),
		totalDelegatedPrefixes:                storkutil.NewBigCounter(0),
		totalDelegatedPrefixesInPools:         storkutil.NewBigCounter(0),
		totalAssignedDelegatedPrefixes:        storkutil.NewBigCounter(0),
		totalAssignedDelegatedPrefixesInPools: storkutil.NewBigCounter(0),
	}
}

// Add the IPv4 subnet statistics to the global state.
func (g *globalStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	g.totalIPv4Addresses.AddUint64(g.totalIPv4Addresses, subnet.totalAddresses)
	g.totalIPv4AddressesInPools.AddUint64(g.totalIPv4AddressesInPools, subnet.totalAddressesInPools)
	g.totalAssignedIPv4Addresses.AddUint64(g.totalAssignedIPv4Addresses, subnet.totalAssignedAddresses)
	g.totalAssignedIPv4AddressesInPools.AddUint64(g.totalAssignedIPv4AddressesInPools, subnet.totalAssignedAddressesInPools)
	g.totalDeclinedIPv4Addresses.AddUint64(g.totalDeclinedIPv4Addresses, subnet.totalDeclinedAddresses)
	g.totalDeclinedIPv4AddressesInPools.AddUint64(g.totalDeclinedIPv4AddressesInPools, subnet.totalDeclinedAddressesInPools)
}

// Add the IPv6 subnet statistics to the global state.
func (g *globalStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	g.totalIPv6Addresses.Add(g.totalIPv6Addresses, subnet.totalAddresses)
	g.totalIPv6AddressesInPools.Add(g.totalIPv6AddressesInPools, subnet.totalAddressesInPools)
	g.totalAssignedIPv6Addresses.Add(g.totalAssignedIPv6Addresses, subnet.totalAssignedAddresses)
	g.totalAssignedIPv6AddressesInPools.Add(g.totalAssignedIPv6AddressesInPools, subnet.totalAssignedAddressesInPools)
	g.totalDeclinedIPv6Addresses.Add(g.totalDeclinedIPv6Addresses, subnet.totalDeclinedAddresses)
	g.totalDeclinedIPv6AddressesInPools.Add(g.totalDeclinedIPv6AddressesInPools, subnet.totalDeclinedAddressesInPools)
	g.totalDelegatedPrefixes.Add(g.totalDelegatedPrefixes, subnet.totalDelegatedPrefixes)
	g.totalDelegatedPrefixesInPools.Add(g.totalDelegatedPrefixesInPools, subnet.totalDelegatedPrefixesInPools)
	g.totalAssignedDelegatedPrefixes.Add(g.totalAssignedDelegatedPrefixes, subnet.totalAssignedDelegatedPrefixes)
	g.totalAssignedDelegatedPrefixesInPools.Add(g.totalAssignedDelegatedPrefixesInPools, subnet.totalAssignedDelegatedPrefixesInPools)
}

// General subnet lease statistics.
// It unifies the IPv4 and IPv6 subnet data.
type subnetStats interface {
	GetAddressUtilization() float64
	GetOutOfPoolAddressUtilization() float64
	GetDelegatedPrefixUtilization() float64
	GetOutOfPoolDelegatedPrefixUtilization() float64
	GetStatistics() dbmodel.Stats
}

// Sum of the subnet statistics from the single shared network.
type sharedNetworkStats struct {
	totalAddresses                        *storkutil.BigCounter
	totalAddressesInPools                 *storkutil.BigCounter
	totalAssignedAddresses                *storkutil.BigCounter
	totalAssignedAddressesInPools         *storkutil.BigCounter
	totalDelegatedPrefixes                *storkutil.BigCounter
	totalDelegatedPrefixesInPools         *storkutil.BigCounter
	totalAssignedDelegatedPrefixes        *storkutil.BigCounter
	totalAssignedDelegatedPrefixesInPools *storkutil.BigCounter
}

// Constructor of the sharedNetworkStats struct with
// all counters set to zero.
func newSharedNetworkStats() *sharedNetworkStats {
	return &sharedNetworkStats{
		totalAddresses:                        storkutil.NewBigCounter(0),
		totalAddressesInPools:                 storkutil.NewBigCounter(0),
		totalAssignedAddresses:                storkutil.NewBigCounter(0),
		totalAssignedAddressesInPools:         storkutil.NewBigCounter(0),
		totalDelegatedPrefixes:                storkutil.NewBigCounter(0),
		totalDelegatedPrefixesInPools:         storkutil.NewBigCounter(0),
		totalAssignedDelegatedPrefixes:        storkutil.NewBigCounter(0),
		totalAssignedDelegatedPrefixesInPools: storkutil.NewBigCounter(0),
	}
}

// Address utilization of the shared network.
func (s *sharedNetworkStats) GetAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return s.totalAssignedAddresses.DivideSafeBy(s.totalAddresses)
}

// Out-of-pool address utilization of the shared network.
func (s *sharedNetworkStats) GetOutOfPoolAddressUtilization() float64 {
	return storkutil.NewBigCounter(0).Subtract(s.totalAssignedAddresses, s.totalAssignedAddressesInPools).
		DivideSafeBy(
			storkutil.NewBigCounter(0).Subtract(s.totalAddresses, s.totalAddressesInPools),
		)
}

// Delegated prefix utilization of the shared network.
func (s *sharedNetworkStats) GetDelegatedPrefixUtilization() float64 {
	return s.totalAssignedDelegatedPrefixes.DivideSafeBy(s.totalDelegatedPrefixes)
}

// Out-of-pool delegated prefix utilization of the shared network.
func (s *sharedNetworkStats) GetOutOfPoolDelegatedPrefixUtilization() float64 {
	return storkutil.NewBigCounter(0).Subtract(s.totalAssignedDelegatedPrefixes, s.totalAssignedDelegatedPrefixesInPools).
		DivideSafeBy(
			storkutil.NewBigCounter(0).Subtract(s.totalDelegatedPrefixes, s.totalDelegatedPrefixesInPools),
		)
}

// Returns set of accumulated statistics from all local subnets belonging to
// a given shared network.
func (s *sharedNetworkStats) GetStatistics() dbmodel.Stats {
	return dbmodel.Stats{
		dbmodel.StatNameTotalNAs:             s.totalAddresses.ConvertToNativeType(),
		dbmodel.StatNameTotalOutOfPoolNAs:    storkutil.NewBigCounter(0).Subtract(s.totalAddresses, s.totalAddressesInPools).ConvertToNativeType(),
		dbmodel.StatNameAssignedNAs:          s.totalAssignedAddresses.ConvertToNativeType(),
		dbmodel.StatNameAssignedOutOfPoolNAs: storkutil.NewBigCounter(0).Subtract(s.totalAssignedAddresses, s.totalAssignedAddressesInPools).ConvertToNativeType(),
		dbmodel.StatNameTotalPDs:             s.totalDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.StatNameTotalOutOfPoolPDs:    storkutil.NewBigCounter(0).Subtract(s.totalDelegatedPrefixes, s.totalDelegatedPrefixesInPools).ConvertToNativeType(),
		dbmodel.StatNameAssignedPDs:          s.totalAssignedDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.StatNameAssignedOutOfPoolPDs: storkutil.NewBigCounter(0).Subtract(s.totalAssignedDelegatedPrefixes, s.totalAssignedDelegatedPrefixesInPools).ConvertToNativeType(),
	}
}

// Add the IPv4 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	s.totalAddresses.AddUint64(s.totalAddresses, subnet.totalAddresses)
	s.totalAddressesInPools.AddUint64(s.totalAddressesInPools, subnet.totalAddressesInPools)
	s.totalAssignedAddresses.AddUint64(s.totalAssignedAddresses, subnet.totalAssignedAddresses)
	s.totalAssignedAddressesInPools.AddUint64(s.totalAssignedAddressesInPools, subnet.totalAssignedAddressesInPools)
}

// Add the IPv6 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	s.totalAddresses.Add(s.totalAddresses, subnet.totalAddresses)
	s.totalAddressesInPools.Add(s.totalAddressesInPools, subnet.totalAddressesInPools)
	s.totalAssignedAddresses.Add(s.totalAssignedAddresses, subnet.totalAssignedAddresses)
	s.totalAssignedAddressesInPools.Add(s.totalAssignedAddressesInPools, subnet.totalAssignedAddressesInPools)
	s.totalDelegatedPrefixes.Add(s.totalDelegatedPrefixes, subnet.totalDelegatedPrefixes)
	s.totalDelegatedPrefixesInPools.Add(s.totalDelegatedPrefixesInPools, subnet.totalDelegatedPrefixesInPools)
	s.totalAssignedDelegatedPrefixes.Add(s.totalAssignedDelegatedPrefixes, subnet.totalAssignedDelegatedPrefixes)
	s.totalAssignedDelegatedPrefixesInPools.Add(s.totalAssignedDelegatedPrefixesInPools, subnet.totalAssignedDelegatedPrefixesInPools)
}

// IPv4 statistics retrieved from the single subnet.
type subnetIPv4Stats struct {
	totalAddresses                uint64
	totalAddressesInPools         uint64
	totalAssignedAddresses        uint64
	totalAssignedAddressesInPools uint64
	totalDeclinedAddresses        uint64
	totalDeclinedAddressesInPools uint64
}

// Return the address utilization for a single IPv4 subnet.
func (s *subnetIPv4Stats) GetAddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	if s.totalAddresses == 0 {
		return 0
	}
	return float64(s.totalAssignedAddresses) / float64(s.totalAddresses)
}

// Returns the out-of-pool address utilization for a single IPv4 subnet.
func (s *subnetIPv4Stats) GetOutOfPoolAddressUtilization() float64 {
	if s.totalAddresses == s.totalAddressesInPools {
		return 0.0
	}

	return float64(s.totalAssignedAddresses-s.totalAssignedAddressesInPools) /
		float64(s.totalAddresses-s.totalAddressesInPools)
}

// Return the delegated prefix utilization for a single IPv4 subnet.
// It's always zero because the delegated prefix doesn't apply to IPv4.
func (s *subnetIPv4Stats) GetDelegatedPrefixUtilization() float64 {
	return 0.0
}

// Returns the out-of-pool delegated prefix utilization for a single IPv4 subnet.
// It's always zero because the delegated prefix doesn't apply to IPv4.
func (s *subnetIPv4Stats) GetOutOfPoolDelegatedPrefixUtilization() float64 {
	return 0.0
}

// Returns set of accumulated statistics from all local subnets belonging to
// a given IPv4 subnet.
func (s *subnetIPv4Stats) GetStatistics() dbmodel.Stats {
	return dbmodel.Stats{
		dbmodel.StatNameTotalAddresses:             s.totalAddresses,
		dbmodel.StatNameTotalOutOfPoolAddresses:    s.totalAddresses - s.totalAddressesInPools,
		dbmodel.StatNameAssignedAddresses:          s.totalAssignedAddresses,
		dbmodel.StatNameAssignedOutOfPoolAddresses: s.totalAssignedAddresses - s.totalAssignedAddressesInPools,
		dbmodel.StatNameDeclinedAddresses:          s.totalDeclinedAddresses,
		dbmodel.StatNameDeclinedOutOfPoolAddresses: s.totalDeclinedAddresses - s.totalDeclinedAddressesInPools,
	}
}

// IPv6 statistics retrieved from the single subnet.
type subnetIPv6Stats struct {
	totalAddresses                        *storkutil.BigCounter
	totalAddressesInPools                 *storkutil.BigCounter
	totalAssignedAddresses                *storkutil.BigCounter
	totalAssignedAddressesInPools         *storkutil.BigCounter
	totalDeclinedAddresses                *storkutil.BigCounter
	totalDeclinedAddressesInPools         *storkutil.BigCounter
	totalDelegatedPrefixes                *storkutil.BigCounter
	totalDelegatedPrefixesInPools         *storkutil.BigCounter
	totalAssignedDelegatedPrefixes        *storkutil.BigCounter
	totalAssignedDelegatedPrefixesInPools *storkutil.BigCounter
}

// Return the IPv6 address utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) GetAddressUtilization() float64 {
	// The assigned addresses include the declined ones that aren't reclaimed yet.
	return s.totalAssignedAddresses.DivideSafeBy(s.totalAddresses)
}

// Returns the out-of-pool IPv6 address utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) GetOutOfPoolAddressUtilization() float64 {
	return storkutil.NewBigCounter(0).Subtract(s.totalAssignedAddresses, s.totalAssignedAddressesInPools).
		DivideSafeBy(
			storkutil.NewBigCounter(0).Subtract(s.totalAddresses, s.totalAddressesInPools),
		)
}

// Return the delegated prefix utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) GetDelegatedPrefixUtilization() float64 {
	return s.totalAssignedDelegatedPrefixes.DivideSafeBy(s.totalDelegatedPrefixes)
}

// Returns the out-of-pool delegated prefix utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) GetOutOfPoolDelegatedPrefixUtilization() float64 {
	return storkutil.NewBigCounter(0).Subtract(s.totalAssignedDelegatedPrefixes, s.totalAssignedDelegatedPrefixesInPools).
		DivideSafeBy(
			storkutil.NewBigCounter(0).Subtract(s.totalDelegatedPrefixes, s.totalDelegatedPrefixesInPools),
		)
}

// Returns set of accumulated statistics from all local subnets belonging to
// a given IPv6 network.
func (s *subnetIPv6Stats) GetStatistics() dbmodel.Stats {
	return dbmodel.Stats{
		dbmodel.StatNameTotalNAs:             s.totalAddresses.ConvertToNativeType(),
		dbmodel.StatNameTotalOutOfPoolNAs:    storkutil.NewBigCounter(0).Subtract(s.totalAddresses, s.totalAddressesInPools).ConvertToNativeType(),
		dbmodel.StatNameAssignedNAs:          s.totalAssignedAddresses.ConvertToNativeType(),
		dbmodel.StatNameAssignedOutOfPoolNAs: storkutil.NewBigCounter(0).Subtract(s.totalAssignedAddresses, s.totalAssignedAddressesInPools).ConvertToNativeType(),
		dbmodel.StatNameDeclinedNAs:          s.totalDeclinedAddresses.ConvertToNativeType(),
		dbmodel.StatNameDeclinedOutOfPoolNAs: storkutil.NewBigCounter(0).Subtract(s.totalDeclinedAddresses, s.totalDeclinedAddressesInPools).ConvertToNativeType(),
		dbmodel.StatNameTotalPDs:             s.totalDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.StatNameTotalOutOfPoolPDs:    storkutil.NewBigCounter(0).Subtract(s.totalDelegatedPrefixes, s.totalDelegatedPrefixesInPools).ConvertToNativeType(),
		dbmodel.StatNameAssignedPDs:          s.totalAssignedDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.StatNameAssignedOutOfPoolPDs: storkutil.NewBigCounter(0).Subtract(s.totalAssignedDelegatedPrefixes, s.totalAssignedDelegatedPrefixesInPools).ConvertToNativeType(),
	}
}

// Contains various out-of-pool numbers that need to be added to the counted
// statistics.
//
// The total IPv4 and IPv6 address and delegated prefix statistics returned by
// Kea exclude out-of-pool reservations, yielding possibly incorrect
// calculations.
// The values can be corrected by including the out-of-pool reservation counts
// from the Stork database.
type outOfPoolShifts struct {
	outOfPoolAddresses       map[int64]uint64 // Subnet ID to out-of-pool addresses. IPv6 and IPv4 both.
	outOfPoolPrefixes        map[int64]uint64 // Subnet ID to out-of-pool prefixes.
	outOfPoolGlobalAddresses uint64           // Global out-of-pool addresses. IPv4 only.
	outOfPoolGlobalNAs       uint64           // Global out-of-pool network addresses. IPv6 only.
	outOfPoolGlobalPrefixes  uint64           // Global out-of-pool prefixes.
}

// Statistics Counter is a helper for calculating the global IPv4 and IPv6
// address, and delegated prefix statistics per subnet and shared network.
type statisticsCounter struct {
	global          *globalStats
	sharedNetworks  map[int64]*sharedNetworkStats
	outOfPoolShifts outOfPoolShifts
	excludedDaemons map[int64]bool
}

// Constructor of the statistics counter.
func newStatisticsCounter() *statisticsCounter {
	return &statisticsCounter{
		sharedNetworks: make(map[int64]*sharedNetworkStats),
		global:         newGlobalStats(),
		outOfPoolShifts: outOfPoolShifts{
			outOfPoolAddresses: make(map[int64]uint64),
			outOfPoolPrefixes:  make(map[int64]uint64),
		},
	}
}

// The total IPv4 and IPv6 address and delegated prefix statistics returned by
// Kea exclude out-of-pool reservations, yielding possibly incorrect
// calculations.
// The values can be corrected by including the out-of-pool
// reservation counts from the Stork database.
func (c *statisticsCounter) setOutOfPoolShifts(shifts outOfPoolShifts) {
	c.outOfPoolShifts = shifts
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

	outOfPoolAddresses, ok := c.outOfPoolShifts.outOfPoolAddresses[subnet.ID]
	if !ok {
		outOfPoolAddresses = 0
	}

	if subnet.GetFamily() == 4 {
		return c.addIPv4Subnet(subnet, outOfPoolAddresses)
	}

	outOfPoolPrefixes, ok := c.outOfPoolShifts.outOfPoolPrefixes[subnet.ID]
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
		totalAddresses:                sumStatLocalSubnetsIPv4(subnet, dbmodel.StatNameTotalAddresses, c.excludedDaemons) + outOfPool,
		totalAddressesInPools:         sumStatAddressPoolsIPv4(subnet, dbmodel.StatNameTotalAddresses, c.excludedDaemons),
		totalAssignedAddresses:        sumStatLocalSubnetsIPv4(subnet, dbmodel.StatNameAssignedAddresses, c.excludedDaemons),
		totalAssignedAddressesInPools: sumStatAddressPoolsIPv4(subnet, dbmodel.StatNameAssignedAddresses, c.excludedDaemons),
		totalDeclinedAddresses:        sumStatLocalSubnetsIPv4(subnet, dbmodel.StatNameDeclinedAddresses, c.excludedDaemons),
		totalDeclinedAddressesInPools: sumStatAddressPoolsIPv4(subnet, dbmodel.StatNameDeclinedAddresses, c.excludedDaemons),
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
		totalAddresses:                        sumStatLocalSubnetsIPv6(subnet, dbmodel.StatNameTotalNAs, c.excludedDaemons),
		totalAddressesInPools:                 sumStatAddressPoolsIPv6(subnet, dbmodel.StatNameTotalNAs, c.excludedDaemons),
		totalAssignedAddresses:                sumStatLocalSubnetsIPv6(subnet, dbmodel.StatNameAssignedNAs, c.excludedDaemons),
		totalAssignedAddressesInPools:         sumStatAddressPoolsIPv6(subnet, dbmodel.StatNameAssignedNAs, c.excludedDaemons),
		totalDeclinedAddresses:                sumStatLocalSubnetsIPv6(subnet, dbmodel.StatNameDeclinedNAs, c.excludedDaemons),
		totalDeclinedAddressesInPools:         sumStatAddressPoolsIPv6(subnet, dbmodel.StatNameDeclinedNAs, c.excludedDaemons),
		totalDelegatedPrefixes:                sumStatLocalSubnetsIPv6(subnet, dbmodel.StatNameTotalPDs, c.excludedDaemons),
		totalDelegatedPrefixesInPools:         sumStatPrefixPoolsIPv6(subnet, dbmodel.StatNameTotalPDs, c.excludedDaemons),
		totalAssignedDelegatedPrefixes:        sumStatLocalSubnetsIPv6(subnet, dbmodel.StatNameAssignedPDs, c.excludedDaemons),
		totalAssignedDelegatedPrefixesInPools: sumStatPrefixPoolsIPv6(subnet, dbmodel.StatNameAssignedPDs, c.excludedDaemons),
	}

	stats.totalAddresses.AddUint64(stats.totalAddresses, outOfPoolTotalAddresses)
	stats.totalDelegatedPrefixes.AddUint64(stats.totalDelegatedPrefixes, outOfPoolDelegatedPrefixes)

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

		sum.Add(sum, value)
	}
	return sum
}

// Return the sum of specific statistics for each address pool in each local
// subnet in the provided subnet.
// It expects that the counting value may exceed uint64 range.
// The local subnets that belong to excluded daemons will not be processed.
func sumStatAddressPoolsIPv6(subnet *dbmodel.Subnet, statName string, excludedDaemons map[int64]bool) *storkutil.BigCounter {
	sum := storkutil.NewBigCounter(0)
	for _, localSubnet := range subnet.LocalSubnets {
		if _, ok := excludedDaemons[localSubnet.DaemonID]; ok {
			continue
		}

		seenIDs := map[int64]bool{} // To avoid double counting of pools with the same ID.
		for _, pool := range localSubnet.AddressPools {
			poolID := pool.KeaParameters.PoolID
			if _, ok := seenIDs[poolID]; ok {
				continue // Skip already seen pool ID.
			}
			seenIDs[poolID] = true

			value := pool.Stats.GetBigCounter(statName)
			if value == nil {
				continue
			}

			sum.Add(sum, value)
		}
	}
	return sum
}

// Return the sum of specific statistics for each prefix pool in each local
// subnet in the provided subnet.
// It expects that the counting value may exceed uint64 range.
// The local subnets that belong to excluded daemons will not be processed.
func sumStatPrefixPoolsIPv6(subnet *dbmodel.Subnet, statName string, excludedDaemons map[int64]bool) *storkutil.BigCounter {
	sum := storkutil.NewBigCounter(0)
	for _, localSubnet := range subnet.LocalSubnets {
		if _, ok := excludedDaemons[localSubnet.DaemonID]; ok {
			continue
		}

		seenIDs := map[int64]bool{} // To avoid double counting of pools with the same ID.
		for _, pool := range localSubnet.PrefixPools {
			poolID := pool.KeaParameters.PoolID
			if _, ok := seenIDs[poolID]; ok {
				continue // Skip already seen pool ID.
			}
			seenIDs[poolID] = true

			value := pool.Stats.GetBigCounter(statName)
			if value == nil {
				continue
			}

			sum.Add(sum, value)
		}
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

// Returns the sum of specific statistics for each address pool in each local
// subnets in the provided local subnet.
// It assumes that the counting value does not exceed uint64 range.
func sumStatAddressPoolsIPv4(subnet *dbmodel.Subnet, statName string, excludedDaemons map[int64]bool) uint64 {
	sum := uint64(0)
	for _, localSubnet := range subnet.LocalSubnets {
		if _, ok := excludedDaemons[localSubnet.DaemonID]; ok {
			continue
		}

		seenIDs := map[int64]bool{} // To avoid double counting of pools with the same ID.
		for _, pool := range localSubnet.AddressPools {
			poolID := pool.KeaParameters.PoolID
			if _, ok := seenIDs[poolID]; ok {
				continue // Skip already seen pool ID.
			}
			seenIDs[poolID] = true

			value := pool.Stats.GetBigCounter(statName)
			if value == nil {
				continue
			}
			valueUint, ok := value.ToUint64()
			if !ok {
				continue
			}

			sum += valueUint
		}
	}
	return sum
}

// Returns the global statistics.
func (c *statisticsCounter) GetStatistics() dbmodel.Stats {
	stats := dbmodel.Stats{}

	// Calculate the total number of IPv4 addresses.
	totalIPv4Addresses := storkutil.NewBigCounter(0).AddUint64(c.global.totalIPv4Addresses, c.outOfPoolShifts.outOfPoolGlobalAddresses)
	stats[dbmodel.StatNameTotalAddresses] = totalIPv4Addresses.ConvertToNativeType()
	// Calculate the total number of out-of-pool IPv4 addresses.
	totalOutOfPoolIPv4Addresses := storkutil.NewBigCounter(0).Subtract(totalIPv4Addresses, c.global.totalIPv4AddressesInPools)
	stats[dbmodel.StatNameTotalOutOfPoolAddresses] = totalOutOfPoolIPv4Addresses.ConvertToNativeType()

	// Calculate the total number of assigned IPv4 addresses.
	assignedIPv4Addresses := c.global.totalAssignedIPv4Addresses
	stats[dbmodel.StatNameAssignedAddresses] = assignedIPv4Addresses.ConvertToNativeType()
	// Calculate the total number of assigned out-of-pool IPv4 addresses.
	assignedOutOfPoolIPv4Addresses := storkutil.NewBigCounter(0).Subtract(assignedIPv4Addresses, c.global.totalAssignedIPv4AddressesInPools)
	stats[dbmodel.StatNameAssignedOutOfPoolAddresses] = assignedOutOfPoolIPv4Addresses.ConvertToNativeType()

	// Calculate the total number of declined IPv4 addresses.
	declinedIPv4Addresses := c.global.totalDeclinedIPv4Addresses
	stats[dbmodel.StatNameDeclinedAddresses] = declinedIPv4Addresses.ConvertToNativeType()
	// Calculate the total number of declined out-of-pool IPv4 addresses.
	declinedOutOfPoolIPv4Addresses := storkutil.NewBigCounter(0).Subtract(declinedIPv4Addresses, c.global.totalDeclinedIPv4AddressesInPools)
	stats[dbmodel.StatNameDeclinedOutOfPoolAddresses] = declinedOutOfPoolIPv4Addresses.ConvertToNativeType()

	// Calculate the total number of IPv6 addresses.
	totalIPv6Addresses := storkutil.NewBigCounter(0).AddUint64(c.global.totalIPv6Addresses, c.outOfPoolShifts.outOfPoolGlobalNAs)
	stats[dbmodel.StatNameTotalNAs] = totalIPv6Addresses.ConvertToNativeType()
	// Calculate the total number of out-of-pool IPv6 addresses.
	totalOutOfPoolIPv6Addresses := storkutil.NewBigCounter(0).Subtract(totalIPv6Addresses, c.global.totalIPv6AddressesInPools)
	stats[dbmodel.StatNameTotalOutOfPoolNAs] = totalOutOfPoolIPv6Addresses.ConvertToNativeType()

	// Calculate the total number of assigned IPv6 addresses.
	assignedIPv6Addresses := c.global.totalAssignedIPv6Addresses
	stats[dbmodel.StatNameAssignedNAs] = assignedIPv6Addresses.ConvertToNativeType()
	// Calculate the total number of assigned out-of-pool IPv6 addresses.
	assignedOutOfPoolIPv6Addresses := storkutil.NewBigCounter(0).Subtract(assignedIPv6Addresses, c.global.totalAssignedIPv6AddressesInPools)
	stats[dbmodel.StatNameAssignedOutOfPoolNAs] = assignedOutOfPoolIPv6Addresses.ConvertToNativeType()

	// Calculate the total number of declined IPv6 addresses.
	declinedIPv6Addresses := c.global.totalDeclinedIPv6Addresses
	stats[dbmodel.StatNameDeclinedNAs] = declinedIPv6Addresses.ConvertToNativeType()
	// Calculate the total number of declined out-of-pool IPv6 addresses.
	declinedOutOfPoolIPv6Addresses := storkutil.NewBigCounter(0).Subtract(declinedIPv6Addresses, c.global.totalDeclinedIPv6AddressesInPools)
	stats[dbmodel.StatNameDeclinedOutOfPoolNAs] = declinedOutOfPoolIPv6Addresses.ConvertToNativeType()

	// Calculate the total number of delegated prefixes.
	totalDelegatedPrefixes := storkutil.NewBigCounter(0).AddUint64(c.global.totalDelegatedPrefixes, c.outOfPoolShifts.outOfPoolGlobalPrefixes)
	stats[dbmodel.StatNameTotalPDs] = totalDelegatedPrefixes.ConvertToNativeType()
	// Calculate the total number of out-of-pool delegated prefixes.
	totalOutOfPoolDelegatedPrefixes := storkutil.NewBigCounter(0).Subtract(totalDelegatedPrefixes, c.global.totalDelegatedPrefixesInPools)
	stats[dbmodel.StatNameTotalOutOfPoolPDs] = totalOutOfPoolDelegatedPrefixes.ConvertToNativeType()

	// Calculate the total number of assigned delegated prefixes.
	assignedDelegatedPrefixes := c.global.totalAssignedDelegatedPrefixes
	stats[dbmodel.StatNameAssignedPDs] = assignedDelegatedPrefixes.ConvertToNativeType()
	// Calculate the total number of assigned out-of-pool delegated prefixes.
	assignedOutOfPoolDelegatedPrefixes := storkutil.NewBigCounter(0).Subtract(assignedDelegatedPrefixes, c.global.totalAssignedDelegatedPrefixesInPools)
	stats[dbmodel.StatNameAssignedOutOfPoolPDs] = assignedOutOfPoolDelegatedPrefixes.ConvertToNativeType()

	return stats
}
