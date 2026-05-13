package oidc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that CommaSeparatedStrings are correctly unmarshaled.
func TestUnmarshalFlag(t *testing.T) {
	// Arrange
	var flags CommaSeparatedStrings

	// Act + Assert
	err := flags.UnmarshalFlag("group1, group2")
	require.NoError(t, err)
	require.NotNil(t, flags)
	require.Len(t, flags, 2)
	require.Equal(t, "group1", flags[0])
	require.Equal(t, "group2", flags[1])

	flags = CommaSeparatedStrings{}
	err = flags.UnmarshalFlag("")
	require.NoError(t, err)
	require.NotNil(t, flags)
	require.Len(t, flags, 0)

	flags = CommaSeparatedStrings{}
	err = flags.UnmarshalFlag(",")
	require.NoError(t, err)
	require.NotNil(t, flags)
	require.Len(t, flags, 0)

	flags = CommaSeparatedStrings{}
	err = flags.UnmarshalFlag(" , group2 ")
	require.NoError(t, err)
	require.NotNil(t, flags)
	require.Len(t, flags, 1)
	require.Equal(t, "group2", flags[0])

	flags = CommaSeparatedStrings{}
	err = flags.UnmarshalFlag(" group1, group2\\,bar , group\\\\3, \\")
	require.NoError(t, err)
	require.NotNil(t, flags)
	require.Len(t, flags, 4)
	require.Equal(t, "group1", flags[0])
	require.Equal(t, "group2,bar", flags[1])
	require.Equal(t, "group\\\\3", flags[2])
	require.Equal(t, "\\", flags[3])
}
