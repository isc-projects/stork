package bind9config

import storkutil "isc.org/stork/util"

var _ formattedElement = (*AddressMatchListElement)(nil)

// AddressMatchListElement is an element of an address match list.
type AddressMatchListElement struct {
	Negation           bool              `parser:"@('!')?"`
	AddressMatchList   *AddressMatchList `parser:"( '{' @@ '}'"`
	KeyID              string            `parser:"| ( 'key' ( @Ident | @String ) )"`
	IPAddressOrACLName string            `parser:"| ( @Ident | @String ) )"`
}

func (amle *AddressMatchListElement) IsIPAddress() bool {
	return storkutil.IsIPAddress(amle.IPAddressOrACLName)
}

// Returns the serialized BIND 9 configuration for the address match list element.
func (amle *AddressMatchListElement) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClause()
	if amle.Negation {
		clause.addToken("!")
	}
	switch {
	case amle.AddressMatchList != nil:
		clauseScope := newFormatterScope()
		if amle.AddressMatchList != nil {
			for _, element := range amle.AddressMatchList.Elements {
				clauseScope.add(element.getFormattedOutput(filter))
			}
		}
		clause.add(clauseScope)
	case amle.KeyID != "":
		clause.addTokenf(`key "%s"`, amle.KeyID)
	case amle.IPAddressOrACLName != "":
		clause.addQuotedToken(amle.IPAddressOrACLName)
	}
	return clause
}
