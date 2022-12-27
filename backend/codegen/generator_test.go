package codegen

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test instantiating a new generator.
func TestNewGenerator(t *testing.T) {
	generator := NewGenerator(NewTypescriptEngine())
	require.NotNil(t, generator)
	require.NotNil(t, generator.engine)
	require.NotNil(t, generator.builder)
}

// Test parsing a JSON array and printing the resulting typescript code.
func TestTypescriptGenerateStructsFromArray(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "codegen-input-*.json")
	require.NoError(t, err)
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	jsonInput := `[
	{
		"foo": "bar",
		"baz": [1, 2, 3],
		"abc": {
			"x": 1,
			"y": 2
		}
	},
	{
		"baz": [2, 3, 4],
		"abc": {
			"x": 12,
			"y": 13
		}
	},
	{
		"foo": "aaa",
		"baz": [3, 5, 7],
		"abc": {
			"x": 11,
			"y": 11
		}
	}
]`
	// Write JSON to the file.
	_, err = f.WriteString(jsonInput)
	require.NoError(t, err)

	// New generator using typescript engine.
	generator := NewGenerator(NewTypescriptEngine())
	require.NotNil(t, generator)

	// Generate the code.
	err = generator.GenerateStructs(f.Name())
	require.NoError(t, err)

	// Print the output to stdout and capture it.
	stdout, _, err := testutil.CaptureOutput(generator.Print)
	require.NoError(t, err)

	// Validate the printed output.
	require.Equal(t,
		`[
    {
        abc: {
            x: 1,
            y: 2,
        },
        baz: [
            1,
            2,
            3,
        ],
        foo: 'bar',
    },
    {
        abc: {
            x: 12,
            y: 13,
        },
        baz: [
            2,
            3,
            4,
        ],
    },
    {
        abc: {
            x: 11,
            y: 11,
        },
        baz: [
            3,
            5,
            7,
        ],
        foo: 'aaa',
    },
]`,
		string(stdout))
}

// Test parsing a JSON array and printing the resulting golang code.
func TestGolangGenerateStructsFromArray(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "codegen-input-*.json")
	require.NoError(t, err)
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	jsonInput := `[
	{
		"foo": "bar",
		"baz": [1, 2, 3],
		"abc": {
			"x": 1,
			"y": 2
		}
	},
	{
		"baz": [2, 3, 4],
		"abc": {
			"x": 12,
			"y": 13
		}
	},
	{
		"foo": "aaa",
		"baz": [3, 5, 7],
		"abc": {
			"x": 11,
			"y": 11
		}
	}
]`
	// Write JSON to the file.
	_, err = f.WriteString(jsonInput)
	require.NoError(t, err)

	// New generator using typescript engine.
	generator := NewGenerator(NewGolangEngine())
	require.NotNil(t, generator)

	// Generate the code.
	err = generator.GenerateStructs(f.Name())
	require.NoError(t, err)

	// Print the output to stdout and capture it.
	stdout, _, err := testutil.CaptureOutput(generator.Print)
	require.NoError(t, err)

	// Validate the printed output.
	require.Equal(t,
		`[]any{
	{
		Abc: any{
			X: 1,
			Y: 2,
		},
		Baz: []any{
			1,
			2,
			3,
		},
		Foo: "bar",
	},
	{
		Abc: any{
			X: 12,
			Y: 13,
		},
		Baz: []any{
			2,
			3,
			4,
		},
	},
	{
		Abc: any{
			X: 11,
			Y: 11,
		},
		Baz: []any{
			3,
			5,
			7,
		},
		Foo: "aaa",
	},
}`,
		string(stdout))
}

// Test parsing JSON map and inserting a generated Typescript code into
// a template.
func TestTypescriptGenerateStructsWithTemplate(t *testing.T) {
	inputFile, err := os.CreateTemp(os.TempDir(), "codegen-input-*.json")
	require.NoError(t, err)
	defer func() {
		inputFile.Close()
		os.Remove(inputFile.Name())
	}()

	jsonInput := `{
		"foo": "bar",
		"baz": [1, 2, 3],
		"abc": {
			"x": 1,
			"y": 2
		}
	}`

	// Write JSON to the file.
	_, err = inputFile.WriteString(jsonInput)
	require.NoError(t, err)

	templateFile, err := os.CreateTemp(os.TempDir(), "codegen-template-*.ts")
	require.NoError(t, err)
	defer func() {
		templateFile.Close()
		os.Remove(templateFile.Name())
	}()

	templateCode := `import { Foo } from 'bar'

export class FunnyClass {
    optionDefs: any = %%%
}`

	_, err = templateFile.WriteString(templateCode)
	require.NoError(t, err)

	// New generator using typescript engine.
	generator := NewGenerator(NewTypescriptEngine())
	require.NotNil(t, generator)

	// Generate the code.
	err = generator.GenerateStructsWithTemplate(inputFile.Name(), templateFile.Name())
	require.NoError(t, err)

	// Print the output to stdout and capture it.
	stdout, _, err := testutil.CaptureOutput(generator.Print)
	require.NoError(t, err)

	// Validate the printed output.
	require.Equal(t,
		`import { Foo } from 'bar'

export class FunnyClass {
    optionDefs: any = {
        abc: {
            x: 1,
            y: 2,
        },
        baz: [
            1,
            2,
            3,
        ],
        foo: 'bar',
    }
}`,
		string(stdout))
}

// Test parsing JSON map and inserting a generated Golang code into
// a template.
func TestGolangGenerateStructsWithTemplate(t *testing.T) {
	inputFile, err := os.CreateTemp(os.TempDir(), "codegen-input-*.json")
	require.NoError(t, err)
	defer func() {
		inputFile.Close()
		os.Remove(inputFile.Name())
	}()

	jsonInput := `{
		"foo": "bar",
		"baz": [1, 2, 3],
		"abc": {
			"x": 1,
			"y": 2
		}
	}`

	// Write JSON to the file.
	_, err = inputFile.WriteString(jsonInput)
	require.NoError(t, err)

	templateFile, err := os.CreateTemp(os.TempDir(), "codegen-template-*.go")
	require.NoError(t, err)
	defer func() {
		templateFile.Close()
		os.Remove(templateFile.Name())
	}()

	templateCode := `package keaconfig

func GetFunnyType() FunnyType {
	gen := %%%
	return gen
}`

	_, err = templateFile.WriteString(templateCode)
	require.NoError(t, err)

	// New generator using typescript engine.
	engine := NewGolangEngine()
	engine.SetTopLevelType("FunnyType")
	generator := NewGenerator(engine)
	require.NotNil(t, generator)

	// Generate the code.
	err = generator.GenerateStructsWithTemplate(inputFile.Name(), templateFile.Name())
	require.NoError(t, err)

	// Print the output to stdout and capture it.
	stdout, _, err := testutil.CaptureOutput(generator.Print)
	require.NoError(t, err)

	// Validate the printed output.
	require.Equal(t,
		`package keaconfig

func GetFunnyType() FunnyType {
	gen := FunnyType{
		Abc: any{
			X: 1,
			Y: 2,
		},
		Baz: []any{
			1,
			2,
			3,
		},
		Foo: "bar",
	}
	return gen
}`,
		string(stdout))
}

// Test that generated field names can be customized for the selected
// JSON key names.
func TestSetStaticFieldNames(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "codegen-input-*.json")
	require.NoError(t, err)
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	jsonInput := `{
		"foo": "bar",
		"baz": [1, 2, 3],
		"abc": {
			"x": 1,
			"y": 2
		}
	}`

	// Write JSON to the file.
	_, err = f.WriteString(jsonInput)
	require.NoError(t, err)

	// New generator using typescript engine.
	generator := NewGenerator(NewGolangEngine())
	require.NotNil(t, generator)

	err = generator.SetStaticFieldNames([]string{"baz:BazField"})
	require.NoError(t, err)

	// Generate the code.
	err = generator.GenerateStructs(f.Name())
	require.NoError(t, err)

	// Print the output to stdout and capture it.
	stdout, _, err := testutil.CaptureOutput(generator.Print)
	require.NoError(t, err)

	// Validate the printed output.
	require.Equal(t,
		`any{
	Abc:      any{
		X: 1,
		Y: 2,
	},
	BazField: []any{
		1,
		2,
		3,
	},
	Foo:      "bar",
}`,
		string(stdout))
}

// Test that the generator can write a generated code to a file.
func TestGeneratorWrite(t *testing.T) {
	generator := NewGenerator(NewGolangEngine())
	require.NotNil(t, generator)

	// Create a temporary file and close it right away.
	tempFile, err := os.CreateTemp(os.TempDir(), "codegen-input-*.json")
	require.NoError(t, err)
	defer func() {
		os.Remove(tempFile.Name())
	}()
	tempFile.Close()

	// Set some dummy generated code and write it out to the existing file.
	generator.append("{}")
	err = generator.Write(tempFile.Name())
	require.NoError(t, err)

	// Make sure the write has been successful.
	contents, err := os.ReadFile(tempFile.Name())
	require.NoError(t, err)
	require.EqualValues(t, "{}", contents)

	// Remove the file and test it again to ensure that the function can
	// create the file on its own.
	err = os.Remove(tempFile.Name())
	require.NoError(t, err)
	err = generator.Write(tempFile.Name())
	require.NoError(t, err)
	contents, err = os.ReadFile(tempFile.Name())
	require.NoError(t, err)
	require.EqualValues(t, "{}", contents)
}
