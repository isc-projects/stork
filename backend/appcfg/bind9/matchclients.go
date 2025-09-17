package bind9config

var _ formattedElement = (*MatchClients)(nil)

// MatchClients is the clause for associations with ACLs. It can be used in the
// view to associate this view with specific ACLs.
type MatchClients struct {
	AddressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
}

// Returns the serialized BIND 9 configuration for the match-clients clause.
func (m *MatchClients) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClause("match-clients")
	clauseScope := clause.addScope()
	if m.AddressMatchList != nil {
		for _, element := range m.AddressMatchList.Elements {
			clauseScope.add(element.getFormattedOutput(filter))
		}
	}
	return clause
}
