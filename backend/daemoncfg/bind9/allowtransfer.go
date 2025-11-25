package bind9config

import (
	"slices"
)

var _ formattedElement = (*AllowTransfer)(nil)

// AllowTransfer is an option for restricting who can perform AXFR
// globally, for a particular view or zone.
//
// The allow-transfer clause has the following format:
//
//	allow-transfer [ port <integer> ] [ transport <string> ] { <address_match_element>; ... };
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#namedconf-statement-allow-transfer
type AllowTransfer struct {
	Port             *int64            `parser:"( 'port' @Ident )?"`
	Transport        *string           `parser:"( 'transport' ( @String | @Ident ) )?"`
	AddressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
}

// Checks if the zone transfer is disabled.
func (at *AllowTransfer) IsDisabled() bool {
	// By default, the transfer is disabled. It is also disabled when it is none.
	// If any of the elements is not none, the transfer is enabled.
	return len(at.AddressMatchList.Elements) == 0 || !slices.ContainsFunc(at.AddressMatchList.Elements, func(ame *AddressMatchListElement) bool {
		return ame.IPAddressOrACLName != "none"
	})
}

// Returns the serialized BIND 9 configuration for the allow-transfer clause.
func (at *AllowTransfer) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClause("allow-transfer")
	if at.Port != nil {
		clause.addTokenf("port %d", *at.Port)
	}
	if at.Transport != nil {
		clause.addTokenf("transport %s", *at.Transport)
	}
	allowTransferClauseScope := clause.addScope()
	if at.AddressMatchList != nil {
		for _, element := range at.AddressMatchList.Elements {
			allowTransferClauseScope.add(element.getFormattedOutput(filter))
		}
	}
	return clause
}
