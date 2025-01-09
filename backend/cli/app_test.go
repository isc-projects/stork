package cli

import (
	"strings"
	"testing"

	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/require"
	"isc.org/stork"
	"isc.org/stork/testutil"
)

// Test that the application instance is created properly.
func TestNewApp(t *testing.T) {
	// Arrange
	parser := flags.NewParser(&struct{}{}, flags.Default)

	// Act
	app := NewApp(parser)

	// Assert
	require.NotNil(t, app)
	require.NotNil(t, app.commandsToFunctions)
	require.NotNil(t, app.parser)
	require.False(t, app.showVersion)
	require.Equal(t, parser, app.parser)
}

// Test that the version printing is handled internally.
func TestRunVersion(t *testing.T) {
	// Arrange
	parser := flags.NewParser(&struct{}{}, flags.Default)
	app := NewApp(parser)
	var err error

	for _, arg := range []string{"-v", "--version"} {
		// Act
		stdout, _, _ := testutil.CaptureOutput(func() {
			err = app.Run([]string{arg})
		})

		// Assert
		require.NoError(t, err)
		require.True(t, app.showVersion)
		require.Equal(t, stork.Version, strings.TrimSpace(string(stdout)))
	}
}

// Test that the error is returned when the command is not provided.
func TestRunNoCommand(t *testing.T) {
	// Arrange
	parser := flags.NewParser(&struct{}{}, flags.Default)
	app := NewApp(parser)

	// Act
	err := app.Run([]string{})

	// Assert
	require.ErrorContains(t, err, "no command specified")
	require.ErrorContains(t, err, "available commands:")
}

// Test that the error is returned when the command is not recognized.
func TestRunUnknownCommand(t *testing.T) {
	// Arrange
	parser := flags.NewParser(&struct{}{}, flags.Default)
	app := NewApp(parser)

	// Act
	err := app.Run([]string{"unknown"})

	// Assert
	require.ErrorContains(t, err, "no command specified")
	require.ErrorContains(t, err, "available commands:")
}

// Test that the command is executed when it is recognized.
func TestRunCommand(t *testing.T) {
	// Arrange
	parser := flags.NewParser(&struct{}{}, flags.Default)
	app := NewApp(parser)
	type settings struct {
		CommandSettings
		Foo string `short:"f" long:"foo" description:"Foo"`
	}
	data := &settings{}
	isCalled := false

	// Act
	app.RegisterCommand("bar", "Bar", data, func() {
		isCalled = true
	})
	err := app.Run([]string{"bar", "-f", "baz"})

	// Assert
	require.NoError(t, err)
	require.True(t, isCalled)
	require.Equal(t, "baz", data.Foo)
}
