package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test getting raw contents as a string.
func TestRawContentsGetString(t *testing.T) {
	rawContents := RawContents("test")
	require.Equal(t, "test", rawContents.GetString())
}

// Test getting raw contents as a string when it is nil.
func TestRawContentsNilGetString(t *testing.T) {
	var rawContents *RawContents
	require.Equal(t, "", rawContents.GetString())
}
