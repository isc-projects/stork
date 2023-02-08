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
	keaconfig "isc.org/stork/appcfg/kea"
	storkutil "isc.org/stork/util"
)

// Workaround wrapper for go-pg limitations with large JSONs.
// The go-pg 10 stores the serialized JSONs in buffers from
// github.com/vmihailenco/bufpool library. This library has a
// limitation for buffer size up to 32MB. Kea Daemon Configuration
// with 10758 subnets and 10 reservations per each exceed this limit.
//
// It isn't possible to replace or patch this mechanism, because
// it is used deeply in the internal part of Go-PG. But it is possible
// to workaround. This wrapper implements a custom serializer for Kea
// Config. It uses a standard JSON parser as the go-pg, but avoids using
// a bufpool.
//
// The behavior of this type should be the same as keaconfig.Map.
// It means that the constructors accept the same arguments and
// return the same output. All methods defined for keaconfig.Map
// work for KeaConfig. The only difference is that the KeaConfig
// cannot be cast directly to map[string]interface{}. It is recommended
// to avoid casting these types and use the methods and polymorphism
// power.
//
// Note that the bun library doesn't use bufpool. After migration to bun
// the KeaConfig structure should be replaced with alias to keaconfig.Map.
type KeaConfig struct {
	*keaconfig.Map
}

// KeaConfig doesn't implement a custom JSON marshaler but only calls
// the marshalling on the internal keaconfig.Map.
var _ json.Marshaler = (*KeaConfig)(nil)

// KeaConfig doesn't implement a custom JSON unmarshaler but only calls
// the unmarshalling on the internal keaconfig.Map.
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
func convertSubnet4FromKea(keaSubnet *keaconfig.Subnet4, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*Subnet, error) {
	// Kea allows providing the subnet prefix in a non-canonical form, but
	// Postgres rejects this value due to validation rules on the CIDR column.
	// E.g., 192.168.1.1/24 is a valid prefix for Kea, but Postgres expects
	// 192.168.1.0/24. We need to convert the prefix to its canonical form to
	// avoid a database error.
	prefix, err := keaSubnet.GetCanonicalPrefix()
	if err != nil {
		return nil, err
	}
	convertedSubnet := &Subnet{
		Prefix: prefix,
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID:      daemon.ID,
				LocalSubnetID: daemon.GetLocalSubnetID(prefix),
				KeaParameters: keaSubnet.GetSubnetParameters(),
			},
		},
	}
	if keaSubnet.ClientClass != nil {
		convertedSubnet.ClientClass = *keaSubnet.ClientClass
	}
	for _, p := range keaSubnet.Pools {
		addressPool, err := NewAddressPoolFromRange(p.Pool)
		if err != nil {
			return nil, err
		}
		addressPool.SubnetID = keaSubnet.ID
		convertedSubnet.AddressPools = append(convertedSubnet.AddressPools, *addressPool)
	}
	for _, r := range keaSubnet.Reservations {
		host, err := NewHostFromKeaConfigReservation(r, daemon, source, lookup)
		if err != nil {
			return nil, err
		}
		convertedSubnet.Hosts = append(convertedSubnet.Hosts, *host)
	}
	for _, d := range keaSubnet.OptionData {
		option, err := NewDHCPOptionFromKea(d, storkutil.IPv4, lookup)
		if err != nil {
			return nil, err
		}
		convertedSubnet.LocalSubnets[0].DHCPOptionSet = append(convertedSubnet.LocalSubnets[0].DHCPOptionSet, *option)
		convertedSubnet.LocalSubnets[0].DHCPOptionSetHash = storkutil.Fnv128(fmt.Sprintf("%+v", convertedSubnet.LocalSubnets[0].DHCPOptionSet))
	}
	return convertedSubnet, nil
}

// Converts a structure holding subnet in Kea format to Stork representation
// of the subnet.
func convertSubnet6FromKea(keaSubnet *keaconfig.Subnet6, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*Subnet, error) {
	// Kea allows providing the subnet prefix in a non-canonical form, but
	// Postgres rejects this value due to validation rules on the CIDR column.
	// E.g., 192.168.1.1/24 is a valid prefix for Kea, but Postgres expects
	// 192.168.1.0/24. We need to convert the prefix to its canonical form to
	// avoid a database error.
	prefix, err := keaSubnet.GetCanonicalPrefix()
	if err != nil {
		return nil, err
	}
	convertedSubnet := &Subnet{
		Prefix: prefix,
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID:      daemon.ID,
				LocalSubnetID: daemon.GetLocalSubnetID(prefix),
				KeaParameters: keaSubnet.GetSubnetParameters(),
			},
		},
	}
	if keaSubnet.ClientClass != nil {
		convertedSubnet.ClientClass = *keaSubnet.ClientClass
	}
	for _, p := range keaSubnet.Pools {
		lb, ub, err := p.GetBoundaries()
		if err != nil {
			return nil, err
		}
		addressPool := NewAddressPool(lb, ub, keaSubnet.ID)
		convertedSubnet.AddressPools = append(convertedSubnet.AddressPools, *addressPool)
	}
	for _, p := range keaSubnet.PDPools {
		prefix := p.GetCanonicalPrefix()
		excludedPrefix := p.GetCanonicalExcludedPrefix()
		prefixPool, err := NewPrefixPool(prefix, p.DelegatedLen, excludedPrefix, keaSubnet.ID)
		if err != nil {
			return nil, err
		}
		convertedSubnet.PrefixPools = append(convertedSubnet.PrefixPools, *prefixPool)
	}
	for _, r := range keaSubnet.Reservations {
		host, err := NewHostFromKeaConfigReservation(r, daemon, source, lookup)
		if err != nil {
			return nil, err
		}
		convertedSubnet.Hosts = append(convertedSubnet.Hosts, *host)
	}
	for _, d := range keaSubnet.OptionData {
		option, err := NewDHCPOptionFromKea(d, storkutil.IPv4, lookup)
		if err != nil {
			return nil, err
		}
		convertedSubnet.LocalSubnets[0].DHCPOptionSet = append(convertedSubnet.LocalSubnets[0].DHCPOptionSet, *option)
		convertedSubnet.LocalSubnets[0].DHCPOptionSetHash = storkutil.Fnv128(fmt.Sprintf("%+v", convertedSubnet.LocalSubnets[0].DHCPOptionSet))
	}
	return convertedSubnet, nil
}

// Creates new shared network instance from the pointer to the map of interfaces.
// The family designates if the shared network contains IPv4 (if 4) or IPv6 (if 6)
// subnets. If none of the subnets match this value, an error is returned.
func NewSharedNetworkFromKea(rawNetwork *map[string]interface{}, family int, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*SharedNetwork, error) {
	var newSharedNetwork *SharedNetwork
	switch family {
	case 4:
		var parsedSharedNetwork keaconfig.SharedNetwork4
		_ = mapstructure.Decode(rawNetwork, &parsedSharedNetwork)
		newSharedNetwork = &SharedNetwork{
			Name: parsedSharedNetwork.Name,
			LocalSharedNetworks: []*LocalSharedNetwork{
				{
					DaemonID:      daemon.ID,
					KeaParameters: parsedSharedNetwork.GetSharedNetworkParameters(),
				},
			},
		}
		for _, s := range parsedSharedNetwork.Subnet4 {
			keaSubnet := s
			subnet, err := convertSubnet4FromKea(&keaSubnet, daemon, source, lookup)
			if err != nil {
				return nil, err
			}
			newSharedNetwork.Subnets = append(newSharedNetwork.Subnets, *subnet)
		}
		for _, d := range parsedSharedNetwork.OptionData {
			option, err := NewDHCPOptionFromKea(d, storkutil.IPv4, lookup)
			if err != nil {
				return nil, err
			}
			newSharedNetwork.LocalSharedNetworks[0].DHCPOptionSet = append(newSharedNetwork.LocalSharedNetworks[0].DHCPOptionSet, *option)
			newSharedNetwork.LocalSharedNetworks[0].DHCPOptionSetHash = storkutil.Fnv128(fmt.Sprintf("%+v", newSharedNetwork.LocalSharedNetworks[0].DHCPOptionSet))
		}

	default:
		var parsedSharedNetwork keaconfig.SharedNetwork6
		_ = mapstructure.Decode(rawNetwork, &parsedSharedNetwork)
		newSharedNetwork = &SharedNetwork{
			Name: parsedSharedNetwork.Name,
			LocalSharedNetworks: []*LocalSharedNetwork{
				{
					DaemonID:      daemon.ID,
					KeaParameters: parsedSharedNetwork.GetSharedNetworkParameters(),
				},
			},
		}
		for _, s := range parsedSharedNetwork.Subnet6 {
			keaSubnet := s
			subnet, err := convertSubnet6FromKea(&keaSubnet, daemon, source, lookup)
			if err != nil {
				return nil, err
			}
			newSharedNetwork.Subnets = append(newSharedNetwork.Subnets, *subnet)
		}
		for _, d := range parsedSharedNetwork.OptionData {
			option, err := NewDHCPOptionFromKea(d, storkutil.IPv4, lookup)
			if err != nil {
				return nil, err
			}
			newSharedNetwork.LocalSharedNetworks[0].DHCPOptionSet = append(newSharedNetwork.LocalSharedNetworks[0].DHCPOptionSet, *option)
			newSharedNetwork.LocalSharedNetworks[0].DHCPOptionSetHash = storkutil.Fnv128(fmt.Sprintf("%+v", newSharedNetwork.LocalSharedNetworks[0].DHCPOptionSet))
		}
	}
	newSharedNetwork.Family = family

	// Update shared network family based on the subnets family.
	if len(newSharedNetwork.Subnets) > 0 {
		newSharedNetwork.Family = newSharedNetwork.Subnets[0].GetFamily()
	}

	return newSharedNetwork, nil
}

// Creates new IPv4 subnet instance in Stork from the Kea subnet configuration.
// It decodes the raw configuration provided as a map of interfaces.
func NewSubnet4FromKea(rawSubnet *map[string]any, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*Subnet, error) {
	var parsedSubnet keaconfig.Subnet4
	_ = mapstructure.Decode(rawSubnet, &parsedSubnet)
	return convertSubnet4FromKea(&parsedSubnet, daemon, source, lookup)
}

// Creates new IPv6 subnet instance in Stork from the Kea subnet configuration.
// It decodes the raw configuration provided as a map of interfaces.
func NewSubnet6FromKea(rawSubnet *map[string]any, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*Subnet, error) {
	var parsedSubnet keaconfig.Subnet6
	_ = mapstructure.Decode(rawSubnet, &parsedSubnet)
	return convertSubnet6FromKea(&parsedSubnet, daemon, source, lookup)
}

// Creates new host instance from the host reservation extracted from the
// Kea configuration.
func NewHostFromKeaConfigReservation(reservation keaconfig.Reservation, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*Host, error) {
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
	// Take the IPv4 reservation.
	if len(reservation.IPAddress) > 0 {
		host.IPReservations = append(host.IPReservations, IPReservation{
			Address: strings.TrimSpace(reservation.IPAddress),
		})
	}
	// Finally, store server specific host information including DHCP options.
	lh := LocalHost{
		DaemonID:       daemon.ID,
		DataSource:     source,
		ClientClasses:  reservation.ClientClasses,
		NextServer:     reservation.NextServer,
		ServerHostname: reservation.ServerHostname,
		BootFileName:   reservation.BootFileName,
	}
	universe := storkutil.IPv4
	if daemon.Name == DaemonNameDHCPv6 {
		universe = storkutil.IPv6
	}
	for _, d := range reservation.OptionData {
		hostOption, err := NewDHCPOptionFromKea(d, universe, lookup)
		if err != nil {
			return nil, err
		}
		lh.DHCPOptionSet = append(lh.DHCPOptionSet, *hostOption)
		lh.DHCPOptionSetHash = storkutil.Fnv128(fmt.Sprintf("%+v", lh.DHCPOptionSet))
	}
	host.LocalHosts = append(host.LocalHosts, lh)
	return &host, nil
}

// Creates new host instance from the pointer to the map of interfaces.
func NewHostFromKea(rawHost *map[string]interface{}, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*Host, error) {
	var parsedHost keaconfig.Reservation
	_ = mapstructure.Decode(rawHost, &parsedHost)
	return NewHostFromKeaConfigReservation(parsedHost, daemon, source, lookup)
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
