package bind9config

var _ formattedElement = (*Option)(nil)

// Option is a generic catch-all option clause. It is used to parse an option
// having the following format:
//
//	<identifier> <block>
//
// Many options in the options statement have this format.
type Option struct {
	Identifier string                 `parser:"@Ident"`
	Switches   []String               `parser:"( @@ )*"`
	Contents   *GenericClauseContents `parser:"( '{' @@ '}' )?"`
	Suboptions []Suboption            `parser:"( @@ )*"`
}

// Returns the serialized BIND 9 configuration for the option statement.
func (o *Option) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClause()
	clause.addToken(o.Identifier)
	for _, s := range o.Switches {
		clause.addToken(s.GetOriginalValue())
	}
	if o.Contents != nil {
		clause.add(o.Contents.getFormattedOutput(filter))
	}
	if len(o.Suboptions) > 0 {
		for _, s := range o.Suboptions {
			clause.add(s.getFormattedOutput(filter))
		}
	}
	return clause
}
