package hooksutil

import (
	"io"
	"plugin"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/hooks"
)

// Plugin mock.
type pluginMock struct {
	// General lookup results
	lookupResult         any
	lookupErr            error
	specificLookupOutput map[string]struct {
		result any
		err    error
	}
}

// Constructs the plugin mock instance.
func newPluginMock(lookupResult any, lookupErr error) *pluginMock {
	return &pluginMock{lookupResult, lookupErr, make(map[string]struct {
		result any
		err    error
	})}
}

// Implements the plugin interface. Returns the fixed values.
func (p *pluginMock) Lookup(symName string) (plugin.Symbol, error) {
	if output, ok := p.specificLookupOutput[symName]; ok {
		return output.result, output.err
	}
	return p.lookupResult, p.lookupErr
}

// Add a dedicated lookup output for the Version symbol.
func (p *pluginMock) addLookupVersion(result any) {
	p.specificLookupOutput["Version"] = struct {
		result any
		err    error
	}{
		result: result,
		err:    nil,
	}
}

// Function with a signature non-matching to Load and Version.
func invalidSignature(int64) bool {
	return false
}

// Creates a valid Load function that returns the given output.
// If the string content is empty, the function will return nil instead.
func validLoad(s string, err error) hooks.HookLoadFunction {
	return func() (hooks.Callout, error) {
		if s == "" {
			return nil, err
		}
		return io.NopCloser(strings.NewReader(s)), err
	}
}

// Creates a valid Version function that returns the given output.
func validVersion(program, version string) hooks.HookVersionFunction {
	return func() (string, string) {
		return program, version
	}
}

// Test that the library constructor returns an error for an unknown file.
func TestNewLibraryManagerReturnErrorForInvalidPath(t *testing.T) {
	// Arrange & Act
	library, err := NewLibraryManager("/non/exist/file")

	// Assert
	require.Nil(t, library)
	require.Error(t, err)
}

// Test that the library manager constructor sets members properly.
func TestNewLibraryManager(t *testing.T) {
	// Arrange
	plugin := newPluginMock(nil, nil)

	// Act
	library := newLibraryManager("foo", plugin)

	// Assert
	require.Equal(t, plugin, library.p)
	require.EqualValues(t, "foo", library.path)
}

// Test that the load library function returns an error if the plugin doesn't
// contain the load function.
func TestLoadReturnErrorForMissingFunction(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(nil, errors.New("symbol not found")))

	// Act
	callout, err := library.Load()

	// Assert
	require.Nil(t, callout)
	require.Error(t, err)
}

// Test that the load library function returns an error if the load plugin
// function has unexpected signature.
func TestLoadReturnErrorForInvalidSignature(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(invalidSignature, nil))

	// Act
	callout, err := library.Load()

	// Assert
	require.Nil(t, callout)
	require.ErrorContains(t, err, "symbol Load has unexpected signature")
}

// Test that the load library function returns an error if the load plugin
// function returns and error.
func TestLoadReturnErrorOnFail(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(
		validLoad(
			"",
			errors.New("error in load"),
		),
		nil,
	))

	// Act
	callout, err := library.Load()

	// Assert
	require.Nil(t, callout)
	require.ErrorContains(t, err, "error in load")
}

// Test that the load library function returns a callout object on success.
func TestLoadReturnCalloutOnSuccess(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(
		validLoad("bar", nil), nil,
	))

	// Act
	callout, err := library.Load()

	// Assert
	require.NotNil(t, callout)
	require.NoError(t, err)
}

// Test that the version library function returns an error if the plugin doesn't
// contain the version function.
func TestVersionReturnErrorForMissingFunction(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(nil, errors.New("symbol not found")))

	// Act
	program, version, err := library.Version()

	// Assert
	require.Empty(t, program)
	require.Empty(t, version)
	require.Error(t, err)
}

// Test that the version library function returns an error if the version plugin
// function has unexpected signature.
func TestVersionReturnErrorForInvalidSignature(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(invalidSignature, nil))

	// Act
	program, version, err := library.Version()

	// Assert
	require.Empty(t, program)
	require.Empty(t, version)
	require.ErrorContains(t, err, "symbol Version has unexpected signature")
}

// Test that the version library function returns an application name and
// version string on success.
func TestVersionReturnAppAndVersionOnSuccess(t *testing.T) {
	// Arrange
	library := newLibraryManager("", newPluginMock(validVersion("bar", "baz"), nil))

	// Act
	program, version, err := library.Version()

	// Assert
	require.EqualValues(t, "bar", program)
	require.EqualValues(t, "baz", version)
	require.NoError(t, err)
}

// Test that the path is returned properly.
func TestGetPath(t *testing.T) {
	// Arrange
	library := newLibraryManager("foo", nil)

	// Act
	path := library.GetPath()

	// Assert
	require.EqualValues(t, "foo", path)
}
