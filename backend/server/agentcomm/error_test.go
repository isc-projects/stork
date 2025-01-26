package agentcomm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test instantiating the ZoneInventoryBusyError.
func TestNewZoneInventoryBusyError(t *testing.T) {
	err := NewZoneInventoryBusyError("agent-foo")
	require.Error(t, err)
	require.Equal(t, "Zone inventory is temporarily busy on the agent agent-foo", err.Error())
}

// Test instantiating the ZoneInventoryNotInitedError.
func TestNewZoneInventoryNotInitedError(t *testing.T) {
	err := NewZoneInventoryNotInitedError("agent-foo")
	require.Error(t, err)
	require.Equal(t, "DNS zones have not been loaded on the agent agent-foo", err.Error())
}
