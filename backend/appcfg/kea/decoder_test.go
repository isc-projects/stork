package keaconfig

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

// Test that the decoder's matcher function removes hyphens
// and uses case-insensitive matching.
func TestDecode(t *testing.T) {
	input := map[string]interface{}{
		"abc-def": "foo",
		"abc":     "baz",
	}
	output := struct {
		ABCDef string
		Abc    string
	}{}
	err := decode(input, &output)
	require.NoError(t, err)
	require.Equal(t, "foo", output.ABCDef)
	require.Equal(t, "baz", output.Abc)
}
