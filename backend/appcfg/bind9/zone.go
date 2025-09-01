package bind9config

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
