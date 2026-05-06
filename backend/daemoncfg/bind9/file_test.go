package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the file clause is formatted correctly when file name is quoted.
func TestFileFormatQuotedName(t *testing.T) {
	file := &File{
		Name: &String{
			Quoted: storkutil.Ptr("/var/lib/log/default"),
		},
		Switches: []String{
			{
				Unquoted: storkutil.Ptr("versions"),
			},
			{
				Quoted: storkutil.Ptr("3"),
			},
			{
				Unquoted: storkutil.Ptr("size"),
			},
			{
				Quoted: storkutil.Ptr("10M"),
			},
		},
	}
	output := file.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `file "/var/lib/log/default" versions "3" size "10M";`, output)
}

// Test that the file clause is formatted correctly when file name is unquoted.
// The output file name is quoted despite the original value being unquoted.
func TestFileFormatUnquotedName(t *testing.T) {
	file := &File{
		Name: &String{
			Unquoted: storkutil.Ptr("/var/lib/log/default"),
		},
	}
	output := file.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `file "/var/lib/log/default";`, output)
}
