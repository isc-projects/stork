package bind9config

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
func (v *View) GetResponsePolicy() *ResponsePolicy {
	for _, clause := range v.Clauses {
		if clause.ResponsePolicy != nil {
			return clause.ResponsePolicy
		}
	}
	return nil
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
