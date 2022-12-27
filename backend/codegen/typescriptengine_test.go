package codegen

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test instantiating a new typescript engine.
func TestNewTypescriptEngine(t *testing.T) {
	engine := NewTypescriptEngine()
	require.NotNil(t, engine)
	require.Equal(t, fourSpaces, engine.getIndentationKind())
	require.Equal(t, TypescriptEngineType, engine.GetEngineType())
}

// Test that slice beginning and ending tokens are correct.
func TestTypescriptGetSliceBeginEnd(t *testing.T) {
	engine := NewTypescriptEngine()
	require.NotNil(t, engine)
	require.Equal(t, "[", engine.beginSlice(newNode(0, arrayNode)))
	require.Equal(t, "]", engine.endSlice())
}

// Test that map beginning and ending tokens are correct.
func TestTypescriptBeginMapEnd(t *testing.T) {
	engine := NewTypescriptEngine()
	require.NotNil(t, engine)
	require.Equal(t, "{", engine.beginMap(newNode(0, mapNode)))
	require.Equal(t, "}", engine.endMap())
}

// Test that a map key is correctly formatted. It must begin with a lower case
// and all special characters must be removed.
func TestTypescriptFormatKey(t *testing.T) {
	engine := NewTypescriptEngine()
	require.NotNil(t, engine)
	require.Equal(t, "fooBarBazAbc", engine.formatKey("foo-bar_baz&abc"))
	require.Equal(t, "fooBaz", engine.formatKey("FOO-BAZ"))
}

// Test that primitive values are formatted correctly.
func TestTypescriptFormatPrimitive(t *testing.T) {
	engine := NewTypescriptEngine()
	require.NotNil(t, engine)

	t.Run("string", func(t *testing.T) {
		value := "foo"
		require.Equal(t, "'foo'", engine.formatPrimitive(reflect.ValueOf(value)))
	})

	t.Run("integer", func(t *testing.T) {
		value := 123
		require.Equal(t, "123", engine.formatPrimitive(reflect.ValueOf(value)))
	})

	t.Run("boolean", func(t *testing.T) {
		value := true
		require.Equal(t, "true", engine.formatPrimitive(reflect.ValueOf(value)))
	})
}

// Test that spaces are used for indentation and that the correct
// number of spaces is returned.
func TestTypescriptIndent(t *testing.T) {
	engine := NewTypescriptEngine()
	require.NotNil(t, engine)
	require.Empty(t, engine.indent(0))
	require.Equal(t, "    ", engine.indent(1))
	require.Equal(t, "        ", engine.indent(2))
}

// Test that empty alignment is always returned for the typescript engine.
func TestTypescriptAlign(t *testing.T) {
	engine := NewTypescriptEngine()
	require.NotNil(t, engine)
	require.Empty(t, engine.align("foo", 3))
	require.Empty(t, engine.align("foo", 6))
}
