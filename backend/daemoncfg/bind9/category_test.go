package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the category statement is formatted correctly when the name and
// the channels are quoted.
func TestCategoryFormatQuoted(t *testing.T) {
	category := &Category{
		Name: String{
			Quoted: storkutil.Ptr("xfer-out"),
		},
		Channels: []String{
			{
				Quoted: storkutil.Ptr("transfers"),
			},
			{
				Quoted: storkutil.Ptr("slog"),
			},
		},
	}
	output := category.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `category "xfer-out" {
		"transfers";
		"slog";
	};`, output)
}

// Test that the category statement is formatted correctly when the name and
// the channels are unquoted.
func TestCategoryFormatUnquoted(t *testing.T) {
	category := &Category{
		Name: String{
			Unquoted: storkutil.Ptr("xfer-out"),
		},
		Channels: []String{
			{
				Unquoted: storkutil.Ptr("transfers"),
			},
		},
	}
	output := category.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `category xfer-out {
		transfers;
	};`, output)
}
