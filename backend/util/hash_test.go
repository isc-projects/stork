package storkutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that FNV128 can be created from string.
func TestFnv128(t *testing.T) {
	require.Equal(t, "78896c3a8731e751b6b4257c4cb584bf", Fnv128("Hello world"))
	require.Equal(t, "cd64e1891967a18bcfaa1ff2635a5724", Fnv128("Hello world!"))
}
