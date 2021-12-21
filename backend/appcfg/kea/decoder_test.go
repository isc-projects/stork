package keaconfig

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

// Test that the decoder's matcher function removes hyphens,
// uses case-insensitive matching and the decoder respects
// embedded structures.
func TestDecode(t *testing.T) {
	input := map[string]interface{}{
		"abc-def":      "foo",
		"abc":          "baz",
		"nested-value": "squash",
	}
	type NestedStruct struct {
		NestedValue string
	}
	output := struct {
		ABCDef string
		Abc    string
		NestedStruct
	}{}
	err := decode(input, &output)
	require.NoError(t, err)
	require.Equal(t, "foo", output.ABCDef)
	require.Equal(t, "baz", output.Abc)
	require.Equal(t, "squash", output.NestedStruct.NestedValue)
}
