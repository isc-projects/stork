package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test checking if the statement contains no-parse directives.
func TestStatementHasNoParseGlobal(t *testing.T) {
	statement := &Statement{
		NoParse: &NoParse{},
	}
	require.True(t, statement.HasNoParse())
}

// Test checking if the statement does not contain no-parse directives.
func TestStatementHasNoParseNone(t *testing.T) {
	statement := &Statement{}
	require.False(t, statement.HasNoParse())
}

// Test checking if the statement contains no-parse directives in the zone.
func TestStatementHasNoParseZone(t *testing.T) {
	statement := &Statement{
		Zone: &Zone{
			Clauses: []*ZoneClause{
				{NoParse: &NoParse{}},
			},
		},
	}
	require.True(t, statement.HasNoParse())
}

// Test checking if the statement contains no-parse directives in the view.
func TestStatementHasNoParseView(t *testing.T) {
	statement := &Statement{
		View: &View{
			Clauses: []*ViewClause{
				{NoParse: &NoParse{}},
			},
		},
	}
	require.True(t, statement.HasNoParse())
}

// Test checking if the statement contains no-parse directives in the options.
func TestStatementHasNoParseOptions(t *testing.T) {
	statement := &Statement{
		Options: &Options{
			Clauses: []*OptionClause{
				{NoParse: &NoParse{}},
			},
		},
	}
	require.True(t, statement.HasNoParse())
}
