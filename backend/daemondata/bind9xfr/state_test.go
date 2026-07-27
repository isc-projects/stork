package bind9xfr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the Status string representation is correct.
func TestStatusString(t *testing.T) {
	require.Equal(t, "started", StatusStarted.String())
	require.Equal(t, "completed", StatusCompleted.String())
	require.Equal(t, "up-to-date", StatusUpToDate.String())
	require.Equal(t, "message", StatusMessage.String())
	require.Equal(t, "failed", StatusFailed.String())
	require.Empty(t, Status("").String())
}

// Test that checking if the state has any of the specified statuses is correct.
func TestStateHasAnyStatus(t *testing.T) {
	state := &State{Status: StatusStarted}
	require.True(t, state.HasAnyStatus(StatusStarted, StatusCompleted))
	require.False(t, state.HasAnyStatus(StatusCompleted, StatusFailed))
}

// Test that the state correctly reports that it is an outgoing zone transfer.
func TestStateIsOutgoingTransfer(t *testing.T) {
	state := &State{Client: "127.0.0.1"}
	require.True(t, state.IsOutgoingTransfer())
}

// Test that the state correctly reports that it is not an outgoing zone transfer.
func TestStateIsNotOutgoingTransfer(t *testing.T) {
	state := &State{Server: "127.0.0.1"}
	require.False(t, state.IsOutgoingTransfer())
}
