package main

import (
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	"isc.org/stork/server"
)

func main() {
	// Setup logging
	stork.SetupLogging()

	// Initialize global state of Stork Server
	storkServer, err := server.NewStorkServer()
	if err != nil {
		log.Fatalf("unexpected error: %+v", err)
	}
	defer storkServer.Shutdown()

	storkServer.Serve();
}
