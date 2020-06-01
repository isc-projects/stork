package restservice

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/auth"
	"isc.org/stork/server/eventcenter"
)

// Install a middleware that traces ReST calls using logrus.
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

// Install a middleware that is serving static files for UI.
func fileServerMiddleware(next http.Handler, staticFilesDir string) http.Handler {
	log.Info("installed file server middleware")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || r.URL.Path == "/swagger.json" {
			// serve API request
			next.ServeHTTP(w, r)
		} else {
			pth := path.Join(staticFilesDir, r.URL.Path)
			if _, err := os.Stat(pth); os.IsNotExist(err) {
				// if file does not exists then return content of index.html
				http.ServeFile(w, r, path.Join(staticFilesDir, "index.html"))
			} else {
				// if file exists then serve it
				http.FileServer(http.Dir(staticFilesDir)).ServeHTTP(w, r)
			}
		}
	})
}

// Install a middleware that is serving `server-sent events` (SSE).
func sseMiddleware(next http.Handler, eventCenter eventcenter.EventCenter) http.Handler {
	log.Info("installed SSE middleware")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/sse") {
			eventCenter.ServeHTTP(w, r)
		} else {
			// pass request to another handler
			next.ServeHTTP(w, r)
		}
	})
}

// Global middleware function provides a common place to setup middlewares for
// the server. It is invoked before everything.
func (r *RestAPI) GlobalMiddleware(handler http.Handler, staticFilesDir string, eventCenter eventcenter.EventCenter) http.Handler {
	// last handler is executed first for incoming request
	handler = fileServerMiddleware(handler, staticFilesDir)
	handler = sseMiddleware(handler, eventCenter)
	handler = loggingMiddleware(handler)
	return handler
}

// Inner middleware function provides a common place to setup middlewares for
// the server. It is invoked after routing but before authentication, binding and validation
func (r *RestAPI) InnerMiddleware(handler http.Handler) http.Handler {
	// last handler is executed first for incoming request
	handler = r.SessionManager.SessionMiddleware(handler)
	return handler
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
