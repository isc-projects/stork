package server

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/hooks"
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

// Test that the error is returned if the environment file is invalid.
func TestEnvironmentFileIsInvalid(t *testing.T) {
	// Arrange
	restorePoint := testutil.CreateEnvironmentRestorePoint()
	defer restorePoint()
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	envPath, _ := sandbox.Write("file.env", `
		wrong entry
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
	require.Error(t, err)
	require.Nil(t, settings)
	require.Equal(t, NoneCommand, command)
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
	os.Args = []string{"stork-server", "--foo-bar-baz"}
	parser := NewCLIParser()

	// Act
	command, settings, err := parser.Parse()

	// Assert
	require.Error(t, err)
	require.Nil(t, settings)
	require.EqualValues(t, NoneCommand, command)
}

// Test that the namespaces are correct.
func TestHookNamespaces(t *testing.T) {
	// Arrange
	hookNames := []string{
		"foo",
		"foo-bar",
		"foo_bar",
		"foo-42",
		"foo-!@#",
		"foo bar",
		"foo.bar",
		"FOO",
		"fOo",
		"FoO",
		"stork-server-foo",
	}
	expectedFlagNamespaces := []string{
		"foo",
		"foo-bar",
		"foo-bar",
		"foo-42",
		"foo-!@#",
		"foo-bar",
		"foo-bar",
		"foo",
		"foo",
		"foo",
		"foo",
	}
	expectedEnvironmentNamespaces := []string{
		"STORK_SERVER_HOOK_FOO",
		"STORK_SERVER_HOOK_FOO_BAR",
		"STORK_SERVER_HOOK_FOO_BAR",
		"STORK_SERVER_HOOK_FOO_42",
		"STORK_SERVER_HOOK_FOO_!@#",
		"STORK_SERVER_HOOK_FOO_BAR",
		"STORK_SERVER_HOOK_FOO_BAR",
		"STORK_SERVER_HOOK_FOO",
		"STORK_SERVER_HOOK_FOO",
		"STORK_SERVER_HOOK_FOO",
		"STORK_SERVER_HOOK_FOO",
	}

	for i := 0; i < len(hookNames); i++ {
		hookName := hookNames[i]
		t.Run(hookName, func(t *testing.T) {
			// Act
			flagNamespace, envNamespace := getHookNamespaces(hookName)
			// Assert
			require.Equal(t, expectedFlagNamespaces[i], flagNamespace)
			require.Equal(t, expectedEnvironmentNamespaces[i], envNamespace)
		})
	}
}

// Test that the error is returned if the hook directory path points to a file.
func TestCollectHookCLIFlagsForNonDirectoryPath(t *testing.T) {
	// Arrange
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	path, _ := sandbox.Join("file.ext")
	defer testutil.CreateOsArgsRestorePoint()()
	parser := NewCLIParser()

	// Act
	os.Args = []string{"stork-server", "--hook-directory", path}
	command, settings, err := parser.Parse()

	// Assert
	require.ErrorContains(t, err, "hook directory path is not pointing to a directory")
	require.Nil(t, settings)
	require.Equal(t, NoneCommand, command)
}

// Test that the no error is returned if the hook directory doesn't exist.
func TestCollectHookCLIFlagsForMissingDirectory(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	parser := NewCLIParser()
	hookSettings := &HookDirectorySettings{
		path.Join(sb.BasePath, "non-exists-directory"),
	}

	// Act
	flags, err := parser.collectHookCLIFlags(hookSettings)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, flags)
	require.Empty(t, flags)
}

// Test that the hook settings are properly parsed from environment variables.
func TestParseHookSettingsFromEnvironmentVariables(t *testing.T) {
	// Arrange
	restore := testutil.CreateEnvironmentRestorePoint()
	defer restore()
	os.Setenv("STORK_SERVER_HOOK_BAZ_FOO_BAR", "fooBar")

	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{"program-name"}

	type hookSettings struct {
		FooBar string `long:"foo-bar" env:"FOO_BAR"`
	}

	hookFlags := map[string]hooks.HookSettings{
		"baz": &hookSettings{},
	}

	parser := NewCLIParser()

	// Act
	settings, err := parser.parseSettings(hookFlags)

	// Assert
	require.NoError(t, err)
	require.Contains(t, settings.HooksSettings, "baz")
	require.Equal(t, "fooBar", settings.HooksSettings["baz"].(*hookSettings).FooBar)
}

// Test that the hook settings are properly parsed from the CLI arguments.
func TestParseHookSettingsFromCLI(t *testing.T) {
	// Arrange
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"program-name",
		"--baz.foo-bar", "fooBar",
	}

	type hookSettings struct {
		FooBar string `long:"foo-bar" env:"FOO_BAR"`
	}

	hookFlags := map[string]hooks.HookSettings{
		"baz": &hookSettings{},
	}

	parser := NewCLIParser()

	// Act
	settings, err := parser.parseSettings(hookFlags)

	// Assert
	require.NoError(t, err)
	require.Contains(t, settings.HooksSettings, "baz")
	require.Equal(t, "fooBar", settings.HooksSettings["baz"].(*hookSettings).FooBar)
}

// Test that an error is returned if the two hooks are solved to the same
// namespace.
func TestPaseHookSettingsDuplicatedNamespace(t *testing.T) {
	// Arrange
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"program-name",
		"--baz.foo-bar", "fooBar",
	}

	type hookSettings struct {
		FooBar string `long:"foo-bar" env:"FOO_BAR"`
	}

	hookFlags := map[string]hooks.HookSettings{
		"baz":              &hookSettings{},
		"stork-server-baz": &hookSettings{},
	}

	parser := NewCLIParser()

	// Act
	settings, err := parser.parseSettings(hookFlags)

	// Assert
	require.ErrorContains(t, err, "two hooks using the same configuration namespace")
	require.Nil(t, settings)
}
