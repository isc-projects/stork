package server

import (
	"os"
	"sync"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/hooks"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps"
	"isc.org/stork/server/apps/bind9"
	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/certs"
	"isc.org/stork/server/config"
	"isc.org/stork/server/configreview"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dnsop"
	"isc.org/stork/server/eventcenter"
	"isc.org/stork/server/hookmanager"
	"isc.org/stork/server/metrics"
	"isc.org/stork/server/restservice"
)

// Global Stork Server state.
type StorkServer struct {
	DBSettings dbops.DatabaseSettings
	DB         *dbops.PgDB

	AgentsSettings agentcomm.AgentsSettings
	Agents         agentcomm.ConnectedAgents

	RestAPISettings restservice.RestAPISettings
	RestAPI         *restservice.RestAPI

	GeneralSettings GeneralSettings

	Pullers *apps.Pullers

	MetricsCollector metrics.Collector

	EventCenter eventcenter.EventCenter

	ReviewDispatcher configreview.Dispatcher
	// Configuration manager instance. Note that it inherits some fields
	// maintained by the server.
	ConfigManager config.Manager
	// Provides lookup functionality for DHCP option definitions.
	DHCPOptionDefinitionLookup keaconfig.DHCPOptionDefinitionLookup
	shutdownOnce               sync.Once

	HookManager   *hookmanager.HookManager
	hooksSettings map[string]hooks.HookSettings
}

// Parse the command line arguments into GO structures.
// Returns the expected command to run and error.
func (ss *StorkServer) ParseArgs() (command Command, err error) {
	parser := NewCLIParser()
	command, settings, err := parser.Parse()
	if command == RunCommand {
		ss.hooksSettings = settings.HooksSettings
		ss.AgentsSettings = *settings.AgentsSettings
		ss.DBSettings = *settings.DatabaseSettings
		ss.GeneralSettings = *settings.GeneralSettings
		ss.RestAPISettings = *settings.RestAPISettings
	}
	return command, err
}

// Init for Stork Server state from the CLI arguments and environment variables.
// Returns the server object (if no error occurs), command to execute and error.
func NewStorkServer() (ss *StorkServer, command Command, err error) {
	ss = &StorkServer{
		HookManager: hookmanager.NewHookManager(),
	}
	command, err = ss.ParseArgs()
	if err != nil {
		return nil, command, err
	}

	return ss, command, nil
}

// Establishes a database connection, runs the background pullers, and
// prepares the REST API. The reload flag indicates if the server is
// starting up (reload=false) or it is being reloaded (reload=true).
func (ss *StorkServer) Bootstrap(reload bool) (err error) {
	err = ss.HookManager.RegisterHooksFromDirectory(
		hooks.HookProgramServer,
		ss.GeneralSettings.HookDirectory,
		ss.hooksSettings,
	)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.
				WithError(err).
				Warnf("The hook directory: '%s' doesn't exist", ss.GeneralSettings.HookDirectory)
		} else {
			return errors.WithMessagef(
				err,
				"failed to load hooks from directory: '%s'",
				ss.GeneralSettings.HookDirectory,
			)
		}
	}

	// setup database connection
	ss.DB, err = dbops.NewApplicationDatabaseConn(&ss.DBSettings)
	if err != nil {
		return err
	}

	// initialize stork settings
	err = dbmodel.InitializeSettings(ss.DB, ss.GeneralSettings.InitialPullerInterval)
	if err != nil {
		return err
	}

	// prepare certificates for establish secure connections
	// between server and agents
	caCertPEM, serverCertPEM, serverKeyPEM, err := certs.SetupServerCerts(ss.DB)
	if err != nil {
		return err
	}

	// setup event center
	ss.EventCenter = eventcenter.NewEventCenter(ss.DB)

	// setup connected agents
	ss.Agents = agentcomm.NewConnectedAgents(&ss.AgentsSettings, ss.EventCenter, caCertPEM, serverCertPEM, serverKeyPEM)
	// TODO: if any operation below fails then this Shutdown here causes segfault.
	// I do not know why and do not know how to fix this. Commenting out for now.
	// defer func() {
	// 	if err != nil {
	// 		ss.Agents.Shutdown()
	// 	}
	// }()

	// Setup configuration review dispatcher.
	ss.ReviewDispatcher = configreview.NewDispatcher(ss.DB)
	configreview.RegisterDefaultCheckers(ss.ReviewDispatcher)
	err = configreview.LoadAndValidateCheckerPreferences(ss.DB, ss.ReviewDispatcher)
	if err != nil {
		return err
	}
	ss.ReviewDispatcher.Start()

	// initialize stork statistics
	err = dbmodel.InitializeStats(ss.DB)
	if err != nil {
		return err
	}

	ss.Pullers = &apps.Pullers{}

	// This instance provides functions to search for option definitions, both in the
	// database and among the standard options. It is required by the config manager.
	ss.DHCPOptionDefinitionLookup = dbmodel.NewDHCPOptionDefinitionLookup()

	// setup apps state puller
	ss.Pullers.AppsStatePuller, err = apps.NewStatePuller(ss.DB, ss.Agents, ss.EventCenter, ss.ReviewDispatcher, ss.DHCPOptionDefinitionLookup)
	if err != nil {
		return err
	}

	// setup bind9 stats puller
	ss.Pullers.Bind9StatsPuller, err = bind9.NewStatsPuller(ss.DB, ss.Agents, ss.EventCenter)
	if err != nil {
		return err
	}

	// setup kea stats puller
	ss.Pullers.KeaStatsPuller, err = kea.NewStatsPuller(ss.DB, ss.Agents)
	if err != nil {
		return err
	}

	// Setup Kea hosts puller.
	ss.Pullers.KeaHostsPuller, err = kea.NewHostsPuller(ss.DB, ss.Agents, ss.ReviewDispatcher, ss.DHCPOptionDefinitionLookup)
	if err != nil {
		return err
	}

	// Setup Kea HA status puller.
	ss.Pullers.HAStatusPuller, err = kea.NewHAStatusPuller(ss.DB, ss.Agents)
	if err != nil {
		return err
	}

	if ss.GeneralSettings.EnableMetricsEndpoint {
		ss.MetricsCollector, err = metrics.NewCollector(
			metrics.NewDatabaseMetricsSource(ss.DB),
		)
		if err != nil {
			return err
		}
		log.Info("The metrics endpoint is enabled (ensure that it is properly secured)")
	} else {
		log.Warn("The metric endpoint is disabled (it can be enabled with the -m flag)")
	}

	// Create the config manager instance. It takes config.ManagerAccessors interface
	// as a parameter. The manager uses this interface to setup its state. For example,
	// it stores the instance of the DHCP option definition lookup. Note, that it is
	// important to maintain one instance of the lookup because it applies indexing
	// on the option definitions it returns. Indexing should be done only once at
	// server startup.
	ss.ConfigManager = apps.NewManager(ss)

	// Check if the machine registration endpoint should be disabled.
	enableMachineRegistration, err := dbmodel.GetSettingBool(ss.DB, "enable_machine_registration")
	if err != nil {
		return err
	}

	// Endpoint control holds the list of explicitly disabled REST API endpoints.
	endpointControl := restservice.NewEndpointControl()
	endpointControl.SetEnabled(restservice.EndpointOpCreateNewMachine, enableMachineRegistration)

	// Create DNS Manager.
	dnsManager := dnsop.NewManager(ss)

	// setup ReST API service
	r, err := restservice.NewRestAPI(&ss.RestAPISettings, &ss.DBSettings,
		ss.DB, ss.Agents, ss.EventCenter,
		ss.Pullers, ss.ReviewDispatcher, ss.MetricsCollector, ss.ConfigManager,
		ss.DHCPOptionDefinitionLookup, ss.HookManager, endpointControl,
		dnsManager)
	if err != nil {
		ss.Pullers.HAStatusPuller.Shutdown()
		ss.Pullers.KeaHostsPuller.Shutdown()
		ss.Pullers.KeaStatsPuller.Shutdown()
		ss.Pullers.Bind9StatsPuller.Shutdown()
		ss.Pullers.AppsStatePuller.Shutdown()
		if ss.MetricsCollector != nil {
			ss.MetricsCollector.Shutdown()
		}

		ss.HookManager.Close()
		ss.DB.Close()

		return err
	}
	ss.RestAPI = r

	if reload {
		ss.EventCenter.AddInfoEvent("reloaded Stork Server", "version: "+stork.Version+"\nbuild date: "+stork.BuildDate)
	} else {
		ss.EventCenter.AddInfoEvent("started Stork Server", "version: "+stork.Version+"\nbuild date: "+stork.BuildDate)
	}

	return nil
}

// Run Stork Server.
func (ss *StorkServer) Serve() {
	// Start listening for requests from ReST API.
	err := ss.RestAPI.Serve()
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}
}

// Shutdown for Stork Server state. The reload flag indicates if the
// Shutdown is called as part of the server reload (reload=true) or
// the process is terminating (reload=false).
func (ss *StorkServer) Shutdown(reload bool) {
	// If this function is called multiple times, make sure we shutdown
	// only once.
	ss.shutdownOnce.Do(func() {
		// Do not issue the event when we're reloading instead of terminating
		// the server process.
		if !reload {
			ss.EventCenter.AddInfoEvent("shutting down Stork Server")
			log.Println("Shutting down Stork Server")
		}
		ss.Pullers.HAStatusPuller.Shutdown()
		ss.Pullers.KeaHostsPuller.Shutdown()
		ss.Pullers.KeaStatsPuller.Shutdown()
		ss.Pullers.Bind9StatsPuller.Shutdown()
		ss.Pullers.AppsStatePuller.Shutdown()
		ss.Agents.Shutdown()
		ss.EventCenter.Shutdown()
		ss.ReviewDispatcher.Shutdown()
		if ss.MetricsCollector != nil {
			ss.MetricsCollector.Shutdown()
		}
		ss.HookManager.Close()
		ss.RestAPI.Shutdown()
		ss.DB.Close()

		if !reload {
			log.Println("Stork Server shut down")
		}
	})
}

// Returns an instance of the database handler used by the configuration manager.
func (ss *StorkServer) GetDB() *pg.DB {
	return ss.DB
}

// Returns an interface to the agents the manager communicates with.
func (ss *StorkServer) GetConnectedAgents() agentcomm.ConnectedAgents {
	return ss.Agents
}

// Returns an interface to the instance providing the DHCP option definition
// lookup logic.
func (ss *StorkServer) GetDHCPOptionDefinitionLookup() keaconfig.DHCPOptionDefinitionLookup {
	return ss.DHCPOptionDefinitionLookup
}
