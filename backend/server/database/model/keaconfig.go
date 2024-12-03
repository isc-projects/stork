package dbmodel

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/go-pg/pg/v10/types"
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
// The behavior of this type should be the same as keaconfig.Config.
// It means that the constructors accept the same arguments and
// return the same output. All methods defined for keaconfig.Config
// work for KeaConfig.
//
// Note that the bun library doesn't use bufpool. After migration to bun
// the KeaConfig structure should be replaced with alias to keaconfig.Config.
type KeaConfig struct {
	*keaconfig.Config
}

// KeaConfig doesn't implement a custom JSON marshaler but only calls
// the marshalling on the internal keaconfig.Config.
var _ json.Marshaler = (*KeaConfig)(nil)

// KeaConfig doesn't implement a custom JSON unmarshaler but only calls
// the unmarshalling on the internal keaconfig.Config.
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
	return json.Marshal(c.Config)
}

// Unmarshal the internal map from JSON.
func (c *KeaConfig) UnmarshalJSON(bytes []byte) error {
	if c.Config == nil {
		c.Config = &keaconfig.Config{}
	}
	return json.Unmarshal(bytes, c.Config)
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
	config := keaconfig.NewConfigFromMap(rawCfg)
	if config == nil {
		return nil
	}
	return &KeaConfig{
		Config: config,
	}
}

// Create new instance from the configuration provided as JSON text.
func NewKeaConfigFromJSON(rawCfg string) (*KeaConfig, error) {
	config, err := keaconfig.NewConfig(rawCfg)
	if err != nil {
		return nil, err
	}
	return &KeaConfig{Config: config}, nil
}

// Converts a structure holding subnet in Kea format to Stork representation
// of the subnet.
func convertSubnetFromKea(keaSubnet keaconfig.Subnet, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*Subnet, error) {
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
				LocalSubnetID: keaSubnet.GetID(),
				KeaParameters: keaSubnet.GetSubnetParameters(),
				UserContext:   keaSubnet.GetUserContext(),
			},
		},
	}
	if keaSubnet.GetSubnetParameters().ClientClass != nil {
		convertedSubnet.ClientClass = *keaSubnet.GetSubnetParameters().ClientClass
	}
	hasher := keaconfig.NewHasher()
	for _, p := range keaSubnet.GetPools() {
		pool := p
		lb, ub, err := pool.GetBoundaries()
		if err != nil {
			return nil, err
		}
		addressPool := NewAddressPool(lb, ub)
		addressPool.KeaParameters = pool.GetPoolParameters()
		var options []DHCPOption
		// Options.
		for _, d := range p.OptionData {
			option, err := NewDHCPOptionFromKea(d, keaSubnet.GetUniverse(), lookup)
			if err != nil {
				return nil, err
			}
			options = append(options, *option)
		}
		addressPool.SetDHCPOptions(options, hasher)
		convertedSubnet.LocalSubnets[0].AddressPools = append(convertedSubnet.LocalSubnets[0].AddressPools, *addressPool)
	}
	for _, p := range keaSubnet.GetPDPools() {
		pool := p
		prefix := pool.GetCanonicalPrefix()
		excludedPrefix := pool.GetCanonicalExcludedPrefix()
		prefixPool, err := NewPrefixPool(prefix, p.DelegatedLen, excludedPrefix)
		if err != nil {
			return nil, err
		}
		prefixPool.KeaParameters = pool.GetPoolParameters()
		// Options.
		for _, d := range p.OptionData {
			option, err := NewDHCPOptionFromKea(d, keaSubnet.GetUniverse(), lookup)
			if err != nil {
				return nil, err
			}
			prefixPool.DHCPOptionSet = append(prefixPool.DHCPOptionSet, *option)
			prefixPool.DHCPOptionSetHash = hasher.Hash(prefixPool.DHCPOptionSet)
		}
		convertedSubnet.LocalSubnets[0].PrefixPools = append(convertedSubnet.LocalSubnets[0].PrefixPools, *prefixPool)
	}
	for _, r := range keaSubnet.GetReservations() {
		host, err := NewHostFromKeaConfigReservation(r, daemon, source, lookup)
		if err != nil {
			return nil, err
		}
		convertedSubnet.Hosts = append(convertedSubnet.Hosts, *host)
	}

	optionSet := []DHCPOption{}
	for _, d := range keaSubnet.GetDHCPOptions() {
		option, err := NewDHCPOptionFromKea(d, keaSubnet.GetUniverse(), lookup)
		if err != nil {
			return nil, err
		}
		optionSet = append(optionSet, *option)
	}
	convertedSubnet.LocalSubnets[0].SetDHCPOptions(optionSet, hasher)

	return convertedSubnet, nil
}

// Creates new shared network instance from the pointer to the map of interfaces.
// The family designates if the shared network contains IPv4 (if 4) or IPv6 (if 6)
// subnets. If none of the subnets match this value, an error is returned.
func NewSharedNetworkFromKea(sharedNetwork keaconfig.SharedNetwork, family int, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*SharedNetwork, error) {
	newSharedNetwork := &SharedNetwork{
		Name:   sharedNetwork.GetName(),
		Family: family,
		LocalSharedNetworks: []*LocalSharedNetwork{
			{
				DaemonID:      daemon.ID,
				KeaParameters: sharedNetwork.GetSharedNetworkParameters(),
			},
		},
	}
	for _, s := range sharedNetwork.GetSubnets() {
		keaSubnet := s
		subnet, err := convertSubnetFromKea(keaSubnet, daemon, source, lookup)
		if err != nil {
			return nil, err
		}
		newSharedNetwork.Subnets = append(newSharedNetwork.Subnets, *subnet)
	}

	optionSet := []DHCPOption{}
	for _, d := range sharedNetwork.GetDHCPOptions() {
		option, err := NewDHCPOptionFromKea(d, storkutil.IPType(family), lookup)
		if err != nil {
			return nil, err
		}
		optionSet = append(optionSet, *option)
	}
	newSharedNetwork.LocalSharedNetworks[0].SetDHCPOptions(optionSet, keaconfig.NewHasher())
	return newSharedNetwork, nil
}

// Creates new subnet instance in Stork from the Kea subnet configuration.
// It decodes the raw configuration provided as a map of interfaces.
func NewSubnetFromKea(subnet keaconfig.Subnet, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*Subnet, error) {
	return convertSubnetFromKea(subnet, daemon, source, lookup)
}

// Creates new host instance from the host reservation extracted from the
// Kea configuration.
func NewHostFromKeaConfigReservation(reservation keaconfig.Reservation, daemon *Daemon, source HostDataSource, lookup keaconfig.DHCPOptionDefinitionLookup) (*Host, error) {
	var host Host
	hostname := reservation.Hostname
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
			// The json tag contains the identifier name.
			tag, ok := fieldType.Tag.Lookup("json")
			if !ok {
				continue
			}
			tagOpts := strings.Split(tag, ",")
			if len(tagOpts) == 0 {
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
				Type:  tagOpts[0],
				Value: bytev,
			}
			// Append the identifier.
			host.HostIdentifiers = append(host.HostIdentifiers, identifier)
		}
	}

	// Finally, store server specific host information including DHCP options.
	lh := LocalHost{
		DaemonID:       daemon.ID,
		DataSource:     source,
		ClientClasses:  reservation.ClientClasses,
		NextServer:     reservation.NextServer,
		ServerHostname: reservation.ServerHostname,
		BootFileName:   reservation.BootFileName,
		Hostname:       hostname,
	}
	universe := storkutil.IPv4
	if daemon.Name == DaemonNameDHCPv6 {
		universe = storkutil.IPv6
	}
	optionSet := []DHCPOption{}
	for _, d := range reservation.OptionData {
		hostOption, err := NewDHCPOptionFromKea(d, universe, lookup)
		if err != nil {
			return nil, err
		}
		optionSet = append(optionSet, *hostOption)
	}
	lh.SetDHCPOptions(optionSet, keaconfig.NewHasher())

	// Iterate over the IPv6 addresses and prefixes and create IP reservations
	// from them
	for _, addrs := range [][]string{reservation.IPAddresses, reservation.Prefixes} {
		for _, addr := range addrs {
			lh.IPReservations = append(lh.IPReservations, IPReservation{
				Address: strings.TrimSpace(addr),
			})
		}
	}
	// Take the IPv4 reservation.
	if len(reservation.IPAddress) > 0 {
		lh.IPReservations = append(lh.IPReservations, IPReservation{
			Address: strings.TrimSpace(reservation.IPAddress),
		})
	}

	host.LocalHosts = append(host.LocalHosts, lh)
	return &host, nil
}

// Creates log targets from Kea logger configuration. The Kea logger configuration
// can comprise multiple output options. Therefore, this function may return multiple
// targets, each corresponding to a single output option.
func NewLogTargetsFromKea(logger keaconfig.Logger) (targets []*LogTarget) {
	for _, opt := range logger.GetAllOutputOptions() {
		target := &LogTarget{
			Name:     logger.Name,
			Severity: strings.ToLower(logger.Severity),
			Output:   opt.Output,
		}
		targets = append(targets, target)
	}
	return targets
}
