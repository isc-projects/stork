package keaconfig

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=keaconfig_test -destination=prefixpoolmock_test.go isc.org/stork/daemoncfg/kea PrefixPool

// Test that unknown parameters are parsed correctly.
func TestParsePrefixPoolWithUnknownParameters(t *testing.T) {
	input := `{
		"prefix": "3001:1::",
		"prefix-len": 80,
		"delegated-len": 96,
		"foo": "bar",
		"baz": "qux"
	}`
	var pool PDPool
	err := json.Unmarshal([]byte(input), &pool)
	require.NoError(t, err)
	require.Equal(t, "3001:1::", pool.Prefix)
	require.EqualValues(t, 80, pool.PrefixLen)
	require.EqualValues(t, 96, pool.DelegatedLen)
	require.NotNil(t, pool.UnknownParameters)
	require.Len(t, pool.UnknownParameters, 2)
	require.Equal(t, "bar", pool.UnknownParameters["foo"])
	require.Equal(t, "qux", pool.UnknownParameters["baz"])
}

// Test that unknown parameters are marshalled correctly.
func TestMarshalPrefixPoolWithUnknownParameters(t *testing.T) {
	pool := PDPool{
		PDPoolKnownParameters: PDPoolKnownParameters{
			Prefix:       "3001:1::",
			PrefixLen:    80,
			DelegatedLen: 96,
		},
		UnknownParameters: map[string]any{
			"foo": "bar",
			"baz": "qux",
		},
	}
	marshalled, err := json.Marshal(pool)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"prefix": "3001:1::",
		"prefix-len": 80,
		"delegated-len": 96,
		"foo": "bar",
		"baz": "qux"
	}`, string(marshalled))
}

// Test getting a delegated prefix pool prefix.
func TestGetCanonicalPrefix(t *testing.T) {
	pool := PDPool{
		PDPoolKnownParameters: PDPoolKnownParameters{
			Prefix:       "3001:1::",
			PrefixLen:    80,
			DelegatedLen: 96,
		},
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
		PDPoolKnownParameters: PDPoolKnownParameters{
			ExcludedPrefix:    "3001:2::",
			ExcludedPrefixLen: 112,
		},
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
		PDPoolKnownParameters: PDPoolKnownParameters{
			PoolID: 2345,
			ClientClassParameters: ClientClassParameters{
				ClientClass:               storkutil.Ptr("foo"),
				ClientClasses:             []string{"baz"},
				RequireClientClasses:      []string{"foo", "bar"},
				EvaluateAdditionalClasses: []string{"baz"},
			},
		},
		UnknownParameters: map[string]any{
			"foo": "bar",
		},
	}
	params := pool.GetPoolParameters()
	require.NotNil(t, params)
	require.NotNil(t, params.ClientClass)
	require.Equal(t, "foo", *params.ClientClass)
	require.Len(t, params.ClientClasses, 1)
	require.Equal(t, "baz", params.ClientClasses[0])
	require.Len(t, params.RequireClientClasses, 2)
	require.Equal(t, "foo", params.RequireClientClasses[0])
	require.Equal(t, "bar", params.RequireClientClasses[1])
	require.Len(t, params.EvaluateAdditionalClasses, 1)
	require.Equal(t, "baz", params.EvaluateAdditionalClasses[0])
	require.EqualValues(t, 2345, params.PoolID)
	require.NotNil(t, params.UnknownParameters)
	require.Len(t, params.UnknownParameters, 1)
	require.Equal(t, "bar", params.UnknownParameters["foo"])
}

// Test that an empty set of parameters can be retrieved.
func TestPrefixPoolGetNoParameters(t *testing.T) {
	pool := PDPool{}
	params := pool.GetPoolParameters()
	require.NotNil(t, params)
	require.Nil(t, params.ClientClass)
	require.Empty(t, params.ClientClasses)
	require.Empty(t, params.RequireClientClasses)
	require.Empty(t, params.EvaluateAdditionalClasses)
	require.Zero(t, params.PoolID)
	require.Nil(t, params.UnknownParameters)
}
