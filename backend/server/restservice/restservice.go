package restservice

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime/flagext"
	"github.com/go-openapi/swag"
	"github.com/go-pg/pg/v10"
	flags "github.com/jessevdk/go-flags"
	pkgerrors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/netutil"

	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps"
	"isc.org/stork/server/config"
	"isc.org/stork/server/configreview"
	dbops "isc.org/stork/server/database"
	dbsession "isc.org/stork/server/database/session"
	"isc.org/stork/server/dnsop"
	"isc.org/stork/server/eventcenter"
	"isc.org/stork/server/gen/restapi"
	"isc.org/stork/server/gen/restapi/operations"
	"isc.org/stork/server/hookmanager"
	"isc.org/stork/server/metrics"
	storkutil "isc.org/stork/util"
)

// The container for REST API settings. It contains the struct tags with the
// CLI flags specification.
type RestAPISettings struct {
	CleanupTimeout  time.Duration    `long:"rest-cleanup-timeout" description:"The waiting period before killing idle connections" default:"10s"`
	GracefulTimeout time.Duration    `long:"rest-graceful-timeout" description:"The waiting period before shutting down the server" default:"15s"`
	MaxHeaderSize   flagext.ByteSize `long:"rest-max-header-size" description:"Controls the maximum number of bytes the server reads when parsing the request header's keys and values, including the request line; it does not limit the size of the request body" default:"1MiB"`

	Host         string        `long:"rest-host" description:"The IP to listen on" default:"" env:"STORK_REST_HOST"`
	Port         int           `long:"rest-port" description:"The port to listen on for connections" default:"8080" env:"STORK_REST_PORT"`
	ListenLimit  int           `long:"rest-listen-limit" description:"Limits the number of outstanding requests"`
	KeepAlive    time.Duration `long:"rest-keep-alive" description:"Sets the TCP keep-alive timeouts on accepted connections; it prunes dead TCP connections ( e.g. closing laptop mid-download)" default:"3m"`
	ReadTimeout  time.Duration `long:"rest-read-timeout" description:"The maximum duration before timing out reading the request" default:"30s"`
	WriteTimeout time.Duration `long:"rest-write-timeout" description:"The maximum duration before timing out writing the response" default:"60s"`

	TLSCertificate    flags.Filename `long:"rest-tls-certificate" description:"The certificate to use for secure connections" env:"STORK_REST_TLS_CERTIFICATE"`
	TLSCertificateKey flags.Filename `long:"rest-tls-key" description:"The private key to use for secure connections" env:"STORK_REST_TLS_PRIVATE_KEY"`
	TLSCACertificate  flags.Filename `long:"rest-tls-ca" description:"The certificate authority file to be used with mutual tls auth" env:"STORK_REST_TLS_CA_CERTIFICATE"`

	StaticFilesDir string `long:"rest-static-files-dir" description:"The directory with static files for the UI" default:"" env:"STORK_REST_STATIC_FILES_DIR"`
	BaseURL        string `long:"rest-base-url" description:"The base URL of the UI. Specify this flag if the UI is served from a subdirectory (not the root URL). It must start and end with a slash. Example: https://www.example.com/admin/stork/ would need to have '/admin/stork/' as the rest-base-url" default:"/" env:"STORK_REST_BASE_URL"`
	VersionsURL    string `long:"rest-versions-url" description:"URL of the file with current Kea, Stork and BIND 9 software versions metadata" env:"STORK_REST_VERSIONS_URL" default:"https://www.isc.org/versions.json"`
}

// Runtime information and settings for RestAPI service.
type RestAPI struct {
	Settings                   *RestAPISettings
	DBSettings                 *dbops.DatabaseSettings
	DB                         *dbops.PgDB
	SessionManager             *dbsession.SessionMgr
	EventCenter                eventcenter.EventCenter
	Pullers                    *apps.Pullers
	ReviewDispatcher           configreview.Dispatcher
	MetricsCollector           metrics.Collector
	ConfigManager              config.Manager
	DHCPOptionDefinitionLookup keaconfig.DHCPOptionDefinitionLookup
	HookManager                *hookmanager.HookManager
	EndpointControl            *EndpointControl
	DNSManager                 dnsop.Manager

	Agents agentcomm.ConnectedAgents

	TLS          bool
	HTTPServer   *http.Server
	srvListener  net.Listener
	api          *operations.StorkAPI
	handler      http.Handler
	hasListeners bool
	Host         string // actual host for listening
	Port         int    // actual port for listening
}

// Instantiates RestAPI structure.
//
// This function exposes flexible interface for passing pointers to the
// structures required by the RestAPI to operate. The RestAPI struct
// comprises several pointers and several interfaces initialized by a
// caller. They are specified as variadic function arguments in no
// particular order. The function will detect their types and assign
// them to appropriate RestAPI fields.
//
// Accepted pointers:
// - *RestAPISettings,
// - *dbops.DatabaseSettings,
// - *pg.DB,
// - *apps.Pullers,
// - *EndpointControl
//
// Accepted interfaces:
// - agentcomm.ConnectedAgents,
// - configreview.Dispatcher
// - eventcenter.EventCenter,
// - metrics.Collector
// - dnsop.Manager
//
// The only mandatory parameter is the *dbops.DatabaseSettings because it
// is used to instantiate the Session Manager. Other parameters are
// optional but, if they are not specified, it may lead to nil pointer
// dereference issues upon using the RestAPI instance. Specifying a
// subset of the arguments is mostly useful in the unit tests which
// test specific RestAPI functionality.
//
// Upon adding new fields to the RestAPI, the function may be easily
// extended to support them.
func NewRestAPI(args ...interface{}) (*RestAPI, error) {
	api := &RestAPI{}

	// Iterate over the variadic arguments.
	for i, arg := range args {
		argType := reflect.TypeOf(arg)

		// If the interface is nil the TypeOf returns nil. Move to
		// the next argument.
		if argType == nil {
			continue
		}

		// Make sure that the specified argument is a pointer.
		if argType.Kind() != reflect.Ptr {
			return nil, pkgerrors.Errorf("non-pointer argument specified for NewRestAPI at position %d", i)
		}

		// If the value is nil, there is nothing to do. Move on.
		if reflect.ValueOf(arg).IsNil() {
			continue
		}

		// The underlying type must be a struct.
		if argType.Elem().Kind() != reflect.Struct {
			return nil, pkgerrors.Errorf("pointer to non-struct argument specified for NewRestAPI at position %d", i)
		}

		// Check if the specified argument is an interface.
		if argType.Implements(reflect.TypeOf((*agentcomm.ConnectedAgents)(nil)).Elem()) {
			api.Agents = arg.(agentcomm.ConnectedAgents)
			continue
		}
		if argType.Implements(reflect.TypeOf((*configreview.Dispatcher)(nil)).Elem()) {
			api.ReviewDispatcher = arg.(configreview.Dispatcher)
			continue
		}
		if argType.Implements(reflect.TypeOf((*eventcenter.EventCenter)(nil)).Elem()) {
			api.EventCenter = arg.(eventcenter.EventCenter)
			continue
		}
		if argType.Implements(reflect.TypeOf((*metrics.Collector)(nil)).Elem()) {
			api.MetricsCollector = arg.(metrics.Collector)
			continue
		}
		if argType.Implements(reflect.TypeOf((*config.Manager)(nil)).Elem()) {
			api.ConfigManager = arg.(config.Manager)
			continue
		}
		if argType.Implements(reflect.TypeOf((*keaconfig.DHCPOptionDefinitionLookup)(nil)).Elem()) {
			api.DHCPOptionDefinitionLookup = arg.(keaconfig.DHCPOptionDefinitionLookup)
			continue
		}
		if argType.Implements(reflect.TypeOf((*dnsop.Manager)(nil)).Elem()) {
			api.DNSManager = arg.(dnsop.Manager)
			continue
		}

		// Check if the specified argument is one of our supported structures.
		if argType.AssignableTo(reflect.TypeOf((*dbops.DatabaseSettings)(nil))) {
			api.DBSettings = arg.(*dbops.DatabaseSettings)
			continue
		}
		if argType.AssignableTo(reflect.TypeOf((*pg.DB)(nil))) {
			api.DB = arg.(*pg.DB)
			continue
		}
		if argType.AssignableTo(reflect.TypeOf((*apps.Pullers)(nil))) {
			api.Pullers = arg.(*apps.Pullers)
			continue
		}
		if argType.AssignableTo(reflect.TypeOf((*RestAPISettings)(nil))) {
			api.Settings = arg.(*RestAPISettings)
			continue
		}
		if argType.AssignableTo(reflect.TypeOf((*hookmanager.HookManager)(nil))) {
			api.HookManager = arg.(*hookmanager.HookManager)
			continue
		}
		if argType.AssignableTo(reflect.TypeOf((*EndpointControl)(nil))) {
			api.EndpointControl = arg.(*EndpointControl)
			continue
		}
		return nil, pkgerrors.Errorf("unknown argument type %s specified for NewRestAPI", argType.Elem().Name())
	}

	// Database settings must be specified because we need to instantiate the
	// session manager.
	if api.DBSettings == nil {
		return nil, pkgerrors.Errorf("dbops.DatabaseSettings parameter is required in NewRestAPI call")
	}

	// Instantiate the session manager.
	sm, err := dbsession.NewSessionMgr(api.DB)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "unable to establish connection to the session database")
	}
	api.SessionManager = sm

	// All ok.
	return api, nil
}

func prepareTLS(httpServer *http.Server, s *RestAPISettings) error {
	var err error

	// Inspired by https://blog.bracebin.com/achieving-perfect-ssl-labs-score-with-go
	httpServer.TLSConfig = &tls.Config{
		// Only use curves which have assembly implementations
		// https://github.com/golang/go/tree/master/src/crypto/elliptic
		CurvePreferences: []tls.CurveID{tls.CurveP256},
		// Use modern tls mode https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility
		NextProtos: []string{"h2", "http/1.1"},
		// https://www.owasp.org/index.php/Transport_Layer_Protection_Cheat_Sheet#Rule_-_Only_Support_Strong_Protocols
		MinVersion: tls.VersionTLS12,
		// These cipher suites support Forward Secrecy: https://en.wikipedia.org/wiki/Forward_secrecy
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
			return pkgerrors.Wrap(err, "problem setting up certificates")
		}
	}

	if s.TLSCACertificate != "" {
		// include specified CA certificate
		caCert, caCertErr := os.ReadFile(string(s.TLSCACertificate))
		if caCertErr != nil {
			return pkgerrors.Wrap(caCertErr, "problem setting up certificates")
		}
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return pkgerrors.New("cannot parse CA certificate")
		}
		httpServer.TLSConfig.ClientCAs = caCertPool
		httpServer.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if len(httpServer.TLSConfig.Certificates) == 0 && httpServer.TLSConfig.GetCertificate == nil {
		// after standard and custom config are passed, this ends up with no certificate
		if s.TLSCertificate == "" {
			if s.TLSCertificateKey == "" {
				log.Fatalf("The required flags `--tls-certificate` and `--tls-key` were not specified")
			}
			log.Fatalf("The required flag `--tls-certificate` was not specified")
		}
		if s.TLSCertificateKey == "" {
			log.Fatalf("The required flag `--tls-key` was not specified")
		}
		// this happens with a wrong custom TLS configuration
		log.Fatalf("No certificate was configured for TLS")
	}

	return nil
}

// Prepare a subdirectory in the static asset directory with the authentication
// icons. The icons are retrieved from registered hooks.
func prepareAuthenticationIcons(hookManager *hookmanager.HookManager, staticFilesDirectory string) error {
	iconDirectory := path.Join(staticFilesDirectory, "assets", "authentication-methods")

	var errs []error

	// Create icons for the authentication provided by hooks.
	for _, metadata := range hookManager.GetAuthenticationMetadata() {
		iconContent, err := metadata.GetIcon()
		if err != nil {
			errs = append(errs, pkgerrors.Wrapf(err, "cannot read icon for hook authentication: %s", metadata.GetID()))
			continue
		}
		defer iconContent.Close()

		iconPath := path.Join(iconDirectory, path.Clean("/"+metadata.GetID())+".png")
		iconFile, err := os.Create(iconPath)
		if err != nil {
			errs = append(errs, pkgerrors.Wrapf(err, "cannot open the icon file to write: %s", iconPath))
			continue
		}
		_, err = io.Copy(iconFile, iconContent)
		if err != nil {
			errs = append(errs, pkgerrors.Wrapf(err, "cannot write the icon file: %s", iconPath))
		}
	}

	return storkutil.CombineErrors("preparing authentication icons failed", errs)
}

// Sets up the base URL in the UI files. It modifies the <base> HTML tag value
// in the index.html file. The base URL is necessary to fetch the assets,
// scripts, and stylesheets and make calls to the API, so it must be set before
// loading the UI. It cannot be dynamically fetched from the backend. The frontend
// doesn't have its own configuration for this value. Therefore, it must be set in
// the general server configuration file or using the server flags. Before loading
// the web application, there is no way to pass the value from the backend to the
// frontend (we cannot use HTTP headers because they are available in Javascript
// only for dynamic AJAX calls; we cannot pass it as a static JSON or JS script
// because it requires a valid resource path that is unknown without a base URL;
// the environment variables are not available because the frontend runs on a user
// machine). This function alters the HTML index file and modifies the <base> HTML
// tag to set the desired path.
// If the configuration does not provide the base URL, it leaves the default
// value ('/').
// The base URL must have leading and trailing slashes.
func setBaseURLInIndexFile(baseURL, staticFilesDir string) error {
	// Leave the existing value if the base URL is empty.
	if baseURL == "" {
		return nil
	}

	// Validate base URL.
	if !strings.HasPrefix(baseURL, "/") {
		return pkgerrors.Errorf("Base URL must start with slash, got: %s", baseURL)
	}
	if !strings.HasSuffix(baseURL, "/") {
		return pkgerrors.Errorf("Base URL must end with slash, got: %s", baseURL)
	}

	// Angular builder (ng) strips the closing slash and space but I'm afraid
	// it is version or configuration specific, so I make them optional.
	baseHrefPattern := regexp.MustCompile(`<base href="(.*)"(/?\s*)>`)
	baseHrefReplacement := fmt.Sprintf(`<base href="%s"$2>`, baseURL)

	// Read the index file.
	indexFilePath := path.Join(staticFilesDir, "index.html")
	indexFileContent, err := os.ReadFile(indexFilePath)

	switch {
	case errors.Is(err, os.ErrNotExist):
		// The UI files may be located on another machine.
		log.WithError(err).Warningf(
			"Cannot read the base URL in the '%s' file because it is missing. "+
				"If the files are located on separate machine, you need "+
				"manually change the 'href' value of the <base> HTML tag to '%s'",
			indexFilePath, baseURL)
		return nil
	case errors.Is(err, os.ErrPermission):
		// The backend doesn't have the permission to operate on index.file.
		log.WithError(err).Warningf(
			"Cannot alter the base URL in the '%s' file due to insufficient "+
				"file permissions. You need to grant access to read and write "+
				"for the Stork Server user or manually change the 'href' value "+
				"of the <base> HTML tag to '%s'",
			indexFilePath, baseURL)
		return nil
	case err != nil:
		// Another error.
		return pkgerrors.Wrapf(err, "cannot read the '%s' file", indexFilePath)
	}

	// Check if the URL differs.
	matches := baseHrefPattern.FindSubmatch(indexFileContent)
	switch {
	case len(matches) == 0:
		return pkgerrors.Errorf("the base tag is missing in the '%s' file", indexFilePath)
	case len(matches) != 3:
		return pkgerrors.Errorf("the base tag is incomplete in the '%s' file", indexFilePath)
	}
	currentURL := string(matches[1])
	if currentURL == baseURL {
		// The URL is not changed.
		return nil
	}

	// Edit the index file.
	indexFileContent = baseHrefPattern.ReplaceAll(indexFileContent, []byte(baseHrefReplacement))
	err = os.WriteFile(indexFilePath, indexFileContent, 0)

	if errors.Is(err, os.ErrPermission) {
		// The backend doesn't have the permission to operate on index.file.
		log.WithError(err).Errorf(
			"Cannot write the base URL in the '%s' file due to insufficient "+
				"file permissions. You need to grant access to read and write "+
				"for the Stork Server user or manually change the 'href' value "+
				"of the <base> HTML tag to '%s'",
			indexFilePath, baseURL)
	}

	return pkgerrors.Wrapf(err, "cannot alter the '%s' file", indexFilePath)
}

// Serve the API.
func (r *RestAPI) Serve() (err error) {
	if r.Settings.StaticFilesDir == "" {
		searchPaths := []string{}
		// The typical installation path is the default.
		defaultPath := "/usr/share/stork/www"
		if executable, err := os.Executable(); err == nil {
			// If we know the executable path we can try to search other typical
			// locations relative to the executable path.
			if executable, err = filepath.EvalSymlinks(executable); err == nil {
				searchPaths = append(searchPaths, path.Join(path.Dir(executable), "..", "..", "..", "/webui/dist/stork"))
				searchPaths = append(searchPaths, path.Join(path.Dir(executable), "..", "/share/stork/www"))
			}
		}
		r.Settings.StaticFilesDir = storkutil.GetFirstExistingPathOrDefault(defaultPath, searchPaths...)
	}

	// Modify the base URL in the index file.
	if err = setBaseURLInIndexFile(r.Settings.BaseURL, r.Settings.StaticFilesDir); err != nil {
		return err
	}

	if r.Settings.VersionsURL != "" {
		log.Infof("Setting URL of the versions metadata file to %s", r.Settings.VersionsURL)
	}

	err = prepareAuthenticationIcons(r.HookManager, r.Settings.StaticFilesDir)
	if err != nil {
		// It is not a critical error.
		log.
			WithError(err).
			Warning("cannot prepare the authentication icons")
	}

	// Initiate the http handler, with the objects that are implementing the business logic.
	h, err := restapi.Handler(restapi.Config{
		GeneralAPI:      r,
		ServicesAPI:     r,
		UsersAPI:        r,
		DhcpAPI:         r,
		SettingsAPI:     r,
		SearchAPI:       r,
		EventsAPI:       r,
		DNSAPI:          r,
		Logger:          log.Infof,
		InnerMiddleware: r.InnerMiddleware,
		Authorizer:      r.Authorizer,
		AuthToken: func(token string) (interface{}, error) {
			// In normal circumstances we'd need to return some
			// user information here, but the authentication is
			// currently done in the middleware anyway, so we
			// bypass this whole mechanism anyway. Let's just
			// return the token.
			return token, nil
		},
	})
	if err != nil {
		return pkgerrors.Wrap(err, "cannot setup RESTful API handler")
	}
	r.handler = h

	if !r.hasListeners {
		if err = r.Listen(); err != nil {
			return err
		}
	}

	// set default handler, if none is set
	if r.handler == nil {
		if r.api == nil {
			return pkgerrors.New("cannot create the default handler, as no API is set")
		}

		r.handler = r.api.Serve(nil)
	}

	s := r.Settings

	httpServer := new(http.Server)
	r.HTTPServer = httpServer
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

	httpServer.Handler = r.GlobalMiddleware(r.handler, s.StaticFilesDir, s.BaseURL, r.EventCenter)

	if r.TLS {
		err = prepareTLS(httpServer, s)
		if err != nil {
			return err
		}
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
	if err := httpServer.Serve(lstnr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return pkgerrors.Wrap(err, "problem serving")
	}
	log.Info("Stopped serving Stork Server")

	return nil
}

// Listen creates the listeners for the server.
func (r *RestAPI) Listen() error {
	if r.hasListeners { // already done this
		return nil
	}

	s := r.Settings

	if s.TLSCertificate == "" {
		r.TLS = false
	} else {
		r.TLS = true
	}

	if !r.TLS {
		// TLS disabled
		listener, err := net.Listen("tcp", net.JoinHostPort(s.Host, strconv.Itoa(s.Port)))
		if err != nil {
			return pkgerrors.Wrap(err, "problem occurred while starting to listen using RESTful API")
		}

		h, p, err := swag.SplitHostPort(listener.Addr().String())
		if err != nil {
			return pkgerrors.Wrap(err, "problem with address")
		}
		r.Host = h
		r.Port = p
		r.srvListener = listener
	} else {
		// TLS enabled

		tlsListener, err := net.Listen("tcp", net.JoinHostPort(s.Host, strconv.Itoa(s.Port)))
		if err != nil {
			return pkgerrors.Wrap(err, "problem occurred while starting to listen using RESTful API")
		}

		sh, sp, err := swag.SplitHostPort(tlsListener.Addr().String())
		if err != nil {
			return pkgerrors.Wrap(err, "problem with address")
		}
		r.Host = sh
		r.Port = sp
		r.srvListener = tlsListener
	}

	r.hasListeners = true
	return nil
}

// Shutdown the HTTP handler of the REST API.
func (r *RestAPI) Shutdown() {
	log.Printf("Stopping RESTful API Service")
	if r.HTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r.HTTPServer.SetKeepAlivesEnabled(false)
		if err := r.HTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("Could not gracefully shut down the server: %v\n", err)
		}
	}
	log.Printf("Stopped RESTful API Service")

	log.Print("Stopping the session manager")
	if r.SessionManager != nil {
		r.SessionManager.Close()
	}
	log.Print("Stopped the session manager")
}
