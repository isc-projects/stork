package bind9config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var _ formatterBuilder = (*formatterStringBuilder)(nil)

// formatterStringBuilder implements the formatterBuilder interface. It exposes
// functions for writing out the serialized BIND 9 configuration into the
// string. This formatter builder is used in testing.
type formatterStringBuilder struct {
	builder       strings.Builder
	indentPattern string
	lastNewLine   bool
}

// Instantiates a new formatter builder with the default indent pattern (single tab).
func newFormatterStringBuilder() *formatterStringBuilder {
	return &formatterStringBuilder{
		builder:       strings.Builder{},
		indentPattern: "\t",
		lastNewLine:   false,
	}
}

// Returns the string representation of the built text.
func (b *formatterStringBuilder) getString() string {
	return b.builder.String()
}

// Writes the indentation to the builder. The level specifies the indentation level.
func (b *formatterStringBuilder) writeIndent(level int) {
	b.builder.WriteString(strings.Repeat(b.indentPattern, level))
	b.lastNewLine = false
}

// Writes a new line to the builder.
func (b *formatterStringBuilder) writeNewLine() {
	b.builder.WriteString("\n")
	b.lastNewLine = true
}

// Writes the specified string to the builder.
func (b *formatterStringBuilder) write(s string) {
	b.builder.WriteString(s)
	b.lastNewLine = false
}

// Returns true if the last call was to write a new line.
func (b *formatterStringBuilder) isLastNewLine() bool {
	return b.lastNewLine
}

// Tests that the formatter outputs correct contents for a combination
// of clauses, scopes, and tokens.
func TestFormatterClause(t *testing.T) {
	formatter := newFormatter(1)
	fooClause := newFormatterClause("foo", "foo-option")
	fooScope := fooClause.addScope()

	barClause := newFormatterClause("bar", "bar-option")
	fooScope.add(barClause)
	barScope := barClause.addScope()
	barScope.add(newFormatterClause("bar3"))
	barScope.add(newFormatterClause("bar4"))

	barClause.add(newFormatterToken("baz"))

	fooScope2 := fooClause.addScope()
	fooScope2.add(newFormatterToken("foo2"))
	fooScope2.add(newFormatterToken("foo3"))

	cabClause := newFormatterClause("cab", "cab-option")
	cabScope := cabClause.addScope()
	cabScope.add(newFormatterClause("abc"))
	fooClause.add(cabClause)

	wooClause := newFormatterClause("woo")
	wooScope := wooClause.addScope()
	wooScope.add(newFormatterToken("wook"))
	wooSub := newFormatterClause()
	wooScope.add(wooSub)
	wooSub.add(newFormatterToken("wookie"))
	fooClause.add(wooClause)

	formatter.addClause(fooClause)

	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)

	require.Equal(t, `foo foo-option {
		bar bar-option {
			bar3;
			bar4;
		} baz;
	} { foo2 foo3 } cab cab-option {
		abc;
	} woo {
		wook wookie;
	};`, strings.TrimSpace(formattedText))
}

func TestFormatterScopeNoParseBetweenTokens(t *testing.T) {
	formatter := newFormatter(0)
	scope := newFormatterScope()
	scope.add(newFormatterToken("foo"))
	scope.add(newFormatterLines("//@stork:no-parse:scope"))
	scope.add(newFormatterToken("bar"))
	scope.add(newFormatterLines("//@stork:no-parse:end"))
	formatter.addClause(scope)
	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)
	require.Equal(t, `{
	foo
//@stork:no-parse:scope
	bar
//@stork:no-parse:end
}`,
		strings.TrimSpace(formattedText))
}

// This test verifies that the @stork:no-parse:scope directive can be placed
// between two clauses in the scope. The output should have proper indentation.
func TestFormatterScopeNoParseBetweenClauses(t *testing.T) {
	formatter := newFormatter(0)
	scope := newFormatterScope()
	scope.add(newFormatterClause("foo"))
	scope.add(newFormatterLines("//@stork:no-parse:scope"))
	scope.add(newFormatterClause("bar"))
	scope.add(newFormatterLines("//@stork:no-parse:end"))
	formatter.addClause(scope)
	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)
	require.Equal(t, `{
	foo;
//@stork:no-parse:scope
	bar;
//@stork:no-parse:end
}`, strings.TrimSpace(formattedText))
}

// This test verifies that the @stork:no-parse:scope directive can be placed
// on all clauses in the scope. The output should have proper indentation.
func TestFormatterScopeNoParseOnAllClauses(t *testing.T) {
	formatter := newFormatter(0)
	scope := newFormatterScope()
	scope.add(newFormatterLines("//@stork:no-parse:scope"))
	scope.add(newFormatterClause("foo"))
	scope.add(newFormatterClause("bar"))
	scope.add(newFormatterLines("//@stork:no-parse:end"))
	formatter.addClause(scope)
	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)
	require.Equal(t, `{
//@stork:no-parse:scope
	foo;
	bar;
//@stork:no-parse:end
}`, strings.TrimSpace(formattedText))
}

// This test verifies that the @stork:no-parse:scope directive can be placed
// between a token and a clause in the scope. The output should have proper
// indentation.
func TestFormatterScopeNoParseClausesAndTokens(t *testing.T) {
	formatter := newFormatter(0)
	scope := newFormatterScope()
	scope.add(newFormatterToken("foo"))
	scope.add(newFormatterLines("//@stork:no-parse:scope"))
	scope.add(newFormatterToken("bar"))
	scope.add(newFormatterLines("//@stork:no-parse:end"))
	scope.add(newFormatterClause("baz"))
	formatter.addClause(scope)
	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)
	require.Equal(t, `{
	foo
//@stork:no-parse:scope
	bar
//@stork:no-parse:end
	baz;
}`, strings.TrimSpace(formattedText))
}

// This test verifies that the @stork:no-parse:scope directive can be placed
// in the scope and when the last element is the clause it should be
// properly indented.
func TestFormatterScopeNoParseLastClause(t *testing.T) {
	formatter := newFormatter(0)
	scope := newFormatterScope()
	scope.add(newFormatterToken("foo"))
	scope.add(newFormatterLines("//@stork:no-parse:scope"))
	scope.add(newFormatterClause("bar"))
	scope.add(newFormatterLines("//@stork:no-parse:end"))
	scope.add(newFormatterClause("baz"))
	formatter.addClause(scope)
	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)
	require.Equal(t, `{
	foo
//@stork:no-parse:scope
	bar;
//@stork:no-parse:end
	baz;
}`, strings.TrimSpace(formattedText))
}

// This test verifies that the tokens and clauses are indented and each
// is in the new line when the last element is a clause.
func TestFormatterScopeClausesAndTokensLastClause(t *testing.T) {
	formatter := newFormatter(0)
	scope := newFormatterScope()
	scope.add(newFormatterToken("foo"))
	scope.add(newFormatterToken("bar"))
	scope.add(newFormatterClause("baz"))
	scope.add(newFormatterToken("qux"))
	scope.add(newFormatterClause("quux"))
	formatter.addClause(scope)
	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)
	require.Equal(t, `{
	foo bar baz;
	qux quux;
}`, strings.TrimSpace(formattedText))
}

func TestFormatterScopeClausesAndTokensLastToken(t *testing.T) {
	formatter := newFormatter(0)
	scope := newFormatterScope()
	scope.add(newFormatterClause("baz"))
	scope.add(newFormatterToken("qux"))
	scope.add(newFormatterToken("quux"))
	formatter.addClause(scope)
	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)
	require.Equal(t, `{
	baz;
	qux quux
}`, strings.TrimSpace(formattedText))
}

// This test verifies that the tokens in the scope are placed inline when
// there are no clauses in the scope.
func TestFormatterScopeOnlyTokens(t *testing.T) {
	formatter := newFormatter(0)
	scope := newFormatterScope()
	scope.add(newFormatterToken("foo"))
	scope.add(newFormatterToken("bar"))
	formatter.addClause(scope)
	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)
	require.Equal(t, `{ foo bar }`, strings.TrimSpace(formattedText))
}

// Test that the formatter outputs correct contents for a sequence of lines.
func TestFormatterLines(t *testing.T) {
	formatter := newFormatter(0)
	lines := newFormatterLines(`
		line1
		line2
		line3
	`)
	require.NotNil(t, lines)
	formatter.addClause(lines)
	var builder strings.Builder
	formatter.getFormattedTextFunc(func(text string) {
		builder.WriteString(text)
		builder.WriteString("\n")
	})
	formattedText := builder.String()
	require.NotEmpty(t, formattedText)
	require.Equal(t, "\t\tline1\n\t\tline2\n\t\tline3\n\t\n\n", formattedText)
}

// Test that an error is returned when the sequence of lines is too long.
func TestFormatterLinesError(t *testing.T) {
	formatter := newFormatter(0)
	lines := newFormatterLines(strings.Repeat("a", maxFormatterLinesBufferSize+1))
	require.NotNil(t, lines)
	formatter.addClause(lines)
	err := formatter.getFormattedTextFunc(func(text string) {})
	require.ErrorContains(t, err, "encountered BIND 9 configuration line exceeding the maximum buffer size: 8192")
}

var _ formattedElement = (*testElement)(nil)

// A structure implementing the formattedElement interface.
type testElement struct {
	name string
}

// Returns the formatted output of the test element.
func (t *testElement) getFormattedOutput(filter *Filter) formatterOutput {
	return newFormatterToken(t.name)
}

// A structure not implementing the formattedElement interface.
type testElementNotImplemented struct{}

// A structure holding a set of fields with different filter tags.
// The first field does not implement the formattedElement interface.
type testStruct struct {
	O *testElementNotImplemented
	A *testElement `filter:"As,All"`
	B *testElement `filter:"Bs,All"`
	C *testElement
}

// Test that getFormatterClauseFromStruct returns the correct formatted clause
// according to the filter.
func TestGetFormattedClauseFromStructByFilter(t *testing.T) {
	// Create a struct with first field pointing to an instance not
	// implementing the formattedElement interface.
	testStruct := &testStruct{
		O: &testElementNotImplemented{},
		A: &testElement{"A"},
		B: &testElement{"B"},
		C: &testElement{"C"},
	}

	t.Run("no filter", func(t *testing.T) {
		// The filter is nil, so all fields implementing the formattedElement interface
		// should be taken into account. The first field implementing the formattedElement
		// interface should be returned.
		output := getFormatterClauseFromStruct(testStruct, nil)
		require.NotNil(t, output)
		requireConfigEq(t, "A", output)
	})

	t.Run("filter Bs", func(t *testing.T) {
		// The filter is "Bs", so only the field implementing the formattedElement
		// interface with the filter tag "Bs" should be taken into account.
		output := getFormatterClauseFromStruct(testStruct, NewFilter("Bs"))
		require.NotNil(t, output)
		requireConfigEq(t, "B", output)
	})

	t.Run("filter Cs", func(t *testing.T) {
		// The filter is "Cs", so only the field implementing the formattedElement
		// interface with the filter tag "Cs" should be taken into account.
		output := getFormatterClauseFromStruct(testStruct, NewFilter("Cs"))
		require.NotNil(t, output)
		requireConfigEq(t, "C", output)
	})

	t.Run("filter All", func(t *testing.T) {
		// The filter is "All", so all fields implementing the formattedElement
		// interface should be taken into account. The first field implementing the
		// formattedElement interface and having the All tag should be returned.
		output := getFormatterClauseFromStruct(testStruct, NewFilter("All"))
		require.NotNil(t, output)
		requireConfigEq(t, "A", output)
	})
}

// Test that getFormatterClauseFromStruct returns nil if its all fields are nil.
func TestGetFormattedClauseFromStructAllNil(t *testing.T) {
	testStruct := &testStruct{}

	output := getFormatterClauseFromStruct(testStruct, nil)
	require.Nil(t, output)
}

// Test that getFormatterClauseFromStruct returns nil if the specified struct is nil.
func TestGetFormattedClauseFromStructNilStruct(t *testing.T) {
	var testStruct *testStruct

	output := getFormatterClauseFromStruct(testStruct, nil)
	require.Nil(t, output)
}

// Test that getFormatterClauseFromStruct returns nil if the specified value is nil.
func TestGetFormattedClauseFromStructNilValue(t *testing.T) {
	output := getFormatterClauseFromStruct(nil, nil)
	require.Nil(t, output)
}

// Test that getFormatterClauseFromStruct returns nil if the specified value is not a struct.
func TestGetFormattedClauseFromStructNotStruct(t *testing.T) {
	output := getFormatterClauseFromStruct(1, nil)
	require.Nil(t, output)
}
