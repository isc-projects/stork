package bind9xfr

import (
	"testing"
	"time"

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
func TestStateIsXFROut(t *testing.T) {
	state := &State{Client: "127.0.0.1"}
	require.True(t, state.IsXFROut())
}

// Test that the state correctly reports that it is not an outgoing zone transfer.
func TestStateIsNotXFROut(t *testing.T) {
	state := &State{Server: "127.0.0.1"}
	require.False(t, state.IsXFROut())
}

// Test that parameters from one state are derived to another state.
func TestStateDerive(t *testing.T) {
	source := &State{
		ViewName:       "view1",
		ZoneName:       "zone1",
		Serial:         1234567890,
		Client:         "127.0.0.1",
		Server:         "127.0.0.2",
		Status:         StatusStarted,
		MessagesCount:  10,
		RecordsCount:   20,
		BytesCount:     30,
		Duration:       10 * time.Second,
		StartTime:      time.Now(),
		CompletionTime: time.Now(),
		TimeFormat:     TimeFormatRFC3339,
		Message:        "message1",
	}
	destination := &State{}
	source.Derive(destination, "ViewName", "ZoneName", "Serial", "Client", "Server", "Status", "MessagesCount", "RecordsCount", "BytesCount", "Duration", "StartTime", "CompletionTime", "TimeFormat", "Message")
	require.EqualValues(t, source, destination)
}

// Test that the parameters are not derived from the source if the respective
// parameters are already set in the destination.
func TestStateDeriveAlreadySet(t *testing.T) {
	source := &State{
		ViewName:       "view1",
		ZoneName:       "zone1",
		Serial:         1234567890,
		Client:         "127.0.0.1",
		Server:         "127.0.0.2",
		Status:         StatusStarted,
		MessagesCount:  10,
		RecordsCount:   20,
		BytesCount:     30,
		Duration:       10 * time.Second,
		StartTime:      time.Now(),
		CompletionTime: time.Now(),
		TimeFormat:     TimeFormatRFC3339,
		Message:        "message1",
	}
	destination := &State{
		ViewName:       "view2",
		ZoneName:       "zone2",
		Serial:         1234567891,
		Client:         "127.0.0.2",
		Server:         "127.0.0.3",
		Status:         StatusCompleted,
		MessagesCount:  20,
		RecordsCount:   30,
		BytesCount:     40,
		Duration:       20 * time.Second,
		StartTime:      time.Now(),
		CompletionTime: time.Now(),
		TimeFormat:     TimeFormatRFC3339,
		Message:        "message2",
	}
	destinationCopy := *destination
	source.Derive(destination, "ViewName", "ZoneName", "Serial", "Client", "Server", "Status", "MessagesCount", "RecordsCount", "BytesCount", "Duration", "StartTime", "CompletionTime", "TimeFormat", "Message")
	require.EqualValues(t, destinationCopy, *destination)
}

// Test that the parameters from a nil source are not derived to the destination.
func TestStateDeriveNilSource(t *testing.T) {
	source := (*State)(nil)
	destination := &State{
		ViewName: "view1",
	}
	destinationCopy := *destination
	source.Derive(destination, "ViewName")
	require.EqualValues(t, destinationCopy, *destination)
}

// Test that the parameters from a non-nil source are not derived to a nil destination.
func TestStateDeriveNilDestination(t *testing.T) {
	source := &State{
		ViewName: "view1",
	}
	destination := (*State)(nil)
	source.Derive(destination, "ViewName")
	require.Nil(t, destination)
}
