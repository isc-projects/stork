package main

import (
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server"
	storkutil "isc.org/stork/util"
)

func main() {
	// Setup logging
	storkutil.SetupLogging()

	// Initialize global state of Stork Server
	storkServer, err := server.NewStorkServer()
	if err != nil {
		log.Fatalf("unexpected error: %+v", err)
	}
	defer storkServer.Shutdown()

	storkServer.Serve()
}
