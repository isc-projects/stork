package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test creating new hasher instance.
func TestNewHasher(t *testing.T) {
	hasher := NewHasher()
	require.NotNil(t, hasher)
	require.Equal(t, hasherSequence, hasher.seq)
}

// Test that generated hashes depend on the input value.
func TestHash(t *testing.T) {
	hasher := NewHasher()
	require.Equal(t, "66ac55acd8757277b806e89cd189adfb", hasher.Hash("abc"))
	require.Equal(t, "66ac54e7a8757277b806e89cd1102b6b", hasher.Hash(123))
	require.Equal(t, "b42dc2db2b65995b0f767c3ff883e7e6", hasher.Hash([]int{2, 3, 4}))
}
