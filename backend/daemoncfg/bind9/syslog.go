package bind9config

var _ formattedElement = (*Syslog)(nil)

// Syslog is the clause specifying that logs should be sent to the syslog.
// The "syslog" clause has the following format:
//
//	syslog [ <facility> ];
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#namedconf-statement-syslog.
type Syslog struct {
	Facility *String `parser:"@@?"`
}

// Returns the serialized BIND 9 configuration for the syslog clause.
func (s *Syslog) getFormattedOutput(filter *Filter) formatterOutput {
	syslogClause := newFormatterClause()
	syslogClause.addToken("syslog")
	if s.Facility != nil {
		syslogClause.addToken(s.Facility.GetOriginalValue())
	}
	return syslogClause
}
