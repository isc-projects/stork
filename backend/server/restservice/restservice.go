package restservice

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	flags "github.com/jessevdk/go-flags"
	"golang.org/x/net/netutil"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/runtime/flagext"
	"github.com/pkg/errors"

	"isc.org/stork/server/gen/restapi"
	"isc.org/stork/server/gen/restapi/operations"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/database"
	"isc.org/stork/server/database/session"
)

type RestApiSettings struct {
	CleanupTimeout   time.Duration    `long:"rest-cleanup-timeout" description:"grace period for which to wait before killing idle connections" default:"10s"`
	GracefulTimeout  time.Duration    `long:"rest-graceful-timeout" description:"grace period for which to wait before shutting down the server" default:"15s"`
	MaxHeaderSize    flagext.ByteSize `long:"rest-max-header-size" description:"controls the maximum number of bytes the server will read parsing the request header's keys and values, including the request line. It does not limit the size of the request body." default:"1MiB"`

	Host         string        `long:"rest-host" description:"the IP to listen on" default:"" env:"STORK_REST_HOST"`
	Port         int           `long:"rest-port" description:"the port to listen on for connections" default:"8080" env:"STORK_REST_PORT"`
	ListenLimit  int           `long:"rest-listen-limit" description:"limit the number of outstanding requests"`
	KeepAlive    time.Duration `long:"rest-keep-alive" description:"set the TCP keep-alive timeouts on accepted connections. It prunes dead TCP connections ( e.g. closing laptop mid-download)" default:"3m"`
	ReadTimeout  time.Duration `long:"rest-read-timeout" description:"maximum duration before timing out read of the request" default:"30s"`
	WriteTimeout time.Duration `long:"rest-write-timeout" description:"maximum duration before timing out write of the response" default:"60s"`

	TLSCertificate    flags.Filename `long:"rest-tls-certificate" description:"the certificate to use for secure connections" env:"STORK_REST_TLS_CERTIFICATE"`
	TLSCertificateKey flags.Filename `long:"rest-tls-key" description:"the private key to use for secure connections" env:"STORK_REST_TLS_PRIVATE_KEY"`
	TLSCACertificate  flags.Filename `long:"rest-tls-ca" description:"the certificate authority file to be used with mutual tls auth" env:"STORK_REST_TLS_CA_CERTIFICATE"`
}

// Runtime information and settings for ReST API service.
type RestAPI struct {
	Settings     RestApiSettings

	Agents       agentcomm.ConnectedAgents

	TLS          bool
	srvListener  net.Listener
	api          *operations.StorkAPI
	handler      http.Handler
	hasListeners bool
	shuttingDown int32
	Host         string  // actual host for listening
	Port         int     // actual port for listening
}

// It installs a middleware that traces ReST calls using logrus.
func loggingMiddleware(next http.Handler) http.Handler {
	log.Info("installed logging middleware");
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddr := r.RemoteAddr
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			remoteAddr = realIP
		}
		entry := log.WithFields(log.Fields{
			"path": r.RequestURI,
			"method": r.Method,
			"remote": remoteAddr,
		})

		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)

		entry = entry.WithFields(log.Fields{
			//"status":      w.Status(),
			//"text_status": http.StatusText(w.Status()),
			"took":        duration,
		})
		entry.Info("served request")
	})
}

// Do API initialization, create a new API http handler.
func (r *RestAPI) Init(agents agentcomm.ConnectedAgents) error {
	r.Agents = agents

	// Initialize sessions with access to the database.
	dbconn := dbops.NewGenericConn()
	*dbconn = dbops.GenericConn{
		User: "storktest",
		Password: "storktest",
		DbName: "storktest",
	}
	sm, err := dbsession.NewSessionMgr(dbconn);

	if err != nil {
		return errors.Wrap(err, "unable to establish connection to the session database")
	}

	// Initiate the http handler, with the objects that are implementing the business logic.
	h, err := restapi.Handler(restapi.Config{
		GeneralAPI: r,
		ServicesAPI: r,
		Logger: log.Infof,
		InnerMiddleware: loggingMiddleware,
	})
	if err != nil {
		return errors.Wrap(err, "cannot setup ReST API handler")
	}
	r.handler = h
	return nil
}


// Serve the API
func (r *RestAPI) Serve() (err error) {

	if !r.hasListeners {
		if err = r.Listen(); err != nil {
			return err
		}
	}

	// set default handler, if none is set
	if r.handler == nil {
		if r.api == nil {
			return errors.New("can't create the default handler, as no API is set")
		}

		r.handler = r.api.Serve(nil)
	}

	s := r.Settings

	httpServer := new(http.Server)
	httpServer.MaxHeaderBytes = int(s.MaxHeaderSize)
	httpServer.ReadTimeout = s.ReadTimeout
	httpServer.WriteTimeout = s.WriteTimeout
	httpServer.SetKeepAlivesEnabled(int64(s.KeepAlive) > 0)
	if s.ListenLimit > 0 {
		r.srvListener = netutil.LimitListener(r.srvListener, s.ListenLimit)
	}
	if int64(s.CleanupTimeout) > 0 {
		httpServer.IdleTimeout = s.CleanupTimeout
	}

	httpServer.Handler = r.handler

	if r.TLS {
		// Inspired by https://blog.bracebin.com/achieving-perfect-ssl-labs-score-with-go
		httpServer.TLSConfig = &tls.Config{
			// Causes servers to use Go's default ciphersuite preferences,
			// which are tuned to avoid attacks. Does nothing on clients.
			PreferServerCipherSuites: true,
			// Only use curves which have assembly implementations
			// https://github.com/golang/go/tree/master/src/crypto/elliptic
			CurvePreferences: []tls.CurveID{tls.CurveP256},
			// Use modern tls mode https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility
			NextProtos: []string{"h2", "http/1.1"},
			// https://www.owasp.org/index.php/Transport_Layer_Protection_Cheat_Sheet#Rule_-_Only_Support_Strong_Protocols
			MinVersion: tls.VersionTLS12,
			// These ciphersuites support Forward Secrecy: https://en.wikipedia.org/wiki/Forward_secrecy
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			},
		}

		// build standard config from server options
		if s.TLSCertificate != "" && s.TLSCertificateKey != "" {
			httpServer.TLSConfig.Certificates = make([]tls.Certificate, 1)
			httpServer.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(string(s.TLSCertificate), string(s.TLSCertificateKey))
			if err != nil {
				return errors.Wrap(err, "problem with setting up certificates")
			}
		}

		if s.TLSCACertificate != "" {
			// include specified CA certificate
			caCert, caCertErr := ioutil.ReadFile(string(s.TLSCACertificate))
			if caCertErr != nil {
				return errors.Wrap(caCertErr, "problem with setting up certificates")
			}
			caCertPool := x509.NewCertPool()
			ok := caCertPool.AppendCertsFromPEM(caCert)
			if !ok {
				return errors.New("cannot parse CA certificate")
			}
			httpServer.TLSConfig.ClientCAs = caCertPool
			httpServer.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		if len(httpServer.TLSConfig.Certificates) == 0 && httpServer.TLSConfig.GetCertificate == nil {
			// after standard and custom config are passed, this ends up with no certificate
			if s.TLSCertificate == "" {
				if s.TLSCertificateKey == "" {
					log.Fatalf("the required flags `--tls-certificate` and `--tls-key` were not specified")
				}
				log.Fatalf("the required flag `--tls-certificate` was not specified")
			}
			if s.TLSCertificateKey == "" {
				log.Fatalf("the required flag `--tls-key` was not specified")
			}
			// this happens with a wrong custom TLS configurator
			log.Fatalf("no certificate was configured for TLS")
		}

		// must have at least one certificate or panics
		httpServer.TLSConfig.BuildNameToCertificate()
	}

	var lstnr net.Listener
	var scheme string
	if !r.TLS {
		lstnr = r.srvListener
		scheme = "http://"
	} else {
		lstnr = tls.NewListener(r.srvListener, httpServer.TLSConfig)
		scheme = "https://"
	}

	log.WithFields(log.Fields{
		"address": scheme + lstnr.Addr().String(),
	}).Infof("Started serving Stork Server")
	if err := httpServer.Serve(lstnr); err != nil && err != http.ErrServerClosed {
		return errors.Wrap(err, "problem with serving")
	}
	log.Info("Stopped serving Stork Server")

	return nil
}

// Listen creates the listeners for the server
func (r *RestAPI) Listen() error {
	if r.hasListeners { // already done this
		return nil
	}

	s := r.Settings

	if s.TLSCertificate == "" {
		r.TLS = false
	} else{
		r.TLS = true
	}


	if !r.TLS {
		// TLS disabled
		listener, err := net.Listen("tcp", net.JoinHostPort(s.Host, strconv.Itoa(s.Port)))
		if err != nil {
			return errors.Wrap(err, "problem occurred while starting to listen using ReST API")
		}

		h, p, err := swag.SplitHostPort(listener.Addr().String())
		if err != nil {
			return errors.Wrap(err, "problem with address")
		}
		r.Host = h
		r.Port = p
		r.srvListener = listener
	} else {
		// TLS enabled

		tlsListener, err := net.Listen("tcp", net.JoinHostPort(s.Host, strconv.Itoa(s.Port)))
		if err != nil {
			return errors.Wrap(err, "problem occurred while starting to listen using ReST API")
		}

		sh, sp, err := swag.SplitHostPort(tlsListener.Addr().String())
		if err != nil {
			return errors.Wrap(err, "problem with address")
		}
		r.Host = sh
		r.Port = sp
		r.srvListener = tlsListener
	}

	r.hasListeners = true
	return nil
}

func (r *RestAPI) Shutdown() {
	// TODO
}
