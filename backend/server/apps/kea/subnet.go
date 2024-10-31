package kea

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keaconfig "isc.org/stork/appcfg/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

// Checks whether the given shared network exists already. It iterates over the
// slice of existing networks. If the network seems to be matching one of them,
// the shared network instance along with all subnets is fetched from the
// database and returned to the caller.
func sharedNetworkExists(dbi dbops.DBI, network *dbmodel.SharedNetwork, existingNetworks []dbmodel.SharedNetwork) (*dbmodel.SharedNetwork, error) {
	for _, existing := range existingNetworks {
		// todo: this is logic should be extended to perform some more sophisticated
		// matching of a shared network with existing shared networks. For now,
		// we only match by the shared network name and we do not resolve any
		// conflicts. This should change soon.
		if existing.Name == network.Name {
			// Get the subnets included in this shared network.
			dbNetwork, err := dbmodel.GetSharedNetworkWithRelations(
				dbi, existing.ID,
				dbmodel.SharedNetworkRelationSubnetsAddressPools,
				dbmodel.SharedNetworkRelationSubnetsPrefixPools,
			)
			if err != nil {
				return nil, err
			}
			return dbNetwork, nil
		}
	}
	return nil, nil
}

// Checks whether the given subnet exists already.
func findMatchingSubnet(subnet *dbmodel.Subnet, existingSubnets *dbmodel.IndexedSubnets) *dbmodel.Subnet {
	// todo: this logic should be extended to perform some more sophisticated
	// matching of the subnet with existing subnets. For now, we only match by
	// the subnet prefix and we do not resolve any conflicts. This should
	// change soon.
	if existingSubnet, ok := existingSubnets.ByPrefix[subnet.Prefix]; ok {
		return existingSubnet
	}
	return nil
}

// Overrides a subnet into the existing database one.
func overrideIntoDatabaseSubnet(dbi dbops.DBI, existingSubnet *dbmodel.Subnet, changedSubnet *dbmodel.Subnet) error {
	// Hosts and local hosts.
	hosts, err := overrideIntoDatabaseHosts(dbi, existingSubnet.ID, changedSubnet.Hosts)
	if err != nil {
		return err
	}
	existingSubnet.Hosts = hosts

	// Client class.
	existingSubnet.ClientClass = changedSubnet.ClientClass

	existingSubnet.Join(changedSubnet)
	return nil
}

func detectSharedNetworks(dbi dbops.DBI, config *dbmodel.KeaConfig, family int, daemon *dbmodel.Daemon, lookup keaconfig.DHCPOptionDefinitionLookup) (networks []dbmodel.SharedNetwork, err error) {
	networksList := config.GetSharedNetworks(false)
	if len(networksList) == 0 {
		return networks, err
	}

	dbNetworks, err := dbmodel.GetAllSharedNetworks(dbi, family)
	if err != nil {
		return networks, err
	}

	for _, n := range networksList {
		var network *dbmodel.SharedNetwork
		network, err = dbmodel.NewSharedNetworkFromKea(n, family, daemon, dbmodel.HostDataSourceConfig, lookup)
		if err != nil {
			log.WithError(err).Warnf("Skipping invalid shared network")
			continue
		}
		var dbNetwork *dbmodel.SharedNetwork
		dbNetwork, err = sharedNetworkExists(dbi, network, dbNetworks)
		if err != nil {
			return []dbmodel.SharedNetwork{}, err
		}
		if dbNetwork != nil {
			// Create indexes for the existing subnets to improve performance of
			// matching new subnets with them.
			indexedSubnets := dbmodel.NewIndexedSubnets(dbNetwork.Subnets)
			if ok := indexedSubnets.Populate(); !ok {
				log.Warnf("Skipping shared network %s; building indexes failed due to duplicates", dbNetwork.Name)

				continue
			}
			networkForUpdate := *dbNetwork
			networkForUpdate.Subnets = []dbmodel.Subnet{}

			// Go over the configured subnets and see if they belong to that
			// shared network already.
			for _, s := range network.Subnets {
				subnet := s
				existingSubnet := findMatchingSubnet(&subnet, indexedSubnets)
				if existingSubnet == nil {
					networkForUpdate.Subnets = append(networkForUpdate.Subnets, subnet)
				} else {
					// Subnet already exists and may contain some updated data. Let's
					// override the data from the new subnet into the existing subnet.
					err := overrideIntoDatabaseSubnet(dbi, existingSubnet, &subnet)
					if err != nil {
						log.WithError(err).Warnf("Skipping subnet %s after override failure",
							subnet.Prefix)
						continue
					}
					networkForUpdate.Subnets = append(networkForUpdate.Subnets, *existingSubnet)
				}
			}
			networkForUpdate.Join(network)
			networks = append(networks, networkForUpdate)
		} else {
			networks = append(networks, *network)
		}
	}
	return networks, nil
}

// For a given Kea configuration it detects the top-level subnets matching
// this configuration. All existing subnets matching the given configuration
// are returned as they are. If there is no match a new subnet instance is
// returned.
func detectSubnets(dbi dbops.DBI, config *dbmodel.KeaConfig, family int, daemon *dbmodel.Daemon, lookup keaconfig.DHCPOptionDefinitionLookup) (subnets []dbmodel.Subnet, err error) {
	subnetList := config.GetSubnets()
	if len(subnetList) == 0 {
		return subnets, err
	}

	// Fetch all global subnets from the database to perform matching. For now
	// it is better to get all of them because this is just a single query rather
	// than many but in the future we should probably revise that when the number
	// of subnets grows.
	dbSubnets, err := dbmodel.GetGlobalSubnets(dbi, family)
	if err != nil {
		return subnets, err
	}
	indexedSubnets := dbmodel.NewIndexedSubnets(dbSubnets)
	if ok := indexedSubnets.Populate(); !ok {
		err = errors.Errorf("failed to build indexes for existing subnets because duplicates are present")

		return subnets, err
	}

	// Iterate over the configured subnets.
	for _, s := range subnetList {
		// Parse the configured subnet.
		subnet, err := dbmodel.NewSubnetFromKea(s, daemon, dbmodel.HostDataSourceConfig, lookup)
		if err != nil {
			log.WithError(err).Warnf("Skipping invalid subnet")
			continue
		}
		existingSubnet := findMatchingSubnet(subnet, indexedSubnets)
		if existingSubnet != nil {
			// Subnet already exists and may contain some updated data. Let's
			// override the data from the new subnet into the existing subnet.
			err := overrideIntoDatabaseSubnet(dbi, existingSubnet, subnet)
			if err != nil {
				log.WithError(err).Warnf("Skipping subnet %s after data override failure",
					subnet.Prefix)
				continue
			}
			existingSubnet.Join(subnet)
			subnets = append(subnets, *existingSubnet)
		} else {
			subnets = append(subnets, *subnet)
		}
	}
	return subnets, nil
}

// For a given Kea daemon it detects the shared networks and subnets this Kea
// daemon has configured. The returned shared networks contain the subnets
// belonging to the shared networks.
func detectDaemonNetworks(dbi dbops.DBI, daemon *dbmodel.Daemon, lookup keaconfig.DHCPOptionDefinitionLookup) (networks []dbmodel.SharedNetwork, subnets []dbmodel.Subnet, err error) {
	// If this is not a Kea daemon or the configuration is unknown
	// there is nothing to do.
	if daemon.KeaDaemon == nil || daemon.KeaDaemon.Config == nil {
		return networks, subnets, err
	}

	var family int
	switch daemon.Name {
	case dhcp4:
		family = 4
	case dhcp6:
		family = 6
	default:
		return networks, subnets, err
	}

	// Detect shared networks and the subnets.
	detectedNetworks, err := detectSharedNetworks(dbi, daemon.KeaDaemon.Config, family, daemon, lookup)
	if err != nil {
		return networks, subnets, err
	}
	networks = append(networks, detectedNetworks...)

	// Detect top level subnets.
	detectedSubnets, err := detectSubnets(dbi, daemon.KeaDaemon.Config, family, daemon, lookup)
	if err != nil {
		return []dbmodel.SharedNetwork{}, []dbmodel.Subnet{}, err
	}
	subnets = append(subnets, detectedSubnets...)
	return networks, subnets, nil
}
