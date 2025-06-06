package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	pkgerrors "github.com/pkg/errors"
	"isc.org/stork/appdata/bind9stats"
	pdnsdata "isc.org/stork/appdata/pdns"
	storkutil "isc.org/stork/util"
)

var _ httpResponse = (*resty.Response)(nil)

// PowerDNS API version. This is the number being a part of
// the URL path, e.g. http://localhost:8080/api/v1, where 1 is
// the API version specified here.
const pdnsAPIVersion int = 1

// Sets base path for the PowerDNS client, e.g. /api/v1.
func setPDNSClientBasePath(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + fmt.Sprintf("/api/v%d", pdnsAPIVersion)
}

// Request over PowerDNS API to a particular host and port.
// It uses a common REST client which is safe for concurrent use.
type pdnsClientRequest struct {
	innerClient *resty.Client
	baseURL     string
	apiKey      string
}

// Creates new PowerDNS request to the host and port.
func newPDNSClientRequest(innerClient *resty.Client, apiKey string, host string, port int64) *pdnsClientRequest {
	return &pdnsClientRequest{
		innerClient: innerClient,
		baseURL:     setPDNSClientBasePath(storkutil.HostWithPortURL(host, port, false)),
		apiKey:      apiKey,
	}
}

// Creates new PowerDNS request to a URL.
func newPDNSClientRequestFromURL(innerClient *resty.Client, apiKey string, url string) *pdnsClientRequest {
	return &pdnsClientRequest{
		innerClient: innerClient,
		baseURL:     setPDNSClientBasePath(url),
		apiKey:      apiKey,
	}
}

// Appends path to the base URL ensuring correct slashes.
func (request *pdnsClientRequest) makeURL(path string) string {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url := request.baseURL + path
	// Make sure there is no trailing slash.
	url = strings.TrimRight(url, "/")
	return url
}

// Makes an HTTP GET request and expects JSON payload in return. The returned
// value is unmarshalled and stored in the result. The path is the path part
// of the URL.
func (request *pdnsClientRequest) getJSON(path string, result any) (httpResponse, error) {
	url := request.makeURL(path)
	response, err := request.innerClient.R().SetHeader("X-API-Key", request.apiKey).SetResult(&result).Get(url)
	if err == nil {
		return response, nil
	}
	return nil, pkgerrors.WithStack(err)
}

// Makes an HTTP GET request and expects JSON payload in return. The returned payload
// is neither validated nor parsed. It is returned as a slice of bytes to a caller.
func (request *pdnsClientRequest) getRawJSON(path string) (httpResponse, []byte, error) {
	url := request.makeURL(path)
	response, err := request.innerClient.R().
		SetHeader("X-API-Key", request.apiKey).
		Get(url)
	if err == nil {
		return response, response.Body(), nil
	}
	return nil, nil, pkgerrors.WithStack(err)
}

// Makes a request to retrieve zones encapsulated in the artificial view (localhost)
// from the PowerDNS server.
func (request *pdnsClientRequest) getViews() (httpResponse, *bind9stats.Views, error) {
	var zones pdnsdata.Zones
	response, err := request.getJSON("/servers/localhost/zones", &zones)
	if err != nil {
		return nil, nil, err
	}
	if response.IsError() {
		return response, nil, nil
	}
	bind9Zones := []*bind9stats.Zone{}
	for zone := range zones.GetIterator() {
		bind9Zone := &bind9stats.Zone{
			ZoneName: strings.TrimSuffix(zone.Name(), "."),
			Class:    "IN",
			Serial:   zone.Serial,
			Type:     zone.Kind,
			Loaded:   time.Now(),
		}
		bind9Zones = append(bind9Zones, bind9Zone)
	}
	view := bind9stats.NewView("localhost", bind9Zones)
	views := bind9stats.NewViews([]*bind9stats.View{view})

	// Extract the views and drop other top-level information.
	return response, views, err
}

// A wrapper for the REST client. It exposes a function to create individual
// HTTP requests to selected hosts/ports.
type pdnsClient struct {
	innerClient *resty.Client
	apiKey      string
}

// Instantiates REST client for PowerDNS.
func NewPDNSClient(apiKey string) *pdnsClient {
	return &pdnsClient{
		innerClient: resty.New(),
		apiKey:      apiKey,
	}
}

// Sets custom timeout for REST client requests.
func (client *pdnsClient) SetRequestTimeout(timeout time.Duration) {
	client.innerClient.SetTimeout(timeout)
}

// Creates new request to the particular host and port.
func (client *pdnsClient) createRequest(host string, port int64) *pdnsClientRequest {
	return newPDNSClientRequest(client.innerClient, client.apiKey, host, port)
}

// Creates new request sent to the specified URL. The URL must exclude the "/api/v{n}" part.
func (client *pdnsClient) createRequestFromURL(url string) *pdnsClientRequest {
	return newPDNSClientRequestFromURL(client.innerClient, client.apiKey, url)
}

// Makes a request to retrieve zones encapsulated in the artificial view (localhost)
// from the PowerDNS server.
func (client *pdnsClient) getViews(host string, port int64) (httpResponse, *bind9stats.Views, error) {
	return client.createRequest(host, port).getViews()
}
