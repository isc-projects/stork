package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests that the configuration root key can be found.
func TestGetRootName(t *testing.T) {
	cfg, err := NewKeaConfigFromJSON(`
        {
            "Logging": { },
            "Dhcp4": {
                "subnet4": [ ]
            }
        }
    `)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	root, ok := cfg.GetRootName()
	require.True(t, ok)
	require.Equal(t, "Dhcp4", root)
}

// Tests that Logging key is ignored as non-root key.
func TestGetRootNameNoRoot(t *testing.T) {
	cfg, err := NewKeaConfigFromJSON(`
        {
            "Logging": { }
        }
    `)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	root, ok := cfg.GetRootName()
	require.False(t, ok)
	require.Empty(t, root)
}
