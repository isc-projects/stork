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
	require.Equal(t, "66ac590193757277b806e89cd2a7da1a", hasher.Hash("abc"))
	require.Equal(t, "66ac5977c3757277b806e89cd2f09aea", hasher.Hash(123))
	require.Equal(t, "3de6abaeb865995b12060efa3356c58f", hasher.Hash([]int{2, 3, 4}))
}
