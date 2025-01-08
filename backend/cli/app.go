package cli

import (
	"fmt"

	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"isc.org/stork"
	storkutil "isc.org/stork/util"
)

// It specifies a method that checks if the specific command was specified in
// the CLI. It is used to create a mapping between the command objects and
// the command handlers.
type command interface {
	isSpecified() bool
}

// The struct that must be embedded in all structures defining the command
// settings. It allows to recognize which command was specified in the CLI.
// It is related to how the go-flags library handles the subcommands.
//
// It may be also used to specify arguments of the command that accepts no
// arguments.
type CLICommand struct {
	// It is true if the register command was specified. Otherwise, it is false.
	commandSpecified bool
}

// Checks if the struct implement the library interface.
var _ flags.Commander = (*CLICommand)(nil)

// Implements the tools/golang/gopath/pkg/mod/github.com/jessevdk/go-flags@v1.5.0/command.go Commander interface.
// It is an only way to recognize which command was specified.
func (s *CLICommand) Execute(_ []string) error {
	s.commandSpecified = true
	return nil
}

// Indicates if the command was specified.
func (s *CLICommand) isSpecified() bool {
	return s.commandSpecified
}

// Prints the Stork version.
func showVersion() {
	fmt.Println(stork.Version)
}

// The type describing the command handler.
// It is a function that takes no arguments and returns no value.
// Maybe it should an error as a return value. Currently, it is not necessary
// but it may be useful in the future refactorings for example to unify the
// error handling in various commands.
type action = func()

// A helper structure that mimics the behavior of the urfave/cli/v2 package.
// It helps to create a CLI application with subcommands.
// It is a wrapper around the go-flags library. It accepts the go-flags parser
// and provides a convenient way to create subcommands that should be used
// instead of the built-in .AddCommand method.
// It has a run method that parses the command line arguments and executes
// the appropriate action.
type App struct {
	commandsToFunctions map[command]action
	parser              *flags.Parser
	showVersion         bool
}

// Constructs a new application instance.
func NewApp(parser *flags.Parser) *App {
	app := &App{
		commandsToFunctions: make(map[command]action),
		parser:              parser,
	}
	app.enableVersionOption()
	return app
}

// Adds a top-level CLI argument to show the software version.
func (a *App) enableVersionOption() {
	a.parser.AddOption(&flags.Option{
		ShortName:   'v',
		LongName:    "version",
		Description: "Show software version",
	}, &a.showVersion)
}

// Registers a command with the parser and associates it with the action.
func (a *App) RegisterCommand(command, shortDescription string, data command, action action) {
	_, err := a.parser.AddCommand(command, shortDescription, "", data)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to add command")
	}
	a.commandsToFunctions[data] = action
}

// Starts the application with the provided arguments.
// Run requested subcommand or show help or version.
func (a *App) Run(args []string) error {
	// Parse command line arguments.
	appParser := NewCLIParser(a.parser, "tool", func() {
		storkutil.SetupLogging()
	})

	_, _, isHelp, err := appParser.Parse(args)
	if err != nil {
		return err
	}
	if isHelp {
		return nil
	}

	// Handle the version argument first.
	if a.showVersion {
		showVersion()
		return nil
	}

	// Find the command that was specified.
	for command, action := range a.commandsToFunctions {
		if command.isSpecified() {
			action()
			return nil
		}
	}

	var availableCommands []string
	for _, command := range a.parser.Commands() {
		availableCommands = append(availableCommands, command.Name)
	}

	return errors.Errorf("no command specified, available commands: %v", availableCommands)
}
