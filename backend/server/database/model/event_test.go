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

	// add app error event
	machine := &Machine{
		Address:   "1.2.3.4",
		AgentPort: 321,
	}
	err = AddMachine(db, machine)
	require.NoError(t, err)
	app := &App{
		MachineID: machine.ID,
		Type:      "kea",
		Daemons: []*Daemon{{
			Name: "dhcp4",
		}},
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)
	require.NotZero(t, app.ID)
	require.NotZero(t, app.Daemons[0].ID)

	aEv := &Event{
		Text:    "some error event",
		Details: "more details about error event",
		Level:   EvError,
		Relations: &Relations{
			AppID: app.ID,
		},
	}

	err = AddEvent(db, aEv)
	require.NoError(t, err)
	require.NotZero(t, aEv.ID)

	// add daemon warning event
	dEv := &Event{
		Text:    "some warning event",
		Details: "more details about warning event",
		Level:   EvWarning,
		Relations: &Relations{
			DaemonID: app.Daemons[0].ID,
		},
	}

	err = AddEvent(db, dEv)
	require.NoError(t, err)
	require.NotZero(t, dEv.ID)

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
		if ev.Level == EvError {
			require.EqualValues(t, aEv.Relations.AppID, ev.Relations.AppID)
			require.EqualValues(t, "some error event", ev.Text)
		} else if ev.Level == EvInfo {
			require.EqualValues(t, mEv.Relations.MachineID, ev.Relations.MachineID)
			require.EqualValues(t, "some info event", ev.Text)
			require.Len(t, ev.SSEStreams, 2)
			require.EqualValues(t, "foo", ev.SSEStreams[0])
			require.EqualValues(t, "bar", ev.SSEStreams[1])
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
	require.EqualValues(t, aEv.Relations.AppID, events[0].Relations.AppID)
	require.EqualValues(t, "some error event", events[0].Text)
	require.Nil(t, events[0].SSEStreams)

	// get daemon events
	d := "dhcp4"
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, &d, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, events, 1)
	require.EqualValues(t, EvWarning, events[0].Level)
	require.EqualValues(t, dEv.Relations.DaemonID, events[0].Relations.DaemonID)
	require.EqualValues(t, "some warning event", events[0].Text)
	require.Nil(t, events[0].SSEStreams)

	// get app events
	a := "kea"
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, &a, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, events, 1)
	require.EqualValues(t, EvError, events[0].Level)
	require.EqualValues(t, aEv.Relations.AppID, events[0].Relations.AppID)
	require.EqualValues(t, "some error event", events[0].Text)
	require.Nil(t, events[0].SSEStreams)

	// get machine events
	m := mEv.Relations.MachineID
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, nil, nil, &m, nil, "", SortDirAny)
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
	unknownDaemonType := "unknownDaemonType"
	events, total, err = GetEventsByPage(db, 0, 10, EvInfo, &unknownDaemonType, nil, nil, &u, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.NotNil(t, events)
	require.Empty(t, events)
}

// Test that the event level is converted to the human-readable form.
func TestConvertLevelToString(t *testing.T) {
	require.EqualValues(t, "info", EvInfo.String())
	require.EqualValues(t, "warning", EvWarning.String())
	require.EqualValues(t, "error", EvError.String())
	require.EqualValues(t, "unknown", EventLevel(42).String())
}
