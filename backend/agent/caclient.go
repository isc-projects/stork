package agent

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// CredentialsFile path to a file holding credentials used in basic authentication of the agent in Kea.
// It is being modified by tests so needs to be writable.
var CredentialsFile = "/etc/stork/agent-credentials.json" //nolint:gochecknoglobals,gosec

// HTTPClient is a normal http client.
type HTTPClient struct {
	client      *http.Client
	transport   *http.Transport
	credentials *CredentialsStore
}

// Creates a client to contact with Kea Control Agent or named statistics-channel.
func NewHTTPClient() *HTTPClient {
	transport := &http.Transport{}
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	} else {
		// The gomock library uses own implementation of the RoundTripper.
		// It should never happen on production.
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
		},
		transport:   transport,
		credentials: nil,
	}
}

// If true then it doesn't verify the server credentials
// over HTTPS. It may be useful when Kea uses a self-signed certificate.
func (c *HTTPClient) SetSkipTLSVerification(skipTLSVerification bool) {
	c.transport.TLSClientConfig.InsecureSkipVerify = skipTLSVerification
}

// Loads the HTTP credentials from a file. The credentials will be used if
// necessary to authenticate the requests.
func (c *HTTPClient) LoadCredentials() error {
	credentialsStore := NewCredentialsStore()
	// Check if the credential file exist
	_, err := os.Stat(CredentialsFile)
	if errors.Is(err, os.ErrNotExist) {
		// The credentials file may not exist.
		log.Infof("The Basic Auth credentials file (%s) is missing - HTTP authentication is not used", CredentialsFile)
		return nil
	}
	if err != nil {
		// Unexpected error.
		return errors.Wrapf(err, "could not access the Basic Auth credentials file (%s)", CredentialsFile)
	}

	file, err := os.Open(CredentialsFile)
	if err != nil {
		return errors.Wrapf(err, "could not read the Basic Auth credentials from file (%s)", CredentialsFile)
	}
	defer file.Close()

	err = credentialsStore.Read(file)
	if err != nil {
		return errors.WithMessagef(err, "could not read the credentials file (%s)", CredentialsFile)
	}

	log.Infof("Configured to use the Basic Auth credentials from file (%s)", CredentialsFile)
	c.credentials = credentialsStore
	return nil
}

// Loads the TLS certificates from a file. The certificates will be attached
// to all sent requests.
// The GRPC certificates are self-signed by default. It means the requests
// will be rejected if the server verifies the client credentials.
func (c *HTTPClient) LoadGRPCCertificates() error {
	tlsCertStore := NewCertStoreDefault()
	isEmpty, err := tlsCertStore.IsEmpty()
	if err != nil {
		return errors.WithMessage(err, "cannot stat the TLS files")
	}
	if isEmpty {
		return errors.Errorf("GRPC certificates not found")
	}

	err = tlsCertStore.IsValid()
	if err != nil {
		return errors.WithMessage(err, "GRPC certificates are not valid")
	}

	tlsCert, err := tlsCertStore.ReadTLSCert()
	if err != nil {
		return errors.WithMessage(err, "cannot read the TLS certificate")
	}

	tlsRootCA, err := tlsCertStore.ReadRootCA()
	if err != nil {
		return errors.WithMessage(err, "cannot read the TLS root CA")
	}

	c.transport.TLSClientConfig.Certificates = []tls.Certificate{*tlsCert}
	c.transport.TLSClientConfig.RootCAs = tlsRootCA
	return nil
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

	if c.credentials != nil {
		if basicAuth, ok := c.credentials.GetBasicAuthByURL(url); ok {
			secret := fmt.Sprintf("%s:%s", basicAuth.User, basicAuth.Password)
			encodedSecret := base64.StdEncoding.EncodeToString([]byte(secret))
			headerContent := fmt.Sprintf("Basic %s", encodedSecret)
			req.Header.Add("Authorization", headerContent)
		}
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
	return c.credentials != nil && !c.credentials.IsEmpty()
}
