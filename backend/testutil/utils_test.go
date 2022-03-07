package testutil

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that capture output restores the stdout and stderr
// to original values.
func TestCaptureOutputRestoreStdoutAndStderr(t *testing.T) {
	// Arrange
	orgStdout := os.Stdout
	orgStderr := os.Stderr

	// Act
	_, _, err := CaptureOutput(func() {})

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
	})

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, []byte("foo"), stdout)
	require.Len(t, stderr, 0)
}

// Test that the stderr is captured.
func TestCaptureOutputReadStderr(t *testing.T) {
	// Act
	stdout, stderr, err := CaptureOutput(func() {
		fmt.Fprint(os.Stderr, "foo")
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, stdout, 0)
	require.EqualValues(t, []byte("foo"), stderr)
}
