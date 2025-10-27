package bind9config

import (
	"fmt"
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
	fmt.Println(formattedText)
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
