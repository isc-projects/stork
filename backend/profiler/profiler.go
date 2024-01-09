//go:build profiler

package profiler

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Starts the profiler endpoint on the given port.
// The endpoint is compliant with the pprof tool.
// It returns a function that can be used to stop the profiler.
//
// Warning: the profiler is not protected by any authentication mechanism. It
// should be used only in development environments.
func Start(port int) func() {
	// The "net/http/pprof" import automatically registers the handlers in the
	// default HTTP multiplexer. But the Stork server and agents use their own
	// multiplexers, so we avoid the default one and create a new one.
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Listen on the wildcard address to be able to access the profiler from
	// outside the container.
	server := &http.Server{
		Addr:    net.JoinHostPort("", strconv.Itoa(port)),
		Handler: mux,
		// Protection against Slowloris Attack (G112).
		ReadHeaderTimeout: 60 * time.Second,
	}

	// Start the profiler endpoint in a goroutine to avoid blocking the main
	// thread.
	go func() {
		log.WithField("address", server.Addr).Info("Starting profiler endpoint")
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Error("Problem serving profiler")
		}
	}()

	// Return a function that can be used to stop the profiler.
	return func() {
		log.Info("Stopping profiler endpoint")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.WithError(err).Warn("Could not gracefully shut down the profiler")
		}
		log.Info("Stopped profiler endpoint")
	}
}
