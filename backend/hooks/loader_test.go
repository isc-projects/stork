package hooks

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork"
)

// Test that the function to load all hooks returns an error if the
// directory doesn't exist.
func TestLoadAllHookCalloutsReturnErrorForInvalidDirectory(t *testing.T) {
	// Arrange & Act
	callouts, err := LoadAllHookCallouts("", "/non/exist/directory")

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "cannot find plugin paths")
	require.ErrorContains(t, err, "no such file or directory")
}

// Test that the function to load all hooks returns an error if the directory
// contains a non-plugin file.
func TestLoadAllHookCalloutsReturnErrorForNonPluginFile(t *testing.T) {
	// Arrange & Act
	callouts, err := LoadAllHookCallouts("", "templates")

	// Assert
	require.Nil(t, callouts)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot open hook library")
	require.ErrorContains(t, err, "invalid ELF header")
}

// Test that the extract callout function returns an error if the Version
// function is missing in the hook.
func TestExtractCalloutMissingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(nil, errors.New("symbol not found")))

	// Act
	callout, err := extractCallout(library, "foo")

	// Assert
	require.Nil(t, callout)
	require.ErrorContains(t, err, "symbol not found")
	require.ErrorContains(t, err, "lookup for symbol: Version")
}

// Test that the extract callout function returns an error if the Version
// function has an invalid signature.
func TestExtractCalloutInvalidVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(invalidSignature, nil))

	// Act
	callout, err := extractCallout(library, "foo")

	// Assert
	require.Nil(t, callout)
	require.ErrorContains(t, err, "symbol Version has unexpected signature")
}

// Test that the extract callout function returns an error if the hook is
// dedicated for another application.
func TestExtractCalloutNonMatchingApplication(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(validVersion("bar", ""), nil))

	// Act
	callout, err := extractCallout(library, "foo")

	// Assert
	require.Nil(t, callout)
	require.ErrorContains(t, err, "hook library dedicated for another program: bar")
}

// Test that the extract callout function returns an error if the hook is
// dedicated for another Stork version.
func TestExtractCalloutNonMatchingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(validVersion("foo", "non.matching.version"), nil))

	// Act
	callout, err := extractCallout(library, "foo")

	// Assert
	require.Nil(t, callout)
	require.ErrorContains(t, err, "incompatible hook version: non.matching.version")
}

// Test that the extract callout function returns an error if the Load
// function is missing in the hook.
func TestExtractCalloutMissingLoad(t *testing.T) {
	// Arrange
	plugin := newPluginMock(nil, errors.New("symbol not found"))
	plugin.addLookupVersion(validVersion("foo", stork.Version))
	library := newLibraryManager("", plugin)

	// Act
	callout, err := extractCallout(library, "foo")

	// Assert
	require.Nil(t, callout)
	require.ErrorContains(t, err, "symbol not found")
	require.ErrorContains(t, err, "lookup for symbol: Load")
}

// Test that the extract callout function returns an error if the Load
// function has an invalid signature.
func TestExtractCalloutInvalidLoad(t *testing.T) {
	// Arrange
	plugin := newPluginMock(invalidSignature, nil)
	plugin.addLookupVersion(validVersion("foo", stork.Version))
	library := newLibraryManager("", plugin)

	// Act
	callout, err := extractCallout(library, "foo")

	// Assert
	require.Nil(t, callout)
	require.ErrorContains(t, err, "symbol Load has unexpected signature")
}

// Test that the extract callout function returns an error if the Load
// function fails.
func TestExtractCalloutLoadFails(t *testing.T) {
	// Arrange
	plugin := newPluginMock(validLoad("", errors.New("error in load")), nil)
	plugin.addLookupVersion(validVersion("foo", stork.Version))
	library := newLibraryManager("", plugin)

	// Act
	callout, err := extractCallout(library, "foo")

	// Assert
	require.Nil(t, callout)
	require.ErrorContains(t, err, "error in load")
}

// Test that the extract callout function return a proper output on success.
func TestExtractCallout(t *testing.T) {
	// Arrange
	plugin := newPluginMock(validLoad("bar", nil), nil)
	plugin.addLookupVersion(validVersion("foo", stork.Version))
	library := newLibraryManager("", plugin)

	// Act
	callout, err := extractCallout(library, "foo")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, callout)
}
