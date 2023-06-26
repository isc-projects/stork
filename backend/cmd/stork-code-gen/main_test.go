package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that help for the stork-code-gen app returns valid switches.
func TestMainHelp(t *testing.T) {
	defer testutil.CreateOsArgsRestorePoint()()
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
	defer testutil.CreateOsArgsRestorePoint()()
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
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create an input file with JSON contents.
	inputFileName, err := sandbox.Write("input", `{"foo": "bar"}`)
	require.NoError(t, err)

	// Create a valid template file.
	templateFileName, err := sandbox.Write("template", `foo = {{.foo}}`)
	require.NoError(t, err)

	// Prepare command line arguments.
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = make([]string, 8)
	os.Args[1] = "std-option-defs"
	os.Args[2] = "--input"
	os.Args[3] = inputFileName
	os.Args[4] = "--template"
	os.Args[5] = templateFileName
	os.Args[6] = "--output"
	os.Args[7] = "stdout"

	// Run main function and capture output.
	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, "foo = bar")
}
