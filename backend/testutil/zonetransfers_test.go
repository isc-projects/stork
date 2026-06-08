package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the GetTestZoneTransfers function generates the correct number of
// zone transfer states.
func TestGetTestZoneTransfers(t *testing.T) {
	zoneTransfers := GetTestZoneTransfers()
	require.Len(t, zoneTransfers, 5)
}
