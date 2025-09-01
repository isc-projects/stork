package bind9config

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
		} else if clause.ListenOnV6 != nil {
			listenOnSet = append(listenOnSet, clause.ListenOnV6)
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
