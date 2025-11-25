package bind9config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Compares the formatted output with the expected textual representation.
// The whitespace is ignored in this comparison.
func requireConfigEq(t *testing.T, expected string, formattedOutput formatterOutput) {
	expected = strings.ReplaceAll(expected, " ;", ";")
	expectedTokens := strings.Fields(expected)
	builder := newFormatterStringBuilder()
	formattedOutput.write(0, false, builder)
	actual := builder.getString()
	actualTokens := strings.Fields(actual)
	require.Equal(t, len(expectedTokens), len(actualTokens), `different number of tokens in expected and actual:

	%s

	vs

	%s`, expected, actual)
	for i, expectedToken := range expectedTokens {
		require.Equal(t, expectedToken, actualTokens[i])
	}
}

// Compares the formatted scope output with the expected textual representation
// of a scope. The whitespace is ignored in this comparison.
func cfgScopeEq(t *testing.T, expected string, formattedOutput formatterOutput) {
	requireConfigEq(t, fmt.Sprintf(`{ %s }`, expected), formattedOutput)
}
