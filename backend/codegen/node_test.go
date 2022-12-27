package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test instantiating a new n.
func TestNewNode(t *testing.T) {
	n := newNode(0, mapNode)
	require.NotNil(t, n)
	require.True(t, n.isRoot())
	require.False(t, n.isParentArray())
	require.False(t, n.isParentMap())
}

// Test checking that a parent n is an array.
func TestIsParentArray(t *testing.T) {
	n := newNode(0, arrayNode)
	require.NotNil(t, n)

	n = n.createChild(mapNode)
	require.True(t, n.isParentArray())
	n = n.createChild(mapNode)
	require.False(t, n.isParentArray())
}

// Test creating a child n.
func TestCreateChild(t *testing.T) {
	n := newNode(0, arrayNode)
	require.NotNil(t, n)

	node2 := n.createChild(arrayNode)
	require.NotNil(t, node2)
	require.NotEqual(t, n, node2)
	require.Equal(t, n, node2.parent)
	require.False(t, node2.isRoot())
}

// Test that indentation is correctly propagated to children.
func TestNodeIndent(t *testing.T) {
	n := newNode(1, arrayNode)
	require.NotNil(t, n)
	require.Equal(t, 1, n.getIndentation())

	n = n.createChild(mapNode)
	require.Equal(t, 2, n.getIndentation())

	n = n.createChild(mapNode)
	require.Equal(t, 3, n.getIndentation())

	n = n.createChild(arrayNode)
	require.Equal(t, 4, n.getIndentation())

	n = n.createChild(arrayNode)
	require.Equal(t, 5, n.getIndentation())

	n = n.createChild(arrayNode)
	require.Equal(t, 6, n.getIndentation())
}
