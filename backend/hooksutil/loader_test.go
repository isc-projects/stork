package hooksutil

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork"
)

// Test that the function to load all hooks returns an empty list if the
// directory doesn't exist.
func TestLoadAllHooksReturnEmptyResultForInvalidDirectory(t *testing.T) {
	// Arrange & Act
	callouts, err := LoadAllHooks("", "/non/exist/directory")

	// Assert
	require.Nil(t, callouts)
	require.Error(t, err)
}

// Test that the extract callouts function returns an error if the Version
// function is missing in the hook.
func TestExtractCalloutsMissingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(nil, errors.New("symbol not found")))

	// Act
	callouts, err := extractCallouts(library, "foo")

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "symbol not found")
	require.ErrorContains(t, err, "lookup for symbol: Version")
}

// Test that the extract callouts function returns an error if the Version
// function has an invalid signature.
func TestExtractCalloutsInvalidVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(invalidSignature, nil))

	// Act
	callouts, err := extractCallouts(library, "foo")

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "symbol Version has unexpected signature")
}

// Test that the extract callouts function returns an error if the hook is
// dedicated for another application.
func TestExtractCalloutsNonMatchingApplication(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(validVersion("bar", ""), nil))

	// Act
	callouts, err := extractCallouts(library, "foo")

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "hook library dedicated for another program: bar")
}

// Test that the extract callouts function returns an error if the hook is
// dedicated for another Stork version.
func TestExtractCalloutsNonMatchingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(validVersion("foo", "non.matching.version"), nil))

	// Act
	callouts, err := extractCallouts(library, "foo")

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "incompatible hook version: non.matching.version")
}

// Test that the extract callouts function returns an error if the Load
// function is missing in the hook.
func TestExtractCalloutsMissingLoad(t *testing.T) {
	// Arrange
	plugin := newPluginMock(nil, errors.New("symbol not found"))
	plugin.addLookupOutput("Version", validVersion("foo", stork.Version), nil)
	library := newLibraryManager("", plugin)

	// Act
	callouts, err := extractCallouts(library, "foo")

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "symbol not found")
	require.ErrorContains(t, err, "lookup for symbol: Load")
}

// Test that the extract callouts function returns an error if the Load
// function has an invalid signature.
func TestExtractCalloutsInvalidLoad(t *testing.T) {
	// Arrange
	plugin := newPluginMock(invalidSignature, nil)
	plugin.addLookupOutput("Version", validVersion("foo", stork.Version), nil)
	library := newLibraryManager("", plugin)

	// Act
	callouts, err := extractCallouts(library, "foo")

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "symbol Load has unexpected signature")
}

// Test that the extract callouts function returns an error if the Load
// function fails.
func TestExtractCalloutsLoadFails(t *testing.T) {
	// Arrange
	plugin := newPluginMock(validLoad("", errors.New("error in load")), nil)
	plugin.addLookupOutput("Version", validVersion("foo", stork.Version), nil)
	library := newLibraryManager("", plugin)

	// Act
	callouts, err := extractCallouts(library, "foo")

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "error in load")
}

// Test that the extract callouts function return a proper output on success.
func TestExtractCallouts(t *testing.T) {
	// Arrange
	plugin := newPluginMock(validLoad("bar", nil), nil)
	plugin.addLookupOutput("Version", validVersion("foo", stork.Version), nil)
	library := newLibraryManager("", plugin)

	// Act
	callouts, err := extractCallouts(library, "foo")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, callouts)
}
