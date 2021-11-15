package dbmodel

import (
	"testing"
	"time"

	require "github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that configuration review information can be inserted and updated
// in the database.
func TestConfigReport(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add an app with two daemons.
	app := &App{
		Type:      AppTypeKea,
		MachineID: machine.ID,
		Daemons: []*Daemon{
			NewKeaDaemon("dhcp4", true),
			NewKeaDaemon("dhcp6", true),
		},
	}
	daemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 2)

	// Add config review entry into the config_review table.
	configReview := &ConfigReview{
		CreatedAt:  time.Date(2021, 11, 15, 10, 0, 0, 0, time.UTC),
		ConfigHash: "1234",
		Signature:  "2345",
		DaemonID:   daemons[0].ID,
	}

	err = AddConfigReview(db, configReview)
	require.NoError(t, err)

	// Fetch inserted config review and verify it is valid.
	returnedConfigReview, err := GetConfigReviewByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedConfigReview)
	require.NotZero(t, returnedConfigReview.ID)
	require.Equal(t, "1234", returnedConfigReview.ConfigHash)
	require.Equal(t, "2345", returnedConfigReview.Signature)
	returnedCreatedAt := returnedConfigReview.CreatedAt
	require.EqualValues(t, configReview.CreatedAt, returnedConfigReview.CreatedAt)
	require.NotNil(t, returnedConfigReview.Daemon)
	require.NotNil(t, returnedConfigReview.Daemon.KeaDaemon)

	// Update the config review data.
	configReview.CreatedAt = time.Time{}
	configReview.ConfigHash = "2345"
	configReview.Signature = "3456"

	err = AddConfigReview(db, configReview)
	require.NoError(t, err)

	// Fetch the config review information again.
	returnedConfigReview, err = GetConfigReviewByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedConfigReview)
	require.NotZero(t, returnedConfigReview.ID)
	require.Equal(t, "2345", returnedConfigReview.ConfigHash)
	require.Equal(t, "3456", returnedConfigReview.Signature)

	// We inserted null createdAt value. The database should set it to NOW().
	// Make sure that the returned time is within 5 seconds duration. There is
	// no way to test the exact time, but the 5 seconds margin is good enough.
	returnedCreatedAt2 := returnedConfigReview.CreatedAt
	require.WithinDuration(t, time.Now(), returnedCreatedAt2, 5*time.Second)
	require.NotEqual(t, returnedCreatedAt2, returnedCreatedAt)

	// Make sure that the daemon information was also returned.
	require.NotNil(t, returnedConfigReview.Daemon)
	require.NotNil(t, returnedConfigReview.Daemon.KeaDaemon)

	// Trying to fetch a non-existing config review should return nil.
	returnedConfigReview, err = GetConfigReviewByDaemonID(db, daemons[1].ID)
	require.NoError(t, err)
	require.Nil(t, returnedConfigReview)
}
