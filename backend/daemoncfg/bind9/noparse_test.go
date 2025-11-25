package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoParseFormat(t *testing.T) {
	noParse := &NoParse{
		NoParseScope: &NoParseScope{
			Contents: RawContents(`type forward; allow-transfer port 853 { any; };`),
		},
	}
	output := noParse.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `//@stork:no-parse:scope
	type forward;
	allow-transfer port 853 { any; };
	//@stork:no-parse:end`, output)
}

func TestNoParseGlobalFormat(t *testing.T) {
	noParse := &NoParse{
		NoParseGlobal: &NoParseGlobal{
			Contents: RawContents(`type forward; allow-transfer port 853 { any; };`),
		},
	}
	output := noParse.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `//@stork:no-parse:global
	type forward; allow-transfer port 853 { any; };`, output)
}

// Test that serializing a no-parse statement with nil values does not panic.
func TestNoParseFormatNilValues(t *testing.T) {
	noParse := &NoParse{}
	require.NotPanics(t, func() { noParse.getFormattedOutput(nil) })
}
