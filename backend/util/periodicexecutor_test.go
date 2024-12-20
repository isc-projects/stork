package storkutil

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Simple test executor implementation.
type testExecutor struct {
	testedExecutor *PeriodicExecutor
	pausedChan     chan bool
	mutex          *sync.Mutex
	done           bool
}

// This function should be invoked periodically by the executor and record the
// boolean flag indicating whether the executor is paused or not while calling
// the handler.
func (executor *testExecutor) mockPull() error {
	if executor.done {
		return nil
	}
	paused := false
	executor.mutex.Lock()
	if executor.testedExecutor != nil {
		paused = executor.testedExecutor.Paused()
	}
	executor.mutex.Unlock()
	executor.done = true
	executor.pausedChan <- paused
	return nil
}

// Test test verifies that the executor is paused while handler function is
// being invoked.
func TestPausedWhileHandling(t *testing.T) {
	getIntervalFunc := func() (time.Duration, error) { return 1 * time.Second, nil }

	// Create an instance of the test executor which implements our mock function to
	// be invoked by the executor under test.
	testExecutorInstance := &testExecutor{
		pausedChan: make(chan bool, 1),
		mutex:      new(sync.Mutex),
	}
	executor, err := NewPeriodicExecutor("test executor",
		testExecutorInstance.mockPull, getIntervalFunc)
	require.NotNil(t, executor)
	require.NoError(t, err)
	defer executor.Shutdown()

	// There is a potential race condition between handler function trying to
	// access the executor's state and assigning the executor instance.
	testExecutorInstance.mutex.Lock()
	testExecutorInstance.testedExecutor = executor
	testExecutorInstance.mutex.Unlock()

	paused := false

	// Wait up to 5 seconds for the executor to be paused. This should happen when the
	// handler function is being invoked.
	require.Eventually(t, func() bool {
		if len(testExecutorInstance.pausedChan) == 0 {
			return false
		}
		// Record the paused flag value captured during handler execution.
		paused = <-testExecutorInstance.pausedChan
		return true
	},
		5*time.Second,
		time.Second,
		"test executor did not invoke a function within a desired time period")

	// The executor should have been paused while handler was invoked.
	require.True(t, paused)
}

// This test verifies that the executor can be paused and resumed.
func TestPauseAndUnpauseOrReset(t *testing.T) {
	testCases := []string{"Unpause", "Reset"}
	getIntervalFunc := func() (time.Duration, error) { return 10 * time.Millisecond, nil }

	// The test is almost the same for both cases. The only difference is
	// that we call Resume or Reset to start the executor again.
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc, func(t *testing.T) {
			// Create an instance of the test executor which implements our mock function to
			// be invoked by the executor under test.
			testExecutorInstance := &testExecutor{
				pausedChan: make(chan bool, 1),
				mutex:      new(sync.Mutex),
			}
			executor, err := NewPeriodicExecutor("test executor",
				testExecutorInstance.mockPull, getIntervalFunc)
			require.NotNil(t, executor)
			require.NoError(t, err)
			defer executor.Shutdown()

			// Pause the executor twice and unpause it once. The executor should remain
			// paused because there were more calls to Pause() than Unpause().
			executor.Pause()
			executor.Pause()
			executor.Unpause()

			// The handler function should not be invoked within next 3 seconds when
			// the executor is paused.
			require.Never(t, func() bool {
				invoked := len(testExecutorInstance.pausedChan) > 0
				if invoked {
					<-testExecutorInstance.pausedChan
				}
				return invoked
			},
				1*time.Second,
				50*time.Millisecond,
				"executor function was invoked but it shouldn't when executor is paused")

			// Make sure that the paused flag is set as expected.
			require.True(t, executor.Paused())

			// Depending on the test case, use Unpause or Reset to start the executor again.
			if tc == "Unpause" {
				executor.Unpause()
			} else {
				executor.reset(1)
			}

			// This should result in handler function being called.
			require.Eventually(t, func() bool {
				return len(testExecutorInstance.pausedChan) > 0
			},
				5*time.Second,
				50*time.Millisecond,
				"test executor did not invoke a function within a desired time period")
		})
	}
}

// Test that the interval is properly updated.
func TestGetInterval(t *testing.T) {
	// Arrange
	intervalValue := int64(1)
	getIntervalFunc := func() (time.Duration, error) {
		return time.Duration(atomic.LoadInt64(&intervalValue)) * time.Second, nil
	}
	executor, _ := NewPeriodicExecutor("", func() error { return nil }, getIntervalFunc)
	defer executor.Shutdown()

	// Act
	atomic.StoreInt64(&intervalValue, 10)

	// Assert
	require.Eventually(t, func() bool {
		return executor.GetInterval() == 10*time.Second
	}, 5*time.Second, time.Second,
		"test executor did not update the interval")
}

// Test that the executor name is returned properly.
func TestGetName(t *testing.T) {
	// Arrange
	executor, _ := NewPeriodicExecutor(
		"foobar",
		func() error { return nil },
		func() (time.Duration, error) { return 1 * time.Second, nil },
	)

	// Act
	name := executor.GetName()

	// Assert
	require.EqualValues(t, "foobar", name)
}
