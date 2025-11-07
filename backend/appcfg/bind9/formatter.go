package bind9config

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	_ formatterOutput  = (*formatterToken)(nil)
	_ formatterOutput  = (*formatterScope)(nil)
	_ formatterOutput  = (*formatterClause)(nil)
	_ formatterBuilder = (*formatterBuilderFunc)(nil)
)

// formattedElement is an interface implemented by all BIND 9
// configuration elements that can be serialized into a
// BIND 9 configuration string representation. The filter
// can be used to select specific parts of the elements to
// be returned, when the elements implement filtering.
type formattedElement interface {
	getFormattedOutput(filter *Filter) formatterOutput
}

// formatterOutput is an interface returned by the configuration
// elements implementing the formattedElement interface. The output
// is used by the formatter to write serialized configuration into
// the formatterBuilder.
type formatterOutput interface {
	// write writes the serialized configuration into the formatterBuilder.
	// the indentLevel is the current indentation level. The inner flag
	// indicates whether or not the output is inside a clause. This is
	// used when one clause holds another clause. In this case, the
	// inner clause does not include the semicolon.
	write(indentLevel int, inner bool, builder formatterBuilder)
}

// The formatter is responsible for serializing BIND 9 configuration into
// a string representation. It contains a list of clauses (which contain
// other clauses, scopes and tokens), added using the addClause function.
// The formatter returns serialized BIND 9 configuration with indentation.
type formatter struct {
	// Holds initial indentation.
	indent int
	// Holds configuration clauses to be output.
	clauses []formatterOutput
}

// Instantiates a new formatter with the specified initial indentation level.
func newFormatter(indent int) *formatter {
	return &formatter{
		indent: indent,
	}
}

// Adds a new clause to the formatter.
func (f *formatter) addClause(clause formatterOutput) {
	if clause != nil {
		f.clauses = append(f.clauses, clause)
	}
}

// Returns the serialized BIND 9 configuration via callbacks. The callback is
// called with each line of the serialized configuration.
func (f *formatter) getFormattedTextFunc(callback func(string)) {
	builder := newFormatterBuilder(callback)
	for _, clause := range f.clauses {
		builder.writeIndent(f.indent)
		clause.write(f.indent, false, builder)
		for i := 0; i < 2; i++ {
			// Ensure one empty line between statements.
			builder.writeNewLine()
		}
	}
}

// formatterToken represents a single token in the BIND 9 configuration.
// It is one of the basic building blocks of the configuration. It
// implements the formatterOutput interface.
type formatterToken struct {
	token string
}

// Instantiates a new formatter token with the specified token value.
func newFormatterToken(token string) *formatterToken {
	return &formatterToken{
		token: token,
	}
}

// Writes the token into the builder.
func (t *formatterToken) write(indent int, inner bool, builder formatterBuilder) {
	builder.write(t.token)
}

// formatterScope represents a scope in the BIND 9 configuration.
// It is used to group statements together. It implements the
// formatterOutput interface. The scope begins and ends with the
// curly braces. The scope typically contains a collection of clauses.
// It typically belongs to a clause.
type formatterScope struct {
	elements []formatterOutput
}

// Instantiates a new formatter scope.
func newFormatterScope() *formatterScope {
	return &formatterScope{
		elements: []formatterOutput{},
	}
}

// Adds a new element to the scope.
func (t *formatterScope) add(element formatterOutput) {
	if element != nil {
		t.elements = append(t.elements, element)
	}
}

// Writes the scope into the builder. The scope contents are surrounded
// with the curly braces. If the element belonging to the scope is a clause,
// it is indented and followed by a new line. Otherwise, it is preceded by
// a space (as a separator from preceding token).
func (t *formatterScope) write(indentLevel int, inner bool, builder formatterBuilder) {
	builder.write("{")
	var isLastClause bool
	if len(t.elements) > 0 {
		if _, ok := t.elements[len(t.elements)-1].(*formatterClause); ok {
			isLastClause = true
		}
	}
	if isLastClause {
		// Add new line before the end of the scope when the last element is a clause.
		// Otherwise, write the contents inline because they can be just tokens or
		// several scopes. Scopes are output inline.
		builder.writeNewLine()
	}

	for _, element := range t.elements {
		if _, ok := element.(*formatterClause); ok || isLastClause {
			// We're dealing with a clause, so let's start with new line
			// and indentation.
			builder.writeIndent(indentLevel + 1)
			element.write(indentLevel+1, false, builder)
			builder.writeNewLine()
		} else {
			// Not a clause. Write inline
			builder.write(" ")
			element.write(indentLevel, false, builder)
		}
	}
	if isLastClause {
		builder.writeIndent(indentLevel)
	} else {
		builder.write(" ")
	}
	// End the scope.
	builder.write("}")
}

// formatterClause represents a clause in the BIND 9 configuration. The
// clause can consist of tokens and/or scopes. It can also embed other
// clauses. It implements the formatterOutput interface. The clause
// is ended with a semicolon if it is not an inner clause. The inner
// clause is embedded with another clause.
type formatterClause struct {
	elements []formatterOutput
}

// Instantiates a new formatter clause with the specified tokens.
// The tokens are optional.
func newFormatterClause(tokens ...string) *formatterClause {
	clause := &formatterClause{
		elements: []formatterOutput{},
	}
	for _, token := range tokens {
		clause.add(newFormatterToken(token))
	}
	return clause
}

// Instantiates a new formatter clause with specified text and formatting
// arguments (similar to fmt.Sprintf).
func newFormatterClausef(clauseText string, args ...any) *formatterClause {
	clauseText = fmt.Sprintf(clauseText, args...)
	tokens := strings.Fields(clauseText)
	clause := newFormatterClause()
	for _, token := range tokens {
		clause.add(newFormatterToken(token))
	}
	return clause
}

// Adds a new element to the clause.
func (c *formatterClause) add(element formatterOutput) {
	if element != nil {
		c.elements = append(c.elements, element)
	}
}

// Adds a new token to the clause.
func (c *formatterClause) addToken(token string) {
	c.elements = append(c.elements, newFormatterToken(token))
}

// Adds a new quoted token to the clause.
func (c *formatterClause) addQuotedToken(token string) {
	c.addToken(fmt.Sprintf(`"%s"`, token))
}

// Adds a new token to the clause with formatting arguments.
func (c *formatterClause) addTokenf(token string, args ...any) {
	c.addToken(fmt.Sprintf(token, args...))
}

// Adds a new scope to the clause.
func (c *formatterClause) addScope() *formatterScope {
	scope := newFormatterScope()
	c.add(scope)
	return scope
}

// Writes the clause into the builder.
func (c *formatterClause) write(indent int, inner bool, builder formatterBuilder) {
	for i, element := range c.elements {
		if el := element; el != nil {
			if i > 0 {
				// Ensure a space between the clause elements. Only the first element
				// is not preceded by a space.
				builder.write(" ")
			}
			el.write(indent, true, builder)
		}
	}
	if !inner {
		// Semicolon is normally written after the clause. It is skipped if the
		// clause is directly embedded in another clause.
		builder.write(";")
	}
}

// formatterBuilder is an interface implemented by the formatterBuilderFunc.
// It exposes functions for writing out the serialized BIND 9 configuration.
// The formatterBuilderFunc stores the serialized parts of the configuration
// until the new line is encountered. In this case, it calls the callback
// function with the serialized configuration. Other implementations are
// used in testing.
type formatterBuilder interface {
	write(s string)
	writeIndent(level int)
	writeNewLine()
}

// formatterBuilderFunc is a concrete implementation of the formatterBuilder
// interface. It stores the serialized parts of the configuration until the
// new line is encountered. In this case, it calls the callback function with
// the serialized configuration.
type formatterBuilderFunc struct {
	builder       strings.Builder
	indentPattern string
	callback      func(string)
}

// Instantiates a new formatter builder with the specified callback.
func newFormatterBuilder(callback func(string)) *formatterBuilderFunc {
	return &formatterBuilderFunc{
		builder:       strings.Builder{},
		indentPattern: "\t",
		callback:      callback,
	}
}

// Writes the indentation to the builder. The level specifies the indentation level.
func (b *formatterBuilderFunc) writeIndent(level int) {
	b.builder.WriteString(strings.Repeat(b.indentPattern, level))
}

// Writes a new line to the builder. It flushes the buffer into the callback,
// so the callback can save the flushed data as a single line.
func (b *formatterBuilderFunc) writeNewLine() {
	b.callback(b.builder.String())
	b.builder.Reset()
}

// Writes the specified string to the builder.
func (b *formatterBuilderFunc) write(s string) {
	b.builder.WriteString(s)
}

// Gets the formatter clause from the specified struct. It is useful in
// cases when the struct holds multiple statements of which one is not
// nil. This function locates the non-nil field that implements the
// formattedElement interface and returns its formatted output. The filter
// specifies the elements to be returned. The elements in the struct can
// be associated with the filter tag. This tag can list one or more filtering
// keywords separated by commas. If the given struct field does not have the
// tag, the field is always returned. If the filter is nil, the field is
// returned regardless of the tag.
func getFormatterClauseFromStruct(s any, filter *Filter) formatterOutput {
	structType := reflect.TypeOf(s)
	if structType == nil || structType.Kind() != reflect.Pointer {
		// If specified nil value or the specified value is not a pointer, return nil.
		return nil
	}
	structType = structType.Elem()
	if structType.Kind() != reflect.Struct {
		// The pointer to a struct type is expected.
		return nil
	}
	structValue := reflect.ValueOf(s)
	if structValue.IsNil() {
		// It is the pointer to a struct but it is nil.
		return nil
	}
	// Everything is fine.
	structValue = structValue.Elem()
	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		t := field.Type()
		if t.Implements(reflect.TypeOf((*formattedElement)(nil)).Elem()) {
			if !field.IsNil() {
				filterTag := structType.Field(i).Tag.Get("filter")
				if filterTag == "" {
					return field.Interface().(formattedElement).getFormattedOutput(filter)
				}
				for _, tag := range strings.Split(filterTag, ",") {
					if filter.IsEnabled(FilterType(tag)) {
						return field.Interface().(formattedElement).getFormattedOutput(filter)
					}
				}
			}
		}
	}
	return nil
}
