package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//go:generate mockgen -package=keaconfig_test -destination=prefixpoolmock_test.go isc.org/stork/appcfg/kea PrefixPool

// Test getting a delegated prefix pool prefix.
func TestGetCanonicalPrefix(t *testing.T) {
	pool := PDPool{
		Prefix:       "3001:1::",
		PrefixLen:    80,
		DelegatedLen: 96,
	}
	require.Equal(t, "3001:1::/80", pool.GetCanonicalPrefix())
}

// Test that an empty string is returned when the delegated prefix
// pool prefix is not specified.
func TestGetCanonicalPrefixEmpty(t *testing.T) {
	pool := PDPool{}
	require.Empty(t, pool.GetCanonicalPrefix())
}

// Test getting an excluded prefix.
func TestGetCanonicalExcludedPrefix(t *testing.T) {
	pool := PDPool{
		ExcludedPrefix:    "3001:2::",
		ExcludedPrefixLen: 112,
	}
	require.Equal(t, "3001:2::/112", pool.GetCanonicalExcludedPrefix())
}

// Test that an empty string is returned when the excluded prefix
// is not specified.
func TestGetCanonicalExcludedPrefixEmpty(t *testing.T) {
	pool := PDPool{}
	require.Empty(t, pool.GetCanonicalExcludedPrefix())
}
