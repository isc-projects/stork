package keaconfig

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that configuration of the selected hooks library can be retrieved.
func TestGetHookLibrary(t *testing.T) {
	hooks := HookLibraries{
		{
			Library: "libdhcp_lease_cmds",
		},
		{
			Library:    "libdhcp_host_cmds",
			Parameters: (json.RawMessage)(`{"name": "foo"}`),
		},
	}

	path, params, ok := hooks.GetHookLibrary("libdhcp_host_cmds")
	require.True(t, ok)
	require.Equal(t, "libdhcp_host_cmds", path)
	require.Contains(t, params, "name")

	path, params, ok = hooks.GetHookLibrary("libdhcp_lease_cmds")
	require.True(t, ok)
	require.Equal(t, "libdhcp_lease_cmds", path)
	require.Nil(t, params)
}

// Test the case when Kea configuration contains empty hooks list and
// one of the hooks is requested by name.
func TestGetHookLibraryEmptyHooks(t *testing.T) {
	cfg := getTestConfigEmptyHooks(t)

	path, params, ok := cfg.GetHookLibrary("libdhcp_ha")
	require.False(t, ok)
	require.Empty(t, path)
	require.Empty(t, params)
}

// Test that the configuration of the HA hooks library can be retrieved
// and parsed.
func TestGetHAHookLibrary(t *testing.T) {
	hooks := HookLibraries{
		{
			Library: "/usr/lib/kea/libdhcp_ha.so",
			Parameters: (json.RawMessage)(`{
				"high-availability": [
					{
						"this-server-name": "server1",
						"mode": "hot-standby",
						"heartbeat-delay": 10000
					}
				]
			}`),
		},
	}

	path, params, ok := hooks.GetHAHookLibrary()
	require.True(t, ok)
	require.Len(t, params.HA, 1)

	relationships := params.GetAllRelationships()
	require.Len(t, relationships, 1)

	require.NotNil(t, relationships[0].ThisServerName)
	require.NotNil(t, relationships[0].Mode)
	require.NotNil(t, relationships[0].HeartbeatDelay)

	require.Equal(t, "/usr/lib/kea/libdhcp_ha.so", path)
	require.Equal(t, "server1", *relationships[0].ThisServerName)
	require.Equal(t, "hot-standby", *relationships[0].Mode)
	require.Equal(t, 10000, *relationships[0].HeartbeatDelay)
}

func TestGetLeaseCmdsHookLibrary(t *testing.T) {
	hooks := HookLibraries{
		{
			Library: "libdhcp_lease_cmds",
		},
	}

	path, _, ok := hooks.GetLeaseCmdsHookLibrary()
	require.True(t, ok)
	require.Equal(t, "libdhcp_lease_cmds", path)
}

func TestGetLegalLogHookLibrary(t *testing.T) {
	hooks := HookLibraries{
		{
			Library: "libdhcp_legal_log",
			Parameters: (json.RawMessage)(`{
				"name": "kea",
				"host": "localhost",
				"path": "/tmp/path"
			}`),
		},
	}

	path, params, ok := hooks.GetLegalLogHookLibrary()
	require.True(t, ok)
	require.Equal(t, "libdhcp_legal_log", path)
	require.NotNil(t, params)
	require.Equal(t, "localhost", params.Host)
	require.Equal(t, "kea", params.Name)
	require.Equal(t, "/tmp/path", params.Path)
}
