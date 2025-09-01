package bind9config

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
