package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	"isc.org/stork/server"
	storkutil "isc.org/stork/util"
)

func main() {
	// Setup logging
	storkutil.SetupLogging()
	log.Printf("Starting Stork Server, version %s, build date %s", stork.Version, stork.BuildDate)

	// Initialize global state of Stork Server
	storkServer, err := server.NewStorkServer()
	if err != nil {
		log.Fatalf("unexpected error: %+v", err)
	}

	// Setup graceful shutdown on Ctrl-C
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		log.Println("Received Ctrl-C signal")
		storkServer.Shutdown()
		os.Exit(130)
	}()

	storkServer.Serve()
	storkServer.Shutdown()
}
