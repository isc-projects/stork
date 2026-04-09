package storkutil

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the pairs zip iterator correctly zips together two slices.
func TestZipPairs(t *testing.T) {
	type pair struct {
		a int
		b string
	}
	consumeIterator := func(iter iter.Seq2[int, string]) []pair {
		var pairs []pair
		iter(func(a int, b string) bool {
			pairs = append(pairs, pair{a, b})
			return true
		})
		return pairs
	}

	t.Run("empty slices", func(t *testing.T) {
		i := ZipPairs([]int{}, []string{})
		require.NotNil(t, i)
		values := consumeIterator(i)
		require.Empty(t, values)
	})

	t.Run("empty and non-empty slices", func(t *testing.T) {
		i := ZipPairs([]int{1, 2, 3}, []string{})
		require.NotNil(t, i)
		values := consumeIterator(i)
		require.Empty(t, values)

		i = ZipPairs([]int{}, []string{"a", "b", "c"})
		require.NotNil(t, i)
		values = consumeIterator(i)
		require.Empty(t, values)
	})

	t.Run("two slices with the same lengths", func(t *testing.T) {
		i := ZipPairs([]int{1, 2, 3}, []string{"a", "b", "c"})
		require.NotNil(t, i)
		values := consumeIterator(i)
		require.Equal(t, []pair{
			{1, "a"},
			{2, "b"},
			{3, "c"},
		}, values)
	})

	t.Run("two slices with different lengths", func(t *testing.T) {
		i := ZipPairs([]int{1, 2}, []string{"a", "b", "c"})
		require.NotNil(t, i)
		values := consumeIterator(i)
		require.Equal(t, []pair{
			{1, "a"},
			{2, "b"},
		}, values)

		i = ZipPairs([]int{1, 2, 3}, []string{"a", "b"})
		require.NotNil(t, i)
		values = consumeIterator(i)
		require.Equal(t, []pair{
			{1, "a"},
			{2, "b"},
		}, values)
	})
}
