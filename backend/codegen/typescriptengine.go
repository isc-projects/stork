package codegen

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Typescript language selector used in the command line.
const TypescriptEngineType = "typescript"

// Implements a code generation engine for Typescript. It is used to
// generate Typescript data structures from JSON data.
type TypescriptEngine struct {
	titleCaser cases.Caser
	lowerCaser cases.Caser
	replacer   *regexp.Regexp
}

// Creates new Typescript engine instance.
func NewTypescriptEngine() *TypescriptEngine {
	return &TypescriptEngine{
		titleCaser: cases.Title(language.AmericanEnglish),
		lowerCaser: cases.Lower(language.AmericanEnglish),
		replacer:   regexp.MustCompile("[^A-Za-z0-9]"),
	}
}

// Returns the engine type.
func (e *TypescriptEngine) GetEngineType() string {
	return TypescriptEngineType
}

// Returns indentation type used by the engine (i.e. four spaces).
func (e *TypescriptEngine) getIndentationKind() Indentation {
	return fourSpaces
}

// Returns the token opening a slice.
func (e *TypescriptEngine) beginSlice(n *node) string {
	return "["
}

// Returns the token ending a slice.
func (e *TypescriptEngine) endSlice() string {
	return "]"
}

// Returns the token beginning a map.
func (e *TypescriptEngine) beginMap(n *node) string {
	return "{"
}

// Returns the token ending a map.
func (e *TypescriptEngine) endMap() string {
	return "}"
}

// Formats a map key. It converts the key to the camel case and removes
// all special characters.
func (e *TypescriptEngine) formatKey(key string) string {
	key = e.replacer.ReplaceAllString(key, "-")
	key = e.titleCaser.String(key)
	key = strings.ReplaceAll(key, "-", "")
	key = e.lowerCaser.String(key[0:1]) + key[1:]
	return key
}

// It formats the primitive value. A string value is returned in quotes.
// Other values are returned without change.
func (e *TypescriptEngine) formatPrimitive(value reflect.Value) string {
	if value.Kind() == reflect.String {
		return fmt.Sprintf("'%s'", value)
	}
	return fmt.Sprint(value)
}

// Returns a indetation up to the specified position.
func (e *TypescriptEngine) indent(position int) string {
	return strings.Repeat(" ", getIndentationCoefficient(fourSpaces)*position)
}

// Returns gap between the map key and the map value. The key is the current key
// for which the gap is to be returned. The longestKeyLength is the length of the
// longest key in the map.
func (e *TypescriptEngine) align(key string, longestKeyLength int) string {
	return ""
}
