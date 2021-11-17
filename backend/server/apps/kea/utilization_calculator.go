package kea

import dbmodel "isc.org/stork/server/database/model"

type globalStats struct {
	TotalAddresses         int64
	TotalAssignedAddresses int64
	TotalDeclinedAddresses int64
	TotalNas               int64
	TotalAssignedNas       int64
	TotalDeclinedNas       int64
	TotalPds               int64
	TotalAssignedPds       int64
}

func (g *globalStats) AddIPv4Subnet(subnet *subnetIPv4Stats) {
	g.TotalAddresses += subnet.TotalAddresses
	g.TotalAssignedAddresses += subnet.TotalAssignedAddresses
	g.TotalDeclinedAddresses += subnet.TotalDeclinedAddresses
}

func (g *globalStats) AddIPv6Subnet(subnet *subnetIPv6Stats) {
	g.TotalNas += subnet.TotalNas
	g.TotalAssignedNas += subnet.TotalAssignedNas
	g.TotalDeclinedNas += subnet.TotalDeclinedNas
	g.TotalPds += subnet.TotalPds
	g.TotalAssignedPds += subnet.TotalAssignedPds
}

type Utilization interface {
	AddressUtilization() float64
	PdUtilization() float64
}

type sharedNetworkStats struct {
	TotalAddresses         int64
	TotalAssignedAddresses int64
	TotalPds               int64
	TotalAssignedPds       int64
}

func (s *sharedNetworkStats) AddressUtilization() float64 {
	return safeFloatingDiv(s.TotalAssignedAddresses, s.TotalAddresses)
}

func (s *sharedNetworkStats) PdUtilization() float64 {
	return safeFloatingDiv(s.TotalAssignedPds, s.TotalPds)
}

func (s *sharedNetworkStats) AddIPv4Subnet(subnet *subnetIPv4Stats) {
	s.TotalAddresses += subnet.TotalAddresses
	s.TotalAssignedAddresses += subnet.TotalAssignedAddresses
}

func (s *sharedNetworkStats) AddIPv6Subnet(subnet *subnetIPv6Stats) {
	s.TotalAddresses += subnet.TotalNas
	s.TotalAssignedAddresses += subnet.TotalAssignedNas
	s.TotalPds += subnet.TotalPds
	s.TotalAssignedPds += subnet.TotalAssignedPds
}

type subnetIPv4Stats struct {
	TotalAddresses         int64
	TotalAssignedAddresses int64
	TotalDeclinedAddresses int64
}

func (s *subnetIPv4Stats) AddressUtilization() float64 {
	return safeFloatingDiv(s.TotalAssignedAddresses, s.TotalAddresses)
}

func (s *subnetIPv4Stats) PdUtilization() float64 {
	return 0.0
}

type subnetIPv6Stats struct {
	TotalNas         int64
	TotalAssignedNas int64
	TotalDeclinedNas int64
	TotalPds         int64
	TotalAssignedPds int64
}

func (s *subnetIPv6Stats) AddressUtilization() float64 {
	return safeFloatingDiv(s.TotalAssignedNas, s.TotalNas)
}

func (s *subnetIPv6Stats) PdUtilization() float64 {
	return safeFloatingDiv(s.TotalAssignedPds, s.TotalPds)
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
		TotalNas:         sumStatLocalSubnets(subnet, "total-nas"),
		TotalAssignedNas: sumStatLocalSubnets(subnet, "assigned-nas"),
		TotalDeclinedNas: sumStatLocalSubnets(subnet, "declined-nas"),
		TotalPds:         sumStatLocalSubnets(subnet, "total-pds"),
		TotalAssignedPds: sumStatLocalSubnets(subnet, "assigned-pds"),
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
