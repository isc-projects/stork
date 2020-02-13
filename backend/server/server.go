package server

import (
	"os"

	flags "github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps/kea"
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

	StatsPuller *kea.StatsPuller
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

	// setup connected agents
	ss.Agents = agentcomm.NewConnectedAgents(&ss.AgentsSettings)

	// setup database connection
	var err error
	ss.Db, err = dbops.NewPgDB(&ss.DbSettings)
	if err != nil {
		ss.Agents.Shutdown()
		return nil, err
	}

	// setup kea stats puller
	ss.StatsPuller = kea.NewStatsPuller(ss.Db, ss.Agents)

	// setup ReST API service
	r, err := restservice.NewRestAPI(&ss.RestAPISettings, &ss.DbSettings, ss.Db, ss.Agents)
	if err != nil {
		ss.StatsPuller.Shutdown()
		ss.Db.Close()
		ss.Agents.Shutdown()
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
	log.Println("Shutting down Stork Server")
	ss.RestAPI.Shutdown()
	ss.StatsPuller.Shutdown()
	ss.Db.Close()
	ss.Agents.Shutdown()
	log.Println("Stork Server shut down")
}
