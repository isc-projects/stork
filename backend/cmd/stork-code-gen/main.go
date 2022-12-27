package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"isc.org/stork"
	"isc.org/stork/codegen"
)

// Creates a code generating engine instance depending on the selected language.
// It also optionally sets the top level type name of the generated structure
// the and mappings between the JSON key names and generated structure field
// types.
func createEngine(language, topLevelType string, fieldTypes []string) (codegen.Engine, error) {
	switch language {
	case codegen.GolangEngineType:
		golangEngine := codegen.NewGolangEngine()
		golangEngine.SetTopLevelType(topLevelType)
		if err := golangEngine.SetStaticFieldTypes(fieldTypes); err != nil {
			return nil, err
		}
		return golangEngine, nil

	case codegen.TypescriptEngineType:
		if len(fieldTypes) > 0 {
			return nil, errors.New("the typescript engine does not support --field-type switch")
		}
		return codegen.NewTypescriptEngine(), nil

	default:
		return nil, errors.Errorf("unsupported language %s", language)
	}
}

// Creates generator instance and sets mappings between JSON keys
// and structure field names.
func createGenerator(engine codegen.Engine, fieldNames []string) (*codegen.Generator, error) {
	generator := codegen.NewGenerator(engine)
	if err := generator.SetStaticFieldNames(fieldNames); err != nil {
		return nil, err
	}
	return generator, nil
}

// Generates a structure with option definitions using selected language's
// syntax.
func generateStdOptionDefs(c *cli.Context) error {
	// Create the code generating engine for the specified language.
	engine, err := createEngine(c.String("language"), c.String("top-level-type"), c.StringSlice("field-type"))
	if err != nil {
		return err
	}
	// Create the generator using the engine.
	generator, err := createGenerator(engine, c.StringSlice("field-name"))
	if err != nil {
		return err
	}
	// Generate the code from a JSON specification and either output the generated
	// code or embed it in the template file contents. The template file must
	// contain %%% placeholder that is substituted by the generated code.
	if c.IsSet("template") {
		err = generator.GenerateStructsWithTemplate(c.String("input"), c.String("template"))
	} else {
		err = generator.GenerateStructs(c.String("input"))
	}
	if err != nil {
		return err
	}
	// Print the output to the stdout or to a file.
	if c.String("output") == "stdout" {
		generator.Print()
		return nil
	}
	return generator.Write(c.String("output"))
}

// Man function exposing command line parameters.
func main() {
	app := &cli.App{
		Name:     "Stork Code Gen",
		Usage:    "Code generator used in Stork development",
		Version:  stork.Version,
		HelpName: "stork-code-gen",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "language",
				Usage:    "Supported languages are 'golang' and 'typescript'.",
				Required: true,
				Aliases:  []string{"l"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "std-option-defs",
				Usage:     "Generate standard option definitions from JSON spec.",
				UsageText: "stork-code-gen std-option-defs",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "input",
						Usage:    "Path to the input file holding option definitions' specification.",
						Required: true,
						Aliases:  []string{"i"},
					},
					&cli.StringFlag{
						Name:     "output",
						Usage:    "Path to the output file or 'stdout' to print the generated code in the terminal.",
						Required: true,
						Aliases:  []string{"o"},
					},
					&cli.StringFlag{
						Name:    "template",
						Usage:   "Path to the template file used to generate the output file. The generated code is embedded in the template file.",
						Aliases: []string{"t"},
					},
					&cli.StringFlag{
						Name:     "top-level-type",
						Usage:    "Generated top-level slice or map type. For example, Golang structures include type names in data assignments.",
						Required: false,
					},
					&cli.StringSliceFlag{
						Name:     "field-name",
						Usage:    "Maps a JSON key name to the struct field name using the <json-key>:<struct-field> notation. It can be specified multiple times.",
						Required: false,
					},
					&cli.StringSliceFlag{
						Name:     "field-type",
						Usage:    "Maps JSON key name to the struct field type using <json-key>:<struct-field-type> notation. It can be specified multiple times.",
						Required: false,
					},
				},
				Action: generateStdOptionDefs,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
