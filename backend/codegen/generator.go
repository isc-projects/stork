package codegen

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

const (
	newLine      = "\n"
	commaNewLine = ",\n"
)

// Code generator processing JSON structure and converting it to the corresponding
// code structure in the selected programming language. The language depends on
// the selected engine.
type Generator struct {
	engine        Engine
	builder       strings.Builder
	fieldMappings map[string]string
}

// Creates new generator instance with the specified language engine.
func NewGenerator(engine Engine) *Generator {
	return &Generator{
		engine:        engine,
		fieldMappings: make(map[string]string),
	}
}

// Converts and populates the static mappings of the JSON key names to the respective
// fields in the output code structure. The mappings are specified in the command line
// using the <json-key>:<field-name> notation and are supplied as a slice to this
// function. Suppose the user specified --field-name type:OptionType. For each JSON
// map key "type", the resulting code structure will use the "OptionType" as a field name,
// instead of the Type field name.
func (g *Generator) SetStaticFieldNames(mappings []string) error {
	return parseMappings(mappings, g.fieldMappings)
}

// Processes JSON file and generates structures in the specified language.
// The filename specifies source JSON file location. The ident is the
// number of indentations the output should begin from.
func (g *Generator) generateStructs(filename string, indent int) error {
	// Open the input file.
	jsonFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	// Read the file contents.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	// Parse JSON contents.
	var parsedJSON any
	err = json.Unmarshal(byteValue, &parsedJSON)
	if err != nil {
		return err
	}
	// JSON contents can be an array or a map.
	switch reflect.TypeOf(parsedJSON).Kind() {
	case reflect.Slice:
		// Walk over the JSON array recursively.
		g.processSlice(newNode(indent, arrayNode), reflect.ValueOf(parsedJSON))
	case reflect.Map:
		// Walk over the JSON map recursively.
		g.processMap(newNode(indent, mapNode), reflect.ValueOf(parsedJSON))
	default:
		return errors.Errorf("require array or map at top level of the JSON file %s", filename)
	}
	return nil
}

// Exported function processing JSON file and generating structures in the
// specified language. The filename specifies source file location.
func (g *Generator) GenerateStructs(filename string) error {
	// Start at indentation level 0.
	return g.generateStructs(filename, 0)
}

// Processes JSON file, generates structures and embeds the output in the
// template file. The template file is expected to include the %%% placeholder
// for the output. It checks the indentation of the placeholder and follows the
// indentation while it generates the structures. The filename is the input file
// holding JSON structure. The template is the location of the template file.
func (g *Generator) GenerateStructsWithTemplate(filename, template string) error {
	// Open the template file.
	templateFile, err := os.Open(template)
	if err != nil {
		return err
	}
	defer templateFile.Close()
	// Read the template file.
	byteValue, err := ioutil.ReadAll(templateFile)
	if err != nil {
		return err
	}
	stringValue := string(byteValue)

	// We use the regexp to find the placeholder and to determine the placeholder's
	// indentation. The expression depends on whether we're using tabs or spaces for
	// indentation.
	var re *regexp.Regexp
	switch g.engine.getIndentationKind() {
	case tabs:
		re = regexp.MustCompile(`(\t*)([^\t])+%%%`)
	default:
		re = regexp.MustCompile(`([    ]*)(.+)%%%`)
	}
	// Check if we found a placeholder.
	match := re.FindStringSubmatch(stringValue)
	if len(match) == 0 {
		return errors.Errorf("the template does not contain a proper %%%%%% placeholder")
	}
	// The initial indentation is the indentation of the line including the placeholder.
	// We check the indentation size using the match group. The indentation coefficient
	// specifies how many characters a single indentation contains. For example, the four
	// spaces indentation kind has a coefficient value of 4.
	err = g.generateStructs(filename, len(match[1])/getIndentationCoefficient(g.engine.getIndentationKind()))
	if err != nil {
		return err
	}
	// Get the generated output.
	output := g.builder.String()
	// Remove the indentation of the first line if the line doesn't start with a
	// placeholder. For example it can start with "let options = ", followed by
	// the placeholder. We don't want the indentation after "=".
	if len(match[2]) != 0 {
		output = strings.TrimLeft(output, "\t ")
	}
	// Insert the generated output into the template.
	g.builder.Reset()
	_, err = g.builder.WriteString(strings.ReplaceAll(stringValue, "%%%", output))
	if err != nil {
		return err
	}
	return nil
}

// Prints the generated output to the stdout.
func (g *Generator) Print() {
	fmt.Print(g.builder.String())
}

// Write the generated output into the file.
func (g *Generator) Write(filename string) error {
	outputFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	_, err = outputFile.WriteString(g.builder.String())
	return err
}

// Internal implemenation of a recursive JSON array processing. The node is
// the JSON node holding the processed array. It holds the information about
// the path to the processed array. The sliceValue contains the array data.
func (g *Generator) processSlice(n *node, sliceValue reflect.Value) {
	if !n.isParentMap() {
		g.append(g.engine.indent(n.getIndentation()))
	}
	// Add opening mark for the array (e.g., "[" sign).
	g.append(g.engine.beginSlice(n), newLine)
	// Schedule closure of the array when this function ends (e.g., adding "]" sign).
	defer func() {
		g.append(g.engine.indent(n.getIndentation()), g.engine.endSlice())
		if !n.isRoot() {
			g.append(commaNewLine)
		}
	}()
	// Iterate over the array elements and process them recursively.
	for i := 0; i < sliceValue.Len(); i++ {
		value := sliceValue.Index(i)
		// Make sure it is not a pointer or interface but actual value.
		if value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
			value = value.Elem()
		}
		// The value can be an array, map or a primitive. Increase the indentation
		// and update the node with the current path.
		switch value.Kind() {
		case reflect.Slice:
			g.processSlice(n.createChild(arrayNode), value)
		case reflect.Map:
			g.processMap(n.createChild(mapNode), value)
		default:
			g.processPrimitive(n.createChild(leaf), value)
		}
	}
}

// Internal implementation of a recursive map processing. The node is the JSON
// node holding the processed map. It holds the information about the path to
// the processed map. The mapValue contains the map data.
func (g *Generator) processMap(n *node, mapValue reflect.Value) {
	// Make sure it is not a pointer or interface but actual value.
	if mapValue.Kind() == reflect.Pointer || mapValue.Kind() == reflect.Interface {
		mapValue = mapValue.Elem()
	}
	// Optionally add indentation. If the parent is a map, the child map
	// begins in the same line, presumably right after the colon sign.
	if !n.isParentMap() {
		g.append(g.engine.indent(n.getIndentation()))
	}
	// Add opening mark for a map (e.g., "{" sign).
	g.append(g.engine.beginMap(n), newLine)
	// Schedule closure of the map when this function ends (e.g., adding "}" sign).
	defer func() {
		g.append(g.engine.indent(n.getIndentation()), g.engine.endMap())
		if !n.isRoot() {
			g.append(commaNewLine)
		}
	}()
	// Get the map keys so we can sort them and find the longest key to align
	// the values.
	reflectKeys := mapValue.MapKeys()
	fields := []string{}
	originalKeys := make(map[string]reflect.Value)
	longest := 0
	for i, k := range reflectKeys {
		field := fmt.Sprint(k)
		// Check if a static mapping of this key has been specified.
		if len(g.fieldMappings[field]) > 0 {
			// Use static mapping to get the field name.
			field = g.fieldMappings[field]
		} else {
			// Generate the field name from the key.
			field = g.engine.formatKey(field)
		}
		// Build the slice of generated keys, so we can sort them.
		fields = append(fields, field)
		if len(field) > longest {
			longest = len(field)
		}
		// Remember the original keys. We will later need them.
		originalKeys[field] = reflectKeys[i]
	}
	// Sort the field names.
	sort.Strings(fields)
	for _, field := range fields {
		// Output the map key plus the whitespace to align the values.
		g.append(g.engine.indent(n.getIndentation()+1), fmt.Sprintf("%s: ", field), g.engine.align(field, longest))
		// Get the value and process it recursively.
		value := mapValue.MapIndex(originalKeys[field])
		// Make sure it is not a pointer or interface but actual value.
		if value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
			value = value.Elem()
		}
		childKey := fmt.Sprint(originalKeys[field])
		switch value.Kind() {
		case reflect.Slice:
			g.processSlice(n.createMapChild(childKey, arrayNode), value)
		case reflect.Map:
			g.processMap(n.createMapChild(childKey, mapNode), value)
		default:
			g.processPrimitive(n.createMapChild(childKey, leaf), value)
		}
	}
}

// Internal implementation processing primitive values.
func (g *Generator) processPrimitive(n *node, value reflect.Value) {
	// Make sure it is not a pointer or interface but the actual value.
	if value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		value = value.Elem()
	}
	if !n.isParentMap() {
		g.append(g.engine.indent(n.getIndentation()))
	}
	// Use the engine to output the value.
	g.append(g.engine.formatPrimitive(value), commaNewLine)
}

// Internal function appending data to the output buffer.
func (g *Generator) append(s ...string) {
	for i := range s {
		g.builder.WriteString(s[i])
	}
}
