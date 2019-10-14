package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"isc.org/stork/server/database"
	"os"
)

// Usage text displayed when invalid command line arguments
// were provided or when help was requested.
const usageText = `usage: stork-db-migrate options action

 options:
  -d <database name>
  -u <database user name>

 actions:
   init         creates versioning table in the database
   up           runs all available mirations
   up <target>  runs migrations up to the target version
   down         reverts last migration
   reset        reverts all migrations
   version      prints current version
   set_version  sets database version without running migrations

`

func main() {

	// Replace standard usage text which would lack the descriptions
	// of migration actions.
	flag.Usage = usage

	// Bind variables to command line parameters and parse the command
	// line.
	var database_name string
	var user_name string
	flag.StringVar(&database_name, "d", "", "database name")
	flag.StringVar(&user_name, "u", "", "database user name")
	flag.Parse()

	// Specifying default values for the database name and the user
	// name doesn't make sense. Therefore, we have to check if the
	// user explicitly specified them.

	// Database and user name are required and we have to check if
	// they were specified.
	requiredFlags := []string{"d", "u"}
	seenFlags := make(map[string]bool)

	// Visit walks over the specified flags. For each such flag
	// we mark it as "seen".
	flag.Visit(func(f *flag.Flag) { seenFlags[f.Name] = true })
	for _, req := range requiredFlags {
		if !seenFlags[req] {
			exitf(usage, "The -%s option is mandatory!", req)
		}
	}

	// Prompt the user for database password.
	fmt.Printf("database password: ")
	password, err := terminal.ReadPassword(0)
	fmt.Printf("\n")

	if err != nil {
		exitf(nil, err.Error())
	}

	// Use the provided credentials to connect to the database.
	oldVersion, newVersion, err := storkdb.Migrate(&storkdb.DbConnOptions{
		User:     user_name,
		Password: string(password),
		Database: database_name,
	})

	if err != nil {
		exitf(nil, err.Error())
	}

	if newVersion != oldVersion {
		fmt.Printf("Migrated database from version %d to %d\n", oldVersion, newVersion)
	} else {
		fmt.Printf("Database version is %d\n", oldVersion)
	}
}

// Prints usage text for the migrations tool.
func usage() {
	fmt.Print(usageText)
}

// Prints error string to stderr.
func errorf(s string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", args...)
}

// Prints error string to stderr and exists with exit code 1.
// If the usagefn is not nil it is invoked to present program
// usage information.
func exitf(usagefn func(), s string, args ...interface{}) {
	errorf(s, args...)
	if usagefn != nil {
		usagefn()
	}
	os.Exit(1)
}
