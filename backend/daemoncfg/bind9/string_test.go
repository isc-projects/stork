package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that quoted value is returned when original value was quoted.
func TestGetOriginalValueQuoted(t *testing.T) {
	string := &String{
		Quoted: storkutil.Ptr("quoted"),
	}
	require.Equal(t, `"quoted"`, string.GetOriginalValue())
}

// Test that unquoted value is returned when original value was unquoted.
func TestGetOriginalValueUnquoted(t *testing.T) {
	string := &String{
		Unquoted: storkutil.Ptr("unquoted"),
	}
	require.Equal(t, "unquoted", string.GetOriginalValue())
}

// Test that empty string is returned when original value is nil.
func TestGetOriginalValueNil(t *testing.T) {
	string := &String{}
	require.Equal(t, "", string.GetOriginalValue())
}

// Test that unquoted value is returned even when original value was quoted.
func TestGetValueQuoted(t *testing.T) {
	string := &String{
		Quoted: storkutil.Ptr("quoted"),
	}
	require.Equal(t, "quoted", string.GetValue())
}

// Test that unquoted value is returned when original value was unquoted.
func TestGetValueUnquoted(t *testing.T) {
	string := &String{
		Unquoted: storkutil.Ptr("unquoted"),
	}
	require.Equal(t, "unquoted", string.GetOriginalValue())
}

// Test that empty string is returned when original value is nil.
func TestGetValueNil(t *testing.T) {
	string := &String{}
	require.Equal(t, "", string.GetOriginalValue())
}
