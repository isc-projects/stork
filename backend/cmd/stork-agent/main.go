package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"isc.org/stork"
	"isc.org/stork/agent"
	storkutil "isc.org/stork/util"
)

// Helper function that starts agent, apps monitor and prometheus exports
// if they are enabled.
func runAgent(settings *cli.Context) {
	// We need to print this statement only after we check if the only purpose is to print a version.
	log.Printf("Starting Stork Agent, version %s, build date %s", stork.Version, stork.BuildDate)

	// try register agent in the server using agent token
	if settings.String("server-url") != "" {
		portStr := strconv.FormatInt(settings.Int64("port"), 10)
		if !agent.Register(settings.String("server-url"), "", settings.String("host"), portStr, false, true) {
			log.Fatalf("problem with agent registration in Stork server, exiting")
		}
	}

	// Start app monitor
	appMonitor := agent.NewAppMonitor()

	// Prepare agent gRPC handler
	storkAgent := agent.NewStorkAgent(settings, appMonitor)

	// Prepare Prometheus exporters
	promKeaExporter := agent.NewPromKeaExporter(settings, appMonitor)
	promBind9Exporter := agent.NewPromBind9Exporter(settings, appMonitor)

	err := storkAgent.Setup()
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	// Let's start the app monitor.
	appMonitor.Start(storkAgent)

	// Only start the exporters if they're enabled.
	if !settings.Bool("stork-only") {
		promKeaExporter.Start()
		defer promKeaExporter.Shutdown()

		promBind9Exporter.Start()
		defer promBind9Exporter.Shutdown()
	}

	// Only start the agent service if it's enabled.
	if !settings.Bool("prometheus-only") {
		go storkAgent.Serve()
		defer storkAgent.Shutdown()
	}

	// We wait for ctl-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	<-c
}

// Helper function that checks command line options and runs registration.
func runRegister(cfg *cli.Context) {
	agentAddr := ""
	agentPort := ""
	var err error
	if cfg.String("agent-address") != "" {
		agentAddr, agentPort, err = net.SplitHostPort(cfg.String("agent-address"))
		if err != nil {
			log.Fatalf("problem with parsing agent address: %s\n", err)
		}
	}

	// check current user - it should be root or stork-agent
	user, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get info about current user: %s", err)
	}
	if user.Username != "root" && user.Username != "stork-agent" {
		log.Fatalf("agent registration should be run by `root` or `stork-agent` user")
	}

	// run Register
	if agent.Register(cfg.String("server-url"), cfg.String("token"), agentAddr, agentPort, true, false) {
		log.Println("registration completed successfully")
	} else {
		log.Fatalf("registration failed")
	}
}

// Prepare urfave cli app with all flags and commands defined.
func setupApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Version)
	}
	app := &cli.App{
		Name:     "Stork Agent",
		Version:  stork.Version,
		HelpName: "stork-agent",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "host",
				Value:   "0.0.0.0",
				Usage:   "the IP or hostname to listen on for incoming Stork server connection",
				EnvVars: []string{"STORK_AGENT_ADDRESS"},
			},
			&cli.IntFlag{
				Name:    "port",
				Value:   8080,
				Usage:   "the TCP port to listen on for incoming Stork server connection",
				EnvVars: []string{"STORK_AGENT_PORT"},
			},
			&cli.BoolFlag{
				Name:    "listen-prometheus-only",
				Usage:   "listen for Prometheus requests only, but not for commands from the Stork Server",
				EnvVars: []string{"STORK_AGENT_LISTEN_PROMETHEUS_ONLY"},
			},
			&cli.BoolFlag{
				Name:    "listen-stork-only",
				Usage:   "listen for commands from the Stork Server only, but not for Prometheus requests",
				EnvVars: []string{"STORK_AGENT_LISTEN_STORK_ONLY"},
			},
			// Prometheus Kea exporter settings
			&cli.StringFlag{
				Name:    "prometheus-kea-exporter-host",
				Value:   "0.0.0.0",
				Usage:   "the IP or hostname to listen on for incoming Prometheus connection",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS"},
			},
			&cli.IntFlag{
				Name:    "prometheus-kea-exporter-port",
				Value:   9547,
				Usage:   "the port to listen on for incoming Prometheus connection",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT"},
			},
			&cli.IntFlag{
				Name:    "prometheus-kea-exporter-interval",
				Value:   10,
				Usage:   "specifies how often the agent collects stats from Kea, in seconds",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL"},
			},
			// Prometheus Bind 9 exporter settings
			&cli.StringFlag{
				Name:    "prometheus-bind9-exporter-host",
				Value:   "0.0.0.0",
				Usage:   "the IP or hostname to listen on for incoming Prometheus connection",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS"},
			},
			&cli.IntFlag{
				Name:    "prometheus-bind9-exporter-port",
				Value:   9119,
				Usage:   "the port to listen on for incoming Prometheus connection",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT"},
			},
			&cli.IntFlag{
				Name:    "prometheus-bind9-exporter-interval",
				Value:   10,
				Usage:   "specifies how often the agent collects stats from BIND 9, in seconds",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_INTERVAL"},
			},
			// Registration related settings
			&cli.StringFlag{
				Name:    "server-url",
				Usage:   "URL of Stork server, used in agent token based registration (optional, alternative to server token based registration)",
				EnvVars: []string{"STORK_AGENT_SERVER_URL"},
			},
		},
		Action: func(c *cli.Context) error {
			if c.String("server-url") != "" && c.String("host") == "0.0.0.0" {
				log.Errorf("registration in Stork server cannot be made because agent host address is not provided")
				log.Fatalf("use --host option or STORK_AGENT_ADDRESS environment variable")
			}

			runAgent(c)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:      "register",
				Usage:     "register this machine in Stork server indicated by <server-url>",
				UsageText: "stork-agent register [options]",
				Description: `Register current agent in Stork server using provided server URL.

If server access token is provided using --server-token then the agent is automatically
authorized (server token based registration). Otherwise, the agent requires explicit
authorization in the server using either UI or ReST API (agent token based registration).`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "server-url",
						Usage:   "URL of Stork server",
						Aliases: []string{"u"},
						EnvVars: []string{"STORK_AGENT_SERVER_URL"},
					},
					&cli.StringFlag{
						Name:    "token",
						Usage:   "access token from Stork server",
						Aliases: []string{"t"},
						EnvVars: []string{"STORK_AGENT_SERVER_TOKEN"},
					},
					&cli.StringFlag{
						Name:    "agent-address",
						Usage:   "IP address or DNS name with port of current agent host, eg: 10.11.12.13:8080",
						Aliases: []string{"a"},
						EnvVars: []string{"STORK_AGENT_ADDRESS"},
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

func main() {
	// Setup logging
	storkutil.SetupLogging()

	app := setupApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
