package bind9config

var _ formattedElement = (*Keys)(nil)

// Keys is a list of key names in the inet and unix clauses.
type Keys struct {
	KeyNames []string `parser:"(( @Ident | @String )';')*"`
}

// Returns the serialized BIND 9 configuration for the keys clause.
func (k *Keys) getFormattedOutput(filter *Filter) formatterOutput {
	keysClause := newFormatterClause("keys")
	scope := keysClause.addScope()
	for _, key := range k.KeyNames {
		scope.add(newFormatterClausef(`"%s"`, key))
	}
	return keysClause
}
