package storkutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test converting an interface to int64.
func TestConvertJSONInt64(t *testing.T) {
	// int64 value should be returned without conversion.
	v, err := ConvertJSONInt64(int64(5))
	require.NoError(t, err)
	require.Equal(t, int64(5), v)

	// json.Number should convert ok.
	v, err = ConvertJSONInt64(json.Number("10"))
	require.NoError(t, err)
	require.Equal(t, int64(10), v)

	// Other values should result in an error.
	v, err = ConvertJSONInt64("10")
	require.Error(t, err)
	require.Zero(t, v)
}

// Test extracting an interface from map to int64.
func TestExtractJSONInt64(t *testing.T) {
	m := make(map[string]interface{})

	// int64 value should be returned as is.
	m["foo"] = int64(6)
	v, err := ExtractJSONInt64(m, "foo")
	require.NoError(t, err)
	require.Equal(t, int64(6), v)

	// json.Number should be converted to int64.
	m["foo"] = json.Number("11")
	v, err = ExtractJSONInt64(m, "foo")
	require.NoError(t, err)
	require.Equal(t, int64(11), v)

	// Non-existing value.
	v, err = ExtractJSONInt64(m, "bar")
	require.Error(t, err)
	require.Zero(t, v)

	// Wrong type.
	m["foo"] = true
	v, err = ExtractJSONInt64(m, "foo")
	require.Error(t, err)
	require.Zero(t, v)
}
