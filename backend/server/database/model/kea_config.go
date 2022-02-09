package dbmodel

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-pg/pg/v10/types"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
)

// Workaround wrapper for go-pg limitations with huge  JSONs.
// The go-pg 10 stores the serialized JSONs in buffers from
// github.com/vmihailenco/bufpool library. This library has a
// limitation for buffer size up to 32MB. Kea Daemon Configuration
// with 10758 subnets and 10 reservations per each exceed this limit.
//
// It isn't possible to replace or patch this mechanism, because
// it is used deeply in the internal part of Go-PG. But it is possible
// to workaround. This wrapper implements a custom serializer for Kea
// Config. It uses a standard JSON parser as the go-pg, but avoid using
// a bufpool.
//
// The behavior of this type should be the same as keaconfig.Map.
// It means that the constructors accept the same arguments and
// return the same output. All methods are defined for keaconfig.Map
// works for KeaConfig. The only difference is that the KeaConfig
// cannot be cast directly to map[string]interface{}. The recommended
// way is to avoid casting these types and use the methods and polymorphism
// power.
//
// Note that the bun library doesn't use bufpool. After migration to bun
// the KeaConfig structure should be replaced with alias to keaconfig.Map.
type KeaConfig struct {
	*keaconfig.Map
}

// KeaConfig doesn't implement a custom JSON marshaler but only calls
// the marshaling on the internal keaconfig.Map.
var _ json.Marshaler = (*KeaConfig)(nil)

// KeaConfig doesn't implement a custom JSON unmarshaler but only calls
// the unmarshaling on the internal keaconfig.Map.
var _ json.Unmarshaler = (*KeaConfig)(nil)

// The database serializer that workarounds the bufpool.
// Serialize the Kea config to JSON using standard JSON
// marshaler and escape the single quotes.
var _ types.ValueAppender = (*KeaConfig)(nil)

// The database deserializer that workarounds the bufpool.
// Deserialize the Kea config to JSON using standard JSON
// unmarshaler and unescape the single quotes.
var _ types.ValueScanner = (*KeaConfig)(nil)

// Marshal the internal map to JSON.
func (c *KeaConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Map)
}

// Unmarshal the internal map from JSON.
func (c *KeaConfig) UnmarshalJSON(bytes []byte) error {
	if c.Map == nil {
		c.Map = keaconfig.New(&map[string]interface{}{})
	}
	return json.Unmarshal(bytes, c.Map)
}

// Implements the go-pg serializer. It marshals the config
// to JSON and escapes all single quotes.
func (c *KeaConfig) AppendValue(b []byte, quote int) ([]byte, error) {
	if c == nil {
		b = append(b, []byte("NULL")...)
		return b, nil
	}

	if quote == 1 {
		b = append(b, '\'')
	}

	jsonBytes, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	jsonBytes = bytes.ReplaceAll(jsonBytes, []byte{'\''}, []byte{'\'', '\''})

	b = append(b, jsonBytes...)
	if quote == 1 {
		b = append(b, '\'')
	}
	return b, nil
}

// Implements the go-pg deserializer. It unescapes all single
// quotes and unmarshals the config from JSON.
func (c *KeaConfig) ScanValue(rd types.Reader, n int) error {
	if n <= 0 {
		return nil
	}

	jsonBytes, err := rd.ReadFullTemp()
	if err != nil {
		return err
	}

	jsonBytes = bytes.ReplaceAll(jsonBytes, []byte{'\'', '\''}, []byte{'\''})

	return json.Unmarshal(jsonBytes, c)
}

// Creates new instance from the pointer to the map of interfaces.
func NewKeaConfig(rawCfg *map[string]interface{}) *KeaConfig {
	if rawCfg == nil {
		return nil
	}
	return &KeaConfig{Map: keaconfig.New(rawCfg)}
}

// Create new instance from the configuration provided as JSON text.
func NewKeaConfigFromJSON(rawCfg string) (*KeaConfig, error) {
	keaConfigMap, err := keaconfig.NewFromJSON(rawCfg)
	if err != nil {
		return nil, err
	}
	return &KeaConfig{Map: keaConfigMap}, nil
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
func NewHostFromKeaConfigReservation(reservation keaconfig.Reservation) (*Host, error) {
	var host Host
	host.Hostname = reservation.Hostname
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

// Creates new host instance from the pointer to the map of interfaces.
func NewHostFromKea(rawHost *map[string]interface{}) (*Host, error) {
	var parsedHost keaconfig.Reservation
	_ = mapstructure.Decode(rawHost, &parsedHost)
	return NewHostFromKeaConfigReservation(parsedHost)
}

// Creates log targets from Kea logger configuration. The Kea logger configuration
// can comprise multiple output options. Therefore, this function may return multiple
// targets, each corresponding to a single output option.
func NewLogTargetsFromKea(logger keaconfig.Logger) (targets []*LogTarget) {
	for _, opt := range logger.OutputOptions {
		target := &LogTarget{
			Name:     logger.Name,
			Severity: strings.ToLower(logger.Severity),
			Output:   opt.Output,
		}
		targets = append(targets, target)
	}
	return targets
}

// Convenience function which populates subnet indexes for each Kea daemon.
func PopulateIndexedSubnets(app *App) error {
	for i := range app.Daemons {
		if app.Daemons[i].KeaDaemon != nil &&
			app.Daemons[i].KeaDaemon.Config != nil &&
			app.Daemons[i].KeaDaemon.KeaDHCPDaemon != nil {
			indexedSubnets := keaconfig.NewIndexedSubnets(app.Daemons[i].KeaDaemon.Config)
			err := indexedSubnets.Populate()
			if err != nil {
				return err
			}
			app.Daemons[i].KeaDaemon.KeaDHCPDaemon.IndexedSubnets = indexedSubnets
		}
	}
	return nil
}
