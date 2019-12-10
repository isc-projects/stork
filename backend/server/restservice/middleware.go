package restservice

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/auth"
)

// It installs a middleware that traces ReST calls using logrus.
func loggingMiddleware(next http.Handler) http.Handler {
	log.Info("installed logging middleware")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddr := r.RemoteAddr
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			remoteAddr = realIP
		}
		entry := log.WithFields(log.Fields{
			"path":   r.RequestURI,
			"method": r.Method,
			"remote": remoteAddr,
		})

		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)

		entry = entry.WithFields(log.Fields{
			//"status":      w.Status(),
			//"text_status": http.StatusText(w.Status()),
			"took": duration,
		})
		entry.Info("served request")
	})
}

// Global middleware function provides a common place to setup middlewares for
// the server.
func (r *RestAPI) GlobalMiddleware(handler http.Handler) http.Handler {
	handler = loggingMiddleware(handler)
	return r.SessionManager.SessionMiddleware(handler)
}

// Checks if the user us authorized to access the system (has session).
func (r *RestAPI) Authorizer(req *http.Request) error {
	ok, u := r.SessionManager.Logged(req.Context())
	if !ok {
		return fmt.Errorf("user unauthorized")
	}

	ok, _ = auth.Authorize(u, req)
	if !ok {
		return fmt.Errorf("user logged in but not allowed to access the resource")
	}

	return nil
}
