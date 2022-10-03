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
	lookup.v6Defs = []dhcpOptionDefinition{
		{
			Code:        94,
			Encapsulate: "s46-cont-mape-options",
			Name:        "s46-cont-mape",
			Space:       "dhcp6",
			OptionType:  EmptyOption,
		},
		{
			Code:        95,
			Encapsulate: "s46-cont-mapt-options",
			Name:        "s46-cont-mapt",
			Space:       "dhcp6",
			OptionType:  EmptyOption,
		},
		{
			Code:        96,
			Encapsulate: "s46-cont-lw-options",
			Name:        "s46-cont-lw",
			Space:       "dhcp6",
			OptionType:  EmptyOption,
		},
		{
			Code:        90,
			Encapsulate: "",
			Name:        "s46-br",
			Space:       "s46-cont-mape-options",
			OptionType:  IPv6AddressOption,
		},
		{
			Code:        89,
			Encapsulate: "s46-rule-options",
			Name:        "s46-rule",
			RecordTypes: []DHCPOptionType{
				Uint8Option,
				Uint8Option,
				Uint8Option,
				IPv4AddressOption,
				IPv6PrefixOption,
			},
			Space:      "s46-cont-mape-options",
			OptionType: RecordOption,
		},
		{
			Code:        89,
			Encapsulate: "s46-rule-options",
			Name:        "s46-rule",
			RecordTypes: []DHCPOptionType{
				Uint8Option,
				Uint8Option,
				Uint8Option,
				IPv4AddressOption,
				IPv6PrefixOption,
			},
			Space:      "s46-cont-mapt-options",
			OptionType: RecordOption,
		},
		{
			Code:        91,
			Encapsulate: "",
			Name:        "s46-dmr",
			Space:       "s46-cont-mapt-options",
			OptionType:  IPv6PrefixOption,
		},
		{
			Code:        90,
			Encapsulate: "",
			Name:        "s46-br",
			Space:       "s46-cont-lw-options",
			OptionType:  IPv6AddressOption,
		},
		{
			Code:        92,
			Encapsulate: "s46-v4v6bind-options",
			Name:        "s46-v4v6bind",
			RecordTypes: []DHCPOptionType{
				IPv4AddressOption,
				IPv6PrefixOption,
			},
			Space:      "s46-cont-lw-options",
			OptionType: RecordOption,
		},
		{
			Code:        93,
			Encapsulate: "",
			Name:        "s46-portparams",
			RecordTypes: []DHCPOptionType{
				Uint8Option,
				PsidOption,
			},
			Space:      "s46-rule-options",
			OptionType: RecordOption,
		},
		{
			Code:        93,
			Encapsulate: "",
			Name:        "s46-portparams",
			RecordTypes: []DHCPOptionType{
				Uint8Option,
				PsidOption,
			},
			Space:      "s46-v4v6bind-options",
			OptionType: RecordOption,
		},
	}
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
