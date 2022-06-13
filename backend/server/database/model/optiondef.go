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
			(option.GetCode() < 101 ||
				(option.GetCode() > 107 && option.GetCode() < 162) ||
				(option.GetCode() > 174 && option.GetCode() < 178) ||
				(option.GetCode() > 207 && option.GetCode() < 214) ||
				(option.GetCode() > 219 && option.GetCode() < 222))
	case storkutil.IPv6:
		return option.GetSpace() == "dhcp6" && option.GetCode() < 144
	}
	return false
}
