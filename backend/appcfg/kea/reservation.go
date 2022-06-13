package keaconfig

import (
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
	GetDHCPOptions(int64) []DHCPOption
}

// Represents host reservation within Kea configuration.
type Reservation struct {
	HWAddress   string             `mapstructure:"hw-address" json:"hw-address,omitempty"`
	DUID        string             `mapstructure:"duid" json:"duid,omitempty"`
	CircuitID   string             `mapstructure:"circuit-id" json:"circuit-id,omitempty"`
	ClientID    string             `mapstructure:"client-id" json:"client-id,omitempty"`
	FlexID      string             `mapstructure:"flex-id" json:"flex-id,omitempty"`
	IPAddress   string             `mapstructure:"ip-address" json:"ip-address,omitempty"`
	IPAddresses []string           `mapstructure:"ip-addresses" json:"ip-addresses,omitempty"`
	Prefixes    []string           `mapstructure:"prefixes" json:"prefixes,omitempty"`
	Hostname    string             `mapstructure:"hostname" json:"hostname,omitempty"`
	OptionData  []SingleOptionData `mapstructure:"option-data" json:"option-data,omitempty"`
}

// Represents host reservation returned and sent via Kea host commands hook library.
type HostCmdsReservation struct {
	Reservation
	SubnetID int64 `json:"subnet-id"`
}

// Converts a host representation in Stork to Kea host reservation format used
// in Kea configuration.
func CreateReservation(daemonID int64, lookup DHCPOptionDefinitionLookup, host Host) (*Reservation, error) {
	reservation := &Reservation{
		Hostname: host.GetHostname(),
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
// can be shared by multiple daemons.
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
