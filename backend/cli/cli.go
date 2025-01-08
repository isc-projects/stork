package cli

import (
	"fmt"
	"os"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
type HookDirectorySettings struct {
	HookDirectory string `long:"hook-directory" description:"The path to the hook directory" env:"STORK_@APP_HOOK_DIRECTORY" default:"/usr/lib/stork-@APP/hooks"`
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
// Accepts the main parser. It should be configured with the application
// settings.
// The application name is used to construct the namespaces for the CLI flags
// and environment variables. It should be either 'server', 'agent', 'tool'.
// The callback is called when the environment file is loaded. Its purpose is
// to allow reconfiguring the logging using the new environment variables as
// soon as they are available.
func NewCLIParser(parser *flags.Parser, app string, onLoadEnvironmentFileCallback func()) *CLIParser {
	if app != "server" && app != "agent" && app != "tool" {
		// Programming error.
		panic("invalid application name")
	}

	return &CLIParser{
		parser:                        parser,
		application:                   strings.ToLower(app),
		onLoadEnvironmentFileCallback: onLoadEnvironmentFileCallback,
	}
}

// Parse the command line arguments into Stork-specific GO structures.
// At the end, it composes the CLI parser from all the flags and runs it.
// Returns a hook directory settings, hook settings extracted from the hooks,
// flag indication if the help was requested and an error if any.
// Accepts the command line arguments. The passed arguments should exclude
// the application name (the first argument).
func (p *CLIParser) Parse(args []string) (*HookDirectorySettings, GroupedHookCLIFlags, bool, error) {
	hookDirectorySettings, allHookFLags, err := p.bootstrap(args)
	if err != nil {
		if isHelpRequest(err) {
			return nil, nil, true, nil
		}
		return nil, nil, false, err
	}

	err = p.parse(args)
	if err != nil {
		if isHelpRequest(err) {
			return nil, nil, true, nil
		}
		return nil, nil, false, err
	}
	return hookDirectorySettings, allHookFLags, false, nil
}

// Parse the CLI flags stored in the main parser.
func (p *CLIParser) parse(args []string) (err error) {
	// Do args parsing.
	if _, err = p.parser.ParseArgs(args); err != nil {
		err = errors.Wrap(err, "cannot parse the CLI flags")
		return err
	}

	return nil
}

// It parses the settings related to an environment file and if the file
// is provided, the content is loaded.
// Next, it parses the hooks location and extracts their CLI flags.
// The hook flags are then merged with the core flags.
func (p *CLIParser) bootstrap(args []string) (*HookDirectorySettings, GroupedHookCLIFlags, error) {
	// Environment variables.
	envFileSettings := &environmentFileSettings{}
	envParser := p.createSubParser(envFileSettings)
	if _, err := envParser.ParseArgs(args); err != nil {
		return nil, nil, err
	}
	err := p.loadEnvironmentFile(envFileSettings)
	if err != nil {
		return nil, nil, err
	}

	// Process the hook directory location.
	hookDirectorySettings := &HookDirectorySettings{}
	hookParser := p.createSubParser(hookDirectorySettings)
	if _, err := hookParser.ParseArgs(args); err != nil {
		return nil, nil, err
	}

	allHookCLIFlags, err := p.collectHookCLIFlags(hookDirectorySettings)
	if err != nil {
		return nil, nil, err
	}
	err = p.mergeHookFlags(allHookCLIFlags)
	if err != nil {
		return nil, nil, err
	}

	// Append the parser-related flags to the main parser.
	group, err := p.parser.AddGroup("Environment File Flags", "", envFileSettings)
	if err != nil {
		err = errors.Wrap(err, "cannot add the environment file group")
		return nil, nil, err
	}
	p.substitutePlaceholdersInGroup(group)

	group, err = p.parser.AddGroup("Hook Directory Flags", "", hookDirectorySettings)
	if err != nil {
		err = errors.Wrap(err, "cannot add the hook directory group")
		return nil, nil, err
	}
	p.substitutePlaceholdersInGroup(group)

	// Verify the environment variables.
	err = p.verifyEnvironmentFile(envFileSettings)
	if err != nil {
		return nil, nil, err
	}

	p.verifySystemEnvironmentVariables()

	return hookDirectorySettings, allHookCLIFlags, nil
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

	parser.Name = p.parser.Name
	parser.ShortDescription = p.parser.ShortDescription
	parser.LongDescription = p.parser.LongDescription
	return parser
}

// Substitutes the placeholders in the defaults and environment variable names.
func (p *CLIParser) substitutePlaceholders(parser *flags.Parser) {
	for _, group := range parser.Groups() {
		p.substitutePlaceholdersInGroup(group)
	}
}

// Substitutes the placeholders in the defaults and environment variable names
// in a group of flags.
func (p *CLIParser) substitutePlaceholdersInGroup(group *flags.Group) {
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

// Verifies if all environment variables in the environment file are known.
// Prints a warning if any of the environment variables is unknown.
func (p *CLIParser) verifyEnvironmentFile(envFileSettings *environmentFileSettings) error {
	if !envFileSettings.UseEnvFile {
		// Nothing to do.
		return nil
	}

	// Load the environment file content.
	entries, err := storkutil.LoadEnvironmentFile(envFileSettings.EnvFile)
	if err != nil {
		err = errors.WithMessagef(err, "invalid environment file: '%s'", envFileSettings.EnvFile)
		return err
	}

	knownEnvironmentVariables := collectKnownEnvironmentVariables(p.parser.Command)

	// Check if all environment variables are known.
	for key := range entries {
		if _, exist := knownEnvironmentVariables[key]; !exist {
			log.Warnf(
				"Unknown environment variable: '%s' in the environment file: '%s'",
				key, envFileSettings.EnvFile,
			)
		}
	}

	return nil
}

// Verifies if the system-wide environment variables doesn't contain any
// unknown Stork-specific environment variables.
func (p *CLIParser) verifySystemEnvironmentVariables() {
	// Collect all known environment variables.
	knownEnvironmentVariables := collectKnownEnvironmentVariables(p.parser.Command)

	// Contains the prefixes of the Stork-specific environment variables.
	// Stork environment variables starts with the 'STORK' part and then
	// the context-specific part e.g.:
	//
	// - STORK_SERVER (server-specific environment variable)
	// - STORK_AGENT (agent-specific environment variable)
	// - STORK_DATABASE (database-specific environment variable)
	//
	// This function analyzes the system-wide environment variables. We cannot
	// assume that there are only environment variables related to one of the
	// Stork components. It should be allowed to have the agent, server, and
	// tool environment variables set in the same shell.
	//
	// The below code block allows us to ignore the environment variables from
	// other Stork components. There is an assumption that all components have
	// exactly the same environment variables for a given prefix/namespace.
	// For example, if the application utilizes the environment variables
	// prefixed with 'STORK_DATABASE_' then its settings specify exactly the
	// same environment variables as the other components.
	// I hope it is fair enough.
	var prefixes []string

	for environmentVariable := range knownEnvironmentVariables {
		parts := strings.SplitN(environmentVariable, "_", 3)
		if len(parts) < 3 {
			// The environment variable doesn't have the context-specific part.
			// The naming convention is violated.
			continue
		}
		if parts[0] != "STORK" {
			// The environment variable doesn't start with the 'STORK' part.
			// The naming convention is violated.
			continue
		}

		prefixes = append(prefixes, fmt.Sprintf("%s_%s_", parts[0], parts[1]))
	}

	// Iterate over all system-wide environment variables.
	for _, env := range os.Environ() {
		key, _, ok := strings.Cut(env, "=")
		if !ok {
			// It should never happen.
			continue
		}

		var isApplicationEnvironmentVariable bool
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				isApplicationEnvironmentVariable = true
				break
			}
		}
		if !isApplicationEnvironmentVariable {
			// Not a Stork-specific environment variable.
			continue
		}

		if _, exist := knownEnvironmentVariables[key]; exist {
			// Known environment variable.
			continue
		}

		log.Warnf("Unknown environment variable: '%s' set in a shell", key)
	}
}

// Extracts the CLI flags from the hooks.
func (p *CLIParser) collectHookCLIFlags(hookDirectorySettings *HookDirectorySettings) (map[string]hooks.HookSettings, error) {
	allCLIFlags := map[string]hooks.HookSettings{}
	stat, err := os.Stat(hookDirectorySettings.HookDirectory)
	switch {
	case err == nil && stat.IsDir():
		// Gather the hook flags.
		hookWalker := hooksutil.NewHookWalker()
		var program string
		switch p.application {
		case "server":
			program = hooks.HookProgramServer
		case "agent":
			program = hooks.HookProgramAgent
		case "tool":
			program = hooks.HookProgramTool
		default:
			// Programming error.
			return nil, errors.Errorf("unknown application name: %s", p.application)
		}

		allCLIFlags, err = hookWalker.CollectCLIFlags(
			program,
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
	envNamespace = fmt.Sprintf(
		"STORK_%s_HOOK_%s",
		application,
		strings.ReplaceAll(hookName, "-", "_"),
	)
	envNamespace = strings.ToUpper(envNamespace)
	return
}

// Collects all known environment variables from a parser. Returns a set of
// the full environment variable names.
func collectKnownEnvironmentVariables(parser *flags.Command) map[string]bool {
	knownEnvironmentVariables := make(map[string]bool)

	// The options of the main group of the top-level parser.
	for _, option := range parser.Options() {
		knownEnvironmentVariables[option.EnvKeyWithNamespace()] = true
	}

	// The groups of the top-level parser.
	for _, group := range parser.Groups() {
		for _, option := range group.Options() {
			knownEnvironmentVariables[option.EnvKeyWithNamespace()] = true
		}
	}

	// The subcommands of the top-level parser.
	for _, subcommand := range parser.Commands() {
		subcommandEnvironmentVariables := collectKnownEnvironmentVariables(subcommand)
		for key := range subcommandEnvironmentVariables {
			knownEnvironmentVariables[key] = true
		}
	}

	return knownEnvironmentVariables
}
