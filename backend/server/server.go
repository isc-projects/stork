package server

import (
	"os"

	flags "github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/restservice"
)

// Global Stork Server state
type StorkServer struct {
	DbSettings dbops.DatabaseSettings
	Db         *dbops.PgDB

	AgentsSettings agentcomm.AgentsSettings
	Agents         agentcomm.ConnectedAgents

	RestAPISettings restservice.RestAPISettings
	RestAPI         *restservice.RestAPI
}

func (ss *StorkServer) ParseArgs() {
	// Process command line flags.
	parser := flags.NewParser(nil, flags.Default) // TODO: change nil to some main group of server settings
	parser.ShortDescription = "Stork Server"
	parser.LongDescription = "Stork Server is a Kea and BIND Dashboard"

	// Process Database specific args.
	_, err := parser.AddGroup("Database ConnectionFlags", "", &ss.DbSettings)
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	// Process ReST API specific args.
	_, err = parser.AddGroup("ReST Server Flags", "", &ss.RestAPISettings)
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	// Process agent comm specific args.
	_, err = parser.AddGroup("Agents Communication Flags", "", &ss.AgentsSettings)
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
}

// Init for Stork Server state
func NewStorkServer() (*StorkServer, error) {
	ss := StorkServer{}
	ss.ParseArgs()

	ss.Agents = agentcomm.NewConnectedAgents(&ss.AgentsSettings)

	var err error
	ss.Db, err = dbops.NewPgDB(&ss.DbSettings)
	if err != nil {
		ss.Agents.Shutdown()
		return nil, err
	}

	r, err := restservice.NewRestAPI(&ss.RestAPISettings, &ss.DbSettings, ss.Db, ss.Agents)
	if err != nil {
		ss.Agents.Shutdown()
		ss.Db.Close()
		return nil, err
	}
	ss.RestAPI = r
	return &ss, nil
}

// Run Stork Server
func (ss *StorkServer) Serve() {
	// Start listening for requests from ReST API.
	err := ss.RestAPI.Serve()
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}
}

// Shutdown for Stork Server state
func (ss *StorkServer) Shutdown() {
	ss.RestAPI.Shutdown()
	ss.Db.Close()
	ss.Agents.Shutdown()
}
