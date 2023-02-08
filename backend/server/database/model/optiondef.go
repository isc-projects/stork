package dbmodel

import (
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// DHCP option definition lookup mechanism.
//
// Its capabilities are currently limited. In the near future it will
// be able to search for runtime option definitions in the database. At
// present, it can find some selected standard option definitions for Kea.
type DHCPOptionDefinitionLookup struct {
	keaStdLookup keaconfig.DHCPStdOptionDefinitionLookup
}

// Creates new lookup instance.
func NewDHCPOptionDefinitionLookup() keaconfig.DHCPOptionDefinitionLookup {
	return &DHCPOptionDefinitionLookup{
		keaStdLookup: keaconfig.NewStdDHCPOptionDefinitionLookup(),
	}
}

// Checks if a definition of the specified option exists for the
// given daemon.
func (lookup DHCPOptionDefinitionLookup) DefinitionExists(daemonID int64, option dhcpmodel.DHCPOptionAccessor) bool {
	switch option.GetUniverse() {
	case storkutil.IPv4:
		return (option.GetSpace() == "dhcp4" &&
			((option.GetCode() >= 1 && option.GetCode() <= 100) ||
				(option.GetCode() >= 108 && option.GetCode() <= 161) ||
				(option.GetCode() >= 175 && option.GetCode() <= 177) ||
				(option.GetCode() >= 208 && option.GetCode() <= 213) ||
				(option.GetCode() >= 220 && option.GetCode() <= 221))) ||
			(lookup.Find(daemonID, option) != nil)
	case storkutil.IPv6:
		return (option.GetSpace() == "dhcp6" && option.GetCode() >= 1 && option.GetCode() <= 143) ||
			(lookup.Find(daemonID, option) != nil)
	}
	return false
}

// Finds option definition for the specified option. Internally, it queries standard
// Kea option definitions defined in the keaconfig package. In the future it will also
// be able to search for the runtime definitions in the database.
func (lookup DHCPOptionDefinitionLookup) Find(daemonID int64, option dhcpmodel.DHCPOptionAccessor) keaconfig.DHCPOptionDefinition {
	return lookup.keaStdLookup.FindByCodeSpace(option.GetCode(), option.GetSpace(), option.GetUniverse())
}
