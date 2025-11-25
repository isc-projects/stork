package bind9config

var _ formattedElement = (*AddressMatchListElement)(nil)

// AddressMatchListElement is an element of an address match list.
type AddressMatchListElement struct {
	Negation           bool   `parser:"@('!')?"`
	ACL                *ACL   `parser:"( '{' @@ '}'"`
	KeyID              string `parser:"| ( 'key' ( @Ident | @String ) )"`
	IPAddressOrACLName string `parser:"| ( @Ident | @String ) )"`
}

// Returns the serialized BIND 9 configuration for the address match list element.
func (amle *AddressMatchListElement) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClause()
	if amle.Negation {
		clause.addToken("!")
	}
	switch {
	case amle.ACL != nil:
		clause.addQuotedToken(amle.ACL.Name)
	case amle.KeyID != "":
		clause.addTokenf(`key "%s"`, amle.KeyID)
	case amle.IPAddressOrACLName != "":
		clause.addQuotedToken(amle.IPAddressOrACLName)
	}
	return clause
}
