package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/codegen"
	"isc.org/stork/testutil"
)

// Test creating golang engine and generator instances.
func TestCreateGolangEngineAndGenerator(t *testing.T) {
	engine, err := createEngine(codegen.GolangEngineType, "foo", []string{"bar:baz"})
	require.NoError(t, err)
	require.NotNil(t, engine)
	require.Equal(t, codegen.GolangEngineType, engine.GetEngineType())

	generator, err := createGenerator(engine, []string{"foo:bar"})
	require.NoError(t, err)
	require.NotNil(t, generator)
}

// Test creating typescript engine and generator instances.
func TestCreateTypescriptEngineAndGenerator(t *testing.T) {
	engine, err := createEngine(codegen.TypescriptEngineType, "", []string{})
	require.NoError(t, err)
	require.NotNil(t, engine)
	require.Equal(t, codegen.TypescriptEngineType, engine.GetEngineType())

	generator, err := createGenerator(engine, []string{})
	require.NoError(t, err)
	require.NotNil(t, generator)
}

// Test that an error is returned when specified language is not
// supported.
func TestCreateGolangEngineWrongType(t *testing.T) {
	engine, err := createEngine("unsupported", "", []string{})
	require.Error(t, err)
	require.Nil(t, engine)
}

// Test that setting static field type mapping causes an error when
// it has invalid format.
func TestCreateGolangEngineWrongFieldType(t *testing.T) {
	engine, err := createEngine(codegen.GolangEngineType, "", []string{"foo:baz:bar"})
	require.Error(t, err)
	require.Nil(t, engine)
}

// Test that setting static field name mapping causes an error when
// it has invalid format.
func TestCreateGolangGeneratorWrongFieldName(t *testing.T) {
	engine, err := createEngine(codegen.GolangEngineType, "foo", []string{"bar:baz"})
	require.NoError(t, err)
	require.NotNil(t, engine)

	generator, err := createGenerator(engine, []string{"foo"})
	require.Error(t, err)
	require.Nil(t, generator)
}

// Test that setting static field type causes an error for a typescript
// engine.
func TestCreateTypescriptEngineFieldTypesUnsupported(t *testing.T) {
	engine, err := createEngine(codegen.TypescriptEngineType, "", []string{"foo:baz"})
	require.Error(t, err)
	require.Nil(t, engine)
}

// Test that help for the stork-code-gen app returns valid switches.
func TestMainHelp(t *testing.T) {
	os.Args = make([]string, 2)
	os.Args[1] = "-h"

	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, "std-option-defs")
	require.Contains(t, stdout, "--language")
	require.Contains(t, stdout, "--help")
	require.Contains(t, stdout, "--version")
}

// Test that help for the stork-code-gen std-option-defs returns valid
// switches.
func TestStdOptionDefsHelp(t *testing.T) {
	os.Args = make([]string, 5)
	os.Args[1] = "--language"
	os.Args[2] = "golang"
	os.Args[3] = "help"
	os.Args[4] = "std-option-defs"

	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, "--input")
	require.Contains(t, stdout, "--output")
	require.Contains(t, stdout, "--template")
	require.Contains(t, stdout, "--top-level-type")
	require.Contains(t, stdout, "--field-name")
	require.Contains(t, stdout, "--field-type")
}

// Test that the main fuction triggers generating option definitions.
func TestGenerateStdOptionDefs(t *testing.T) {
	// Create an input file with JSON contents.
	inputFile, err := os.CreateTemp(os.TempDir(), "code-gen-input-*.json")
	require.NoError(t, err)
	require.NotNil(t, inputFile)

	_, err = inputFile.WriteString(`{
    "foo": "bar"
}`)
	require.NoError(t, err)
	defer func() {
		inputFile.Close()
		os.Remove(inputFile.Name())
	}()

	// Prepare command line arguments generating a typescript code.
	os.Args = make([]string, 8)
	os.Args[1] = "--language"
	os.Args[2] = "typescript"
	os.Args[3] = "std-option-defs"
	os.Args[4] = "--input"
	os.Args[5] = inputFile.Name()
	os.Args[6] = "--output"
	os.Args[7] = "stdout"

	// Run main function and capture output.
	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, `{
    foo: 'bar',
}`)
}
