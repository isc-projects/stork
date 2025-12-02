package bind9config

import (
	"bufio"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

var (
	_ formatterOutput  = (*formatterToken)(nil)
	_ formatterOutput  = (*formatterScope)(nil)
	_ formatterOutput  = (*formatterClause)(nil)
	_ formatterOutput  = (*formatterLines)(nil)
	_ formatterBuilder = (*formatterBuilderFunc)(nil)
)

const (
	minFormatterLinesBufferSize = 128
	maxFormatterLinesBufferSize = 8192
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
	write(indentLevel int, inner bool, builder formatterBuilder) error
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
func (f *formatter) getFormattedTextFunc(callback func(string)) error {
	builder := newFormatterBuilder(callback)
	for _, clause := range f.clauses {
		builder.writeIndent(f.indent)
		if err := clause.write(f.indent, false, builder); err != nil {
			return err
		}
		for i := 0; i < 2; i++ {
			// Ensure one empty line between statements.
			builder.writeNewLine()
		}
	}
	return nil
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
func (t *formatterToken) write(indent int, inner bool, builder formatterBuilder) error {
	builder.write(t.token)
	return nil
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
// with the curly braces. If any element belonging to the scope is a clause
// or @stork:no-parse directive, the scope contents are written as a block
// (with new line following the opening brace and preceding the curly brace).
// The @stork:no-parse directive contents start with a new line and end with
// a new line. A clause starts inline with preceding tokens, and it starts with
// a new line and indentation after the opening brace, previous clause and the
// @stork:no-parse directive. A token starts inline with preceding tokens.
func (t *formatterScope) write(indentLevel int, inner bool, builder formatterBuilder) error {
	builder.write("{")
	// This variable will indicate if the scope contents are written as a
	// block (with new lines before each element) or inline. It is a block
	// if any configuration element is a clause (terminated with a semicolon)
	// or a sequence of lines (i.e., @stork:no-parse:scope directive).
	var isBlock bool
	for _, element := range t.elements {
		switch element.(type) {
		case *formatterClause, *formatterLines:
			isBlock = true
		}
	}
	if isBlock {
		// Begin the scope with a new line because it is a block.
		builder.writeNewLine()
	}
	for _, element := range t.elements {
		switch element.(type) {
		case *formatterLines:
			if !builder.isLastNewLine() {
				// Write new line before the @stork:no-parse directive.
				// There is no indentation.
				builder.writeNewLine()
			}
			if err := element.write(0, false, builder); err != nil {
				return err
			}
			// The @stork:no-parse directive contents must be followed by
			// a new line.
			builder.writeNewLine()
		case *formatterClause:
			if builder.isLastNewLine() {
				// If the there was a new line after the previous element
				// make sure that the next clause is indented.
				builder.writeIndent(indentLevel + 1)
			} else {
				// There is no new line so the previous element was a token
				// or a scope. Add a space and put it inline.
				builder.write(" ")
			}
			if err := element.write(indentLevel+1, false, builder); err != nil {
				return err
			}
			// The clause ends with a new line.
			builder.writeNewLine()
		default:
			if builder.isLastNewLine() {
				// If there was a new line (e.g., after a previous clause),
				// make sure that the next element is indented.
				builder.writeIndent(indentLevel + 1)
			} else {
				// Write another element inline instead.
				builder.write(" ")
			}
			if err := element.write(indentLevel, false, builder); err != nil {
				return err
			}
		}
	}
	if isBlock {
		// If it is a block, the end of the scope must be indented.
		if !builder.isLastNewLine() {
			builder.writeNewLine()
		}
		builder.writeIndent(indentLevel)
	} else {
		// If it is not a block, add a space before the closing brace.
		builder.write(" ")
	}
	// End the scope.
	builder.write("}")
	return nil
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
func (c *formatterClause) write(indent int, inner bool, builder formatterBuilder) error {
	for i, element := range c.elements {
		if el := element; el != nil {
			if i > 0 {
				// Ensure a space between the clause elements. Only the first element
				// is not preceded by a space.
				builder.write(" ")
			}
			if err := el.write(indent, true, builder); err != nil {
				return err
			}
		}
	}
	if !inner {
		// Semicolon is normally written after the clause. It is skipped if the
		// clause is directly embedded in another clause.
		builder.write(";")
	}
	return nil
}

// formatterLines represents a sequence of lines in the BIND 9 configuration.
// It is used to represent the unparsed contents between the @stork:no-parse:scope
// and @stork:no-parse:end directives, or after the @stork:no-parse:global directive.
// It implements the formatterOutput interface.
type formatterLines struct {
	contents string
}

// Instantiates a new formatterLines with the specified contents. It splits the provided
// contents into lines and removes the empty lines at the beginning and the end.
func newFormatterLines(contents string) *formatterLines {
	return &formatterLines{
		contents: contents,
	}
}

// Writes the lines into the builder. It ignores the indentation because the indentation
// is a part of the line contents.
func (l *formatterLines) write(indent int, inner bool, builder formatterBuilder) error {
	var lines []string
	// By default, the scanner splits the input into lines.
	scanner := bufio.NewScanner(strings.NewReader(l.contents))
	// Set the minimum and maximum buffer sizes for each line.
	scanner.Buffer(make([]byte, 0, minFormatterLinesBufferSize), maxFormatterLinesBufferSize)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return errors.Wrapf(err, "encountered BIND 9 configuration line exceeding the maximum buffer size: %d", maxFormatterLinesBufferSize)
		}
		return errors.Wrap(err, "failed to format BIND 9 configuration file")
	}
	for i, content := range lines {
		builder.write(content)
		if i < len(lines)-1 {
			builder.writeNewLine()
		}
	}
	return nil
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
	isLastNewLine() bool
}

// formatterBuilderFunc is a concrete implementation of the formatterBuilder
// interface. It stores the serialized parts of the configuration until the
// new line is encountered. In this case, it calls the callback function with
// the serialized configuration.
type formatterBuilderFunc struct {
	builder       strings.Builder
	indentPattern string
	callback      func(string)
	lastNewLine   bool
}

// Instantiates a new formatter builder with the specified callback.
func newFormatterBuilder(callback func(string)) *formatterBuilderFunc {
	return &formatterBuilderFunc{
		builder:       strings.Builder{},
		indentPattern: "\t",
		callback:      callback,
		lastNewLine:   false,
	}
}

// Writes the indentation to the builder. The level specifies the indentation level.
func (b *formatterBuilderFunc) writeIndent(level int) {
	b.builder.WriteString(strings.Repeat(b.indentPattern, level))
	b.lastNewLine = false
}

// Writes a new line to the builder. It flushes the buffer into the callback,
// so the callback can save the flushed data as a single line.
func (b *formatterBuilderFunc) writeNewLine() {
	b.callback(b.builder.String())
	b.builder.Reset()
	b.lastNewLine = true
}

// Writes the specified string to the builder.
func (b *formatterBuilderFunc) write(s string) {
	b.builder.WriteString(s)
	b.lastNewLine = false
}

// Returns true if the last call was to write a new line.
func (b *formatterBuilderFunc) isLastNewLine() bool {
	return b.lastNewLine
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
