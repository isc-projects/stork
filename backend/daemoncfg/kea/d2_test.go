package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
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

// Test that the control sockets are correctly retrieved for a D2 server.
// The array member takes precedence over the single member.
func TestGetD2ControlSockets(t *testing.T) {
	// Arrange
	cfg := &D2Config{
		ControlSockets: []ControlSocket{
			{
				SocketType: "unix",
				SocketName: storkutil.Ptr("/var/run/kea/kea-d2.sock"),
			},
			{
				SocketType:    "http",
				SocketAddress: storkutil.Ptr("localhost"),
				SocketPort:    storkutil.Ptr(int64(1111)),
			},
		},
		ControlSocket: &ControlSocket{
			SocketType: "unix",
			SocketName: storkutil.Ptr("/var/run/kea/kea-d2-legacy.sock"),
		},
	}

	t.Run("ControlSockets defined", func(t *testing.T) {
		// Act
		sockets := cfg.GetListeningControlSockets()

		// Assert
		require.Len(t, sockets, 2)

		require.Equal(t, "unix", sockets[0].SocketType)
		require.Equal(t, "/var/run/kea/kea-d2.sock", *sockets[0].SocketName)

		require.Equal(t, "http", sockets[1].SocketType)
		require.Equal(t, "localhost", *sockets[1].SocketAddress)
		require.EqualValues(t, 1111, *sockets[1].SocketPort)
	})

	cfg.ControlSockets = nil

	t.Run("Only ControlSocket defined", func(t *testing.T) {
		// Act
		sockets := cfg.GetListeningControlSockets()

		// Assert
		require.Len(t, sockets, 1)

		require.Equal(t, "unix", sockets[0].SocketType)
		require.Equal(t, "/var/run/kea/kea-d2-legacy.sock", *sockets[0].SocketName)
	})

	cfg.ControlSocket = nil

	t.Run("Neither ControlSockets nor ControlSocket defined", func(t *testing.T) {
		// Act
		sockets := cfg.GetListeningControlSockets()

		// Assert
		require.Nil(t, sockets)
	})
}
