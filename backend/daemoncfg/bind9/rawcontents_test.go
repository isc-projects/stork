package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that capturing the raw contents returns an empty string when there are no values.
func TestRawContentsCaptureEmpty(t *testing.T) {
	var rawContents RawContents
	rawContents.Capture([]string{})
	require.Equal(t, "", rawContents.GetString())
}

// Test that capturing the raw contents removes the trailing @stork:no-parse: suffix.
func TestRawContentsCaptureTrimEmpty(t *testing.T) {
	var rawContents RawContents
	rawContents.Capture([]string{"game", "over", "//@stork:no-parse:"})
	require.Equal(t, "game over", rawContents.GetString())
}

// Test that capturing the raw contents removes the trailing @stork:no-parse: suffix
// and trims the trailing whitespace.
func TestRawContentsCaptureTrimTrailingWhitespace(t *testing.T) {
	var rawContents RawContents
	rawContents.Capture([]string{"game", "over", "token \t\r\n//@stork:no-parse:"})
	require.Equal(t, "game over token", rawContents.GetString())
}

// Test that captured raw contents are correct when there is no suffix.
func TestRawContentsCaptureNoSuffix(t *testing.T) {
	var rawContents RawContents
	rawContents.Capture([]string{"over", "game"})
	require.Equal(t, "over game", rawContents.GetString())
}

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
