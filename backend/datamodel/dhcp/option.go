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
	// Returns a boolean flag indicating if the option should never
	// be returned to a DHCP client.
	IsNeverSend() bool
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
	// Returns a list of client classes associated with this option.
	// See: https://kea.readthedocs.io/en/kea-3.0.0/arm/classify.html#option-class-tagging
	GetClientClasses() []string
	// Returns unknown (unsupported by Stork) parameters.
	GetUnknownParameters() map[string]any
}
