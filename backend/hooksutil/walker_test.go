package hooksutil

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork"
	"isc.org/stork/hooks"
)

//go:generate mockgen -package=hooksutil -destination=hooklookupmock_test.go isc.org/stork/hooksutil HookLookup

// Test that the function walk over the compatible hooks returns an error if
// it finds an incompatible library.
func TestWalkCompatiblePluginLibrariesReturnsErrorOnIncompatibleLibrary(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lookup := NewMockHookLookup(ctrl)

	lookup.EXPECT().ListFilePaths(gomock.Any()).Return([]string{"foo"}, nil)
	lookup.EXPECT().OpenLibrary("foo").Return(newLibraryManager("foo",
		newPluginMock().
			addLookupVersion(validVersion("zab", "incompatible"), nil).
			addLookupLoad(validLoad(nil), nil),
	), nil)

	walker := newHookWalker(lookup)

	// Act
	err := walker.WalkCompatiblePluginLibraries("baz", "fake-directory",
		func(path string, library *LibraryManager, err error) bool {
			require.Fail(t, "is should never be called")
			return false
		},
	)

	// Assert
	require.ErrorContains(t, err, "hook library dedicated for another program")
}

// Test that the function to load all hooks returns an error if the
// directory doesn't exist.
func TestLoadAllHooksReturnErrorForInvalidDirectory(t *testing.T) {
	// Arrange & Act
	walker := NewHookWalker()
	calloutCarriers, err := walker.LoadAllHooks("", "/non/exist/directory", map[string]hooks.HookSettings{})

	// Assert
	require.Nil(t, calloutCarriers)
	require.ErrorContains(t, err, "cannot find plugin paths")
	require.ErrorContains(t, err, "no such file or directory")
}

// Test that the function to load all hooks returns an error if the directory
// contains a non-plugin file.
func TestLoadAllHooksReturnErrorForNonPluginFile(t *testing.T) {
	// Arrange
	walker := NewHookWalker()

	// Act
	calloutCarriers, err := walker.LoadAllHooks("", "boilerplate", map[string]hooks.HookSettings{})

	// Assert
	require.Nil(t, calloutCarriers)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot open hook library")
}

// Test that the function to load all hooks returns an error if the problem
// occurs while extracting the callout carrier.
func TestLoadAllHooksReturnErrorForCarrierExtractingFailure(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lookup := NewMockHookLookup(ctrl)

	lookup.EXPECT().ListFilePaths(gomock.Any()).Return([]string{"foo", "bar"}, nil)
	lookup.EXPECT().OpenLibrary("foo").Return(newLibraryManager("foo",
		newPluginMock().
			addLookupVersion(validVersion("baz", stork.Version), nil).
			addLookupLoad(validLoad(errors.New("cannot load")), nil),
	), nil)

	walker := newHookWalker(lookup)

	// Act
	calloutCarriers, err := walker.LoadAllHooks("baz", "boilerplate", map[string]hooks.HookSettings{})

	// Assert
	require.Nil(t, calloutCarriers)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot load")
}

// Test that the function to load all hooks returns callouts if all paths are
// pointed on the valid hooks.
func TestLoadAllHooksReturnCalloutCarriersOnSuccess(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lookup := NewMockHookLookup(ctrl)

	lookup.EXPECT().ListFilePaths(gomock.Any()).Return([]string{"foo", "bar"}, nil)
	lookup.EXPECT().OpenLibrary("foo").Return(newLibraryManager("foo",
		newPluginMock().
			addLookupVersion(validVersion("baz", stork.Version), nil).
			addLookupLoad(validLoad(nil), nil),
	), nil)
	lookup.EXPECT().OpenLibrary("bar").Return(newLibraryManager("bar",
		newPluginMock().
			addLookupVersion(validVersion("baz", stork.Version), nil).
			addLookupLoad(validLoad(nil), nil),
	), nil)

	walker := newHookWalker(lookup)
	settings := map[string]hooks.HookSettings{"foo": &struct{}{}}

	// Act
	carriers, err := walker.LoadAllHooks("baz", "fake-directory", settings)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, carriers)
	require.Len(t, carriers, 2)
	require.NotNil(t, carriers[0].(*calloutCarrierMock).settings)
	require.Nil(t, carriers[1].(*calloutCarrierMock).settings)
}

// Test that the verification returns an error if the Version function is missing
// in the hook.
func TestCheckLibraryCompatibilityMissingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock().
		addLookupVersion(nil, errors.New("symbol not found")),
	)

	// Act
	err := checkLibraryCompatibility(library, "foo")

	// Assert
	require.ErrorContains(t, err, "symbol not found")
	require.ErrorContains(t, err, "lookup for symbol: Version")
}

// Test that the verification returns an error if the Version function has an
// invalid signature.
func TestCheckLibraryCompatibilityInvalidVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock().
		addLookupVersion(invalidSignature, nil),
	)

	// Act
	err := checkLibraryCompatibility(library, "foo")

	// Assert
	require.ErrorContains(t, err, "symbol Version has unexpected signature")
}

// Test that the verification returns an error if the hook is dedicated for
// another application.
func TestCheckLibraryCompatibilityNonMatchingApplication(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock().
		addLookupVersion(validVersion("bar", ""), nil),
	)

	// Act
	err := checkLibraryCompatibility(library, "foo")

	// Assert
	require.ErrorContains(t, err, "hook library dedicated for another program: bar")
}

// Test that the verification returns an error if the hook is dedicated for
// another Stork version.
func TestCheckLibraryCompatibilityNonMatchingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock().
		addLookupVersion(validVersion("foo", "non.matching.version"), nil),
	)

	// Act
	err := checkLibraryCompatibility(library, "foo")

	// Assert
	require.ErrorContains(t, err, "incompatible hook version: non.matching.version")
}

// Test that the extract carrier function returns an error if the Load
// function is missing in the hook.
func TestExtractCalloutCarrierMissingLoad(t *testing.T) {
	// Arrange
	plugin := newPluginMock().
		addLookupVersion(validVersion("foo", stork.Version), nil).
		addLookupLoad(nil, errors.New("symbol not found"))
	library := newLibraryManager("", plugin)

	// Act
	carrier, err := extractCarrier(library, nil)

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "symbol not found")
	require.ErrorContains(t, err, "lookup for symbol: Load")
}

// Test that the extract carrier function returns an error if the Load
// function has an invalid signature.
func TestExtractCalloutCarrierInvalidLoad(t *testing.T) {
	// Arrange
	plugin := newPluginMock().
		addLookupVersion(validVersion("foo", stork.Version), nil).
		addLookupLoad(invalidSignature, nil)
	library := newLibraryManager("", plugin)

	// Act
	carrier, err := extractCarrier(library, nil)

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "symbol Load has unexpected signature")
}

// Test that the extract carrier function returns an error if the Load
// function fails.
func TestExtractCalloutCarrierLoadFails(t *testing.T) {
	// Arrange
	plugin := newPluginMock().
		addLookupVersion(validVersion("foo", stork.Version), nil).
		addLookupLoad(validLoad(errors.New("error in load")), nil)
	library := newLibraryManager("", plugin)

	// Act
	carrier, err := extractCarrier(library, nil)

	// Assert
	require.Nil(t, carrier)
	require.ErrorContains(t, err, "error in load")
}

// Test that the extract carrier function return a proper output on success.
func TestExtractCalloutCarrier(t *testing.T) {
	// Arrange
	plugin := newPluginMock().
		addLookupVersion(validVersion("foo", stork.Version), nil).
		addLookupLoad(validLoad(nil), nil)
	library := newLibraryManager("", plugin)

	// Act
	carrier, err := extractCarrier(library, nil)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, carrier)
	require.Nil(t, carrier.(*calloutCarrierMock).settings)
}

// Test that the the provided settings are passed to the load function.
func TestExtractCalloutCarrierPassSettings(t *testing.T) {
	// Arrange
	plugin := newPluginMock().
		addLookupVersion(validVersion("foo", stork.Version), nil).
		addLookupLoad(validLoad(nil), nil)
	library := newLibraryManager("", plugin)
	settings := &struct{}{}

	// Act
	carrier, err := extractCarrier(library, settings)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, carrier)
	require.NotNil(t, carrier.(*calloutCarrierMock).settings)
}

// Test that the function to extract the prototype of all hooks returns an
// error if the  directory doesn't exist.
func TestCollectCLIFlagsReturnErrorForInvalidDirectory(t *testing.T) {
	// Arrange & Act
	walker := NewHookWalker()
	data, err := walker.CollectCLIFlags("", "/non/exist/directory")

	// Assert
	require.Nil(t, data)
	require.ErrorContains(t, err, "cannot find plugin paths")
	require.ErrorContains(t, err, "no such file or directory")
}

// Test that the function to extract the prototype of all hooks returns an
// error if the directory contains a non-plugin file.
func TestCollectCLIFlagsReturnErrorForNonPluginFile(t *testing.T) {
	// Arrange & Act
	walker := NewHookWalker()
	data, err := walker.CollectCLIFlags("", "boilerplate")

	// Assert
	require.Nil(t, data)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot open hook library")
}

// Test that the function to collect all hook settings returns an error if the
// symbol is invalid.
func TestCollectCLIFlagsReturnErrorForInvalidSymbol(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lookup := NewMockHookLookup(ctrl)

	lookup.EXPECT().ListFilePaths(gomock.Any()).Return([]string{"foo"}, nil)
	lookup.EXPECT().OpenLibrary("foo").Return(newLibraryManager("foo",
		newPluginMock().
			addLookupVersion(validVersion("baz", stork.Version), nil).
			addLookupCLIFlags(invalidSignature, nil),
	), nil)

	walker := newHookWalker(lookup)

	// Act
	settings, err := walker.CollectCLIFlags("baz", "boilerplate")

	// Assert
	require.Nil(t, settings)
	require.ErrorContains(t, err, "symbol CLIFlags has unexpected signature")
}

// Test that the function to collect all hook settings returns settings on
// success.
func TestCollectCLIFlagsOnSuccess(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lookup := NewMockHookLookup(ctrl)

	lookup.EXPECT().ListFilePaths(gomock.Any()).Return([]string{"foo", "bar"}, nil)
	lookup.EXPECT().OpenLibrary("foo").Return(newLibraryManager("foo",
		newPluginMock().
			addLookupVersion(validVersion("baz", stork.Version), nil).
			addLookupCLIFlags(validCLIFlags(&struct{}{}), nil),
	), nil)
	lookup.EXPECT().OpenLibrary("bar").Return(newLibraryManager("bar",
		newPluginMock().
			addLookupVersion(validVersion("baz", stork.Version), nil).
			addLookupCLIFlags(validCLIFlags(nil), nil),
	), nil)

	walker := newHookWalker(lookup)

	// Act
	settings, err := walker.CollectCLIFlags("baz", "fake-directory")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.Len(t, settings, 2)
	require.Contains(t, settings, "foo")
	require.Contains(t, settings, "bar")
	require.NotNil(t, settings["foo"])
	require.Nil(t, settings["bar"])
}
