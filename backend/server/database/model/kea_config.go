package dbmodel

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
)

type KeaConfig = keaconfig.Map

// Creates new instance from the pointer to the map of interfaces.
func NewKeaConfig(rawCfg *map[string]interface{}) *KeaConfig {
	return keaconfig.New(rawCfg)
}

// Create new instance from the configuration provided as JSON text.
func NewKeaConfigFromJSON(rawCfg string) (*KeaConfig, error) {
	return keaconfig.NewFromJSON(rawCfg)
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
	var parsedHost KeaConfigReservation
	_ = mapstructure.Decode(rawHost, &parsedHost)
	return NewHostFromKeaConfigReservation(parsedHost)
}

// Creates log targets from Kea logger configuration. The Kea logger configuration
// can comprise multiple output options. Therefore, this function may return multiple
// targets, each correposnding to a single output option.
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
