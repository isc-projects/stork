package agentcomm

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Simple test puller implementation.
type testPuller struct {
	testedPuller *PeriodicPuller
	pausedChan   chan bool
	mutex        *sync.Mutex
	done         bool
}

// This function should be invoked periodically by the puller and record the
// boolean flag indicating whether the puller is paused or not while calling
// the handler.
func (puller *testPuller) mockPull() (int, error) {
	if puller.done {
		return 0, nil
	}
	paused := false
	puller.mutex.Lock()
	if puller.testedPuller != nil {
		paused = puller.testedPuller.Paused()
	}
	puller.mutex.Unlock()
	puller.done = true
	puller.pausedChan <- paused
	return 0, nil
}

// Test test verifies that the puller is paused while handler function is
// being invoked.
func TestPausedWhileHandling(t *testing.T) {
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
	// be invoked by the puller under test.
	testPullerInstance := &testPuller{
		pausedChan: make(chan bool, 1),
		mutex:      new(sync.Mutex),
	}
	puller, err := NewPeriodicPuller(db, nil, "Test", "kea_hosts_puller_interval",
		testPullerInstance.mockPull)
	require.NoError(t, err)
	require.NotNil(t, puller)
	defer puller.Shutdown()

	// There is a potential race condition between handler function trying to
	// access the puller's state and assigning the puller instance.
	testPullerInstance.mutex.Lock()
	testPullerInstance.testedPuller = puller
	testPullerInstance.mutex.Unlock()

	paused := false

	// Wait up to 5 seconds for the puller to be paused. This should happen when the
	// handler function is being invoked.
	require.Eventually(t, func() bool {
		if len(testPullerInstance.pausedChan) == 0 {
			return false
		}
		// Record the paused flag value captured during handler execution.
		paused = <-testPullerInstance.pausedChan
		return true
	},
		5*time.Second,
		time.Second,
		"test puller did not invoke a function within a desired time period")

	// The puller should have been paused while handler was invoked.
	require.True(t, paused)
}

// This test verifies that the puller can be paused and resumed.
func TestPauseAndUnapuseOrReset(t *testing.T) {
	testCases := []string{"Unpause", "Reset"}

	// The test is almost the same for both cases. The only difference is
	// that we call Resume or Reset to start the puller again.
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc, func(t *testing.T) {
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
			// be invoked by the puller under test.
			testPullerInstance := &testPuller{
				pausedChan: make(chan bool, 1),
				mutex:      new(sync.Mutex),
			}
			puller, err := NewPeriodicPuller(db, nil, "Test", "kea_hosts_puller_interval",
				testPullerInstance.mockPull)
			require.NoError(t, err)
			require.NotNil(t, puller)
			defer puller.Shutdown()

			// Pause the puller twice and unpause it once. The puller should remain
			// paused because there were more calls to Pause() than Unpause().
			puller.Pause()
			puller.Pause()
			puller.Unpause()

			// The handler function should not be invoked within next 3 seconds when
			// the puller is paused.
			require.Never(t, func() bool {
				invoked := len(testPullerInstance.pausedChan) > 0
				if invoked {
					<-testPullerInstance.pausedChan
				}
				return invoked
			},
				3*time.Second,
				time.Second,
				"puller function was invoked but it shouldn't when puller is paused")

			// Make sure that the paused flag is set as expected.
			require.True(t, puller.Paused())

			// Depending on the test case, use Unpause or Reset to start the puller again.
			if tc == "Unpause" {
				puller.Unpause(1)
			} else {
				puller.Reset(1)
			}

			// This should result in handler function being called.
			require.Eventually(t, func() bool {
				return len(testPullerInstance.pausedChan) > 0
			},
				5*time.Second,
				time.Second,
				"test puller did not invoke a function within a desired time period")
		})
	}
}
