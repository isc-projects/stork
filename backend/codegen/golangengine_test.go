package codegen

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test instantiating a new golang engine.
func TestNewGolangEngine(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)
	require.Equal(t, tabs, engine.getIndentationKind())
	require.Equal(t, GolangEngineType, engine.GetEngineType())
}

// Test that correct beginning and ending tokens are returned for a
// slice without an explicit type.
func TestGolangBeginEndSliceAny(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)
	require.Equal(t, "[]any{", engine.beginSlice(newNode(0, arrayNode)))
	require.Equal(t, "}", engine.endSlice())
}

// Test that correct beginning token is returned for a slice of a
// specified type.
func TestGolangBeginSliceTopLevelType(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)
	engine.SetTopLevelType("dhcpOptionDefinition")
	require.Equal(t, "[]dhcpOptionDefinition{", engine.beginSlice(newNode(0, arrayNode)))
	require.Equal(t, "{", engine.beginSlice(newNode(0, arrayNode).createChild(arrayNode)))
}

// Test that slice beginning token contains statically assigned type.
func TestGolangBeginSliceStatic(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)
	// For JSON key "bar" let's statically assign the golang type "foo".
	engine.SetStaticFieldTypes([]string{"bar:foo"})
	n := newNode(0, mapNode).createMapChild("bar", arrayNode)
	require.NotNil(t, n)
	require.Equal(t, "[]foo{", engine.beginSlice(n))
}

// Test that correct beginning and ending tokens are returned for a
// map without an explicit type.
func TestGolangBeginMapEndAny(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)
	require.Equal(t, "any{", engine.beginMap(newNode(0, mapNode)))
	require.Equal(t, "}", engine.endMap())
}

// Test that correct beginning token is returned for a map of a
// specified type.
func TestGolangBeginMapTopLevelType(t *testing.T) {
	engine := NewGolangEngine()
	engine.SetTopLevelType("dhcpOptionDefinition")
	require.NotNil(t, engine)
	require.Equal(t, "dhcpOptionDefinition{", engine.beginMap(newNode(0, mapNode)))
	require.Equal(t, "any{", engine.beginMap(newNode(0, mapNode).createChild(mapNode)))
}

// Test that map beginning token contains statically assigned type.
func TestGolangBeginMapStatic(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)
	// For JSON key "bar" let's statically assign the golang type "foo".
	engine.SetStaticFieldTypes([]string{"bar:foo"})
	n := newNode(0, mapNode).createMapChild("bar", mapNode)
	require.NotNil(t, n)
	require.Equal(t, "foo{", engine.beginMap(n))
}

// Test that a map key is correctly formatted. It must begin with an upper case
// and all special characters must be removed.
func TestGolangGetFormattedKey(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)
	require.Equal(t, "FooBarBazAbc", engine.formatKey("foo-bar_baz&abc"))
	require.Equal(t, "FooBaz", engine.formatKey("FOO-BAZ"))
}

// Test that primitive values are formatted correctly.
func TestGolangFormatPrimitive(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)

	t.Run("string", func(t *testing.T) {
		value := "foo"
		require.Equal(t, `"foo"`, engine.formatPrimitive(reflect.ValueOf(value)))
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

// Test that tabs are used for indentation and that the correct
// number of tabs is returned.
func TestGolangIndent(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)
	require.Empty(t, engine.indent(0))
	require.Equal(t, "\t", engine.indent(1))
	require.Equal(t, "\t\t", engine.indent(2))
}

// Test that golang map values are properly aligned.
func TestGolangAlign(t *testing.T) {
	engine := NewGolangEngine()
	require.NotNil(t, engine)
	require.Empty(t, engine.align("foo", 3))
	require.Equal(t, "   ", engine.align("foo", 6))
}
