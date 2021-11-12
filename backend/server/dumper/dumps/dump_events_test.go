package dumps_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/dumper/dumps"
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
		// Info level event - it shouldn't be in the dump.
		Level: dbmodel.EvInfo,
		Relations: &dbmodel.Relations{
			MachineID: m.ID,
		},
	})

	dump := dumps.NewEventsDump(db, m)

	// Act
	err := dump.Execute()

	// Assert
	require.NoError(t, err)

	require.EqualValues(t, 1, dump.NumberOfArtifacts())
	artifact := dump.GetArtifact(0).(dumps.StructArtifact)
	artifactContent := artifact.GetStruct()
	events, ok := artifactContent.([]dbmodel.Event)
	require.True(t, ok)

	require.Len(t, events, 1)
	event := events[0]
	require.EqualValues(t, "foo", event.Text)
}
