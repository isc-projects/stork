package storkutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test parsing a full FQDN.
func TestParseFullFqdn(t *testing.T) {
	fqdn, err := ParseFqdn("foo.example.org.")
	require.NoError(t, err)
	require.NotNil(t, fqdn)
	require.False(t, fqdn.IsPartial())
	fqdnBytes, err := fqdn.ToBytes()
	require.NoError(t, err)
	require.Equal(t, []byte{0x3, 0x66, 0x6f, 0x6f, 0x7, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x3, 0x6f, 0x72, 0x67, 0x0}, fqdnBytes)
}

// Testing parsing a partial FQDN.
func TestParsePartialFqdn(t *testing.T) {
	fqdn, err := ParseFqdn("foo.exa-mple")
	require.NoError(t, err)
	require.NotNil(t, fqdn)
	require.True(t, fqdn.IsPartial())
	fqdnBytes, err := fqdn.ToBytes()
	require.NoError(t, err)
	require.Equal(t, []byte{0x3, 0x66, 0x6f, 0x6f, 0x8, 0x65, 0x78, 0x61, 0x2d, 0x6d, 0x70, 0x6c, 0x65}, fqdnBytes)
}

// Test that parsing an invalid FQDN yields an error.
func TestParseInvalidFqdn(t *testing.T) {
	invalidFqdns := []string{
		"",
		"foo..example.org",
		"foo.",
		"foo-.example.org.",
		"foo.example.or-g.",
		"-foo.example.org.",
		"foo.exa&ple.org.",
		"foo. example.org.",
		"foo.example.or1.",
		"foo.example.o.",
	}
	for _, fqdn := range invalidFqdns {
		parsed, err := ParseFqdn(fqdn)
		require.Error(t, err, "FQDN parsing did not fail for %s", fqdn)
		require.Nil(t, parsed)
	}
}
