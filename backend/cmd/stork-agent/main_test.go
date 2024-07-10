package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

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
		"--prometheus-bind9-exporter-port",
		"--env-file", "--use-env-file", "--hook-directory",
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
const AgentMan = "../../../doc/user/man/stork-agent.8.rst"

// This test checks if stork-agent -h reports all expected command-line switches.
func TestCommandLineSwitches(t *testing.T) {
	// Arrange
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = make([]string, 2)
	os.Args[1] = "-h"

	// Act
	stdout, _, err := testutil.CaptureOutput(main)

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
	defer testutil.CreateOsArgsRestorePoint()()
	for _, opt := range []string{"-v", "--version"} {
		arg := opt
		t.Run(arg, func(t *testing.T) {
			// Arrange
			os.Args = make([]string, 2)
			os.Args[1] = arg

			// Act
			stdout, _, err := testutil.CaptureOutput(main)

			// Assert
			require.NoError(t, err)

			ver := strings.TrimSpace(string(stdout))
			require.Equal(t, ver, stork.Version)
		})
	}
}

// This test checks if stork-agent -h reports all expected command-line switches.
func TestRegisterCommandLineSwitches(t *testing.T) {
	// Arrange
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = make([]string, 3)
	os.Args[1] = "register"
	os.Args[2] = "-h"

	// Act
	stdout, _, err := testutil.CaptureOutput(main)

	// Assert
	require.NoError(t, err)

	// Now check that all expected command-line switches are really there.
	require.True(t, checkOutput(string(stdout), getExpectedRegisterSwitches(), "stork-agent register -h output"))
}

// Check if stork-agent uses --agent-host parameter.
func TestRegistrationParams(t *testing.T) {
	// Arrange
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"agent-program",
		"register",
		"--server-url",
		"http://localhost",
		"--agent-host",
		"127.4.5.678::8080",
	}
	// The Stork Agent exits using a log.Fatal for these parameters.
	// We replace the standard error handler with a dumb one to prevent
	// interrupting the unit tests.
	defer func() {
		logrus.StandardLogger().ExitFunc = nil
	}()
	logrus.StandardLogger().ExitFunc = func(int) {
		// No exit
	}

	// Act
	stdout, _, _ := testutil.CaptureOutput(main)

	require.Contains(t, string(stdout), "127.4.5.678")
}

// Check if stork-agent uses --agent-host parameter from the environment file.
func TestRegistrationParamsFromEnvironmentFile(t *testing.T) {
	// Arrange
	defer testutil.CreateOsArgsRestorePoint()()
	defer testutil.CreateEnvironmentRestorePoint()()
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	envPath, _ := sandbox.Write("file.env", `
		STORK_AGENT_SERVER_URL=http://localhost:8080
		STORK_AGENT_HOST=localhost::8080
	`)

	os.Args = []string{
		"agent-program",
		"--use-env-file",
		"--env-file", envPath,
		"register",
	}

	// The Stork Agent exits using a log.Fatal for these parameters.
	// We replace the standard error handler with a dumb one to prevent
	// interrupting the unit tests.
	defer func() {
		logrus.StandardLogger().ExitFunc = nil
	}()
	logrus.StandardLogger().ExitFunc = func(int) {
		// No exit
	}

	// Act
	stdout, _, _ := testutil.CaptureOutput(main)

	require.Contains(t, string(stdout), "localhost::8080")
	// There is no message about an incorrect environment variable.
	require.NotContains(t, string(stdout), "cannot set")
	require.NotContains(t, string(stdout), "STORK_AGENT_HOST")
}

// Test that the SIGHUP error text is correct.
func TestSighupError(t *testing.T) {
	err := &sighupError{}
	require.Equal(t, "received SIGHUP signal", err.Error())
}

// Test that the Ctrl-C error text is correct.
func TestCtrlCError(t *testing.T) {
	err := &ctrlcError{}
	require.Equal(t, "received Ctrl-C signal", err.Error())
}
