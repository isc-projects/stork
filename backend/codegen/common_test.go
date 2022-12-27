package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test parsing valid <key>:<value> mappings.
func TestParseMappings(t *testing.T) {
	mappings := []string{"foo:bar", "baz:xyz"}
	output := make(map[string]string)
	err := parseMappings(mappings, output)
	require.NoError(t, err)
	require.Contains(t, output, "foo")
	require.Contains(t, output, "baz")
	require.Equal(t, "bar", output["foo"])
	require.Equal(t, "xyz", output["baz"])
}

// Test parsing empty array of mappings.
func TestParseMappingsEmpty(t *testing.T) {
	mappings := []string{}
	output := make(map[string]string)
	err := parseMappings(mappings, output)
	require.NoError(t, err)
	require.Empty(t, output)
}

// Test parsing invalid mappings.
func TestParseMappingsInvalid(t *testing.T) {
	output := make(map[string]string)
	t.Run("empty string", func(t *testing.T) {
		mappings := []string{""}
		err := parseMappings(mappings, output)
		require.Error(t, err)
	})
	t.Run("key only", func(t *testing.T) {
		mappings := []string{"foo"}
		err := parseMappings(mappings, output)
		require.Error(t, err)
	})
	t.Run("too many tokens", func(t *testing.T) {
		mappings := []string{"foo:bar:baz"}
		err := parseMappings(mappings, output)
		require.Error(t, err)
	})
}
