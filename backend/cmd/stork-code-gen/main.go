package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/urfave/cli/v2"
	"isc.org/stork"
	"isc.org/stork/codegen"
)

// The struct that must be embedded in all structures defining the command
// settings. It allows to recognize which command was specified in the CLI.
// It is related to how the go-flags library handles the subcommands.
//
// It may be also used to specify arguments of the command that accepts no
// arguments.
type cliCommand struct {
	// It is true if the register command was specified. Otherwise, it is false.
	commandSpecified bool
}

// Checks if the struct implement the library interface.
var _ flags.Commander = (*cliCommand)(nil)

// Implements the tools/golang/gopath/pkg/mod/github.com/jessevdk/go-flags@v1.5.0/command.go Commander interface.
// It is an only way to recognize which command was specified.
func (s *cliCommand) Execute(_ []string) error {
	s.commandSpecified = true
	return nil
}

// The CLI flags not related to any specific command.
type generalCommand struct {
	cliCommand
	// If true, the version of Stork  is printed. It takes precedence
	// over all other commands and arguments.
	Version bool `short:"v" long:"version" description:"Show software version"`
}

type stdOptionDefinitionsCommand struct {
	cliCommand
	Input    string `short:"i" long:"input" description:"Path to the input file holding option definitions' specification." required:"true"`
	Output   string `short:"o" long:"output" description:"Path to the output file or 'stdout' to print the generated code in the terminal." required:"true"`
	Template string `short:"t" long:"template" description:"Path to the template file used to generate the output file. The generated code is embedded in the template file."`
}

// Generates the code defining standard option definitions to stdout or
// to a file.
func generateStdOptionDefs(c *cli.Context) error {
	// Print the output to the stdout or to a file.
	if c.String("output") == "stdout" {
		return codegen.GenerateToStdout(c.String("input"), c.String("template"))
	}
	return codegen.GenerateToFile(c.String("input"), c.String("template"), c.String("output"))
}

// Man function exposing command line parameters.
func main() {
	app := &cli.App{
		Name:     "Stork Code Gen",
		Usage:    "Code generator used in Stork development",
		Version:  stork.Version,
		HelpName: "stork-code-gen",
		Flags:    []cli.Flag{},
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
