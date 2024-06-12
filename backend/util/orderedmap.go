package storkutil

// The specialized map that keeps the order of the keys.
//
// It supports three ways of iterating over the map:
//
// 1. Iterating by index:
//
//	for i := 0; i < m.GetSize(); i++ {
//		key, value := m.GetAt(i)
//		// Do something with the key and value.
//	}
//
// 2. Iterating with callback function:
//
//	m.ForEach(func(key TKey, value TValue) {
//		// Do something with the key and value.
//	})
//
// 3. Iterating with key-value pairs:
//
//	for _, entry := range m.GetEntries() {
//		key := entry.Key
//		value := entry.Value
//		// Do something with the key and value.
//	}
type OrderedMap[TKey comparable, TValue any] struct {
	keys []TKey
	data map[TKey]TValue
}

// Creates a new instance of the ordered map.
func NewOrderedMap[TKey comparable, TValue any]() *OrderedMap[TKey, TValue] {
	return &OrderedMap[TKey, TValue]{
		keys: make([]TKey, 0),
		data: make(map[TKey]TValue),
	}
}

// Creates a new instance of the ordered map from the given keys and values.
// The length of the keys and values must be the same.
func NewOrderedMapFromEntries[TKey comparable, TValue any](keys []TKey, values []TValue) *OrderedMap[TKey, TValue] {
	m := NewOrderedMap[TKey, TValue]()
	for i, key := range keys {
		m.Set(key, values[i])
	}
	return m
}

// Sets the value for the given key. If the key already exists, the value will
// be updated.
func (m *OrderedMap[TKey, TValue]) Set(key TKey, value TValue) {
	if _, ok := m.data[key]; !ok {
		m.keys = append(m.keys, key)
	}
	m.data[key] = value
}

// Gets the value for the given key. If the key does not exist, the second
// return value will be false.
func (m *OrderedMap[TKey, TValue]) Get(key TKey) (TValue, bool) {
	value, ok := m.data[key]
	return value, ok
}

// Gets the key and value at the given index. It panics if the index is out of
// range.
func (m *OrderedMap[TKey, TValue]) GetAt(index int) (TKey, TValue) {
	key := m.keys[index]
	value := m.data[key]
	return key, value
}

// Deletes the key from the map. If the key does not exist, it does nothing.
func (m *OrderedMap[TKey, TValue]) Delete(key TKey) {
	if _, ok := m.data[key]; !ok {
		return
	}

	delete(m.data, key)
	for i, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
}

// Clears the map.
func (m *OrderedMap[TKey, TValue]) Clear() {
	m.keys = make([]TKey, 0)
	m.data = make(map[TKey]TValue)
}

// Returns a slice of keys in the order they were inserted.
func (m *OrderedMap[TKey, TValue]) GetKeys() []TKey {
	return m.keys
}

// Returns a slice of values in the order they were inserted.
func (m *OrderedMap[TKey, TValue]) GetValues() []TValue {
	values := make([]TValue, 0)
	for _, key := range m.keys {
		values = append(values, m.data[key])
	}
	return values
}

// Returns a slice of key-value pairs in the order they were inserted.
func (m *OrderedMap[TKey, TValue]) GetEntries() []struct {
	Key   TKey
	Value TValue
} {
	entries := make([]struct {
		Key   TKey
		Value TValue
	}, 0)
	for _, key := range m.keys {
		entries = append(entries, struct {
			Key   TKey
			Value TValue
		}{Key: key, Value: m.data[key]})
	}
	return entries
}

// Returns the number of key-value pairs in the map.
func (m *OrderedMap[TKey, TValue]) GetSize() int {
	return len(m.keys)
}

// Iterates over the key-value pairs in the map in the order they were inserted.
// The iteration can be stopped by returning false from the callback function.
func (m *OrderedMap[TKey, TValue]) ForEach(callback func(TKey, TValue) bool) {
	for _, key := range m.keys {
		if !callback(key, m.data[key]) {
			break
		}
	}
}
