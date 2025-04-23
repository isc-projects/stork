package bind9config

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
