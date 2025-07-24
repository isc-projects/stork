package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test getting the inet clause from the controls statement.
func TestGetInetClause(t *testing.T) {
	controls := &Controls{
		Clauses: []*ControlClause{
			{UnixClause: &UnixClause{}},
			{InetClause: &InetClause{}},
		},
	}
	inetClause := controls.GetInetClause()
	require.NotNil(t, inetClause)
	require.Equal(t, controls.Clauses[1].InetClause, inetClause)
}

// Test that nil is returned when getting the inet clause from the controls statement
// when the inet clause does not exist.
func TestGetInetClauseNone(t *testing.T) {
	controls := &Controls{}
	require.Nil(t, controls.GetInetClause())
}
