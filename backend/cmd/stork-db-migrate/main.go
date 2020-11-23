package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"isc.org/stork"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Structure defining options for all commands except "up" and "down".
type cmdOpts struct {
}

// Structure defining options for "up" and "down" commands.
type verOpts struct {
	Target string `short:"t" long:"target" description:"Target database schema version"`
}

// Common application options.
type Opts struct {
	dbops.DatabaseSettings
	Init       cmdOpts `command:"init" description:"Create schema versioning table in the database"`
	Up         verOpts `command:"up" description:"Run all available migrations or use -t to specify version"`
	Down       verOpts `command:"down" description:"Revert last migration or use -t to specify version to downgrade to"`
	Reset      cmdOpts `command:"reset" description:"Revert all migrations"`
	Version    cmdOpts `command:"version" description:"Print current migration version"`
	SetVersion cmdOpts `command:"set_version" description:"Set database version without running migrations"`
}

func main() {
	// Setup logging
	storkutil.SetupLogging()

	log.Printf("Starting Stork Database Migration Tool, version %s, build date %s", stork.Version, stork.BuildDate)

	// Parse command line options and commands.
	opts := Opts{}
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		// Printing help is not an error.
		var flagsError *flags.Error
		if errors.As(err, &flagsError) && flagsError.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			// We should print out what exactly went wrong.
			log.Fatalf("Error during parsing options: %+v", err)
			os.Exit(1)
		}
	}

	// Password from the environment variable takes precedence.
	dbops.Password(&opts.DatabaseSettings)

	// The up and down commands require special treatment. If the target version is specified
	// it must be appended to the arguments we pass to the go-pg migrations.
	var args []string
	args = append(args, parser.Active.Name)
	if parser.Active.Name == "up" && len(opts.Up.Target) > 0 {
		args = append(args, opts.Up.Target)
		log.Infof("Requested migrating up to version %s", opts.Up.Target)
	}
	if parser.Active.Name == "down" && len(opts.Down.Target) > 0 {
		args = append(args, opts.Down.Target)
		log.Infof("Requested migrating down to version %s", opts.Down.Target)
	}

	if opts.DatabaseSettings.TraceSQL != "" {
		log.Infof("SQL queries tracing set to %s", opts.DatabaseSettings.TraceSQL)
	}

	// Use the provided credentials to connect to the database.
	db, err := dbops.NewPgDbConn(&dbops.PgOptions{
		User:     opts.User,
		Password: opts.Password,
		Database: opts.DbName,
		Addr:     fmt.Sprintf("%s:%d", opts.Host, opts.Port),
	}, opts.DatabaseSettings.TraceSQL != "")
	if err != nil {
		log.Fatalf("unexpected error: %+v", err)
	}
	// Theoretically, it should not happen but let's make sure in case someone
	// modifies the NewPgDB function.
	if db == nil {
		log.Fatal("unable to create database instance")
	}

	oldVersion, newVersion, err := dbops.Migrate(db, args...)
	if err != nil {
		db.Close()
		log.Fatalf(err.Error())
	}

	defer db.Close()

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
