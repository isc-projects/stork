package agent

import (
	"bufio"
	"context"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"isc.org/stork/testutil"
)

// Test that the text file log reader is constructed properly.
func TestNewTextFileLogReader(t *testing.T) {
	reader := newTextFileLogReader("test.log", textLogReaderConfig{
		poll: true,
	})
	require.NotNil(t, reader)
	require.True(t, reader.isSupported())
	require.Equal(t, "test.log", reader.path)
	require.True(t, reader.config.poll)
}

// Test that the log reader can be hooked up at the start of the file, read the file
// contents, and follow the new lines.
func TestTextFileLogReaderFollowFromStart(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Write the first log line to a file.
	_, err := sandbox.Write("test.log", "This is the first log line\n")
	require.NoError(t, err)

	// Create text file log reader. It is necessary to poll the file to ensure test reliability
	// on various systems (macOS, BSD). On these systems kqueue is used to monitor the file changes.
	// It can miss file changes, especially when the temporary files (sandbox) are written to /tmp.
	reader := newTextFileLogReader(filepath.Join(sandbox.BasePath, "test.log"), textLogReaderConfig{
		poll: true,
	})
	require.NotNil(t, reader)

	// Create the context with cancellation to make sure that the read is stopped when the test is done.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Seek to the start of the file and follow the new lines. It should return the existing
	// first log line.
	lines, err := reader.capture(ctx, logReaderCaptureOptionFollow())
	require.NoError(t, err)
	require.NotNil(t, lines)

	// Protect the captured lines with a mutex to avoid race conditions.
	mutex := sync.RWMutex{}
	capturedLines := make([]string, 0)

	// Start the goroutine that read the lines from the channel in background.
	go func() {
		for {
			select {
			case line, ok := <-lines:
				// Read the lines from the file.
				if !ok {
					return
				}
				mutex.Lock()
				capturedLines = append(capturedLines, line.text)
				mutex.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for the goroutine to receive the first line.
	require.Eventually(t, func() bool {
		mutex.RLock()
		defer mutex.RUnlock()
		return len(capturedLines) > 0
	}, 10*time.Second, 100*time.Millisecond)

	// Append a new line to the file.
	_, err = sandbox.Append("test.log", "This is the second log line\n")
	require.NoError(t, err)

	// Wait for the goroutine to read the new line to up to 10 seconds.
	require.Eventually(t, func() bool {
		mutex.RLock()
		defer mutex.RUnlock()
		return len(capturedLines) > 1
	}, 10*time.Second, 100*time.Millisecond)

	// Ensure that the goroutine has read the existing and the new line.
	require.Len(t, capturedLines, 2)
	require.Equal(t, "This is the first log line", capturedLines[0])
	require.Equal(t, "This is the second log line", capturedLines[1])
}

// Test that the log reader can be hooked up at the start of the file and read the file
// contents.
func TestTextFileLogReaderReadFromStart(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Write the first log line to a file.
	_, err := sandbox.Write("test.log", "This is the first log line\nThis is the second log line\n")
	require.NoError(t, err)

	// Create text file log reader.
	reader := newTextFileLogReader(filepath.Join(sandbox.BasePath, "test.log"), textLogReaderConfig{})
	require.NotNil(t, reader)

	// Create the context with cancellation to make sure that the read is stopped when the test is done.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Read the file contents.
	lines, err := reader.capture(ctx)
	require.NoError(t, err)
	require.NotNil(t, lines)

	// Protect the captured lines with a mutex to avoid race conditions.
	capturedLines := make([]string, 0)

	// Start the goroutine that reads the lines from the channel in background.
	done := atomic.Bool{}
	go func() {
		for line := range lines {
			capturedLines = append(capturedLines, line.text)
		}
		done.Store(true)
	}()
	require.Eventually(t, done.Load, 10*time.Second, 100*time.Millisecond)

	// Ensure that the goroutine has read the existing and the new line.
	require.Len(t, capturedLines, 2)
	require.Equal(t, "This is the first log line", capturedLines[0])
	require.Equal(t, "This is the second log line", capturedLines[1])
}

// Test that the log reader can be hooked up at the end of the file and follow the new lines.
func TestTextFileLogReaderFollowFromEnd(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Write the first log line to a file.
	_, err := sandbox.Write("test.log", "This is the first log line\n")
	require.NoError(t, err)

	// Create text file log reader. It is necessary to poll the file to ensure test reliability
	// on various systems (macOS, BSD). On these systems kqueue is used to monitor the file changes.
	// It can miss file changes, especially when the temporary files (sandbox) are written to /tmp.
	reader := newTextFileLogReader(filepath.Join(sandbox.BasePath, "test.log"), textLogReaderConfig{
		poll: true,
	})
	require.NotNil(t, reader)

	// Create the context with cancellation to make sure that the read is stopped when the test is done.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Seek to the end of the file and follow the new lines. It should skip the existing
	// first log line.
	lines, err := reader.capture(ctx, logReaderCaptureOptionFromEnd(), logReaderCaptureOptionFollow())
	require.NoError(t, err)
	require.NotNil(t, lines)

	// Protect the captured lines with a mutex to avoid race conditions.
	mutex := sync.RWMutex{}
	capturedLines := make([]string, 0)

	// Create a blocking channel to stop the test until we ensure that the goroutine
	// picks it up. It means that it is up-and-running and reads lines.
	ch := make(chan struct{})
	defer close(ch)

	// Start the goroutine that reads the lines from the channel in background.
	go func() {
		for {
			select {
			case line, ok := <-lines:
				// Read the lines from the file.
				if !ok {
					return
				}
				mutex.Lock()
				capturedLines = append(capturedLines, line.text)
				mutex.Unlock()
			case _, ok := <-ch:
				// Main thread is signalling that it is ready to append new lines
				// to the file. Reading from this channel here unblocks the main thread,
				// and ensures that this goroutine is up-and-running and reads lines.
				if !ok {
					return
				}
			}
		}
	}()

	// Blocking write to a channel to signal that the main thread is ready to append new lines.
	// It will be unblocked when the goroutine reads the channel.
	ch <- struct{}{}

	// Wait for the goroutine to read the new line to up to 10 seconds.
	require.Eventually(t, func() bool {
		// Keep appending the new lines to the file until tail picks them up.
		// A single append does not guarantee that tail picks them because the
		// file watcher may not be fully started yet. There is no mechanism in the
		// tail package to check if the file watcher is fully started. If we append
		// to the file before each check, it guarantees that one of the appends will
		// finally be noticed.
		_, err = sandbox.Append("test.log", "This is the second log line\n")
		require.NoError(t, err)
		mutex.RLock()
		defer mutex.RUnlock()
		return len(capturedLines) > 0
	}, 10*time.Second, 100*time.Millisecond)

	// Ensure that the goroutine has read the new line. The first line must not be
	// captured because we're following the new lines from the end of the file.
	require.NotEmpty(t, capturedLines)
	require.Equal(t, "This is the second log line", capturedLines[0])
}

// Test that an error is returned upon trying to read from the end of the file without following.
func TestTextFileLogReaderReadFromEnd(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Write to a log file, so it is non-empty.
	_, err := sandbox.Write("test.log", "This is the first log line\nThis is the second log line\n")
	require.NoError(t, err)

	// Create text file log reader.
	reader := newTextFileLogReader(filepath.Join(sandbox.BasePath, "test.log"), textLogReaderConfig{})
	require.NotNil(t, reader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Try to read the file contents from the end without following. It should return an error.
	lines, err := reader.capture(ctx, logReaderCaptureOptionFromEnd())
	require.ErrorContains(t, err, "cannot read from the end of the file without following")
	require.Nil(t, lines)
}

// Test that the log reader can be cancelled.
func TestTextFileLogReaderCaptureCancel(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Write the first log line to a file.
	_, err := sandbox.Write("test.log", "This is the first log line\nThis is the second log line\n")
	require.NoError(t, err)

	// Create text file log reader.
	reader := newTextFileLogReader(filepath.Join(sandbox.BasePath, "test.log"), textLogReaderConfig{
		poll: true,
	})
	require.NotNil(t, reader)

	// Create the context with cancellation to make sure that the read is stopped when the test is done.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Read the file contents.
	lines, err := reader.capture(ctx, logReaderCaptureOptionFollow())
	require.NoError(t, err)
	require.NotNil(t, lines)

	// Read the first line.
	line := <-lines
	require.Equal(t, "This is the first log line", line.text)

	// Cancel reading the second line.
	cancel()

	// It should stop reading.
	require.Eventually(t, func() bool {
		_, ok := <-lines
		// Ensure that the channel is closed.
		return !ok
	}, 10*time.Second, 100*time.Millisecond)
}

// Test that the log reader can be hooked up when the file does not exist yet.
func TestTextFileLogReaderFollowBeforeFileExists(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create text file log reader. It is necessary to poll the file to ensure test reliability
	// on various systems (macOS, BSD). On these systems kqueue is used to monitor the file changes.
	// It can miss file changes, especially when the temporary files (sandbox) are written to /tmp.
	reader := newTextFileLogReader(filepath.Join(sandbox.BasePath, "test.log"), textLogReaderConfig{
		poll: true,
	})
	require.NotNil(t, reader)

	// Create the context with cancellation to make sure that the read is stopped when the test is done.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Follow the new lines.
	lines, err := reader.capture(ctx, logReaderCaptureOptionFollow())
	require.NoError(t, err)
	require.NotNil(t, lines)

	// Protect the captured lines with a mutex to avoid race conditions.
	mutex := sync.RWMutex{}
	capturedLines := make([]string, 0)

	// Create a blocking channel to stop the test until we ensure that the goroutine
	// picks it up. It means that it is up-and-running and reads lines.
	ch := make(chan struct{})
	defer close(ch)

	// Start the goroutine that reads the lines from the channel in background.
	go func() {
		for {
			select {
			case line, ok := <-lines:
				// Read the lines from the file.
				if !ok {
					return
				}
				mutex.Lock()
				capturedLines = append(capturedLines, line.text)
				mutex.Unlock()
			case _, ok := <-ch:
				// Main thread is signalling that it is ready to create the file.
				// Reading from this channel here unblocks the main thread, and
				// ensures that this goroutine is up-and-running and reads lines.
				if !ok {
					return
				}
			}
		}
	}()

	// Blocking write to a channel to signal that the main thread is ready to create the file.
	// It will be unblocked when the goroutine reads the channel.
	ch <- struct{}{}

	_, err = sandbox.Write("test.log", "This is the first log line\nThis is the second log line\n")
	require.NoError(t, err)

	// Wait for the goroutine to read the new lines to up to 10 seconds.
	require.Eventually(t, func() bool {
		mutex.RLock()
		defer mutex.RUnlock()
		return len(capturedLines) > 0
	}, 10*time.Second, 100*time.Millisecond)

	// Ensure that the goroutine has read the file.
	require.Len(t, capturedLines, 2)
	require.Equal(t, "This is the first log line", capturedLines[0])
	require.Equal(t, "This is the second log line", capturedLines[1])
}

// Test that the text file log reader is constructed properly.
func TestNewSystemdLogReader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock command executor.
	executor := NewMockCommandExecutor(ctrl)
	// Make sure that the journalctl command is found.
	executor.EXPECT().LookPath("journalctl").Return("journalctl", nil)

	// Create the systemd log reader.
	reader := newSystemdLogReader(executor)
	require.NotNil(t, reader)
	require.True(t, reader.isSupported())
}

// Test that the systemd log reader is not supported when the journalctl command is not found.
func TestUnsupportedSystemdLogReader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock command executor.
	executor := NewMockCommandExecutor(ctrl)
	// Make sure that the journalctl command is not found.
	executor.EXPECT().LookPath("journalctl").Return("", errors.New("command not found"))

	// Create the systemd log reader.
	reader := newSystemdLogReader(executor)
	require.NotNil(t, reader)
	require.False(t, reader.isSupported())
}

// Test cases checking that correct journalctl command line is issued according to
// the specified capture options.
func TestSystemdFileLogReaderOptions(t *testing.T) {
	testCases := []struct {
		name         string
		options      []logReaderCaptureOption
		expectedArgs []any
	}{
		{
			name:         "follow with unit name",
			options:      []logReaderCaptureOption{logReaderCaptureOptionFollow(), logReaderCaptureOptionUnitName("named.service")},
			expectedArgs: []any{"-f", "-u", "named.service", "--no-tail", "-n", "+1"},
		},
		{
			name:         "follow from end",
			options:      []logReaderCaptureOption{logReaderCaptureOptionFromEnd(), logReaderCaptureOptionFollow()},
			expectedArgs: []any{"-f", "-n", "0"},
		},
		{
			name:         "follow since days ago",
			options:      []logReaderCaptureOption{logReaderCaptureOptionSinceDaysAgo(4), logReaderCaptureOptionFollow()},
			expectedArgs: []any{"-f", "--since", "4 days ago"},
		},
		{
			name:         "follow from start",
			options:      []logReaderCaptureOption{logReaderCaptureOptionFollow()},
			expectedArgs: []any{"-f", "--no-tail", "-n", "+1"},
		},
		{
			name:         "read since days ago",
			options:      []logReaderCaptureOption{logReaderCaptureOptionSinceDaysAgo(4)},
			expectedArgs: []any{"--since", "4 days ago"},
		},
		{
			name:         "read from start",
			options:      []logReaderCaptureOption{},
			expectedArgs: []any{"--no-tail", "-n", "+1"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create the context with cancellation to make sure that the read is stopped when the test is done.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create a scanner that returns the log lines.
			scanner := bufio.NewScanner(strings.NewReader("This is the first log line\nThis is the second log line\n"))
			// Wrap the scanner in the command executor command.
			command := NewMockCommandExecutorCommand(ctrl)
			// Make the scanner available to the reader.
			command.EXPECT().GetScanner().AnyTimes().Return(scanner)
			// Make sure that the command is waited after the scanner is closed.
			command.EXPECT().Wait().Return(nil)

			// Create a mock command executor.
			executor := NewMockCommandExecutor(ctrl)
			// Make sure that the correct command line switches were used for the
			// specified capture options.
			executor.EXPECT().Start(ctx, "journalctl", testCase.expectedArgs...).Return(command, nil)

			// Create the systemd log reader.
			reader := newSystemdLogReader(executor)
			require.NotNil(t, reader)

			// Start capturing the log lines.
			lines, err := reader.capture(ctx, testCase.options...)
			require.NoError(t, err)
			require.NotNil(t, lines)

			// Protect the captured lines with a mutex to avoid race conditions.
			mutex := sync.RWMutex{}
			capturedLines := make([]string, 0)

			// Start the goroutine that read the lines from the channel in background.
			go func() {
				for {
					select {
					case line, ok := <-lines:
						// Read the lines from the channel.
						if !ok {
							return
						}
						mutex.Lock()
						capturedLines = append(capturedLines, line.text)
						mutex.Unlock()
					case <-ctx.Done():
						return
					}
				}
			}()

			// Wait for the goroutine to read the log lines for up to 10 seconds.
			require.Eventually(t, func() bool {
				mutex.RLock()
				defer mutex.RUnlock()
				return len(capturedLines) > 1
			}, 10*time.Second, 100*time.Millisecond)

			// Ensure that the goroutine has read the log lines.
			require.Len(t, capturedLines, 2)
			require.Equal(t, "This is the first log line", capturedLines[0])
			require.Equal(t, "This is the second log line", capturedLines[1])
		})
	}
}

// Test that an error is returned upon trying to read from the end of the systemd log
// without following.
func TestSystemdLogReaderReadFromEnd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock command executor.
	executor := NewMockCommandExecutor(ctrl)
	require.NotNil(t, executor)

	// Create the systemd log reader.
	reader := newSystemdLogReader(executor)
	require.NotNil(t, reader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Try to read the log contents from the end without following. It should return an error.
	lines, err := reader.capture(ctx, logReaderCaptureOptionFromEnd())
	require.ErrorContains(t, err, "cannot read from the end of the systemd log without following")
	require.Nil(t, lines)
}
