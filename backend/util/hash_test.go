package storkutil

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that FNV128 can be created from string.
func TestFnv128(t *testing.T) {
	require.Equal(t, "6c62272e07bb014262b821756295c58d", Fnv128())
	require.Equal(t, "78896c3a8731e751b6b4257c4cb584bf", Fnv128("Hello world"))
	require.Equal(t, "cd64e1891967a18bcfaa1ff2635a5724", Fnv128("Hello world!"))
}

// Test that FNV128 can be created from a slice.
func TestFnv128Slice(t *testing.T) {
	require.Equal(t, "5fef38dc538c46f04f834f10b76bcaa4", Fnv128([]int{10, 50, 70}))
	require.Equal(t, "741167c0758c46f04f8353d94d7fd92e", Fnv128([]int{10, 30, 70}))
}

// Test that FNV128 can be computed from multiple parameters.
func TestFnv128MultipleParams(t *testing.T) {
	require.Equal(t, "cd64e1891967a18bcfaa1ff2635a5734", Fnv128("Hello world", 1))
	require.Equal(t, "cd64e1891967a18bcfaa1ff2635a5737", Fnv128("Hello world", 2))
	require.Equal(t, "ad84de097683c70886554f4040294cce", Fnv128("Hello world", 1, 2))
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
