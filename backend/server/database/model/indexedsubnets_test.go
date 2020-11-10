package dbmodel

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// This test verifies that new instance of the IndexedSubnets structure
// can be created and that the indexes are initially empty.
func TestNewIndexedSubnets(t *testing.T) {
	is := NewIndexedSubnets(nil)
	require.NotNil(t, is)
	require.Empty(t, is.RandomAccess)
	require.Nil(t, is.ByPrefix)
}

// This test verifies that subnets can be inserted into the IndexedSubnets
// structure and that duplicated entries are rejected.
func TestIndexedSubnetsPopulate(t *testing.T) {
	subnets := []Subnet{
		{
			Prefix: "192.0.2.0/24",
		},
	}
	is := NewIndexedSubnets(subnets)
	require.NotNil(t, is)

	require.True(t, is.Populate())

	// Make sure that indexes contain the new subnet.
	require.Len(t, is.RandomAccess, 1)
	require.Len(t, is.ByPrefix, 1)

	// Insert another subnet.
	s := Subnet{
		Prefix: "10.0.0.0/8",
	}
	is.RandomAccess = append(is.RandomAccess, s)
	require.True(t, is.Populate())

	// Both subnets should be now stored in random access index.
	require.Len(t, is.RandomAccess, 2)
	require.Equal(t, "192.0.2.0/24", is.RandomAccess[0].Prefix)
	require.Equal(t, "10.0.0.0/8", is.RandomAccess[1].Prefix)

	// Both subnets should ne stored in the by-prefix index.
	require.Len(t, is.ByPrefix, 2)
	require.Contains(t, is.ByPrefix, "192.0.2.0/24")
	require.Contains(t, is.ByPrefix, "10.0.0.0/8")

	// An attempt to store the same subnet twice should fail.
	is.RandomAccess = append(is.RandomAccess, s)
	require.False(t, is.Populate())

	// We should still have two subnets in the by prefix index.
	require.Len(t, is.ByPrefix, 2)
}

// Benchmark measuring performance of indexing many subnets by prefix.
func BenchmarkIndexedSubnetsPopulate(b *testing.B) {
	// Create many subnets.
	subnets := []Subnet{}
	for i := 0; i < 10000; i++ {
		subnet := Subnet{
			Prefix: fmt.Sprintf("%d.%d.%d.%d/24", byte(i>>24), byte(i>>16), byte(i>>8), byte(i)),
		}
		subnets = append(subnets, subnet)
	}
	// Index the subnets.
	indexedSubnets := NewIndexedSubnets(subnets)

	// Actual benchmark starts here.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indexedSubnets.Populate()
	}
}
