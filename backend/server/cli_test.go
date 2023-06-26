package server

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that the CLI parser is constructed properly.
func TestNewCLIParser(t *testing.T) {
	// Arrange & Act
	parser := NewCLIParser()

	// Assert
	require.NotNil(t, parser)
}

// Test that the environment variables from the environment file are loaded
// and parsed by the CLI parser.
func TestEnvironmentFileIsLoaded(t *testing.T) {
	// Arrange
	restorePoint := testutil.CreateEnvironmentRestorePoint()
	defer restorePoint()
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	envPath, _ := sandbox.Write("file.env", `
		STORK_DATABASE_HOST=foo
		STORK_SERVER_HOOK_DIRECTORY=bar
		STORK_REST_HOST=baz
	`)

	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"program-name",
		"--use-env-file",
		"--env-file", envPath,
	}

	parser := NewCLIParser()

	// Act
	command, settings, err := parser.Parse()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.Equal(t, RunCommand, command)

	require.Equal(t, "foo", settings.DatabaseSettings.Host)
	require.Equal(t, "bar", settings.GeneralSettings.HookDirectory)
	require.Equal(t, "baz", settings.RestAPISettings.Host)
}

// Test that the CLI arguments take precedence over the environment file and
// that the environment file has higher order than the environment variables.
func TestParseArgsFromMultipleSources(t *testing.T) {
	// Arrange
	// Environment variables - the lowest priority.
	restore := testutil.CreateEnvironmentRestorePoint()
	defer restore()

	os.Setenv("STORK_DATABASE_HOST", "database-host-envvar")
	os.Setenv("STORK_REST_HOST", "rest-host-envvar")
	os.Setenv("STORK_REST_TLS_CERTIFICATE", "certificate-envvar")

	// Environment file. Takes precedence over the environment variables.
	environmentFile, _ := os.CreateTemp("", "stork-envfile-test-*")
	defer func() {
		environmentFile.Close()
		os.Remove(environmentFile.Name())
	}()
	environmentFile.WriteString("STORK_REST_HOST=rest-host-envfile\n")
	environmentFile.WriteString("STORK_REST_TLS_CERTIFICATE=certificate-envfile\n")

	// CLI arguments - the highest priority.
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"--rest-tls-certificate", "certificate-cli",
		"--use-env-file",
		"--env-file", environmentFile.Name(),
	}

	parser := NewCLIParser()
	// Act
	command, settings, err := parser.Parse()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, RunCommand, command)
	require.EqualValues(t, "database-host-envvar", settings.DatabaseSettings.Host)
	require.EqualValues(t, "rest-host-envfile", settings.RestAPISettings.Host)
	require.EqualValues(t, "certificate-envfile", settings.RestAPISettings.TLSCertificate)
}

// Test that the parser throws an error if the arguments are wrong.
func TestCLIParserRejectsWrongCLIArguments(t *testing.T) {
	// Arrange
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = make([]string, 0)
	os.Args = append(os.Args, "stork-server", "--foo-bar-baz")
	parser := NewCLIParser()

	// Act
	command, settings, err := parser.Parse()

	// Assert
	require.Error(t, err)
	require.Nil(t, settings)
	require.EqualValues(t, NoneCommand, command)
}
