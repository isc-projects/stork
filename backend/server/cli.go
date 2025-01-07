package server

import (
	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	storkconfig "isc.org/stork/appcfg/stork"
	"isc.org/stork/hooks"
	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/restservice"
	storkutil "isc.org/stork/util"
)

// The passed command to the server by the CLI.
type Command string

// Valid commands supported by the Stork server.
const (
	// None command provided.
	NoneCommand Command = "none"
	// Run the server.
	RunCommand Command = "run"
	// Show help message.
	HelpCommand Command = "help"
	// Show version.
	VersionCommand Command = "version"
)

// Read environment file settings. It's parsed before the main settings.
type EnvironmentFileSettings struct {
	EnvFile    string `long:"env-file" description:"Environment file location; applicable only if the use-env-file is provided" default:"/etc/stork/server.env"`
	UseEnvFile bool   `long:"use-env-file" description:"Read the environment variables from the environment file"`
}

// Read hook directory settings. They are parsed after environment file
// settings but before the main settings.
// It allows us to merge the hook flags with the core flags into a single output.
type HookDirectorySettings struct {
	HookDirectory string `long:"hook-directory" description:"The path to the hook directory; if relative, it is resolved against the stork-server executable directory" env:"STORK_SERVER_HOOK_DIRECTORY" default:"../lib/stork-server/hooks"`
}

// General server settings.
type GeneralSettings struct {
	Version               bool  `short:"v" long:"version" description:"Show software version"`
	EnableMetricsEndpoint bool  `short:"m" long:"metrics" description:"Enable Prometheus /metrics endpoint (no auth)" env:"STORK_SERVER_ENABLE_METRICS"`
	InitialPullerInterval int64 `long:"initial-puller-interval" description:"Initial interval used by pullers fetching data from Kea; if not provided the recommended values for each puller are used" env:"STORK_SERVER_INITIAL_PULLER_INTERVAL"`
}

// Groups all Stork settings.
type Settings struct {
	GeneralSettings  *GeneralSettings
	RestAPISettings  *restservice.RestAPISettings
	AgentsSettings   *agentcomm.AgentsSettings
	HooksSettings    map[string]hooks.HookSettings
	DatabaseSettings *dbops.DatabaseSettings
	HookDirectory    string
}

// Constructs a new settings instance.
// The members must be initialized because the go-flags library requires
// non-empty pointers.
func newSettings() *Settings {
	return &Settings{
		GeneralSettings:  &GeneralSettings{},
		RestAPISettings:  &restservice.RestAPISettings{},
		AgentsSettings:   &agentcomm.AgentsSettings{},
		HooksSettings:    make(map[string]hooks.HookSettings),
		DatabaseSettings: &dbops.DatabaseSettings{},
	}
}

// Parse the command line arguments into Stork-specific GO structures.
// First, it parses the settings related to an environment file and if the file
// is provided, the content is loaded.
// Next, it parses the hooks location and extracts their CLI flags.
// At the end, it composes the CLI parser from all the flags and runs it.
func ParseCLIFlags() (command Command, settings *Settings, err error) {
	command = NoneCommand
	settings = newSettings()

	parser := flags.NewParser(settings.GeneralSettings, flags.Default)
	parser.ShortDescription = "Stork Server"
	parser.LongDescription = `Stork Server is a Kea and BIND 9 dashboard

Stork logs on INFO level by default. Other levels can be configured using the
STORK_LOG_LEVEL variable. Allowed values are: DEBUG, INFO, WARN, ERROR.`

	databaseFlags := &dbops.DatabaseCLIFlags{}
	// Process Database specific args.
	_, err = parser.AddGroup("Database ConnectionFlags", "", databaseFlags)
	if err != nil {
		err = errors.Wrap(err, "cannot add the database group")
		return
	}

	// Process ReST API specific args.
	_, err = parser.AddGroup("HTTP ReST Server Flags", "", settings.RestAPISettings)
	if err != nil {
		err = errors.Wrap(err, "cannot add the ReST group")
		return
	}

	// Process agent comm specific args.
	_, err = parser.AddGroup("Agents Communication Flags", "", settings.AgentsSettings)
	if err != nil {
		err = errors.Wrap(err, "cannot add the agents group")
		return
	}

	// Parse CLI flags.
	appParser := storkconfig.NewCLIParser(parser, "server", func() {
		storkutil.SetupLogging()
	})

	hookDirSettings, hookFlags, isHelp, err := appParser.Parse()
	if err != nil {
		return NoneCommand, nil, err
	}

	if isHelp {
		return HelpCommand, nil, nil
	}

	if settings.GeneralSettings.Version {
		// If user specified --version or -v, print the version and quit.
		return VersionCommand, nil, nil
	}

	settings.DatabaseSettings, err = databaseFlags.ConvertToDatabaseSettings()
	if err != nil {
		return NoneCommand, nil, err
	}

	settings.HooksSettings = hookFlags
	settings.HookDirectory = hookDirSettings.HookDirectory

	return RunCommand, settings, nil
}
