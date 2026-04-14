package agent

import (
	bufio "bufio"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Test instantiating the XFR tracker.
func TestNewXfrTracker(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	sandbox.Write("test.log", "This is a test log\n")

	logTracker := newLogTracker(storkutil.NewSystemCommandExecutor(), logTrackerConfig{})

	xfrTracker := newXfrTracker(logTracker)
	require.NotNil(t, xfrTracker)
	require.Equal(t, logTracker, xfrTracker.logTracker)
	require.Nil(t, xfrTracker.subscriber)
	require.Nil(t, xfrTracker.cancelFn)
	require.Nil(t, xfrTracker.cancelCh)
}

// Test tracking the log file by the XFR tracker.
func TestXfrTrackerTrackFile(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	sandbox.Write("test1.log", "This is a test 1 log\n")
	sandbox.Write("test2.log", "This is a test 2 log\n")

	// Create the log tracker using the default log tracker configuration (using unbuffered channel).
	logTracker := newLogTracker(storkutil.NewSystemCommandExecutor(), logTrackerConfig{
		textLogReaderConfig: textLogReaderConfig{
			poll: true,
		},
	})

	// Create the XFR tracker using the log tracker.
	xfrTracker := newXfrTracker(logTracker)
	require.NotNil(t, xfrTracker)

	// Track the test1.log file. It should create a new subscription.
	err := xfrTracker.trackFile(filepath.Join(sandbox.BasePath, "test1.log"))
	require.NoError(t, err)
	require.NotNil(t, xfrTracker.subscriber)
	require.NotNil(t, xfrTracker.cancelFn)
	require.NotNil(t, xfrTracker.cancelCh)

	// Remember the subscriber instance. It will be used later to verify that
	// another subscription is created.
	subscriber1 := xfrTracker.subscriber

	// Track the test2.log file. It should close the previous subscription.
	xfrTracker.trackFile(filepath.Join(sandbox.BasePath, "test2.log"))
	require.NoError(t, err)
	require.NotNil(t, xfrTracker.subscriber)
	require.NotNil(t, xfrTracker.cancelFn)
	require.NotNil(t, xfrTracker.cancelCh)

	// Remember the new subscriber instance.
	subscriber2 := xfrTracker.subscriber

	// Stop the tracker and ensure that the subscription is closed.
	xfrTracker.stop()
	require.Nil(t, xfrTracker.subscriber)
	require.Nil(t, xfrTracker.cancelFn)
	require.Nil(t, xfrTracker.cancelCh)

	// Verify that the two subscribers were different.
	require.NotEqual(t, subscriber1, subscriber2)
}

// Test tracking the systemd unit logs by the XFR tracker.
func TestXfrTrackerTrackSystemdUnit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Return some output to the subscriber.
		output := NewMockCommandExecutorOutput(ctrl)
		output.EXPECT().GetScanner().AnyTimes().Return(bufio.NewScanner(strings.NewReader("This is a test log\n")))
		output.EXPECT().Wait().AnyTimes().Return(nil)

		// Create a mock command executor that expects journalctl invocations for
		// two different subscribers. They both return the same output.
		executor := NewMockCommandExecutor(ctrl)
		executor.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), "journalctl", "-f", "-u", "named.service", "--since", "1 days ago").Return(output, nil)
		executor.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), "journalctl", "-f", "-u", "xfr.service", "--since", "1 days ago").Return(output, nil)
		executor.EXPECT().LookPath(gomock.Any()).AnyTimes().Return("", nil)

		// Crete the log tracker using the mock command executor and the
		// default log tracker configuration (using unbuffered channel).
		logTracker := newLogTracker(executor, logTrackerConfig{})
		xfrTracker := newXfrTracker(logTracker)
		require.NotNil(t, xfrTracker)

		// Track the named.service logs.
		err := xfrTracker.trackSystemdUnit("named.service")
		require.NoError(t, err)
		// Ensure that the subscription is created.
		require.NotNil(t, xfrTracker.subscriber)
		require.NotNil(t, xfrTracker.cancelFn)
		require.NotNil(t, xfrTracker.cancelCh)

		synctest.Wait()

		// Remember the subscriber instance. It will be used later to verify that
		// another subscription is created.
		subscriber1 := xfrTracker.subscriber

		// Create a different subscription. It should close the previous subscription.
		err = xfrTracker.trackSystemdUnit("xfr.service")
		require.NoError(t, err)
		require.NotNil(t, xfrTracker.subscriber)
		require.NotNil(t, xfrTracker.cancelFn)
		require.NotNil(t, xfrTracker.cancelCh)

		synctest.Wait()

		// Remember the new subscriber instance.
		subscriber2 := xfrTracker.subscriber

		// Stop the tracker and ensure that the subscription is closed.
		xfrTracker.stop()
		require.Nil(t, xfrTracker.subscriber)
		require.Nil(t, xfrTracker.cancelFn)
		require.Nil(t, xfrTracker.cancelCh)

		synctest.Wait()

		// Verify that the two subscribers were different.
		require.NotEqual(t, subscriber1, subscriber2)
	})
}
