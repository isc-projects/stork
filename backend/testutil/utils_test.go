package testutil

import (
	"fmt"
	"net"
	"os"
	"strings"
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
		logrus.Info("Foo")
	})

	// Assert
	require.NoError(t, err)
	require.Contains(t, string(stdout), "Foo")
	require.Len(t, stderr, 0)
}

// Function for a valid timestamp suffix should return no error.
func TestParseTimestampFilenameNoErrorForValid(t *testing.T) {
	// Arrange
	timestamp := time.Date(2022, 5, 20, 12, 7, 0, 0, time.UTC).Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("foo_%s.ext", timestamp)

	// Act
	prefix, parsedTimestamp, extension, err := ParseTimestampFilename(filename)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, time.Date(2022, 5, 20, 12, 7, 0, 0, time.UTC), parsedTimestamp)
	require.EqualValues(t, "foo_", prefix)
	require.EqualValues(t, ".ext", extension)
}

// Function for a missing delimiter in filename should return error.
func TestParseTimestampFilenameErrorForNoDelimiter(t *testing.T) {
	// Arrange
	timestamp := time.Date(2022, 5, 20, 12, 7, 0, 0, time.UTC).Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("foo%s.ext", timestamp)

	// Act
	prefix, timestampObj, extension, err := ParseTimestampFilename(filename)

	// Assert
	require.Error(t, err)
	require.Empty(t, prefix)
	require.Zero(t, timestampObj)
	require.Empty(t, extension)
}

// Function for a invalid timestamp should return error.
func TestParseTimestampFilenameErrorForInvalid(t *testing.T) {
	// Arrange
	timestamp := "bar"
	filename := fmt.Sprintf("foo_%s.ext", timestamp)

	// Act
	prefix, timestampObj, extension, err := ParseTimestampFilename(filename)

	// Assert
	require.Error(t, err)
	require.NotEmpty(t, prefix)
	require.NotNil(t, timestampObj)
	require.NotEmpty(t, extension)
}

// Function for too short timestamp should return error.
func TestParseTimestampFilenameTooShort(t *testing.T) {
	// Arrange
	timestamp := "2021-11-15T12:00:00"
	filename := fmt.Sprintf("foo_%s.ext", timestamp)

	// Act
	prefix, timestampObj, extension, err := ParseTimestampFilename(filename)

	// Assert
	require.Error(t, err)
	require.EqualValues(t, "foo_", prefix)
	require.NotNil(t, timestampObj)
	require.EqualValues(t, ".ext", extension)
}

// Function for a valid timestamp should return prefix and extension of filename.
func TestParseTimestampFilenamePrefixOfFilenameForValid(t *testing.T) {
	// Arrange
	timestamp := time.Time{}.Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("foo-bar_%s.ext", timestamp)

	// Act
	prefix, _, extension, err := ParseTimestampFilename(filename)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "foo-bar_", prefix)
	require.EqualValues(t, ".ext", extension)
}

// Function for a valid filename should return the parsed timestamp.
func TestParseTimestampFilenameTimestampForValid(t *testing.T) {
	// Arrange
	timestamp := time.Date(2022, 5, 20, 12, 7, 0, 0, time.UTC).Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("foo_%s.ext", timestamp)

	// Act
	_, timestampObj, extension, err := ParseTimestampFilename(filename)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, time.Date(2022, 5, 20, 12, 7, 0, 0, time.UTC), timestampObj)
	require.EqualValues(t, ".ext", extension)
}

// Function for a double extension should return a full extension.
func TestParseTimestampFilenameDoubleExtension(t *testing.T) {
	// Arrange
	timestamp := time.Time{}.Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("foo_%s.bar.baz", timestamp)

	// Act
	_, _, extension, err := ParseTimestampFilename(filename)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, ".bar.baz", extension)
}

// Test that the restore point clears the environment variables.
func TestCreateEnvironmentRestorePoint(t *testing.T) {
	// Arrange
	os.Unsetenv("STORK_TEST_KEY1")
	os.Setenv("STORK_TEST_KEY2", "foo")
	os.Setenv("STORK_TEST_KEY3", "bar")

	// Act
	restore := CreateEnvironmentRestorePoint()
	os.Setenv("STORK_TEST_KEY1", "baz")
	os.Unsetenv("STORK_TEST_KEY2")
	os.Setenv("STORK_TEST_KEY3", "boz")
	restore()

	// Assert
	_, existKey1 := os.LookupEnv("STORK_TEST_KEY1")
	require.False(t, existKey1)

	value2 := os.Getenv("STORK_TEST_KEY2")
	require.EqualValues(t, "foo", value2)

	value3 := os.Getenv("STORK_TEST_KEY3")
	require.EqualValues(t, "bar", value3)
}

// Test that the restore point clears the OS arguments.
func TestCreateOsArgsRestorePoint(t *testing.T) {
	// Arrange & Act
	restorePoint := CreateOsArgsRestorePoint()

	os.Args = []string{
		"program-name",
		"foobar",
	}

	restorePoint()

	// Assert
	require.NotContains(t, os.Args, "foobar")
}

// Test that the free TCP port is returned properly.
func TestGetFreeLocalTCPPort(t *testing.T) {
	// Arrange & Act
	port, err := GetFreeLocalTCPPort()

	// Assert
	require.NoError(t, err)
	require.NotZero(t, port)
	// Check that the port is not in use.
	addr := net.JoinHostPort("localhost", fmt.Sprint(port))
	listener, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	listener.Close()
}
