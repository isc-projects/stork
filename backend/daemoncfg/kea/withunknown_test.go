package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the WithUnknown type can be used with struct types.
func TestWithUnknownStruct(t *testing.T) {
	// Define a struct for known parameters but with some parameters
	// lacking name in the json tag.
	type Known struct {
		Foo string `json:"foo"`
		Bar string `json:"bar,omitempty"`
		Baz string `json:",omitempty"`
		Qux string `json:"-"`
	}
	s := WithUnknown[Known]{}

	// Unmarshal the JSON input into the WithUnknown structure.
	jsonInput := `
	{
		"foo": "bar",
		"baz": "qux",
		"bar": "quux",
		"qux": "corge"
	}`
	err := s.UnmarshalJSON([]byte(jsonInput))
	require.NoError(t, err)

	// Verify that known parameters are correctly parsed.
	require.Equal(t, "bar", s.Known.Foo)
	require.Equal(t, "qux", s.Known.Baz)
	require.Equal(t, "quux", s.Known.Bar)
	// This parameter should be omitted because it is excluded from parsing
	// using the json tag "-".
	require.Empty(t, s.Known.Qux)
	// Parameters lacking name in the json tag should be picked by the Unknown map.
	require.Contains(t, s.Unknown, "baz")
	require.Contains(t, s.Unknown, "qux")

	// Verify that the WithUnknown structure can be marshalled back to JSON.
	// It contains extraneous Baz parameter because it lacks the name in the json tag.
	// Therefore, its name is generated from the field name.
	marshalled, err := s.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, `
		{
		"foo": "bar",
		"Baz": "qux",
		"baz": "qux",
		"bar": "quux",
		"qux": "corge"
	}`, string(marshalled))
}

// Test that the WithUnknown type can be used with embedded struct types.
func TestWithUnknownEmbeddedStruct(t *testing.T) {
	// Create a structure with the embedded structure.
	type Embedded struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}
	type Known struct {
		Embedded
	}
	s := WithUnknown[Known]{}

	// Unmarshal the JSON input into the WithUnknown structure.
	jsonInput := `
	{
		"foo": "bar",
		"baz": "qux",
		"bar": "quux",
		"qux": "corge"
	}`
	err := s.UnmarshalJSON([]byte(jsonInput))
	require.NoError(t, err)

	// The parameters should be parsed into the embedded structure.
	require.Equal(t, "bar", s.Known.Foo)
	require.Equal(t, "quux", s.Known.Bar)

	// Parameters lacking in the embedded structure should be picked by the Unknown map.
	require.Contains(t, s.Unknown, "baz")
	require.Contains(t, s.Unknown, "qux")

	// Verify that the WithUnknown structure can be marshalled back to JSON.
	marshalled, err := s.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, jsonInput, string(marshalled))
}

// Test that the WithUnknown type can be used with non-struct types.
func TestWithUnknownNonStruct(t *testing.T) {
	type Known int
	s := WithUnknown[Known]{}
	err := s.UnmarshalJSON([]byte(`123`))
	require.NoError(t, err)
	require.EqualValues(t, 123, s.Known)
	require.Empty(t, s.Unknown)

	marshalled, err := s.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, `123`, string(marshalled))
}
