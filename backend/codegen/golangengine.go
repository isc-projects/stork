package codegen

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Golang engine selector used in the command line.
const GolangEngineType = "golang"

// Implements a code generation engine for Golang. It is used to
// generate Golang data structures from JSON data.
type GolangEngine struct {
	titleCaser   cases.Caser
	topLevelType string
	replacer     *regexp.Regexp
	typeMappings map[string]string
}

// Creates new Golang engine instance.
func NewGolangEngine() *GolangEngine {
	return &GolangEngine{
		titleCaser:   cases.Title(language.AmericanEnglish),
		replacer:     regexp.MustCompile("[^A-Za-z0-9]"),
		typeMappings: make(map[string]string),
	}
}

// Sets top-level data type. In Golang, it precedes an array or map opening
// token. For example, for someType, the opening token will be []someType{.
func (e *GolangEngine) SetTopLevelType(topLevelType string) {
	e.topLevelType = topLevelType
}

// Converts and populates the static mappings of the JSON keys to the respective
// types in the output code structure. The mappings are specified in the command line
// using the <json-key>:<field-type> notation and are supplied as a slice to this
// function. Suppose the user specified --field-type record-types:DHCPOptionType. For
// each JSON map key "record-types", the resulting code structure will use DHCPOptionType
// or []DHCPOptionType (depending on whether it is a slice or map value) as a type in
// the generated code. If the mapping is not specified, the engine will use "any"
// interface instead.
func (e *GolangEngine) SetStaticFieldTypes(mappings []string) error {
	return parseMappings(mappings, e.typeMappings)
}

// Returns the engine type.
func (e *GolangEngine) GetEngineType() string {
	return GolangEngineType
}

// Returns indentation type used by the engine (i.e. tabs).
func (e *GolangEngine) getIndentationKind() Indentation {
	return tabs
}

// Returns the token opening a slice.
func (e *GolangEngine) beginSlice(n *node) string {
	switch {
	case n.isRoot() && len(e.topLevelType) > 0:
		return fmt.Sprintf("[]%s{", e.topLevelType)
	case n.isParentArray():
		return "{"
	case len(n.key) > 0 && len(e.typeMappings[n.key]) > 0:
		return fmt.Sprintf("[]%s{", e.typeMappings[n.key])
	default:
		return "[]any{"
	}
}

// Returns the token ending a slice.
func (e *GolangEngine) endSlice() string {
	return "}"
}

// Returns the token beginning a map.
func (e *GolangEngine) beginMap(n *node) string {
	switch {
	case n.isRoot() && len(e.topLevelType) > 0:
		return fmt.Sprintf("%s{", e.topLevelType)
	case n.isParentArray():
		return "{"
	case len(n.key) > 0 && len(e.typeMappings[n.key]) > 0:
		return fmt.Sprintf("%s{", e.typeMappings[n.key])
	default:
		return "any{"
	}
}

// Returns the token ending a map.
func (e *GolangEngine) endMap() string {
	return "}"
}

// Formats a map key. It converts the key to the Pascal case and removes
// all special characters.
func (e *GolangEngine) formatKey(key string) string {
	key = e.replacer.ReplaceAllString(key, "-")
	key = e.titleCaser.String(key)
	key = strings.ReplaceAll(key, "-", "")
	return key
}

// It formats the primitive value. A string value is returned in quotes.
// Other values are returned without change.
func (e *GolangEngine) formatPrimitive(value reflect.Value) string {
	if value.Kind() == reflect.String {
		return fmt.Sprintf("\"%s\"", value)
	}
	return fmt.Sprint(value)
}

// Returns a indetation up to the specified position.
func (e *GolangEngine) indent(position int) string {
	return strings.Repeat("\t", position)
}

// Returns gap between the map key and the map value. The key is the current key
// for which the gap is to be returned. The longestKeyLength is the length of the
// longest key in the map.
func (e *GolangEngine) align(key string, longestKeyLength int) string {
	return strings.Repeat(" ", longestKeyLength-len(key))
}
