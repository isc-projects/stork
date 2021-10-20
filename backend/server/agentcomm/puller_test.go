package agentcomm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the puller is properly created.
func TestNewPeriodicPuller(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db)
	_ = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 1)
	agents := NewConnectedAgents(nil, nil, nil, nil, nil)
	defer agents.Shutdown()

	// Act
	puller, err := NewPeriodicPuller(db, agents, "Test", "kea_hosts_puller_interval",
		func() error { return nil })
	defer puller.Shutdown()

	// Assert
	require.NotNil(t, puller)
	require.NoError(t, err)
	require.NotNil(t, puller.Agents)
	require.NotNil(t, puller.DB)
}

// Test that the puller read interval from the database.
func TestReadIntervalFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db)
	_ = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 1)

	puller, _ := NewPeriodicPuller(db, nil, "Test", "kea_hosts_puller_interval",
		func() error { return nil })
	defer puller.Shutdown()

	initialInterval := puller.PeriodicExecutor.GetInterval()

	// Act
	_ = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 10)

	// Assert
	require.EqualValues(t, 1, initialInterval)
	require.Eventually(t, func() bool {
		currentInterval := puller.GetInterval()
		return currentInterval == 10
	}, 5*time.Second, time.Second, "puller didn't update the interval")
}
