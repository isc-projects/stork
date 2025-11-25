package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the option statement is formatted correctly.
func TestOptionFormat(t *testing.T) {
	option := &Option{
		Identifier: "test-option",
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
		Suboptions: []Suboption{
			{
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
					tokens: []string{"token3", ";", "token4", ";"},
				},
			},
		},
	}
	output := option.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `test-option "string" ident {
		token1;
		token2;
	} test-suboption "string" ident {
		token3;
		token4;
	};`, output)
}

// Test that serializing an option statement with nil values does not panic.
func TestOptionFormatNilValues(t *testing.T) {
	option := &Option{}
	require.NotPanics(t, func() { option.getFormattedOutput(nil) })
}

// Test that serializing a suboption statement with nil values does not panic.
func TestSuboptionFormatNilValues(t *testing.T) {
	suboption := &Suboption{}
	require.NotPanics(t, func() { suboption.getFormattedOutput(nil) })
}
