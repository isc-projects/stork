package kea

import dbmodel "isc.org/stork/server/database/model"

type globalStats struct {
	totalAddresses         int64
	totalAssignedAddresses int64
	totalDeclinedAddresses int64
	totalNAs               int64
	totalAssignedNAs       int64
	totalDeclinedNAs       int64
	totalPDs               int64
	totalAssignedPDs       int64
}

func (g *globalStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	g.totalAddresses += subnet.totalAddresses
	g.totalAssignedAddresses += subnet.totalAssignedAddresses
	g.totalDeclinedAddresses += subnet.totalDeclinedAddresses
}

func (g *globalStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	g.totalNAs += subnet.totalNAs
	g.totalAssignedNAs += subnet.totalAssignedNAs
	g.totalDeclinedNAs += subnet.totalDeclinedNAs
	g.totalPDs += subnet.totalPDs
	g.totalAssignedPDs += subnet.totalAssignedPDs
}

type utilization interface {
	addressUtilization() float64
	pdUtilization() float64
}

type sharedNetworkStats struct {
	totalAddresses         int64
	totalAssignedAddresses int64
	totalPDs               int64
	totalAssignedPDs       int64
}

func (s *sharedNetworkStats) addressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return safeFloatingDiv(s.totalAssignedAddresses, s.totalAddresses)
}

func (s *sharedNetworkStats) pdUtilization() float64 {
	// The assigned pds includes the declined pds that aren't reclaimed yet.
	return safeFloatingDiv(s.totalAssignedPDs, s.totalPDs)
}

func (s *sharedNetworkStats) addIPv4Subnet(subnet *subnetIPv4Stats) {
	s.totalAddresses += subnet.totalAddresses
	s.totalAssignedAddresses += subnet.totalAssignedAddresses
}

func (s *sharedNetworkStats) addIPv6Subnet(subnet *subnetIPv6Stats) {
	s.totalAddresses += subnet.totalNAs
	s.totalAssignedAddresses += subnet.totalAssignedNAs
	s.totalPDs += subnet.totalPDs
	s.totalAssignedPDs += subnet.totalAssignedPDs
}

type subnetIPv4Stats struct {
	totalAddresses         int64
	totalAssignedAddresses int64
	totalDeclinedAddresses int64
}

func (s *subnetIPv4Stats) addressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return safeFloatingDiv(s.totalAssignedAddresses, s.totalAddresses)
}

func (s *subnetIPv4Stats) pdUtilization() float64 {
	return 0.0
}

type subnetIPv6Stats struct {
	totalNAs         int64
	totalAssignedNAs int64
	totalDeclinedNAs int64
	totalPDs         int64
	totalAssignedPDs int64
}

func (s *subnetIPv6Stats) addressUtilization() float64 {
	// The assigned NAs include the declined nas that aren't reclaimed yet.
	return safeFloatingDiv(s.totalAssignedNAs, s.totalNAs)
}

func (s *subnetIPv6Stats) pdUtilization() float64 {
	// The assigned pds includes the declined pds that aren't reclaimed yet.
	return safeFloatingDiv(s.totalAssignedPDs, s.totalPDs)
}

type utilizationCalculator struct {
	global         globalStats
	sharedNetworks map[int64]*sharedNetworkStats
}

func newUtilizationCalculator() *utilizationCalculator {
	return &utilizationCalculator{
		sharedNetworks: make(map[int64]*sharedNetworkStats),
	}
}

func (c *utilizationCalculator) add(subnet *dbmodel.Subnet) utilization {
	if subnet.SharedNetworkID != 0 {
		_, ok := c.sharedNetworks[subnet.SharedNetworkID]
		if !ok {
			c.sharedNetworks[subnet.SharedNetworkID] = &sharedNetworkStats{}
		}
	}

	if subnet.GetFamily() == 6 {
		return c.addIPv6Subnet(subnet)
	}
	return c.addIPv4Subnet(subnet)
}

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

func sumStatLocalSubnets(subnet *dbmodel.Subnet, statName string) int64 {
	var sum int64 = 0
	for _, localSubnet := range subnet.LocalSubnets {
		sum += getLocalSubnetStatValueIntOrDefault(localSubnet, statName)
	}
	return sum
}

func getLocalSubnetStatValueIntOrDefault(localSubnet *dbmodel.LocalSubnet, name string) int64 {
	value, ok := localSubnet.Stats[name]
	if !ok {
		return 0
	}

	valueInt, ok := value.(float64)
	if !ok {
		return 0
	}

	return int64(valueInt)
}

func safeFloatingDiv(a, b int64) float64 {
	if b == 0.0 {
		return 0.0
	}
	return float64(a) / float64(b)
}
