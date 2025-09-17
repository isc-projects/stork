package bind9config

var _ formattedElement = (*ACL)(nil)

// ACL is the statement used to define an access control list.
// The "acl" statement has the following format:
//
//	acl <name> { <address-match-list> };
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#acl-block-grammar.
type ACL struct {
	// The name of the ACL.
	Name string `parser:"( @String | @Ident )"`
	// The list of address match list elements between curly braces.
	AddressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
}

// Returns the serialized BIND 9 configuration for the acl statement.
func (a *ACL) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClausef(`acl "%s"`, a.Name)
	clauseScope := newFormatterScope()
	if a.AddressMatchList != nil {
		for _, element := range a.AddressMatchList.Elements {
			clauseScope.add(element.getFormattedOutput(filter))
		}
	}
	clause.add(clauseScope)
	return clause
}
