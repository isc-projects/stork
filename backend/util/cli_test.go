package storkutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the optional string flag is constructed properly.
func TestNewOptionalStringFlag(t *testing.T) {
	// Act
	flag := NewOptionalStringFlag("foo")

	// Assert
	require.NotNil(t, flag)
}

// Test that the optional string flag is represented as string properly.
func TestOptionalStringRepresentation(t *testing.T) {
	// Arrange
	flag := NewOptionalStringFlag("foo")

	// Act
	str := flag.String()

	// Assert
	require.EqualValues(t, "foo", str)
}

// Test that the flag value is set properly.
func TestOptionalStringFlagSet(t *testing.T) {
	// Arrange
	flag := NewOptionalStringFlag("foo")

	// Act
	err := flag.Set("bar")

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "bar", flag.String())
}

// Test that the set method ignores the empty value.
func TestOptionalStringFlagSetEmpty(t *testing.T) {
	// Arrange
	flag := NewOptionalStringFlag("foo")

	// Act
	err := flag.Set("")

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "foo", flag.String())
}

// Test that the set method ignores the boolean true serialized to string.
func TestOptionalStringFlagSetBooleanTrue(t *testing.T) {
	// Arrange
	flag := NewOptionalStringFlag("foo")

	// Act
	err := flag.Set("true")

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "foo", flag.String())
}

// Test that the optional string flag always is boolean.
func TestOptionalStringFlagIsBool(t *testing.T) {
	// Arrange
	flags := []*OptionalStringFlag{
		NewOptionalStringFlag("foo"),
		NewOptionalStringFlag(""),
		NewOptionalStringFlag("true"),
		NewOptionalStringFlag("false"),
		NewOptionalStringFlag("nil"),
		NewOptionalStringFlag("42"),
	}

	// Act
	for _, flag := range flags {
		t.Run(flag.String(), func(t *testing.T) {
			// Assert
			require.True(t, flag.IsBoolFlag())
		})
	}
}
