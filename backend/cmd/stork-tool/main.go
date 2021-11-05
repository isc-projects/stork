package main

import (
	"fmt"
	"net"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"isc.org/stork"
	"isc.org/stork/server/certs"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Establish connection to a database with opts from command line.
func getDBConn(settings *cli.Context) *dbops.PgDB {
	var opts *dbops.PgOptions
	var err error

	dbURL := settings.String("db-url")
	if dbURL != "" {
		opts, err = dbops.ParseURL(dbURL)
		if err != nil {
			log.Fatal("cannot parse db URL")
		}
		opts.TLSConfig = nil // ParseURL sets it automatically but we do not use TLS so reset it
	} else {
		var passwd string
		if settings.IsSet("db-password") {
			passwd = settings.String("db-password")
		} else {
			// if password is missing then prompt for it
			passwd = storkutil.GetSecretInTerminal("database password: ")
		}

		addrPort := net.JoinHostPort(settings.String("db-host"), settings.String("db-port"))

		// TLS configuration.
		tlsConfig, err := dbops.GetTLSConfig(settings.String("db-sslmode"),
			settings.String("db-host"),
			settings.String("db-sslcert"),
			settings.String("db-sslkey"),
			settings.String("db-sslrootcert"))
		if err != nil {
			log.Fatal(err.Error())
		}

		// Use the provided credentials to connect to the database.
		opts = &dbops.PgOptions{
			User:      settings.String("db-user"),
			Password:  passwd,
			Database:  settings.String("db-name"),
			Addr:      addrPort,
			TLSConfig: tlsConfig,
		}
	}

	db, err := dbops.NewPgDBConn(opts, settings.String("db-trace-queries") != "")
	if err != nil {
		log.Fatalf("unexpected error: %+v", err)
	}

	// Theoretically, it should not happen but let's make sure in case someone
	// modifies the NewPgDB function.
	if db == nil {
		log.Fatal("unable to create database instance")
	}
	return db
}

// Execute DB migration command.
func runDBMigrate(settings *cli.Context, command, version string) {
	// The up and down commands require special treatment. If the target version is specified
	// it must be appended to the arguments we pass to the go-pg migrations.
	var args []string
	args = append(args, command)
	if command == "up" && len(version) > 0 {
		args = append(args, version)
		log.Infof("Requested migrating up to version %s", version)
	}
	if command == "down" && len(version) > 0 {
		args = append(args, version)
		log.Infof("Requested migrating down to version %s", version)
	}
	if command == "set_version" {
		if version == "" {
			log.Fatal("flag --version/-t is missing but required")
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
	db.Close()
	if err != nil {
		log.Fatalf(err.Error())
	}

	if newVersion != oldVersion {
		log.Infof("Migrated database from version %d to %d\n", oldVersion, newVersion)
	} else {
		availVersion := dbops.AvailableVersion()
		if availVersion == oldVersion {
			log.Infof("Database version is %d (up to date)\n", oldVersion)
		} else {
			log.Infof("Database version is %d (new version %d available)\n", oldVersion, availVersion)
		}
	}
}

// Execute cert export command.
func runCertExport(settings *cli.Context) error {
	db := getDBConn(settings)

	return certs.ExportSecret(db, settings.String("object"), settings.String("file"))
}

// Execute cert import command.
func runCertImport(settings *cli.Context) error {
	db := getDBConn(settings)

	return certs.ImportSecret(db, settings.String("object"), settings.String("file"))
}

// Prepare urfave cli app with all flags and commands defined.
func setupApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Version)
	}

	dbFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "db-url",
			Usage:   "The URL to locate Stork PostgreSQL database.",
			EnvVars: []string{"STORK_DATABASE_URL"},
		},
		&cli.StringFlag{
			Name:    "db-user",
			Usage:   "The user name to be used for database connections.",
			Aliases: []string{"u"},
			Value:   "stork",
			EnvVars: []string{"STORK_DATABASE_USER_NAME"},
		},
		&cli.StringFlag{
			Name:    "db-password",
			Usage:   "The database password to be used for database connections.",
			EnvVars: []string{"STORK_DATABASE_PASSWORD"},
		},
		&cli.StringFlag{
			Name:    "db-host",
			Usage:   "The name of the host where the database is available.",
			Value:   "localhost",
			EnvVars: []string{"STORK_DATABASE_HOST"},
		},
		&cli.StringFlag{
			Name:    "db-port",
			Usage:   "The port on which the database is available.",
			Aliases: []string{"p"},
			Value:   "5432",
			EnvVars: []string{"STORK_DATABASE_PORT"},
		},
		&cli.StringFlag{
			Name:    "db-name",
			Usage:   "The name of the database to connect to.",
			Aliases: []string{"d"},
			Value:   "stork",
			EnvVars: []string{"STORK_DATABASE_NAME"},
		},
		&cli.StringFlag{
			Name:    "db-sslmode",
			Usage:   "The SSL mode for connecing to the database (i.e., disable, require, verify-ca or verify-full).",
			Value:   "disable",
			EnvVars: []string{"STORK_DATABASE_SSLMODE"},
		},
		&cli.StringFlag{
			Name:    "db-sslcert",
			Usage:   "The location of the SSL certificate used by the server to connect to the database.",
			EnvVars: []string{"STORK_DATABASE_SSLCERT"},
		},
		&cli.StringFlag{
			Name:    "db-sslkey",
			Usage:   "The location of the SSL key used by the server to connect to the database.",
			EnvVars: []string{"STORK_DATABASE_SSLKEY"},
		},
		&cli.StringFlag{
			Name:    "db-sslrootcert",
			Usage:   "The location of the root certificate file used to verify the database server's certificate.",
			EnvVars: []string{"STORK_DATABASE_SSLROOTCERT"},
		},
		&cli.StringFlag{
			Name:    "db-trace-queries",
			Usage:   "Enable tracing SQL queries: \"run\" - only run-time, without migrations, \"all\" - migrations and run-time.",
			Value:   "",
			EnvVars: []string{"STORK_DATABASE_TRACE_QUERIES"},
		},
	}

	dbVerFlags := append(dbFlags,
		&cli.StringFlag{
			Name:    "version",
			Usage:   "Target database schema version (optional).",
			Aliases: []string{"t"},
			EnvVars: []string{"STORK_TOOL_DB_VERSION"},
		})

	certExportFlags := append(dbFlags,
		&cli.StringFlag{
			Name:     "object",
			Usage:    "The object to dump, it can be one of 'cakey', 'cacert', 'srvkey', 'srvcert', 'srvtkn'.",
			Required: true,
			Aliases:  []string{"f"},
			EnvVars:  []string{"STORK_TOOL_CERT_OBJECT"},
		},
		&cli.StringFlag{
			Name:    "file",
			Usage:   "The file location where the object should be saved. If not provided then object is printed to stdout.",
			Aliases: []string{"o"},
			EnvVars: []string{"STORK_TOOL_CERT_FILE"},
		})

	certImportFlags := append(dbFlags,
		&cli.StringFlag{
			Name:     "object",
			Usage:    "The object to dump, it can be one of 'cakey', 'cacert', 'srvkey', 'srvcert', 'srvtkn'.",
			Required: true,
			Aliases:  []string{"f"},
			EnvVars:  []string{"STORK_TOOL_CERT_OBJECT"},
		},
		&cli.StringFlag{
			Name:    "file",
			Usage:   "The file location from which the object will be read. If not provided then the object is read from stdin.",
			Aliases: []string{"i"},
			EnvVars: []string{"STORK_TOOL_CERT_FILE"},
		})

	app := &cli.App{
		Name: "Stork Tool",
		Usage: `A tool for managing Stork Server

   The tool operates in two areas:

   - Certificates Management - it allows for exporting Stork Server keys, certificates
     and token that are used for securing communication between Stork Server
     and Stork Agents

   - Database Migration - it allows for performing database schema migrations,
     overwriting db schema version and getting its current value`,
		Version:  stork.Version,
		HelpName: "stork-tool",
		Commands: []*cli.Command{
			// DATABASE MIGRATION COMMANDS
			{
				Name:        "db-init",
				Usage:       "Create schema versioning table in the database",
				UsageText:   "stork-tool db-init [options for db connecion]",
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
				UsageText:   "stork-tool db-up [options for db connecion] [-t version]",
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
				UsageText:   "stork-tool db-down [options for db connecion] [-t version]",
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
				UsageText:   "stork-tool db-reset [options for db connecion]",
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
				UsageText:   "stork-tool db-version [options for db connecion]",
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
				UsageText:   "stork-tool db-set-version [options for db connecion] [-t version]",
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
				UsageText:   "stork-tool cert-export [options for db connecion] [-f object] [-o filename]",
				Description: ``,
				Flags:       certExportFlags,
				Category:    "Certificates Management",
				Action:      runCertExport,
			},
			{
				Name:        "cert-import",
				Usage:       "Import certificate or other secret data",
				UsageText:   "stork-tool cert-import [options for db connecion] [-f object] [-i filename]",
				Description: ``,
				Flags:       certImportFlags,
				Category:    "Certificates Management",
				Action:      runCertImport,
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
