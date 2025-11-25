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
}

// Instantiates a new formatter builder with the default indent pattern (single tab).
func newFormatterStringBuilder() *formatterStringBuilder {
	return &formatterStringBuilder{
		builder:       strings.Builder{},
		indentPattern: "\t",
	}
}

// Returns the string representation of the built text.
func (b *formatterStringBuilder) getString() string {
	return b.builder.String()
}

// Writes the indentation to the builder. The level specifies the indentation level.
func (b *formatterStringBuilder) writeIndent(level int) {
	b.builder.WriteString(strings.Repeat(b.indentPattern, level))
}

// Writes a new line to the builder.
func (b *formatterStringBuilder) writeNewLine() {
	b.builder.WriteString("\n")
}

// Writes the specified string to the builder.
func (b *formatterStringBuilder) write(s string) {
	b.builder.WriteString(s)
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

	require.Contains(t, `
	foo foo-option {
		bar bar-option {
			bar3;
			bar4;
		} baz;
	} { foo2 foo3 } cab cab-option {
		abc;
	} woo {
		wook
		wookie;
	};

`, formattedText)
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
