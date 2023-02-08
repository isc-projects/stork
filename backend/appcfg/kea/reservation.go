package keaconfig

import (
	"github.com/pkg/errors"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// Interface of a host convertible to a Kea host reservation.
type Host interface {
	GetHostIdentifiers() []struct {
		Type  string
		Value []byte
	}
	GetIPReservations() []string
	GetHostname() string
	GetSubnetID(int64) (int64, error)
	GetClientClasses(int64) []string
	GetNextServer(int64) string
	GetServerHostname(int64) string
	GetBootFileName(int64) string
	GetDHCPOptions(int64) []dhcpmodel.DHCPOptionAccessor
}

// Represents host reservation within Kea configuration.
type Reservation struct {
	HWAddress      string             `mapstructure:"hw-address" json:"hw-address,omitempty"`
	DUID           string             `mapstructure:"duid" json:"duid,omitempty"`
	CircuitID      string             `mapstructure:"circuit-id" json:"circuit-id,omitempty"`
	ClientID       string             `mapstructure:"client-id" json:"client-id,omitempty"`
	FlexID         string             `mapstructure:"flex-id" json:"flex-id,omitempty"`
	IPAddress      string             `mapstructure:"ip-address" json:"ip-address,omitempty"`
	IPAddresses    []string           `mapstructure:"ip-addresses" json:"ip-addresses,omitempty"`
	Prefixes       []string           `mapstructure:"prefixes" json:"prefixes,omitempty"`
	Hostname       string             `mapstructure:"hostname" json:"hostname,omitempty"`
	ClientClasses  []string           `mapstructure:"client-classes" json:"client-classes,omitempty"`
	NextServer     string             `mapstructure:"next-server" json:"next-server,omitempty"`
	BootFileName   string             `mapstructure:"boot-file-name" json:"boot-file-name,omitempty"`
	ServerHostname string             `mapstructure:"server-hostname" json:"server-hostname,omitempty"`
	OptionData     []SingleOptionData `mapstructure:"option-data" json:"option-data,omitempty"`
}

// Represents host reservation returned and sent via Kea host commands hook library.
type HostCmdsReservation struct {
	Reservation
	SubnetID int64 `json:"subnet-id"`
}

// Represents deleted host reservation. It includes the fields required by
// Kea to find the reservation and delete it.
type HostCmdsDeletedReservation struct {
	IdentifierType string `mapstructure:"identifier-type" json:"identifier-type,omitempty"`
	Identifier     string `mapstructure:"identifier" json:"identifier,omitempty"`
	SubnetID       int64  `mapstructure:"subnet-id" json:"subnet-id"`
}

// Converts a host representation in Stork to Kea host reservation format used
// in Kea configuration. The lookup interface must not be nil.
func CreateReservation(daemonID int64, lookup DHCPOptionDefinitionLookup, host Host) (*Reservation, error) {
	reservation := &Reservation{
		Hostname:       host.GetHostname(),
		ClientClasses:  host.GetClientClasses(daemonID),
		NextServer:     host.GetNextServer(daemonID),
		ServerHostname: host.GetServerHostname(daemonID),
		BootFileName:   host.GetBootFileName(daemonID),
	}
	for _, id := range host.GetHostIdentifiers() {
		value := storkutil.BytesToHex(id.Value)
		switch id.Type {
		case "hw-address":
			reservation.HWAddress = value
		case "duid":
			reservation.DUID = value
		case "circuit-id":
			reservation.CircuitID = value
		case "client-id":
			reservation.ClientID = value
		case "flex-id":
			reservation.FlexID = value
		}
	}
	for _, ipr := range host.GetIPReservations() {
		parsed := storkutil.ParseIP(ipr)
		switch {
		case parsed == nil:
			continue
		case parsed.Prefix:
			reservation.Prefixes = append(reservation.Prefixes, parsed.NetworkAddress)
		case parsed.Protocol == storkutil.IPv6:
			reservation.IPAddresses = append(reservation.IPAddresses, parsed.NetworkAddress)
		case len(reservation.IPAddress) == 0:
			reservation.IPAddress = parsed.NetworkAddress
		}
	}
	for _, option := range host.GetDHCPOptions(daemonID) {
		optionData, err := CreateSingleOptionData(daemonID, lookup, option)
		if err != nil {
			return nil, err
		}
		reservation.OptionData = append(reservation.OptionData, *optionData)
	}
	return reservation, nil
}

// Converts a host representation in Stork to Kea host reservation format used
// in host_cmds hook library (includes subnet ID). The daemonID selects host
// reservation data appropriate for a given daemon. Note that a host in Stork
// can be shared by multiple daemons. The lookup interface must not be nil.
func CreateHostCmdsReservation(daemonID int64, lookup DHCPOptionDefinitionLookup, host Host) (reservation *HostCmdsReservation, err error) {
	var (
		base     *Reservation
		subnetID int64
	)
	if subnetID, err = host.GetSubnetID(daemonID); err != nil {
		return
	}
	base, err = CreateReservation(daemonID, lookup, host)
	if err != nil {
		return
	}
	reservation = &HostCmdsReservation{
		Reservation: *base,
		SubnetID:    subnetID,
	}
	return
}

// Converts a host representation in Stork to a structure accepted by the
// reservation-del command in Kea. This structure comprises a DHCP identifier
// type, DHCP identifier value and a subnet ID.
func CreateHostCmdsDeletedReservation(daemonID int64, host Host) (reservation *HostCmdsDeletedReservation, err error) {
	var subnetID int64
	if subnetID, err = host.GetSubnetID(daemonID); err != nil {
		return
	}
	ids := host.GetHostIdentifiers()
	if len(ids) == 0 {
		err = errors.New("no DHCP identifiers found for the host")
		return
	}
	// Create the reservation using the first found identifier.
	reservation = &HostCmdsDeletedReservation{
		IdentifierType: ids[0].Type,
		Identifier:     storkutil.BytesToHex(ids[0].Value),
		SubnetID:       subnetID,
	}
	return
}
