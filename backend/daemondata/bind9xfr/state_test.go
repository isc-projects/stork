package bind9xfr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the Status string representation is correct.
func TestStatusString(t *testing.T) {
	require.Equal(t, "started", StatusStarted.String())
	require.Equal(t, "connected", StatusConnected.String())
	require.Equal(t, "completed", StatusCompleted.String())
	require.Equal(t, "message", StatusMessage.String())
	require.Empty(t, Status("").String())
}
