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
		Text:    "some info event",
		Details: "more details about info event",
		Level:   EvInfo,
		Relations: &Relations{
			MachineID: 1,
		},
	}

	err := AddEvent(db, ev1)
	require.NoError(t, err)
	require.NotZero(t, ev1.ID)

	// add error event
	ev2 := &Event{
		Text:    "some error event",
		Details: "more details about error event",
		Level:   EvError,
		Relations: &Relations{
			AppID: 2,
		},
	}

	err = AddEvent(db, ev2)
	require.NoError(t, err)
	require.NotZero(t, ev2.ID)

	// add warning event
	ev3 := &Event{
		Text:    "some warning event",
		Details: "more details about warning event",
		Level:   EvWarning,
		Relations: &Relations{
			DaemonID: 3,
		},
	}

	err = AddEvent(db, ev3)
	require.NoError(t, err)
	require.NotZero(t, ev3.ID)

	// get events
	events, total, err := GetEventsByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
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
	require.EqualValues(t, 2, erroEv.Relations.AppID)
	require.EqualValues(t, "some error event", erroEv.Text)
	require.EqualValues(t, EvInfo, infoEv.Level)
	require.EqualValues(t, 1, infoEv.Relations.MachineID)
	require.EqualValues(t, "some info event", infoEv.Text)

	// get daemon events
	d := int64(3)
	events, total, err = GetEventsByPage(db, 0, 10, &d, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, events, 1)
	require.EqualValues(t, EvWarning, events[0].Level)
	require.EqualValues(t, 3, events[0].Relations.DaemonID)
	require.EqualValues(t, "some warning event", events[0].Text)

	// get machine events
	m := int64(1)
	events, total, err = GetEventsByPage(db, 0, 10, nil, &m, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, events, 1)
	require.EqualValues(t, EvInfo, events[0].Level)
	require.EqualValues(t, 1, events[0].Relations.MachineID)
	require.EqualValues(t, "some info event", events[0].Text)
}
