package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
	// Arrange & Act
	hookManager := NewHookManagerFromCallouts([]any{})

	// Assert
	require.NotNil(t, hookManager)
}
