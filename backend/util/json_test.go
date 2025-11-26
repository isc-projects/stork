package storkutil

import (
	"encoding/json"
	"fmt"
	"strings"
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

// Test instantiating new Nullable value from pointer.
func TestNewNullable(t *testing.T) {
	nullable := NewNullable(Ptr("value"))
	require.NotNil(t, nullable)
	require.NotNil(t, nullable.GetValue())
	require.Equal(t, "value", *nullable.GetValue())
}

// Test instantiating new Nullable value from value.
func TestNewNullableFromValue(t *testing.T) {
	nullable := NewNullableFromValue(4)
	require.NotNil(t, nullable)
	require.NotNil(t, nullable.GetValue())
	require.Equal(t, 4, *nullable.GetValue())
}

// Test instantiating new nullable value from nil.
func TestNewNullableFromNil(t *testing.T) {
	nullable := NewNullable[string](nil)
	require.NotNil(t, nullable)
	require.Nil(t, nullable.GetValue())
}

// Test serializing a structure with nullable values.
func TestMarshalNullable(t *testing.T) {
	type S1 struct {
		Baz *Nullable[int]  `json:"baz"`
		Abc *Nullable[bool] `json:"abc"`
	}
	type S2 struct {
		Foo *Nullable[string] `json:"foo"`
		Bar *Nullable[S1]     `json:"bar"`
	}
	s := S2{
		Foo: NewNullableFromValue("first"),
		Bar: NewNullable(&S1{
			Baz: NewNullable[int](nil),
			Abc: NewNullableFromValue(true),
		}),
	}
	marshalled, err := json.Marshal(s)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"foo": "first",
		"bar": {
			"baz": null,
			"abc": true
		}
	}`, string(marshalled))
}

// Test parsing into a structure with nullable values.
func TestUnmarshalNullable(t *testing.T) {
	type S1 struct {
		Baz Nullable[int]   `json:"baz"`
		Abc *Nullable[bool] `json:"abc"`
	}
	type S2 struct {
		Foo *Nullable[string] `json:"foo"`
		Bar *Nullable[S1]     `json:"bar"`
	}
	marshalled := []byte(`{
		"foo": "first",
		"bar": {
			"baz": null,
			"abc": true
		}
	}`)
	var s S2
	err := json.Unmarshal(marshalled, &s)
	require.NoError(t, err)

	require.NotNil(t, s.Foo)
	require.NotNil(t, s.Foo.GetValue())
	require.Equal(t, "first", *s.Foo.GetValue())
	require.NotNil(t, s.Bar)
	require.NotNil(t, s.Bar.GetValue())
	require.Nil(t, s.Bar.GetValue().Baz.GetValue())
	require.NotNil(t, s.Bar.GetValue().Abc)
	require.NotNil(t, s.Bar.GetValue().Abc.GetValue())
	require.True(t, *s.Bar.GetValue().Abc.GetValue())
}

// Test instantiating new nullable array.
func TestNewNullableArray(t *testing.T) {
	nullable := NewNullableArray([]string{"value1", "value2"})
	require.NotNil(t, nullable)
	require.NotNil(t, nullable.GetValue())
	require.ElementsMatch(t, nullable.GetValue(), []string{"value1", "value2"})
}

// Test instantiating new nullable array from nil.
func TestNewNullableArrayFromNil(t *testing.T) {
	nullable := NewNullableArray[string](nil)
	require.NotNil(t, nullable)
	require.Nil(t, nullable.GetValue())
}

// Test serializing a structure with nullable arrays.
func TestMarshalNullableArray(t *testing.T) {
	type S1 struct {
		Baz *NullableArray[int]  `json:"baz"`
		Abc *NullableArray[bool] `json:"abc"`
	}
	type S2 struct {
		Foo *NullableArray[string] `json:"foo"`
		Bar *Nullable[S1]          `json:"bar"`
	}
	s := S2{
		Foo: NewNullableArray([]string{"first", "second"}),
		Bar: NewNullable(&S1{
			Baz: NewNullableArray[int](nil),
			Abc: NewNullableArray([]bool{}),
		}),
	}
	marshalled, err := json.Marshal(s)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"foo": ["first", "second"],
		"bar": {
			"baz": null,
			"abc": []
		}
	}`, string(marshalled))
}

// Test parsing into a structure with nullable arrays.
func TestUnmarshalNullableArray(t *testing.T) {
	type S1 struct {
		Baz NullableArray[int]   `json:"baz"`
		Abc *NullableArray[bool] `json:"abc"`
	}
	type S2 struct {
		Foo *NullableArray[string] `json:"foo"`
		Bar *Nullable[S1]          `json:"bar"`
	}
	marshalled := []byte(`{
		"foo": ["first", "second"],
		"bar": {
			"baz": null,
			"abc": []
		}
	}`)
	var s S2
	err := json.Unmarshal(marshalled, &s)
	require.NoError(t, err)

	require.NotNil(t, s.Foo)
	require.ElementsMatch(t, []string{"first", "second"}, s.Foo.GetValue())
	require.NotNil(t, s.Bar)
	require.NotNil(t, s.Bar.GetValue())
	require.Nil(t, s.Bar.GetValue().Baz.GetValue())
	require.NotNil(t, s.Bar.GetValue().Abc)
	require.ElementsMatch(t, []bool{}, s.Bar.GetValue().Abc.GetValue())
}

// Test that the Kea JSON normalization works as expected.
func TestNormalizeKeaJSON(t *testing.T) {
	t.Run("standard-compliant JSON", func(t *testing.T) {
		// Arrange
		input := `{
			"number": 123,
			"string": "value",
			"array": [1, 2, 3],
			"object": {
				"key": "val"
			},
			"link": "http://example.com?a=1&b=2",
			"escaped": "Line1\nLine2\tTabbed"
		}`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		require.JSONEq(t, input, string(output))
	})

	t.Run("invalid standard-compliant JSON", func(t *testing.T) {
		// Arrange
		input := `{"foo":`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		require.Equal(t, input, string(output))
	})

	t.Run("JSON with single-line C-style comment - after value", func(t *testing.T) {
		// Arrange
		input := `{
			"a": 1 // This is a comment
		}`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{
			"a": 1
		}`
		require.JSONEq(t, expected, string(output))
	})

	t.Run("JSON with single-line C-style comment - commented entry", func(t *testing.T) {
		// Arrange
		input := `{
			// "a": 1
		}`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{}`
		require.JSONEq(t, expected, string(output))
	})

	t.Run("JSON with a string with comment characters", func(t *testing.T) {
		// Arrange
		input := `{
			"s": "///**/#"
		}`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{ "s": "///**/#" }`
		require.JSONEq(t, expected, string(output))
	})

	t.Run("JSON with a string with quote", func(t *testing.T) {
		// Arrange
		input := `{
			"s": "\""
		}`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{ "s": "\"" }`
		require.JSONEq(t, expected, string(output))
	})

	t.Run("JSON with multi-line C-style comment - single line", func(t *testing.T) {
		// Arrange
		input := `{ "a": /* 1 */ 2 }`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{ "a": 2 }`
		require.JSONEq(t, expected, string(output))
	})

	t.Run("JSON with multi-line C-style comment - multi line", func(t *testing.T) {
		// Arrange
		input := `{ "a": /* 
		  foo
		*/ 2 }`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{ "a": 2 }`
		require.JSONEq(t, expected, string(output))
	})

	t.Run("JSON with Python-style comment", func(t *testing.T) {
		// Arrange
		input := `{
			# "a": 1,
			"b": 2 # This is a comment
		}`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{
			"b": 2
		}`
		require.JSONEq(t, expected, string(output))
	})

	t.Run("JSON with mixed comments", func(t *testing.T) {
		// Arrange
		input := `{
			// "a": 1,
			"b": 2, /* multi-line
			comment */
			# Another comment
			"c": 3 // End comment
		}`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{
			"b": 2,
			"c": 3
		}`
		require.JSONEq(t, expected, string(output))
	})

	t.Run("JSON with comments in comments", func(t *testing.T) {
		// Arrange
		input := `{
			// This is a // nested comment
			"a": 1, // Comment with /* nested */ comment and # leading comment
			"b": 2 # Python-style # nested comment
		}`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{
			"a": 1,
			"b": 2
		}`
		require.JSONEq(t, expected, string(output))
	})

	t.Run("JSON with only comments", func(t *testing.T) {
		// Arrange
		input := `// Full line comment
		/* Multi-line
		comment */
		# Python-style comment`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		require.Empty(t, strings.TrimSpace(string(output)))
	})

	t.Run("Block comments cannot be nested", func(t *testing.T) {
		// Arrange
		input := `{ /* /* */ */ }`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{  */ }`
		require.Equal(t, expected, string(output))
	})

	t.Run("JSON with two single-line comments in a row with a quote", func(t *testing.T) {
		// Arrange
		input := `{
			//"
			//"
		}`

		// Act
		output := NormalizeKeaJSON([]byte(input))

		// Assert
		expected := `{ }`
		require.JSONEq(t, expected, string(output))
	})

	// Check the test cases from the jsonc library to ensure compatibility.
	validCases := []string{
		`{"foo":"bar foo","true":false,"number":42,"object":{"test":"done"},"array":[1,2,3],"url":"https://github.com","escape":"\"wo//rking"}`,
		`{"foo": /** this is a bloc/k comm\"ent */ "bar foo", "true": /* true */ false, "number": 42, "object": { "test": "done" }, "array" : [1, 2, 3], "url" : "https://github.com", "escape":"\"wo//rking" }`,
		"{\"foo\": // this is a \"single **/line comm\\\"ent\n\"bar foo\", \"true\": false, \"number\": 42, \"object\": { \"test\": \"done\" }, \"array\" : [1, 2, 3], \"url\" : \"https://github.com\", \"escape\":\"\\\"wo//rking\" }",
		"{\"foo\": # this is a single line comm\\\"ent\n\"bar foo\", \"true\": false, \"number\": 42, \"object\": { \"test\": \"done\" }, \"array\" : [1, 2, 3], \"url\" : \"https://github.com\", \"escape\":\"\\\"wo//rking\" }",
	}

	for i, c := range validCases {
		t.Run(fmt.Sprintf("jsonc-valid-case-%d", i), func(t *testing.T) {
			// Act
			output := NormalizeKeaJSON([]byte(c))

			// Assert
			require.True(t, json.Valid(output))
		})
	}

	invalidCases := []string{
		`{"foo": /* this is a block comment "bar foo", "true": false, "number": 42, "object": { "test": "done" }, "array" : [1, 2, 3], "url" : "https://github.com", "escape":"\"wo//rking }`,
		`{"foo": // this is a single line comment "bar foo", "true": false, "number": 42, "object": { "test": "done" }, "array" : [1, 2, 3], "url" : "https://github.com", "escape":"\"wo//rking" }`,
		`{"foo": # this is a single line comment "bar foo", "true": false, "number": 42, "object": { "test": "done" }, "array" : [1, 2, 3], "url" : "https://github.com", "escape":"\"wo//rking" }`,
	}

	for i, c := range invalidCases {
		t.Run(fmt.Sprintf("jsonc-invalid-case-%d", i), func(t *testing.T) {
			// Act
			output := NormalizeKeaJSON([]byte(c))

			// Assert
			require.False(t, json.Valid(output))
		})
	}
}
