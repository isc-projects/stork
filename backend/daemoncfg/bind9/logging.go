package bind9config

import storkutil "isc.org/stork/util"

var (
	_ formattedElement = (*Logging)(nil)
	_ formattedElement = (*LoggingClause)(nil)
)

// Logging is the statement used to define logging configuration.
// The "logging" statement has the following format:
//
//	logging {
//		category <string> { <string>; ... }; // may occur multiple times
//		channel <string> { ...	}; // may occur multiple times
//	}
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#logging-block-grammar.
type Logging struct {
	// The list of category and channel clauses.
	Clauses []*LoggingClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// Returns the serialized BIND 9 configuration for the logging statement.
func (l *Logging) getFormattedOutput(filter *Filter) formatterOutput {
	loggingClause := newFormatterClause()
	loggingClause.addToken("logging")
	loggingClauseScope := newFormatterScope()
	for _, clause := range l.Clauses {
		c := clause.getFormattedOutput(filter)
		if c != nil {
			loggingClauseScope.add(c)
		}
	}
	loggingClause.add(loggingClauseScope)
	return loggingClause
}

// Instantiates a default debug channel as specified in the BIND 9 documentation.
func newDefaultDebugChannel() *Channel {
	return &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("default_debug"),
		},
		Clauses: []*ChannelClause{
			{
				File: &File{
					Name: &String{
						Unquoted: storkutil.Ptr("named.run"),
					},
				},
			},
		},
	}
}

// Instantiates a default syslog channel as specified in the BIND 9 documentation.
func newDefaultSyslogChannel() *Channel {
	return &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("default_syslog"),
		},
		Clauses: []*ChannelClause{
			{
				Syslog: &Syslog{
					Facility: &String{
						Unquoted: storkutil.Ptr("daemon"),
					},
				},
			},
		},
	}
}

// Instantiates a default stderr channel as specified in the BIND 9 documentation.
func newDefaultStderrChannel() *Channel {
	return &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("default_stderr"),
		},
		Clauses: []*ChannelClause{
			{
				Stderr: true,
			},
		},
	}
}

// Instantiates a default null channel as specified in the BIND 9 documentation.
func newNullChannel() *Channel {
	return &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("null"),
		},
		Clauses: []*ChannelClause{
			{
				Null: true,
			},
		},
	}
}

// Instantiates a new default log file channel with the default log file name
// specified as the parameter.
func newDefaultLogFileChannel(defaultLogFile string) *Channel {
	return &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("default_logfile"),
		},
		Clauses: []*ChannelClause{
			{
				File: &File{
					Name: &String{
						Unquoted: storkutil.Ptr(defaultLogFile),
					},
				},
			},
		},
	}
}

// Returns the channels for the given logging category. It walks over the explicitly
// specified channels. Then, it tries to find the category by name and associates the
// channel names in the category with the channels. If the category is not explicitly
// specified in the logging statement, it returns the channels for the default category.
// If that category is also not specified, it returns empty list. The defaultLogFile
// parameter should be specified when named was started with the -L option. In that case,
// the default_logfile channel is returned instead of default_syslog when nothing else fits.
// The receiver parameter can be nil, in which case the default channels are returned.
func (l *Logging) getChannelsForCategory(categoryName string, defaultLogFile string) []*Channel {
	// Initialize the default channels. Some categories may reference these
	// channels directly.
	defaultDebugChannel := newDefaultDebugChannel()
	defaultSyslogChannel := newDefaultSyslogChannel()
	defaultStderrChannel := newDefaultStderrChannel()
	nullChannel := newNullChannel()
	defaultLogfileChannel := newDefaultLogFileChannel(defaultLogFile)

	// Create a map of all channels, so we can map the categories to channels.
	allChannels := make(map[string]*Channel, 5)
	allChannels[defaultDebugChannel.Name.GetValue()] = defaultDebugChannel
	allChannels[defaultSyslogChannel.Name.GetValue()] = defaultSyslogChannel
	allChannels[defaultStderrChannel.Name.GetValue()] = defaultStderrChannel
	allChannels[nullChannel.Name.GetValue()] = nullChannel
	allChannels[defaultLogfileChannel.Name.GetValue()] = defaultLogfileChannel

	// Collect the channels for the selected category.
	var (
		channels []*Channel
		clauses  []*LoggingClause
	)
	if l != nil {
		clauses = l.Clauses
	}
	for _, clause := range clauses {
		if clause.Channel != nil {
			allChannels[clause.Channel.Name.GetValue()] = clause.Channel
		}
	}
	// Map the categories to channels.
	for _, clause := range clauses {
		if clause.Category != nil && clause.Category.Name.GetValue() == categoryName {
			for _, channelName := range clause.Category.Channels {
				if channel, ok := allChannels[channelName.GetValue()]; ok {
					channels = append(channels, channel)
				}
			}
			break
		}
	}
	// If we found some channels already, return them. It appears that the admin
	// specified the channels explicitly.
	if len(channels) != 0 {
		return channels
	}
	if categoryName != "default" {
		// If the current category is not the default but we found no channels,
		// let's return the default channel settings, if specified.
		return l.getChannelsForCategory("default", defaultLogFile)
	}
	if defaultLogFile != "" {
		// We're asking about the default category that wasn't specified explicitly.
		// We can return the default channels, but the channels depend on the runtime
		// settings. Specifically, if the admin specified named -L option, the default
		// log file channel is used.
		return []*Channel{defaultLogfileChannel, defaultDebugChannel}
	}
	// Without the -L option, the default is to use syslog.
	return []*Channel{defaultSyslogChannel, defaultDebugChannel}
}

// Returns the channels for the given logging category. This function should be called
// when named was not started with the -L option. In that case, the default_syslog channel
// is returned when nothing else fits.
func (l *Logging) GetChannelsForCategory(categoryName string) []*Channel {
	return l.getChannelsForCategory(categoryName, "")
}

// Returns the channels for the given logging category and the default log file.
// The default log file should be specified when named was started with the -L option.
// In that case, the default_logfile channel is returned instead of default_syslog when
// nothing else fits.
func (l *Logging) GetChannelsForCategoryWithDefaultFile(categoryName string, defaultLogFile string) []*Channel {
	return l.getChannelsForCategory(categoryName, defaultLogFile)
}

// LoggingClause is a single clause specifying an option within the logging statement.
// It is one of the category or channel clause.
type LoggingClause struct {
	Category *Category `parser:"'category' @@"`
	Channel  *Channel  `parser:"| 'channel' @@"`
}

// Returns the serialized BIND 9 configuration for the logging clause.
func (l *LoggingClause) getFormattedOutput(filter *Filter) formatterOutput {
	switch {
	case l.Category != nil:
		// It is a category clause.
		return l.Category.getFormattedOutput(filter)
	case l.Channel != nil:
		// It is a channel clause.
		return l.Channel.getFormattedOutput(filter)
	}
	return nil
}
