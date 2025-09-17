package bind9config

import (
	"sync"
)

var (
	_ formattedElement = (*View)(nil)
	_ formattedElement = (*ViewClause)(nil)
)

// View is the statement used to define a DNS view. The view is a logical
// DNS server instance including its own set of zones. The view has the
// following format:
//
//	view <name> [ <class> ] {
//		<view-clauses> ...
//	};
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#view-block-grammar.
type View struct {
	// Cache the response-policy only once.
	responsePolicyOnce sync.Once
	// The response-policy clause cache for better access performance.
	responsePolicy *ResponsePolicy
	// The name of the view statement.
	Name string `parser:"( @String | @Ident )"`
	// An optional class of the view statement.
	Class string `parser:"( @String | @Ident )?"`
	// The list of clauses (e.g., match-clients, zone etc.).
	Clauses []*ViewClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// ViewClause is a single clause of a view statement.
type ViewClause struct {
	// A Stork-specific annotation to skip parsing statements between the
	// @stork:no-parse:scope and @stork:no-parse:end directives, or after
	// the @stork:no-parse:global directive.
	NoParse *NoParse `parser:"@@" filter:"no-parse"`
	// The match-clients clause associating the view with ACLs.
	MatchClients *MatchClients `parser:"| 'match-clients' @@" filter:"view"`
	// The allow-transfer clause restricting who can perform AXFR.
	AllowTransfer *AllowTransfer `parser:"| 'allow-transfer' @@" filter:"view"`
	// The response-policy clause specifying the response policy zones.
	ResponsePolicy *ResponsePolicy `parser:"| 'response-policy' @@" filter:"view"`
	// The zone clause associating the zone with a view.
	Zone *Zone `parser:"| 'zone' @@" filter:"zone"`
	// Any option clause.
	Option *Option `parser:"| @@" filter:"view"`
}

// Returns the serialized BIND 9 configuration for the view statement.
func (v *View) getFormattedOutput(filter *Filter) formatterOutput {
	viewClause := newFormatterClausef(`view "%s"`, v.Name)
	if v.Class != "" {
		viewClause.addToken(v.Class)
	}
	viewClauseScope := newFormatterScope()
	for _, clause := range v.Clauses {
		viewClauseScope.add(clause.getFormattedOutput(filter))
	}
	viewClause.add(viewClauseScope)
	return viewClause
}

// Checks if the view contains no-parse directives.
func (v *View) HasNoParse() bool {
	for _, clause := range v.Clauses {
		switch {
		case clause.NoParse != nil:
			return true
		case clause.Zone != nil:
			return clause.Zone.HasNoParse()
		}
	}
	return false
}

// Returns the allow-transfer clause for the view or nil if it is not found.
func (v *View) GetAllowTransfer() *AllowTransfer {
	for _, clause := range v.Clauses {
		if clause.AllowTransfer != nil {
			return clause.AllowTransfer
		}
	}
	return nil
}

// Returns the match-clients clause for the view or nil if it is not found.
func (v *View) GetMatchClients() *MatchClients {
	for _, clause := range v.Clauses {
		if clause.MatchClients != nil {
			return clause.MatchClients
		}
	}
	return nil
}

// Returns the response-policy clause for the view or nil if it is not found.
// The result of calling this function is cached because it can be accessed
// frequently (for each zone returned to the server).
func (v *View) GetResponsePolicy() *ResponsePolicy {
	v.responsePolicyOnce.Do(func() {
		// Return the cached response-policy clause if it was already accessed.
		if v.responsePolicy != nil {
			return
		}
		for _, clause := range v.Clauses {
			if clause.ResponsePolicy != nil {
				v.responsePolicy = clause.ResponsePolicy
				return
			}
		}
	})
	return v.responsePolicy
}

// Returns the zone with the specified name or nil if the zone is not found.
func (v *View) GetZone(zoneName string) *Zone {
	for _, clause := range v.Clauses {
		if clause.Zone != nil && clause.Zone.Name == zoneName {
			return clause.Zone
		}
	}
	return nil
}

// Returns the serialized BIND 9 configuration for the view clause.
func (v *ViewClause) getFormattedOutput(filter *Filter) formatterOutput {
	return getFormatterClauseFromStruct(v, filter)
}
