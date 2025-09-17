package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the unix clause is formatted correctly.
func TestUnixClauseFormat(t *testing.T) {
	unix := &UnixClause{
		Path:  "/var/run/rndc.sock",
		Perm:  0o600,
		Owner: 25,
		Group: 26,
		Keys: &Keys{
			KeyNames: []string{"rndc-key"},
		},
		ReadOnly: storkutil.Ptr(Boolean(true)),
	}
	output := unix.getFormattedOutput(nil)
	require.NotNil(t, output)
	cfgEq(t, `unix "/var/run/rndc.sock" perm 0600 owner 25 group 26 keys { "rndc-key"; } read-only true;`, output)
}

// Test that serializing a unix clause with nil values does not panic.
func TestUnixClauseFormatNilValues(t *testing.T) {
	unix := &UnixClause{}
	require.NotPanics(t, func() { unix.getFormattedOutput(nil) })
}
