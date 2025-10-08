package bind9config

// Returns the first inet clause from the statistics-channels statement or nil if it is not found.
func (c *StatisticsChannels) GetFirstInetClause() *InetClause {
	if len(c.Clauses) > 0 {
		return c.Clauses[0]
	}
	return nil
}
