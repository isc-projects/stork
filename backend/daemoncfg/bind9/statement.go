package bind9config

var _ formattedElement = (*Statement)(nil)

// Statement is a single top-level configuration element.
type Statement struct {
	// A Stork-specific annotation to skip parsing statements between the
	// @stork:no-parse:scope and @stork:no-parse:end directives, or after
	// the @stork:no-parse:global directive.
	NoParse *NoParse `parser:"@@" filter:"no-parse"`

	// The "include statement is used to include another configuration file.
	Include *Include `parser:"| 'include' @@" filter:"config"`

	// The "acl" statement is used to define an access control list.
	ACL *ACL `parser:"| 'acl' @@" filter:"config"`

	// The "key" statement is used to define a secure key.
	Key *Key `parser:"| 'key' @@" filter:"config"`

	// The "controls" statement is used to define access points for the
	// remote control of the server using rndc.
	Controls *Controls `parser:"| 'controls' @@" filter:"config"`

	// The "statistics-channels" statement is used to define access points for
	// the 	statistics channels.
	StatisticsChannels *StatisticsChannels `parser:"| 'statistics-channels' @@" filter:"config"`

	// The "options" statement is used to define global options.
	Options *Options `parser:"| 'options' @@" filter:"config"`

	// The "view" statement is used to define a view (i.e., a logical
	// DNS server instance)
	View *View `parser:"| 'view' @@" filter:"view"`

	// The "zone" statement is used to define a DNS zone.
	Zone *Zone `parser:"| 'zone' @@" filter:"zone"`

	// Any statement that looks like an option with potentially
	// several switches followed by a block.
	Option *Option `parser:"| @@" filter:"config"`
}

// Returns the serialized BIND 9 configuration for the statement.
func (s *Statement) getFormattedOutput(filter *Filter) formatterOutput {
	return getFormatterClauseFromStruct(s, filter)
}

// Checks if the statement contains no-parse directives.
func (s *Statement) HasNoParse() bool {
	switch {
	case s.NoParse != nil:
		return true
	case s.Zone != nil:
		return s.Zone.HasNoParse()
	case s.View != nil:
		return s.View.HasNoParse()
	case s.Options != nil:
		return s.Options.HasNoParse()
	default:
		return false
	}
}
