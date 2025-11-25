package bind9config

var _ formattedElement = (*StatisticsChannels)(nil)

// A statistics-channels statement is used to define access points for
// the statistics channels.
//
//	statistics-channels {
//		inet ( <ipv4_address> | <ipv6_address> | * ) [ port ( <integer> | * ) ] allow { <address_match_element>; ... };
//	};
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#statistics-channels-block-grammar.
type StatisticsChannels struct {
	Clauses []*InetClause `parser:"'{' ( 'inet' @@ ';'* )* '}'"`
}

// Returns the first inet clause from the statistics-channels statement or nil if it is not found.
func (c *StatisticsChannels) GetFirstInetClause() *InetClause {
	if len(c.Clauses) > 0 {
		return c.Clauses[0]
	}
	return nil
}

// Returns the serialized BIND 9 configuration for the statistics-channels statement.
func (c *StatisticsChannels) getFormattedOutput(filter *Filter) formatterOutput {
	statisticsChannelsClause := newFormatterClause("statistics-channels")
	scope := statisticsChannelsClause.addScope()
	for _, clause := range c.Clauses {
		scope.add(clause.getFormattedOutput(filter))
	}
	return statisticsChannelsClause
}
