// !!! ALWAYS BUILD go:build profiler

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

func Start(port int) func() {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(port))

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
		// Protection against Slowloris Attack (G112).
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() {
		log.WithField("address", addr).Info("Starting profiler")
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Error("Problem serving profiler")
		}
	}()

	return func() {
		log.Info("Stopping profiler")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.WithError(err).Warn("Could not gracefully shut down the profiler")
		}
		log.Info("Stopped profiler")
	}
}
