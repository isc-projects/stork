package storkutil

import (
	"testing"
	"github.com/stretchr/testify/require"
)

// Test that LocalHostWithPort function generates proper output.
func TestLocalHostWithPort(t *testing.T) {
	require.Equal(t, "http://localhost:1000/", LocalHostWithPort(1000))
	require.Equal(t, "http://localhost:1/", LocalHostWithPort(1))
}
