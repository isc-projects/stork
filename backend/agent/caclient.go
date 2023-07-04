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
	storkutil "isc.org/stork/util"
)

// CredentialsFile path to a file holding credentials used in basic authentication of the agent in Kea.
// It is being modified by tests so needs to be writable.
var CredentialsFile = "/etc/stork/agent-credentials.json" //nolint:gochecknoglobals,gosec

// HTTPClient is a normal http client.
type HTTPClient struct {
	client      *http.Client
	credentials *CredentialsStore
}

// Create a client to contact with Kea Control Agent or named statistics-channel.
// If @skipTLSVerification is true then it doesn't verify the server credentials
// over HTTPS. It may be useful when Kea uses a self-signed certificate.
func NewHTTPClient(certStore *CertStore, skipTLSVerification bool) *HTTPClient {
	// Kea only supports HTTP/1.1. By default, the client here would use HTTP/2.
	// The instance of the client which is created here disables HTTP/2 and should
	// be used whenever the communication with the Kea servers is required.
	// append the client certificates from the CA
	tlsConfig := tls.Config{
		InsecureSkipVerify: skipTLSVerification, //nolint:gosec
	}

	certPool, err1 := certStore.GetRootCA()
	certificate, err2 := certStore.GetTLSCert()
	if err1 == nil && err2 == nil {
		tlsConfig.RootCAs = certPool
		tlsConfig.Certificates = []tls.Certificate{*certificate}
	} else {
		err := storkutil.CombineErrors("cannot read certificates", []error{err1, err2})
		log.WithError(err).Warnf("Cannot read TLS credentials, use HTTP protocol")
	}

	httpTransport := &http.Transport{
		// Creating empty, non-nil map here disables the HTTP/2.
		TLSNextProto:    make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		TLSClientConfig: &tlsConfig,
	}

	httpClient := &http.Client{
		Transport: httpTransport,
	}

	credentialsStore := NewCredentialsStore()
	// Check if the credential file exist
	if _, err := os.Stat(CredentialsFile); err == nil {
		file, err := os.Open(CredentialsFile)
		if err == nil {
			defer file.Close()
			err = credentialsStore.Read(file)
			err = errors.WithMessagef(err, "cannot read the credentials file (%s)", CredentialsFile)
		}
		if err == nil {
			log.Infof("Configured to use the Basic Auth credentials from file (%s)", CredentialsFile)
		} else {
			log.Warnf("Cannot read the Basic Auth credentials from file (%s), %+v", CredentialsFile, err)
		}
	} else {
		log.Infof("The Basic Auth credentials file (%s) is missing - HTTP authentication is not used", CredentialsFile)
	}

	client := &HTTPClient{
		client:      httpClient,
		credentials: credentialsStore,
	}

	return client
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

	if basicAuth, ok := c.credentials.GetBasicAuthByURL(url); ok {
		secret := fmt.Sprintf("%s:%s", basicAuth.User, basicAuth.Password)
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
	return !c.credentials.IsEmpty()
}
