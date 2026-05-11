package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"isc.org/stork/cli"
	"isc.org/stork/codegen"
)

type stdOptionDefinitionsSettings struct {
	cli.CommandSettings
	Input    string `short:"i" long:"input" description:"Path to the input file holding option definitions' specification." required:"true"`
	Output   string `short:"o" long:"output" description:"Path to the output file or 'stdout' to print the generated code in the terminal." required:"true"`
	Template string `short:"t" long:"template" description:"Path to the template file used to generate the output file. The generated code is embedded in the template file."`
}

// Generates the code defining standard option definitions to stdout or
// to a file.
func generateStdOptionDefs(settings *stdOptionDefinitionsSettings) error {
	// Print the output to the stdout or to a file.
	if settings.Output == "stdout" {
		return codegen.GenerateToStdout(settings.Input, settings.Template)
	}
	return codegen.GenerateToFile(settings.Input, settings.Template, settings.Output)
}

// Man function exposing command line parameters.
func main() {
	nothing := struct{}{}
	parser := flags.NewParser(&nothing, flags.Default)
	parser.Name = "Stork Code Gen"
	parser.ShortDescription = "Code generator used in Stork development"
	parser.Usage = "stork-code-gen [command] [options]"

	app := cli.NewApp(parser)
	stdOptionDefinitionsSettings := &stdOptionDefinitionsSettings{}
	app.RegisterCommand(
		"std-option-defs",
		"Generate standard option definitions from JSON spec.",
		stdOptionDefinitionsSettings,
		func() {
			err := generateStdOptionDefs(stdOptionDefinitionsSettings)
			if err != nil {
				logrus.WithError(err).Fatal("Failed to generate standard option definitions.")
			}
		},
	)

	err := app.Run(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
