package datamodel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test conversion of an app type to string.
func TestAppTypeToString(t *testing.T) {
	// Test Kea.
	appType := AppTypeKea
	require.Equal(t, "kea", appType.String())
	// Test Bind9.
	appType = AppTypeBind9
	require.Equal(t, "bind9", appType.String())
}

// Test checking if an app type is Kea.
func TestAppTypeIsKea(t *testing.T) {
	appType := AppTypeKea
	require.True(t, appType.IsKea())
	require.False(t, appType.IsBind9())
}

// Test checking if an app type is Bind9.
func TestAppTypeIsBind9(t *testing.T) {
	appType := AppTypeBind9
	require.True(t, appType.IsBind9())
	require.False(t, appType.IsKea())
}
