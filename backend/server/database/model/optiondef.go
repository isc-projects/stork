package dbmodel

import (
	keaconfig "isc.org/stork/appcfg/kea"
	storkutil "isc.org/stork/util"
)

// DHCP option definition lookup mechanism. It can check if a definition
// of a given option exists for the specified daemon.
//
// The lookup mechanism is currently very simple. It does not take into
// account any runtime option definitions returned by Kea. It merely
// checks if the specified option is a standard option, and assumes that
// Kea knows its definition if it is a standard option. We are going to
// extend the lookup mechanism to take into account runtime option
// definitions once we gather them from the monitored DHCP servers.
type DHCPOptionDefinitionLookup struct {
}

// Checks if a definition of the specified option exists for the
// given daemon.
func (lookup DHCPOptionDefinitionLookup) DefinitionExists(daemonID int64, option keaconfig.DHCPOption) bool {
	switch option.GetUniverse() {
	case storkutil.IPv4:
		return option.GetSpace() == "dhcp4" &&
			((option.GetCode() >= 1 && option.GetCode() <= 100) ||
				(option.GetCode() >= 108 && option.GetCode() <= 161) ||
				(option.GetCode() >= 175 && option.GetCode() <= 177) ||
				(option.GetCode() >= 208 && option.GetCode() <= 213) ||
				(option.GetCode() >= 220 && option.GetCode() <= 221))
	case storkutil.IPv6:
		return option.GetSpace() == "dhcp6" && option.GetCode() >= 1 && option.GetCode() <= 143
	}
	return false
}
