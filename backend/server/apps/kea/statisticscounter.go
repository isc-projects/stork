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
	g.totalIPv4Addresses.AddUint64(subnet.totalAddresses)
	g.totalIPv4AddressesInPools.AddUint64(subnet.totalAddressesInPools)
	g.totalAssignedIPv4Addresses.AddUint64(subnet.totalAssignedAddresses)
	g.totalAssignedIPv4AddressesInPools.AddUint64(subnet.totalAssignedAddressesInPools)
	g.totalDeclinedIPv4Addresses.AddUint64(subnet.totalDeclinedAddresses)
	g.totalDeclinedIPv4AddressesInPools.AddUint64(subnet.totalDeclinedAddressesInPools)
}

// Add the IPv6 subnet statistics to the global state.
func (g *globalStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	g.totalIPv6Addresses.Add(subnet.totalAddresses)
	g.totalIPv6AddressesInPools.Add(subnet.totalAddressesInPools)
	g.totalAssignedIPv6Addresses.Add(subnet.totalAssignedAddresses)
	g.totalAssignedIPv6AddressesInPools.Add(subnet.totalAssignedAddressesInPools)
	g.totalDeclinedIPv6Addresses.Add(subnet.totalDeclinedAddresses)
	g.totalDeclinedIPv6AddressesInPools.Add(subnet.totalDeclinedAddressesInPools)
	g.totalDelegatedPrefixes.Add(subnet.totalDelegatedPrefixes)
	g.totalDelegatedPrefixesInPools.Add(subnet.totalDelegatedPrefixesInPools)
	g.totalAssignedDelegatedPrefixes.Add(subnet.totalAssignedDelegatedPrefixes)
	g.totalAssignedDelegatedPrefixesInPools.Add(subnet.totalAssignedDelegatedPrefixesInPools)
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

// Delegated prefix utilization of the shared network.
func (s *sharedNetworkStats) GetDelegatedPrefixUtilization() float64 {
	return s.totalAssignedDelegatedPrefixes.DivideSafeBy(s.totalDelegatedPrefixes)
}

// Returns set of accumulated statistics from all local subnets belonging to
// a given shared network.
func (s *sharedNetworkStats) GetStatistics() dbmodel.SubnetStats {
	return dbmodel.SubnetStats{
		dbmodel.SubnetStatsNameTotalNAs:             s.totalAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameTotalOutOfPoolNAs:    s.totalAddresses.Subtract(s.totalAddressesInPools).ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedNAs:          s.totalAssignedAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedOutOfPoolNAs: s.totalAssignedAddresses.Subtract(s.totalAssignedAddressesInPools).ConvertToNativeType(),
		dbmodel.SubnetStatsNameTotalPDs:             s.totalDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.SubnetStatsNameTotalOutOfPoolPDs:    s.totalDelegatedPrefixes.Subtract(s.totalDelegatedPrefixesInPools).ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedPDs:          s.totalAssignedDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedOutOfPoolPDs: s.totalAssignedDelegatedPrefixes.Subtract(s.totalAssignedDelegatedPrefixesInPools).ConvertToNativeType(),
	}
}

// Add the IPv4 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	s.totalAddresses.AddUint64(subnet.totalAddresses)
	s.totalAddressesInPools.AddUint64(subnet.totalAddressesInPools)
	s.totalAssignedAddresses.AddUint64(subnet.totalAssignedAddresses)
	s.totalAssignedAddressesInPools.AddUint64(subnet.totalAssignedAddressesInPools)
}

// Add the IPv6 subnet statistics to the shared network state.
func (s *sharedNetworkStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	s.totalAddresses.Add(subnet.totalAddresses)
	s.totalAddressesInPools.Add(subnet.totalAddressesInPools)
	s.totalAssignedAddresses.Add(subnet.totalAssignedAddresses)
	s.totalAssignedAddressesInPools.Add(subnet.totalAssignedAddressesInPools)
	s.totalDelegatedPrefixes.Add(subnet.totalDelegatedPrefixes)
	s.totalDelegatedPrefixesInPools.Add(subnet.totalDelegatedPrefixesInPools)
	s.totalAssignedDelegatedPrefixes.Add(subnet.totalAssignedDelegatedPrefixes)
	s.totalAssignedDelegatedPrefixesInPools.Add(subnet.totalAssignedDelegatedPrefixesInPools)
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

// Return the delegated prefix utilization for a single IPv4 subnet.
// It's always zero because the delegated prefix doesn't apply to IPv4.
func (s *subnetIPv4Stats) GetDelegatedPrefixUtilization() float64 {
	return 0.0
}

// Returns set of accumulated statistics from all local subnets belonging to
// a given IPv4 subnet.
func (s *subnetIPv4Stats) GetStatistics() dbmodel.SubnetStats {
	return dbmodel.SubnetStats{
		dbmodel.SubnetStatsNameTotalAddresses:             s.totalAddresses,
		dbmodel.SubnetStatsNameTotalOutOfPoolAddresses:    s.totalAddresses - s.totalAddressesInPools,
		dbmodel.SubnetStatsNameAssignedAddresses:          s.totalAssignedAddresses,
		dbmodel.SubnetStatsNameAssignedOutOfPoolAddresses: s.totalAssignedAddresses - s.totalAssignedAddressesInPools,
		dbmodel.SubnetStatsNameDeclinedAddresses:          s.totalDeclinedAddresses,
		dbmodel.SubnetStatsNameDeclinedOutOfPoolAddresses: s.totalDeclinedAddresses - s.totalDeclinedAddressesInPools,
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

// Return the delegated prefix utilization for a single IPv6 subnet.
func (s *subnetIPv6Stats) GetDelegatedPrefixUtilization() float64 {
	return s.totalAssignedDelegatedPrefixes.DivideSafeBy(s.totalDelegatedPrefixes)
}

// Returns set of accumulated statistics from all local subnets belonging to
// a given IPv6 network.
func (s *subnetIPv6Stats) GetStatistics() dbmodel.SubnetStats {
	return dbmodel.SubnetStats{
		dbmodel.SubnetStatsNameTotalNAs:             s.totalAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameTotalOutOfPoolNAs:    s.totalAddresses.Subtract(s.totalAddressesInPools).ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedNAs:          s.totalAssignedAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedOutOfPoolNAs: s.totalAssignedAddresses.Subtract(s.totalAssignedAddressesInPools).ConvertToNativeType(),
		dbmodel.SubnetStatsNameDeclinedNAs:          s.totalDeclinedAddresses.ConvertToNativeType(),
		dbmodel.SubnetStatsNameDeclinedOutOfPoolNAs: s.totalDeclinedAddresses.Subtract(s.totalDeclinedAddressesInPools).ConvertToNativeType(),
		dbmodel.SubnetStatsNameTotalPDs:             s.totalDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.SubnetStatsNameTotalOutOfPoolPDs:    s.totalDelegatedPrefixes.Subtract(s.totalDelegatedPrefixesInPools).ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedPDs:          s.totalAssignedDelegatedPrefixes.ConvertToNativeType(),
		dbmodel.SubnetStatsNameAssignedOutOfPoolPDs: s.totalAssignedDelegatedPrefixes.Subtract(s.totalAssignedDelegatedPrefixesInPools).ConvertToNativeType(),
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
		totalAddresses:                sumStatLocalSubnetsIPv4(subnet, dbmodel.SubnetStatsNameTotalAddresses, c.excludedDaemons) + outOfPool,
		totalAddressesInPools:         sumStatAddressPoolsIPv4(subnet, dbmodel.SubnetStatsNameTotalAddresses, c.excludedDaemons),
		totalAssignedAddresses:        sumStatLocalSubnetsIPv4(subnet, dbmodel.SubnetStatsNameAssignedAddresses, c.excludedDaemons),
		totalAssignedAddressesInPools: sumStatAddressPoolsIPv4(subnet, dbmodel.SubnetStatsNameAssignedAddresses, c.excludedDaemons),
		totalDeclinedAddresses:        sumStatLocalSubnetsIPv4(subnet, dbmodel.SubnetStatsNameDeclinedAddresses, c.excludedDaemons),
		totalDeclinedAddressesInPools: sumStatAddressPoolsIPv4(subnet, dbmodel.SubnetStatsNameDeclinedAddresses, c.excludedDaemons),
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
		totalAddresses:                        sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameTotalNAs, c.excludedDaemons).AddUint64(outOfPoolTotalAddresses),
		totalAddressesInPools:                 sumStatAddressPoolsIPv6(subnet, dbmodel.SubnetStatsNameTotalNAs, c.excludedDaemons),
		totalAssignedAddresses:                sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameAssignedNAs, c.excludedDaemons),
		totalAssignedAddressesInPools:         sumStatAddressPoolsIPv6(subnet, dbmodel.SubnetStatsNameAssignedNAs, c.excludedDaemons),
		totalDeclinedAddresses:                sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameDeclinedNAs, c.excludedDaemons),
		totalDeclinedAddressesInPools:         sumStatAddressPoolsIPv6(subnet, dbmodel.SubnetStatsNameDeclinedNAs, c.excludedDaemons),
		totalDelegatedPrefixes:                sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameTotalPDs, c.excludedDaemons).AddUint64(outOfPoolDelegatedPrefixes),
		totalDelegatedPrefixesInPools:         sumStatPrefixPoolsIPv6(subnet, dbmodel.SubnetStatsNameTotalPDs, c.excludedDaemons),
		totalAssignedDelegatedPrefixes:        sumStatLocalSubnetsIPv6(subnet, dbmodel.SubnetStatsNameAssignedPDs, c.excludedDaemons),
		totalAssignedDelegatedPrefixesInPools: sumStatPrefixPoolsIPv6(subnet, dbmodel.SubnetStatsNameAssignedPDs, c.excludedDaemons),
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

		for _, pool := range localSubnet.AddressPools {
			value := pool.Stats.GetBigCounter(statName)
			if value == nil {
				continue
			}

			sum.Add(value)
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

		for _, pool := range localSubnet.PrefixPools {
			value := pool.Stats.GetBigCounter(statName)
			if value == nil {
				continue
			}

			sum.Add(value)
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

		for _, pool := range localSubnet.AddressPools {
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
func (c *statisticsCounter) GetStatistics() dbmodel.SubnetStats {
	return dbmodel.SubnetStats{
		dbmodel.SubnetStatsNameTotalAddresses:             c.global.totalIPv4Addresses.Clone().AddUint64(c.outOfPoolShifts.outOfPoolGlobalAddresses).ToBigInt(),
		dbmodel.SubnetStatsNameTotalOutOfPoolAddresses:    c.global.totalIPv4Addresses.Clone().AddUint64(c.outOfPoolShifts.outOfPoolGlobalAddresses).Subtract(c.global.totalIPv4AddressesInPools).ToBigInt(),
		dbmodel.SubnetStatsNameAssignedAddresses:          c.global.totalAssignedIPv4Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameAssignedOutOfPoolAddresses: c.global.totalAssignedIPv4Addresses.Clone().Subtract(c.global.totalAssignedIPv4AddressesInPools).ToBigInt(),
		dbmodel.SubnetStatsNameDeclinedAddresses:          c.global.totalDeclinedIPv4Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameDeclinedOutOfPoolAddresses: c.global.totalDeclinedIPv4Addresses.Subtract(c.global.totalDeclinedIPv4AddressesInPools).ToBigInt(),
		dbmodel.SubnetStatsNameTotalNAs:                   c.global.totalIPv6Addresses.Clone().AddUint64(c.outOfPoolShifts.outOfPoolGlobalNAs).ToBigInt(),
		dbmodel.SubnetStatsNameTotalOutOfPoolNAs:          c.global.totalIPv6Addresses.Clone().AddUint64(c.outOfPoolShifts.outOfPoolGlobalNAs).Subtract(c.global.totalIPv6AddressesInPools).ToBigInt(),
		dbmodel.SubnetStatsNameAssignedNAs:                c.global.totalAssignedIPv6Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameAssignedOutOfPoolNAs:       c.global.totalAssignedIPv6Addresses.Clone().Subtract(c.global.totalAssignedIPv6AddressesInPools).ToBigInt(),
		dbmodel.SubnetStatsNameDeclinedNAs:                c.global.totalDeclinedIPv6Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameDeclinedOutOfPoolNAs:       c.global.totalDeclinedIPv6Addresses.Clone().Subtract(c.global.totalDeclinedIPv6AddressesInPools).ToBigInt(),
		dbmodel.SubnetStatsNameTotalPDs:                   c.global.totalDelegatedPrefixes.Clone().AddUint64(c.outOfPoolShifts.outOfPoolGlobalPrefixes).ToBigInt(),
		dbmodel.SubnetStatsNameTotalOutOfPoolPDs:          c.global.totalDelegatedPrefixes.Clone().AddUint64(c.outOfPoolShifts.outOfPoolGlobalPrefixes).Subtract(c.global.totalDelegatedPrefixesInPools).ToBigInt(),
		dbmodel.SubnetStatsNameAssignedPDs:                c.global.totalAssignedDelegatedPrefixes.ToBigInt(),
		dbmodel.SubnetStatsNameAssignedOutOfPoolPDs:       c.global.totalAssignedDelegatedPrefixes.Clone().Subtract(c.global.totalAssignedDelegatedPrefixesInPools).ToBigInt(),
	}
}
