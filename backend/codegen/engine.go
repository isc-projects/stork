package codegen

import "reflect"

// An interface of the engines generating the code from JSON.
type Engine interface {
	// Returns the engine type.
	GetEngineType() string
	// Returns indentation kind used by the engine (e.g. tabs).
	getIndentationKind() Indentation
	// Returns a token opening a slice.
	beginSlice(n *node) string
	// Returns a token ending a slice.
	endSlice() string
	// Returns a token opening a map.
	beginMap(n *node) string
	// Returns a token ending a map.
	endMap() string
	// Returns a formatted map key, using appropriate keys and character set.
	formatKey(key string) string
	// Returns formatted primitive value.
	formatPrimitive(value reflect.Value) string
	// Adds indentation up to the specified position.
	indent(position int) string
	// Adds spaces between the map key and the value for the specified key.
	align(key string, longestKeyLength int) string
}
