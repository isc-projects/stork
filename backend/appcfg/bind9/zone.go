package bind9config

var (
	_ formattedElement = (*Zone)(nil)
	_ formattedElement = (*ZoneClause)(nil)
)

// Zone is the statement used to define a zone. The zone has the following format:
//
//	zone <name> [ <class> ] {
//		<zone-clauses> ...
//	};
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#zone-block-grammar.
type Zone struct {
	// The name of the zone statement.
	Name string `parser:"( @String | @Ident )"`
	// The class of the zone statement.
	Class string `parser:"( @String | @Ident )?"`
	// The list of clauses (e.g., match-clients, allow-transfer etc.).
	// This is made optional to allow quicker parsing of the zone definition,
	// with the zone-level options elided.
	Clauses []*ZoneClause `parser:"( '{' ( @@ ';'* )* '}' )?"`
}

// Returns the serialized BIND 9 configuration for the zone statement.
func (z *Zone) getFormattedOutput(filter *Filter) formatterOutput {
	zoneClause := newFormatterClausef(`zone "%s"`, z.Name)
	if z.Class != "" {
		zoneClause.addToken(z.Class)
	}
	zoneClauseScope := newFormatterScope()
	for _, clause := range z.Clauses {
		zoneClauseScope.add(clause.getFormattedOutput(filter))
	}
	zoneClause.add(zoneClauseScope)
	return zoneClause
}

// Checks if the zone contains no-parse directives.
func (z *Zone) HasNoParse() bool {
	for _, clause := range z.Clauses {
		if clause.NoParse != nil {
			return true
		}
	}
	return false
}

// Returns the allow-transfer clause for the zone or nil if it is not found.
func (z *Zone) GetAllowTransfer() *AllowTransfer {
	for _, clause := range z.Clauses {
		if clause.AllowTransfer != nil {
			return clause.AllowTransfer
		}
	}
	return nil
}

// ZoneClause is a single clause of a zone statement.
type ZoneClause struct {
	NoParse *NoParse `parser:"@@"`
	// The allow-transfer clause restricting who can perform AXFR.
	AllowTransfer *AllowTransfer `parser:"| 'allow-transfer' @@"`
	// Any option clause.
	Option *Option `parser:"| @@"`
}

// Returns the serialized BIND 9 configuration for the zone clause.
func (z *ZoneClause) getFormattedOutput(filter *Filter) formatterOutput {
	return getFormatterClauseFromStruct(z, filter)
}
