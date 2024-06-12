package storkutil_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the NewOrderedMap function creates a new instance of the ordered
// map.
func TestNewOrderedMap(t *testing.T) {
	// Arrange & Act
	orderedMap := storkutil.NewOrderedMap[string, int]()

	// Assert
	require.NotNil(t, orderedMap)
	require.EqualValues(t, 0, orderedMap.GetSize())
}

// Test that the ordered map instance can be created from the given keys and
// values.
func TestNewOrderedMapFromEntries(t *testing.T) {
	// Arrange
	keys := []string{"key0", "key1", "key2"}
	values := []int{0, 1, 2}

	// Act
	orderedMap := storkutil.NewOrderedMapFromEntries(keys, values)

	// Assert
	require.NotNil(t, orderedMap)
	require.EqualValues(t, 3, orderedMap.GetSize())
	key, value := orderedMap.GetAt(0)
	require.Equal(t, "key0", key)
	require.Equal(t, 0, value)
	key, value = orderedMap.GetAt(1)
	require.Equal(t, "key1", key)
	require.Equal(t, 1, value)
	key, value = orderedMap.GetAt(2)
	require.Equal(t, "key2", key)
	require.Equal(t, 2, value)
}

// Test that the extra values are ignored when creating an ordered map from the
// given keys and values.
func TestNewOrderedMapFromEntriesExtraValues(t *testing.T) {
	// Arrange
	keys := []string{"key0", "key1", "key2"}
	values := []int{0, 1, 2, 3}

	// Act
	orderedMap := storkutil.NewOrderedMapFromEntries(keys, values)

	// Assert
	require.NotNil(t, orderedMap)
	require.EqualValues(t, 3, orderedMap.GetSize())
	actualValues := orderedMap.GetValues()
	require.Equal(t, values[0:3], actualValues)
}

// Test that the extra keys cause the panic when creating an ordered map from
// the given keys and values.
func TestNewOrderedMapFromEntriesExtraKeys(t *testing.T) {
	// Arrange
	keys := []string{"key0", "key1", "key2", "key3"}
	values := []int{0, 1, 2}

	// Act & Assert
	require.Panics(t, func() {
		storkutil.NewOrderedMapFromEntries(keys, values)
	})
}

// Test that the Set method sets the value for the given key and keeps the
// order of the keys.
func TestOrderedMapSet(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMap[string, int]()

	// Act
	orderedMap.Set("key0", 0)
	orderedMap.Set("key1", 1)
	orderedMap.Set("key2", 2)
	orderedMap.Set("key2", 3)

	// Assert
	require.EqualValues(t, 3, orderedMap.GetSize())
	key, value := orderedMap.GetAt(0)
	require.Equal(t, "key0", key)
	require.Equal(t, 0, value)
	key, value = orderedMap.GetAt(1)
	require.Equal(t, "key1", key)
	require.Equal(t, 1, value)
	key, value = orderedMap.GetAt(2)
	require.Equal(t, "key2", key)
	require.Equal(t, 3, value)
}

// Test that the Get method returns the value at the given key.
func TestOrderedMapGet(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)

	// Act
	value1, ok1 := orderedMap.Get("key1")
	valueUnknown, okUnknown := orderedMap.Get("unknown")

	// Assert
	require.True(t, ok1)
	require.Equal(t, 1, value1)

	require.False(t, okUnknown)
	require.Zero(t, valueUnknown)
}

// Test that the GetAt method returns the key and value at the given index.
func TestOrderedMapGetAt(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)

	t.Run("valid index", func(t *testing.T) {
		// Act
		key, value := orderedMap.GetAt(1)

		// Assert
		require.Equal(t, "key1", key)
		require.Equal(t, 1, value)
	})

	t.Run("invalid index", func(t *testing.T) {
		// Act & Assert
		require.Panics(t, func() {
			orderedMap.GetAt(42)
		})
	})
}

// Test that the Delete method removes the key and value from the ordered map.
func TestOrderedMapDelete(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)

	t.Run("delete unknown key", func(t *testing.T) {
		// Act
		orderedMap.Delete("unknown")

		// Assert
		require.EqualValues(t, 3, orderedMap.GetSize())
	})

	t.Run("delete known key", func(t *testing.T) {
		// Act
		orderedMap.Delete("key1")

		// Assert
		require.EqualValues(t, 2, orderedMap.GetSize())
		key, value := orderedMap.GetAt(0)
		require.Equal(t, "key0", key)
		require.Equal(t, 0, value)
		key, value = orderedMap.GetAt(1)
		require.Equal(t, "key2", key)
		require.Equal(t, 2, value)
	})
}

// Test that the GetKeys method returns the keys in the order they were added.
func TestOrderedMapGetKeys(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)

	// Act
	keys := orderedMap.GetKeys()

	// Assert
	require.Equal(t, []string{"key0", "key1", "key2"}, keys)
}

// Test that the GetValues method returns the values in the order they were added.
func TestOrderedMapGetValues(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)

	// Act
	values := orderedMap.GetValues()

	// Assert
	require.Equal(t, []int{0, 1, 2}, values)
}

// Test that the GetEntries method returns the key-value pairs in the order they
// were added.
func TestOrderedMapGetEntries(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)

	// Act
	entries := orderedMap.GetEntries()

	// Assert
	for i := 0; i < orderedMap.GetSize(); i++ {
		require.Equal(t, fmt.Sprintf("key%d", i), entries[i].Key)
		require.Equal(t, i, entries[i].Value)
	}
}

// Test that the Clear method removes all key-value pairs from the ordered map.
func TestOrderedMapClear(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)

	// Act
	orderedMap.Clear()

	// Assert
	require.Zero(t, orderedMap.GetSize())
}

// Test that the ForEach method iterates over the key-value pairs in the order
// they were added.
func TestOrderedMapForEach(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)
	var keys []string
	var values []int

	// Act
	orderedMap.ForEach(func(key string, value int) bool {
		keys = append(keys, key)
		values = append(values, value)
		return true
	})

	// Assert
	require.Equal(t, []string{"key0", "key1", "key2"}, keys)
	require.Equal(t, []int{0, 1, 2}, values)
}

// Test that the ForEach method does not iterate over the key-value pairs when
// the ordered map is empty.
func TestOrderedMapForEachEmpty(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMap[string, int]()
	var keys []string
	var values []int

	// Act
	orderedMap.ForEach(func(key string, value int) bool {
		keys = append(keys, key)
		values = append(values, value)
		return true
	})

	// Assert
	require.Empty(t, keys)
	require.Empty(t, values)
}

// Test that the ForEach method stops the iteration when the callback returns
// false.
func TestOrderedMapForEachStop(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)
	var keys []string
	var values []int

	// Act
	orderedMap.ForEach(func(key string, value int) bool {
		keys = append(keys, key)
		values = append(values, value)
		return key != "key1"
	})

	// Assert
	require.Equal(t, []string{"key0", "key1"}, keys)
	require.Equal(t, []int{0, 1}, values)
}

// Test that the GetSize method returns the number of key-value pairs in the
// ordered map.
func TestOrderedMapGetSize(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMapFromEntries(
		[]string{"key0", "key1", "key2"},
		[]int{0, 1, 2},
	)

	// Act
	size := orderedMap.GetSize()

	// Assert
	require.EqualValues(t, 3, size)
}

// Test that the GetSize method returns zero when the ordered map is empty.
func TestOrderedMapGetSizeEmpty(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMap[string, int]()

	// Act
	size := orderedMap.GetSize()

	// Assert
	require.Zero(t, size)
}

// Test that the GetSize method returns the proper number of key-value pairs
// after modifying the ordered map.
func TestOrderedMapGetSizeAfterModification(t *testing.T) {
	// Arrange
	orderedMap := storkutil.NewOrderedMap[string, int]()

	// Act
	orderedMap.Set("key0", 0)
	orderedMap.Set("key1", 1)
	orderedMap.Delete("key0")

	// Assert
	require.EqualValues(t, 1, orderedMap.GetSize())
}

// Test that the map can be iterated over using various methods.
func TestOrderedMapIteration(t *testing.T) {
	// Arrange
	expectedKeys := []string{"key0", "key1", "key2"}
	expectedValues := []int{0, 1, 2}
	orderedMap := storkutil.NewOrderedMapFromEntries(
		expectedKeys,
		expectedValues,
	)

	t.Run("by index", func(t *testing.T) {
		var actualKeys []string
		var actualValues []int

		// Act
		for i := 0; i < orderedMap.GetSize(); i++ {
			key, value := orderedMap.GetAt(i)
			actualKeys = append(actualKeys, key)
			actualValues = append(actualValues, value)
		}

		// Assert
		require.Equal(t, expectedKeys, actualKeys)
		require.Equal(t, expectedValues, actualValues)
	})

	t.Run("with callback function", func(t *testing.T) {
		var actualKeys []string
		var actualValues []int

		// Act
		orderedMap.ForEach(func(key string, value int) bool {
			actualKeys = append(actualKeys, key)
			actualValues = append(actualValues, value)
			return true
		})

		// Assert
		require.Equal(t, expectedKeys, actualKeys)
		require.Equal(t, expectedValues, actualValues)
	})

	t.Run("with key-value pairs", func(t *testing.T) {
		var actualKeys []string
		var actualValues []int

		// Act
		for _, entry := range orderedMap.GetEntries() {
			actualKeys = append(actualKeys, entry.Key)
			actualValues = append(actualValues, entry.Value)
		}

		// Assert
		require.Equal(t, expectedKeys, actualKeys)
		require.Equal(t, expectedValues, actualValues)
	})
}
