package protocoltype

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the secure protocol types are indicated properly.
func TestProtocolTypeIsSecure(t *testing.T) {
	require.True(t, HTTPS.IsSecure())
	require.True(t, RNDC.IsSecure())

	require.False(t, HTTP.IsSecure())
	require.False(t, Socket.IsSecure())
}

// Test that parsing protocol types from strings works properly.
func TestParseProtocolType(t *testing.T) {
	t.Run("HTTP", func(t *testing.T) {
		pt, ok := Parse("http")
		require.True(t, ok)
		require.Equal(t, HTTP, pt)
	})

	t.Run("HTTPS", func(t *testing.T) {
		pt, ok := Parse("https")
		require.True(t, ok)
		require.Equal(t, HTTPS, pt)
	})

	t.Run("Socket", func(t *testing.T) {
		pt, ok := Parse("unix")
		require.True(t, ok)
		require.Equal(t, Socket, pt)
	})

	t.Run("RNDC", func(t *testing.T) {
		pt, ok := Parse("rndc")
		require.True(t, ok)
		require.Equal(t, RNDC, pt)
	})

	t.Run("Unknown protocol type", func(t *testing.T) {
		pt, ok := Parse("unknown")
		require.False(t, ok)
		require.Equal(t, Unspecified, pt)
	})

	t.Run("HTTP uppercase", func(t *testing.T) {
		pt, ok := Parse("HTTP")
		require.False(t, ok)
		require.Equal(t, Unspecified, pt)
	})
}
