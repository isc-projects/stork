package dbmodel

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// Kea daemon configuration map. It comprises a set of functions
// which retrieve complex data structures from the configuration.
type KeaConfig map[string]interface{}

// Structure representing a configuration of the single hooks library.
type KeaConfigHooksLibrary struct {
	Library    string
	Parameters map[string]interface{}
}

// Structure representing a configuration of a HA peer.
type Peer struct {
	Name         *string
	URL          *string
	Role         *string
	AutoFailover *bool `mapstructure:"auto-failover"`
}

// Structure representing a configuration of the HA hooks library
type KeaConfigHA struct {
	ThisServerName    *string `mapstructure:"this-server-name"`
	Mode              *string
	HeartbeatDelay    *int `mapstructure:"heartbeat-delay"`
	MaxResponseDelay  *int `mapstructure:"max-response-delay"`
	MaxAckDelay       *int `mapstructure:"max-ack-delay"`
	MaxUnackedClients *int `mapstructure:"max-unacked-clients"`
	Peers             []Peer
}

// Creates new instance from the pointer to the map of interfaces.
func NewKeaConfig(rawCfg *map[string]interface{}) *KeaConfig {
	newCfg := KeaConfig(*rawCfg)
	return &newCfg
}

// Create new instance from the configuration provided as JSON text.
func NewKeaConfigFromJSON(rawCfg string) (*KeaConfig, error) {
	var cfg KeaConfig
	err := json.Unmarshal([]byte(rawCfg), &cfg)
	if err != nil {
		err := errors.Wrapf(err, "problem with parsing JSON text: %s", rawCfg)
		return nil, err
	}
	return &cfg, nil
}

// Converts a structure holding subnet in Kea format to Stork representation
// of the subnet.
func convertSubnetFromKea(keaSubnet *KeaConfigSubnet) (*Subnet, error) {
	convertedSubnet := &Subnet{
		Prefix:      keaSubnet.Subnet,
		ClientClass: keaSubnet.ClientClass,
	}
	for _, p := range keaSubnet.Pools {
		addressPool, err := NewAddressPoolFromRange(p.Pool)
		if err != nil {
			return nil, err
		}
		addressPool.SubnetID = keaSubnet.ID
		convertedSubnet.AddressPools = append(convertedSubnet.AddressPools, *addressPool)
	}
	for _, p := range keaSubnet.PdPools {
		prefixPool, err := NewPrefixPool(fmt.Sprintf("%s/%d", p.Prefix, p.PrefixLen), p.DelegatedLen)
		if err != nil {
			return nil, err
		}
		prefixPool.SubnetID = keaSubnet.ID
		convertedSubnet.PrefixPools = append(convertedSubnet.PrefixPools, *prefixPool)
	}
	for _, r := range keaSubnet.Reservations {
		host, err := NewHostFromKeaConfigReservation(r)
		if err != nil {
			return nil, err
		}

		// We need to check if the host with the same set of reservations already
		// exists. If so, then we may need to merge the new host with it.
		found := false
		for i, c := range convertedSubnet.Hosts {
			if c.HasEqualIPReservations(host) {
				// Go over the identifiers of the new host and for each that doesn't
				// exist, create it in the existing host.
				for j, id := range host.HostIdentifiers {
					if exists, _ := c.HasIdentifier(id.Type, id.Value); !exists {
						convertedSubnet.Hosts[i].HostIdentifiers = append(convertedSubnet.Hosts[i].HostIdentifiers, host.HostIdentifiers[j])
					}
					// We just take the first identifier, because Kea host reservation
					// never has more than one.
					break
				}
				found = true
				break
			}
		}
		if !found {
			// Existing reservation not found, add the whole host.
			convertedSubnet.Hosts = append(convertedSubnet.Hosts, *host)
		}
	}
	return convertedSubnet, nil
}

// Creates new shared network instance from the pointer to the map of interfaces.
// The family designates if the shared network contains IPv4 (if 4) or IPv6 (if 6)
// subnets. If any of the subnets doesn't match this value, an error is returned.
func NewSharedNetworkFromKea(rawNetwork *map[string]interface{}, family int) (*SharedNetwork, error) {
	var parsedSharedNetwork KeaConfigSharedNetwork
	_ = mapstructure.Decode(rawNetwork, &parsedSharedNetwork)
	newSharedNetwork := &SharedNetwork{
		Name:   parsedSharedNetwork.Name,
		Family: family,
	}

	for _, subnetList := range [][]KeaConfigSubnet{parsedSharedNetwork.Subnet4, parsedSharedNetwork.Subnet6} {
		for _, s := range subnetList {
			keaSubnet := s
			subnet, err := convertSubnetFromKea(&keaSubnet)
			if err == nil {
				if subnet.GetFamily() != family {
					return nil, errors.Errorf("non matching family of the subnet %s with the shared network %s",
						subnet.Prefix, newSharedNetwork.Name)
				}
				newSharedNetwork.Subnets = append(newSharedNetwork.Subnets, *subnet)
			} else {
				return nil, err
			}
		}
	}

	// Update shared network family based on the subnets family.
	if len(newSharedNetwork.Subnets) > 0 {
		newSharedNetwork.Family = newSharedNetwork.Subnets[0].GetFamily()
	}

	return newSharedNetwork, nil
}

// Creates new subnet instance from the pointer to the map of interfaces.
func NewSubnetFromKea(rawSubnet *map[string]interface{}) (*Subnet, error) {
	var parsedSubnet KeaConfigSubnet
	_ = mapstructure.Decode(rawSubnet, &parsedSubnet)
	return convertSubnetFromKea(&parsedSubnet)
}

// Creates new host instance from the host reservation extracted from the
// Kea configuration.
func NewHostFromKeaConfigReservation(reservation KeaConfigReservation) (*Host, error) {
	var host Host
	structType := reflect.TypeOf(reservation)
	value := reflect.ValueOf(reservation)

	// Iterate over the struct fields which may hold host identifiers.
	for _, name := range []string{"HWAddress", "DUID", "CircuitID", "ClientID", "FlexID"} {
		fieldValue := strings.TrimSpace(value.FieldByName(name).String())
		if len(fieldValue) > 0 {
			// Struct field type is required to map the struct field to
			// an identifier name.
			fieldType, ok := structType.FieldByName(name)
			if !ok {
				continue
			}
			// The mapstructure tag contains the identifier name.
			tag, ok := fieldType.Tag.Lookup("mapstructure")
			if !ok {
				continue
			}
			// Remove colons from the string of hexadecimal values.
			hexv := strings.ReplaceAll(fieldValue, ":", "")
			// Convert the identifier to binary.
			bytev, err := hex.DecodeString(hexv)
			if err != nil {
				return nil, err
			}
			identifier := HostIdentifier{
				Type:  tag,
				Value: bytev,
			}
			// Append the identifier.
			host.HostIdentifiers = append(host.HostIdentifiers, identifier)
		}
	}
	// Iterate over the IPv6 addresses and prefixes and create IP reservations
	// from them
	for _, addrs := range [][]string{reservation.IPAddresses, reservation.Prefixes} {
		for _, addr := range addrs {
			host.IPReservations = append(host.IPReservations, IPReservation{
				Address: strings.TrimSpace(addr),
			})
		}
	}
	// Finally, take the IPv4 reservation.
	if len(reservation.IPAddress) > 0 {
		host.IPReservations = append(host.IPReservations, IPReservation{
			Address: strings.TrimSpace(reservation.IPAddress),
		})
	}
	return &host, nil
}

// Returns name of the root configuration node, e.g. Dhcp4.
// The second returned value designates whether the root node
// name was successfully found or not.
func (c *KeaConfig) GetRootName() (string, bool) {
	// This map will typically hold just a single element, but
	// in the past Kea supported Logging parameter aside of the
	// DHCP server configuration so we need to eliminate this one.
	for key := range *c {
		if key != "Logging" {
			return key, true
		}
	}
	return "", false
}

// Returns a list found at the top level of the configuration under
// a given name. If the given parameter does not exist or it is
// not a list, the ok value returned is set to false.
func (c *KeaConfig) GetTopLevelList(name string) (list []interface{}, ok bool) {
	root, ok := c.GetRootName()
	if !ok {
		return list, ok
	}
	if cfg, ok := (*c)[root]; ok {
		if rootNode, ok := cfg.(map[string]interface{}); ok {
			if listNode, ok := rootNode[name].([]interface{}); ok {
				return listNode, ok
			}
		}
	}

	return list, false
}

// Returns a list of all hooks libraries found in the configuration.
func (c *KeaConfig) GetHooksLibraries() (parsedLibraries []KeaConfigHooksLibrary) {
	if hooksLibrariesList, ok := c.GetTopLevelList("hooks-libraries"); ok {
		_ = mapstructure.Decode(hooksLibrariesList, &parsedLibraries)
	}
	return parsedLibraries
}

// Returns the information about a hooks library having a specified name
// if it exists in the configuration. The name parameter designates the
// name of the library, e.g. libdhcp_ha. The returned values include the
// path to the library, library configuration and the flag indicating
// whether the library exists or not.
func (c *KeaConfig) GetHooksLibrary(name string) (path string, params map[string]interface{}, ok bool) {
	libraries := c.GetHooksLibraries()
	for _, lib := range libraries {
		if strings.Contains(lib.Library, name) {
			path = lib.Library
			params = lib.Parameters
			ok = true
		}
	}
	return path, params, ok
}

// Returns configuration of the HA hooks library in a parsed form.
func (c *KeaConfig) GetHAHooksLibrary() (path string, params KeaConfigHA, ok bool) {
	path, paramsMap, ok := c.GetHooksLibrary("libdhcp_ha")
	if !ok {
		return path, params, ok
	}

	// HA hooks library should contain high-availability parameter being a
	// single element list. If it doesn't exist, it is an error.
	if haParamsList, ok := paramsMap["high-availability"].([]interface{}); !ok {
		path = ""
	} else {
		// Parse the list of HA configurations into a list of structures.
		var paramsList []KeaConfigHA
		err := mapstructure.Decode(haParamsList, &paramsList)
		if err != nil || len(paramsList) == 0 {
			path = ""
		} else {
			// HA configuration found, return it.
			params = paramsList[0]
		}
	}

	return path, params, ok
}

// Matches the prefix of a subnet with the given IP network. If the match is
// found the local subnet id of that subnet is returned. Otherwise, the value
// of 0 is returned.
func getMatchingSubnetLocalID(subnet interface{}, ipNet *net.IPNet) int64 {
	var parsedSubnet struct {
		ID     int64
		Subnet string
	}
	// Get the subnet's ID and prefix.
	_ = mapstructure.Decode(subnet, &parsedSubnet)

	// Parse the prefix into a common form that can be used for comparison.
	_, localNetwork, err := net.ParseCIDR(parsedSubnet.Subnet)
	if err != nil {
		return 0
	}
	// Compare the prefix of the subnet we have found and the specified prefix.
	if (localNetwork != nil) && net.IP.Equal(ipNet.IP, localNetwork.IP) &&
		bytes.Equal(ipNet.Mask, localNetwork.Mask) {
		return parsedSubnet.ID
	}
	// No match.
	return 0
}

// Scans subnets within the Kea configuration and returns the ID of the subnet having
// the specified prefix.
func (c *KeaConfig) GetLocalSubnetID(prefix string) int64 {
	_, globalNetwork, err := net.ParseCIDR(prefix)
	if err != nil || globalNetwork == nil {
		return 0
	}

	// Depending on the DHCP server type, we need to use different name of the list
	// holding the subnets.
	rootName, ok := c.GetRootName()
	if !ok {
		return 0
	}
	var subnetParamName string
	switch rootName {
	case "Dhcp4":
		subnetParamName = "subnet4"
	case "Dhcp6":
		subnetParamName = "subnet6"
	default:
		// If this is neither the DHCPv4 nor DHCPv6 server, there is nothing to do.
		return 0
	}

	// First, let's iterate over the subnets which are not associated with any
	// shared network.
	if subnetList, ok := c.GetTopLevelList(subnetParamName); ok {
		for _, s := range subnetList {
			id := getMatchingSubnetLocalID(s, globalNetwork)
			if id > 0 {
				return id
			}
		}
	}

	// No match. Let's get the subnets belonging to the shared networks.
	if networkList, ok := c.GetTopLevelList("shared-networks"); ok {
		for _, n := range networkList {
			if network, ok := n.(map[string]interface{}); ok {
				if subnetList, ok := network[subnetParamName].([]interface{}); ok {
					for _, s := range subnetList {
						id := getMatchingSubnetLocalID(s, globalNetwork)
						if id > 0 {
							return id
						}
					}
				}
			}
		}
	}

	return 0
}

// Checks if the mandatory peer parameters are set. It doesn't check if the
// values are correct.
func (p Peer) IsSet() bool {
	return p.Name != nil && p.URL != nil && p.Role != nil
}

// Checks if the mandatory Kea HA configuration parameters are set. It doesn't
// check parameters consistency, though.
func (c KeaConfigHA) IsSet() bool {
	// Check if peers are valid.
	for _, p := range c.Peers {
		if !p.IsSet() {
			return false
		}
	}
	// Check other required parameters.
	return c.ThisServerName != nil && c.Mode != nil
}
