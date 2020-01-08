package storkutil

import (
	"testing"
	"github.com/stretchr/testify/require"
)

// Test that HostWithPort function generates proper output.
func TestHostWithPortUrl(t *testing.T) {
	require.Equal(t, "http://localhost:1000/", HostWithPortUrl("localhost", 1000))
	require.Equal(t, "http://192.0.2.0:1/", HostWithPortUrl("192.0.2.0", 1))
}
