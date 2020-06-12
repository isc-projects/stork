package server

import (
	"fmt"
	"testing"
	"os/exec"
	"io/ioutil"
	"os"
	"strings"

	"github.com/stretchr/testify/require"

	"isc.org/stork"
)



// Aux function checks if a list of expected strings is present in the string
func checkOutput(output string, exp []string, reason string) bool {
	for _, x := range exp {
		fmt.Printf("Checking if %s exists in %s.\n", x, reason)
		if strings.Index(output, x) == -1 {
			fmt.Printf("ERROR: Expected string [%s] not found in %s.\n", x, reason)
			return false
		}
	}
	return true
}

// This is the list of all parameters we expect to see there.
var	exp_switches = []string { "-v", "--version", "-d", "--db-name", "-u", "--db-user", "--db-host",
	"-p", "--db-port", "--db-trace-queries", "--rest-cleanup-timeout", "--rest-graceful-timeout",
	"--rest-max-header-size", "--rest-host", "--rest-port", "--rest-listen-limit",
	"--rest-keep-alive", "--rest-read-timeout", "--rest-write-timeout", "--rest-tls-certificate",
	"--rest-tls-key", "--rest-tls-ca", "--rest-static-files-dir" }

// Location of the stork-server binary
const AGENT_BIN = "../cmd/stork-server/stork-server"

// Location of the stork-server man page
const AGENT_MAN = "../../doc/man/stork-server.8.rst"

// This test checks if stork-agent -h reports all expected command-line switches
func TestCommandLineSwitches(t *testing.T) {
	// Run the --help version and get its output.
	agentCmd := exec.Command(AGENT_BIN, "-h")
	output, err := agentCmd.Output()
	require.NoError(t, err)

	// Now check that all expected command-line switches are really there.
	require.True(t, checkOutput(string(output), exp_switches, "stork-agent -h output"))
}

// This test checks if all expected command-line switches are documented
func TestCommandLineSwitchesDoc(t *testing.T) {
	// Read the contents of the man page
	file, err := os.Open(AGENT_MAN)
	require.NoError(t, err)
	man, err := ioutil.ReadAll(file)

	// And check that all expected switches are mentioned there.
	require.True(t, checkOutput(string(man), exp_switches, "stork-agent.8.rst"))
}

// This test checks if stork-agent --version (and -v) report expected version.
func TestCommandLineVersion(t *testing.T) {
	// Let's repeat the test twice (for -v and then for --version)
	for _, opt := range []string {"-v", "--version"} {
		fmt.Printf("Checking %s\n", opt)

		// Run the agent with specific switch.
		agentCmd := exec.Command(AGENT_BIN, opt)
		output, err := agentCmd.Output()
		require.NoError(t, err)

		// Clean up the output (remove end of line)
		ver := strings.TrimSpace(string(output))

		// Check if it equals expected version.
		require.Equal(t, ver, stork.Version)
	}
}

// This test attempts to test ParseArgs code. There are couple problems with it.
// First, if -v or -h option is used, the parser calls os.Exit(), which ends the test.
// It is reported as successful, but the later part of the test is never used.
//
// When other parameters are used (such as setting ports, hostnames, etc.), the test
// aborts, because the server tries to connect to a database and prints an error
// about inappropriate ioctl for device. This can be worked around by setting up
// STORK_DATABASE_PASSWORD env variable to storktest. If that's done, then the server
// actually starts and the test never finishes.
func TestParseArgs(t *testing.T) {

	// Let's set some fake command line parameters. We need to remember the actual params
	// and later restore them.
	testArgs := []string{os.Args[0], "-v"}
	oldArgs := os.Args
	os.Args = testArgs

	// Now create the server.
	ss, err := NewStorkServer()
	require.NoError(t, err)
	require.NotNil(t, ss)

	// TODO: Check actual parameters.

	// Ok, let's restore the old parameters.
	os.Args = oldArgs
}