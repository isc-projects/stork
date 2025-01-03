package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	pkgerrors "github.com/pkg/errors"
	"isc.org/stork/appdata/bind9stats"
	storkutil "isc.org/stork/util"
)

var _ httpResponse = (*resty.Response)(nil)

// BIND9 stats API version. This is the number being a part of
// the URL path, e.g. http://localhost:8080/json/v1, where 1 is
// the API version specified here.
const bind9StatsAPIVersion int = 1

// Sets base path for the BIND9 stats client, e.g. /json/v1.
func setBind9StatsClientBasePath(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + fmt.Sprintf("/json/v%d", bind9StatsAPIVersion)
}

// Request over BIND9 stats channel to a particular host and port.
// It uses a common REST client which is safe for concurrent use.
type bind9StatsClientRequest struct {
	innerClient *resty.Client
	baseURL     string
}

// Interface to the HTTP response exposing functions to check the response status.
// The resty.Response implements this interface.
type httpResponse interface {
	IsError() bool
	StatusCode() int
	String() string
}

// Creates new BIND9 stats request to the host and port.
func newBind9StatsClientRequest(innerClient *resty.Client, host string, port int64) *bind9StatsClientRequest {
	return &bind9StatsClientRequest{
		innerClient: innerClient,
		baseURL:     setBind9StatsClientBasePath(storkutil.HostWithPortURL(host, port, false)),
	}
}

// Creates new BIND9 stats request to a URL.
func newBind9StatsClientRequestFromURL(innerClient *resty.Client, url string) *bind9StatsClientRequest {
	return &bind9StatsClientRequest{
		innerClient: innerClient,
		baseURL:     setBind9StatsClientBasePath(url),
	}
}

// Appends path to the base URL ensuring correct slashes.
func (request *bind9StatsClientRequest) makeURL(path string) string {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url := request.baseURL + path
	// Make sure there is no trailing slash. BIND9 returns only partial answer when the
	// trailing slash is included. The partial answer only contains version information.
	url = strings.TrimRight(url, "/")
	return url
}

// Makes an HTTP GET request and expects JSON payload in return. The returned
// value is unmarshalled and stored in the result. The path is the path part
// of the URL.
func (request *bind9StatsClientRequest) getJSON(path string, result any) (httpResponse, error) {
	url := request.makeURL(path)
	response, err := request.innerClient.R().SetHeader("Accept", "application/json").SetResult(&result).Get(url)
	if err == nil {
		return response, nil
	}
	return nil, pkgerrors.WithStack(err)
}

// Makes an HTTP GET request and expects JSON payload in return. The returned payload
// is neither validated nor parsed. It is returned as a slice of bytes to a caller.
func (request *bind9StatsClientRequest) getRawJSON(path string) (httpResponse, []byte, error) {
	url := request.makeURL(path)
	response, err := request.innerClient.R().SetHeader("Accept", "application/json").Get(url)
	if err == nil {
		return response, response.Body(), nil
	}
	return nil, nil, pkgerrors.WithStack(err)
}

// Makes a request to retrieve BIND9 views over the stats channel.
func (request *bind9StatsClientRequest) getViews() (httpResponse, *bind9stats.Views, error) {
	// The /zones path returns the top level stats structure. Besides the
	// map of views it returns other top-level information. We need to embed
	// the Views field in the structure to fit the returned data. Next
	// we will extract the views map from it.
	var result struct {
		Views *bind9stats.Views
	}
	response, err := request.getJSON("/zones", &result)
	if err != nil {
		return nil, nil, err
	}
	// Extract the views and drop other top-level information.
	return response, result.Views, err
}

// Get the whole statistics structure. It should be a map[string]any
// structure wrapped in the any interface. The interface should cast
// to map[string]any.
func (request *bind9StatsClientRequest) getRawStats() (httpResponse, any, error) {
	var stats any
	response, err := request.getJSON("/", &stats)
	if err != nil {
		return nil, nil, err
	}
	return response, stats, nil
}

// A wrapper for the REST client. It exposes a function to create individual
// HTTP requests to selected hosts/ports.
type bind9StatsClient struct {
	innerClient *resty.Client
}

// Instantiates REST client for BIND9 statistics.
func NewBind9StatsClient() *bind9StatsClient {
	return &bind9StatsClient{
		innerClient: resty.New(),
	}
}

// Sets custom timeout for REST client requests.
func (client *bind9StatsClient) SetRequestTimeout(timeout time.Duration) {
	client.innerClient.SetTimeout(timeout)
}

// Creates new request to the particular host and port.
func (client *bind9StatsClient) createRequest(host string, port int64) *bind9StatsClientRequest {
	return newBind9StatsClientRequest(client.innerClient, host, port)
}

// Creates new request sent to the specified URL. The URL must exclude the "/json/v{n}" part.
func (client *bind9StatsClient) createRequestFromURL(url string) *bind9StatsClientRequest {
	return newBind9StatsClientRequestFromURL(client.innerClient, url)
}

// Makes a request to retrieve BIND9 views over the stats channel.
func (client *bind9StatsClient) getViews(host string, port int64) (httpResponse, *bind9stats.Views, error) {
	return client.createRequest(host, port).getViews()
}
