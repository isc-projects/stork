package storkutil

import (
	"bufio"
	"context"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that the system command executor command returns the correct scanner.
func TestSystemCommandExecutorCommandGetScanner(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("test"))
	command := &systemCommandExecutorOutput{
		scanner: scanner,
	}
	require.Equal(t, scanner, command.GetScanner())
}

// Test that the system command executor is constructed properly.
func TestNewSystemCommandExecutor(t *testing.T) {
	// Arrange & Act
	executor := NewSystemCommandExecutor()

	// Assert
	require.NotNil(t, executor)

	lsPath, err := executor.LookPath("ls")
	require.NotNil(t, lsPath)
	require.Nil(t, err)
	require.True(t, executor.IsFileExist(lsPath))
	sb := testutil.NewSandbox()
	defer sb.Close()
	require.False(t, executor.IsFileExist(path.Join(sb.BasePath, "not-exists")))
}

// Test that the executor returns coorrect file info for a file.
func TestGetFileInfo(t *testing.T) {
	executor := NewSystemCommandExecutor()
	sb := testutil.NewSandbox()
	defer sb.Close()
	_, err := sb.Write("test.txt", "test")
	require.NoError(t, err)

	info, err := executor.GetFileInfo(filepath.Join(sb.BasePath, "test.txt"))
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, info.Name(), "test.txt")
	require.Equal(t, info.IsDir(), false)
	require.Equal(t, info.Size(), int64(4))
	require.WithinDuration(t, info.ModTime(), time.Now(), 10*time.Second)
}

// Test that the executor returns an error when attempting to get the file
// information for a non-existent file.
func TestGetFileInfoNotFound(t *testing.T) {
	executor := NewSystemCommandExecutor()
	sb := testutil.NewSandbox()
	defer sb.Close()

	info, err := executor.GetFileInfo(filepath.Join(sb.BasePath, "not-exists"))
	require.ErrorContains(t, err, "cannot get file info")
	require.Nil(t, info)
}

// Test that the executor can start a blocking command, and that the command can
// be cancelled using the context. Note that this command relies on the existence
// of the "tail" command in the system. If the "tail" command is not available,
// the test is skipped.
func TestSystemCommandStartScanCancel(t *testing.T) {
	executor := NewSystemCommandExecutor()
	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a file with some content.
	_, err := sb.Write("test.txt", "test\n")
	require.NoError(t, err)

	// Try to run the blocking tail command.
	ctx, cancel := context.WithCancel(context.Background())
	output, err := executor.Start(ctx, "tail", "-f", filepath.Join(sb.BasePath, "test.txt"))
	if err != nil {
		// If tail is not available, skip the test. It should not be the case in  most of the
		// systems, but still possible in some minimal environments. We don't want to fail
		// the test in such a case.
		t.Skip("tail command is not available")
	}
	require.NotNil(t, output)

	captured := atomic.Bool{}
	done := atomic.Bool{}

	// Start a goroutine that reads the output of the command.
	go func() {
		for output.GetScanner().Scan() {
			// The tail command should capture the content present in the file.
			// Let's signal to the main thread that the content is captured,
			// so the main thread can now cancel the command.
			// Subsequent scans should block.
			captured.Store(true)
		}
		output.Wait()
		done.Store(true)
	}()

	// Make sure that the content was captured but the goroutine is still running.
	require.Eventually(t, func() bool {
		return captured.Load() && !done.Load()
	}, 10*time.Second, 100*time.Millisecond)

	// Cancel the command.
	cancel()

	// The goroutine should exit because the tail command was cancelled
	// and the scanner is closed.
	require.Eventually(t, done.Load, 10*time.Second, 100*time.Millisecond)
}
