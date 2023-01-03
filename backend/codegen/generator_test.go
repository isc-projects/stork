package codegen

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Creates temporary file with contents. It returns a file handle and the
// function closing and removing the file when it is no longer needed.
func createTempFile(t *testing.T, contents string) (*os.File, func()) {
	file, err := os.CreateTemp(os.TempDir(), "*")
	require.NoError(t, err)
	_, err = file.WriteString(contents)
	require.NoError(t, err)
	return file, func() {
		file.Close()
		os.Remove(file.Name())
	}
}

// Test generating the code using JSON and template file into stdout.
func TestGenerateToStdout(t *testing.T) {
	// Create input JSON file.
	inputFile, closer := createTempFile(t, `[
		{
			"foo": "bar",
			"baz": 1
		},
		{
			"baz": 2
		}
	]`)
	defer closer()

	// Create a template file that corresponds to the JSON file.
	templateFile, closer := createTempFile(t, `[
	{{- range .}}
	{
		{{- if .foo}}
		foo: '{{.foo}}',
		{{- end}}
		baz: {{.baz}},
	},{{end}}
]`)
	defer closer()

	// Generate the code to stdout and capture it.
	stdout, _, err := testutil.CaptureOutput(func() {
		err := GenerateToStdout(inputFile.Name(), templateFile.Name())
		require.NoError(t, err)
	})
	require.NoError(t, err)
	require.Equal(t, `[
	{
		foo: 'bar',
		baz: 1,
	},
	{
		baz: 2,
	},
]`,
		string(stdout))
}

// Test generating the code using JSON and template file into a file.
func TestGenerateToFile(t *testing.T) {
	// Create input JSON file.
	inputFile, closer := createTempFile(t, `{ "foo": "bar" }`)
	defer closer()

	// Create a template file that corresponds to the JSON file.
	templateFile, closer := createTempFile(t, `foo = {{.foo}}`)
	defer closer()

	// Create a temporary output file and close it right away.
	outputFile, err := os.CreateTemp(os.TempDir(), "*")
	require.NoError(t, err)
	defer func() {
		os.Remove(outputFile.Name())
	}()
	outputFile.Close()

	// Generate to the existing file.
	err = GenerateToFile(inputFile.Name(), templateFile.Name(), outputFile.Name())
	require.NoError(t, err)

	// Make sure the output file has expected contents.
	outputContents, err := os.ReadFile(outputFile.Name())
	require.NoError(t, err)
	require.EqualValues(t, "foo = bar", outputContents)

	// Remove the file.
	os.Remove(outputFile.Name())

	// Run the same test again and make sure the file is recreated.
	err = GenerateToFile(inputFile.Name(), templateFile.Name(), outputFile.Name())
	require.NoError(t, err)

	outputContents, err = os.ReadFile(outputFile.Name())
	require.NoError(t, err)
	require.EqualValues(t, "foo = bar", outputContents)
}

// Test that an error is returned when input file is not a valid JSON.
func TestGenerateToStdoutInvalidInput(t *testing.T) {
	// Create input file with malformed JSON.
	inputFile, closer := createTempFile(t, `{]`)
	defer closer()

	// Create a valid template file.
	templateFile, closer := createTempFile(t, `foo = {{.foo}}`)
	defer closer()

	err := GenerateToStdout(inputFile.Name(), templateFile.Name())
	require.Error(t, err)
}

// Test that an error is returned when a template file is invalid.
func TestGenerateToStdoutInvalidTemplate(t *testing.T) {
	// Create an valid input JSON file.
	inputFile, closer := createTempFile(t, `{ "foo": "bar" }`)
	defer closer()

	// Create a template file with invalid structure.
	templateFile, closer := createTempFile(t, `{{.foo-}}`)
	defer closer()

	err := GenerateToStdout(inputFile.Name(), templateFile.Name())
	require.Error(t, err)
}

// Test that an error is returned when the input file is missing.
func TestGenerateToStdoutMissingInput(t *testing.T) {
	// Create a template file but no input file.
	templateFile, closer := createTempFile(t, `{{.foo}}`)
	defer closer()

	err := GenerateToStdout("non-existing", templateFile.Name())
	require.Error(t, err)
}

// Test that an error is returned when the template file is missing.
func TestGenerateToStdoutMissingTemplate(t *testing.T) {
	// Create an input file but no template file.
	inputFile, closer := createTempFile(t, `{ "foo": "bar" }`)
	defer closer()

	err := GenerateToStdout(inputFile.Name(), "non-existing")
	require.Error(t, err)
}
