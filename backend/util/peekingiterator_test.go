package storkutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the peeking iterator can be created and used.
func TestPeekingIterator(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	iterator := NewPeekingIterator(items)

	item, ok := iterator.Next()
	require.True(t, ok)
	require.Equal(t, "item1", item)

	item, ok = iterator.PeekBack()
	require.False(t, ok)
	require.Empty(t, item)

	item, ok = iterator.Peek()
	require.True(t, ok)
	require.Equal(t, "item2", item)

	item, ok = iterator.PeekBack()
	require.False(t, ok)
	require.Empty(t, item)

	item, ok = iterator.Next()
	require.True(t, ok)
	require.Equal(t, "item2", item)

	item, ok = iterator.Peek()
	require.True(t, ok)
	require.Equal(t, "item3", item)

	item, ok = iterator.PeekBack()
	require.True(t, ok)
	require.Equal(t, "item1", item)

	item, ok = iterator.Next()
	require.True(t, ok)
	require.Equal(t, "item3", item)

	item, ok = iterator.PeekBack()
	require.True(t, ok)
	require.Equal(t, "item2", item)

	item, ok = iterator.Peek()
	require.False(t, ok)
	require.Equal(t, "", item)

	item, ok = iterator.Next()
	require.False(t, ok)
	require.Equal(t, "", item)
}

// Test that the peeking iterator can be created and used when the list is empty.
func TestPeekingIteratorEmpty(t *testing.T) {
	iterator := NewPeekingIterator([]string{})

	item, ok := iterator.Next()
	require.False(t, ok)
	require.Empty(t, item)

	item, ok = iterator.Peek()
	require.False(t, ok)
	require.Empty(t, item)

	item, ok = iterator.PeekBack()
	require.False(t, ok)
	require.Empty(t, item)
}

// Test that the peeking iterator can peek all subsequent items without consuming them.
func TestPeekingIteratorPeekSubsequent(t *testing.T) {
	// Create a new peeking iterator with the given items.
	items := []string{"item1", "item2", "item3"}
	iterator := NewPeekingIterator(items)

	// Peek all subsequent items. All should be returned.
	items = iterator.PeekSubsequent()
	require.Equal(t, []string{"item1", "item2", "item3"}, items)

	// Consume the first item.
	item, ok := iterator.Next()
	require.True(t, ok)
	require.Equal(t, "item1", item)

	// Only two remaining items should be returned.
	items = iterator.PeekSubsequent()
	require.Equal(t, []string{"item2", "item3"}, items)

	// Consume the second item.
	item, ok = iterator.Next()
	require.True(t, ok)
	require.Equal(t, "item2", item)

	// Only one remaining item should be returned.
	items = iterator.PeekSubsequent()
	require.Equal(t, []string{"item3"}, items)

	// Consume the third item.
	item, ok = iterator.Next()
	require.True(t, ok)
	require.Equal(t, "item3", item)

	// No remaining items should be returned.
	items = iterator.PeekSubsequent()
	require.Empty(t, items)
}
