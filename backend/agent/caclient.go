package agent

import (
	"bytes"
	"crypto/tls"
	"net/http"

	"github.com/pkg/errors"
)

// CA client is a normal http Client.
// TODO: change to a more generic name, it is not only used for the Kea Control
// Agent, but also for the named statistics-channel.
type CAClient struct {
	client *http.Client
}

// Create a client to contact with Kea Control Agent or named statistics-channel.
func NewCAClient() *CAClient {
	// Kea only supports HTTP/1.1. By default, the client here would use HTTP/2.
	// The instance of the client which is created here disables HTTP/2 and should
	// be used whenever the communication with the Kea servers is required.
	httpTransport := &http.Transport{
		// Creating empty, non-nil map here disables the HTTP/2.
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}
	httpClient := &http.Client{
		Transport: httpTransport,
	}

	caClient := &CAClient{
		client: httpClient,
	}

	return caClient
}

func (c *CAClient) Call(caURL string, payload *bytes.Buffer) (*http.Response, error) {
	caRsp, err := c.client.Post(caURL, "application/json", payload)
	if err != nil {
		err = errors.Wrapf(err, "problem with sending POST to %s", caURL)
	}
	return caRsp, err
}
