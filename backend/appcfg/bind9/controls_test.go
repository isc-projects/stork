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

// Test that the controls statement is formatted correctly.
func TestControlsFormat(t *testing.T) {
	controls := &Controls{
		Clauses: []*ControlClause{
			{
				UnixClause: &UnixClause{
					Path:  "/var/run/named.sock",
					Perm:  0o640,
					Owner: 1000,
					Group: 1000,
				},
			},
			{
				InetClause: &InetClause{
					Address: "127.0.0.1",
				},
			},
		},
	}
	output := controls.getFormattedOutput(nil)
	require.NotNil(t, output)
	cfgEq(t, `
		controls {
			unix "/var/run/named.sock" perm 0640 owner 1000 group 1000;
			inet "127.0.0.1";
		};
	`, output)
}

// Test that serializing a controls statement with nil values does not panic.
func TestControlsFormatNilValues(t *testing.T) {
	controls := &Controls{}
	require.NotPanics(t, func() { controls.getFormattedOutput(nil) })
}
