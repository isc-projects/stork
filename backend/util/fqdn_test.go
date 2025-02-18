package storkutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test names comparison.
func TestCompareNames(t *testing.T) {
	require.Negative(t, CompareNames("authors.bind", "version.bind"))
	require.Positive(t, CompareNames("version.bind", "authors.bind"))
	require.Zero(t, CompareNames("version.bind", "version.bind"))
	require.Negative(t, CompareNames("example.com", "www.example.com"))
	require.Positive(t, CompareNames("host.example.com", "example.com"))
	require.Negative(t, CompareNames("host.example.com", "example.org"))
	require.Negative(t, CompareNames("com", "org"))
	require.Positive(t, CompareNames("com", ""))
	require.Zero(t, CompareNames("", ""))
}

// Tests converting names to a the names with labels ordered backwards.
func TestConvertNameToRname(t *testing.T) {
	require.Equal(t, "org.example.zone", ConvertNameToRname("zone.example.org"))
	require.Equal(t, "org.example.www", ConvertNameToRname("www.example.org."))
	require.Empty(t, ConvertNameToRname(""))
}

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
