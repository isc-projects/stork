package bind9config

var _ formattedElement = (*Suboption)(nil)

// Suboption is a generic catch-all clause being an optional part of an
// option. Suboptions can appear after curly braces in the option.
type Suboption struct {
	Identifier string                 `parser:"@Ident"`
	Switches   []OptionSwitch         `parser:"( @@ )*"`
	Contents   *GenericClauseContents `parser:"( '{' @@ '}' )?"`
}

// Returns the serialized BIND 9 configuration for the suboption.
func (s *Suboption) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClause()
	clause.addToken(s.Identifier)
	for _, s := range s.Switches {
		if s.StringSwitch != nil {
			clause.addTokenf(`"%s"`, *s.StringSwitch)
		} else if s.IdentSwitch != nil {
			clause.addToken(*s.IdentSwitch)
		}
	}
	if s.Contents != nil {
		clause.add(s.Contents.getFormattedOutput(filter))
	}
	return clause
}
