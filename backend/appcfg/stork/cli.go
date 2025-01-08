package storkconfig

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
func NewCLIParser(parser *flags.Parser, app string, onLoadEnvironmentFileCallback func()) *CLIParser {
	if app != "server" && app != "agent" {
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
func (p *CLIParser) Parse() (*HookDirectorySettings, GroupedHookCLIFlags, bool, error) {
	hookDirectorySettings, allHookFLags, err := p.bootstrap()
	if err != nil {
		if isHelpRequest(err) {
			return nil, nil, true, nil
		}
		return nil, nil, false, err
	}

	err = p.parse()
	if err != nil {
		if isHelpRequest(err) {
			return nil, nil, true, nil
		}
		return nil, nil, false, err
	}
	return hookDirectorySettings, allHookFLags, false, nil
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
func (p *CLIParser) bootstrap() (*HookDirectorySettings, GroupedHookCLIFlags, error) {
	// Environment variables.
	envFileSettings := &environmentFileSettings{}
	envParser := p.createSubParser(envFileSettings)
	if _, err := envParser.Parse(); err != nil {
		return nil, nil, err
	}
	err := p.loadEnvironmentFile(envFileSettings)
	if err != nil {
		return nil, nil, err
	}

	// Process the hook directory location.
	hookDirectorySettings := &HookDirectorySettings{}
	hookParser := p.createSubParser(hookDirectorySettings)
	if _, err := hookParser.Parse(); err != nil {
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

	// Collect all known environment variables.
	knownEnvironmentVariables := make(map[string]bool)
	for _, group := range p.parser.Groups() {
		for _, option := range group.Options() {
			knownEnvironmentVariables[option.EnvKeyWithNamespace()] = true
		}
	}

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
	knownEnvironmentVariables := make(map[string]bool)
	for _, group := range p.parser.Groups() {
		for _, option := range group.Options() {
			knownEnvironmentVariables[option.EnvKeyWithNamespace()] = true
		}
	}

	// The prefix that is used for the Stork-specific environment variables.
	applicationPrefix := fmt.Sprintf("STORK_%s_", strings.ToUpper(p.application))

	// Iterate over all system-wide environment variables.
	for _, env := range os.Environ() {
		key, _, ok := strings.Cut(env, "=")
		if !ok {
			// It should never happen.
			continue
		}

		if !strings.HasPrefix(key, applicationPrefix) {
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
		program := hooks.HookProgramServer
		if p.application == "agent" {
			program = hooks.HookProgramAgent
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
