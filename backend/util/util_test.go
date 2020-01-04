package storkutil

import (
	"testing"
	"github.com/stretchr/testify/require"
)

// Test that HostWithPort function generates proper output.
func TestHostWithPort(t *testing.T) {
	require.Equal(t, "http://localhost:1000/", HostWithPort("localhost", 1000))
	require.Equal(t, "http://192.0.2.0:1/", HostWithPort("192.0.2.0", 1))
}
