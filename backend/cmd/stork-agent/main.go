package main

import (
	"os"

	flags "github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	"isc.org/stork/agent"
	storkutil "isc.org/stork/util"
)

func main() {
	// Setup logging
	storkutil.SetupLogging()
	log.Printf("Starting Stork Agent, version %s, build date %s", stork.Version, stork.BuildDate)

	// Start app monitor
	appMonitor := agent.NewAppMonitor()

	// Prepare agent gRPC handler
	storkAgent := agent.NewStorkAgent(appMonitor)

	// Prepare Prometheus exporters
	promKeaExporter := agent.NewPromKeaExporter(appMonitor)
	promBind9Exporter := agent.NewPromBind9Exporter(appMonitor)

	// Prepare parse for command line flags.
	parser := flags.NewParser(&storkAgent.Settings, flags.Default)
	parser.ShortDescription = "Stork Agent"
	parser.LongDescription = "Stork Agent"

	_, err := parser.AddGroup("Prometheus Kea Exporter flags", "", &promKeaExporter.Settings)
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	_, err = parser.AddGroup("Prometheus BIND 9 Exporter flags", "", &promBind9Exporter.Settings)
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}

	if _, err := parser.Parse(); err != nil {
		code := 1
		if fe, ok := err.(*flags.Error); ok {
			if fe.Type == flags.ErrHelp {
				code = 0
			}
		}
		os.Exit(code)
	}

	promKeaExporter.Start()
	defer promKeaExporter.Shutdown()

	promBind9Exporter.Start()
	defer promBind9Exporter.Shutdown()

	storkAgent.Serve()
}
