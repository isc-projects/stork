package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test getting hook libraries for a D2 server.
func TestGetD2HookLibraries(t *testing.T) {
	cfg := &D2Config{
		HookLibraries: []HookLibrary{
			{
				Library: "libdhcp_lease_cmds",
			},
		},
	}

	hooks := cfg.GetHookLibraries()
	require.Len(t, hooks, 1)
	require.Equal(t, "libdhcp_lease_cmds", hooks[0].Library)
}

// Test getting loggers configurations for a D2 server.
func TestGetD2Loggers(t *testing.T) {
	cfg := &D2Config{
		Loggers: []Logger{
			{
				Name:       "kea-dhcp-ddns",
				Severity:   "DEBUG",
				DebugLevel: 99,
			},
		},
	}

	libraries := cfg.GetLoggers()
	require.Len(t, libraries, 1)

	require.Equal(t, "kea-dhcp-ddns", libraries[0].Name)
	require.Equal(t, "DEBUG", libraries[0].Severity)
	require.EqualValues(t, 99, libraries[0].DebugLevel)
}
