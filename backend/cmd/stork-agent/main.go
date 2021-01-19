package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"isc.org/stork"
	"isc.org/stork/agent"
	storkutil "isc.org/stork/util"
)

func runAgent(config *cli.Context) {
	// We need to print this statement only after we check if the only purpose is to print a version.
	log.Printf("Starting Stork Agent, version %s, build date %s", stork.Version, stork.BuildDate)

	// Start app monitor
	appMonitor := agent.NewAppMonitor()

	// Prepare agent gRPC handler
	storkAgent := agent.NewStorkAgent(config, appMonitor)

	// Prepare Prometheus exporters
	promKeaExporter := agent.NewPromKeaExporter(config, appMonitor)
	promBind9Exporter := agent.NewPromBind9Exporter(config, appMonitor)

	// Let's start the app monitor.
	appMonitor.Start(storkAgent)

	// Only start the exporters if they're enabled.
	if !config.Bool("stork-only") {
		promKeaExporter.Start()
		defer promKeaExporter.Shutdown()

		promBind9Exporter.Start()
		defer promBind9Exporter.Shutdown()
	}

	// Only start the agent service if it's enabled.
	if !config.Bool("prometheus-only") {
		go storkAgent.Serve()
		defer storkAgent.Shutdown()
	}

	// We wait for ctl-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	<-c
}

func setupApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Version)
	}
	app := &cli.App{
		Name:    "Stork Agent",
		Version: stork.Version,
		// Compiled: stork.BuildDate,
		Copyright: "(c) 2019-2021 ISC",
		HelpName:  "stork-agent",
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
				Usage:   "specifies how often the agent collects stats from Kea, in seconds",
				EnvVars: []string{"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_INTERVAL"},
			},
			// Registration related settings
			&cli.StringFlag{
				Name:    "server-url",
				Usage:   "URL of Stork server, used in agent-token based registration (optional, alternative to server-token based registration)",
				EnvVars: []string{"STORK_AGENT_SERVER_URL"},
			},
		},
		Action: func(c *cli.Context) error {
			if c.String("server-url") != "" && c.String("host") == "0.0.0.0" {
				log.Errorf("registration in Stork server cannot be made because agent host address is not provided")
				log.Errorf("use --host option or STORK_AGENT_HOST environment variable")
				os.Exit(1)
			}

			runAgent(c)
			return nil
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
