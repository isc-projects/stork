package agentcomm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Simple test puller implementation.
type testPuller struct {
	mockChan  chan int
	callCount int
}

// This function should be invoked periodically by the puller and record the
// number of calls made.
func (puller *testPuller) mockPull() (int, error) {
	puller.callCount++
	puller.mockChan <- puller.callCount
	return 0, nil
}

// Tests that an instanec of the periodic puller would trigger execution
// of the caller's custom function.
func TestNewPeriodicPuller(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// We need to initialize default settings in the database which include intervals
	// of the pullers.
	err := dbmodel.InitializeSettings(db)
	require.NoError(t, err)

	// Override the default interval of the hosts puller and set it to 1 second.
	err = dbmodel.SetSettingInt(db, "kea_hosts_puller_interval", 1)
	require.NoError(t, err)

	// Create an instance of the test puller which implements our mock function to
	// be invoked  by the puller.
	testPullerInstance := &testPuller{
		mockChan:  make(chan int, 1),
		callCount: 0,
	}
	puller, err := NewPeriodicPuller(db, nil, "Test", "kea_hosts_puller_interval",
		testPullerInstance.mockPull)
	require.NoError(t, err)
	require.NotNil(t, puller)
	defer puller.Shutdown()

	// Wait up to 5 seconds for the mock function to be invoked. Probe whether
	// the function has been invoked or not every 1 second.
	require.Eventually(t, func() bool {
		if len(testPullerInstance.mockChan) == 0 {
			return false
		}
		return <-testPullerInstance.mockChan > 0
	},
		5*time.Second,
		time.Second,
		"test puller did not invoke a function within a desired time period")
}
