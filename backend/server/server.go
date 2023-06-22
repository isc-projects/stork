package server

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/go-pg/pg/v10"
	flags "github.com/jessevdk/go-flags"
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
	"isc.org/stork/server/eventcenter"
	"isc.org/stork/server/hookmanager"
	"isc.org/stork/server/metrics"
	"isc.org/stork/server/restservice"
	storkutil "isc.org/stork/util"
)

// The passed command to the server by the CLI.
type Command string

// Valid commands supported by the Stork server.
const (
	// None command provided.
	NoneCommand Command = "none"
	// Run the server.
	RunCommand Command = "run"
	// Show help message.
	HelpCommand Command = "help"
	// Show version.
	VersionCommand Command = "version"
)

// Global Stork Server state.
type StorkServer struct {
	DBSettings dbops.DatabaseSettings
	DB         *dbops.PgDB

	AgentsSettings agentcomm.AgentsSettings
	Agents         agentcomm.ConnectedAgents

	RestAPISettings restservice.RestAPISettings
	RestAPI         *restservice.RestAPI

	GeneralSettings Settings

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

	HookManager *hookmanager.HookManager
}

// Read environment file settings. It's parsed before the main settings.
type EnvironmentFileSettings struct {
	EnvFile    string `long:"env-file" description:"Environment file location; applicable only if the use-env-file is provided" default:"/etc/stork/server.env"`
	UseEnvFile bool   `long:"use-env-file" description:"Read the environment variables from the environment file"`
}

// Read the hook directory settings. It's parsed after environment file
// settings but before the main settings.
// It allows us to merge the hook flags with the core flags into a single output.
type HookDirectorySettings struct {
	HookDirectory string `long:"hook-directory" description:"The path to the hook directory" env:"STORK_SERVER_HOOK_DIRECTORY" default:"/var/lib/stork-server/hooks"`
}

// Global server settings (called application settings in go-flags nomenclature).
type Settings struct {
	EnvironmentFileSettings
	HookDirectorySettings
	Version               bool  `short:"v" long:"version" description:"Show software version"`
	EnableMetricsEndpoint bool  `short:"m" long:"metrics" description:"Enable Prometheus /metrics endpoint (no auth)" env:"STORK_SERVER_ENABLE_METRICS"`
	InitialPullerInterval int64 `long:"initial-puller-interval" description:"Initial interval used by pullers fetching data from Kea; if not provided the recommended values for each puller are used" env:"STORK_SERVER_INITIAL_PULLER_INTERVAL"`
}

// Parse the command line arguments into GO structures.
// Returns the expected command to run and error.
func (ss *StorkServer) ParseArgs() (command Command, err error) {
	shortDescription := "Stork Server"
	longDescription := `Stork Server is a Kea and BIND 9 dashboard

Stork logs on INFO level by default. Other levels can be configured using the
STORK_LOG_LEVEL variable. Allowed values are: DEBUG, INFO, WARN, ERROR.`

	// Process command line flags.
	// Process the environment file flag.
	var envFileSettings EnvironmentFileSettings
	parser := flags.NewParser(&envFileSettings, flags.IgnoreUnknown)
	parser.ShortDescription = shortDescription
	parser.LongDescription = longDescription

	if _, err := parser.Parse(); err != nil {
		err = errors.Wrap(err, "invalid CLI argument")
		return NoneCommand, err
	}

	if envFileSettings.UseEnvFile {
		err = storkutil.LoadEnvironmentFileToSetter(
			envFileSettings.EnvFile,
			storkutil.NewProcessEnvironmentVariableSetter(),
		)
		if err != nil {
			err = errors.WithMessagef(err, "invalid environment file: '%s'", envFileSettings.EnvFile)
			return NoneCommand, err
		}

		// Reconfigures logging using new environment variables.
		storkutil.SetupLogging()
	}

	// Process the hook directory location.
	var hookDirectorySettings HookDirectorySettings
	parser = flags.NewParser(&hookDirectorySettings, flags.IgnoreUnknown)
	parser.ShortDescription = shortDescription
	parser.LongDescription = longDescription

	if _, err := parser.Parse(); err != nil {
		err = errors.Wrap(err, "invalid CLI argument")
		return NoneCommand, err
	}

	var allProtoSettings map[string]hooks.HookSettings
	stat, err := os.Stat(hookDirectorySettings.HookDirectory)
	switch {
	case err == nil && stat.IsDir():
		// Gather the hook flags.
		err = ss.HookManager.CollectProtoSettingsFromDirectory(hooks.HookProgramServer, hookDirectorySettings.HookDirectory)
		if err != nil {
			err = errors.WithMessage(err, "cannot collect the prototypes of the hook settings")
			return NoneCommand, err
		}
	case err == nil && !stat.IsDir():
		// Hook directory is not a directory.
		err = errors.Errorf(
			"the provided hook directory path is not pointing to a directory: %s",
			hookDirectorySettings.HookDirectory,
		)
		return NoneCommand, err
	case errors.Is(err, os.ErrNotExist):
		// Hook directory doesn't exist. Skip and continue.
		log.WithField("path", hookDirectorySettings.HookDirectory).
			WithError(err).
			Warning("the provided hook directory doesn't exist")
	default:
		// Unexpected problem.
		err = errors.Wrapf(err,
			"cannot stat the hook directory: %s",
			hookDirectorySettings.HookDirectory,
		)
		return NoneCommand, err
	}

	// Process the rest of the flags.
	parser = flags.NewParser(&ss.GeneralSettings, flags.Default)
	parser.ShortDescription = shortDescription
	parser.LongDescription = longDescription

	databaseFlags := &dbops.DatabaseCLIFlags{}
	// Process Database specific args.
	_, err = parser.AddGroup("Database ConnectionFlags", "", databaseFlags)
	if err != nil {
		return
	}

	// Process ReST API specific args.
	_, err = parser.AddGroup("HTTP ReST Server Flags", "", &ss.RestAPISettings)
	if err != nil {
		return
	}

	// Process agent comm specific args.
	_, err = parser.AddGroup("Agents Communication Flags", "", &ss.AgentsSettings)
	if err != nil {
		return
	}

	// Append hook flags.
	envVarNameReplacePattern := regexp.MustCompile("[^a-zA-Z0-9_]")
	flagNameReplacePattern := regexp.MustCompile("[^a-zA-Z0-9-]")

	for hookName, protoSettings := range allProtoSettings {
		if protoSettings == nil {
			continue
		}
		group, err := parser.AddGroup(fmt.Sprintf("Hook '%s' Flags", hookName), "", protoSettings)
		if err != nil {
			err = errors.Wrapf(err, "invalid settings for the '%s' hook", hookName)
			return NoneCommand, err
		}

		// Prepare conventional namespaces for the CLI flags and environment
		// variables.
		// Environment variables:
		//     - Starts with Stork-specific prefix
		//     - Have a component derived from the hook filename
		//     - Contains only upper cases, digits and underscore
		envNamespace := "STORK_SERVER_HOOK_" + envVarNameReplacePattern.ReplaceAllString(hookName, "_")
		envNamespace = strings.ToUpper(envNamespace)
		// CLI flags:
		//     - Have a component derived from the hook filename
		//     - Contains only lower cases, digits and dash
		flagNamespace := flagNameReplacePattern.ReplaceAllString(hookName, "-")
		flagNamespace = strings.ToLower(flagNamespace)

		group.EnvNamespace = envNamespace
		group.Namespace = flagNamespace
	}

	// Do args parsing.
	if _, err := parser.Parse(); err != nil {
		var flagsError *flags.Error
		if errors.As(err, &flagsError) {
			if flagsError.Type == flags.ErrHelp {
				return HelpCommand, nil
			}
		}
		return NoneCommand, err
	}

	dbSettings, err := databaseFlags.ConvertToDatabaseSettings()
	if err != nil {
		return NoneCommand, err
	}
	ss.DBSettings = *dbSettings

	if ss.GeneralSettings.Version {
		// If user specified --version or -v, print the version and quit.
		return VersionCommand, nil
	}

	return RunCommand, nil
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
	err = ss.HookManager.RegisterHooksFromDirectory(hooks.HookProgramServer, ss.GeneralSettings.HookDirectory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.
				WithError(err).
				Warnf("The hook directory: '%s' doesn't exist", ss.GeneralSettings.HookDirectory)
		} else {
			return err
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
		ss.MetricsCollector, err = metrics.NewCollector(ss.DB)
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

	// setup ReST API service
	r, err := restservice.NewRestAPI(&ss.RestAPISettings, &ss.DBSettings,
		ss.DB, ss.Agents, ss.EventCenter,
		ss.Pullers, ss.ReviewDispatcher, ss.MetricsCollector, ss.ConfigManager,
		ss.DHCPOptionDefinitionLookup, ss.HookManager)
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
		ss.RestAPI.Shutdown()
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
