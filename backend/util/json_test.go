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
