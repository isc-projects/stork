package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the suboption with switches and generic contents
// is formatted correctly.
func TestSuboptionFormat(t *testing.T) {
	suboption := &Suboption{
		Identifier: "test-suboption",
		Switches: []OptionSwitch{
			{
				StringSwitch: storkutil.Ptr("string"),
			},
			{
				IdentSwitch: storkutil.Ptr("ident"),
			},
		},
		Contents: &GenericClauseContents{
			tokens: []string{"token1", ";", "token2", ";"},
		},
	}
	output := suboption.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `test-suboption "string" ident {
		token1;
		token2;
	};`, output)
}

// Test that the suboption with generic contents is formatted correctly.
func TestSuboptionFormatNoSwitches(t *testing.T) {
	suboption := &Suboption{
		Identifier: "test-suboption",
		Contents: &GenericClauseContents{
			tokens: []string{"token1", ";", "token2", ";"},
		},
	}
	output := suboption.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `test-suboption {
		token1;
		token2;
	};`, output)
}

// Test that the suboption with switches is formatted correctly.
func TestSuboptionFormatNoContents(t *testing.T) {
	suboption := &Suboption{
		Identifier: "test-suboption",
		Switches: []OptionSwitch{
			{
				StringSwitch: storkutil.Ptr("string"),
			},
			{
				IdentSwitch: storkutil.Ptr("ident"),
			},
		},
	}
	output := suboption.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `test-suboption "string" ident;`, output)
}
