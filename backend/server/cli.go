package server

import (
	"fmt"
	"os"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"isc.org/stork/hooks"
	"isc.org/stork/hooksutil"
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
	HookDirectory string `long:"hook-directory" description:"The path to the hook directory" env:"STORK_SERVER_HOOK_DIRECTORY" default:"/usr/lib/stork-server/hooks"`
}

// General server settings.
type GeneralSettings struct {
	EnvironmentFileSettings
	HookDirectorySettings
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

// Stork server-specific CLI arguments/flags parser.
type CLIParser struct {
	shortDescription string
	longDescription  string
}

// Constructs CLI parser.
func NewCLIParser() *CLIParser {
	return &CLIParser{
		shortDescription: "Stork Server",
		longDescription: `Stork Server is a Kea and BIND 9 dashboard

Stork logs on INFO level by default. Other levels can be configured using the
STORK_LOG_LEVEL variable. Allowed values are: DEBUG, INFO, WARN, ERROR.`,
	}
}

// Parse the command line arguments into Stork-specific GO structures.
// First, it parses the settings related to an environment file and if the file
// is provided, the content is loaded.
// Next, it parses the hooks location and extracts their CLI flags.
// At the end, it composes the CLI parser from all the flags and runs it.
func (p *CLIParser) Parse() (command Command, settings *Settings, err error) {
	command = NoneCommand

	envFileSettings, err := p.parseEnvironmentFileSettings()
	if err != nil {
		return
	}

	err = p.loadEnvironmentFile(envFileSettings)
	if err != nil {
		return
	}

	hookDirectorySettings, err := p.parseHookDirectory()
	if err != nil {
		return
	}

	allHookCLIFlags, err := p.collectHookCLIFlags(hookDirectorySettings)
	if err != nil {
		return
	}

	settings, err = p.parseSettings(allHookCLIFlags)
	if err != nil {
		if isHelpRequest(err) {
			return HelpCommand, nil, nil
		}
		return NoneCommand, nil, err
	}

	if settings.GeneralSettings.Version {
		// If user specified --version or -v, print the version and quit.
		return VersionCommand, nil, nil
	}

	return RunCommand, settings, nil
}

// Check if a given error is a request to display the help.
func isHelpRequest(err error) bool {
	var flagsError *flags.Error
	if errors.As(err, &flagsError) {
		if flagsError.Type == flags.ErrHelp {
			return true
		}
	}
	return false
}

// Parses the CLI flags related to the environment file.
func (p *CLIParser) parseEnvironmentFileSettings() (*EnvironmentFileSettings, error) {
	// Process command line flags.
	// Process the environment file flag.
	envFileSettings := &EnvironmentFileSettings{}
	parser := flags.NewParser(envFileSettings, flags.IgnoreUnknown)
	parser.ShortDescription = p.shortDescription
	parser.LongDescription = p.longDescription

	if _, err := parser.Parse(); err != nil {
		err = errors.Wrap(err, "invalid CLI argument")
		return nil, err
	}
	return envFileSettings, nil
}

// Loads the environment file content to the environment dictionary of the
// current process.
func (p *CLIParser) loadEnvironmentFile(envFileSettings *EnvironmentFileSettings) error {
	if !envFileSettings.UseEnvFile {
		// Nothing to do.
		return nil
	}

	err := storkutil.LoadEnvironmentFileToSetter(
		envFileSettings.EnvFile,
		storkutil.NewProcessEnvironmentVariableSetter(),
	)
	if err != nil {
		err = errors.WithMessagef(err, "invalid environment file: '%s'", envFileSettings.EnvFile)
		return err
	}

	// Reconfigures logging using new environment variables.
	storkutil.SetupLogging()

	return nil
}

// Parses the CLI flags related to the location of the hook directory.
func (p *CLIParser) parseHookDirectory() (*HookDirectorySettings, error) {
	// Process the hook directory location.
	hookDirectorySettings := &HookDirectorySettings{}
	parser := flags.NewParser(hookDirectorySettings, flags.IgnoreUnknown)
	parser.ShortDescription = p.shortDescription
	parser.LongDescription = p.longDescription

	if _, err := parser.Parse(); err != nil {
		err = errors.Wrap(err, "invalid CLI argument")
		return nil, err
	}
	return hookDirectorySettings, nil
}

// Extracts the CLI flags from the hooks.
func (p *CLIParser) collectHookCLIFlags(hookDirectorySettings *HookDirectorySettings) (map[string]hooks.HookSettings, error) {
	allCLIFlags := map[string]hooks.HookSettings{}
	stat, err := os.Stat(hookDirectorySettings.HookDirectory)
	switch {
	case err == nil && stat.IsDir():
		// Gather the hook flags.
		hookWalker := hooksutil.NewHookWalker()
		allCLIFlags, err = hookWalker.CollectCLIFlags(
			hooks.HookProgramServer,
			hookDirectorySettings.HookDirectory,
		)
		if err != nil {
			err = errors.WithMessage(err, "cannot collect the prototypes of the hook settings")
			return nil, err
		}
	case err == nil && !stat.IsDir():
		// Hook directory is not a directory.
		err = errors.Errorf(
			"the provided hook directory path is not pointing to a directory: %s",
			hookDirectorySettings.HookDirectory,
		)
		return nil, err
	case errors.Is(err, os.ErrNotExist):
		// Hook directory doesn't exist. Skip and continue.
		break
	default:
		// Unexpected problem.
		err = errors.Wrapf(err,
			"cannot stat the hook directory: %s",
			hookDirectorySettings.HookDirectory,
		)
		return nil, err
	}

	return allCLIFlags, nil
}

// Prepare conventional namespaces for the CLI flags and environment
// variables.
// CLI flags:
//   - Have a component derived from the hook filename
//   - Contains none upper cases, dots or spaces
//   - Underscores are replaced with dashes
//
// Environment variables:
//   - Starts with Stork-specific prefix
//   - Have a component derived from the hook filename
//   - Contains none lower cases, dots or spaces
//   - Dashes are replaced with underscored
func getHookNamespaces(hookName string) (flagNamespace, envNamespace string) {
	// Trim the app-specific prefix for simplicity.
	hookName, _ = strings.CutPrefix(hookName, "stork-server-")

	// Replace all invalid characters with dashes.
	hookName = strings.ReplaceAll(hookName, " ", "-")
	hookName = strings.ReplaceAll(hookName, ".", "-")

	flagNamespace = strings.ReplaceAll(hookName, "_", "-")
	flagNamespace = strings.ToLower(flagNamespace)

	// Prepend the common prefix for environment variables.
	envNamespace = "STORK_SERVER_HOOK_" + strings.ReplaceAll(hookName, "-", "_")
	envNamespace = strings.ToUpper(envNamespace)
	return
}

// Parses all CLI flags including the hooks-related ones.
func (p *CLIParser) parseSettings(allHooksCLIFlags map[string]hooks.HookSettings) (*Settings, error) {
	settings := newSettings()

	parser := flags.NewParser(settings.GeneralSettings, flags.Default)
	parser.ShortDescription = p.shortDescription
	parser.LongDescription = p.longDescription

	databaseFlags := &dbops.DatabaseCLIFlags{}
	// Process Database specific args.
	_, err := parser.AddGroup("Database ConnectionFlags", "", databaseFlags)
	if err != nil {
		err = errors.Wrap(err, "cannot add the database group")
		return nil, err
	}

	// Process ReST API specific args.
	_, err = parser.AddGroup("HTTP ReST Server Flags", "", settings.RestAPISettings)
	if err != nil {
		err = errors.Wrap(err, "cannot add the ReST group")
		return nil, err
	}

	// Process agent comm specific args.
	_, err = parser.AddGroup("Agents Communication Flags", "", settings.AgentsSettings)
	if err != nil {
		err = errors.Wrap(err, "cannot add the agents group")
		return nil, err
	}

	// Append hook flags.
	for hookName, cliFlags := range allHooksCLIFlags {
		if cliFlags == nil {
			continue
		}
		group, err := parser.AddGroup(fmt.Sprintf("Hook '%s' Flags", hookName), "", cliFlags)
		if err != nil {
			err = errors.Wrapf(err, "invalid settings for the '%s' hook", hookName)
			return nil, err
		}

		flagNamespace, envNamespace := getHookNamespaces(hookName)
		group.EnvNamespace = envNamespace
		group.Namespace = flagNamespace
	}

	// Check if there are no two groups with the same namespace.
	// It may happen if the one of the hooks has the expected common prefix,
	// but another one doesn't. For example, if we have two hooks named:
	//   - stork-server-ldap
	//   - ldap
	// Both of them will have the same namespace: ldap.
	// We suppose it will be a rare case, so we just return an error.
	groupNamespaces := make(map[string]any)
	for _, group := range parser.Groups() {
		if group.Namespace == "" {
			// Non-hook group. Skip.
			continue
		}
		_, exist := groupNamespaces[group.Namespace]
		if exist {
			return nil, errors.Errorf(
				"There are two hooks using the same configuration namespace "+
					"in the CLI flags: '%s'. The hook libraries for the "+
					"Stork server should use the following naming pattern, "+
					"e.g. 'stork-server-%s.so' instead of just '%s.so'",
				group.Namespace, group.Namespace, group.Namespace,
			)
		}
		groupNamespaces[group.Namespace] = nil
	}

	// Do args parsing.
	if _, err = parser.Parse(); err != nil {
		err = errors.Wrap(err, "cannot parse the CLI flags")
		return nil, err
	}

	settings.DatabaseSettings, err = databaseFlags.ConvertToDatabaseSettings()
	if err != nil {
		return nil, err
	}
	settings.HooksSettings = allHooksCLIFlags

	return settings, nil
}
