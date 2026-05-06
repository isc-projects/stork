package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the channel statement is formatted correctly when file
// and option clause are present.
func TestChannelFormat(t *testing.T) {
	channel := &Channel{
		Name: String{
			Quoted: storkutil.Ptr("default"),
		},
		Clauses: []*ChannelClause{
			{
				File: &File{
					Name: &String{
						Quoted: storkutil.Ptr("/var/lib/log/default"),
					},
					Switches: []String{
						{
							Unquoted: storkutil.Ptr("versions"),
						},
						{
							Quoted: storkutil.Ptr("3"),
						},
						{
							Unquoted: storkutil.Ptr("size"),
						},
						{
							Quoted: storkutil.Ptr("10M"),
						},
					},
				},
			},
			{
				Option: &Option{
					Identifier: "print-time",
					Switches: []OptionSwitch{
						{
							IdentSwitch: storkutil.Ptr("yes"),
						},
					},
				},
			},
		},
	}
	output := channel.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `channel "default" {
		file "/var/lib/log/default" versions "3" size "10M";
		print-time yes;
	};`, output)
}

// Test that the channel statement is formatted correctly when syslog
// clause with facility is specified.
func TestChannelFormatSyslogWithFacility(t *testing.T) {
	channel := &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				Syslog: &Syslog{
					Facility: &String{
						Unquoted: storkutil.Ptr("local7"),
					},
				},
			},
		},
	}
	output := channel.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `channel docker {
		syslog local7;
	};`, output)
}

// Test that the channel statement is formatted correctly when syslog
// clause without facility is specified.
func TestChannelFormatSyslogWithoutFacility(t *testing.T) {
	channel := &Channel{
		Name: String{
			Quoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				Syslog: &Syslog{},
			},
		},
	}
	output := channel.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `channel "docker" {
		syslog;
	};`, output)
}

// Test that the channel statement is formatted correctly when null
// logging target is specified.
func TestChannelFormatNull(t *testing.T) {
	channel := &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				Null: true,
			},
		},
	}
	output := channel.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `channel docker {
		null;
	};`, output)
}

// Test that the channel statement is formatted correctly when stderr
// logging target is specified.
func TestChannelFormatStderr(t *testing.T) {
	channel := &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				Stderr: true,
			},
		},
	}
	output := channel.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `channel docker {
		stderr;
	};`, output)
}

// Test that file name is returned for a channel type that is a file.
func TestChannelGetFileName(t *testing.T) {
	channel := &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				File: &File{
					Name: &String{
						Unquoted: storkutil.Ptr("/var/lib/log/default"),
					},
				},
			},
		},
	}
	require.Equal(t, "/var/lib/log/default", channel.GetFileName())
}

// Test that empty string is returned for a channel type that is not a file.
func TestChannelGetFileNameNotFile(t *testing.T) {
	channel := &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				Syslog: &Syslog{
					Facility: &String{
						Unquoted: storkutil.Ptr("local7"),
					},
				},
			},
		},
	}
	require.Empty(t, channel.GetFileName())
}

// Test that true is returned for a channel type that is a file.
func TestChannelIsFile(t *testing.T) {
	channel := &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				File: &File{
					Name: &String{
						Unquoted: storkutil.Ptr("/var/lib/log/default"),
					},
				},
			},
		},
	}
	require.True(t, channel.IsFile())
	require.False(t, channel.IsSyslog())
	require.False(t, channel.IsNull())
	require.False(t, channel.IsStderr())
}

// Test that true is returned for a channel type that is a syslog.
func TestChannelIsSyslog(t *testing.T) {
	channel := &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				Syslog: &Syslog{},
			},
		},
	}
	require.True(t, channel.IsSyslog())
	require.False(t, channel.IsFile())
	require.False(t, channel.IsNull())
	require.False(t, channel.IsStderr())
}

// Test that true is returned for a channel type that is a null.
func TestChannelIsNull(t *testing.T) {
	channel := &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				Null: true,
			},
		},
	}
	require.True(t, channel.IsNull())
	require.False(t, channel.IsFile())
	require.False(t, channel.IsSyslog())
	require.False(t, channel.IsStderr())
}

// Test that true is returned for a channel type that is a stderr.
func TestChannelIsStderr(t *testing.T) {
	channel := &Channel{
		Name: String{
			Unquoted: storkutil.Ptr("docker"),
		},
		Clauses: []*ChannelClause{
			{
				Stderr: true,
			},
		},
	}
	require.True(t, channel.IsStderr())
	require.False(t, channel.IsFile())
	require.False(t, channel.IsSyslog())
	require.False(t, channel.IsNull())
}
