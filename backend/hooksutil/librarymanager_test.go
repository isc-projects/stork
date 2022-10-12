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
	content any
	err     error
}

func newPluginMock(content any, err error) *pluginMock {
	return &pluginMock{content, err}
}

func (p *pluginMock) Lookup(symName string) (plugin.Symbol, error) {
	return p.content, p.err
}

func invalidLoad(int64) bool {
	return false
}

func validLoadWithoutError() (hooks.Closer, error) {
	obj := io.NopCloser(strings.NewReader("foobar"))
	return obj, nil
}

func validLoadWithError() (hooks.Closer, error) {
	return nil, errors.New("error in load")
}

var _ hooks.HookLoadFunction = validLoadWithError

var _ hooks.HookLoadFunction = validLoadWithoutError

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
	library := newLibraryManager("foo", newPluginMock(nil, errors.New("symbol not found")))

	// Act
	callouts, err := library.Load()

	// Assert
	require.Nil(t, callouts)
	require.Error(t, err)
}

// Test that the load library function returns an error if the load plugin
// function has unexpected signature.
func TestLoadReturnErrorForInvalidSignature(t *testing.T) {
	// Arrange
	library := newLibraryManager("foo", newPluginMock(
		invalidLoad, nil),
	)

	// Act
	callouts, err := library.Load()

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "symbol Load has unexpected signature")
}

// Test that the load library function returns an error if the load plugin
// function returns and error.
func TestLoadReturnErrorOnFail(t *testing.T) {
	// Arrange
	library := newLibraryManager("foo", newPluginMock(
		validLoadWithError, nil),
	)

	// Act
	callouts, err := library.Load()

	// Assert
	require.Nil(t, callouts)
	require.ErrorContains(t, err, "error in load")
}

// Test that the load library function returns a callout object on success.
func TestLoadReturnCalloutsOnSuccess(t *testing.T) {
	// Arrange
	library := newLibraryManager("foo", newPluginMock(
		validLoadWithoutError, nil),
	)

	// Act
	callouts, err := library.Load()

	// Assert
	require.NotNil(t, callouts)
	require.NoError(t, err)
}

// Test that the version library function returns an error if the plugin doesn't
// contain the version function.
func TestVersionReturnErrorForMissingFunction(t *testing.T) {
	// Arrange

	// Act

	// Assert

}

// Test that the version library function returns an error if the version plugin
// function has unexpected signature.
func TestVersionReturnErrorForInvalidSignature(t *testing.T) {
	// Arrange

	// Act

	// Assert

}

// Test that the version library function returns a callout object on success.
func TestVersionReturnCalloutsOnSuccess(t *testing.T) {
	// Arrange

	// Act

	// Assert

}

// Test that the path is returned properly.
func TestGetPath(t *testing.T) {
	// Arrange

	// Act

	// Assert

}
