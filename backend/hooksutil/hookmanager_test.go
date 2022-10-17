package hooksutil

import (
	"io"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/hooks"
)

// Test that the hook manager is constructed properly.
func TestNewHookManager(t *testing.T) {
	// Arrange
	type foo interface{}
	type bar interface{}

	// Act
	hookManager := NewHookManager([]reflect.Type{
		reflect.TypeOf((*foo)(nil)).Elem(),
		reflect.TypeOf((*bar)(nil)).Elem(),
	})

	// Assert
	require.NotNil(t, hookManager)
	require.Len(t, hookManager.executor.registeredCallouts, 2)
	require.Contains(t, hookManager.executor.registeredCallouts, reflect.TypeOf((*foo)(nil)).Elem())
}

// Test that the hook manager can be constructed from the empty slice of the
// supported types.
func TestNewHookManagerNoSupportedTypes(t *testing.T) {
	// Arrange & Act
	hookManager := NewHookManager([]reflect.Type{})

	// Assert
	require.NotNil(t, hookManager)
	require.Len(t, hookManager.executor.registeredCallouts, 0)
}

// Test that register method returns an error if the directory doesn't exist.
func TestRegisterCalloutsFromDirectoryReturnErrorForInvalidPath(t *testing.T) {
	// Arrange
	hookManager := NewHookManager(nil)

	// Act
	err := hookManager.RegisterCalloutsFromDirectory("/non/exist/dir")

	// Assert
	require.Error(t, err)
}

// Test that the callouts are registered properly.
func TestRegisterCallouts(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*io.Closer)(nil)).Elem()

	hookManager := NewHookManager([]reflect.Type{
		calloutType,
	})

	// Act
	hookManager.RegisterCallouts([]hooks.Callout{
		io.NopCloser(nil),
	})

	// Assert
	require.True(t, hookManager.GetExecutor().HasRegistered(calloutType))
}

// Test that the executor getter returns a proper object.
func TestGetExecutor(t *testing.T) {
	// Arrange
	hookManager := NewHookManager(nil)

	// Act
	executor := hookManager.GetExecutor()

	// Assert
	require.Equal(t, hookManager.executor, executor)
}

// Test that the hook manager unregisters all callouts on close.
func TestClose(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*io.Closer)(nil)).Elem()
	mock := newMockCalloutFoo()

	hookManager := NewHookManager([]reflect.Type{
		calloutType,
	})

	hookManager.RegisterCallouts([]hooks.Callout{
		mock,
	})

	// Act
	err := hookManager.Close()

	// Assert
	require.EqualValues(t, 1, mock.closeCount)
	require.NoError(t, err)
}

// Test that the hook manager combines the errors returned on close.
func TestCloseCombineErrors(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*io.Closer)(nil)).Elem()

	mock1 := newMockCalloutFoo()
	mock1.closeErr = errors.New("foo")

	mock2 := newMockCalloutFoo()
	mock2.closeErr = errors.New("bar")

	hookManager := NewHookManager([]reflect.Type{
		calloutType,
	})

	hookManager.RegisterCallouts([]hooks.Callout{
		mock1,
		mock2,
	})

	// Act
	err := hookManager.Close()

	// Assert
	require.EqualValues(t, 1, mock1.closeCount)
	require.EqualValues(t, 1, mock2.closeCount)

	require.ErrorContains(t, err, "bar: foo")
}
