package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test generating random zones.
func TestGenerateRandomZones(t *testing.T) {
	zones := GenerateRandomZones(1000)
	require.Len(t, zones, 1000)

	// Ensure we got 10 distinct zones.
	distinctZones := make(map[string]struct{})
	for _, zone := range zones {
		distinctZones[zone.Name] = struct{}{}
	}
	require.Len(t, distinctZones, 1000)
}
