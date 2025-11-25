package bind9config

var _ formattedElement = (*OptionClause)(nil)

// OptionClause is a single clause of an options statement.
type OptionClause struct {
	// A Stork-specific annotation to skip parsing statements between the
	// @stork:no-parse:scope and @stork:no-parse:end directives, or after
	// the @stork:no-parse:global directive.
	NoParse *NoParse `parser:"@@"`
	// The allow-transfer clause restricting who can perform AXFR.
	AllowTransfer *AllowTransfer `parser:"| 'allow-transfer' @@"`
	// The listen-on or listen-on-v6 clause specifying the addresses the
	// server listens on the DNS requests.
	ListenOn *ListenOn `parser:"| @@"`
	// The response-policy clause specifying the response policy zones.
	ResponsePolicy *ResponsePolicy `parser:"| 'response-policy' @@"`
	// Any option clause.
	Option *Option `parser:"| @@"`
}

// Returns the serialized BIND 9 configuration for the option clause.
func (o *OptionClause) getFormattedOutput(filter *Filter) formatterOutput {
	return getFormatterClauseFromStruct(o, filter)
}
