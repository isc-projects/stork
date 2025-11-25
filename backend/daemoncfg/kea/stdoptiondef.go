package keaconfig

import (
	storkutil "isc.org/stork/util"
)

// Implements lookup mechanism for standard DHCP option definitions.
// Standard options are static. To find a definition, it currently
// performs a full scan but indexing mechanisms will be soon introduced
// to improve performance.
type dhcpStdOptionDefinitionLookup struct {
	v4Defs []dhcpOptionDefinition
	v6Defs []dhcpOptionDefinition
}

// Interface to a lookup mechanism for finding DHCP standard options.
type DHCPStdOptionDefinitionLookup interface {
	// Finds DHCP option definition by code and space.
	FindByCodeSpace(code uint16, space string, universe storkutil.IPType) DHCPOptionDefinition
}

// Creates standard DHCP option definition lookup instance. It prepares
// static lists of standard DHCP options. Right now, only limited set of
// options is supported.
func NewStdDHCPOptionDefinitionLookup() DHCPStdOptionDefinitionLookup {
	lookup := &dhcpStdOptionDefinitionLookup{}
	lookup.v4Defs = getStdDHCPv4OptionDefs()
	lookup.v6Defs = getStdDHCPv6OptionDefs()
	return lookup
}

// Finds a DHCP option definition by option code and space. The last argument
// specifies whether it should look for a DHCPv4 or DHCPv6 option.
func (lookup dhcpStdOptionDefinitionLookup) FindByCodeSpace(code uint16, space string, universe storkutil.IPType) DHCPOptionDefinition {
	var defs []dhcpOptionDefinition
	switch universe {
	case storkutil.IPv4:
		defs = lookup.v4Defs
	case storkutil.IPv6:
		defs = lookup.v6Defs
	}
	// todo: add indexing to this search.
	for _, def := range defs {
		if def.Code == code && def.Space == space {
			return def
		}
	}
	return nil
}
