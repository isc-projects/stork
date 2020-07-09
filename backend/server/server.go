package server

import (
	"fmt"
	"os"

	flags "github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps"
	"isc.org/stork/server/apps/bind9"
	"isc.org/stork/server/apps/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
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

	AppsStatePuller  *apps.StatePuller
	Bind9StatsPuller *bind9.StatsPuller
	KeaStatsPuller   *kea.StatsPuller
	KeaHostsPuller   *kea.HostsPuller
	HAStatusPuller   *kea.HAStatusPuller

	EventCenter eventcenter.EventCenter
}

// Global server settings (called application settings in go-flags nomenclature).
type Settings struct {
	Version bool `short:"v" long:"version" description:"show software version"`
}

func (ss *StorkServer) ParseArgs() {
	// Process command line flags.
	var serverSettings Settings
	parser := flags.NewParser(&serverSettings, flags.Default)
	parser.ShortDescription = "Stork Server"
	parser.LongDescription = "Stork Server is a Kea and BIND 9 Dashboard"

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

	if serverSettings.Version {
		// If user specified --version or -v, print the version and quit.
		fmt.Printf("%s\n", stork.Version)
		os.Exit(0)
	}
}

// Init for Stork Server state
func NewStorkServer() (ss *StorkServer, err error) {
	ss = &StorkServer{}
	ss.ParseArgs()

	// setup database connection
	ss.Db, err = dbops.NewPgDB(&ss.DbSettings)
	if err != nil {
		return nil, err
	}

	// initialize stork settings
	err = dbmodel.InitializeSettings(ss.Db)
	if err != nil {
		return nil, err
	}

	// setup event center
	ss.EventCenter = eventcenter.NewEventCenter(ss.Db)

	// setup connected agents
	ss.Agents = agentcomm.NewConnectedAgents(&ss.AgentsSettings, ss.EventCenter)
	// TODO: if any operation below fails then this Shutdown here causes segfault.
	// I do not know why and do not how to fix this. Commenting out for now.
	// defer func() {
	// 	if err != nil {
	// 		ss.Agents.Shutdown()
	// 	}
	// }()

	// initialize stork statistics
	err = dbmodel.InitializeStats(ss.Db)
	if err != nil {
		return nil, err
	}

	// setup apps state puller
	ss.AppsStatePuller, err = apps.NewStatePuller(ss.Db, ss.Agents, ss.EventCenter)
	if err != nil {
		return nil, err
	}

	// setup bind9 stats puller
	ss.Bind9StatsPuller, err = bind9.NewStatsPuller(ss.Db, ss.Agents, ss.EventCenter)
	if err != nil {
		return nil, err
	}

	// setup kea stats puller
	ss.KeaStatsPuller, err = kea.NewStatsPuller(ss.Db, ss.Agents)
	if err != nil {
		return nil, err
	}

	// Setup Kea hosts puller.
	ss.KeaHostsPuller, err = kea.NewHostsPuller(ss.Db, ss.Agents)
	if err != nil {
		return nil, err
	}

	// Setup Kea HA status puller.
	ss.HAStatusPuller, err = kea.NewHAStatusPuller(ss.Db, ss.Agents)
	if err != nil {
		return nil, err
	}

	// setup ReST API service
	r, err := restservice.NewRestAPI(&ss.RestAPISettings, &ss.DbSettings, ss.Db, ss.Agents, ss.EventCenter)
	if err != nil {
		ss.HAStatusPuller.Shutdown()
		ss.KeaHostsPuller.Shutdown()
		ss.KeaStatsPuller.Shutdown()
		ss.Bind9StatsPuller.Shutdown()
		ss.AppsStatePuller.Shutdown()
		ss.Db.Close()
		return nil, err
	}
	ss.RestAPI = r

	ss.EventCenter.AddInfoEvent("started Stork server", "version: "+stork.Version+"\nbuild date: "+stork.BuildDate)

	return ss, nil
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
	ss.EventCenter.AddInfoEvent("shutting down Stork server")
	log.Println("Shutting down Stork Server")
	ss.RestAPI.Shutdown()
	ss.HAStatusPuller.Shutdown()
	ss.KeaHostsPuller.Shutdown()
	ss.KeaStatsPuller.Shutdown()
	ss.Bind9StatsPuller.Shutdown()
	ss.AppsStatePuller.Shutdown()
	ss.Agents.Shutdown()
	ss.EventCenter.Shutdown()
	ss.Db.Close()
	log.Println("Stork Server shut down")
}
