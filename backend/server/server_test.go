package server

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

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

// This is the list of all parameters we expect to see there.
func getExpectedSwitches() []string {
	return []string{
		"-v", "-m", "--metrics", "--version", "-d", "--db-name", "-u", "--db-user", "--db-host",
		"-p", "--db-port", "--db-trace-queries", "--rest-cleanup-timeout", "--rest-graceful-timeout",
		"--rest-max-header-size", "--rest-host", "--rest-port", "--rest-listen-limit",
		"--rest-keep-alive", "--rest-read-timeout", "--rest-write-timeout", "--rest-tls-certificate",
		"--rest-tls-key", "--rest-tls-ca", "--rest-static-files-dir",
	}
}

// Location of the stork-server man page.
const Man = "../../doc/man/stork-server.8.rst"

// This test checks if stork-agent -h reports all expected command-line switches.
func TestCommandLineSwitches(t *testing.T) {
	// Arrange
	os.Args = make([]string, 2)
	os.Args[1] = "-h"

	// Act
	ss := &StorkServer{}
	var done bool
	var err error
	stdout, _, _ := testutil.CaptureOutput(func() {
		done, err = ss.ParseArgs()
	})

	// Assert
	require.True(t, done)
	require.NoError(t, err)
	// Now check that all expected command-line switches are really there.
	require.True(t, checkOutput(string(stdout), getExpectedSwitches(), "stork-agent -h output"))
}

// This test checks if all expected command-line switches are documented.
func TestCommandLineSwitchesDoc(t *testing.T) {
	// Read the contents of the man page
	file, err := os.Open(Man)
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
			ss := &StorkServer{}
			var done bool
			var err error
			stdout, _, _ := testutil.CaptureOutput(func() {
				done, err = ss.ParseArgs()
			})

			// Assert
			require.NoError(t, err)
			require.True(t, done)
			// Clean up the output (remove end of line)
			ver := strings.TrimSpace(string(stdout))

			// Check if it equals expected version.
			require.Equal(t, ver, stork.Version)
		})
	}
}
