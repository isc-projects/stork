package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"isc.org/stork"
	"isc.org/stork/testutil"
)

// Aux function checks if a list of expected strings is present in the string.
func checkOutput(output string, exp []string, reason string) bool {
	for _, x := range exp {
		fmt.Printf("Checking if %s exists in %s.\n", x, reason)
		if !strings.Contains(output, x) {
			fmt.Printf("ERROR: Expected string [%s] not found in %s.\n", x, reason)
			return false
		}
	}
	return true
}

// This is the list of all parameters we expect to be supported by stork-agent.
func getExpectedSwitches() []string {
	return []string{
		"-v", "--version", "--listen-prometheus-only", "--listen-stork-only",
		"--host", "--port", "--prometheus-kea-exporter-address", "--prometheus-kea-exporter-port",
		"--prometheus-kea-exporter-interval", "--prometheus-bind9-exporter-address",
		"--prometheus-bind9-exporter-port", "--prometheus-bind9-exporter-interval",
	}
}

// This is the list of all register parameters we expect to be supported by stork-agent.
func getExpectedRegisterSwitches() []string {
	return []string{
		"-u", "--server-url",
		"-t", "--server-token", "-a", "--agent-host",
	}
}

// Location of the stork-agent man page.
const AgentMan = "../../../doc/man/stork-agent.8.rst"

// This test checks if stork-agent -h reports all expected command-line switches.
func TestCommandLineSwitches(t *testing.T) {
	// Arrange
	os.Args = make([]string, 2)
	os.Args[1] = "-h"

	// Act
	stdout, _, err := testutil.CaptureOutput(main, nil, 0)

	// Assert
	require.NoError(t, err)

	// Now check that all expected command-line switches are really there.
	require.True(t, checkOutput(string(stdout), getExpectedSwitches(), "stork-agent -h output"))
}

// This test checks if all expected command-line switches are documented in the man page.
func TestCommandLineSwitchesDoc(t *testing.T) {
	// Read the contents of the man page
	file, err := os.Open(AgentMan)
	require.NoError(t, err)
	man, err := io.ReadAll(file)
	require.NoError(t, err)

	// And check that all expected switches are mentioned there.
	require.True(t, checkOutput(string(man), getExpectedSwitches(), "stork-agent.8.rst"))
}

// This test checks if stork-agent --version (and -v) report expected version.
func TestCommandLineVersion(t *testing.T) {
	// Let's repeat the test twice (for -v and then for --version)
	for _, opt := range []string{"-v", "--version"} {
		arg := opt
		t.Run(arg, func(t *testing.T) {
			// Arrange
			os.Args = make([]string, 2)
			os.Args[1] = arg

			// Act
			stdout, _, err := testutil.CaptureOutput(main, nil, 0)

			// Assert
			require.NoError(t, err)

			ver := strings.TrimSpace(string(stdout))
			require.Equal(t, ver, stork.Version)
		})
	}
}

// Check if stork-agent uses --host and --port parameters.
func TestHostAndPortParams(t *testing.T) {
	// Arrange
	os.Args = make([]string, 5)
	os.Args[1] = "--host"
	os.Args[2] = "127.1.2.3"
	os.Args[3] = "--port"
	os.Args[4] = "9876"

	// Act
	// The Stork Agent runs the server at the startup and waits infinitely for
	// the requests. It causes the unit test to be blocked. We wait a short
	// time for the head stdout and send the termination signal. The
	// termination cannot be done too early because the Agent must register
	// the signal handler first. Otherwise, the test will fail without any
	// message. Unfortunately, there is no possibility to check if the handler
	// is already registered.
	startTime := time.Now()
	stdout, _, _ := testutil.CaptureOutput(main, func(stdout []byte, n int) {
		stdoutStr := string(stdout)
		hasExpected := strings.Contains(stdoutStr, "127.1.2.3") &&
			strings.Contains(stdoutStr, "9876")
		isTimeExpired := time.Since(startTime) > time.Second

		if hasExpected || isTimeExpired {
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}
	}, 10000)

	stdoutStr := string(stdout)
	require.Contains(t, stdoutStr, "127.1.2.3")
	require.Contains(t, stdoutStr, "9876")
}

// This test checks if stork-agent -h reports all expected command-line switches.
func TestRegisterCommandLineSwitches(t *testing.T) {
	// Arrange
	os.Args = make([]string, 3)
	os.Args[1] = "register"
	os.Args[2] = "-h"

	// Act
	stdout, _, err := testutil.CaptureOutput(main, nil, 0)

	// Assert
	require.NoError(t, err)

	// Now check that all expected command-line switches are really there.
	require.True(t, checkOutput(string(stdout), getExpectedRegisterSwitches(), "stork-agent register -h output"))
}

// Check if stork-agent uses --agent-host parameter.
func TestRegistrationParams(t *testing.T) {
	// Arrange
	os.Args = make([]string, 4)
	os.Args[1] = "register"
	os.Args[2] = "--agent-host"
	os.Args[3] = "127.4.5.6"

	// The Stork Agent exists using a log.Fatal for these parameters.
	// We replace the standard error handler with a dumb one to prevent
	// interrupt the unit tests.
	defer func() {
		logrus.StandardLogger().ExitFunc = nil
	}()
	logrus.StandardLogger().ExitFunc = func(int) {
		// No exit
	}

	// Act
	stdout, _, _ := testutil.CaptureOutput(main, nil, 0)

	require.Contains(t, string(stdout), "127.4.5.6")
}
