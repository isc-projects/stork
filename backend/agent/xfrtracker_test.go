package agent

import (
	"bufio"
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
	require.Empty(t, xfrTracker.subscribers)
	require.Nil(t, xfrTracker.cancelFn)
	require.Nil(t, xfrTracker.cancelCh)
}

// Test tracking the log files by the XFR tracker.
func TestXfrTrackerTrackFiles(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	sandbox.Write("xfr-in.1.log", "This is an incoming XFR request log 1\n")
	sandbox.Write("xfr-out.1.log", "This is an outgoing XFR request log 1\n")
	sandbox.Write("xfr-in.2.log", "This is an incoming XFR request log 2\n")
	sandbox.Write("xfr-out.2.log", "This is an outgoing XFR request log 2\n")

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
	err := xfrTracker.trackFiles(filepath.Join(sandbox.BasePath, "xfr-in.1.log"), filepath.Join(sandbox.BasePath, "xfr-out.1.log"))
	require.NoError(t, err)
	require.Len(t, xfrTracker.subscribers, 2)
	require.NotNil(t, xfrTracker.subscribers[0])
	require.NotNil(t, xfrTracker.subscribers[1])
	require.NotNil(t, xfrTracker.cancelFn)
	require.NotNil(t, xfrTracker.cancelCh)

	// Remember the subscriber instances. It will be used later to verify that
	// another subscription is created.
	firstSubscriber0 := xfrTracker.subscribers[0]
	firstSubscriber1 := xfrTracker.subscribers[1]

	// Track the test2.log file. It should close the previous subscription.
	xfrTracker.trackFiles(filepath.Join(sandbox.BasePath, "xfr-in.2.log"), filepath.Join(sandbox.BasePath, "xfr-out.2.log"))
	require.NoError(t, err)
	require.Len(t, xfrTracker.subscribers, 2)
	require.NotNil(t, xfrTracker.subscribers[0])
	require.NotNil(t, xfrTracker.subscribers[1])
	require.NotNil(t, xfrTracker.cancelFn)
	require.NotNil(t, xfrTracker.cancelCh)

	// Remember the new subscriber instances.
	secondSubscriber0 := xfrTracker.subscribers[0]
	secondSubscriber1 := xfrTracker.subscribers[1]

	// Stop the tracker and ensure that the subscriptions are closed.
	xfrTracker.stop()
	require.Empty(t, xfrTracker.subscribers)
	require.Nil(t, xfrTracker.cancelFn)
	require.Nil(t, xfrTracker.cancelCh)

	// Verify that the subscribers were different.
	require.NotEqual(t, firstSubscriber0, secondSubscriber0)
	require.NotEqual(t, firstSubscriber1, secondSubscriber1)
}

// Test tracking only the log file containing incoming XFR requests by the XFR tracker.
func TestXfrTrackerTrackFilesXfrInOnly(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	sandbox.Write("xfr-in.1.log", "This is an incoming XFR request log 1\n")
	sandbox.Write("xfr-in.2.log", "This is an incoming XFR request log 2\n")

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
	err := xfrTracker.trackFiles(filepath.Join(sandbox.BasePath, "xfr-in.1.log"), "")
	require.NoError(t, err)
	require.Len(t, xfrTracker.subscribers, 1)
	require.NotNil(t, xfrTracker.subscribers[0])
	require.NotNil(t, xfrTracker.cancelFn)
	require.NotNil(t, xfrTracker.cancelCh)

	// Remember the subscriber instance. It will be used later to verify that
	// another subscription is created.
	firstSubscriber0 := xfrTracker.subscribers[0]

	// Track the test2.log file. It should close the previous subscription.
	xfrTracker.trackFiles(filepath.Join(sandbox.BasePath, "xfr-in.2.log"), "")
	require.NoError(t, err)
	require.Len(t, xfrTracker.subscribers, 1)
	require.NotNil(t, xfrTracker.subscribers[0])
	require.NotNil(t, xfrTracker.cancelFn)
	require.NotNil(t, xfrTracker.cancelCh)

	// Remember the new subscriber instance.
	secondSubscriber0 := xfrTracker.subscribers[0]

	// Stop the tracker and ensure that the subscriptions are closed.
	xfrTracker.stop()
	require.Empty(t, xfrTracker.subscribers)
	require.Nil(t, xfrTracker.cancelFn)
	require.Nil(t, xfrTracker.cancelCh)

	// Verify that the subscribers were different.
	require.NotEqual(t, firstSubscriber0, secondSubscriber0)
}

// Test tracking only the log file containing outgoing XFR requests by the XFR tracker.
func TestXfrTrackerTrackFilesXfrOutOnly(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	sandbox.Write("xfr-out.1.log", "This is an outgoing XFR request log 1\n")
	sandbox.Write("xfr-out.2.log", "This is an outgoing XFR request log 2\n")

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
	err := xfrTracker.trackFiles("", filepath.Join(sandbox.BasePath, "xfr-out.1.log"))
	require.NoError(t, err)
	require.Len(t, xfrTracker.subscribers, 1)
	require.NotNil(t, xfrTracker.subscribers[0])
	require.NotNil(t, xfrTracker.cancelFn)
	require.NotNil(t, xfrTracker.cancelCh)

	// Remember the subscriber instance. It will be used later to verify that
	// another subscription is created.
	firstSubscriber0 := xfrTracker.subscribers[0]

	// Track the test2.log file. It should close the previous subscription.
	xfrTracker.trackFiles("", filepath.Join(sandbox.BasePath, "xfr-out.2.log"))
	require.NoError(t, err)
	require.Len(t, xfrTracker.subscribers, 1)
	require.NotNil(t, xfrTracker.subscribers[0])
	require.NotNil(t, xfrTracker.cancelFn)
	require.NotNil(t, xfrTracker.cancelCh)

	// Remember the new subscriber instance.
	secondSubscriber0 := xfrTracker.subscribers[0]

	// Stop the tracker and ensure that the subscriptions are closed.
	xfrTracker.stop()
	require.Empty(t, xfrTracker.subscribers)
	require.Nil(t, xfrTracker.cancelFn)
	require.Nil(t, xfrTracker.cancelCh)

	// Verify that the subscribers were different.
	require.NotEqual(t, firstSubscriber0, secondSubscriber0)
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
		require.Len(t, xfrTracker.subscribers, 1)
		require.NotNil(t, xfrTracker.subscribers[0])
		require.NotNil(t, xfrTracker.cancelFn)
		require.NotNil(t, xfrTracker.cancelCh)

		synctest.Wait()

		// Remember the subscriber instance. It will be used later to verify that
		// another subscription is created.
		firstSubscriber0 := xfrTracker.subscribers[0]

		// Create a different subscription. It should close the previous subscription.
		err = xfrTracker.trackSystemdUnit("xfr.service")
		require.NoError(t, err)
		require.Len(t, xfrTracker.subscribers, 1)
		require.NotNil(t, xfrTracker.subscribers[0])
		require.NotNil(t, xfrTracker.cancelFn)
		require.NotNil(t, xfrTracker.cancelCh)

		synctest.Wait()

		// Remember the new subscriber instance.
		secondSubscriber0 := xfrTracker.subscribers[0]

		// Stop the tracker and ensure that the subscription is closed.
		xfrTracker.stop()
		require.Empty(t, xfrTracker.subscribers)
		require.Nil(t, xfrTracker.cancelFn)
		require.Nil(t, xfrTracker.cancelCh)

		synctest.Wait()

		// Verify that the two subscribers were different.
		require.NotEqual(t, firstSubscriber0, secondSubscriber0)
	})
}
