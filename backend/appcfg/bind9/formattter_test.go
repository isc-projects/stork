package bind9config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

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

	formattedText := formatter.getFormattedText()
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
