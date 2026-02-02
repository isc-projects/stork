package dbmodel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	dbtest "isc.org/stork/server/database/test"
)

// Test that all system groups can be fetched from the database.
func TestEvent(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine info event
	mEv := &Event{
		Text:    "some info event",
		Details: "more details about info event",
		Level:   EvInfo,
		Relations: &Relations{
			MachineID: 1,
		},
		SSEStreams: []SSEStream{"foo", "bar"},
	}

	err := AddEvent(db, mEv)
	require.NoError(t, err)
	require.NotZero(t, mEv.ID)

	// add daemon error event
	machine := &Machine{
		Address:   "1.2.3.4",
		AgentPort: 321,
	}
	err = AddMachine(db, machine)
	require.NoError(t, err)

	daemon := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)
	require.NotZero(t, daemon.ID)

	d1Ev := &Event{
		Text:    "some error event",
		Details: "more details about error event",
		Level:   EvError,
		Relations: &Relations{
			DaemonID: daemon.ID,
		},
	}

	err = AddEvent(db, d1Ev)
	require.NoError(t, err)
	require.NotZero(t, d1Ev.ID)

	// add daemon warning event
	d2Ev := &Event{
		Text:    "some warning event",
		Details: "more details about warning event",
		Level:   EvWarning,
		Relations: &Relations{
			DaemonID: daemon.ID,
		},
	}

	err = AddEvent(db, d2Ev)
	require.NoError(t, err)
	require.NotZero(t, d2Ev.ID)

	// add user warning event
	uEv := &Event{
		Text:    "some warning event",
		Details: "more details about warning event",
		Level:   EvWarning,
		Relations: &Relations{
			UserID: 4,
		},
	}

	err = AddEvent(db, uEv)
	require.NoError(t, err)
	require.NotZero(t, uEv.ID)

	// get all events
	events, total, err := GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, events, 4)
	for _, ev := range events {
		switch ev.Level {
		case EvError:
			require.EqualValues(t, d1Ev.Relations.DaemonID, ev.Relations.DaemonID)
			require.EqualValues(t, "some error event", ev.Text)
		case EvInfo:
			require.EqualValues(t, mEv.Relations.MachineID, ev.Relations.MachineID)
			require.EqualValues(t, "some info event", ev.Text)
			require.Len(t, ev.SSEStreams, 2)
			require.EqualValues(t, "foo", ev.SSEStreams[0])
			require.EqualValues(t, "bar", ev.SSEStreams[1])
		case EvWarning:
			if ev.Relations.UserID != 0 {
				require.EqualValues(t, uEv.Relations.UserID, ev.Relations.UserID)
			} else {
				require.EqualValues(t, d2Ev.Relations.DaemonID, ev.Relations.DaemonID)
			}
			require.EqualValues(t, "some warning event", ev.Text)
		}
	}

	// get warning and error events
	events, total, err = GetEventsByPage(db, 0, 10, EvWarning, nil, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	for _, ev := range events {
		require.Contains(t, []EventLevel{EvWarning, EvError}, ev.Level)
	}

	// get only error events
	events, total, err = GetEventsByPage(db, 0, 10, EvError, nil, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, events, 1)
	require.EqualValues(t, EvError, events[0].Level)
	require.EqualValues(t, d1Ev.Relations.DaemonID, events[0].Relations.DaemonID)
	require.EqualValues(t, "some error event", events[0].Text)
	require.Nil(t, events[0].SSEStreams)

	// get daemon events
	d := "dhcp4"
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, &d, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, events, 2)

	require.EqualValues(t, EvError, events[0].Level)
	require.EqualValues(t, d1Ev.Relations.DaemonID, events[0].Relations.DaemonID)
	require.EqualValues(t, "some error event", events[0].Text)
	require.Nil(t, events[0].SSEStreams)

	require.EqualValues(t, EvWarning, events[1].Level)
	require.EqualValues(t, d2Ev.Relations.DaemonID, events[1].Relations.DaemonID)
	require.EqualValues(t, "some warning event", events[1].Text)
	require.Nil(t, events[1].SSEStreams)

	dID := d1Ev.Relations.DaemonID
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, &dID, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, events, 2)

	require.EqualValues(t, EvError, events[0].Level)
	require.EqualValues(t, d1Ev.Relations.DaemonID, events[0].Relations.DaemonID)
	require.EqualValues(t, "some error event", events[0].Text)
	require.Nil(t, events[0].SSEStreams)

	require.EqualValues(t, EvWarning, events[1].Level)
	require.EqualValues(t, d2Ev.Relations.DaemonID, events[1].Relations.DaemonID)
	require.EqualValues(t, "some warning event", events[1].Text)
	require.Nil(t, events[1].SSEStreams)

	// get machine events
	m := mEv.Relations.MachineID
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, &m, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, events, 1)
	require.EqualValues(t, EvInfo, events[0].Level)
	require.EqualValues(t, m, events[0].Relations.MachineID)
	require.EqualValues(t, "some info event", events[0].Text)
	require.Len(t, events[0].SSEStreams, 2)
	require.EqualValues(t, "foo", events[0].SSEStreams[0])
	require.EqualValues(t, "bar", events[0].SSEStreams[1])

	// get user events
	u := uEv.Relations.UserID
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, &u, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, events, 1)
	require.EqualValues(t, EvWarning, events[0].Level)
	require.EqualValues(t, u, events[0].Relations.UserID)
	require.EqualValues(t, "some warning event", events[0].Text)
	require.Nil(t, events[0].SSEStreams)

	// no events
	unknownDaemonName := "unknownDaemon"
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, &unknownDaemonName, nil, nil, &u, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.NotNil(t, events)
	require.Empty(t, events)
}

// Test that TestGetEventsByPage returns results with correct sorting.
func TestGetEventsByPageSorting(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add test events
	ev := &Event{
		Text:      "text a",
		Level:     EvError,
		Details:   "detail a",
		CreatedAt: time.Now().UTC(),
	}

	err := AddEvent(db, ev)
	require.NoError(t, err)

	ev = &Event{
		Text:      "text b",
		Level:     EvInfo,
		Details:   "detail b",
		CreatedAt: time.Now().UTC().Add(time.Second),
	}

	err = AddEvent(db, ev)
	require.NoError(t, err)

	ev = &Event{
		Text:      "text a",
		Level:     EvError,
		Details:   "detail c",
		CreatedAt: time.Now().UTC().Add(time.Second),
	}

	err = AddEvent(db, ev)
	require.NoError(t, err)

	// get all events sorted by created_at
	events, total, err := GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "created_At", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	for i := range events {
		if i > 0 {
			require.GreaterOrEqual(t, events[i].CreatedAt, events[i-1].CreatedAt)
		}
	}

	// get all events sorted by created_at DESC
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "created_At", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	for i := range events {
		if i > 0 {
			require.LessOrEqual(t, events[i].CreatedAt, events[i-1].CreatedAt)
		}
	}

	// get all events sorted by text
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "text", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	require.EqualValues(t, events[0].Text, events[1].Text)       // First two events have the same text.
	require.Greater(t, events[1].CreatedAt, events[0].CreatedAt) // So after this, they should be sorted by created_at.
	require.Greater(t, events[2].Text, events[1].Text)

	// get all events sorted by text DESC
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "text", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	require.EqualValues(t, events[1].Text, events[2].Text)       // Last two events have the same text.
	require.Greater(t, events[1].CreatedAt, events[2].CreatedAt) // So after this, they should be sorted by created_at.
	require.Greater(t, events[0].Text, events[1].Text)

	// get all events sorted by level
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "level", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	require.EqualValues(t, events[1].Level, events[2].Level)     // Last two events have the same level.
	require.Greater(t, events[2].CreatedAt, events[1].CreatedAt) // So after this, they should be sorted by created_at.
	require.Greater(t, events[1].Level, events[0].Level)

	// get all events sorted by level DESC
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "level", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	require.EqualValues(t, events[0].Level, events[1].Level)     // First two events have the same level.
	require.Greater(t, events[0].CreatedAt, events[1].CreatedAt) // So after this, they should be sorted by created_at.
	require.Greater(t, events[1].Level, events[2].Level)

	// get all events sorted by details
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "details", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	for i := range events {
		if i > 0 {
			require.GreaterOrEqual(t, events[i].Details, events[i-1].Details)
		}
	}

	// get all events sorted by details DESC
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "details", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	for i := range events {
		if i > 0 {
			require.LessOrEqual(t, events[i].Details, events[i-1].Details)
		}
	}
}

// Test that the event level is converted to the human-readable form.
func TestConvertLevelToString(t *testing.T) {
	require.EqualValues(t, "info", EvInfo.String())
	require.EqualValues(t, "warning", EvWarning.String())
	require.EqualValues(t, "error", EvError.String())
	require.EqualValues(t, "unknown", EventLevel(42).String())
}

func TestDeleteAllEvents(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add example event
	exEv := &Event{
		Text:    "some info event",
		Details: "more details about info event",
		Level:   EvInfo,
		Relations: &Relations{
			MachineID: 1,
		},
		SSEStreams: []SSEStream{"foo", "bar"},
	}

	err := AddEvent(db, exEv)
	require.NoError(t, err)
	require.NotZero(t, exEv.ID)

	delCount, err := DeleteAllEvents(db)

	require.NoError(t, err)
	require.EqualValues(t, 1, delCount)

	events, total, err := GetEventsByPage(db, 0, 10, EvInfo, nil, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.Len(t, events, 0)
}
