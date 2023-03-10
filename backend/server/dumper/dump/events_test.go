package dump_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	dumppkg "isc.org/stork/server/dumper/dump"
)

// Test that the dump is executed properly.
func TestEventsDumpExecute(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	_ = dbmodel.AddMachine(db, m)
	_ = dbmodel.AddEvent(db, &dbmodel.Event{
		CreatedAt: time.Time{},
		Text:      "foo",
		Level:     dbmodel.EvWarning,
		Relations: &dbmodel.Relations{
			MachineID: m.ID,
		},
	})
	_ = dbmodel.AddEvent(db, &dbmodel.Event{
		CreatedAt: time.Time{},
		Text:      "bar",
		// Info level event - it should be in the dump too.
		Level: dbmodel.EvInfo,
		Relations: &dbmodel.Relations{
			MachineID: m.ID,
		},
	})

	dump := dumppkg.NewEventsDump(db, m)

	// Act
	err := dump.Execute()

	// Assert
	require.NoError(t, err)

	require.EqualValues(t, 1, dump.GetArtifactsNumber())
	artifact := dump.GetArtifact(0).(dumppkg.StructArtifact)
	artifactContent := artifact.GetStruct()
	events, ok := artifactContent.([]dumppkg.EventExtended)
	require.True(t, ok)

	require.Len(t, events, 2)
	event := events[0]
	require.EqualValues(t, "bar", event.Text)
	require.EqualValues(t, "info", event.LevelText)
	event = events[1]
	require.EqualValues(t, "foo", event.Text)
	require.EqualValues(t, "warning", event.LevelText)
}

// Test that the dump contains an empty list if there is no event.
func TestEventsDumpExecuteNoEvents(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	_ = dbmodel.AddMachine(db, m)
	dump := dumppkg.NewEventsDump(db, m)

	// Act
	err := dump.Execute()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 1, dump.GetArtifactsNumber())
	artifact := dump.GetArtifact(0).(dumppkg.StructArtifact)
	artifactContent := artifact.GetStruct()
	require.NotNil(t, artifactContent)
	events, ok := artifactContent.([]dumppkg.EventExtended)
	require.True(t, ok)
	require.Len(t, events, 0)
}
