package pdnsconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test getting a parsed boolean value.
func TestConfigGetBool(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-enabled": {
			{
				boolValue: storkutil.Ptr(true),
			},
			{
				stringValue: storkutil.Ptr("foo"),
			},
		},
	})
	require.NotNil(t, config)

	t.Run("get existing key", func(t *testing.T) {
		enabled := config.GetBool("api-enabled")
		require.NotNil(t, enabled)
		require.True(t, *enabled)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		enabled := config.GetBool("non-existing-key")
		require.Nil(t, enabled)
	})
}

// Test that nil value is returned when there is no boolean value
// for a key.
func TestConfigGetBoolEmpty(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-enabled": {},
	})
	require.NotNil(t, config)

	enabled := config.GetBool("api-enabled")
	require.Nil(t, enabled)
}

// Test getting a parsed integer value.
func TestConfigGetInt64(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-port": {
			{
				int64Value: storkutil.Ptr(int64(8080)),
			},
			{
				stringValue: storkutil.Ptr("foo"),
			},
		},
	})
	require.NotNil(t, config)

	t.Run("get existing key", func(t *testing.T) {
		port := config.GetInt64("api-port")
		require.NotNil(t, port)
		require.Equal(t, int64(8080), *port)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		port := config.GetInt64("non-existing-key")
		require.Nil(t, port)
	})
}

// Test that nil value is returned when there is no integer value
// for a key.
func TestConfigGetInt64Empty(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-port": {},
	})
	require.NotNil(t, config)

	port := config.GetInt64("api-port")
	require.Nil(t, port)
}

// Test getting a parsed string value.
func TestConfigGetString(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-key": {
			{
				stringValue: storkutil.Ptr("stork"),
			},
			{
				stringValue: storkutil.Ptr("foo"),
			},
		},
	})
	require.NotNil(t, config)

	t.Run("get existing key", func(t *testing.T) {
		apiKey := config.GetString("api-key")
		require.NotNil(t, apiKey)
		require.Equal(t, "stork", *apiKey)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		apiKey := config.GetString("non-existing-key")
		require.Nil(t, apiKey)
	})
}

// Test that nil value is returned when there is no string value
// for a key.
func TestConfigGetStringEmpty(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-key": {},
	})
	require.NotNil(t, config)

	apiKey := config.GetString("api-key")
	require.Nil(t, apiKey)
}

// Test getting all values for a key.
func TestConfigGetValues(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-key": {
			{
				stringValue: storkutil.Ptr("stork"),
			},
			{
				stringValue: storkutil.Ptr("kea"),
			},
		},
	})
	require.NotNil(t, config)

	values := config.GetValues("api-key")
	require.Len(t, values, 2)
	stork := values[0].GetString()
	require.NotNil(t, stork)
	require.Equal(t, "stork", *stork)
	kea := values[1].GetString()
	require.NotNil(t, kea)
	require.Equal(t, "kea", *kea)
}

// Test that an empty slice is returned when there are no values
// for a key.
func TestConfigGetValuesEmpty(t *testing.T) {
	config := newConfig(map[string][]ParsedValue{
		"api-key": {},
	})
	require.NotNil(t, config)

	values := config.GetValues("api-key")
	require.Len(t, values, 0)
}
