package main

import (
	"fmt"
	"os"
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

	db, err := dbops.NewPgDBConn(settings)
	if err != nil {
		log.WithError(err).Fatal("Unexpected error")
	}

	// Try to create the database and the user with access using
	// specified password.
	err = dbops.CreateDatabase(db, flags.DBName, flags.User, flags.Password, context.Bool("force"))
	// Close the current connection. We will have to connect to the
	// newly created database instead to create the pgcrypto extension.
	db.Close()
	if err != nil {
		log.Fatalf("%s", err)
	}

	// Re-use all admin credentials but connect to the new database.
	settings, err = flags.ConvertToDatabaseSettingsWithMaintenanceCredentials()
	if err != nil {
		log.WithError(err).Fatal("Invalid maintenance database settings")
	}

	db, err = dbops.NewPgDBConn(settings)
	if err != nil {
		log.WithError(err).Fatal("Unexpected error")
	}
	defer db.Close()

	// Try to create the pgcrypto extension.
	err = dbops.CreatePgCryptoExtension(db)
	if err != nil {
		log.WithError(err).Fatal("Cannot create the database extension")
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
		Description: `The tool operates in three areas:

   - Certificate Management - it allows for exporting Stork Server keys, certificates,
     and tokens that are used to secure communication between the Stork Server
     and Stork Agents;

   - Database Creation - it facilitates creating a new database for the Stork Server,
     and a user that can access this database with a generated password;

   - Database Migration - it allows for performing database schema migrations,
     overwriting the db schema version and getting its current value.`,
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
