package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test getting the first inet clause from the controls statement.
func TestGetFirstInetClause(t *testing.T) {
	controls := &Controls{
		Clauses: []*ControlClause{
			{UnixClause: &UnixClause{}},
			{InetClause: &InetClause{}},
			{InetClause: &InetClause{}},
		},
	}
	inetClause := controls.GetFirstInetClause()
	require.NotNil(t, inetClause)
	require.Equal(t, controls.Clauses[1].InetClause, inetClause)
}

// Test that nil is returned when getting the first inet clause from the controls statement
// when the inet clause does not exist.
func TestGetFirstInetClauseNone(t *testing.T) {
	controls := &Controls{}
	require.Nil(t, controls.GetFirstInetClause())
}
