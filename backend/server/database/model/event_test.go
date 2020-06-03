package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that all system groups can be fetched from the database.
func TestEvent(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add info event
	ev1 := &Event{
		Text:  "some info event",
		Level: EvInfo,
		Relations: &Relations{
			Machine: 1,
		},
	}

	err := AddEvent(db, ev1)
	require.NoError(t, err)
	require.NotZero(t, ev1.ID)

	// add erro event
	ev2 := &Event{
		Text:  "some erro event",
		Level: EvError,
		Relations: &Relations{
			App: 2,
		},
	}

	err = AddEvent(db, ev2)
	require.NoError(t, err)
	require.NotZero(t, ev2.ID)

	// get events
	events, total, err := GetEventsByPage(db, 0, 10, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, events, 2)
	var erroEv Event
	var infoEv Event
	if events[0].Level == EvError {
		erroEv = events[0]
		infoEv = events[1]
	} else {
		erroEv = events[1]
		infoEv = events[0]
	}
	require.EqualValues(t, EvError, erroEv.Level)
	require.EqualValues(t, 2, erroEv.Relations.App)
	require.EqualValues(t, EvInfo, infoEv.Level)
	require.EqualValues(t, 1, infoEv.Relations.Machine)
}
