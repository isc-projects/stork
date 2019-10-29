package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	flags "github.com/jessevdk/go-flags"

	"isc.org/stork/server"
	"isc.org/stork/server/database"
)

func main() {
	// Initialize global state of Stork Server
	storkServer := server.NewStorkServer()
	defer storkServer.Shutdown()

	// Setup logging
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Process command line flags.
	parser := flags.NewParser(nil, flags.Default) // TODO: change nil to some main group of server settings
	parser.ShortDescription = "Stork Server"
	parser.LongDescription = "Stork Server is a Kea and BIND Dashboard"

	// Process Database specific args.
	_, err := parser.AddGroup("Database ConnectionFlags", "", &storkServer.Database)
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	// Process ReST API specific args.
	_, err = parser.AddGroup("ReST Server Flags", "", &storkServer.RestAPI.Settings)
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	// Process agent comm specific args.
	_, err = parser.AddGroup("Agents Communication Flags", "", storkServer.Agents.GetSettings())
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	// Do args parsing.
	if _, err := parser.Parse(); err != nil {
		code := 1
		if fe, ok := err.(*flags.Error); ok {
			if fe.Type == flags.ErrHelp {
				code = 0
			}
		}
		os.Exit(code)
	}

	// Fetch password from the env variable or prompt for password.
	dbops.Password(&storkServer.Database)

	storkServer.Serve();
}
