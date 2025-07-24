package bind9config

// Returns the inet clause from the controls statement or nil if it is not found.
func (c *Controls) GetInetClause() *InetClause {
	for _, clause := range c.Clauses {
		if clause.InetClause != nil {
			return clause.InetClause
		}
	}
	return nil
}
