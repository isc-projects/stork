package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	"isc.org/stork/server"
	storkutil "isc.org/stork/util"
)

func main() {
	// Initialize random numbers generator.
	rand.Seed(time.Now().UnixNano())

	// Setup logging
	storkutil.SetupLogging()

	// Initialize global state of Stork Server
	storkServer, err := server.NewStorkServer()
	if err != nil {
		log.Fatalf("unexpected error: %+v", err)
	}

	log.Printf("Starting Stork Server, version %s, build date %s", stork.Version, stork.BuildDate)

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
