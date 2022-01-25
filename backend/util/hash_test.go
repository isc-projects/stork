package storkutil

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that FNV128 can be created from string.
func TestFnv128(t *testing.T) {
	require.Equal(t, "78896c3a8731e751b6b4257c4cb584bf", Fnv128("Hello world"))
	require.Equal(t, "cd64e1891967a18bcfaa1ff2635a5724", Fnv128("Hello world!"))
}

// Test that the random hash is generated and encoded with base64.
func TestBase64Random(t *testing.T) {
	hash1, err := Base64Random(12)
	require.NoError(t, err)
	require.Len(t, hash1, 16)
	rnd, err := base64.StdEncoding.DecodeString(hash1)
	require.NoError(t, err)
	require.Len(t, rnd, 12)

	hash2, err := Base64Random(24)
	require.NoError(t, err)
	require.Len(t, hash2, 32)
	rnd, err = base64.StdEncoding.DecodeString(hash2)
	require.NoError(t, err)
	require.Len(t, rnd, 24)

	hash3, err := Base64Random(24)
	require.NoError(t, err)
	require.Len(t, hash3, 32)
	rnd, err = base64.StdEncoding.DecodeString(hash3)
	require.NoError(t, err)
	require.Len(t, rnd, 24)

	require.NotEqual(t, hash2, hash3)
}
