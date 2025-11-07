package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the key statement is formatted correctly.
func TestKeyFormat(t *testing.T) {
	key := &Key{
		Name: "test-key",
		Clauses: []*KeyClause{
			{
				Algorithm: "hmac-sha256",
			},
			{
				Secret: "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=",
			},
		},
	}
	output := key.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `key "test-key" { algorithm "hmac-sha256"; secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU="; };`, output)
}

// Test that serializing a key statement with nil values does not panic.
func TestKeyFormatNilValues(t *testing.T) {
	key := &Key{}
	require.NotPanics(t, func() { key.getFormattedOutput(nil) })
}
