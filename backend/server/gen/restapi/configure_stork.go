// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"fmt"
	"crypto/tls"
	"net/http"

	errors "github.com/go-openapi/errors"
	runtime "github.com/go-openapi/runtime"
	middleware "github.com/go-openapi/runtime/middleware"
	strfmt "github.com/go-openapi/strfmt"

	"github.com/rs/cors"

	"isc.org/stork/server/gen/restapi/operations"
	models "isc.org/stork/server/gen/models"
)

//go:generate swagger generate server --target ../../gen --name Stork --spec ../../../swagger.yaml

func configureFlags(api *operations.StorkAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.StorkAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	api.GetVersionHandler = operations.GetVersionHandlerFunc(func(params operations.GetVersionParams) middleware.Responder {
		t := "stable"
		v := "0.1"
		d, err := strfmt.ParseDateTime("0001-01-01T00:00:00.000Z")
		if err != nil {
			fmt.Printf("problem\n")
		}
		var ver models.Version
		ver.Date = &d
		ver.Type = &t
		ver.Version = &v
		return operations.NewGetVersionOK().WithPayload(&ver)
	})

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix"
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	corsHandler := cors.New(cors.Options{
		Debug: false,
		AllowedHeaders:[]string{"*"},
		AllowedOrigins:[]string{"*"},
		AllowedMethods:[]string{},
		MaxAge:1000,
	})
	return corsHandler.Handler(handler)
}
