package agent

import (
	"bufio"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

//go:embed testdata/xfr-mixed-logs.txt
var xfrMixedLogs string

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

		// Create the log tracker using the mock command executor and the
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

// Test parsing the time in the BIND 9 format.
func TestXfrTrackerParseTimeBind9Format(t *testing.T) {
	iterator := storkutil.NewPeekingIterator([]string{"23-Feb-2026", "10:41:27.071"})
	parsedTime, timeFormat := parseTime(iterator)
	require.Equal(t, xfrTimeFormatBind9, timeFormat)
	require.NotZero(t, parsedTime)
	require.Equal(t, time.Date(2026, 2, 23, 10, 41, 27, 71*int(time.Millisecond), time.Local), parsedTime)
	_, ok := iterator.Peek()
	// The tokens should be consumed.
	require.False(t, ok)
}

// Test parsing the time in the RFC3339 format.
func TestXfrTrackerParseTimeRFC3339Format(t *testing.T) {
	iterator := storkutil.NewPeekingIterator([]string{"2026-02-23T10:41:27.071Z"})
	parsedTime, timeFormat := parseTime(iterator)
	require.Equal(t, xfrTimeFormatRFC3339, timeFormat)
	require.NotZero(t, parsedTime)
	require.Equal(t, time.Date(2026, 2, 23, 10, 41, 27, 71*int(time.Millisecond), time.UTC), parsedTime)
	_, ok := iterator.Peek()
	// The tokens should be consumed.
	require.False(t, ok)
}

// Test that unrecognized time formats are ignored, and that further
// parsing starts from first token.
func TestXfrTrackerParseTimeUnknownFormat(t *testing.T) {
	type testCase struct {
		name     string
		timeText string
	}
	tests := []testCase{
		{
			name:     "no time",
			timeText: "foo",
		},
		{
			name:     "invalid month name",
			timeText: "23-Febr-2026 10:41:27.071",
		},
		{
			name:     "extraneous character in RFC3339 format",
			timeText: "2026-02-23T10:41:27.071X",
		},
		{
			name:     "no separator in RFC3339 format",
			timeText: "2026-02-23 10:41:27.071",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokens := strings.Fields(test.timeText)
			iterator := storkutil.NewPeekingIterator(tokens)
			parsedTime, timeFormat := parseTime(iterator)
			// The time format should be unknown.
			require.Equal(t, xfrTimeFormatUnknown, timeFormat)
			require.Zero(t, parsedTime)
			// The tokens should not be consumed.
			nextToken, ok := iterator.Peek()
			require.True(t, ok)
			require.Equal(t, tokens[0], nextToken)
		})
	}
}

// Test parsing the incoming transfer started message.
func TestXfrTrackerParseIncomingTransferStarted(t *testing.T) {
	xfrState := parseTransferLogLine("23-Feb-2026 10:41:27.071 zone bind9.example.org/IN: Transfer started.")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusStarted, xfrState.status)
	require.Empty(t, xfrState.viewName)
	require.Equal(t, "bind9.example.org", xfrState.zoneName)
	require.Zero(t, xfrState.serial)
	require.Empty(t, xfrState.client)
	require.Empty(t, xfrState.server)
	require.Zero(t, xfrState.messagesCount)
	require.Zero(t, xfrState.recordsCount)
	require.Zero(t, xfrState.bytesCount)
	require.Zero(t, xfrState.duration)
	require.NotZero(t, xfrState.startTime)
	require.Zero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatBind9, xfrState.timeFormat)
	require.Equal(t, "Transfer started", xfrState.message)
}

func TestXfrTrackerParseIncomingTransferStartedMalformed(t *testing.T) {
	type testCase struct {
		name    string
		logLine string
	}
	tests := []testCase{
		{
			name:    "no zone keyword",
			logLine: "bind9.example.org/IN: Transfer started.",
		},
		{
			name:    "wrong zone keyword",
			logLine: "zonee bind9.example.org/IN: Transfer started.",
		},
		{
			name:    "no transfer keyword",
			logLine: "zone bind9.example.org/IN: started.",
		},
		{
			name:    "no started keyword",
			logLine: "zone bind9.example.org/IN: Transfer.",
		},
		{
			name:    "no zone name",
			logLine: "zone: Transfer started.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			xfrState := parseTransferLogLine(test.logLine)
			require.Nil(t, xfrState)
		})
	}
}

// Test parsing the incoming transfer connected message.
func TestXfrTrackerParseIncomingTransferConnected(t *testing.T) {
	xfrState := parseTransferLogLine("23-Feb-2026 10:41:27.115 0x7ffff943c000: transfer of 'bind9.example.org/IN' from 192.5.5.241#53: connected using 192.5.5.241#53")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusConnected, xfrState.status)
	require.Empty(t, xfrState.viewName)
	require.Equal(t, "bind9.example.org", xfrState.zoneName)
	require.Zero(t, xfrState.serial)
	require.Empty(t, xfrState.client)
	require.Equal(t, "192.5.5.241", xfrState.server)
	require.Zero(t, xfrState.messagesCount)
	require.Zero(t, xfrState.recordsCount)
	require.Zero(t, xfrState.bytesCount)
	require.Zero(t, xfrState.duration)
	require.Zero(t, xfrState.startTime)
	require.Zero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatBind9, xfrState.timeFormat)
	require.Equal(t, "connected using 192.5.5.241#53", xfrState.message)
}

// Test parsing the incoming transfer status success message.
func TestXfrTrackerParseIncomingTransferStatusSuccess(t *testing.T) {
	xfrState := parseTransferLogLine("23-Feb-2026 10:41:27.147 0x7ffffb63b000: transfer of 'drop.rpz.example.com/IN' from 172.24.0.53#53: Transfer status: success")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusMessage, xfrState.status)
	require.Empty(t, xfrState.viewName)
	require.Equal(t, "drop.rpz.example.com", xfrState.zoneName)
	require.Zero(t, xfrState.serial)
	require.Empty(t, xfrState.client)
	require.Equal(t, "172.24.0.53", xfrState.server)
	require.Zero(t, xfrState.messagesCount)
	require.Zero(t, xfrState.recordsCount)
	require.Zero(t, xfrState.bytesCount)
	require.Zero(t, xfrState.duration)
	require.Zero(t, xfrState.startTime)
	require.Zero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatBind9, xfrState.timeFormat)
	require.Equal(t, "Transfer status: success", xfrState.message)
}

// Test parsing the incoming transfer completed message.
func TestXfrTrackerParseIncomingTransferCompleted(t *testing.T) {
	xfrState := parseTransferLogLine("23-Feb-2026 10:41:27.147 0x7ffffb63b000: transfer of 'bind9.example.org/IN' from 172.24.0.53#53: Transfer completed: 1 messages, 5 records, 294 bytes, 0.014 secs (21000 bytes/sec) (serial 201702121)")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusCompleted, xfrState.status)
	require.Empty(t, xfrState.viewName)
	require.Equal(t, "bind9.example.org", xfrState.zoneName)
	require.EqualValues(t, 201702121, xfrState.serial)
	require.Empty(t, xfrState.client)
	require.Equal(t, "172.24.0.53", xfrState.server)
	require.EqualValues(t, 1, xfrState.messagesCount)
	require.EqualValues(t, 5, xfrState.recordsCount)
	require.EqualValues(t, 294, xfrState.bytesCount)
	require.Equal(t, 14*time.Millisecond, xfrState.duration)
	require.Zero(t, xfrState.startTime)
	require.NotZero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatBind9, xfrState.timeFormat)
	require.Equal(t, "Transfer completed: 1 messages, 5 records, 294 bytes, 0.014 secs (21000 bytes/sec) (serial 201702121)", xfrState.message)
}

// Test parsing the incoming transfer started message containing a view name.
func TestXfrTrackerParseIncomingTransferStartedView(t *testing.T) {
	xfrState := parseTransferLogLine("zone drop.rpz.example.com/IN/trusted: Transfer started.")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusStarted, xfrState.status)
	require.Equal(t, "trusted", xfrState.viewName)
	require.Equal(t, "drop.rpz.example.com", xfrState.zoneName)
	require.Zero(t, xfrState.serial)
	require.Empty(t, xfrState.client)
	require.Empty(t, xfrState.server)
	require.Zero(t, xfrState.messagesCount)
	require.Zero(t, xfrState.recordsCount)
	require.Zero(t, xfrState.bytesCount)
	require.Zero(t, xfrState.duration)
	require.Zero(t, xfrState.startTime)
	require.Zero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatUnknown, xfrState.timeFormat)
	require.Equal(t, "Transfer started", xfrState.message)
}

// Test parsing the incoming transfer connected message containing a view name.
func TestXfrTrackerParseIncomingTransferConnectedView(t *testing.T) {
	xfrState := parseTransferLogLine("0x7ffff7c3b000: transfer of 'drop.rpz.example.com/IN/trusted' from 172.24.0.53#53: connected using 172.24.0.53#53 TSIG trusted-key")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusConnected, xfrState.status)
	require.Equal(t, "trusted", xfrState.viewName)
	require.Equal(t, "drop.rpz.example.com", xfrState.zoneName)
	require.Zero(t, xfrState.serial)
	require.Empty(t, xfrState.client)
	require.Equal(t, "172.24.0.53", xfrState.server)
	require.Zero(t, xfrState.messagesCount)
	require.Zero(t, xfrState.recordsCount)
	require.Zero(t, xfrState.bytesCount)
	require.Zero(t, xfrState.duration)
	require.Zero(t, xfrState.startTime)
	require.Zero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatUnknown, xfrState.timeFormat)
	require.Equal(t, "connected using 172.24.0.53#53 TSIG trusted-key", xfrState.message)
}

// Test parsing the outgoing AXFR transfer started message.
func TestXfrTrackerParseOutgoingTransferStarted(t *testing.T) {
	xfrState := parseTransferLogLine("23-Feb-2026 10:41:27.138 client @0x7ffffaa28c00 172.24.0.54#34961/key trusted-key (drop.rpz.example.com): view trusted: transfer of 'drop.rpz.example.com/IN': AXFR started: TSIG trusted-key (serial 201702121)")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusStarted, xfrState.status)
	require.Equal(t, "trusted", xfrState.viewName)
	require.Equal(t, "drop.rpz.example.com", xfrState.zoneName)
	require.EqualValues(t, 201702121, xfrState.serial)
	require.Equal(t, "172.24.0.54", xfrState.client)
	require.Empty(t, xfrState.server)
	require.Zero(t, xfrState.messagesCount)
	require.Zero(t, xfrState.recordsCount)
	require.Zero(t, xfrState.bytesCount)
	require.Zero(t, xfrState.duration)
	require.NotZero(t, xfrState.startTime)
	require.Zero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatBind9, xfrState.timeFormat)
	require.Equal(t, "AXFR started: TSIG trusted-key (serial 201702121)", xfrState.message)
}

// Test parsing the outgoing AXFR transfer completed message.
func TestXfrTrackerParseOutgoingTransferCompleted(t *testing.T) {
	xfrState := parseTransferLogLine("23-Feb-2026 10:41:27.141 client @0x7ffffaa28c00 172.24.0.54#34961/key trusted-key (drop.rpz.example.com): view trusted: transfer of 'drop.rpz.example.com/IN': AXFR ended: 1 messages, 5 records, 294 bytes, 0.004 secs (73500 bytes/sec) (serial 201702121)")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusCompleted, xfrState.status)
	require.Equal(t, "trusted", xfrState.viewName)
	require.Equal(t, "drop.rpz.example.com", xfrState.zoneName)
	require.EqualValues(t, 201702121, xfrState.serial)
	require.Equal(t, "172.24.0.54", xfrState.client)
	require.Empty(t, xfrState.server)
	require.EqualValues(t, 1, xfrState.messagesCount)
	require.EqualValues(t, 5, xfrState.recordsCount)
	require.EqualValues(t, 294, xfrState.bytesCount)
	require.Equal(t, 4*time.Millisecond, xfrState.duration)
	require.Zero(t, xfrState.startTime)
	require.NotZero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatBind9, xfrState.timeFormat)
	require.Equal(t, "AXFR ended: 1 messages, 5 records, 294 bytes, 0.004 secs (73500 bytes/sec) (serial 201702121)", xfrState.message)
}

// Test parsing the setting up zone transfer failed message.
func TestXfrTrackerParseSettingUpZoneTransferFailed(t *testing.T) {
	xfrState := parseTransferLogLine("16-Apr-2026 12:09:11.650 client @0x7ffffa436000 127.0.0.1#55256 (bind.example.org): transfer of 'bind.example.org/IN': setting up zone transfer: failed")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusMessage, xfrState.status)
	require.Empty(t, xfrState.viewName)
	require.Equal(t, "bind.example.org", xfrState.zoneName)
	require.Zero(t, xfrState.serial)
	require.Equal(t, "127.0.0.1", xfrState.client)
	require.Empty(t, xfrState.server)
	require.Zero(t, xfrState.messagesCount)
	require.Zero(t, xfrState.recordsCount)
	require.Zero(t, xfrState.bytesCount)
	require.Zero(t, xfrState.duration)
	require.Zero(t, xfrState.startTime)
	require.Zero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatBind9, xfrState.timeFormat)
	require.Equal(t, "setting up zone transfer: failed", xfrState.message)
}

// Test parsing the setting up zone transfer aborted message.
func TestXfrTrackerParseSettingUpZoneTransferAborted(t *testing.T) {
	xfrState := parseTransferLogLine("16-Apr-2026 12:09:11.650 client @0x7ffffa436000 127.0.0.1#55256 (bind.example.org): transfer of 'bind.example.org/IN': aborted")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusMessage, xfrState.status)
	require.Empty(t, xfrState.viewName)
	require.Equal(t, "bind.example.org", xfrState.zoneName)
	require.Zero(t, xfrState.serial)
	require.Equal(t, "127.0.0.1", xfrState.client)
	require.Empty(t, xfrState.server)
	require.Zero(t, xfrState.messagesCount)
	require.Zero(t, xfrState.recordsCount)
	require.Zero(t, xfrState.bytesCount)
	require.Zero(t, xfrState.duration)
	require.Zero(t, xfrState.startTime)
	require.Zero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatBind9, xfrState.timeFormat)
	require.Equal(t, "aborted", xfrState.message)
}

// Test parsing the systemd log line containing the AXFR started message.
func TestXfrTrackerParseSystemdLogs(t *testing.T) {
	xfrState := parseTransferLogLine("2026-04-21T20:38:52+02:00 lightning named[1411]: client @0x7ffffa436000 127.0.0.1#55256 (bind.example.org): transfer of 'bind.example.org/IN': AXFR started")
	require.NotNil(t, xfrState)
	require.Equal(t, xfrStatusStarted, xfrState.status)
	require.Empty(t, xfrState.viewName)
	require.Equal(t, "bind.example.org", xfrState.zoneName)
	require.Zero(t, xfrState.serial)
	require.Equal(t, "127.0.0.1", xfrState.client)
	require.Empty(t, xfrState.server)
	require.Zero(t, xfrState.messagesCount)
	require.Zero(t, xfrState.recordsCount)
	require.Zero(t, xfrState.bytesCount)
	require.Zero(t, xfrState.duration)
	require.NotZero(t, xfrState.startTime)
	require.Zero(t, xfrState.completionTime)
	require.Equal(t, xfrTimeFormatRFC3339, xfrState.timeFormat)
	require.Equal(t, "AXFR started", xfrState.message)
}

// Test feeding the XFR tracker with mixed log lines.
func TestXfrTrackerFeed(t *testing.T) {
	xfrTracker := newXfrTracker(nil)
	xfrMixedLogs := strings.Split(xfrMixedLogs, "\n")
	for _, logLine := range xfrMixedLogs {
		xfrTracker.feed(logLine)
	}

	// There is one failed zone transfer.
	notCompleted := xfrTracker.getNotCompleted()
	require.Len(t, notCompleted, 1)
	require.Equal(t, notCompleted[0].zoneName, "bind.example.org")
	require.Equal(t, notCompleted[0].status, xfrStatusMessage)
	require.Equal(t, notCompleted[0].client, "127.0.0.1")
	require.Equal(t, notCompleted[0].server, "")
	require.NotZero(t, notCompleted[0].startTime)
	require.Zero(t, notCompleted[0].completionTime)
	require.Equal(t, notCompleted[0].message, "setting up zone transfer: failed")

	// There are four completed zone transfers.
	completed := xfrTracker.getCompleted()
	require.Len(t, completed, 4)

	require.Equal(t, completed[0].zoneName, "drop.rpz.example.com")
	require.Equal(t, completed[0].status, xfrStatusCompleted)
	require.NotZero(t, completed[0].startTime)
	require.NotZero(t, completed[0].completionTime)
	require.Equal(t, completed[0].server, "172.24.0.53")
	require.EqualValues(t, completed[0].messagesCount, 1)
	require.EqualValues(t, completed[0].recordsCount, 5)
	require.EqualValues(t, completed[0].bytesCount, 294)
	require.Equal(t, completed[0].duration, 14*time.Millisecond)
	require.EqualValues(t, completed[0].serial, 201702121)
	require.Equal(t, completed[0].message, "Transfer completed: 1 messages, 5 records, 294 bytes, 0.014 secs (21000 bytes/sec) (serial 201702121)")

	require.Equal(t, completed[1].zoneName, ".")
	require.Equal(t, completed[1].status, xfrStatusCompleted)
	require.NotZero(t, completed[1].startTime)
	require.NotZero(t, completed[1].completionTime)
	require.Equal(t, completed[1].server, "192.5.5.241")
	require.EqualValues(t, completed[1].messagesCount, 123)
	require.EqualValues(t, completed[1].recordsCount, 24872)
	require.EqualValues(t, completed[1].bytesCount, 1321903)
	require.Equal(t, completed[1].duration, 335*time.Millisecond)
	require.EqualValues(t, completed[1].serial, 2026041600)
	require.Equal(t, completed[1].message, "Transfer completed: 123 messages, 24872 records, 1321903 bytes, 0.335 secs (3945979 bytes/sec) (serial 2026041600)")

	require.Equal(t, completed[2].zoneName, "drop.rpz.example.com")
	require.Equal(t, completed[2].status, xfrStatusCompleted)
	require.NotZero(t, completed[2].startTime)
	require.NotZero(t, completed[2].completionTime)
	require.EqualValues(t, completed[2].messagesCount, 1)
	require.EqualValues(t, completed[2].recordsCount, 5)
	require.EqualValues(t, completed[2].bytesCount, 199)
	require.Equal(t, completed[2].duration, 3*time.Millisecond)
	require.EqualValues(t, completed[2].serial, 201702121)
	require.Equal(t, completed[2].message, "AXFR ended: 1 messages, 5 records, 199 bytes, 0.003 secs (66333 bytes/sec) (serial 201702121)")

	require.Equal(t, completed[3].zoneName, ".")
	require.Equal(t, completed[3].status, xfrStatusCompleted)
	require.NotZero(t, completed[3].startTime)
	require.NotZero(t, completed[3].completionTime)
	require.EqualValues(t, completed[3].messagesCount, 79)
	require.EqualValues(t, completed[3].recordsCount, 24872)
	require.EqualValues(t, completed[3].bytesCount, 1320233)
	require.Equal(t, completed[3].duration, 52*time.Millisecond)
	require.EqualValues(t, completed[3].serial, 2026041600)
	require.Equal(t, completed[3].message, "AXFR ended: 79 messages, 24872 records, 1320233 bytes, 0.052 secs (25389096 bytes/sec) (serial 2026041600)")
}

// Test feeding the XFR tracker with started and connected log lines.
// It is expected that the connected message is used to update the server
// address.
func TestXfrTrackerFeedStartedConnected(t *testing.T) {
	xfrTracker := newXfrTracker(nil)
	xfrTracker.feed("23-Feb-2026 10:41:27.071 zone ./IN: Transfer started.")
	xfrTracker.feed("23-Feb-2026 10:41:27.141 @0x7ffffaa28c00 172.24.0.54#34961: transfer of './IN': connected using 172.24.0.54#34961")
	transfers := xfrTracker.getNotCompleted()
	require.Len(t, transfers, 1)
	require.Equal(t, transfers[0].zoneName, ".")
	require.Equal(t, transfers[0].status, xfrStatusConnected)
	require.Equal(t, transfers[0].server, "172.24.0.54")
}

// Test that when the connected message appears before the started message, the
// connected state is recorded.
func TestXfrTrackerFeedConnectedBeforeStarted(t *testing.T) {
	xfrTracker := newXfrTracker(nil)
	xfrTracker.feed("23-Feb-2026 10:41:27.141 @0x7ffffaa28c00 172.24.0.54#34961: transfer of './IN': connected using 172.24.0.54#34961")
	transfers := xfrTracker.getNotCompleted()
	require.Len(t, transfers, 1)
	require.Equal(t, transfers[0].zoneName, ".")
	require.Equal(t, transfers[0].status, xfrStatusConnected)
	require.Equal(t, transfers[0].server, "172.24.0.54")
	require.Zero(t, transfers[0].startTime)
}

// Test that the completed message overrides the started message but the start time is preserved.
func TestXfrTrackerFeedStartedCompleted(t *testing.T) {
	xfrTracker := newXfrTracker(nil)
	xfrTracker.feed("23-Feb-2026 10:41:27.071 zone ./IN: Transfer started.")
	xfrTracker.feed("23-Feb-2026 10:41:27.141 @0x7ffffaa28c00 172.24.0.54#34961: transfer of './IN': AXFR ended")
	transfers := xfrTracker.getNotCompleted()
	require.Empty(t, transfers)
	completed := xfrTracker.getCompleted()
	require.Len(t, completed, 1)
	require.Equal(t, completed[0].zoneName, ".")
	require.Equal(t, completed[0].status, xfrStatusCompleted)
	require.NotZero(t, completed[0].startTime)
	require.NotZero(t, completed[0].completionTime)
}

// Test that the completed message is recorded even if there is no corresponding started message.
func TestXfrTrackerFeedCompletedBeforeStarted(t *testing.T) {
	xfrTracker := newXfrTracker(nil)
	xfrTracker.feed("23-Feb-2026 10:41:27.141 @0x7ffffaa28c00 172.24.0.54#34961: transfer of 'example.com/IN': AXFR ended")
	transfers := xfrTracker.getNotCompleted()
	require.Empty(t, transfers)
	completed := xfrTracker.getCompleted()
	require.Len(t, completed, 1)
	require.Equal(t, completed[0].zoneName, "example.com")
	require.Equal(t, completed[0].status, xfrStatusCompleted)
	require.Zero(t, completed[0].startTime, time.Time{})
	require.NotZero(t, completed[0].completionTime)
}

// Test the limits of the XFR tracker containers.
func TestXfrTrackerLimits(t *testing.T) {
	xfrTracker := newXfrTracker(nil)
	xfrTracker.maxStates = 3
	startedTemplate := "23-Feb-2026 10:41:27.141 @0x7ffffaa28c00 172.24.0.54#34961: transfer of '%d.example.com/IN': AXFR started"
	for i := 0; i < xfrTracker.maxStates; i++ {
		xfrTracker.feed(fmt.Sprintf(startedTemplate, i))
		notCompleted := xfrTracker.getNotCompleted()
		require.Len(t, notCompleted, i+1)
		completed := xfrTracker.getCompleted()
		require.Empty(t, completed)
	}

	xfrTracker.feed(fmt.Sprintf(startedTemplate, 3))
	notCompleted := xfrTracker.getNotCompleted()
	require.Len(t, notCompleted, 3)
	require.Equal(t, notCompleted[0].zoneName, "1.example.com")
	require.Equal(t, notCompleted[1].zoneName, "2.example.com")
	require.Equal(t, notCompleted[2].zoneName, "3.example.com")
	require.Empty(t, xfrTracker.getCompleted())

	xfrTracker.feed(fmt.Sprintf(startedTemplate, 4))
	notCompleted = xfrTracker.getNotCompleted()
	require.Len(t, notCompleted, 3)
	require.Equal(t, notCompleted[0].zoneName, "2.example.com")
	require.Equal(t, notCompleted[1].zoneName, "3.example.com")
	require.Equal(t, notCompleted[2].zoneName, "4.example.com")
	require.Empty(t, xfrTracker.getCompleted())

	endedTemplate := "23-Feb-2026 10:41:27.141 @0x7ffffaa28c00 172.24.0.54#34961: transfer of '%d.example.com/IN': AXFR completed"
	xfrTracker.feed(fmt.Sprintf(endedTemplate, 3))
	completed := xfrTracker.getCompleted()
	require.Len(t, completed, 1)
	require.Equal(t, completed[0].zoneName, "3.example.com")
	notCompleted = xfrTracker.getNotCompleted()
	require.Len(t, notCompleted, 2)
	require.Equal(t, notCompleted[0].zoneName, "2.example.com")
	require.Equal(t, notCompleted[1].zoneName, "4.example.com")

	xfrTracker.feed(fmt.Sprintf(endedTemplate, 2))
	completed = xfrTracker.getCompleted()
	require.Len(t, completed, 2)
	require.Equal(t, completed[0].zoneName, "3.example.com")
	require.Equal(t, completed[1].zoneName, "2.example.com")
	notCompleted = xfrTracker.getNotCompleted()
	require.Len(t, notCompleted, 1)
	require.Equal(t, notCompleted[0].zoneName, "4.example.com")

	xfrTracker.feed(fmt.Sprintf(startedTemplate, 5))
	notCompleted = xfrTracker.getNotCompleted()
	require.Len(t, notCompleted, 2)
	require.Equal(t, notCompleted[0].zoneName, "4.example.com")
	require.Equal(t, notCompleted[1].zoneName, "5.example.com")
	completed = xfrTracker.getCompleted()
	require.Len(t, completed, 2)

	xfrTracker.feed(fmt.Sprintf(endedTemplate, 5))
	completed = xfrTracker.getCompleted()
	require.Len(t, completed, 3)
	require.Equal(t, completed[0].zoneName, "3.example.com")
	require.Equal(t, completed[1].zoneName, "2.example.com")
	require.Equal(t, completed[2].zoneName, "5.example.com")
	notCompleted = xfrTracker.getNotCompleted()
	require.Len(t, notCompleted, 1)
	require.Equal(t, notCompleted[0].zoneName, "4.example.com")

	xfrTracker.feed(fmt.Sprintf(endedTemplate, 4))
	completed = xfrTracker.getCompleted()
	require.Len(t, completed, 3)
	require.Equal(t, completed[0].zoneName, "2.example.com")
	require.Equal(t, completed[1].zoneName, "5.example.com")
	require.Equal(t, completed[2].zoneName, "4.example.com")
	notCompleted = xfrTracker.getNotCompleted()
	require.Empty(t, notCompleted)
}

// Fuzz the XFR tracker parse function to ensure that it does not panic
// for invalid log lines.
func FuzzXfrTrackerParse(f *testing.F) {
	logs := strings.Split(xfrMixedLogs, "\n")
	for _, tc := range logs {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, logLine string) {
		require.NotPanics(t, func() { parseTransferLogLine(logLine) })
	})
}

// Benchmark the XFR tracker parse function.
// During the XFR tracker implementation, the benchmark returned the following results
// on the MacBook pro with Apple M3 chip, 36GB RAM:
//
// BenchmarkXfrTrackerParse-12    72302    16684 ns/op    2288 B/op   12 allocs/op.
func BenchmarkXfrTrackerParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xfrState := parseTransferLogLine("23-Feb-2026 10:41:27.141 client @0x7ffffaa28c00 172.24.0.54#34961/key trusted-key (drop.rpz.example.com): view trusted: transfer of 'drop.rpz.example.com/IN': AXFR ended: 1 messages, 5 records, 294 bytes, 0.004 secs (73500 bytes/sec) (serial 201702121)")
		require.NotNil(b, xfrState)
	}
}
