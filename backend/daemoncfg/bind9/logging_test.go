package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the logging statement is formatted correctly.
func TestLoggingFormat(t *testing.T) {
	logging := &Logging{
		Clauses: []*LoggingClause{
			{
				Channel: &Channel{
					Name: String{
						Unquoted: storkutil.Ptr("docker"),
					},
					Clauses: []*ChannelClause{
						{
							Stderr: true,
						},
					},
				},
			},
			{
				Channel: &Channel{
					Name: String{
						Unquoted: storkutil.Ptr("default"),
					},
					Clauses: []*ChannelClause{
						{
							Null: true,
						},
					},
				},
			},
			{
				Category: &Category{
					Name: String{
						Unquoted: storkutil.Ptr("xfer-out"),
					},
					Channels: []String{
						{
							Unquoted: storkutil.Ptr("docker"),
						},
						{
							Unquoted: storkutil.Ptr("default"),
						},
					},
				},
			},
			{
				Category: &Category{
					Name: String{
						Unquoted: storkutil.Ptr("xfer-in"),
					},
					Channels: []String{
						{
							Unquoted: storkutil.Ptr("docker"),
						},
					},
				},
			},
		},
	}
	output := logging.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `logging {
		channel docker {
			stderr;
		};
		channel default {
			null;
		};
		category xfer-out {
			docker;
			default;
		};
		category xfer-in {
			docker;
		};
	};`, output)
}

// Test that for explicitly specified category, the channels associated with that category
// are returned.
func TestLoggingGetChannelsForCategory(t *testing.T) {
	logging := &Logging{
		Clauses: []*LoggingClause{
			{
				Channel: &Channel{
					Name: String{
						Unquoted: storkutil.Ptr("docker"),
					},
					Clauses: []*ChannelClause{
						{
							Stderr: true,
						},
					},
				},
			},
			{
				Channel: &Channel{
					Name: String{
						Unquoted: storkutil.Ptr("default"),
					},
					Clauses: []*ChannelClause{
						{
							Null: true,
						},
					},
				},
			},
			{
				Category: &Category{
					Name: String{
						Unquoted: storkutil.Ptr("xfer-out"),
					},
					Channels: []String{
						{
							Unquoted: storkutil.Ptr("docker"),
						},
						{
							Unquoted: storkutil.Ptr("default"),
						},
					},
				},
			},
		},
	}
	channels := logging.GetChannelsForCategory("xfer-out")
	require.NotNil(t, channels)
	require.Equal(t, 2, len(channels))
	require.Equal(t, "docker", channels[0].GetName())
	require.True(t, true, channels[0].IsStderr())
	require.Equal(t, "default", channels[1].GetName())
	require.True(t, true, channels[1].IsNull())
}

// Test the case that the category can be associated with the default channels,
// and that the default channels are returned.
func TestLoggingGetChannelsForCategoryWithDefaultChannels(t *testing.T) {
	tests := []string{
		"default_debug",
		"default_syslog",
		"default_stderr",
		"null",
		"default_logfile",
	}
	for _, test := range tests {
		logging := &Logging{
			Clauses: []*LoggingClause{
				{
					Category: &Category{
						Name: String{
							Unquoted: storkutil.Ptr("xfer-in"),
						},
						Channels: []String{
							{
								Unquoted: storkutil.Ptr(test),
							},
						},
					},
				},
			},
		}
		channels := logging.GetChannelsForCategory("xfer-in")
		require.NotNil(t, channels)
		require.Equal(t, 1, len(channels))
		require.Equal(t, test, channels[0].GetName())
	}
}

// Test that the default category can be explicitly configured to use a specified channel,
// and that this channel is returned when specified category does not have its own configuration.
// In that case, the default channels are returned.
func TestLoggingGetChannelsForDefaultCategory(t *testing.T) {
	logging := &Logging{
		Clauses: []*LoggingClause{
			{
				Channel: &Channel{
					Name: String{
						Unquoted: storkutil.Ptr("docker"),
					},
					Clauses: []*ChannelClause{
						{
							Stderr: true,
						},
					},
				},
			},
			{
				Category: &Category{
					Name: String{
						Unquoted: storkutil.Ptr("default"),
					},
					Channels: []String{
						{
							Unquoted: storkutil.Ptr("docker"),
						},
					},
				},
			},
		},
	}
	channels := logging.GetChannelsForCategory("xfer-in")
	require.NotNil(t, channels)
	require.Len(t, channels, 1)
	require.Equal(t, "docker", channels[0].GetName())
	require.True(t, true, channels[0].IsStderr())
}

// Test that when neither the category nor the default category is configured,
// nor the -L flag is used, the default_syslog and default_debug channels are returned.
func TestLoggingGetChannelsNoDefaultCategory(t *testing.T) {
	logging := &Logging{}
	channels := logging.GetChannelsForCategory("xfer-in")
	require.Len(t, channels, 2)
	require.Equal(t, "default_syslog", channels[0].GetName())
	require.True(t, channels[0].IsSyslog())
	require.Equal(t, "default_debug", channels[1].GetName())
	require.True(t, channels[1].IsFile())
}

// Test that default channels for a category are returned when the logging
// statement is nil.
func TestLoggingGetChannelsForNilLogging(t *testing.T) {
	var logging *Logging
	channels := logging.GetChannelsForCategory("xfer-in")
	require.Len(t, channels, 2)
	require.Equal(t, "default_syslog", channels[0].GetName())
	require.True(t, channels[0].IsSyslog())
	require.Equal(t, "default_debug", channels[1].GetName())
	require.True(t, channels[1].IsFile())
}

// Test that when the -L flag is used, the default_logfile and default_debug channels are returned.
func TestLoggingGetChannelsNoDefaultCategoryWithDefaultLogFile(t *testing.T) {
	logging := &Logging{}
	channels := logging.GetChannelsForCategoryWithDefaultFile("xfer-in", "/var/log/named.log")
	require.Len(t, channels, 2)
	require.Equal(t, "default_logfile", channels[0].GetName())
	require.True(t, channels[0].IsFile())
	require.Equal(t, "default_debug", channels[1].GetName())
	require.True(t, channels[1].IsFile())
}
