package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"isc.org/stork"
	"isc.org/stork/agent"
	"isc.org/stork/hooks"
	storkutil "isc.org/stork/util"
)

// Sighup error is used to indicate that Stork Agent  received a
// SIGHUP signal.
type sighupError struct{}

// Returns sighupError error text.
func (e *sighupError) Error() string {
	return "received SIGHUP signal"
}

// Error used to indicate that Ctrl-C was pressed to terminate the
// Stork Agent.
type ctrlcError struct{}

// Returns ctrlcError error text.
func (e *ctrlcError) Error() string {
	return "received Ctrl-C signal"
}

// Helper function that starts agent, apps monitor and prometheus exports
// if they are enabled.
func runAgent(settings *generalSettings, reload bool) error {
	if !reload {
		// We need to print this statement only after we check if the only purpose is to print a version.
		log.Printf("Starting Stork Agent, version %s, build date %s", stork.Version, stork.BuildDate)
	}

	// Read the hook libraries.
	hookDirectory := settings.HookDirectory
	hookManager := agent.NewHookManager()
	// TODO: There is missing support for configuring agent hooks because the
	// agent uses a different library to handle CLI/environment variables than
	// the server. I think we should unify the CLI libraries to avoid
	// duplicating the code.
	err := hookManager.RegisterHooksFromDirectory(
		hooks.HookProgramAgent,
		hookDirectory,
		map[string]hooks.HookSettings{},
	)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.
				WithError(err).
				Warnf("The hook directory: '%s' doesn't exist", hookDirectory)
		} else {
			log.
				WithError(err).
				Fatal("Problem with loading hook libraries")
		}
	}

	skipTLSCertVerification := settings.SkipTLSCertVerification
	// Prepare the general HTTP client. It has no HTTP credentials or TLS
	// certificates.
	httpClient := agent.NewHTTPClient()
	httpClient.SetSkipTLSVerification(skipTLSCertVerification)

	// Try registering the agent in the server using the agent token.
	if settings.ServerURL != "" {
		portStr := strconv.FormatInt(int64(settings.Port), 10)

		if !agent.Register(settings.ServerURL, "", settings.Host, portStr, false, true, httpClient) {
			log.Fatalf("Problem with agent registration in Stork Server, exiting")
		}
	}

	// Start app monitor.
	appMonitor := agent.NewAppMonitor()

	// Prepare Kea HTTP client. It may use the certificates obtained during
	// the registration and GRPC credentials as TLS credentials.
	keaHTTPClient := agent.NewHTTPClient()
	keaHTTPClient.SetSkipTLSVerification(skipTLSCertVerification)

	ok, err := keaHTTPClient.LoadCredentials()
	switch {
	case err != nil:
		log.WithError(err).Fatal("Could not load the HTTP credentials")
	case !ok:
		log.Infof("The Basic Auth credentials file (%s) is missing - HTTP authentication is not used", agent.CredentialsFile)
	default:
		log.Infof("The Basic Auth credentials have been loaded from file (%s)", agent.CredentialsFile)
	}

	ok, err = keaHTTPClient.LoadGRPCCertificates()
	switch {
	case err != nil:
		log.WithError(err).Error("Could not load the GRPC credentials")
	case !ok:
		log.Warn("The GRPC credentials file is missing - the requests to Kea will not contain the client TLS certificate")
	default:
		log.Info("The GRPC credentials will be used as the client TLS certificate when connecting to Kea")
	}

	// Prepare agent gRPC handler
	storkAgent := agent.NewStorkAgent(settings.Host, settings.Port, appMonitor, httpClient, keaHTTPClient, hookManager)

	prometheusKeaExporterPerSubnetStats, err := storkutil.ParseBoolFlag(settings.PrometheusKeaExporterPerSubnetStats)
	if err != nil {
		return errors.WithMessage(err, "wrong value of the --prometheus-kea-exporter-per-subnet-stats flag")
	}

	// Prepare Prometheus exporters.
	promKeaExporter := agent.NewPromKeaExporter(
		settings.PrometheusKeaExporterAddress,
		settings.PrometheusKeaExporterPort,
		time.Duration(settings.PrometheusKeaExporterInterval)*time.Second,
		prometheusKeaExporterPerSubnetStats,
		appMonitor,
		keaHTTPClient,
	)
	promBind9Exporter := agent.NewPromBind9Exporter(
		settings.PrometheusBind9ExporterAddress,
		settings.PrometheusBind9ExporterPort,
		appMonitor,
		httpClient,
	)

	err = storkAgent.Setup()
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	// Let's start the app monitor.
	appMonitor.Start(storkAgent)

	// Only start the exporters if they're enabled.
	if !settings.ListenStorkOnly {
		promKeaExporter.Start()
		defer promKeaExporter.Shutdown()

		promBind9Exporter.Start()
		defer promBind9Exporter.Shutdown()
	}

	// Only start the agent service if it's enabled.
	if !settings.ListenPrometheusOnly {
		go func() {
			if err := storkAgent.Serve(); err != nil {
				log.Fatalf("Failed to serve the Stork Agent: %+v", err)
			}
		}()
		defer storkAgent.Shutdown(reload)
	}

	// Handle signals.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGHUP)
	sig := <-c
	switch sig {
	case syscall.SIGHUP:
		log.Info("Reloading Stork Agent after receiving SIGHUP signal")
		// Trigger shutdown with setting the reload flag. It doesn't
		// matter we have deferred another shutdown already. It will
		// be executed only once.
		storkAgent.Shutdown(true)
		return &sighupError{}
	default:
		log.Info("Received Ctrl-C signal")
		return &ctrlcError{}
	}
}

// Helper function that checks command line options and runs registration.
func runRegister(settings *registerSettings) {
	agentAddr := ""
	agentPort := ""
	var err error
	if settings.AgentHost != "" {
		agentAddr, agentPort, err = net.SplitHostPort(settings.AgentHost)
		if err != nil {
			log.Fatalf("Problem parsing agent host: %s\n", err)
		}
	}

	// check current user - it should be root or stork-agent
	user, err := user.Current()
	if err != nil {
		log.Fatalf("Cannot get info about current user: %s", err)
	}
	if user.Username != "root" && user.Username != "stork-agent" {
		log.Fatalf("Agent registration should be run by the user `root` or `stork-agent`")
	}

	// run Register
	httpClient := agent.NewHTTPClient()
	httpClient.SetSkipTLSVerification(settings.SkipTLSCertVerification)

	if agent.Register(settings.ServerURL, settings.ServerToken, agentAddr, agentPort, true, false, httpClient) {
		log.Println("Registration completed successfully")
	} else {
		log.Fatalf("Registration failed")
	}
}

// Read environment file settings. It's parsed before the main settings.
type environmentFileSettings struct {
	EnvFile    string `long:"env-file" description:"Environment file location; applicable only if the use-env-file is provided" default:"/etc/stork/server.env"`
	UseEnvFile bool   `long:"use-env-file" description:"Read the environment variables from the environment file"`
}

// General Stork Agent settings. They are used when no command is specified.
type generalSettings struct {
	environmentFileSettings
	Version                             bool   `short:"v" long:"version" description:"Show software version"`
	Host                                string `long:"host" description:"The IP or hostname to listen on for incoming Stork Server connections" default:"0.0.0.0" env:"STORK_AGENT_HOST"`
	Port                                int    `long:"port" description:"The TCP port to listen on for incoming Stork Server connections" default:"8080" env:"STORK_AGENT_PORT"`
	ListenPrometheusOnly                bool   `long:"listen-prometheus-only" description:"Listen for Prometheus requests only, but not for commands from the Stork Server" env:"STORK_AGENT_LISTEN_PROMETHEUS_ONLY"`
	ListenStorkOnly                     bool   `long:"listen-stork-only" description:"Listen for commands from the Stork Server only, but not for Prometheus requests" env:"STORK_AGENT_LISTEN_STORK_ONLY"`
	PrometheusKeaExporterAddress        string `long:"prometheus-kea-exporter-address" description:"The IP or hostname to listen on for incoming Prometheus connections" default:"0.0.0.0" env:"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS"`
	PrometheusKeaExporterPort           int    `long:"prometheus-kea-exporter-port" description:"The port to listen on for incoming Prometheus connections" default:"9547" env:"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT"`
	PrometheusKeaExporterInterval       int    `long:"prometheus-kea-exporter-interval" description:"How often the Stork Agent collects stats from Kea, in seconds" default:"10" env:"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL"`
	PrometheusKeaExporterPerSubnetStats string `long:"prometheus-kea-exporter-per-subnet-stats" description:"Enable or disable collecting per-subnet stats from Kea" optional:"true" optional-value:"true" default:"true" env:"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PER_SUBNET_STATS"`
	PrometheusBind9ExporterAddress      string `long:"prometheus-bind9-exporter-address" description:"The IP or hostname to listen on for incoming Prometheus connections" default:"0.0.0.0" env:"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS"`
	PrometheusBind9ExporterPort         int    `long:"prometheus-bind9-exporter-port" description:"The port to listen on for incoming Prometheus connections" default:"9119" env:"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT"`
	SkipTLSCertVerification             bool   `long:"skip-tls-cert-verification" description:"Skip TLS certificate verification when the Stork Agent makes HTTP calls over TLS" env:"STORK_AGENT_SKIP_TLS_CERT_VERIFICATION"`
	ServerURL                           string `long:"server-url" description:"The URL of the Stork Server, used in agent-token-based registration (optional alternative to server-token-based registration)" env:"STORK_AGENT_SERVER_URL"`
	HookDirectory                       string `long:"hook-directory" description:"The path to the hook directory" default:"/var/lib/stork-agent/hooks" env:"STORK_AGENT_HOOK_DIRECTORY"`
	Bind9Path                           string `long:"bind9-path" description:"Specify the path to BIND 9 config file. Does not need to be specified, unless the location is very uncommon." env:"STORK_BIND9_CONFIG"`
}

// Register command settings.
type registerSettings struct {
	environmentFileSettings
	// It is true if the register command was specified. Otherwise, it is false.
	commandSpecified        bool
	SkipTLSCertVerification bool   `long:"skip-tls-cert-verification" description:"Skip TLS certificate verification when the Stork Agent makes HTTP calls over TLS" env:"STORK_AGENT_SKIP_TLS_CERT_VERIFICATION"`
	ServerURL               string `short:"u" long:"server-url" description:"URL of Stork Server" env:"STORK_AGENT_SERVER_URL"`
	ServerToken             string `short:"t" long:"server-token" description:"Access token from Stork Server" env:"STORK_AGENT_SERVER_TOKEN"`
	AgentHost               string `short:"a" long:"agent-host" description:"IP address or DNS name with port of current agent host, e.g.: 10.11.12.13:8080" env:"STORK_AGENT_HOST"`
}

var _ flags.Commander = (*registerSettings)(nil)

// Implements the tools/golang/gopath/pkg/mod/github.com/jessevdk/go-flags@v1.5.0/command.go Commander interface.
// It is an only way to recognize which command was specified.
func (s *registerSettings) Execute(_ []string) error {
	s.commandSpecified = true
	return nil
}

// Parses the command line arguments. Returns the general settings if no command
// is specified, the register settings if the register command is specified,
// or an error if the arguments are invalid, the command is unknown, or the
// help is requested.
func parseArgs() (*generalSettings, *registerSettings, error) {
	shortGeneralDescription := "Stork Agent"
	longGeneralDescription := `This component is required on each machine to be monitored by the Stork Server

Stork logs at INFO level by default. Other levels can be configured using the
STORK_LOG_LEVEL variable. Allowed values are: DEBUG, INFO, WARN, ERROR.`

	// Parse environment file settings.
	envFileSettings := &environmentFileSettings{}
	parser := flags.NewParser(envFileSettings, flags.IgnoreUnknown)
	parser.ShortDescription = shortGeneralDescription
	parser.LongDescription = longGeneralDescription

	if _, err := parser.Parse(); err != nil {
		err = errors.Wrap(err, "invalid CLI argument")
		return nil, nil, err
	}

	// Load environment variables from the environment file.
	if envFileSettings.UseEnvFile {
		err := storkutil.LoadEnvironmentFileToSetter(
			envFileSettings.EnvFile,
			storkutil.NewProcessEnvironmentVariableSetter(),
		)
		if err != nil {
			err = errors.WithMessagef(err, "invalid environment file: '%s'", envFileSettings.EnvFile)
			return nil, nil, err
		}

		// Reconfigures logging using new environment variables.
		storkutil.SetupLogging()
	}

	// Prepare main parser.
	generalSettings := &generalSettings{}
	registerSettings := &registerSettings{}

	parser = flags.NewParser(generalSettings, flags.Default)
	parser.ShortDescription = shortGeneralDescription
	parser.LongDescription = longGeneralDescription

	parser.SubcommandsOptional = true
	_, err := parser.AddCommand(
		"register",
		"Register this machine in the Stork Server indicated by <server-url>",
		`Register the current agent in the Stork Server using provided server URL.

If server access token is provided using --server-token, then the agent is automatically
authorized (server-token-based registration). Otherwise, the agent requires explicit
authorization in the server using either the UI or the ReST API (agent-token-based registration).`,
		registerSettings,
	)
	if err != nil {
		err = errors.Wrap(err, "invalid CLI 'register' command")
		return nil, nil, err
	}

	// Parse command line arguments.
	_, err = parser.Parse()
	if err != nil {
		err = errors.Wrap(err, "invalid CLI argument")
		return nil, nil, err
	}

	if registerSettings.commandSpecified {
		generalSettings = nil
	} else {
		registerSettings = nil
	}

	return generalSettings, registerSettings, nil
}

// Check if a given error is a request to display the help.
func isHelpRequest(err error) bool {
	var flagsError *flags.Error
	if errors.As(err, &flagsError) {
		if flagsError.Type == flags.ErrHelp {
			return true
		}
	}
	return false
}

// Parses the command line arguments and runs the specific Stork Agent command.
func runApp(reload bool) error {
	generalSettings, registerSettings, err := parseArgs()
	if err != nil {
		if isHelpRequest(err) {
			return nil
		}
		return err
	}

	if generalSettings != nil {
		if generalSettings.Version {
			fmt.Println(stork.Version)
			return nil
		}

		if generalSettings.ServerURL != "" && generalSettings.Host == "0.0.0.0" {
			log.Errorf("Registration in Stork Server cannot be made because agent host address is not provided")
			log.Fatalf("Use --host option or the STORK_AGENT_HOST environment variable")
			return nil
		}

		return runAgent(generalSettings, reload)
	}

	if registerSettings != nil {
		runRegister(registerSettings)
		return nil
	}

	log.Fatalf("Unknown command specified")
	return nil
}

// Main stork-agent function.
func main() {
	reload := false
	for {
		storkutil.SetupLogging()
		err := runApp(reload)
		switch {
		case err == nil:
			return
		case errors.Is(err, &ctrlcError{}):
			// Ctrl-C pressed.
			os.Exit(130)
		case errors.Is(err, &sighupError{}):
			// SIGHUP signal received.
			reload = true
		default:
			// Error occurred.
			log.Fatal(err)
			// The default exit handler of logrus is suppressed in unit tests
			// to avoid interrupting the execution. So we need to explicitly
			// return here.
			return
		}
	}
}
