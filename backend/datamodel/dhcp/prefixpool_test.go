package dhcpmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that a prefix can be retrieved from the pool.
func TestGetPrefix(t *testing.T) {
	pool := PrefixPool{
		Prefix:       "2001:db8:1::/64",
		DelegatedLen: 80,
	}
	prefix, prefixLen, err := pool.GetPrefix()
	require.NoError(t, err)
	require.Equal(t, prefix, "2001:db8:1::")
	require.EqualValues(t, 64, prefixLen)
}

// Test that an error is returned upon retrieving an invalid prefix
// from a pool.
func TestGetInvalidPrefix(t *testing.T) {
	pool := PrefixPool{
		Prefix:       "/64",
		DelegatedLen: 80,
	}
	_, _, err := pool.GetPrefix()
	require.Error(t, err)
}

// Test that a valid excluded prefix is parsed and returned.
func TestGetExcludedPrefix(t *testing.T) {
	pool := PrefixPool{
		ExcludedPrefix: "2001:db8:1:2::/80",
	}
	prefix, length, err := pool.GetExcludedPrefix()
	require.NoError(t, err)
	require.Equal(t, "2001:db8:1:2::", prefix)
	require.EqualValues(t, 80, length)
}

// Test that an empty excluded prefix is returned when the parsed
// prefix is empty.
func TestGetEmptyExcludedPrefix(t *testing.T) {
	pool := PrefixPool{
		ExcludedPrefix: "",
	}
	prefix, length, err := pool.GetExcludedPrefix()
	require.NoError(t, err)
	require.Empty(t, prefix)
	require.Zero(t, length)
}

// Test that an error is returned as a result of parsing an invalid
// excluded prefix.
func TestGetExcludedPrefixError(t *testing.T) {
	pool := PrefixPool{
		ExcludedPrefix: "3000::",
	}
	_, _, err := pool.GetExcludedPrefix()
	require.Error(t, err)
}
