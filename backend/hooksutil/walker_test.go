package hooksutil

import (
	"path"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"isc.org/stork"
	"isc.org/stork/hooks"
	"isc.org/stork/testutil"
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
			addLookupGetVersion(validGetVersion("zab", "incompatible"), nil).
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
	require.ErrorContains(t, err, "dedicated for another program")
}

// Test that the function to load all hooks returns an error if the
// directory doesn't exist.
func TestLoadAllHooksReturnErrorForInvalidDirectory(t *testing.T) {
	// Arrange & Act
	sb := testutil.NewSandbox()
	defer sb.Close()
	walker := NewHookWalker()
	calloutCarriers, err := walker.LoadAllHooks(
		"",
		path.Join(sb.BasePath, "non-exist-directory"),
		map[string]hooks.HookSettings{},
	)

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
			addLookupGetVersion(validGetVersion("baz", stork.Version), nil).
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
			addLookupGetVersion(validGetVersion("baz", stork.Version), nil).
			addLookupLoad(validLoad(nil), nil),
	), nil)
	lookup.EXPECT().OpenLibrary("bar").Return(newLibraryManager("bar",
		newPluginMock().
			addLookupGetVersion(validGetVersion("baz", stork.Version), nil).
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

// Test that the verification returns an error if the GetVersion function is missing
// in the hook.
func TestCheckLibraryCompatibilityMissingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock().
		addLookupGetVersion(nil, errors.New("symbol not found")),
	)

	// Act
	err := checkLibraryCompatibility(library, "foo")

	// Assert
	require.ErrorContains(t, err, "symbol not found")
	require.ErrorContains(t, err, "lookup for symbol: GetVersion")
}

// Test that the verification returns an error if the GetVersion function has an
// invalid signature.
func TestCheckLibraryCompatibilityInvalidGetVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock().
		addLookupGetVersion(invalidSignature, nil),
	)

	// Act
	err := checkLibraryCompatibility(library, "foo")

	// Assert
	require.ErrorContains(t, err, "symbol GetVersion has unexpected signature")
}

// Test that the verification returns an error if the hook is dedicated for
// another application.
func TestCheckLibraryCompatibilityNonMatchingApplication(t *testing.T) {
	// Arrange
	library := newLibraryManager("bar", newPluginMock().
		addLookupGetVersion(validGetVersion("bar", ""), nil),
	)

	// Act
	err := checkLibraryCompatibility(library, "foo")

	// Assert
	require.ErrorContains(t, err, "hook library (bar) dedicated for another program: bar")
}

// Test that the verification returns an error if the hook is dedicated for
// another Stork version.
func TestCheckLibraryCompatibilityNonMatchingVersion(t *testing.T) {
	// Arrange
	library := newLibraryManager("bar", newPluginMock().
		addLookupGetVersion(validGetVersion("foo", "non.matching.version"), nil),
	)

	// Act
	err := checkLibraryCompatibility(library, "foo")

	// Assert
	require.ErrorContains(t, err, "incompatible hook (bar) version: non.matching.version")
}

// Test that the extract carrier function returns an error if the Load
// function is missing in the hook.
func TestExtractCalloutCarrierMissingLoad(t *testing.T) {
	// Arrange
	plugin := newPluginMock().
		addLookupGetVersion(validGetVersion("foo", stork.Version), nil).
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
		addLookupGetVersion(validGetVersion("foo", stork.Version), nil).
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
		addLookupGetVersion(validGetVersion("foo", stork.Version), nil).
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
		addLookupGetVersion(validGetVersion("foo", stork.Version), nil).
		addLookupLoad(validLoad(nil), nil)
	library := newLibraryManager("", plugin)

	// Act
	carrier, err := extractCarrier(library, nil)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, carrier)
	require.Nil(t, carrier.(*calloutCarrierMock).settings)
}

// Test that the provided settings are passed to the load function.
func TestExtractCalloutCarrierPassSettings(t *testing.T) {
	// Arrange
	plugin := newPluginMock().
		addLookupGetVersion(validGetVersion("foo", stork.Version), nil).
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
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	walker := NewHookWalker()

	// Act
	data, err := walker.CollectCLIFlags(
		"",
		path.Join(sb.BasePath, "non-exist-directory"),
	)

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
			addLookupGetVersion(validGetVersion("baz", stork.Version), nil).
			addLookupCreateCLIFlags(invalidSignature, nil),
	), nil)

	walker := newHookWalker(lookup)

	// Act
	flags, err := walker.CollectCLIFlags("baz", "boilerplate")

	// Assert
	require.Nil(t, flags)
	require.ErrorContains(t, err, "symbol CreateCLIFlags has unexpected signature")
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
			addLookupGetVersion(validGetVersion("baz", stork.Version), nil).
			addLookupCreateCLIFlags(validCreateCLIFlags(&struct{}{}), nil),
	), nil)
	lookup.EXPECT().OpenLibrary("bar").Return(newLibraryManager("bar",
		newPluginMock().
			addLookupGetVersion(validGetVersion("baz", stork.Version), nil).
			addLookupCreateCLIFlags(validCreateCLIFlags(nil), nil),
	), nil)

	walker := newHookWalker(lookup)

	// Act
	flags, err := walker.CollectCLIFlags("baz", "fake-directory")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, flags)
	require.Len(t, flags, 2)
	require.Contains(t, flags, "foo")
	require.Contains(t, flags, "bar")
	require.NotNil(t, flags["foo"])
	require.Nil(t, flags["bar"])
}
