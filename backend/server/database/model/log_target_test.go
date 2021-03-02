package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the log target can be fetched from the database by ID.
func TestGetLogTargetByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	a := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      AppTypeKea,
		Active:    true,
		Daemons: []*Daemon{
			{
				Name:    "kea-dhcp4",
				Version: "1.7.5",
				Active:  true,
				LogTargets: []*LogTarget{
					{
						Output: "stdout",
					},
					{
						Output: "/tmp/filename.log",
					},
				},
			},
		},
	}
	_, err = AddApp(db, a)
	require.NoError(t, err)
	require.NotZero(t, a.ID)

	require.Len(t, a.Daemons, 1)
	require.Len(t, a.Daemons[0].LogTargets, 2)

	// Make sure that the log targets have been assigned IDs.
	require.NotZero(t, a.Daemons[0].LogTargets[0].ID)
	require.NotZero(t, a.Daemons[0].LogTargets[1].ID)

	// Get the first log target from the database by id.
	logTarget, err := GetLogTargetByID(db, a.Daemons[0].LogTargets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, logTarget)
	require.Equal(t, "stdout", logTarget.Output)
	require.NotNil(t, logTarget.Daemon)
	require.NotNil(t, logTarget.Daemon.App)
	require.NotNil(t, logTarget.Daemon.App.Machine)

	// Get the second log target by id.
	logTarget, err = GetLogTargetByID(db, a.Daemons[0].LogTargets[1].ID)
	require.NoError(t, err)
	require.NotNil(t, logTarget)
	require.Equal(t, "/tmp/filename.log", logTarget.Output)
	require.NotNil(t, logTarget.Daemon)
	require.NotNil(t, logTarget.Daemon.App)
	require.NotNil(t, logTarget.Daemon.App.Machine)

	// Use the non existing id. This should return nil.
	logTarget, err = GetLogTargetByID(db, a.Daemons[0].LogTargets[1].ID+1000)
	require.NoError(t, err)
	require.Nil(t, logTarget)
}
