package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test instantiating UnsupportedConfigParameter error.
func TestNewUnsupportedConfigParameter(t *testing.T) {
	err := NewUnsupportedConfigParameter("valid-lifetime")
	require.EqualError(t, err, "unsupported configuration parameter valid-lifetime")
}
