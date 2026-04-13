package agent

import (
	bufio "bufio"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Test that subscriptions can be created in the log tracker that
// concurrently read from the same log file, and then they can
// be detached.
func TestLogTrackerSubscriptions(t *testing.T) {
	// In each test case there are several different check points where
	// we verify the data returned by the log tracker. Each check point
	// contains indexes for subscriber 1 and subscriber 2 that indicate
	// which line numbers should have been captured by the subscribers.
	type checkPoint struct {
		// Expected line indexes for subscriber 1.
		s1 []int
		// Expected line indexes for subscriber 2.
		s2 []int
	}
	// Test cases.
	type testCase struct {
		// Test case name.
		name string
		// Capture options for subscriber 1. The tests really differ by
		// the capture options.
		options1 []logReaderCaptureOption
		// Capture options for subscriber 2. The tests really differ by
		// the capture options.
		options2 []logReaderCaptureOption
		// Expected returned line indexes in first check point.
		checkPoint1 checkPoint
		// Expected returned line indexes in second check point.
		checkPoint2 checkPoint
		// Expected returned line indexes in third check point.
		checkPoint3 checkPoint
	}

	// Test cases.
	testCases := []testCase{
		{
			name:     "both subscriptions with start and follow",
			options1: []logReaderCaptureOption{logReaderCaptureOptionFollow()},
			options2: []logReaderCaptureOption{logReaderCaptureOptionFollow()},
			checkPoint1: checkPoint{
				// Subscriber 1 and subscriber 2 should capture all the lines
				// emitted so far because both subscribers start from the beginning
				// of the log file.
				s1: []int{0, 1},
				s2: []int{0, 1},
			},
			checkPoint2: checkPoint{
				// Both subscribers should also capture the newly appended line
				// because both subscribers are following the log.
				s1: []int{0, 1, 2},
				s2: []int{0, 1, 2},
			},
			checkPoint3: checkPoint{
				// First subscriber was shut down, so it should not capture any new lines.
				// The second subscriber captures one more line.
				s1: []int{0, 1, 2},
				s2: []int{0, 1, 2, 3},
			},
		},
		{
			name:     "first subscription with start and follow, second with end and follow",
			options1: []logReaderCaptureOption{logReaderCaptureOptionFollow()},
			options2: []logReaderCaptureOption{logReaderCaptureOptionFromEnd(), logReaderCaptureOptionFollow()},
			checkPoint1: checkPoint{
				// Subscriber 1 captures all the lines emitted so far. The second subscriber
				// starts from the end so it should not capture initial lines.
				s1: []int{0, 1},
				s2: []int{},
			},
			checkPoint2: checkPoint{
				// Subscriber 1 captures new line in addition to the initial lines.
				// The second subscriber captures the new line.
				s1: []int{0, 1, 2},
				s2: []int{2},
			},
			// Subscriber 1 was shut down, so it captures no more lines. Subscriber 2
			// captures one more line in addition to the line it had captured.
			checkPoint3: checkPoint{
				s1: []int{0, 1, 2},
				s2: []int{2, 3},
			},
		},
		{
			name:     "first subscription with end and follow, second with start and follow",
			options1: []logReaderCaptureOption{logReaderCaptureOptionFromEnd(), logReaderCaptureOptionFollow()},
			options2: []logReaderCaptureOption{logReaderCaptureOptionFollow()},
			checkPoint1: checkPoint{
				// Subscriber 1 starts from the end so it should not capture initial lines.
				// Subscriber 2 captures the initial lines.
				s1: []int{},
				s2: []int{0, 1},
			},
			checkPoint2: checkPoint{
				// Subscriber 1 captures newly appended line. Subscriber 2 also captures this
				// new line.
				s1: []int{2},
				s2: []int{0, 1, 2},
			},
			checkPoint3: checkPoint{
				// Subscriber 1 was shut down, so it captures no more lines. Subscriber 2
				// captures one more line in addition to the lines it had captured.
				s1: []int{2},
				s2: []int{0, 1, 2, 3},
			},
		},
		{
			name:     "both subscriptions with start and no follow",
			options1: []logReaderCaptureOption{},
			options2: []logReaderCaptureOption{},
			checkPoint1: checkPoint{
				// Both subscribers should capture all the lines emitted so far
				// because both subscribers start from the beginning of the log file.
				s1: []int{0, 1},
				s2: []int{0, 1},
			},
			checkPoint2: checkPoint{
				// Neither subscriber captures new lines because they were not
				// configured to follow the log.
				s1: []int{0, 1},
				s2: []int{0, 1},
			},
			checkPoint3: checkPoint{
				// Still no new lines are captured because the subscribers were not
				// configured to follow the log.
				s1: []int{0, 1},
				s2: []int{0, 1},
			},
		},
		{
			name:     "first subscription with no follow, second with follow",
			options1: []logReaderCaptureOption{},
			options2: []logReaderCaptureOption{logReaderCaptureOptionFollow()},
			checkPoint1: checkPoint{
				// Both subscribers should capture initial lines.
				s1: []int{0, 1},
				s2: []int{0, 1},
			},
			checkPoint2: checkPoint{
				// Subscriber 1 is not configured to follow the log, so it captures no new lines.
				// Subscriber 2 captures the new line in addition to the initial lines.
				s1: []int{0, 1},
				s2: []int{0, 1, 2},
			},
			checkPoint3: checkPoint{
				// Subscriber 1 was shut down and it doesn't follow the log, so it captures
				// no new lines. Subscriber 2 captures all the new lines as it is configured
				// to follow the log.
				s1: []int{0, 1},
				s2: []int{0, 1, 2, 3},
			},
		},
		{
			name:     "first subscription with follow, second with no follow",
			options1: []logReaderCaptureOption{logReaderCaptureOptionFollow()},
			options2: []logReaderCaptureOption{},
			checkPoint1: checkPoint{
				// Both subscribers should capture initial lines.
				s1: []int{0, 1},
				s2: []int{0, 1},
			},
			checkPoint2: checkPoint{
				// Subscriber 1 follows the log, so it captures all the new lines.
				// Subscriber 2 is not configured to follow the log, so it captures no new lines.
				s1: []int{0, 1, 2},
				s2: []int{0, 1},
			},
			checkPoint3: checkPoint{
				// Subscriber 1 was shut down so it doesn't capture any new lines.
				// Subscriber 2 does not follow the log, so it doesn't capture any new lines.
				s1: []int{0, 1, 2},
				s2: []int{0, 1},
			},
		},
	}

	// Generate some contents. There are four lines with indexes starting from
	// 0 up to 3. These indexes are used in the test cases specifications.
	testLines := []string{
		"This is the first log line",
		"This is the second log line",
		"This is the third log line",
		"This is the fourth log line",
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sandbox := testutil.NewSandbox()
			defer sandbox.Close()

			// Create the log file with two initial lines.
			_, err := sandbox.Write("test.log", strings.Join(testLines[0:2], "\n")+"\n")
			require.NoError(t, err)

			// Create the log tracker.
			tracker := newLogTracker(storkutil.NewSystemCommandExecutor(), logTrackerConfig{
				textLogReaderConfig: textLogReaderConfig{
					poll: true,
				},
				channelSize: 128,
			})
			require.NotNil(t, tracker)
			defer tracker.stop()

			// Create first subscription.
			sub1, err := tracker.subscribe(append(testCase.options1, logReaderCaptureOptionFileName(filepath.Join(sandbox.BasePath, "test.log")))...)
			require.NoError(t, err)
			require.NotNil(t, sub1)
			defer sub1.stop()

			mutex := sync.RWMutex{}

			// Capture all the lines into first subscription.
			capturedLines := make([]string, 0)
			go func() {
				for line := range sub1.ch {
					mutex.Lock()
					capturedLines = append(capturedLines, line.text)
					mutex.Unlock()
				}
			}()

			// Ensure that first subscriber captured all the lines.
			require.Eventually(t, func() bool {
				mutex.RLock()
				defer mutex.RUnlock()
				return len(capturedLines) >= len(testCase.checkPoint1.s1)
			}, 10*time.Second, 10*time.Millisecond)

			// Create second subscription attached to the same capture. It reads the
			// logs from the beginning of the log file.
			sub2, err := tracker.subscribe(append(testCase.options2, logReaderCaptureOptionFileName(filepath.Join(sandbox.BasePath, "test.log")))...)
			require.NoError(t, err)
			require.NotNil(t, sub2)
			defer sub2.stop()

			// Create second subscription.
			capturedLines2 := make([]string, 0)
			go func() {
				for line := range sub2.ch {
					mutex.Lock()
					capturedLines2 = append(capturedLines2, line.text)
					mutex.Unlock()
				}
			}()

			// Make sure that the second subscriber captured all the lines.
			require.Eventually(t, func() bool {
				mutex.RLock()
				defer mutex.RUnlock()
				return len(capturedLines2) >= len(testCase.checkPoint1.s2)
			}, 10*time.Second, 10*time.Millisecond)

			// Check point 1: Make sure that subscribers captured their respective lines.
			for i, index := range testCase.checkPoint1.s1 {
				require.Equal(t, testLines[index], capturedLines[i])
			}
			for i, index := range testCase.checkPoint1.s2 {
				require.Equal(t, testLines[index], capturedLines2[i])
			}

			// Append the third line to the log file.
			_, err = sandbox.Append("test.log", testLines[2]+"\n")
			require.NoError(t, err)

			// Make sure that subscribers captured their respective lines.
			require.Eventually(t, func() bool {
				mutex.RLock()
				defer mutex.RUnlock()
				return len(capturedLines) >= len(testCase.checkPoint2.s1) && len(capturedLines2) >= len(testCase.checkPoint2.s2)
			}, 10*time.Second, 10*time.Millisecond)

			// Check point 2: Make sure that following subscribers captured this new line
			// and non-following subscribers captured no new lines.
			for i, index := range testCase.checkPoint2.s1 {
				require.Equal(t, testLines[index], capturedLines[i])
			}
			for i, index := range testCase.checkPoint2.s2 {
				require.Equal(t, testLines[index], capturedLines2[i])
			}

			// Detach the first subscription.
			sub1.stop()

			// Append another line that may be only captured by the second subscriber,
			// assuming that the second subscriber is following the log.
			_, err = sandbox.Append("test.log", testLines[3]+"\n")
			require.NoError(t, err)

			// Make sure that the second subscriber captured the new line.
			require.Eventually(t, func() bool {
				mutex.RLock()
				defer mutex.RUnlock()
				return len(capturedLines2) >= len(testCase.checkPoint3.s2)
			}, 10*time.Second, 10*time.Millisecond)

			// Check point 3: Make sure that the subscribers captured their respective lines.
			// The first subscriber was shut down, so it captures no new lines.
			for i, index := range testCase.checkPoint2.s1 {
				require.Equal(t, testLines[index], capturedLines[i])
			}
			for i, index := range testCase.checkPoint2.s2 {
				require.Equal(t, testLines[index], capturedLines2[i])
			}

			// Detach the second subscription.
			sub2.stop()

			// Make sure that the capture is garbage collected.
			require.Eventually(t, func() bool {
				return !tracker.isBusy()
			}, 10*time.Second, 10*time.Millisecond)
			require.False(t, tracker.isStopped())

			// Stop the log tracker and ensure it was stopped.
			tracker.stop()
			require.True(t, tracker.isStopped())
		})
	}
}

// This test checks that the log tracker can be stopped and all the subscriptions
// are subsequently cancelled.
func TestLogTrackerStop(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create two separate log files. One for each capture.
	_, err := sandbox.Write("a.log", "This is in the first file\n")
	require.NoError(t, err)
	_, err = sandbox.Write("b.log", "This is in the second file\n")
	require.NoError(t, err)

	tracker := newLogTracker(storkutil.NewSystemCommandExecutor(), logTrackerConfig{
		textLogReaderConfig: textLogReaderConfig{
			poll: true,
		},
		channelSize: 128,
	})
	require.NotNil(t, tracker)

	// Create several subscriptions.
	var subs []*logTrackingSubscriber
	for i := 0; i < 10; i++ {
		// Even and odd subscriptions will be attached to different captures.
		fileName := "a.log"
		if i%2 == 0 {
			fileName = "b.log"
		}
		sub, err := tracker.subscribe(logReaderCaptureOptionFileName(filepath.Join(sandbox.BasePath, fileName)), logReaderCaptureOptionFollow())
		require.NoError(t, err)
		require.NotNil(t, sub)
		subs = append(subs, sub)
	}

	// Stop the log tracker and ensure it was stopped.
	tracker.stop()
	require.True(t, tracker.isStopped())

	// Make sure that all the subscriptions are cancelled.
	for _, sub := range subs {
		// There must be some data buffered but the channel should be closed.
		require.LessOrEqual(t, len(sub.ch), 2)
		for range sub.ch {
			// Drain the channel.
		}
		_, ok := <-sub.ch
		// Check that the channel is closed.
		require.False(t, ok)
		// Check that the subscriber was cancelled.
		require.True(t, sub.isStopped())
	}
}

// This test checks that the log tracker can be stopped and all the subscriptions
// are subsequently cancelled.
func TestLogTrackerStopSubscriptions(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create two separate log files. One for each capture.
	_, err := sandbox.Write("a.log", "This is in the first file\n")
	require.NoError(t, err)
	_, err = sandbox.Write("b.log", "This is in the second file\n")
	require.NoError(t, err)

	tracker := newLogTracker(storkutil.NewSystemCommandExecutor(), logTrackerConfig{
		textLogReaderConfig: textLogReaderConfig{
			poll: true,
		},
		channelSize: 128,
	})
	require.NotNil(t, tracker)

	// Create several subscriptions.
	var subs []*logTrackingSubscriber
	for i := 0; i < 10; i++ {
		// Even and odd subscriptions will be attached to different captures.
		fileName := "a.log"
		if i%2 == 0 {
			fileName = "b.log"
		}
		sub, err := tracker.subscribe(logReaderCaptureOptionFileName(filepath.Join(sandbox.BasePath, fileName)), logReaderCaptureOptionFollow())
		require.NoError(t, err)
		require.NotNil(t, sub)
		subs = append(subs, sub)
	}

	for i := 0; i < 10; i += 2 {
		// Stop the even subscriptions.
		subs[i].stop()
	}

	// Make sure that even subscriptions are cancelled.
	for i := 0; i < 10; i += 2 {
		sub := subs[i]
		// There must be some data buffered but the channel should be closed.
		require.LessOrEqual(t, len(sub.ch), 2)
		for range sub.ch {
			// Drain the channel.
		}
		_, ok := <-sub.ch
		// Check that the channel is closed.
		require.False(t, ok)
		// Check that the subscriber was cancelled.
		require.True(t, sub.isStopped())
	}

	for i := 1; i < 10; i += 2 {
		// Check that the subscriber was not cancelled.
		require.False(t, subs[i].isStopped())
	}

	for i := 1; i < 10; i += 2 {
		subs[i].stop()
	}

	require.Eventually(t, func() bool {
		return !tracker.isBusy()
	}, 10*time.Second, 10*time.Millisecond)

	require.False(t, tracker.isStopped())

	tracker.stop()
	require.True(t, tracker.isStopped())
}

// This test verifies that it is possible to stop subscription that
// got stuck on infinite write/read over the channel.
func TestLogTrackerStopStuckRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create a scanner that returns the log lines.
		scanner := bufio.NewScanner(strings.NewReader("This is the first log line\nThis is the second log line\n"))
		// Wrap the scanner in the command executor command.
		command := NewMockCommandExecutorOutput(ctrl)
		// Make the scanner available to the reader.
		command.EXPECT().GetScanner().AnyTimes().Return(scanner)
		// Make sure that the command is waited after the scanner is closed.
		waited := atomic.Bool{}
		command.EXPECT().Wait().DoAndReturn(func() error {
			waited.Store(true)
			return nil
		})

		executor := NewMockCommandExecutor(ctrl)
		executor.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), "journalctl", "-f", "-u", "test.service", "-n", "0").Return(command, nil)
		executor.EXPECT().LookPath(gomock.Any()).Return("", nil)

		tracker := newLogTracker(executor, logTrackerConfig{
			// Use unbuffered channel so that the first write blocks
			// until the reader reads it.
			channelSize: 0,
		})
		require.NotNil(t, tracker)
		defer tracker.stop()

		// Create subscription.
		sub, err := tracker.subscribe(logReaderCaptureOptionUnitName("test.service"), logReaderCaptureOptionFollow(), logReaderCaptureOptionFromEnd())
		require.NoError(t, err)
		require.NotNil(t, sub)
		defer sub.stop()

		// This is the semaphore to ensure that the read is not reading
		// the log lines fed by the writer until we release it.
		waitBeforeReadCh := make(chan struct{})
		defer close(waitBeforeReadCh)
		go func() {
			// Do not read the log lines. Write should block.
			<-waitBeforeReadCh
			for range sub.ch {
				// Drain the channel.
			}
		}()

		// Wait until all goroutines are durably blocked. The writer
		// can only be unblocked by the reader or cancellation.
		synctest.Wait()

		// Cancel the subscription. This should unblock the writer.
		sub.stop()

		// Wait until the reader is unblocked.
		synctest.Wait()
		require.True(t, waited.Load())
	})
}

// Test that an error is returned when executor fails to start the log reader.
func TestLogTrackerStartReaderError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Starting the reader returns an error.
	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), "journalctl", "-f").Return(nil, errors.New("test error"))
	executor.EXPECT().LookPath(gomock.Any()).Return("", nil)

	tracker := newLogTracker(executor, logTrackerConfig{
		channelSize: 128,
	})
	require.NotNil(t, tracker)
	defer tracker.stop()

	// Make sure that the subscription returns the error to the caller.
	sub, err := tracker.subscribe(logReaderCaptureOptionUnitName("test.service"), logReaderCaptureOptionFollow())
	require.ErrorContains(t, err, "test error")
	require.Nil(t, sub)

	require.False(t, tracker.isBusy())
}

// Test that an error is returned to the caller when an attempt to find
// the log tracking binary (i.e., journalctl) fails.
func TestLogTrackerStartReaderUnsupported(t *testing.T) {
	testCases := []struct {
		name    string
		options []logReaderCaptureOption
	}{
		{
			name:    "follow",
			options: []logReaderCaptureOption{logReaderCaptureOptionFollow(), logReaderCaptureOptionFromEnd()},
		},
		{
			name:    "backfill",
			options: []logReaderCaptureOption{logReaderCaptureOptionFromEnd()},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Return an error when trying to find the log tracking binary.
			executor := NewMockCommandExecutor(ctrl)
			executor.EXPECT().LookPath(gomock.Any()).Return("", errors.New("command not found"))

			tracker := newLogTracker(executor, logTrackerConfig{
				channelSize: 128,
			})
			require.NotNil(t, tracker)
			defer tracker.stop()

			// A descriptive error should be returned informing that there are no
			// log tracking methods to be used. Note that tailing the file is not
			// taken into account because no file name was specified. Only the
			// systemd log tracking could be used in this case.
			sub, err := tracker.subscribe(testCase.options...)
			require.ErrorContains(t, err, "no supported log tracking method available")
			require.Nil(t, sub)

			require.False(t, tracker.isBusy())
		})
	}
}

// Test that an error is returned when the second subscriber attempts to attach to the
// same capture as the first subscriber, but this attempt failed for some reason. In
// this case, the second subscriber should be cancelled but the first subscriber
// should remain active.
func TestLogTrackerBackfillError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// The scanner should return some log contents.
		scanner := bufio.NewScanner(strings.NewReader("This is the first log line\nThis is the second log line\n"))
		// Wrap the scanner in the command executor command.
		command := NewMockCommandExecutorOutput(ctrl)
		// Make the scanner available to the reader.
		command.EXPECT().GetScanner().AnyTimes().Return(scanner)
		command.EXPECT().Wait().AnyTimes().Return(nil)

		executor := NewMockCommandExecutor(ctrl)
		executor.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), "journalctl", "-f", "-n", "0").Return(command, nil)
		// Creating log reader results in checking whether or not journalctl is available.
		// We expect that it is called once for each subscriber, and it should be successful.
		executor.EXPECT().LookPath(gomock.Any()).Times(2).Return("", nil)
		// The third attempt results from second's reader attempt to backfill the logs
		// from the beginning before it starts tailing. Let's return an error to simulate
		// backfill failure.
		executor.EXPECT().LookPath(gomock.Any()).Times(1).Return("", errors.New("command not found"))

		// Create the log tracker with unbuffered channel. Since reading from the
		// unbuffered channel blocks the synctest framework is able to detect blocked
		// goroutines and coordinate the test.
		tracker := newLogTracker(executor, logTrackerConfig{})
		require.NotNil(t, tracker)
		defer tracker.stop()

		sub1, err := tracker.subscribe(logReaderCaptureOptionFollow(), logReaderCaptureOptionFromEnd())
		require.NoError(t, err)
		require.NotNil(t, sub1)
		defer sub1.stop()

		// This is the semaphore to ensure that the read is not reading
		// the log lines fed by the writer until we release it.
		waitBeforeReadCh := make(chan struct{})
		defer close(waitBeforeReadCh)
		go func() {
			// Do not read the log lines. It should block.
			<-waitBeforeReadCh
			for range sub1.ch {
				// Drain the channel to cleanup.
			}
		}()

		// Wait until the first subscriber is settled.
		synctest.Wait()

		// Try to create another subscription while the first subscriber
		// is still active.
		sub2, err := tracker.subscribe(logReaderCaptureOptionFollow())
		require.ErrorContains(t, err, "no supported log tracking method available")
		require.Nil(t, sub2)

		// Make sure that the first subscriber is still active.
		require.True(t, tracker.isBusy())
	})
}
