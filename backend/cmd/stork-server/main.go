package main

import (
	"fmt"
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

	// Initialize global state of Stork Server
	storkServer, command, err := server.NewStorkServer()
	if err != nil {
		log.Fatalf("Unexpected error: %+v", err)
	}

	switch command {
	case server.HelpCommand:
		// The help command is handled internally by flags framework.
		return
	case server.VersionCommand:
		fmt.Printf("%s\n", stork.Version)
		return
	case server.NoneCommand:
		// Nothing to do.
		return
	case server.RunCommand:
		// Handled below.
		break
	default:
		log.Fatalf("not implemented command: %s", command)
	}

	err = storkServer.Bootstrap()
	if err != nil {
		log.Fatalf("cannot start the Stork Server: %+v", err)
	}

	log.Printf("Starting Stork Server, version %s, build date %s", stork.Version, stork.BuildDate)

	// Setup graceful shutdown on Ctrl-C
	c := make(chan os.Signal, 1)
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
