package bind9config

var (
	_ formattedElement = (*Channel)(nil)
	_ formattedElement = (*ChannelClause)(nil)
)

// Channel is the statement used within the logging scope to specify where the
// messages from a given category are logged. In particular, it may specify the
// log file location, null location or syslog. It can also specify whether or not
// the time, category or severity should be printed.
// The "channel" statement has the following format:
//
//	channel <name> { <channel-clause> ... };
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#the-channel-phrase.
type Channel struct {
	Name    String           `parser:"@@"`
	Clauses []*ChannelClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// Returns the serialized BIND 9 configuration for the channel statement.
func (c *Channel) getFormattedOutput(filter *Filter) formatterOutput {
	channelClause := newFormatterClause()
	channelClause.addToken("channel")
	channelClause.addToken(c.Name.GetOriginalValue())
	channelClauseScope := newFormatterScope()
	for _, clause := range c.Clauses {
		channelClauseScope.add(clause.getFormattedOutput(filter))
	}
	channelClause.add(channelClauseScope)
	return channelClause
}

// Returns the name of the channel or empty string if the channel name
// is not set.
func (c *Channel) GetName() string {
	return c.Name.GetValue()
}

// Returns true if the channel is a file channel.
func (c *Channel) IsFile() bool {
	for _, clause := range c.Clauses {
		if clause.File != nil {
			return true
		}
	}
	return false
}

// Returns the name of the file channel or empty string if the channel is not a file channel.
func (c *Channel) GetFileName() string {
	for _, clause := range c.Clauses {
		if clause.File != nil {
			return clause.File.Name.GetValue()
		}
	}
	return ""
}

// Returns true if the channel is a syslog channel.
func (c *Channel) IsSyslog() bool {
	for _, clause := range c.Clauses {
		if clause.Syslog != nil {
			return true
		}
	}
	return false
}

// Returns true if the channel is a null channel.
func (c *Channel) IsNull() bool {
	for _, clause := range c.Clauses {
		if clause.Null {
			return true
		}
	}
	return false
}

// Returns true if the channel is a stderr channel.
func (c *Channel) IsStderr() bool {
	for _, clause := range c.Clauses {
		if clause.Stderr {
			return true
		}
	}
	return false
}

// ChannelClause is a single clause specifying an option within the channel statement.
// Certain options are parsed into dedicated structures. Options unused in Stork are
// parsed into the generic Option structure.
type ChannelClause struct {
	// The file clause specifying the log file location.
	File *File `parser:"'file' @@"`
	// The syslog clause specifying that logs should be sent to the syslog.
	Syslog *Syslog `parser:"| 'syslog' @@"`
	// The null clause specifying that the messages should be discarded.
	Null bool `parser:"| @('null')"`
	// The stderr clause specifying that the messages should be sent to the stderr.
	Stderr bool `parser:"| @('stderr')"`
	// Other option type (not listed above).
	Option *Option `parser:"| @@"`
}

// Returns the serialized BIND 9 configuration for the channel clause.
func (c *ChannelClause) getFormattedOutput(filter *Filter) formatterOutput {
	switch {
	case c.File != nil:
		return c.File.getFormattedOutput(filter)
	case c.Syslog != nil:
		return c.Syslog.getFormattedOutput(filter)
	case c.Null:
		return newFormatterClause("null")
	case c.Stderr:
		return newFormatterClause("stderr")
	case c.Option != nil:
		return c.Option.getFormattedOutput(filter)
	}
	return nil
}
