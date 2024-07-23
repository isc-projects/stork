package agentcomm

import (
	"sync/atomic"
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
	_ = dbmodel.InitializeSettings(db, 0)
	_ = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 1)
	agents := NewConnectedAgents(nil, nil, nil, nil, nil)
	defer agents.Shutdown()

	// Act
	puller, err := NewPeriodicPuller(db, agents, "test puller", "kea_hosts_puller_interval",
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
	_ = dbmodel.InitializeSettings(db, 0)
	_ = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 1)

	puller, _ := NewPeriodicPuller(db, nil, "test puller", "kea_hosts_puller_interval",
		func() error { return nil })
	defer puller.Shutdown()

	initialInterval := puller.PeriodicExecutor.GetInterval()

	// Act
	_ = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 10)

	// Assert
	require.EqualValues(t, 1*time.Second, initialInterval)
	require.Eventually(t, func() bool {
		currentInterval := puller.GetInterval()
		return currentInterval == 10*time.Second
	}, 5*time.Second, time.Second, "puller didn't update the interval")
}

// Test that  the puller doesn't stop when the read interval from the database
// fails.
func TestExecutePullerWhileDatabaseIsDown(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	_ = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 1)

	var callCount atomic.Uint64
	callCount.Store(0)

	puller, _ := NewPeriodicPuller(db, nil, "test puller", "kea_hosts_puller_interval",
		func() error {
			// Increment a counter to check if the puller is still running.
			callCount.Add(1)
			return nil
		})
	defer puller.Shutdown()

	// Wait for the initial puller execution.
	require.Eventually(t, func() bool {
		return callCount.Load() > 0
	}, 5*time.Second, time.Second)

	// Stop the database to simulate a failure.
	teardown()

	// Get the counter value after the database failure.
	callCountAfterFailure := callCount.Load()

	// Act & Assert
	require.Eventually(t, func() bool {
		// Periodic executor updates the interval after the pulling. So, the
		// counter is incremented on failure. If the puller is still running,
		// the counter should be incremented more times.
		currentCallCount := callCount.Load()
		return currentCallCount >= callCountAfterFailure+2
	}, 5*time.Second, time.Second)
}

// Test that the interval setting name is returned properly.
func TestGetIntervalName(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	puller, _ := NewPeriodicPuller(db, nil, "test puller", "kea_hosts_puller_interval",
		func() error { return nil })
	defer puller.Shutdown()

	// Act
	intervalName := puller.GetIntervalSettingName()

	// Assert
	require.EqualValues(t, "kea_hosts_puller_interval", intervalName)
}

// Test that the puller returns properly last finished time.
func TestPullerSavesLastExecutionTime(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	_ = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 1)

	var pullTimeWrapper atomic.Value
	pullTimeWrapper.Store((*time.Time)(nil))

	puller, _ := NewPeriodicPuller(db, nil, "test puller", "kea_hosts_puller_interval",
		func() error {
			if pullTimeWrapper.Load() == (*time.Time)(nil) {
				current := time.Now()
				pullTimeWrapper.Swap(&current)
			}
			return nil
		})
	startTime := time.Now()

	// Act
	require.Eventually(t, func() bool {
		return pullTimeWrapper.Load() != (*time.Time)(nil)
	}, 5*time.Second, 500*time.Millisecond)
	finishTime := puller.GetLastFinishedAt()

	// Assert
	puller.Shutdown()
	pullTime := pullTimeWrapper.Load().(*time.Time)
	require.LessOrEqual(t, startTime, *pullTime)
	require.LessOrEqual(t, *pullTime, finishTime)
}

// Test that the puller returns properly last invoked time.
func TestPullerSavesLastInvokedTime(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)
	_ = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 1)

	var pullTimeWrapper atomic.Value
	pullTimeWrapper.Store((*time.Time)(nil))

	puller, _ := NewPeriodicPuller(db, nil, "test puller", "kea_hosts_puller_interval",
		func() error {
			if pullTimeWrapper.Load() == (*time.Time)(nil) {
				current := time.Now()
				pullTimeWrapper.Swap(&current)
			}
			return nil
		})
	startTime := time.Now()

	// Act
	require.Eventually(t, func() bool {
		return pullTimeWrapper.Load() != (*time.Time)(nil)
	}, 5*time.Second, 500*time.Millisecond)
	invokedTime := puller.GetLastInvokedAt()

	// Assert
	puller.Shutdown()
	pullTime := pullTimeWrapper.Load().(*time.Time)
	require.LessOrEqual(t, startTime, *pullTime)
	require.LessOrEqual(t, invokedTime, *pullTime)
}
