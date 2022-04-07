package testutil

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test that capture output restores the stdout and stderr
// to original values.
func TestCaptureOutputRestoreStdoutAndStderr(t *testing.T) {
	// Arrange
	orgStdout := os.Stdout
	orgStderr := os.Stderr

	// Act
	_, _, err := CaptureOutput(func() {}, nil, 0)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, orgStdout, os.Stdout)
	require.EqualValues(t, orgStderr, os.Stderr)
}

// Test that the stdout is captured.
func TestCaptureOutputReadStdout(t *testing.T) {
	// Act
	stdout, stderr, err := CaptureOutput(func() {
		fmt.Print("foo")
		time.Sleep(10 * time.Millisecond)
		fmt.Print("bar")
		time.Sleep(10 * time.Millisecond)
		fmt.Print("!")
	}, nil, 0)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, []byte("foobar!"), stdout)
	require.Len(t, stderr, 0)
}

// Test that the stderr is captured.
func TestCaptureOutputReadStderr(t *testing.T) {
	// Act
	stdout, stderr, err := CaptureOutput(func() {
		fmt.Fprint(os.Stderr, "foo")
	}, nil, 0)

	// Assert
	require.NoError(t, err)
	require.Len(t, stdout, 0)
	require.EqualValues(t, "foo", string(stderr))
}

// Test that the chunk function is called.
func TestCaptureOutputChunkCallback(t *testing.T) {
	// Arrange
	f := func() {
		fmt.Print("foo")
		time.Sleep(100 * time.Millisecond)
		fmt.Print("bar")
		time.Sleep(100 * time.Millisecond)
		fmt.Print("!")
	}

	totalBytes := 0
	chunk := func(stdout []byte, n int) {
		totalBytes += n
		require.EqualValues(t, len(stdout), totalBytes)
	}

	// Act
	stdout, _, _ := CaptureOutput(f, chunk, 3)

	// Assert
	require.EqualValues(t, 7, totalBytes)
	require.EqualValues(t, "foobar!", string(stdout))
}
