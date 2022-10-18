package agent

import (
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/hooks"
)

//go:generate mockgen -package=agent -destination=hook_mock.go isc.org/stork/agent BeforeForwardToKeaOverHTTPCallout

// Test that the hook manager is constructed properly.
func TestNewHookManager(t *testing.T) {
	// Arrange & Act
	hookManager := NewHookManager()

	// Assert
	require.NotNil(t, hookManager)
	supportedTypes := hookManager.HookManager.GetExecutor().GetSupportedCalloutTypes()
	require.Len(t, supportedTypes, 1)
}

// Test that constructing the hook manager from the directory fails if the
// directory doesn't exist.
func TestHookManagerFromDirectoryReturnErrorOnInvalidDirectory(t *testing.T) {
	// Arrange
	hookManager := NewHookManager()

	// Arrange & Act
	err := hookManager.RegisterCalloutsFromDirectory("/non/exist/dir")

	// Assert
	require.Error(t, err)
}

// Test that the hook manager is constructed properly from the callout objects.
func TestHookManagerFromCallouts(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockBeforeForwardToKeaOverHTTPCallout(ctrl)

	hookManager := NewHookManager()

	// Act
	hookManager.RegisterCallouts([]hooks.Callout{
		mock,
	})

	// Assert
	require.True(t, hookManager.GetExecutor().HasRegistered(reflect.TypeOf((*BeforeForwardToKeaOverHTTPCallout)(nil)).Elem()))
}

// Test that the hook manager is closing properly.
func TestHookManagerClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockBeforeForwardToKeaOverHTTPCallout(ctrl)
	mock.
		EXPECT().
		Close().
		Return(nil).
		Times(1)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{
		mock,
	})

	// Act
	err := hookManager.Close()

	// Assert
	require.NoError(t, err)
}

// Test that the hook manager pass through the error from the callout and
// the error raising doesn't interrupt the close operation.
func TestHookManagerCloseWithErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWithoutErr1 := NewMockBeforeForwardToKeaOverHTTPCallout(ctrl)
	mockWithoutErr1.
		EXPECT().
		Close().
		Return(nil).
		Times(1)

	mockWithoutErr2 := NewMockBeforeForwardToKeaOverHTTPCallout(ctrl)
	mockWithoutErr2.
		EXPECT().
		Close().
		Return(nil).
		Times(1)

	mockWithErr := NewMockBeforeForwardToKeaOverHTTPCallout(ctrl)
	mockWithErr.
		EXPECT().
		Close().
		Return(errors.New("foo")).
		Times(1)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{
		mockWithoutErr1,
		mockWithErr,
		mockWithoutErr2,
	})

	// Act
	err := hookManager.Close()

	// Assert
	require.ErrorContains(t, err, "foo")
}
