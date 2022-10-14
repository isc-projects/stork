package agent

import (
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"isc.org/stork/hooks/agent/forwardtokeaoverhttpcallout"
)

//go:generate mockgen -package=agent -destination=hook_mock.go isc.org/stork/hooks/agent/forwardtokeaoverhttpcallout BeforeForwardToKeaOverHTTPCallout

// Test that the hook executor is constructed with all supported callout types
// registered.
func TestHookExecutorHasRegisteredCalloutTypes(t *testing.T) {
	// Arrange
	hookExecutor := newHookExecutor()

	// Assert
	supportedTypes := hookExecutor.GetSupportedCalloutTypes()
	require.Len(t, supportedTypes, 1)
}

// Test that the hook manager is constructed properly.
func TestNewHookManager(t *testing.T) {
	// Arrange & Act
	hookManager := NewHookManager()

	// Assert
	require.NotNil(t, hookManager)
	require.NotNil(t, hookManager.executor)
}

// Test that constructing the hook manager from the directory fails if the
// directory doesn't exist.
func TestHookManagerFromDirectoryReturnErrorOnInvalidDirectory(t *testing.T) {
	// Arrange & Act
	hookManager, err := NewHookManagerFromDirectory("/non/exist/dir")

	// Assert
	require.Error(t, err)
	require.Nil(t, hookManager)
}

// Test that the hook manager is constructed properly from the callout objects.
func TestHookManagerFromCallouts(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockBeforeForwardToKeaOverHTTPCallout(ctrl)

	// Act
	hookManager := NewHookManagerFromCallouts([]any{
		mock,
	})

	// Assert
	require.NotNil(t, hookManager)
	require.True(t, hookManager.executor.HasRegistered(reflect.TypeOf((*forwardtokeaoverhttpcallout.BeforeForwardToKeaOverHTTPCallout)(nil)).Elem()))
}
