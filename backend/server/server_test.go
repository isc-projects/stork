package server

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
		"--rest-tls-key", "--rest-tls-ca", "--rest-static-files-dir", "--initial-interval",
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
	var command Command
	var err error
	stdout, _, _ := testutil.CaptureOutput(func() {
		command, err = ss.ParseArgs()
	})

	// Assert
	require.EqualValues(t, HelpCommand, command)
	require.NoError(t, err)
	// Now check that all expected command-line switches are really there.
	require.True(t, checkOutput(string(stdout), getExpectedSwitches(), "stork-server -h output"))
}

// This test checks if all expected command-line switches are documented.
func TestCommandLineSwitchesDoc(t *testing.T) {
	// Read the contents of the man page
	file, err := os.Open(Man)
	require.NoError(t, err)
	man, err := io.ReadAll(file)
	require.NoError(t, err)

	// And check that all expected switches are mentioned there.
	require.True(t, checkOutput(string(man), getExpectedSwitches(), "stork-server.8.rst"))
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
			command, err := ss.ParseArgs()

			// Assert
			require.NoError(t, err)
			require.EqualValues(t, VersionCommand, command)
		})
	}
}

// Test that the Stork Server is constructed properly.
func TestNewStorkServer(t *testing.T) {
	// Arrange
	os.Args = make([]string, 0)
	os.Args = append(os.Args, "stork-server",
		"-m",
		"--initial-interval", "42",
		"-d", "dbname",
		"-u", "dbuser",
		"--db-host", "dbhost",
		"-p", "9876",
		"--db-sslmode", "verify-ca",
		"--db-sslcert", "sslcert",
		"--db-sslkey", "sslkey",
		"--db-sslrootcert", "sslrootcert",
		"--db-trace-queries", "all",
		"--rest-cleanup-timeout", "12s",
		"--rest-graceful-timeout", "34m",
		"--rest-max-header-size", "56",
		"--rest-host", "resthost",
		"--rest-port", "1234",
		"--rest-listen-limit", "78",
		"--rest-keep-alive", "90h",
		"--rest-read-timeout", "98s",
		"--rest-write-timeout", "76s",
		"--rest-tls-certificate", "tlscert",
		"--rest-tls-key", "tlskey",
		"--rest-tls-ca", "tlsca",
		"--rest-static-files-dir", "staticdir",
	)

	// Act
	ss, command, err := NewStorkServer()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, RunCommand, command)

	require.True(t, ss.EnableMetricsEndpoint)
	require.EqualValues(t, 42, ss.InitialPullerInterval)
	require.EqualValues(t, "dbname", ss.DBSettings.DBName)
	require.EqualValues(t, "dbuser", ss.DBSettings.User)
	require.EqualValues(t, "dbhost", ss.DBSettings.Host)
	require.EqualValues(t, 9876, ss.DBSettings.Port)
	require.EqualValues(t, "verify-ca", ss.DBSettings.SSLMode)
	require.EqualValues(t, "sslcert", ss.DBSettings.SSLCert)
	require.EqualValues(t, "sslkey", ss.DBSettings.SSLKey)
	require.EqualValues(t, "sslrootcert", ss.DBSettings.SSLRootCert)
	require.EqualValues(t, "all", ss.DBSettings.TraceSQL)
	require.EqualValues(t, 12*time.Second, ss.RestAPISettings.CleanupTimeout)
	require.EqualValues(t, 34*time.Minute, ss.RestAPISettings.GracefulTimeout)
	require.EqualValues(t, 56, ss.RestAPISettings.MaxHeaderSize)
	require.EqualValues(t, "resthost", ss.RestAPISettings.Host)
	require.EqualValues(t, 1234, ss.RestAPISettings.Port)
	require.EqualValues(t, 78, ss.RestAPISettings.ListenLimit)
	require.EqualValues(t, 90*time.Hour, ss.RestAPISettings.KeepAlive)
	require.EqualValues(t, 98*time.Second, ss.RestAPISettings.ReadTimeout)
	require.EqualValues(t, 76*time.Second, ss.RestAPISettings.WriteTimeout)
	require.EqualValues(t, "tlscert", ss.RestAPISettings.TLSCertificate)
	require.EqualValues(t, "tlskey", ss.RestAPISettings.TLSCertificateKey)
	require.EqualValues(t, "tlsca", ss.RestAPISettings.TLSCACertificate)
	require.EqualValues(t, "staticdir", ss.RestAPISettings.StaticFilesDir)
}

// Test that the Stork Server is not constructed if the arguments are wrong.
func TestNewStorkServerWithWrongCLIArguments(t *testing.T) {
	// Arrange
	os.Args = make([]string, 0)
	os.Args = append(os.Args, "stork-server", "--foo-bar-baz")

	// Act
	ss, command, err := NewStorkServer()

	// Assert
	require.Error(t, err)
	require.Nil(t, ss)
	require.EqualValues(t, NoneCommand, command)
}

// Test that the Stork Server is constructed if no arguments are provided.
func TestNewStorkServerNoArguments(t *testing.T) {
	// Arrange
	os.Args = []string{"stork-server"}

	// Act
	ss, command, err := NewStorkServer()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, ss)
	require.EqualValues(t, RunCommand, command)
}
