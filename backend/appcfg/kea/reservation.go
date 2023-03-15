package keaconfig

import (
	"github.com/pkg/errors"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// Interface of a host convertible to a Kea host reservation.
type HostAccessor interface {
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
	HWAddress      string             `json:"hw-address,omitempty"`
	DUID           string             `json:"duid,omitempty"`
	CircuitID      string             `json:"circuit-id,omitempty"`
	ClientID       string             `json:"client-id,omitempty"`
	FlexID         string             `json:"flex-id,omitempty"`
	IPAddress      string             `json:"ip-address,omitempty"`
	IPAddresses    []string           `json:"ip-addresses,omitempty"`
	Prefixes       []string           `json:"prefixes,omitempty"`
	Hostname       string             `json:"hostname,omitempty"`
	ClientClasses  []string           `json:"client-classes,omitempty"`
	NextServer     string             `json:"next-server,omitempty"`
	BootFileName   string             `json:"boot-file-name,omitempty"`
	ServerHostname string             `json:"server-hostname,omitempty"`
	OptionData     []SingleOptionData `json:"option-data,omitempty"`
}

// Represents host reservation returned and sent via Kea host commands hook library.
type HostCmdsReservation struct {
	Reservation
	SubnetID int64 `json:"subnet-id"`
}

// Represents deleted host reservation. It includes the fields required by
// Kea to find the reservation and delete it.
type HostCmdsDeletedReservation struct {
	IdentifierType string `json:"identifier-type,omitempty"`
	Identifier     string `json:"identifier,omitempty"`
	SubnetID       int64  `json:"subnet-id"`
}

// A structure comprising host reservation modes at the particular
// configuration level. This structure can be embedded in the
// structures for decoding subnets and shared networks. In that
// case, the reservation modes configured at the subnet or shared
// network level will be decoded into the embedded structure.
// Do not read the decoded modes directly from the structure.
// Call appropriate functions on this structure to test the
// decoded modes. The Deprecated field holds the value of the
// reservation-mode setting that was deprecated since Kea 1.9.x.
type ReservationModes struct {
	OutOfPool  *bool   `json:"reservations-out-of-pool,omitempty"`
	InSubnet   *bool   `json:"reservations-in-subnet,omitempty"`
	Global     *bool   `json:"reservations-global,omitempty"`
	Deprecated *string `json:"reservation-mode,omitempty"`
}

// Converts a host representation in Stork to Kea host reservation format used
// in Kea configuration. The lookup interface must not be nil.
func CreateReservation(daemonID int64, lookup DHCPOptionDefinitionLookup, host HostAccessor) (*Reservation, error) {
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
func CreateHostCmdsReservation(daemonID int64, lookup DHCPOptionDefinitionLookup, host HostAccessor) (reservation *HostCmdsReservation, err error) {
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
func CreateHostCmdsDeletedReservation(daemonID int64, host HostAccessor) (reservation *HostCmdsDeletedReservation, err error) {
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

// Checks if the global reservation mode has been enabled.
// Returns (first parameter):
// - reservations-global value if set OR
// - true when reservation-mode is "global".
// The second parameter indicates whether the returned value was set
// explicitly (when true) or is a default value (when false).
func (modes *ReservationParameters) IsGlobal() (bool, bool) {
	if modes.ReservationsGlobal != nil {
		return *modes.ReservationsGlobal, true
	}
	if modes.ReservationMode != nil {
		return *modes.ReservationMode == "global", true
	}
	return false, false
}

// Checks if the in-subnet reservation mode has been enabled.
// Returns (first parameter):
// - reservations-in-subnet value if set OR
// - true when reservation-mode is set and is "all" or "out-of-pool" OR
// - false when reservation-mode is set and configured to other values OR
// - true when no mode is explicitly configured.
// The second parameter indicates whether the returned value was set
// explicitly (when true) or is a default value (when false).
func (modes *ReservationParameters) IsInSubnet() (bool, bool) {
	if modes.ReservationsInSubnet != nil {
		return *modes.ReservationsInSubnet, true
	}
	if modes.ReservationMode != nil {
		return *modes.ReservationMode == "all" || *modes.ReservationMode == "out-of-pool", true
	}
	return true, false
}

// Checks if the out-of-pool reservation mode has been enabled.
// Returns (first parameter):
// - reservations-out-of-pool value if set OR,
// - true when reservation-mode is "out-of-pool",
// - false otherwise.
// The second parameter indicates whether the returned value was set
// explicitly (when true) or is a default value (when false).
func (modes *ReservationParameters) IsOutOfPool() (bool, bool) {
	if modes.ReservationsOutOfPool != nil {
		return *modes.ReservationsOutOfPool, true
	}
	if modes.ReservationMode != nil {
		return *modes.ReservationMode == "out-of-pool", true
	}
	return false, false
}

// Convenience function used to check if a given host reservation
// mode has been enabled at one of the levels at which the
// reservation mode can be configured. The reservation modes specified
// using the variadic parameters should be ordered from the lowest to
// highest configuration level, e.g., subnet-level, shared network-level,
// and finally global-level host reservation configuration. The first
// argument is a function implementing a condition to be checked for
// each ReservationModes. The example condition is:
//
//	func (modes ReservationParameters) (bool, bool) {
//		return modes.IsOutOfPool()
//	}
//
// The function returns true when the condition function returns
// (true, true) for one of the N-1 reservation modes. If it doesn't,
// it returns true when the last reservation mode returns (true, true)
// or (true, false).
//
// Note that this function handles Kea configuration inheritance scheme.
// It checks for explicitly set values at subnet and shared network levels
// which override the global-level setting. The global-level setting
// applies regardless whether or not it is specified. If it is not
// specified a default value is used.
func IsInAnyReservationModes(condition func(modes ReservationParameters) (bool, bool), modes ...ReservationParameters) bool {
	for i, mode := range modes {
		cond, explicit := condition(mode)
		if cond && (explicit || i >= len(modes)-1) {
			return true
		}
	}
	return false
}
