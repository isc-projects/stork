package agent

import (
	"crypto/tls"
	"net/http"
)

// HTTP/1.1 client.
var httpClient11 *http.Client

// Kea only supports HTTP/1.1. By default, the client here would use HTTP/2.
// The instance of the client which is created here disables HTTP/2 and should
// be used whenever the communication with the Kea servers is required.
func init() {
	httpTransport := http.DefaultTransport.(*http.Transport)
	// Creating empty, non-nil map here disables the HTTP/2.
	httpTransport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	httpClient11 = &http.Client{
		Transport: httpTransport,
	} 
}

