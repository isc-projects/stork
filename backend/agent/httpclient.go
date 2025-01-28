package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Basic auth credentials.
type basicAuthCredentials struct {
	User     string
	Password string
}

// Default HTTP client timeout.
const defaultHTTPClientTimeout = 10 * time.Second

// Indicates if the credentials are empty.
func (b basicAuthCredentials) IsZero() bool {
	return b.User == "" && b.Password == ""
}

// HTTP client configuration.
type HTTPClientConfig struct {
	// SkipTLSVerification indicates if the client should skip the verification
	// of the server credentials over HTTPS.
	SkipTLSVerification bool
	// Basic auth credentials. Nil if the client should not attach the
	// credentials to the requests.
	BasicAuth basicAuthCredentials
	TLSCert   *tls.Certificate
	TLSRootCA *x509.CertPool
	Timeout   time.Duration
}

// Loads the TLS certificates from a file. The certificates will be attached
// to all sent requests.
// The GRPC certificates are self-signed by default. It means the requests
// will be rejected if the server verifies the client credentials.
// Returns true if the certificates have been loaded successfully. Returns
// false if the certificates file does not exist.
func (c *HTTPClientConfig) LoadGRPCCertificates() (bool, error) {
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

	c.TLSCert = tlsCert
	c.TLSRootCA = tlsRootCA
	return true, nil
}

// httpClient is a normal http client.
type httpClient struct {
	client    *http.Client
	basicAuth basicAuthCredentials
}

// Returns the reference to the http.Transport object of the underlying
// http.Client. The changes performed on the transport object will be
// reflected in the client.
func (c *httpClient) getTransport() *http.Transport {
	return c.client.Transport.(*http.Transport)
}

// Creates a client to contact with Kea Control Agent or named statistics-channel.
func NewHTTPClient(config HTTPClientConfig) *httpClient {
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
		//nolint:gosec // It may be insecure, but it is required in some cases.
		InsecureSkipVerify: config.SkipTLSVerification,
		RootCAs:            config.TLSRootCA,
	}

	if config.TLSCert != nil {
		transport.TLSClientConfig.Certificates = []tls.Certificate{*config.TLSCert}
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

	timeout := config.Timeout
	if timeout == 0 {
		timeout = defaultHTTPClientTimeout
	}

	return &httpClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		basicAuth: config.BasicAuth,
	}
}

// Sends a request to a given endpoint using the HTTP POST method. The payload
// must contain the valid JSON. If the authentication credentials or TLS
// certificates are provided in the application configuration, they are added
// to the request.
func (c *httpClient) Call(url string, payload io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, payload)
	if err != nil {
		err = errors.Wrapf(err, "problem creating POST request to %s", url)

		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	if !c.basicAuth.IsZero() {
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
func (c *httpClient) HasAuthenticationCredentials() bool {
	return !c.basicAuth.IsZero()
}
