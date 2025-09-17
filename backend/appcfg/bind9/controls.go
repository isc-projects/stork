package bind9config

var _ formattedElement = (*Controls)(nil)

// Controls is the statement used to define controls. It has the following format:
//
//	controls {
//		inet ( <ipv4_address> | <ipv6_address> | * ) [ port ( <integer> | * ) ] allow { <address_match_element>; ... } [ keys { <string>; ... } ] [ read-only <boolean> ];
//		unix <quoted_string> perm <integer> owner <integer> group <integer> [ keys { <string>; ... } ] [ read-only <boolean> ];
//	};
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#controls-block-grammar.
type Controls struct {
	Clauses []*ControlClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// An inet or unix clause within the controls statement.
type ControlClause struct {
	InetClause *InetClause `parser:"'inet' @@"`
	UnixClause *UnixClause `parser:"| 'unix' @@"`
}

// Returns the first inet clause from the controls statement or nil if it is not found.
func (c *Controls) GetFirstInetClause() *InetClause {
	for _, clause := range c.Clauses {
		if clause.InetClause != nil {
			return clause.InetClause
		}
	}
	return nil
}

// Returns the serialized BIND 9 configuration for the controls statement.
func (c *Controls) getFormattedOutput(filter *Filter) formatterOutput {
	controlsClause := newFormatterClause("controls")
	scope := controlsClause.addScope()
	for _, clause := range c.Clauses {
		scope.add(getFormatterClauseFromStruct(clause, filter))
	}
	return controlsClause
}
