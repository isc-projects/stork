package kea

import dbmodel "isc.org/stork/server/database/model"

type globalStats struct {
	TotalAddresses         int64
	TotalAssignedAddresses int64
	TotalDeclinedAddresses int64
	TotalNAs               int64
	TotalAssignedNAs       int64
	TotalDeclinedNAs       int64
	TotalPDs               int64
	TotalAssignedPDs       int64
}

func (g *globalStats) AddIPv4Subnet(subnet *subnetIPv4Stats) {
	g.TotalAddresses += subnet.TotalAddresses
	g.TotalAssignedAddresses += subnet.TotalAssignedAddresses
	g.TotalDeclinedAddresses += subnet.TotalDeclinedAddresses
}

func (g *globalStats) AddIPv6Subnet(subnet *subnetIPv6Stats) {
	g.TotalNAs += subnet.TotalNAs
	g.TotalAssignedNAs += subnet.TotalAssignedNAs
	g.TotalDeclinedNAs += subnet.TotalDeclinedNAs
	g.TotalPDs += subnet.TotalPDs
	g.TotalAssignedPDs += subnet.TotalAssignedPDs
}

type Utilization interface {
	AddressUtilization() float64
	PDUtilization() float64
}

type sharedNetworkStats struct {
	TotalAddresses         int64
	TotalAssignedAddresses int64
	TotalPDs               int64
	TotalAssignedPDs       int64
}

func (s *sharedNetworkStats) AddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return safeFloatingDiv(s.TotalAssignedAddresses, s.TotalAddresses)
}

func (s *sharedNetworkStats) PDUtilization() float64 {
	// The assigned pds includes the declined pds that aren't reclaimed yet.
	return safeFloatingDiv(s.TotalAssignedPDs, s.TotalPDs)
}

func (s *sharedNetworkStats) AddIPv4Subnet(subnet *subnetIPv4Stats) {
	s.TotalAddresses += subnet.TotalAddresses
	s.TotalAssignedAddresses += subnet.TotalAssignedAddresses
}

func (s *sharedNetworkStats) AddIPv6Subnet(subnet *subnetIPv6Stats) {
	s.TotalAddresses += subnet.TotalNAs
	s.TotalAssignedAddresses += subnet.TotalAssignedNAs
	s.TotalPDs += subnet.TotalPDs
	s.TotalAssignedPDs += subnet.TotalAssignedPDs
}

type subnetIPv4Stats struct {
	TotalAddresses         int64
	TotalAssignedAddresses int64
	TotalDeclinedAddresses int64
}

func (s *subnetIPv4Stats) AddressUtilization() float64 {
	// The assigned addresses include the declined addresses that aren't reclaimed yet.
	return safeFloatingDiv(s.TotalAssignedAddresses, s.TotalAddresses)
}

func (s *subnetIPv4Stats) PDUtilization() float64 {
	return 0.0
}

type subnetIPv6Stats struct {
	TotalNAs         int64
	TotalAssignedNAs int64
	TotalDeclinedNAs int64
	TotalPDs         int64
	TotalAssignedPDs int64
}

func (s *subnetIPv6Stats) AddressUtilization() float64 {
	// The assigned NAs include the declined nas that aren't reclaimed yet.
	return safeFloatingDiv(s.TotalAssignedNAs, s.TotalNAs)
}

func (s *subnetIPv6Stats) PDUtilization() float64 {
	// The assigned pds includes the declined pds that aren't reclaimed yet.
	return safeFloatingDiv(s.TotalAssignedPDs, s.TotalPDs)
}

type UtilizationCalculator struct {
	Global         globalStats
	SharedNetworks map[int64]*sharedNetworkStats
}

func NewUtilizationCalculator() *UtilizationCalculator {
	return &UtilizationCalculator{
		SharedNetworks: make(map[int64]*sharedNetworkStats),
	}
}

func (c *UtilizationCalculator) Add(subnet *dbmodel.Subnet) Utilization {
	if subnet.SharedNetworkID != 0 {
		_, ok := c.SharedNetworks[subnet.SharedNetworkID]
		if !ok {
			c.SharedNetworks[subnet.SharedNetworkID] = &sharedNetworkStats{}
		}
	}

	if subnet.GetFamily() == 6 {
		return c.addIPv6Subnet(subnet)
	}
	return c.addIPv4Subnet(subnet)
}

func (c *UtilizationCalculator) addIPv4Subnet(subnet *dbmodel.Subnet) *subnetIPv4Stats {
	stats := &subnetIPv4Stats{
		TotalAddresses:         sumStatLocalSubnets(subnet, "total-addresses"),
		TotalAssignedAddresses: sumStatLocalSubnets(subnet, "assigned-addresses"),
		TotalDeclinedAddresses: sumStatLocalSubnets(subnet, "declined-addresses"),
	}

	if subnet.SharedNetworkID != 0 {
		c.SharedNetworks[subnet.SharedNetworkID].AddIPv4Subnet(stats)
	}

	c.Global.AddIPv4Subnet(stats)

	return stats
}

func (c *UtilizationCalculator) addIPv6Subnet(subnet *dbmodel.Subnet) *subnetIPv6Stats {
	stats := &subnetIPv6Stats{
		TotalNAs:         sumStatLocalSubnets(subnet, "total-nas"),
		TotalAssignedNAs: sumStatLocalSubnets(subnet, "assigned-nas"),
		TotalDeclinedNAs: sumStatLocalSubnets(subnet, "declined-nas"),
		TotalPDs:         sumStatLocalSubnets(subnet, "total-pds"),
		TotalAssignedPDs: sumStatLocalSubnets(subnet, "assigned-pds"),
	}

	if subnet.SharedNetworkID != 0 {
		c.SharedNetworks[subnet.SharedNetworkID].AddIPv6Subnet(stats)
	}

	c.Global.AddIPv6Subnet(stats)

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
