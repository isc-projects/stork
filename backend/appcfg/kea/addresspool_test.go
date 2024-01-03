package keaconfig

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=keaconfig_test -destination=addresspoolmock_test.go isc.org/stork/appcfg/kea AddressPool

// Test parsing a pool address range, ensuring that the whitespace
// is removed between the lower bound and the upper bound.
func TestParseAddressPoolRange(t *testing.T) {
	input := `{
		"pool": "192.0.2.1 -   192.0.2.10"	
	}`
	var pool Pool
	err := json.Unmarshal([]byte(input), &pool)
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1-192.0.2.10", pool.Pool)
}

// Test that a pool specified using the prefix notation is converted into
// the correct address range.
func TestParseAddressPoolPrefix(t *testing.T) {
	input := `{
		"pool": "192.1.0.0/16"
	}`
	var pool Pool
	err := json.Unmarshal([]byte(input), &pool)
	require.NoError(t, err)
	require.Equal(t, "192.1.0.0-192.1.255.255", pool.Pool)
}

// Test parsing an IPv6 address range, ensuring that the whitespace
// is removed between the lower bound and the upper bound.
func TestParseAddressPoolIPv6Range(t *testing.T) {
	input := `{
		"pool": "2001:db8:1:: - 2001:db8:1::ffff"
	}`
	var pool Pool
	err := json.Unmarshal([]byte(input), &pool)
	require.NoError(t, err)
	require.NoError(t, err)
	require.Equal(t, "2001:db8:1::-2001:db8:1::ffff", pool.Pool)
}

// Test that an IPv6 pool specified using the prefix notation is converted into
// the correct address range.
func TestParseAddressPoolIPv6Prefix(t *testing.T) {
	input := `{
		"pool": "3000::/64"
	}`
	var pool Pool
	err := json.Unmarshal([]byte(input), &pool)
	require.NoError(t, err)
	require.NoError(t, err)
	require.Equal(t, "3000::-3000::ffff:ffff:ffff:ffff", pool.Pool)
}

// Test that an error is returned when the parsed pool is invalid.
func TestParseAddressPoolInvalidRange(t *testing.T) {
	input := `{
		"pool": "192.1.0.1"
	}`
	var pool Pool
	err := json.Unmarshal([]byte(input), &pool)
	require.Error(t, err)
}

// Test that an error is returned when the parsed pool is empty.
func TestParseAddressPoolEmptyRange(t *testing.T) {
	input := `{
		"pool": ""
	}`
	var pool Pool
	err := json.Unmarshal([]byte(input), &pool)
	require.Error(t, err)
}

// Test retrieving the address pool boundaries.
func TestAddressPoolGetBoundaries(t *testing.T) {
	pool := Pool{
		Pool: "192.0.2.1-192.0.2.254",
	}
	lb, ub, err := pool.GetBoundaries()
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", lb.String())
	require.Equal(t, "192.0.2.254", ub.String())
}

// Test that an error is returned upon an attempt to retrieve the address
// bool boundaries when the lower bound is invalid.
func TestAddressPoolGetBoundariesLowerBoundError(t *testing.T) {
	pool := Pool{
		Pool: "192.0.2.X-192.0.2.254",
	}
	_, _, err := pool.GetBoundaries()
	require.Error(t, err)
}

// Test that an error is returned upon an attempt to retrieve the address
// bool boundaries when the upper bound is invalid.
func TestAddressPoolGetBoundariesUpperBoundError(t *testing.T) {
	pool := Pool{
		Pool: "192.0.2.1-192.0.2.",
	}
	_, _, err := pool.GetBoundaries()
	require.Error(t, err)
}

// Test retrieving the pool parameters.
func TestAddressPoolGetParameters(t *testing.T) {
	pool := Pool{
		Pool:   "192.0.2.1-192.0.2.254",
		PoolID: storkutil.Ptr(int64(1234)),
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
	require.EqualValues(t, 1234, *params.PoolID)
}

// Test that an empty set of parameters can be retrieved.
func TestAddressPoolGetNoParameters(t *testing.T) {
	pool := Pool{
		Pool: "192.0.2.1-192.0.2.254",
	}
	params := pool.GetPoolParameters()
	require.NotNil(t, params)
	require.Nil(t, params.ClientClass)
	require.Empty(t, params.RequireClientClasses)
	require.Nil(t, params.PoolID)
}
