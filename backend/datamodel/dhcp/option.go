package dhcpmodel

import storkutil "isc.org/stork/util"

// DHCP option space (one of dhcp4 or dhcp6).
type DHCPOptionSpace = string

// Top level DHCP option spaces.
const (
	DHCPv4OptionSpace DHCPOptionSpace = "dhcp4"
	DHCPv6OptionSpace DHCPOptionSpace = "dhcp6"
)

// A common interface to a DHCP option. Database model representing
// DHCP options implements this interface.
type DHCPOptionAccessor interface {
	// Returns a boolean flag indicating if the option should be
	// always returned, regardless whether it is requested or not.
	IsAlwaysSend() bool
	// Returns option code.
	GetCode() uint16
	// Returns encapsulated option space name.
	GetEncapsulate() string
	// Returns option fields.
	GetFields() []DHCPOptionFieldAccessor
	// Returns option name.
	GetName() string
	// Returns option space.
	GetSpace() string
	// Returns the universe (i.e., IPv4 or IPv6).
	GetUniverse() storkutil.IPType
}
