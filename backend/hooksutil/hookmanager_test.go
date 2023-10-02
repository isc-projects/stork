package hooksutil

import (
	"io"
	"path"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/hooks"
	"isc.org/stork/testutil"
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
	require.Len(t, hookManager.executor.registeredCarriers, 2)
	require.Contains(t, hookManager.executor.registeredCarriers, reflect.TypeOf((*foo)(nil)).Elem())
}

// Test that the hook manager can be constructed from the empty slice of the
// supported types.
func TestNewHookManagerNoSupportedTypes(t *testing.T) {
	// Arrange & Act
	hookManager := NewHookManager([]reflect.Type{})

	// Assert
	require.NotNil(t, hookManager)
	require.Len(t, hookManager.executor.registeredCarriers, 0)
}

// Test that register method returns an error if the directory doesn't exist.
func TestRegisterHooksFromDirectoryReturnErrorForInvalidPath(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	hookManager := NewHookManager(nil)

	// Act
	err := hookManager.RegisterHooksFromDirectory(
		"foo",
		path.Join(sb.BasePath, "not-exists-directory"),
		map[string]hooks.HookSettings{},
	)

	// Assert
	require.Error(t, err)
}

// Test that the callout carriers are registered properly.
func TestRegisterCalloutCarriers(t *testing.T) {
	// Arrange
	specificationType := reflect.TypeOf((*io.Closer)(nil)).Elem()

	hookManager := NewHookManager([]reflect.Type{
		specificationType,
	})

	// Act
	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{
		io.NopCloser(nil),
	})

	// Assert
	require.True(t, hookManager.GetExecutor().HasRegistered(specificationType))
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

// Test that the hook manager unregisters all callout carriers on close.
func TestClose(t *testing.T) {
	// Arrange
	specificationType := reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem()
	mock := newMockCalloutCarrierFoo()

	hookManager := NewHookManager([]reflect.Type{
		specificationType,
	})

	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{
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
	mock1 := newMockCalloutCarrierFoo()
	mock1.closeErr = errors.New("foo")

	mock2 := newMockCalloutCarrierFoo()
	mock2.closeErr = errors.New("bar")

	hookManager := NewHookManager([]reflect.Type{
		reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutSpecificationBar)(nil)).Elem(),
	})

	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{
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
