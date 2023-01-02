package hookmanager

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the hook manager is constructed properly.
func TestNewHookManager(t *testing.T) {
	// Arrange & Act
	hookManager := NewHookManager()

	// Assert
	require.NotNil(t, hookManager)
	supportedTypes := hookManager.HookManager.GetExecutor().GetTypesOfSupportedCalloutSpecifications()
	require.Len(t, supportedTypes, 1)
}
