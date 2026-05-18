package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the directory statement is formatted correctly.
func TestDirectoryFormat(t *testing.T) {
	directory := &Directory{
		Path: "/var/lib/bind",
	}
	output := directory.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `directory "/var/lib/bind";`, output)
}

// Test that the directory statement is formatted correctly when the path is empty.
func TestDirectoryFormatEmptyPath(t *testing.T) {
	directory := &Directory{}
	output := directory.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `directory "";`, output)
}
