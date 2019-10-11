package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-pg/migrations/v7"
	"github.com/go-pg/pg/v9"

	"golang.org/x/crypto/ssh/terminal"
)

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

	flag.Usage = usage

	var database_name string
	var user_name string

	flag.StringVar(&database_name, "d", "", "database name")
	flag.StringVar(&user_name, "u", "", "database user name")

	required := []string{"d", "u"}

	flag.Parse()

	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			usage()
			os.Exit(2)
		}
	}

	fmt.Printf("password: ")
	password, _ := terminal.ReadPassword(0)
	fmt.Printf("\n")

	db := pg.Connect(&pg.Options{
		User:     user_name,
		Password: string(password),
		Database: database_name,
	})

	oldVersion, newVersion, err := migrations.Run(db, flag.Args()...)
	if err != nil {
		exitf(err.Error())
	}
	if newVersion != oldVersion {
		fmt.Printf("migrated from version %d to %d\n", oldVersion, newVersion)
	} else {
		fmt.Printf("version is %d\n", oldVersion)
	}
}

func usage() {
	fmt.Print(usageText)
//	os.Exit(2)
}

func errorf(s string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", args...)
}

func exitf(s string, args ...interface{}) {
	errorf(s, args...)
	os.Exit(1)
}
