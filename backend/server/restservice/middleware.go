package restservice

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/auth"
	"isc.org/stork/server/eventcenter"
	"isc.org/stork/server/metrics"
)

var (
	_ http.Flusher        = (*loggingResponseWriter)(nil)
	_ http.ResponseWriter = (*loggingResponseWriter)(nil)
)

// Struct for holding response details.
type responseData struct {
	status int
	size   int
}

// Our http.ResponseWriter implementation.
type loggingResponseWriter struct {
	rw           http.ResponseWriter // compose original http.ResponseWriter
	responseData *responseData
}

// http.ResponseWriter Write implementation wrapper that captures size
// of the response.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// write response using original http.ResponseWriter
	size, err := r.rw.Write(b)
	// capture size
	r.responseData.size += size
	return size, err
}

// http.ResponseWriter WriteHeader implementation wrapper
// that captures status code of the response.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// write status code using original http.ResponseWriter
	r.rw.WriteHeader(statusCode)
	// capture status code
	r.responseData.status = statusCode
}

// http.ResponseWriter Header implementation wrapper
// that returns the header.
func (r *loggingResponseWriter) Header() http.Header {
	return r.rw.Header()
}

// http.Flusher implementation wrapper.
func (r *loggingResponseWriter) Flush() {
	if flusher, ok := r.rw.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Install a middleware that traces ReST calls using logrus.
func loggingMiddleware(next http.Handler) http.Handler {
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

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lrw := &loggingResponseWriter{
			rw:           w, // compose original http.ResponseWriter
			responseData: responseData,
		}

		entry.Info("HTTP request incoming")

		start := time.Now()

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)

		entry = entry.WithFields(log.Fields{
			"status":      responseData.status,
			"text_status": http.StatusText(responseData.status),
			"took":        duration,
			"size":        responseData.size,
		})
		entry.Info("HTTP request served")
	})
}

// Install a middleware that is serving static files for UI
// and assets/pkgs content ie. stork rpm and deb packages.
func fileServerMiddleware(next http.Handler, staticFilesDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || r.URL.Path == "/swagger.json" {
			// Serve API request.
			next.ServeHTTP(w, r)
		} else {
			// The "r.URL.Path" is provided by the user and must be treated as
			// untrusted. Otherwise, it can be used to perform the Path
			// Traversal attack. The below "resourcePath" has limited usage; it
			// is only used to check the existence of the resource. It means
			// that without the "path.Clean" call, the attacker could check the
			// existence of any file on the filesystem (available for a user
			// that runs the Stork) but couldn't read its content because the
			// resource reading is performed by "http.FileServer" call that
			// sanitizes the path on its own.
			urlPath := r.URL.Path
			if !strings.HasPrefix(urlPath, "/") {
				// The resource path must be rooted to work the "path.Clean"
				// function properly. It causes the returned path to not point
				// to any file outside the root directory (in this context, it
				// is the "staticFilesDir" directory).
				// The web framework always returns a rooted path, but the
				// "url" package doesn't guarantee it.
				urlPath = "/" + urlPath
			}
			resourcePath := path.Join(staticFilesDir, path.Clean(urlPath))
			if _, err := os.Stat(resourcePath); os.IsNotExist(err) {
				// The static-page-content subdirectory contains optional files that
				// can hold html to be embedded in different components. It is not an
				// error if these files do not exist. We return HTTP NoContent status
				// to indicate that the requested file does not exist.
				if strings.HasPrefix(urlPath, "/assets/static-page-content") {
					w.WriteHeader(http.StatusNoContent)
				} else {
					// If file does not exist then return content of index.html.
					http.ServeFile(w, r, path.Join(staticFilesDir, "index.html"))
				}
			} else {
				// If file exists then serve it.
				http.FileServer(http.Dir(staticFilesDir)).ServeHTTP(w, r)
			}
		}
	})
}

// Install a middleware that is serving `server-sent events` (SSE).
func sseMiddleware(next http.Handler, eventCenter eventcenter.EventCenter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/sse") {
			eventCenter.ServeHTTP(w, r)
		} else {
			// pass request to another handler
			next.ServeHTTP(w, r)
		}
	})
}

// Install a middleware that is serving Agent installer.
func agentInstallerMiddleware(next http.Handler, staticFilesDir string) http.Handler {
	// Agent installer as Bash script.
	const agentInstallerScript = `#!/bin/sh
set -e -x

rm -f /tmp/isc-stork-agent.{deb,rpm,apk}

{{ if .DebPath }}
if [ -e /etc/debian_version ]; then
    curl -o /tmp/isc-stork-agent.deb "{{.ServerAddress}}/{{.DebPath}}"
    DEBIAN_FRONTEND=noninteractive dpkg -i --force-confold /tmp/isc-stork-agent.deb
fi
{{ end }}
{{ if .ApkPath }}
if [ -e /etc/alpine-release ]; then
	wget -O /tmp/isc-stork-agent.apk "{{.ServerAddress}}/{{.ApkPath}}"
	apk add --no-cache --no-network /tmp/isc-stork-agent.apk
fi
{{ end }}
{{ if .RpmPath }}
if [ -e /etc/redhat-release ]; then
    curl -o /tmp/isc-stork-agent.rpm "{{.ServerAddress}}/{{.RpmPath}}"
    yum install -y /tmp/isc-stork-agent.rpm
fi
{{ end }}

systemctl daemon-reload
systemctl enable isc-stork-agent
systemctl restart isc-stork-agent
systemctl status isc-stork-agent

su stork-agent -s /bin/sh -c 'stork-agent register -u {{.ServerAddress}}'

`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/stork-install-agent.sh") {
			pkgsRelativeDir := "assets/pkgs"
			pkgsDir := path.Join(staticFilesDir, pkgsRelativeDir)
			files, err := os.ReadDir(pkgsDir)
			if err != nil {
				msg := fmt.Sprintf("Problem reading '%s' directory with packages\n", pkgsDir)
				log.WithError(err).Error(msg)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, msg)
				return
			}

			packageExtensions := []string{".deb", ".rpm", ".apk"}
			packageFiles := map[string]string{}
			for _, f := range files {
				if !strings.HasPrefix(f.Name(), "isc-stork-agent") {
					continue
				}

				for _, extension := range packageExtensions {
					if strings.HasSuffix(f.Name(), extension) {
						packageFiles[extension] = path.Join(pkgsRelativeDir, f.Name())
					}
				}
			}

			if len(packageFiles) == 0 {
				msg := fmt.Sprintf(
					"Cannot find any agent package in '%s' directory. You "+
						"must download the Stork agent packages from "+
						"CloudSmith and put them in that directory to make "+
						"stork-install-agent.sh work.\n",
					pkgsDir,
				)
				log.Error(msg)
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, msg)
				return
			} else if len(packageFiles) < len(packageExtensions) {
				var availableExtensions []string
				for extension := range packageFiles {
					availableExtensions = append(availableExtensions, extension)
				}

				log.Warningf(
					"Not all supported stork-agent package types are "+
						"available for the agent installation using a server "+
						"token. It will cause the agent installation errors "+
						"on the systems using packages other than these: %s",
					strings.Join(availableExtensions, ", "),
				)
			}

			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}

			serverAddress := url.URL{
				Scheme: scheme,
				Host:   r.Host,
			}

			data := map[string]string{
				"ServerAddress": serverAddress.String(),
			}

			for extension, path := range packageFiles {
				key := strings.TrimLeft(extension, ".")
				key = strings.ToUpper(key[0:1]) + key[1:] + "Path"
				data[key] = path
			}

			t := template.Must(template.New("script").Parse(agentInstallerScript))
			err = t.Execute(w, data)
			if err != nil {
				msg := "Problem preparing install script"
				log.WithError(err).Error(msg)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, msg)
				return
			}
		} else {
			// pass request to another handler
			next.ServeHTTP(w, r)
		}
	})
}

// Metric collector middleware that handles the metric endpoint.
func metricsMiddleware(next http.Handler, collector metrics.Collector) http.Handler {
	var handler http.Handler
	if collector != nil {
		// Proper handler
		handler = collector.GetHTTPHandler(next)
	} else {
		// Placeholder handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			help := "The metrics collector endpoint is disabled."
			_, _ = w.Write([]byte(help))
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/metrics") {
			handler.ServeHTTP(w, r)
		} else {
			// pass request to another handler
			next.ServeHTTP(w, r)
		}
	})
}

// Middelware that trims the base URL from the request URL.
func trimBaseURLMiddleware(next http.Handler, baseURL string) http.Handler {
	if baseURL == "" || baseURL == "/" {
		// Nothing to do.
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		if !strings.HasPrefix(urlPath, "/") {
			urlPath = "/" + urlPath
		}
		if strings.HasPrefix(urlPath, baseURL) {
			urlPath = "/" + urlPath[len(baseURL):]
		}
		r.URL.Path = urlPath

		next.ServeHTTP(w, r)
	})
}

// Global middleware function provides a common place to setup middlewares for
// the server. It is invoked before everything.
func (r *RestAPI) GlobalMiddleware(handler http.Handler, staticFilesDir, baseURL string, eventCenter eventcenter.EventCenter) http.Handler {
	// last handler is executed first for incoming request
	handler = fileServerMiddleware(handler, staticFilesDir)
	handler = agentInstallerMiddleware(handler, staticFilesDir)
	handler = sseMiddleware(handler, eventCenter)
	handler = metricsMiddleware(handler, r.MetricsCollector)
	handler = trimBaseURLMiddleware(handler, baseURL)
	handler = loggingMiddleware(handler)
	return handler
}

// Inner middleware function provides a common place to setup middlewares for
// the server. It is invoked after routing but before authentication, binding and validation.
func (r *RestAPI) InnerMiddleware(handler http.Handler) http.Handler {
	// last handler is executed first for incoming request
	handler = r.SessionManager.SessionMiddleware(handler)
	return handler
}

// Checks if the user us authorized to access the system (has session).
func (r *RestAPI) Authorizer(req *http.Request) error {
	ok, u := r.SessionManager.Logged(req.Context())
	if !ok {
		return errors.Errorf("user unauthorized")
	}

	ok, _ = auth.Authorize(u, req)
	if !ok {
		return errors.Errorf("user logged in but not allowed to access the resource")
	}

	return nil
}
