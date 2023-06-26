package hooksutil

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork"
)

// Test that the function to load all hooks returns an error if the
// directory doesn't exist.
func TestLoadAllHooksReturnErrorForInvalidDirectory(t *testing.T) {
	// Arrange & Act
	calloutCarriers, err := LoadAllHooks("", "/non/exist/directory")

	// Assert
	require.Nil(t, calloutCarriers)
	require.ErrorContains(t, err, "cannot find plugin paths")
	require.ErrorContains(t, err, "no such file or directory")
}

// Test that the function to load all hooks returns an error if the directory
// contains a non-plugin file.
func TestLoadAllHooksReturnErrorForNonPluginFile(t *testing.T) {
	// Arrange & Act
	calloutCarriers, err := LoadAllHooks("", "boilerplate")

	// Assert
	require.Nil(t, calloutCarriers)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot open hook library")
}

// Test that the extract callout carrier function returns an error if the
// Version function is missing in the hook.
func TestExtractCalloutCarrierMissingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(nil, errors.New("symbol not found")))

	// Act
	carrier, err := extractCarrier(library, "foo")

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "symbol not found")
	require.ErrorContains(t, err, "lookup for symbol: Version")
}

// Test that the extract carrier function returns an error if the Version
// function has an invalid signature.
func TestExtractCalloutCarrierInvalidVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(invalidSignature, nil))

	// Act
	carrier, err := extractCarrier(library, "foo")

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "symbol Version has unexpected signature")
}

// Test that the extract carrier function returns an error if the hook is
// dedicated for another application.
func TestExtractCalloutCarrierNonMatchingApplication(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(validVersion("bar", ""), nil))

	// Act
	carrier, err := extractCarrier(library, "foo")

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "hook library dedicated for another program: bar")
}

// Test that the extract carrier function returns an error if the hook is
// dedicated for another Stork version.
func TestExtractCalloutCarrierNonMatchingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(validVersion("foo", "non.matching.version"), nil))

	// Act
	carrier, err := extractCarrier(library, "foo")

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "incompatible hook version: non.matching.version")
}

// Test that the extract carrier function returns an error if the Load
// function is missing in the hook.
func TestExtractCalloutCarrierMissingLoad(t *testing.T) {
	// Arrange
	plugin := newPluginMock(nil, errors.New("symbol not found"))
	plugin.addLookupVersion(validVersion("foo", stork.Version))
	library := newLibraryManager("", plugin)

	// Act
	carrier, err := extractCarrier(library, "foo")

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "symbol not found")
	require.ErrorContains(t, err, "lookup for symbol: Load")
}

// Test that the extract carrier function returns an error if the Load
// function has an invalid signature.
func TestExtractCalloutCarrierInvalidLoad(t *testing.T) {
	// Arrange
	plugin := newPluginMock(invalidSignature, nil)
	plugin.addLookupVersion(validVersion("foo", stork.Version))
	library := newLibraryManager("", plugin)

	// Act
	carrier, err := extractCarrier(library, "foo")

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "symbol Load has unexpected signature")
}

// Test that the extract carrier function returns an error if the Load
// function fails.
func TestExtractCalloutCarrierLoadFails(t *testing.T) {
	// Arrange
	plugin := newPluginMock(validLoad("", errors.New("error in load")), nil)
	plugin.addLookupVersion(validVersion("foo", stork.Version))
	library := newLibraryManager("", plugin)

	// Act
	carrier, err := extractCarrier(library, "foo")

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "error in load")
}

// Test that the extract carrier function return a proper output on success.
func TestExtractCalloutCarrier(t *testing.T) {
	// Arrange
	plugin := newPluginMock(validLoad("bar", nil), nil)
	plugin.addLookupVersion(validVersion("foo", stork.Version))
	library := newLibraryManager("", plugin)

	// Act
	carrier, err := extractCarrier(library, "foo")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, carrier)
}
