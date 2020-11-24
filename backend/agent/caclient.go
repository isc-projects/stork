package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"

	"github.com/pkg/errors"
)

// HTTPClient is a normal http client.
type HTTPClient struct {
	client *http.Client
}

// Create a client to contact with Kea Control Agent or named statistics-channel.
func NewHTTPClient() *HTTPClient {
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

	client := &HTTPClient{
		client: httpClient,
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
	rsp, err := c.client.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "problem with sending POST to %s", url)
	}
	return rsp, err
}
