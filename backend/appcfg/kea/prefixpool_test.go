package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
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

// Test retrieving the pool parameters.
func TestPrefixPoolGetParameters(t *testing.T) {
	pool := PDPool{
		PoolID: storkutil.Ptr(int64(2345)),
		ClientClassParameters: ClientClassParameters{
			ClientClass:          storkutil.Ptr("foo"),
			RequireClientClasses: []string{"foo", "bar"},
		},
	}
	params := pool.GetPoolParameters()
	require.NotNil(t, params)
	require.NotNil(t, params.ClientClass)
	require.Equal(t, "foo", *params.ClientClass)
	require.Len(t, params.RequireClientClasses, 2)
	require.Equal(t, "foo", params.RequireClientClasses[0])
	require.Equal(t, "bar", params.RequireClientClasses[1])
	require.NotNil(t, params.PoolID)
	require.EqualValues(t, 2345, *params.PoolID)
}

// Test that an empty set of parameters can be retrieved.
func TestPrefixPoolGetNoParameters(t *testing.T) {
	pool := PDPool{}
	params := pool.GetPoolParameters()
	require.NotNil(t, params)
	require.Nil(t, params.ClientClass)
	require.Empty(t, params.RequireClientClasses)
	require.Nil(t, params.PoolID)
}
