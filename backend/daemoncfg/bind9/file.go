package bind9config

var _ formattedElement = (*File)(nil)

// File is the clause specifying the log file location.
// The "file" clause has the following format:
//
//	file <quoted_string> [ versions ( unlimited | <integer> ) ] [ size <size> ] [ suffix ( increment | timestamp ) ];
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#the-channel-phrase.
type File struct {
	// The log file name.
	Name *String `parser:"@@"`
	// The remaining log file switches.
	Switches []String `parser:"( @@ )*"`
}

// Returns the serialized BIND 9 configuration for the file clause.
func (c *File) getFormattedOutput(filter *Filter) formatterOutput {
	channelClauseFile := newFormatterClause()
	channelClauseFile.addToken("file")
	// File name should always be quoted.
	channelClauseFile.addTokenf(`"%s"`, c.Name.GetValue())
	for _, s := range c.Switches {
		channelClauseFile.addToken(s.GetOriginalValue())
	}
	return channelClauseFile
}
