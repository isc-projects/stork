package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// CredentialsFile path to a file holding credentials used in basic authentication of the agent in Kea.
// It is being modified by tests so needs to be writable.
var CredentialsFile = "/etc/stork/agent-credentials.json" // nolint:gochecknoglobals

// HTTPClient is a normal http client.
type HTTPClient struct {
	client      *http.Client
	credentials *CredentialsStore
}

// Create a client to contact with Kea Control Agent or named statistics-channel.
// If @skipTLSVerification is true then it doesn't verify the server credentials
// over HTTPS. It may be useful when Kea uses a self-signed certificate.
func NewHTTPClient(skipTLSVerification bool) *HTTPClient {
	// Kea only supports HTTP/1.1. By default, the client here would use HTTP/2.
	// The instance of the client which is created here disables HTTP/2 and should
	// be used whenever the communication with the Kea servers is required.
	// append the client certificates from the CA
	tlsConfig := tls.Config{
		InsecureSkipVerify: skipTLSVerification, //nolint:gosec
	}

	certPool, certificates, err := readTLSCredentials()
	if err == nil {
		tlsConfig.RootCAs = certPool
		tlsConfig.Certificates = certificates
	} else {
		log.Warnf("cannot read TLS credentials, use HTTP protocol, %+v", err)
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
			log.Infof("configured to use the Basic Auth credentials from file (%s)", CredentialsFile)
		} else {
			log.Warnf("cannot read the Basic Auth credentials from file (%s), %+v", CredentialsFile, err)
		}
	} else {
		log.Infof("the Basic Auth credentials file (%s) is missing - HTTP authentication is not used", CredentialsFile)
	}

	client := &HTTPClient{
		client:      httpClient,
		credentials: credentialsStore,
	}

	return client
}

func (c *HTTPClient) Call(url string, payload *bytes.Buffer) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, payload)
	if err != nil {
		err = errors.Wrapf(err, "problem with creating POST request to %s", url)

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
		err = errors.Wrapf(err, "problem with sending POST to %s", url)
	}
	return rsp, err
}

// TLS support - inspired by https://sirsean.medium.com/mutually-authenticated-tls-from-a-go-client-92a117e605a1
func readTLSCredentials() (*x509.CertPool, []tls.Certificate, error) {
	// Certificates
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(RootCAFile)
	if err != nil {
		err = errors.Wrapf(err, "could not read CA certificate: %s", RootCAFile)
		log.Errorf("%+v", err)
		return nil, nil, err
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		err = errors.New("failed to append client certs")
		log.Errorf("%+v", err)
		return nil, nil, err
	}

	certificate, err := tls.LoadX509KeyPair(CertPEMFile, KeyPEMFile)
	if err != nil {
		err = errors.Wrapf(err, "could not setup TLS key pair")
		log.Errorf("%+v", err)
		return nil, nil, err
	}

	return certPool, []tls.Certificate{certificate}, nil
}
