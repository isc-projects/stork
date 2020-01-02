package main

import (
	"os"

	flags "github.com/jessevdk/go-flags"

	"isc.org/stork/agent"
	"isc.org/stork/util"
)


func main() {
	// Setup logging
	storkutil.SetupLogging()

	// Start app monitor
	sm := agent.NewAppMonitor()
	storkAgent := agent.StorkAgent{
		AppMonitor: sm,
	}

	// Prepare parse for command line flags.
	parser := flags.NewParser(&storkAgent.Settings, flags.Default)
	parser.ShortDescription = "Stork Agent"
	parser.LongDescription = "Stork Agent"

	if _, err := parser.Parse(); err != nil {
		code := 1
		if fe, ok := err.(*flags.Error); ok {
			if fe.Type == flags.ErrHelp {
				code = 0
			}
		}
		os.Exit(code)
	}

	storkAgent.Serve()
}
