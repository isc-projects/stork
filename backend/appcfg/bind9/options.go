package bind9config

import "sync"

var _ formattedElement = (*Options)(nil)

// Options is the statement used to define global options.
// This section has the following format:
//
//	options {
//		<option-clauses> ...
//	};
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#options-block-grammar.
type Options struct {
	// Cache the response-policy only once.
	responsePolicyOnce sync.Once
	// The response-policy clause cache for better access performance.
	responsePolicy *ResponsePolicy
	// The list of clauses (e.g., allow-transfer, listen-on, response-policy etc.).
	Clauses []*OptionClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// Returns the serialized BIND 9 configuration for the options statement.
func (o *Options) getFormattedOutput(filter *Filter) formatterOutput {
	optionsClause := newFormatterClause()
	optionsClause.addToken("options")
	optionsClauseScope := newFormatterScope()
	for _, clause := range o.Clauses {
		c := clause.getFormattedOutput(filter)
		if c != nil {
			optionsClauseScope.add(c)
		}
	}
	optionsClause.add(optionsClauseScope)
	return optionsClause
}

// Checks if the options contain no-parse directives.
func (o *Options) HasNoParse() bool {
	for _, clause := range o.Clauses {
		if clause.NoParse != nil {
			return true
		}
	}
	return false
}

// Gets the allow-transfer clause from options.
func (o *Options) GetAllowTransfer() *AllowTransfer {
	for _, clause := range o.Clauses {
		if clause.AllowTransfer != nil {
			return clause.AllowTransfer
		}
	}
	return nil
}

// Gets the listen-on and listen-on-v6 clauses from options. The result is
// combined into a single slice.
func (o *Options) GetListenOnSet() *ListenOnClauses {
	var listenOnSet ListenOnClauses
	for _, clause := range o.Clauses {
		if clause.ListenOn != nil {
			listenOnSet = append(listenOnSet, clause.ListenOn)
		}
	}
	return &listenOnSet
}

// Gets the response-policy clause from options or nil if it is not found. The result
// of calling this function is cached because it can be accessed frequently (for each zone
// returned to the server).
func (o *Options) GetResponsePolicy() *ResponsePolicy {
	o.responsePolicyOnce.Do(func() {
		// Return the cached response-policy clause if it was already accessed.
		if o.responsePolicy != nil {
			return
		}
		for _, clause := range o.Clauses {
			if clause.ResponsePolicy != nil {
				// Cache the response-policy clause for better access performance
				// in the future.
				o.responsePolicy = clause.ResponsePolicy
				return
			}
		}
	})
	return o.responsePolicy
}
