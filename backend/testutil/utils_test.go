package testutil

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Test that capture output restores the stdout and stderr
// to original values.
func TestCaptureOutputRestoreStdoutAndStderr(t *testing.T) {
	// Arrange
	orgStdout := os.Stdout
	orgStderr := os.Stderr
	orgLogrus := logrus.StandardLogger().Out

	// Act
	_, _, err := CaptureOutput(func() {})

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, orgStdout, os.Stdout)
	require.EqualValues(t, orgStderr, os.Stderr)
	require.EqualValues(t, orgLogrus, logrus.StandardLogger().Out)
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
	})

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
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, stdout, 0)
	require.EqualValues(t, "foo", string(stderr))
}

// Test that the log output is captured.
func TestCaptureOutputReadLog(t *testing.T) {
	// Act
	stdout, stderr, err := CaptureOutput(func() {
		logrus.Info("foo")
	})

	// Assert
	require.NoError(t, err)
	require.Contains(t, string(stdout), "foo")
	require.Len(t, stderr, 0)
}
