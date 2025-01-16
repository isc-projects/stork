package agent

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Basic auth credentials.
type BasicAuthCredentials struct {
	User     string
	Password string
}

// Default HTTP client timeout.
const DefaultHTTPClientTimeout = 10 * time.Second

// HTTPClient is a normal http client.
type HTTPClient struct {
	client    *http.Client
	basicAuth *BasicAuthCredentials
}

// A cloner interface for HTTP client.
// The Clone() methods creates a new instance of the HTTP client with the same
// configuration as the original client. The cloned client doesn't depend on
// the original client and can be configured independently.
//
// The main motivation of creating it was allowing preparing a client with the
// general configuration that acts as a template for creating new clients that
// will be tweaked for specific purposes.
//
// For example, the first application is the HTTP client for Kea. The original
// client is created at the startup and configured by the CLI flags. Then, we
// clone this client for each detected Kea application and set the specific
// authentication credentials.
//
// The interface ensures that the base client is not accidentally used or
// modified as it allows only creating new instances.
type HTTPClientCloner interface {
	Clone() *HTTPClient
}

// Returns the reference to the http.Transport object of the underlying
// http.Client. The changes performed on the transport object will be
// reflected in the client.
func (c *HTTPClient) getTransport() *http.Transport {
	return c.client.Transport.(*http.Transport)
}

// Creates a client to contact with Kea Control Agent or named statistics-channel.
func NewHTTPClient() *HTTPClient {
	transport := &http.Transport{}
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	} else {
		// The gomock library uses own implementation of the RoundTripper.
		// It should never happen in production.
		log.Warn("Could not clone default transport, using empty")
	}

	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	// Kea only supports HTTP/1.1. By default, the client here would use HTTP/2.
	// The instance of the client which is created here disables HTTP/2 and should
	// be used whenever the communication with the Kea servers is required.
	// append the client certificates from the CA
	//
	// Creating empty, non-nil map here disables the HTTP/2.
	// In fact the not-nil TLSClientConfig disables HTTP/2 anyway but it is
	// not documented strictly.
	transport.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	return &HTTPClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   DefaultHTTPClientTimeout,
		},
	}
}

// Clones the HTTP client instance. The cloned client has the same TLS
// configuration and credentials as the original client.
func (c *HTTPClient) Clone() *HTTPClient {
	var basicAuth *BasicAuthCredentials
	if c.basicAuth != nil {
		basicAuthCopy := *c.basicAuth
		basicAuth = &basicAuthCopy
	}

	return &HTTPClient{
		client: &http.Client{
			Transport: c.getTransport().Clone(),
		},
		basicAuth: basicAuth,
	}
}

// Sets custom timeout for HTTP client requests.
func (c *HTTPClient) SetRequestTimeout(timeout time.Duration) {
	c.client.Timeout = timeout
}

// If true then it doesn't verify the server credentials
// over HTTPS. It may be useful when Kea uses a self-signed certificate.
func (c *HTTPClient) SetSkipTLSVerification(skipTLSVerification bool) {
	c.getTransport().TLSClientConfig.InsecureSkipVerify = skipTLSVerification
}

// Loads the TLS certificates from a file. The certificates will be attached
// to all sent requests.
// The GRPC certificates are self-signed by default. It means the requests
// will be rejected if the server verifies the client credentials.
// Returns true if the certificates have been loaded successfully. Returns
// false if the certificates file does not exist.
func (c *HTTPClient) LoadGRPCCertificates() (bool, error) {
	tlsCertStore := NewCertStoreDefault()
	isEmpty, err := tlsCertStore.IsEmpty()
	if err != nil {
		return false, errors.WithMessage(err, "cannot stat the TLS files")
	}
	if isEmpty {
		return false, nil
	}

	err = tlsCertStore.IsValid()
	if err != nil {
		return false, errors.WithMessage(err, "GRPC certificates are not valid")
	}

	tlsCert, err := tlsCertStore.ReadTLSCert()
	if err != nil {
		return false, errors.WithMessage(err, "cannot read the TLS certificate")
	}

	tlsRootCA, err := tlsCertStore.ReadRootCA()
	if err != nil {
		return false, errors.WithMessage(err, "cannot read the TLS root CA")
	}

	transport := c.getTransport()
	transport.TLSClientConfig.Certificates = []tls.Certificate{*tlsCert}
	transport.TLSClientConfig.RootCAs = tlsRootCA
	return true, nil
}

// Set the basic auth credentials to the client. The credentials will be
// attached to all sent requests.
func (c *HTTPClient) SetBasicAuth(user, password string) {
	c.basicAuth = &BasicAuthCredentials{
		User:     user,
		Password: password,
	}
}

// Sends a request to a given endpoint using the HTTP POST method. The payload
// must contain the valid JSON. If the authentication credentials or TLS
// certificates are provided in the application configuration, they are added
// to the request.
func (c *HTTPClient) Call(url string, payload io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, payload)
	if err != nil {
		err = errors.Wrapf(err, "problem creating POST request to %s", url)

		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	if c.basicAuth != nil {
		secret := fmt.Sprintf("%s:%s", c.basicAuth.User, c.basicAuth.Password)
		encodedSecret := base64.StdEncoding.EncodeToString([]byte(secret))
		headerContent := fmt.Sprintf("Basic %s", encodedSecret)
		req.Header.Add("Authorization", headerContent)
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "problem sending POST to %s", url)
	}
	return rsp, err
}

// Indicates if the Stork Agent attaches the authentication credentials to
// the requests.
func (c *HTTPClient) HasAuthenticationCredentials() bool {
	return c.basicAuth != nil
}
