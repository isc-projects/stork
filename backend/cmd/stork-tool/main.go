package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	storkconfig "isc.org/stork/appcfg/stork"
	"isc.org/stork/hooksutil"
	"isc.org/stork/server/certs"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Random hash size in the generated password.
const passwordGenRandomLength = 24

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

// Indicates if the command was specified.
func (s *cliCommand) isSpecified() bool {
	return s.commandSpecified
}

// The CLI flags not related to any specific command.
type GeneralCommand struct {
	cliCommand
	// If true, the version of the Stork tool is printed. It takes precedence
	// over all other commands and arguments.
	Version bool `short:"v" long:"version" description:"Show software version"`
}

// The CLI flags for the db-create command.
type DatabaseCreateCommand struct {
	cliCommand
	DatabaseSettings dbops.DatabaseCLIFlagsWithMaintenance
	Force            bool `long:"force" short:"f" description:"Recreate the database and the user if they exist" env:"STORK_TOOL_DB_FORCE"`
}

// The CLI flags for the db-init, db-up, db-down, db-reset, db-version commands.
type DatabaseCommand struct {
	cliCommand
	DatabaseSettings dbops.DatabaseCLIFlags
}

// The CLI flags for the db-up, db-down, and db-set-version commands.
type DatabaseVersionCommand struct {
	cliCommand
	DatabaseSettings dbops.DatabaseCLIFlags
	Version          string `long:"version" short:"t" description:"Target database schema version (optional)" env:"STORK_TOOL_DB_VERSION"`
}

// The CLI flags for the cert-import command.
type CertificateImportCommand struct {
	cliCommand
	DatabaseSettings dbops.DatabaseCLIFlags
	Object           string `long:"object" short:"f" description:"The object to import; it can be one of 'cakey', 'cacert', 'srvkey', 'srvcert', 'srvtkn'" env:"STORK_TOOL_CERT_OBJECT" choice:"cakey" choice:"cacert" choice:"srvkey" choice:"srvcert" choice:"srvtkn"`
	File             string `long:"file" short:"i" description:"The file location from which the object should be imported" env:"STORK_TOOL_CERT_FILE"`
}

// The CLI flags for the cert-export command.
type CertificateExportCommand struct {
	cliCommand
	DatabaseSettings dbops.DatabaseCLIFlags
	Object           string `long:"object" short:"f" description:"The object to dump; it can be one of 'cakey', 'cacert', 'srvkey', 'srvcert', 'srvtkn'" env:"STORK_TOOL_CERT_OBJECT" choice:"cakey" choice:"cacert" choice:"srvkey" choice:"srvcert" choice:"srvtkn"`
	File             string `long:"file" short:"o" description:"The file location where the object should be saved; if not provided, then object is printed to stdout" env:"STORK_TOOL_CERT_FILE"`
}

// The CLI flags for the hook-inspect command.
type HookInspectCommand struct {
	cliCommand
	HookPath string `long:"hook-path" short:"p" description:"The path to the hook file or directory" env:"STORK_TOOL_HOOK_PATH"`
}

// The CLI flags for the deploy-login-page-welcome command.
type LoginScreenWelcomeDeployCommand struct {
	cliCommand
	File               string `long:"file" short:"i" description:"HTML source file with a custom welcome message" env:"STORK_TOOL_LOGIN_SCREEN_WELCOME_FILE"`
	RestStaticFilesDir string `long:"rest-static-files-dir" short:"d" description:"The directory with static files for the UI; if not provided the tool will try to use default locations" env:"STORK_TOOL_REST_STATIC_FILES_DIR"`
}

// The CLI flags for the undeploy-login-page-welcome command.
type LoginScreenWelcomeUndeployCommand struct {
	cliCommand
	RestStaticFilesDir string `long:"rest-static-files-dir" short:"d" description:"The directory with static files for the UI; if not provided the tool will try to use default locations" env:"STORK_TOOL_REST_STATIC_FILES_DIR"`
}

// Establish connection to a database with opts from command line.
// Returns the database instance. It must be closed by caller.
func getDBConn(flags dbops.DatabaseCLIFlags) *dbops.PgDB {
	settings, err := flags.ConvertToDatabaseSettings()
	if err != nil {
		log.WithError(err).Fatal("Invalid database settings")
	}

	db, err := dbops.NewPgDBConn(settings)
	if err != nil {
		log.WithError(err).Fatal("Unexpected error")
	}

	// Theoretically, it should not happen but let's make sure in case someone
	// modifies the NewPgDB function.
	if db == nil {
		log.Fatal("Unable to create database instance")
	}
	return db
}

// Execute db-create command. It prepares new database for the Stork
// server. It also creates a user that can access this database using
// a generated or user-specified password and the pgcrypto extension.
func runDBCreate(command *DatabaseCreateCommand) {
	var err error

	// Prepare logging fields.
	logFields := log.Fields{
		"database_name": command.DatabaseSettings.DBName,
		"user":          command.DatabaseSettings.User,
	}

	// Check if the password has been specified explicitly. Otherwise,
	// generate the password.
	password := command.DatabaseSettings.Password
	if len(password) == 0 {
		password, err = storkutil.Base64Random(passwordGenRandomLength)
		if err != nil {
			log.WithError(err).Fatal("Failed to generate random database password")
		}
		// Only log the password if it has been generated. Otherwise, the
		// user should know the password.
		logFields["password"] = password
		command.DatabaseSettings.Password = password
	}

	// Connect to the postgres database using admin credentials.
	maintenanceSettings, err := command.DatabaseSettings.ConvertToMaintenanceDatabaseSettings()
	if err != nil {
		log.WithError(err).Fatal("Invalid database settings")
	}

	// Try to create the database and the user with access using
	// specified password.
	err = dbops.CreateDatabase(
		*maintenanceSettings,
		command.DatabaseSettings.DBName,
		command.DatabaseSettings.User,
		command.DatabaseSettings.Password,
		command.Force,
	)
	if err != nil {
		log.WithError(err).Fatal("Could not create the database and the user")
	}

	// Database setup successful.
	log.WithFields(logFields).Info("Created database and user for the server with the following credentials")
}

// Execute db-password-gen command. It generates random password that can be
// used for securing Stork database.
func runDBPasswordGen() {
	password, err := storkutil.Base64Random(passwordGenRandomLength)
	if err != nil {
		log.WithError(err).Fatal("Failed to generate random database password")
	}
	log.WithFields(log.Fields{
		"password": password,
	}).Info("Generated new database password")
}

// Execute DB migration command.
func runDBMigrate(databaseSettings dbops.DatabaseCLIFlags, command, version string) {
	// The up and down commands require special treatment. If the target version is specified
	// it must be appended to the arguments we pass to the go-pg migrations.
	var args []string
	args = append(args, command)
	if command == "up" && len(version) > 0 {
		args = append(args, version)
		log.Infof("Requested migration up to version %s", version)
	}
	if command == "down" && len(version) > 0 {
		args = append(args, version)
		log.Infof("Requested migration down to version %s", version)
	}
	if command == "set_version" {
		if version == "" {
			log.Fatal("Flag --version/-t is missing but required")
		}
		args = append(args, version)
		log.Infof("Requested setting version to %s", version)
	}

	traceSQL := databaseSettings.TraceSQL
	if traceSQL != "" {
		log.Infof("SQL queries tracing set to %s", traceSQL)
	}

	db := getDBConn(databaseSettings)

	oldVersion, newVersion, err := dbops.Migrate(db, args...)
	if err == nil && newVersion == 0 {
		// Init operation doesn't fetch the database version but it doesn't
		// change the version.
		newVersion, err = dbops.CurrentVersion(db)
		oldVersion = newVersion
	}
	_ = db.Close()
	if err != nil {
		log.WithError(err).Fatal("Failed to migrate database")
	}

	availVersion := dbops.AvailableVersion()

	switch {
	case newVersion != oldVersion:
		log.Infof("Migrated database from version %d to %d\n", oldVersion, newVersion)
	case newVersion == 0:
		log.Infof("Database schema is empty (version 0)")
	case availVersion == oldVersion:
		log.Infof("Database version is %d (up-to-date)\n", oldVersion)
	default:
		log.Infof("Database version is %d (new version %d available)\n", oldVersion, availVersion)
	}
}

// Execute cert export command.
func runCertExport(certificateCommand *CertificateExportCommand) error {
	db := getDBConn(certificateCommand.DatabaseSettings)
	defer db.Close()

	return certs.ExportSecret(db, certificateCommand.Object, certificateCommand.File)
}

// Execute cert import command.
func runCertImport(certificateCommand *CertificateImportCommand) error {
	db := getDBConn(certificateCommand.DatabaseSettings)
	defer db.Close()

	return certs.ImportSecret(db, certificateCommand.Object, certificateCommand.File)
}

// Inspect the hook file.
func inspectHookFile(path string, library *hooksutil.LibraryManager, err error) {
	if err != nil {
		log.
			WithField("file", path).
			Error(err)
		return
	}

	hookProgram, hookVersion, err := library.GetVersion()
	if err != nil {
		log.
			WithField("file", path).
			Error(err)
		return
	}

	log.
		WithField("file", path).
		Infof("Hook is compatible with %s@%s", hookProgram, hookVersion)
}

// Execute inspect hook command.
func runHookInspect(hookInspectCommand *HookInspectCommand) error {
	hookPath := hookInspectCommand.HookPath
	fileInfo, err := os.Stat(hookPath)
	if err != nil {
		return errors.Wrapf(err, "cannot stat the hook path: '%s'", hookPath)
	}

	walker := hooksutil.NewHookWalker()

	mode := fileInfo.Mode()
	switch {
	case mode.IsDir():
		return walker.WalkPluginLibraries(hookPath, func(path string, library *hooksutil.LibraryManager, err error) bool {
			inspectHookFile(path, library, err)
			return true
		})
	case mode.IsRegular():
		library, err := hooksutil.NewLibraryManager(hookPath)
		inspectHookFile(hookPath, library, err)
		return nil
	default:
		return errors.Errorf("unsupported file mode: '%s'", mode.String())
	}
}

// Deploy specified static file view into assets/static-page-content.
func runStaticViewDeploy(settings *LoginScreenWelcomeDeployCommand, outFilename string) (err error) {
	// Basic checks on the input file.
	inFilename := settings.File
	if _, err = os.Stat(inFilename); err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			// This is a frequent error returned when user specified wrong
			// input filename. Let's handle this error to produce our own
			// error message.
			err = errors.Errorf("input file '%s' does not exist", inFilename)
			return
		default:
			// Other errors are more rare and it is overkill to handle of them.
			// Let's rely on the stat() function error message with the stack
			// trace appended.
			err = errors.WithStack(err)
			return
		}
	}
	// Get the directory where our file is to be copied.
	var outDirectory string
	outDirectory, err = getOrLocateStaticPageContentDir(settings.RestStaticFilesDir)
	if err != nil {
		return
	}
	// Create the destination path by concatenating the output directory
	// and the file name specified as the function arguments.
	outFilename = filepath.Join(outDirectory, outFilename)

	// Open the input file name for reading.
	var inFile *os.File
	inFile, err = os.Open(inFilename)
	if err != nil {
		err = errors.Wrapf(err, "failed to open input file '%s'", inFilename)
		return
	}
	defer func() {
		if closeErr := inFile.Close(); closeErr != nil && err == nil {
			err = errors.Wrapf(closeErr, "failed to close input file '%s'", inFilename)
		}
	}()
	reader := bufio.NewReader(inFile)

	// Open the output file for writing.
	var outFile *os.File
	outFile, err = os.OpenFile(outFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		err = errors.Wrapf(err, "failed to open output file '%s'", outFilename)
		return
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil && err == nil {
			err = errors.Wrapf(closeErr, "failed to close output file '%s'", outFilename)
		}
	}()
	writer := bufio.NewWriter(outFile)

	// Copy the file.
	_, err = io.Copy(writer, reader)
	if err != nil {
		err = errors.Wrapf(err, "failed to copy file '%s' to '%s'", inFilename, outFilename)
	}
	return
}

// Undeploy specified static file view from assets/static-page-content.
func runStaticViewUndeploy(settings *LoginScreenWelcomeUndeployCommand, filename string) error {
	// Get the directory where our file is to be copied.
	directory, err := getOrLocateStaticPageContentDir(settings.RestStaticFilesDir)
	if err != nil {
		return err
	}
	// Get the target path by concatenating the directory and the file name
	// specified as the function argument.
	filename = path.Join(directory, filename)

	err = os.Remove(filename)
	return errors.Wrapf(err, "failed to remove file '%s'", filename)
}

// Reads the location of the static files from the settings and returns
// the path to assets/static-page-content relative to this path. If the
// path is not specified it tries to locate the static-page-content path
// relative to the stork-tool binary name.
func getOrLocateStaticPageContentDir(restStaticFilesDirectory string) (string, error) {
	// Get the directory where our file is to be copied.
	directory := restStaticFilesDirectory
	if directory == "" {
		// The directory hasn't been specified. Let's try to locate that directory
		// relative to the stork-tool binary location.
		executable, err := os.Executable()
		if err != nil {
			return "", errors.New("unable to locate static files directory; please use -d option to specify its location")
		}
		// Remove the binary name and the containing directory name.
		// Append the relative location of the static files.
		directory = path.Join(path.Dir(executable), "..", "/share/stork/www/assets/static-page-content/")
		if _, err := os.Stat(directory); err != nil {
			return "", errors.New("unable to locate destination directory; please use -d option to specify the correct path")
		}
		return directory, nil
	}
	// If the directory has been specified, let's check if it exists.
	// Also, check if the subdirectory where we store assets exists.
	for i, dir := range []string{directory, path.Join(directory, "/assets/static-page-content/")} {
		if _, err := os.Stat(dir); err != nil {
			switch {
			case errors.Is(err, fs.ErrNotExist):
				return "", errors.Errorf("directory '%s' does not exist", dir)
			default:
				return "", errors.WithStack(err)
			}
		}
		if i > 0 {
			directory = dir
		}
	}
	return directory, nil
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

// The main application structure.
type app struct {
	commandsToFunctions map[command]action
	parser              *flags.Parser
	generalCommand      *GeneralCommand
}

// Registers a command with the parser and associates it with the action.
// It shouldn't be called outside of the newApp function.
func (a *app) registerCommand(command, shortDescription string, data command, action action) {
	_, err := a.parser.AddCommand(command, shortDescription, "", data)
	if err != nil {
		log.WithError(err).Fatal("Failed to add command")
	}
	a.commandsToFunctions[data] = action
}

// Prepare CLI app with all flags and commands defined.
func newApp() *app {
	generalCommand := &GeneralCommand{}
	parser := flags.NewParser(generalCommand, flags.Default)

	app := &app{
		commandsToFunctions: make(map[command]action),
		parser:              parser,
		generalCommand:      generalCommand,
	}

	parser.Name = "Stork Tool"
	parser.SubcommandsOptional = true
	parser.ShortDescription = "A tool for managing Stork Server."
	parser.LongDescription = `The tool operates in four areas:

   - Certificate Management - it allows for exporting Stork Server keys, certificates,
     and tokens that are used to secure communication between the Stork Server
     and Stork Agents;

   - Database Creation - it facilitates creating a new database for the Stork Server,
     and a user that can access this database with a generated password;

   - Database Migration - it allows for performing database schema migrations,
     overwriting the db schema version and getting its current value;

   - Static Views Deployment - it allows for setting custom content in selected
     Stork views (e.g., custom welcome message on the login page).`

	// Disclaimer: The previous version grouped the commands into separate
	// sections. Unfortunately, the go-flags library does not support this
	// feature.

	// Database creation commands.
	databaseCreateCommand := &DatabaseCreateCommand{}
	app.registerCommand(
		"db-create", "Create new Stork database", databaseCreateCommand,
		func() {
			runDBCreate(databaseCreateCommand)
		},
	)

	databasePasswordGenCommand := &cliCommand{}
	app.registerCommand(
		"db-password-gen", "Generate random Stork database password",
		databasePasswordGenCommand, runDBPasswordGen,
	)

	databaseInitCommand := &DatabaseCommand{}
	app.registerCommand(
		"db-init", "Create schema versioning table in the database",
		databaseInitCommand, func() {
			runDBMigrate(databaseInitCommand.DatabaseSettings, "init", "")
		},
	)

	databaseUpCommand := &DatabaseVersionCommand{}
	app.registerCommand(
		"db-up", "Run all available migrations or use -t to specify version",
		databaseUpCommand, func() {
			runDBMigrate(
				databaseUpCommand.DatabaseSettings,
				"up",
				databaseUpCommand.Version,
			)
		},
	)

	databaseDownCommand := &DatabaseVersionCommand{}
	app.registerCommand(
		"db-down", "Revert last migration or use -t to specify version to downgrade to",
		databaseDownCommand, func() {
			runDBMigrate(
				databaseDownCommand.DatabaseSettings,
				"down",
				databaseDownCommand.Version,
			)
		},
	)

	databaseResetCommand := &DatabaseCommand{}
	app.registerCommand(
		"db-reset", "Reset the database to the initial state",
		databaseResetCommand, func() {
			runDBMigrate(databaseResetCommand.DatabaseSettings, "reset", "")
		},
	)

	databaseVersionCommand := &DatabaseCommand{}
	app.registerCommand(
		"db-version", "Get the current database schema version",
		databaseVersionCommand, func() {
			runDBMigrate(databaseVersionCommand.DatabaseSettings, "version", "")
		},
	)

	databaseSetVersionCommand := &DatabaseVersionCommand{}
	app.registerCommand(
		"db-set-version", "Set the database schema version",
		databaseSetVersionCommand, func() {
			runDBMigrate(
				databaseSetVersionCommand.DatabaseSettings,
				"set_version",
				databaseSetVersionCommand.Version,
			)
		},
	)

	// Certificate management commands.
	certificateExportCommand := &CertificateExportCommand{}
	app.registerCommand(
		"cert-export", "Export Stork Server keys, certificates, and tokens",
		certificateExportCommand, func() {
			err := runCertExport(certificateExportCommand)
			if err != nil {
				log.WithError(err).Fatal("Failed to export the certificate")
			}
		},
	)

	certificateImportCommand := &CertificateImportCommand{}
	app.registerCommand(
		"cert-import", "Import Stork Server keys, certificates, and tokens",
		certificateImportCommand, func() {
			err := runCertImport(certificateImportCommand)
			if err != nil {
				log.WithError(err).Fatal("Failed to import the certificate")
			}
		},
	)

	// Hook inspection command.
	hookInspectCommand := &HookInspectCommand{}
	app.registerCommand(
		"hook-inspect", "Inspect the hook file or directory",
		hookInspectCommand, func() {
			err := runHookInspect(hookInspectCommand)
			if err != nil {
				log.WithError(err).Fatal("Failed to inspect the hook")
			}
		},
	)

	// Static views deployment commands.
	loginScreenWelcomeDeployCommand := &LoginScreenWelcomeDeployCommand{}
	app.registerCommand(
		"deploy-login-page-welcome",
		"Deploy custom welcome message on the login screen",
		loginScreenWelcomeDeployCommand, func() {
			err := runStaticViewDeploy(
				loginScreenWelcomeDeployCommand, "login-screen-welcome.html",
			)
			if err != nil {
				log.WithError(err).
					Fatal("Failed to deploy the custom welcome message")
			}
		},
	)

	loginScreenWelcomeUndeployCommand := &LoginScreenWelcomeUndeployCommand{}
	app.registerCommand(
		"undeploy-login-page-welcome",
		"Undeploy custom welcome message on the login screen",
		loginScreenWelcomeUndeployCommand, func() {
			err := runStaticViewUndeploy(
				loginScreenWelcomeUndeployCommand, "login-screen-welcome.html",
			)
			if err != nil {
				log.WithError(err).
					Fatal("Failed to undeploy the custom welcome message")
			}
		},
	)

	return app
}

// Starts the application with the provided arguments.
func (a *app) run(args []string) error {
	// Parse command line arguments.
	appParser := storkconfig.NewCLIParser(a.parser, "tool", func() {
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
	if a.generalCommand.Version {
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

	return errors.New("no command specified")
}

// The main function of the Stork tool.
func main() {
	// Setup logging
	storkutil.SetupLogging()

	app := newApp()
	err := app.run(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
