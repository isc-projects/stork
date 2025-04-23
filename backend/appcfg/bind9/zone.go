package bind9config

// Returns the allow-transfer clause for the zone or nil if it is not found.
func (z *Zone) GetAllowTransfer() *AllowTransfer {
	for _, clause := range z.Clauses {
		if clause.AllowTransfer != nil {
			return clause.AllowTransfer
		}
	}
	return nil
}
