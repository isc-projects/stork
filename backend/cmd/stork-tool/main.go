package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"isc.org/stork"
	"isc.org/stork/hooksutil"
	"isc.org/stork/server/certs"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Random hash size in the generated password.
const passwordGenRandomLength = 24

// Establish connection to a database with opts from command line.
// Returns the database instance. It must be closed by caller.
func getDBConn(rawFlags *cli.Context) *dbops.PgDB {
	flags := &dbops.DatabaseCLIFlags{}
	flags.ReadFromCLI(rawFlags)
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
func runDBCreate(context *cli.Context) {
	flags := &dbops.DatabaseCLIFlagsWithMaintenance{}
	flags.ReadFromCLI(context)

	var err error

	// Prepare logging fields.
	logFields := log.Fields{
		"database_name": flags.DBName,
		"user":          flags.User,
	}

	// Check if the password has been specified explicitly. Otherwise,
	// generate the password.
	password := flags.Password
	if len(password) == 0 {
		password, err = storkutil.Base64Random(passwordGenRandomLength)
		if err != nil {
			log.WithError(err).Fatal("Failed to generate random database password")
		}
		// Only log the password if it has been generated. Otherwise, the
		// user should know the password.
		logFields["password"] = password
		flags.Password = password
	}

	// Connect to the postgres database using admin credentials.
	settings, err := flags.ConvertToMaintenanceDatabaseSettings()
	if err != nil {
		log.WithError(err).Fatal("Invalid database settings")
	}

	// Try to create the database and the user with access using
	// specified password.
	err = dbops.CreateDatabase(
		*settings,
		flags.DBName,
		flags.User,
		flags.Password,
		context.Bool("force"),
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
func runDBMigrate(settings *cli.Context, command, version string) {
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

	traceSQL := settings.String("db-trace-queries")
	if traceSQL != "" {
		log.Infof("SQL queries tracing set to %s", traceSQL)
	}

	db := getDBConn(settings)

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
func runCertExport(settings *cli.Context) error {
	db := getDBConn(settings)
	defer db.Close()

	return certs.ExportSecret(db, settings.String("object"), settings.String("file"))
}

// Execute cert import command.
func runCertImport(settings *cli.Context) error {
	db := getDBConn(settings)
	defer db.Close()

	return certs.ImportSecret(db, settings.String("object"), settings.String("file"))
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
func runHookInspect(settings *cli.Context) error {
	hookPath := settings.String("path")
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
func runStaticViewDeploy(settings *cli.Context, outFilename string) error {
	// Basic checks on the input file.
	inFilename := settings.String("file")
	if _, err := os.Stat(inFilename); err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			// This is a frequent error returned when user specified wrong
			// input filename. Let's handle this error to produce our own
			// error message.
			return errors.Errorf("input file '%s' does not exist", inFilename)
		default:
			// Other errors are more rare and it is overkill to handle of them.
			// Let's rely on the stat() function error message with the stack
			// trace appended.
			return errors.WithStack(err)
		}
	}
	// Get the directory where our file is to be copied.
	outDirectory, err := getOrLocateStaticPageContentDir(settings)
	if err != nil {
		return err
	}
	// Create the destination path by concatenating the output directory
	// and the file name specified as the function arguments.
	outFilename = path.Join(outDirectory, outFilename)

	// Open the input file name for reading.
	inFile, err := os.Open(inFilename)
	if err != nil {
		return errors.Wrapf(err, "failed to open input file '%s'", inFilename)
	}
	defer inFile.Close()
	reader := bufio.NewReader(inFile)

	// Open the output file for writing.
	outFile, err := os.OpenFile(outFilename, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return errors.Wrapf(err, "failed to open output file '%s'", outFilename)
	}
	defer outFile.Close()
	writer := bufio.NewWriter(outFile)

	// Copy the file.
	_, err = io.Copy(writer, reader)
	if err != nil {
		return errors.Wrapf(err, "failed to copy file '%s' to '%s'", inFilename, outFilename)
	}
	return nil
}

// Undeploy specified static file view from assets/static-page-content.
func runStaticViewUndeploy(settings *cli.Context, filename string) error {
	// Get the directory where our file is to be copied.
	directory, err := getOrLocateStaticPageContentDir(settings)
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
func getOrLocateStaticPageContentDir(settings *cli.Context) (string, error) {
	// Get the directory where our file is to be copied.
	directory := settings.String("rest-static-files-dir")
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

// Parse the general flag definitions into the objects compatible with the CLI library.
func parseFlagDefinitions(flagDefinitions []*dbops.CLIFlagDefinition) ([]cli.Flag, error) {
	var flags []cli.Flag
	for _, definition := range flagDefinitions {
		var flag cli.Flag

		var aliases []string
		if definition.Short != "" {
			aliases = append(aliases, definition.Short)
		}

		var envVars []string
		if definition.EnvironmentVariable != "" {
			envVars = append(envVars, definition.EnvironmentVariable)
		}

		if definition.Kind == reflect.Int {
			valueInt, err := strconv.ParseInt(definition.Default, 10, 0)
			if err != nil {
				return nil, errors.Wrapf(
					err, "invalid default value ('%s') for parameter ('%s')",
					definition.Default, definition.Long,
				)
			}

			flag = &cli.Int64Flag{
				Name:    definition.Long,
				Aliases: aliases,
				Usage:   definition.Description,
				EnvVars: envVars,
				Value:   valueInt,
			}
		} else {
			flag = &cli.StringFlag{
				Name:    definition.Long,
				Aliases: aliases,
				Usage:   definition.Description,
				EnvVars: envVars,
				Value:   definition.Default,
			}
		}

		flags = append(flags, flag)
	}

	return flags, nil
}

// Prepare urfave cli app with all flags and commands defined.
func setupApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Version)
	}

	dbFlags, err := parseFlagDefinitions((*dbops.DatabaseCLIFlags)(nil).ConvertToCLIFlagDefinitions())
	if err != nil {
		log.WithError(err).Fatal("Invalid database CLI flag definitions")
	}

	dbCreateFlags, err := parseFlagDefinitions((*dbops.DatabaseCLIFlagsWithMaintenance)(nil).ConvertToCLIFlagDefinitions())
	if err != nil {
		log.WithError(err).Fatal("Invalid create database CLI flag definitions")
	}

	dbCreateFlags = append(dbCreateFlags, &cli.BoolFlag{
		Name:    "force",
		Usage:   "Recreate the database and the user if they exist",
		Aliases: []string{"f"},
	})

	var dbVerFlags []cli.Flag
	dbVerFlags = append(dbVerFlags, dbFlags...)
	dbVerFlags = append(dbVerFlags,
		&cli.StringFlag{
			Name:    "version",
			Usage:   "Target database schema version (optional)",
			Aliases: []string{"t"},
			EnvVars: []string{"STORK_TOOL_DB_VERSION"},
		})

	var certExportFlags []cli.Flag
	certExportFlags = append(certExportFlags, dbFlags...)
	certExportFlags = append(certExportFlags,
		&cli.StringFlag{
			Name:     "object",
			Usage:    "The object to dump; it can be one of 'cakey', 'cacert', 'srvkey', 'srvcert', 'srvtkn'",
			Required: true,
			Aliases:  []string{"f"},
			EnvVars:  []string{"STORK_TOOL_CERT_OBJECT"},
		},
		&cli.StringFlag{
			Name:    "file",
			Usage:   "The file location where the object should be saved; if not provided, then object is printed to stdout",
			Aliases: []string{"o"},
			EnvVars: []string{"STORK_TOOL_CERT_FILE"},
		})

	var certImportFlags []cli.Flag
	certImportFlags = append(certImportFlags, dbFlags...)
	certImportFlags = append(certImportFlags,
		&cli.StringFlag{
			Name:     "object",
			Usage:    "The object to dump; it can be one of 'cakey', 'cacert', 'srvkey', 'srvcert', 'srvtkn'",
			Required: true,
			Aliases:  []string{"f"},
			EnvVars:  []string{"STORK_TOOL_CERT_OBJECT"},
		},
		&cli.StringFlag{
			Name:    "file",
			Usage:   "The file location from which the object will be read; if not provided, then the object is read from stdin",
			Aliases: []string{"i"},
			EnvVars: []string{"STORK_TOOL_CERT_FILE"},
		})

	hookInspectFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     "path",
			Usage:    "The hook file or directory path",
			Required: true,
			Aliases:  []string{"p"},
			EnvVars:  []string{"STORK_TOOL_HOOK_PATH"},
		},
	}

	loginScreenWelcomeDeployFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     "file",
			Usage:    "HTML source file with a custom welcome message",
			Required: true,
			Aliases:  []string{"i"},
			EnvVars:  []string{"STORK_TOOL_LOGIN_SCREEN_WELCOME_FILE"},
		},
		&cli.StringFlag{
			Name:    "rest-static-files-dir",
			Usage:   "The directory with static files for the UI; if not provided the tool will try to use default locations",
			Aliases: []string{"d"},
			EnvVars: []string{"STORK_TOOL_REST_STATIC_FILES_DIR"},
		},
	}

	loginScreenWelcomeUndeployFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "rest-static-files-dir",
			Usage:   "The directory with static files for the UI; if not provided the tool will try to use default locations",
			Aliases: []string{"d"},
			EnvVars: []string{"STORK_TOOL_REST_STATIC_FILES_DIR"},
		},
	}

	cli.HelpFlag = &cli.BoolFlag{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   "Show help",
	}

	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Print the version",
	}

	app := &cli.App{
		Name:  "Stork Tool",
		Usage: "A tool for managing Stork Server.",
		Description: `The tool operates in four areas:

   - Certificate Management - it allows for exporting Stork Server keys, certificates,
     and tokens that are used to secure communication between the Stork Server
     and Stork Agents;

   - Database Creation - it facilitates creating a new database for the Stork Server,
     and a user that can access this database with a generated password;

   - Database Migration - it allows for performing database schema migrations,
     overwriting the db schema version and getting its current value;

   - Static Views Deployment - it allows for setting custom content in selected
     Stork views (e.g., custom welcome message on the login page).`,
		Version:  stork.Version,
		HelpName: "stork-tool",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "",
				Usage:   "Logging level can be specified using env variable only. Allowed values: are DEBUG, INFO, WARN, ERROR",
				Value:   "INFO",
				EnvVars: []string{"STORK_LOG_LEVEL"},
			},
		},
		Commands: []*cli.Command{
			// DATABASE CREATION COMMANDS
			{
				Name:        "db-create",
				Usage:       "Create new Stork database",
				UsageText:   "stork-tool db-create [options for db creation] -f",
				Description: ``,
				Flags:       dbCreateFlags,
				Category:    "Database Creation",
				Action: func(c *cli.Context) error {
					runDBCreate(c)
					return nil
				},
			},
			{
				Name:        "db-password-gen",
				Usage:       "Generate random Stork database password",
				UsageText:   "stork-tool db-password-gen",
				Description: ``,
				Flags:       []cli.Flag{},
				Category:    "Database Creation",
				Action: func(c *cli.Context) error {
					runDBPasswordGen()
					return nil
				},
			},
			// DATABASE MIGRATION COMMANDS
			{
				Name:        "db-init",
				Usage:       "Create schema versioning table in the database",
				UsageText:   "stork-tool db-init [options for db connection]",
				Description: ``,
				Flags:       dbFlags,
				Category:    "Database Migration",
				Action: func(c *cli.Context) error {
					runDBMigrate(c, "init", "")
					return nil
				},
			},
			{
				Name:        "db-up",
				Usage:       "Run all available migrations or use -t to specify version",
				UsageText:   "stork-tool db-up [options for db connection] [-t version]",
				Description: ``,
				Flags:       dbVerFlags,
				Category:    "Database Migration",
				Action: func(c *cli.Context) error {
					runDBMigrate(c, "up", c.String("version"))
					return nil
				},
			},
			{
				Name:        "db-down",
				Usage:       "Revert last migration or use -t to specify version to downgrade to",
				UsageText:   "stork-tool db-down [options for db connection] [-t version]",
				Description: ``,
				Flags:       dbVerFlags,
				Category:    "Database Migration",
				Action: func(c *cli.Context) error {
					runDBMigrate(c, "down", c.String("version"))
					return nil
				},
			},
			{
				Name:        "db-reset",
				Usage:       "Revert all migrations",
				UsageText:   "stork-tool db-reset [options for db connection]",
				Description: ``,
				Flags:       dbFlags,
				Category:    "Database Migration",
				Action: func(c *cli.Context) error {
					runDBMigrate(c, "reset", "")
					return nil
				},
			},
			{
				Name:        "db-version",
				Usage:       "Print current migration version",
				UsageText:   "stork-tool db-version [options for db connection]",
				Description: ``,
				Flags:       dbFlags,
				Category:    "Database Migration",
				Action: func(c *cli.Context) error {
					runDBMigrate(c, "version", "")
					return nil
				},
			},
			{
				Name:        "db-set-version",
				Usage:       "Set database version without running migrations",
				UsageText:   "stork-tool db-set-version [options for db connection] [-t version]",
				Description: ``,
				Flags:       dbVerFlags,
				Category:    "Database Migration",
				Action: func(c *cli.Context) error {
					runDBMigrate(c, "set_version", c.String("version"))
					return nil
				},
			},
			// CERTIFICATE MANAGEMENT
			{
				Name:        "cert-export",
				Usage:       "Export certificate or other secret data",
				UsageText:   "stork-tool cert-export [options for db connection] [-f object] [-o filename]",
				Description: ``,
				Flags:       certExportFlags,
				Category:    "Certificates Management",
				Action:      runCertExport,
			},
			{
				Name:        "cert-import",
				Usage:       "Import certificate or other secret data",
				UsageText:   "stork-tool cert-import [options for db connection] [-f object] [-i filename]",
				Description: ``,
				Flags:       certImportFlags,
				Category:    "Certificates Management",
				Action:      runCertImport,
			},
			{
				Name:        "hook-inspect",
				Usage:       "Prints details about hooks",
				UsageText:   "stork-tool hook-inspect -p file-or-directory",
				Description: "",
				Flags:       hookInspectFlags,
				Action:      runHookInspect,
			},
			// STATIC VIEWS DEPLOYMENT
			{
				Name:        "deploy-login-page-welcome",
				Usage:       "Deploy custom welcome message on the login page",
				UsageText:   "stork-tool deploy-login-page-welcome [-i filename] [-d directory]",
				Description: ``,
				Flags:       loginScreenWelcomeDeployFlags,
				Category:    "Static Views Deployment",
				Action: func(c *cli.Context) error {
					return runStaticViewDeploy(c, "login-screen-welcome.html")
				},
			},
			{
				Name:        "undeploy-login-page-welcome",
				Usage:       "Undeploy custom welcome message from the login page",
				UsageText:   "stork-tool undeploy-login-page-welcome [-d directory]",
				Description: ``,
				Flags:       loginScreenWelcomeUndeployFlags,
				Category:    "Static Views Deployment",
				Action: func(c *cli.Context) error {
					return runStaticViewUndeploy(c, "login-screen-welcome.html")
				},
			},
		},
	}

	return app
}

func main() {
	// Setup logging
	storkutil.SetupLogging()

	app := setupApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
