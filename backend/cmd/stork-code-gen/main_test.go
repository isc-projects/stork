package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that help for the stork-code-gen app returns valid switches.
func TestMainHelp(t *testing.T) {
	os.Args = make([]string, 2)
	os.Args[1] = "-h"

	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, "std-option-defs")
	require.Contains(t, stdout, "--help")
	require.Contains(t, stdout, "--version")
}

// Test that help for the stork-code-gen std-option-defs returns valid
// switches.
func TestStdOptionDefsHelp(t *testing.T) {
	os.Args = make([]string, 3)
	os.Args[1] = "help"
	os.Args[2] = "std-option-defs"

	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, "--input")
	require.Contains(t, stdout, "--output")
	require.Contains(t, stdout, "--template")
}

// Test that the main fuction triggers generating option definitions.
func TestGenerateStdOptionDefs(t *testing.T) {
	// Create an input file with JSON contents.
	inputFile, err := os.CreateTemp(os.TempDir(), "*")
	require.NoError(t, err)
	require.NotNil(t, inputFile)

	_, err = inputFile.WriteString(`{"foo": "bar"}`)
	require.NoError(t, err)
	defer func() {
		inputFile.Close()
		os.Remove(inputFile.Name())
	}()

	// Create a valid template file.
	templateFile, err := os.CreateTemp(os.TempDir(), "*")
	require.NoError(t, err)
	require.NotNil(t, templateFile)

	_, err = templateFile.WriteString(`foo = {{.foo}}`)
	require.NoError(t, err)
	defer func() {
		inputFile.Close()
		os.Remove(inputFile.Name())
	}()

	// Prepare command line arguments.
	os.Args = make([]string, 8)
	os.Args[1] = "std-option-defs"
	os.Args[2] = "--input"
	os.Args[3] = inputFile.Name()
	os.Args[4] = "--template"
	os.Args[5] = templateFile.Name()
	os.Args[6] = "--output"
	os.Args[7] = "stdout"

	// Run main function and capture output.
	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, "foo = bar")
}
