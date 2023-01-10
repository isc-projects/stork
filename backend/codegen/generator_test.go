package codegen

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test generating the code using JSON and template file into stdout.
func TestGenerateToStdout(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create input JSON file.
	inputFileName, err := sandbox.Write("input", `[
		{
			"foo": "bar",
			"baz": 1
		},
		{
			"foo": "",
			"baz": 2
		}
	]`)
	require.NoError(t, err)

	// Create a template file that corresponds to the JSON file.
	templateFileName, err := sandbox.Write("template", `[
	{{- range .}}
	{
		{{- if .foo}}
		foo: '{{.foo}}',
		{{- end}}
		baz: {{.baz}},
	},{{end}}
]`)
	require.NoError(t, err)

	// Generate the code to stdout and capture it.
	stdout, _, err := testutil.CaptureOutput(func() {
		err := GenerateToStdout(inputFileName, templateFileName)
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
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create input JSON file.
	inputFileName, err := sandbox.Write("input", `{ "foo": "bar" }`)
	require.NoError(t, err)

	// Create a template file that corresponds to the JSON file.
	templateFileName, err := sandbox.Write("template", `foo = {{.foo}}`)
	require.NoError(t, err)

	// Create a temporary output file and close it right away.
	outputFileName, err := sandbox.Write("output", `foo = "foo"`)
	require.NoError(t, err)

	// Generate to the existing file.
	err = GenerateToFile(inputFileName, templateFileName, outputFileName)
	require.NoError(t, err)

	// Make sure the output file has expected contents.
	outputContents, err := os.ReadFile(outputFileName)
	require.NoError(t, err)
	require.EqualValues(t, "foo = bar", outputContents)

	// Remove the file.
	os.Remove(outputFileName)

	// Run the same test again and make sure the file is recreated.
	err = GenerateToFile(inputFileName, templateFileName, outputFileName)
	require.NoError(t, err)

	outputContents, err = os.ReadFile(outputFileName)
	require.NoError(t, err)
	require.EqualValues(t, "foo = bar", outputContents)
}

// Test that an error is returned when input file is not a valid JSON.
func TestGenerateToStdoutInvalidInput(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create input file with malformed JSON.
	inputFileName, err := sandbox.Write("input", `{]`)
	require.NoError(t, err)

	// Create a valid template file.
	templateFileName, err := sandbox.Write("template", `foo = {{.foo}}`)
	require.NoError(t, err)

	err = GenerateToStdout(inputFileName, templateFileName)
	require.Error(t, err)
}

// Test that an error is returned when input file lacks map parameters.
func TestGenerateToStdoutMissingMapParameter(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create input file with lacking baz key.
	inputFileName, err := sandbox.Write("input", `{ "foo": "bar" }`)
	require.NoError(t, err)

	// Create a valid template file.
	templateFileName, err := sandbox.Write("template", `foo = {{.foo}}, baz = {{.baz}}`)
	require.NoError(t, err)

	err = GenerateToStdout(inputFileName, templateFileName)
	require.Error(t, err)
}

// Test that an error is returned when a template file is invalid.
func TestGenerateToStdoutInvalidTemplate(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create an valid input JSON file.
	inputFileName, err := sandbox.Write("input", `{ "foo": "bar" }`)
	require.NoError(t, err)

	// Create a template file with invalid structure.
	templateFileName, err := sandbox.Write("template", `{{.foo-}}`)
	require.NoError(t, err)

	err = GenerateToStdout(inputFileName, templateFileName)
	require.Error(t, err)
}

// Test that an error is returned when the input file is missing.
func TestGenerateToStdoutMissingInput(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create a template file but no input file.
	templateFileName, err := sandbox.Write("template", `{{.foo}}`)
	require.NoError(t, err)

	err = GenerateToStdout("non-existing", templateFileName)
	require.Error(t, err)
}

// Test that an error is returned when the template file is missing.
func TestGenerateToStdoutMissingTemplate(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create an input file but no template file.
	inputFileName, err := sandbox.Write("template", `{ "foo": "bar" }`)
	require.NoError(t, err)

	err = GenerateToStdout(inputFileName, "non-existing")
	require.Error(t, err)
}
