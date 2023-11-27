package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
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
func runAgent(settings *cli.Context, reload bool) error {
	if !reload {
		// We need to print this statement only after we check if the only purpose is to print a version.
		log.Printf("Starting Stork Agent, version %s, build date %s", stork.Version, stork.BuildDate)
	}

	// Read the hook libraries.
	hookDirectory := settings.Path("hook-directory")
	if !settings.IsSet("hook-directory") {
		// Default hook directory is used. Try to create it if it doesn't exist.
		err := os.MkdirAll(hookDirectory, 0o700)
		if err != nil {
			log.WithError(err).
				WithField("path", hookDirectory).
				Warning("Could not create the default hook directory")
		}
	}
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
				Warnf("The hook directory: '%s' doesn't exist", settings.Path("hook-directory"))
		} else {
			log.
				WithError(err).
				Fatal("Problem with loading hook libraries")
		}
	}

	skipTLSCertVerification := settings.Bool("skip-tls-cert-verification")
	// Prepare the general HTTP client. It has no HTTP credentials or TLS
	// certificates.
	httpClient := agent.NewHTTPClient()
	httpClient.SetSkipTLSVerification(skipTLSCertVerification)

	// Try registering the agent in the server using the agent token.
	if settings.String("server-url") != "" {
		portStr := strconv.FormatInt(settings.Int64("port"), 10)
		if err != nil {
			log.WithError(err).Fatal("Could not initialize the HTTP client")
		}

		if !agent.Register(settings.String("server-url"), "", settings.String("host"), portStr, false, true, httpClient) {
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
	storkAgent := agent.NewStorkAgent(settings, appMonitor, httpClient, keaHTTPClient, hookManager)

	// Prepare Prometheus exporters.
	promKeaExporter := agent.NewPromKeaExporter(settings, appMonitor, keaHTTPClient)
	promBind9Exporter := agent.NewPromBind9Exporter(settings, appMonitor, httpClient)

	err = storkAgent.Setup()
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	// Let's start the app monitor.
	appMonitor.Start(storkAgent)

	// Only start the exporters if they're enabled.
	if !settings.Bool("listen-stork-only") {
		promKeaExporter.Start()
		defer promKeaExporter.Shutdown()

		promBind9Exporter.Start()
		defer promBind9Exporter.Shutdown()
	}

	// Only start the agent service if it's enabled.
	if !settings.Bool("listen-prometheus-only") {
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
func runRegister(cfg *cli.Context) {
	agentAddr := ""
	agentPort := ""
	var err error
	if cfg.String("agent-host") != "" {
		agentAddr, agentPort, err = net.SplitHostPort(cfg.String("agent-host"))
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
	httpClient.SetSkipTLSVerification(cfg.Bool("skip-tls-cert-verification"))

	if agent.Register(cfg.String("server-url"), cfg.String("server-token"), agentAddr, agentPort, true, false, httpClient) {
		log.Println("Registration completed successfully")
	} else {
		log.Fatalf("Registration failed")
	}
}

// Prepare urfave cli app with all flags and commands defined.
func setupApp(reload bool) *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Version)
	}

	cli.HelpFlag = &cli.BoolFlag{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   "Show help",
	}

	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Print the version",
	}

	app := &cli.App{
		Name:     "Stork Agent",
		Usage:    "This component is required on each machine to be monitored by the Stork Server",
		Version:  stork.Version,
		HelpName: "stork-agent",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "host",
				Value:   "0.0.0.0",
				Usage:   "The IP or hostname to listen on for incoming Stork Server connections",
				EnvVars: []string{"STORK_AGENT_HOST"},
			},
			&cli.IntFlag{
				Name:    "port",
				Value:   8080,
				Usage:   "The TCP port to listen on for incoming Stork Server connections",
				EnvVars: []string{"STORK_AGENT_PORT"},
			},
			&cli.BoolFlag{
				Name:    "listen-prometheus-only",
				Usage:   "Listen for Prometheus requests only, but not for commands from the Stork Server",
				EnvVars: []string{"STORK_AGENT_LISTEN_PROMETHEUS_ONLY"},
			},
			&cli.BoolFlag{
				Name:    "listen-stork-only",
				Usage:   "Listen for commands from the Stork Server only, but not for Prometheus requests",
				EnvVars: []string{"STORK_AGENT_LISTEN_STORK_ONLY"},
			},
			// Prometheus Kea exporter settings
			&cli.StringFlag{
				Name:    "prometheus-kea-exporter-address",
				Value:   "0.0.0.0",
				Usage:   "The IP or hostname to listen on for incoming Prometheus connections",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS"},
			},
			&cli.IntFlag{
				Name:    "prometheus-kea-exporter-port",
				Value:   9547,
				Usage:   "The port to listen on for incoming Prometheus connections",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT"},
			},
			&cli.IntFlag{
				Name:    "prometheus-kea-exporter-interval",
				Value:   10,
				Usage:   "How often the Stork Agent collects stats from Kea, in seconds",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL"},
			},
			&cli.BoolFlag{
				Name:    "prometheus-kea-exporter-per-subnet-stats",
				Value:   true,
				Usage:   "Enable or disable collecting per-subnet stats from Kea",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PER_SUBNET_STATS"},
			},
			// Prometheus Bind 9 exporter settings
			&cli.StringFlag{
				Name:    "prometheus-bind9-exporter-address",
				Value:   "0.0.0.0",
				Usage:   "The IP or hostname to listen on for incoming Prometheus connections",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS"},
			},
			&cli.IntFlag{
				Name:    "prometheus-bind9-exporter-port",
				Value:   9119,
				Usage:   "The port to listen on for incoming Prometheus connections",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT"},
			},
			&cli.IntFlag{
				Name:    "prometheus-bind9-exporter-interval",
				Value:   10,
				Usage:   "How often the Stork Agent collects stats from BIND 9, in seconds",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_INTERVAL"},
			},
			&cli.BoolFlag{
				Name:    "skip-tls-cert-verification",
				Value:   false,
				Usage:   "Skip TLS certificate verification when the Stork Agent makes HTTP calls over TLS",
				EnvVars: []string{"STORK_AGENT_SKIP_TLS_CERT_VERIFICATION"},
			},
			// Registration related settings
			&cli.StringFlag{
				Name:    "server-url",
				Usage:   "The URL of the Stork Server, used in agent-token-based registration (optional alternative to server-token-based registration)",
				EnvVars: []string{"STORK_AGENT_SERVER_URL"},
			},
			&cli.PathFlag{
				Name:    "hook-directory",
				Usage:   "The path to the hook directory",
				EnvVars: []string{"STORK_AGENT_HOOK_DIRECTORY"},
				Value:   "/var/lib/stork-agent/hooks",
			},
			&cli.BoolFlag{
				Name:  "use-env-file",
				Usage: "Read the environment variables from the environment file",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "env-file",
				Usage: "Environment file location; applicable only if the use-env-file is provided",
				Value: "/etc/stork/agent.env",
			},
			// Those two parameters are read directly from the environment variable. They're listed here only for
			// the help.
			&cli.StringFlag{
				Name:    "bind9-path",
				Usage:   "Specify the path to BIND 9 config file. Does not need to be specified, unless the location is very uncommon.",
				Value:   "",
				EnvVars: []string{"STORK_BIND9_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "",
				Usage:   "Logging level can be specified using env variable only. Allowed values: are DEBUG, INFO, WARN, ERROR",
				Value:   "INFO",
				EnvVars: []string{"STORK_LOG_LEVEL"},
			},
		},
		Before: func(c *cli.Context) error {
			if c.Bool("use-env-file") {
				err := storkutil.LoadEnvironmentFileToSetter(
					c.Path("env-file"),
					// Loads environment variables into context.
					c,
					// Loads environment variables into process.
					storkutil.NewProcessEnvironmentVariableSetter(),
				)
				if err != nil {
					err = errors.WithMessagef(err, "the '%s' environment file is invalid", c.String("env-file"))
					return err
				}

				// Reconfigures logging using new environment variables.
				storkutil.SetupLogging()
			} else if c.IsSet("env-file") {
				log.Warning("The environment file is provided but it is not used because the '--use-env-file' flag is not set")
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			if c.String("server-url") != "" && c.String("host") == "0.0.0.0" {
				log.Errorf("Registration in Stork Server cannot be made because agent host address is not provided")
				log.Fatalf("Use --host option or the STORK_AGENT_HOST environment variable")
			}

			err := runAgent(c, reload)
			return err
		},
		Commands: []*cli.Command{
			{
				Name:      "register",
				Usage:     "Register this machine in the Stork Server indicated by <server-url>",
				UsageText: "stork-agent register [options]",
				Description: `Register the current agent in the Stork Server using provided server URL.

If server access token is provided using --server-token, then the agent is automatically
authorized (server-token-based registration). Otherwise, the agent requires explicit
authorization in the server using either the UI or the ReST API (agent-token-based registration).`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "server-url",
						Usage:   "URL of Stork Server",
						Aliases: []string{"u"},
						EnvVars: []string{"STORK_AGENT_SERVER_URL"},
					},
					&cli.StringFlag{
						Name:    "server-token",
						Usage:   "Access token from Stork Server",
						Aliases: []string{"t"},
						EnvVars: []string{"STORK_AGENT_SERVER_TOKEN"},
					},
					&cli.StringFlag{
						Name:    "agent-host",
						Usage:   "IP address or DNS name with port of current agent host, e.g.: 10.11.12.13:8080",
						Aliases: []string{"a"},
						EnvVars: []string{"STORK_AGENT_HOST"},
					},
				},
				Action: func(c *cli.Context) error {
					runRegister(c)
					return nil
				},
			},
		},
	}

	return app
}

// Main stork-agent function.
func main() {
	reload := false
	for {
		storkutil.SetupLogging()
		app := setupApp(reload)
		err := app.Run(os.Args)
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
			return
		}
	}
}
