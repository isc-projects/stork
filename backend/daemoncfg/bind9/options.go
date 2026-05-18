package bind9config

import "sync"

var _ formattedElement = (*Options)(nil)

// cachedOption implements a simple get-and-cache mechanism for options.
// When the option is accessed for the first time, it is cached. If the option
// doesn't exist the nil value is cached. In both cases, subsequent calls to
// get the value will return the cached value or cached nil.
type cachedOption[T any] struct {
	// Once is used to ensure that the option is looked up in the option clauses
	// only once.
	once sync.Once
	// Value is the cached value of the option or nil if the option doesn't exist.
	value *T
}

// Retrieves the value of the option from the option clauses, and caches the value
// or nil if the option doesn't exist. The selectorFn must be implemented for each
// cached option type. It must check if the given clause holds the desired option
// and return it.
func (c *cachedOption[T]) get(clauses []*OptionClause, selectorFn func(clause *OptionClause) *T) *T {
	c.once.Do(func() {
		for _, clause := range clauses {
			if value := selectorFn(clause); value != nil {
				c.value = value
			}
		}
	})
	return c.value
}

// Options is the statement used to define global options.
// This section has the following format:
//
//	options {
//		<option-clauses> ...
//	};
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#options-block-grammar.
type Options struct {
	// Cached directory option.
	directoryOption cachedOption[Directory]
	// Cached response-policy option.
	responsePolicyOption cachedOption[ResponsePolicy]
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

// Gets the directory clause from options or nil if it is not found. The result
// of calling this function is cached.
func (o *Options) GetDirectory() *Directory {
	return o.directoryOption.get(o.Clauses, func(clause *OptionClause) *Directory {
		if clause.Directory != nil {
			return clause.Directory
		}
		return nil
	})
}

// Gets the response-policy clause from options or nil if it is not found. The result
// of calling this function is cached because it can be accessed frequently (for each zone
// returned to the server).
func (o *Options) GetResponsePolicy() *ResponsePolicy {
	return o.responsePolicyOption.get(o.Clauses, func(clause *OptionClause) *ResponsePolicy {
		if clause.ResponsePolicy != nil {
			return clause.ResponsePolicy
		}
		return nil
	})
}
