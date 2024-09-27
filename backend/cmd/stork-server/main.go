package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	"isc.org/stork/profiler"
	"isc.org/stork/server"
	storkutil "isc.org/stork/util"
)

// Main stork-server function. It starts the main loop and is the higher-level
// error handler.
func main() {
	// Start profiler.
	profilerShutdown := profiler.Start(profiler.ServerProfilerPort)

	returnCode, err := mainLoop()

	// We cannot defer the shutdown because the deferred functions are not
	// executed when we exit with os.Exit() or log.Fatal.
	profilerShutdown()

	if err != nil {
		log.WithError(err).Fatal("Unexpected error")
	}
	os.Exit(returnCode)
}

// Main execution loop. It starts the application and handles the system
// signals.
func mainLoop() (int, error) {
	// Both variables are used in cases when the server reloads as a result
	// of receiving the SIGHUP signal. The password is saved to avoid prompting
	// the user for the password again. The reload flag indicates whether we
	// are starting the server or we reload it.
	var (
		savedPassword string
		reload        bool
	)
	for {
		// Setup logging.
		storkutil.SetupLogging()

		// Initialize global Stork Server state.
		storkServer, command, err := server.NewStorkServer()
		if err != nil {
			return 1, err
		}

		switch command {
		case server.HelpCommand:
			// The help command is handled internally by flags framework.
			return 0, nil
		case server.VersionCommand:
			fmt.Printf("%s\n", stork.Version)
			return 0, nil
		case server.NoneCommand:
			// Nothing to do.
			return 0, nil
		case server.RunCommand:
			// Handled below.
			break
		default:
			return 1, errors.Errorf("Not implemented command: %s", command)
		}

		// If we reload the server after receiving the SIGHUP signal we may already
		// have the password. Let's use the same password, so the user is not
		// prompted again.
		if len(savedPassword) != 0 {
			storkServer.DBSettings.Password = savedPassword
		}

		// Actually run the server according to the command line switches.
		err = storkServer.Bootstrap(reload)
		if err != nil {
			return 1, errors.WithMessage(err, "Cannot start the Stork Server")
		}

		// Only indicate that the server is starting when we don't reload it.
		if !reload {
			log.Printf("Starting Stork Server, version %s, build date %s", stork.Version, stork.BuildDate)
		}

		// Handle signals.
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGHUP)
		c := make(chan os.Signal, 1)
		go func() {
			sig := <-sigs
			switch sig {
			case syscall.SIGHUP:
				log.Info("Reloading Stork Server after receiving SIGHUP signal")
				reload = true
			default:
				log.Info("Received Ctrl-C signal")
				reload = false
			}
			// Trigger server shutdown breaking the Serve() function.
			storkServer.Shutdown(reload)
			// Pass the signal received to the main routine.
			c <- sig
		}()

		// Start blocking Serve().
		storkServer.Serve()
		// This second Shutdown can only be triggered if we are stopping due
		// to something else than SIGINT, SIGHUP.
		storkServer.Shutdown(false)

		select {
		case sig := <-c:
			// If we have received Ctrl-C signal we should exit with appropriate
			// error code.
			if sig != syscall.SIGHUP {
				return 130, nil
			}
			// For the SIGHUP, we don't exit and reload the server.
			// Save the current database password to avoid prompting.
			savedPassword = storkServer.DBSettings.Password
		default:
			// We're terminating for some other reason than Ctrl-C.
			return 0, nil
		}
	}
}
