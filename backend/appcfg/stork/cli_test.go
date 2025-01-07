package storkconfig

import (
	"os"
	"path"
	"testing"

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/require"
	"isc.org/stork/hooks"
	"isc.org/stork/testutil"
)

// Test that the CLI parser is constructed properly.
func TestNewCLIParser(t *testing.T) {
	// Arrange & Act
	parser := NewCLIParser(nil, "server", func() {})

	// Assert
	require.NotNil(t, parser)
	require.Nil(t, parser.parser)
	require.Equal(t, "server", parser.application)
	require.NotNil(t, parser.onLoadEnvironmentFileCallback)
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

	type settings struct {
		DBHost   string `long:"db-host" description:"The host name, IP address or socket where database is available" env:"STORK_DATABASE_HOST" default:""`
		RESTHost string `long:"rest-host" description:"The IP to listen on" default:"" env:"STORK_REST_HOST"`
	}

	data := &settings{}

	flagParser := flags.NewParser(data, flags.Default)

	parser := NewCLIParser(flagParser, "server", func() {})

	// Act
	hookDirSettings, hookFlags, isHelp, err := parser.Parse()

	// Assert
	require.NoError(t, err)
	require.False(t, isHelp)
	require.Empty(t, hookFlags)

	require.Equal(t, "foo", data.DBHost)
	require.Equal(t, "bar", hookDirSettings.HookDirectory)
	require.Equal(t, "baz", data.RESTHost)
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

	data := &struct{}{}
	flagParser := flags.NewParser(data, flags.Default)
	parser := NewCLIParser(flagParser, "server", func() {})

	// Act
	hookDirSettings, hookSettings, isHelp, err := parser.Parse()

	// Assert
	require.Error(t, err)
	require.False(t, isHelp)
	require.Empty(t, hookSettings)
	require.Nil(t, hookDirSettings)
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

	type settings struct {
		DBHost   string `long:"db-host" description:"The host name, IP address or socket where database is available" env:"STORK_DATABASE_HOST" default:""`
		RESTHost string `long:"rest-host" description:"The IP to listen on" default:"" env:"STORK_REST_HOST"`
		TLSCert  string `long:"rest-tls-certificate" description:"The path to the TLS certificate" env:"STORK_REST_TLS_CERTIFICATE" default:""`
	}

	data := &settings{}

	parser := NewCLIParser(flags.NewParser(data, flags.Default), "server", func() {})
	// Act
	hookDirSettings, hookSettings, isHelp, err := parser.Parse()

	// Assert
	require.NoError(t, err)
	require.False(t, isHelp)
	require.Empty(t, hookSettings)
	require.NotNil(t, hookDirSettings)
	require.Equal(t, "/usr/lib/stork-server/hooks", hookDirSettings.HookDirectory)
	require.EqualValues(t, "database-host-envvar", data.DBHost)
	require.EqualValues(t, "rest-host-envfile", data.RESTHost)
	require.EqualValues(t, "certificate-envfile", data.TLSCert)
}

// Test that the parser throws an error if the arguments are wrong.
func TestCLIParserRejectsWrongCLIArguments(t *testing.T) {
	// Arrange
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{"stork-server", "--foo-bar-baz"}

	type settings struct {
		DBHost string `long:"db-host" description:"The host name, IP address or socket where database is available" env:"STORK_DATABASE_HOST" default:""`
	}

	data := &settings{}
	parser := NewCLIParser(flags.NewParser(data, flags.Default), "server", func() {})

	// Act
	hookDirSettings, hookSettings, isHelp, err := parser.Parse()

	// Assert
	require.Error(t, err)
	require.False(t, isHelp)
	require.Empty(t, hookSettings)
	require.Nil(t, hookDirSettings)
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
			flagNamespace, envNamespace := getHookNamespaces("server", hookName)
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

	type settings struct{}
	data := &settings{}

	parser := NewCLIParser(flags.NewParser(data, flags.Default), "server", func() {})

	// Act
	os.Args = []string{"stork-server", "--hook-directory", path}
	hookDirSettings, hookFlags, isHelp, err := parser.Parse()

	// Assert
	require.ErrorContains(t, err, "hook directory path is not pointing to a directory")
	require.False(t, isHelp)
	require.Empty(t, hookFlags)
	require.Nil(t, hookDirSettings)
}

// Test that the no error is returned if the hook directory doesn't exist.
func TestCollectHookCLIFlagsForMissingDirectory(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	parser := NewCLIParser(nil, "server", func() {})
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

	data := &struct{}{}
	parser := NewCLIParser(flags.NewParser(data, flags.Default), "server", func() {})

	// Act
	mergeErr := parser.mergeHookFlags(hookFlags)
	parseErr := parser.parse()

	// Assert
	require.NoError(t, mergeErr)
	require.NoError(t, parseErr)

	// ToDO: Fix this test
	// require.Contains(t, settings.HooksSettings, "baz")
	require.Equal(t, "fooBar", hookFlags["baz"].(*hookSettings).FooBar)
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

	data := &struct{}{}
	parser := NewCLIParser(flags.NewParser(data, flags.Default), "server", func() {})

	// Act
	mergeErr := parser.mergeHookFlags(hookFlags)
	parseErr := parser.parse()

	// Assert
	require.NoError(t, mergeErr)
	require.NoError(t, parseErr)
	// TODO: Fix this test
	// require.Contains(t, settings.HooksSettings, "baz")
	require.Equal(t, "fooBar", hookFlags["baz"].(*hookSettings).FooBar)
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

	data := &struct{}{}
	parser := NewCLIParser(flags.NewParser(data, flags.Default), "server", func() {})

	// Act
	err := parser.mergeHookFlags(hookFlags)

	// Assert
	require.ErrorContains(t, err, "two hooks using the same configuration namespace")
}

// Test that the help is properly printed and it includes the hook settings.
func TestParseHelp(t *testing.T) {
	// Arrange
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"program-name",
		"--help",
	}

	type hookSettings struct {
		FooBar string `long:"foo-bar" description:"Lorem ipsum" env:"FOO_BAR"`
	}

	hookFlags := map[string]hooks.HookSettings{
		"baz": &hookSettings{},
	}

	type settings struct {
		TLSCert string `long:"tls-cert" env:"TLS_CERT" description:"The path to the TLS certificate"`
	}
	data := &settings{}

	parser := NewCLIParser(flags.NewParser(data, flags.Default), "server", func() {})
	_ = parser.mergeHookFlags(hookFlags)

	// Act
	var isHelp bool
	var err error
	stdout, stderr, captureErr := testutil.CaptureOutput(func() {
		_, _, isHelp, err = parser.Parse()
	})

	// Assert
	require.NoError(t, err)
	require.NoError(t, captureErr)
	require.True(t, isHelp)
	require.Empty(t, stderr)

	expectedHelp := `Usage:
  program-name [OPTIONS]

Application Options:
      --tls-cert=       The path to the TLS certificate [$TLS_CERT]

Hook 'baz' Flags:
      --baz.foo-bar=    Lorem ipsum [$STORK_SERVER_HOOK_BAZ_FOO_BAR]

Environment File Flags:
      --env-file=       Environment file location; applicable only if the
                        use-env-file is provided (default:
                        /etc/stork/server.env)
      --use-env-file    Read the environment variables from the environment file

Hook Directory Flags:
      --hook-directory= The path to the hook directory (default:
                        /usr/lib/stork-server/hooks)
                        [$STORK_SERVER_HOOK_DIRECTORY]

Help Options:
  -h, --help            Show this help message

`

	require.Equal(t, expectedHelp, string(stdout))
}
