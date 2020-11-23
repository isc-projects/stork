package keaconfig

import (
	errors "github.com/pkg/errors"
)

// Structure representing a container for subnets. The container allows
// for accessing stored subnets using various keys (indexes). This
// significantly improves subnet lookup time comparing to the case when
// subnets are stored as a slice. New indexes can be added as needed
// in the future.
type IndexedSubnets struct {
	Config   *Map
	ByPrefix map[string]map[string]interface{}
}

// Creates new instance of the IndexedSubnets structure. It takes a raw
// Kea configuration as an input, iterates over the shared networks and
// global subnets and builds appropriate indexes.
func NewIndexedSubnets(rawConfig *Map) *IndexedSubnets {
	if rawConfig == nil {
		panic("provided DHCP configuration must not be nil when indexing subnets")
	}
	return &IndexedSubnets{
		Config: rawConfig,
	}
}

// Rebuild indexes using subnets stored in the Config field as input.
// It returns an error if duplicates are found or subnets have wrong
// structure.
func (is *IndexedSubnets) Populate() error {
	// List of subnets is available under a different parameter name depending
	// on whether this is a DHCPv4 or DHCPv6 configuration. Find this name
	// first.
	rootName, ok := is.Config.GetRootName()
	if !ok {
		return errors.New("failed to index subnets because given configuration is invalid")
	}
	var subnetParamName string
	switch rootName {
	case "Dhcp4":
		subnetParamName = "subnet4"
	case "Dhcp6":
		subnetParamName = "subnet6"
	default:
		return errors.New("failed to index subnets because given configuration is not DHCP configuration")
	}

	// Create empty indexes.
	subnets := []interface{}{}
	byPrefix := make(map[string]map[string]interface{})

	// Go over the shared networks and for each of them gather the subnets list.
	networks, _ := is.Config.GetTopLevelList("shared-networks")
	for i := range networks {
		network, ok := networks[i].(map[string]interface{})
		if !ok {
			return errors.New("failed to index subnets because one of the shared networks is not a map")
		}
		if networkSubnets, ok := network[subnetParamName].([]interface{}); ok {
			subnets = append(subnets, networkSubnets...)
		}
	}

	// Finally, append global subnets.
	if globalSubnets, ok := is.Config.GetTopLevelList(subnetParamName); ok {
		subnets = append(subnets, globalSubnets...)
	}

	// Go over the subnets and build the by prefix index.
	for i := range subnets {
		subnet, ok := subnets[i].(map[string]interface{})
		if !ok {
			return errors.New("failed to index subnets because one of the subnets is not a map")
		}
		prefix, ok := subnet["subnet"].(string)
		if !ok {
			return errors.New("failed to index subnets because subnet definition lacks prefix")
		}
		if _, ok = byPrefix[prefix]; ok {
			return errors.Errorf("failed to index subnets because duplicate entry was found for %s", prefix)
		}
		byPrefix[prefix] = subnet
	}

	// Everything went fine, so replace indexes with fresh ones.
	is.ByPrefix = byPrefix
	return nil
}
