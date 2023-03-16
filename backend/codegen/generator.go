package codegen

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"text/template"

	"github.com/pkg/errors"
)

// Parses an input file containing JSON structures and the template file.
// It creates a template instance that can be executed and its output
// written to a file or stdout.
func prepare(inputFilename, templateFilename string, parsedJSON any) (*template.Template, error) {
	// Open JSON file.
	inputFile, err := os.Open(inputFilename)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	// Read the input file.
	bytes, err := io.ReadAll(inputFile)
	if err != nil {
		return nil, err
	}

	// Parse the JSON content from the input file.
	err = json.Unmarshal(bytes, parsedJSON)
	if err != nil {
		return nil, errors.WithMessagef(err, "error parsing JSON input file %s", inputFilename)
	}

	// Create the template from the template file.
	template, err := template.New(path.Base(templateFilename)).Option("missingkey=error").ParseFiles(templateFilename)
	return template, err
}

// Parses an input file containing JSON structures and generates the code from it
// using the specified template file. This function variant writes output to a
// file.
func GenerateToFile(inputFilename, templateFilename, outputFilename string) error {
	var parsedJSON any
	template, err := prepare(inputFilename, templateFilename, &parsedJSON)
	if err != nil {
		return err
	}
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	err = template.Execute(outputFile, &parsedJSON)
	return err
}

// Parses an input file containing JSON structures and generates the code from it
// using the specified template file. This function variant writes output to stdout.
func GenerateToStdout(inputFilename, templateFilename string) error {
	var parsedJSON any
	template, err := prepare(inputFilename, templateFilename, &parsedJSON)
	if err != nil {
		return err
	}
	err = template.Execute(os.Stdout, parsedJSON)
	return err
}
