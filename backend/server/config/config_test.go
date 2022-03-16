package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test creating new config update instance.
func TestNewUpdate(t *testing.T) {
	cu := NewUpdate("kea", "host_add", 1, 2, 3)
	require.NotNil(t, cu)
	require.Equal(t, "kea", cu.Target)
	require.Equal(t, "host_add", cu.Operation)
	require.Len(t, cu.DaemonIDs, 3)
	require.Contains(t, cu.DaemonIDs, int64(1))
	require.Contains(t, cu.DaemonIDs, int64(2))
	require.Contains(t, cu.DaemonIDs, int64(3))
}

// Test that the DecodeContextData decodes a map into structure.
func TestDecodeContextData(t *testing.T) {
	input := map[string]interface{}{
		"foo": "bar",
	}
	output := struct {
		Foo string
	}{}
	err := DecodeContextData(input, &output)
	require.NoError(t, err)
	require.Equal(t, "bar", output.Foo)
}
