package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the keys clause is formatted correctly.
func TestKeysFormat(t *testing.T) {
	keys := &Keys{
		KeyNames: []string{"trusted", "guest"},
	}
	output := keys.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `keys { "trusted"; "guest"; };`, output)
}

// Test that serializing a keys clause with nil values does not panic.
func TestKeysFormatNilValues(t *testing.T) {
	keys := &Keys{}
	require.NotPanics(t, func() { keys.getFormattedOutput(nil) })
}
