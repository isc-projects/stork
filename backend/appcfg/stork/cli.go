package storkconfig

import (
	"fmt"
	"os"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"isc.org/stork/hooks"
	"isc.org/stork/hooksutil"
	storkutil "isc.org/stork/util"
)

// Read environment file settings. It's parsed before the main settings.
type environmentFileSettings struct {
	EnvFile    string `long:"env-file" description:"Environment file location; applicable only if the use-env-file is provided" default:"/etc/stork/@APP.env"`
	UseEnvFile bool   `long:"use-env-file" description:"Read the environment variables from the environment file"`
}

// Read hook directory settings. They are parsed after environment file
// settings but before the main settings.
// It allows us to merge the hook flags with the core flags into a single output.
type hookDirectorySettings struct {
	HookDirectory string `long:"hook-directory" description:"The path to the hook directory" env:"STORK_@APP_HOOK_DIRECTORY" default:"/usr/lib/stork-@APP/hooks"`
}

type BaseSettings struct {
	environmentFileSettings
	hookDirectorySettings
}

// Defines the type for set of hook settings grouped by the hook name.
type GroupedHookCLIFlags map[string]hooks.HookSettings

// Stork server-specific CLI arguments/flags parser.
type CLIParser struct {
	parser                        *flags.Parser
	application                   string
	onLoadEnvironmentFileCallback func()
}

// Constructs CLI parser.
func NewCLIParser(parser *flags.Parser, app string, onLoadEnvironmentFileCallback func()) *CLIParser {
	return &CLIParser{
		parser:                        parser,
		application:                   strings.ToLower(app),
		onLoadEnvironmentFileCallback: onLoadEnvironmentFileCallback,
	}
}

// Parse the command line arguments into Stork-specific GO structures.
// At the end, it composes the CLI parser from all the flags and runs it.
func (p *CLIParser) Parse() (hookFlags GroupedHookCLIFlags, isHelp bool, err error) {
	allHookFLags, err := p.bootstrap()
	if err != nil {
		if isHelpRequest(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	err = p.parse()
	if err != nil {
		if isHelpRequest(err) {
			return nil, true, nil
		}
		return nil, false, err
	}
	return allHookFLags, false, nil
}

// Parse the CLI flags stored in the main parser.
func (p *CLIParser) parse() (err error) {
	// Do args parsing.
	if _, err = p.parser.Parse(); err != nil {
		err = errors.Wrap(err, "cannot parse the CLI flags")
		return err
	}

	return nil
}

// It parses the settings related to an environment file and if the file
// is provided, the content is loaded.
// Next, it parses the hooks location and extracts their CLI flags.
// The hook flags are then merged with the core flags.
func (p *CLIParser) bootstrap() (GroupedHookCLIFlags, error) {
	// Environment variables.
	envFileSettings := &environmentFileSettings{}
	envParser := p.createSubParser(envFileSettings)
	if _, err := envParser.Parse(); err != nil {
		return nil, err
	}
	err := p.loadEnvironmentFile(envFileSettings)
	if err != nil {
		return nil, err
	}

	// Process the hook directory location.
	hookDirectorySettings := &hookDirectorySettings{}
	hookParser := p.createSubParser(hookDirectorySettings)
	if _, err := hookParser.Parse(); err != nil {
		return nil, err
	}

	allHookCLIFlags, err := p.collectHookCLIFlags(hookDirectorySettings)
	if err != nil {
		return nil, err
	}
	err = p.mergeHookFlags(allHookCLIFlags)
	if err != nil {
		return nil, err
	}

	return allHookCLIFlags, nil
}

// Merges the CLI flags of the hooks with the core CLI flags.
func (p *CLIParser) mergeHookFlags(allHooksCLIFlags map[string]hooks.HookSettings) error {
	// Append hook flags.
	for hookName, cliFlags := range allHooksCLIFlags {
		if cliFlags == nil {
			continue
		}
		group, err := p.parser.AddGroup(fmt.Sprintf("Hook '%s' Flags", hookName), "", cliFlags)
		if err != nil {
			err = errors.Wrapf(err, "invalid settings for the '%s' hook", hookName)
			return err
		}

		flagNamespace, envNamespace := getHookNamespaces(p.application, hookName)
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
	for _, group := range p.parser.Groups() {
		if group.Namespace == "" {
			// Non-hook group. Skip.
			continue
		}
		_, exist := groupNamespaces[group.Namespace]
		if exist {
			return errors.Errorf(
				"There are two hooks using the same configuration namespace "+
					"in the CLI flags: '%s'. The hook libraries for the "+
					"Stork server should use the following naming pattern, "+
					"e.g. 'stork-server-%s.so' instead of just '%s.so'",
				group.Namespace, group.Namespace, group.Namespace,
			)
		}
		groupNamespaces[group.Namespace] = nil
	}

	return nil
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

// Creates a new parser for the CLI flags that parsers a set of flags related
// to the CLI parsing itself rather than the application settings and parses
// the flags specified in the tags of the provided structure.
// It inherits the descriptions from the main parser and substitutes the
// placeholders in the defaults and environment variable names.
func (p *CLIParser) createSubParser(settings any) *flags.Parser {
	parser := flags.NewParser(settings, flags.IgnoreUnknown)

	p.substitutePlaceholders(parser)

	parser.ShortDescription = p.parser.ShortDescription
	parser.LongDescription = p.parser.LongDescription
	return parser
}

// Substitutes the placeholders in the defaults and environment variable names.
func (p *CLIParser) substitutePlaceholders(parser *flags.Parser) {
	for _, group := range parser.Groups() {
		for _, option := range group.Options() {
			// Defaults.
			for i, d := range option.Default {
				option.Default[i] = strings.Replace(d, "@APP", p.application, 1)
			}

			// Environment variables.
			option.EnvDefaultKey = strings.Replace(
				option.EnvDefaultKey,
				"@APP",
				strings.ToUpper(p.application),
				1,
			)
		}
	}
}

// Loads the environment file content to the environment dictionary of the
// current process.
func (p *CLIParser) loadEnvironmentFile(envFileSettings *environmentFileSettings) error {
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

	// Call the callback when the environment file is loaded. It allows to
	// reconfigure the logging using the new environment variables.
	p.onLoadEnvironmentFileCallback()

	return nil
}

// Extracts the CLI flags from the hooks.
func (p *CLIParser) collectHookCLIFlags(hookDirectorySettings *hookDirectorySettings) (map[string]hooks.HookSettings, error) {
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
// Accepts the application (i.e., 'server' or 'agent') and the hook name.
//
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
func getHookNamespaces(application string, hookName string) (flagNamespace, envNamespace string) {
	// Trim the app-specific prefix for simplicity.
	prefix := fmt.Sprintf("stork-%s-", application)
	hookName, _ = strings.CutPrefix(hookName, prefix)

	// Replace all invalid characters with dashes.
	hookName = strings.ReplaceAll(hookName, " ", "-")
	hookName = strings.ReplaceAll(hookName, ".", "-")

	flagNamespace = strings.ReplaceAll(hookName, "_", "-")
	flagNamespace = strings.ToLower(flagNamespace)

	// Prepend the common prefix for environment variables.
	envNamespace = fmt.Sprintf("STORK_%s_HOOK_%s", application, hookName)
	envNamespace = "STORK_SERVER_HOOK_" + strings.ReplaceAll(hookName, "-", "_")
	envNamespace = strings.ToUpper(envNamespace)
	return
}

// // Parses all CLI flags including the hooks-related ones.
// func (p *CLIParser) parseSettings(allHooksCLIFlags map[string]hooks.HookSettings) (*Settings, error) {
// 	settings := newSettings()

// 	parser := flags.NewParser(settings.GeneralSettings, flags.Default)
// 	parser.ShortDescription = p.shortDescription
// 	parser.LongDescription = p.longDescription

// 	databaseFlags := &dbops.DatabaseCLIFlags{}
// 	// Process Database specific args.
// 	_, err := parser.AddGroup("Database ConnectionFlags", "", databaseFlags)
// 	if err != nil {
// 		err = errors.Wrap(err, "cannot add the database group")
// 		return nil, err
// 	}

// 	// Process ReST API specific args.
// 	_, err = parser.AddGroup("HTTP ReST Server Flags", "", settings.RestAPISettings)
// 	if err != nil {
// 		err = errors.Wrap(err, "cannot add the ReST group")
// 		return nil, err
// 	}

// 	// Process agent comm specific args.
// 	_, err = parser.AddGroup("Agents Communication Flags", "", settings.AgentsSettings)
// 	if err != nil {
// 		err = errors.Wrap(err, "cannot add the agents group")
// 		return nil, err
// 	}

// 	// Do args parsing.
// 	if _, err = parser.Parse(); err != nil {
// 		err = errors.Wrap(err, "cannot parse the CLI flags")
// 		return nil, err
// 	}

// 	settings.DatabaseSettings, err = databaseFlags.ConvertToDatabaseSettings()
// 	if err != nil {
// 		return nil, err
// 	}
// 	settings.HooksSettings = allHooksCLIFlags

// 	return settings, nil
// }
