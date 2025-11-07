package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that an include statement is formatted correctly.
func TestIncludeFormat(t *testing.T) {
	include := &Include{
		Path: "test.conf",
	}
	output := include.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `include "test.conf";`, output)
}

// Test that serializing an include statement with nil values does not panic.
func TestIncludeFormatNilValues(t *testing.T) {
	include := &Include{}
	require.NotPanics(t, func() { include.getFormattedOutput(nil) })
}
