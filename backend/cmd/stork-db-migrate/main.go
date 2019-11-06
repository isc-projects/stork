package main

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/server/database"
	"isc.org/stork/server/database/migrations"
	"os"
)

// Structure defining options for all commands except "up".
type cmdOpts struct {
}

// Structure defining options for "up" command.
type upOpts struct {
	Target string `short:"t" long:"target" description:"Target database schema version"`
}

// Common application options.
type Opts struct{
	DatabaseName string `short:"d" long:"database" description:"database name" env:"STORK_DATABASE_NAME" default:"stork"`
	UserName string `short:"u" long:"user" description:"database user name" env:"STORK_DATABASE_USER_NAME" default:"stork"`
	Host string `long:"host" description:"database user name" env:"STORK_DATABASE_HOST" default:"localhost"`
	Port int `short:"p" long:"port" description:"database user name" env:"STORK_DATABASE_PORT" default:"5432"`
	Init cmdOpts `command:"init" description:"Create schema versioning table in the database"`
	Up upOpts `command:"up" description:"Run all available migrations or up to a selected version"`
	Down cmdOpts `command:"down" description:"Revert last migration"`
	Reset cmdOpts `command:"reset" description:"Revert all migrations"`
	Version cmdOpts `command:"version" description:"Print current migration version"`
	SetVersion cmdOpts `command:"set_version" description:"Set database version without running migrations"`
}

func main() {
	// Parse command line options and commands.
	opts := Opts{}
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		// Printing help is not an error.
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	// Password from the environment variable takes precedence.
	password := os.Getenv("STORK_DATABASE_PASSWORD")
	if len(password) == 0 {
		// Prompt the user for database password.
		fmt.Printf("database password: ")
		pass, err := terminal.ReadPassword(0)
		fmt.Printf("\n")

		if err != nil {
			log.Fatal(err.Error())
		}

		password = string(pass)
	}

	// The up command requires special treatment. If the target version is specified
	// it must be appended to the arguments we pass to the go-pg migrations.
	var args []string
	args = append(args, parser.Active.Name)
	if parser.Active.Name == "up" && len(opts.Up.Target) > 0 {
		args = append(args, opts.Up.Target)
	}

	// Use the provided credentials to connect to the database.
	db := dbops.NewPgDB(&dbops.PgOptions{
		User:     opts.UserName,
		Password: string(password),
		Database: opts.DatabaseName,
		Addr:     fmt.Sprintf("%s:%d", opts.Host, opts.Port),
	})
	defer db.Close()

	// Theoretically, it should not happen but let's make sure in case someone
	// modifies the NewPgDB function.
	if db == nil {
		log.Fatal("unable to create database instance")
	}

	oldVersion, newVersion, err := dbmigs.Migrate(db, args...)
	if err != nil {
		log.Fatal(err.Error())
	}

	if newVersion != oldVersion {
		log.Infof("Migrated database from version %d to %d\n", oldVersion, newVersion)

	} else {
		availVersion := dbmigs.AvailableVersion()
		if availVersion == oldVersion {
			log.Infof("Database version is %d (up to date)\n", oldVersion)
		} else {
			log.Infof("Database version is %d (new version %d available)\n", oldVersion, availVersion)
		}
	}
}
