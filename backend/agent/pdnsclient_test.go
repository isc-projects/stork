package agent

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

//go:embed testdata/pdns-api-zones.json
var pdnsZones []byte

//go:embed testdata/pdns-api-statistics.json
var pdnsStats []byte

// Test creating base URL by appending the /api/v{n} path to the host and
// port with ensuring correct slashes.
func TestSetPDNSClientBasePath(t *testing.T) {
	require.Equal(t, fmt.Sprintf("http://example.com:8081/api/v%d", pdnsAPIVersion), setPDNSClientBasePath("http://example.com:8081/"))
	require.Equal(t, fmt.Sprintf("http://example.com:8081/api/v%d", pdnsAPIVersion), setPDNSClientBasePath("http://example.com:8081//"))
	require.Equal(t, fmt.Sprintf("http://example.com:8081/api/v%d", pdnsAPIVersion), setPDNSClientBasePath("http://example.com:8081"))
}

// Test making an URL by appending the path to the base URL.
func TestPDNSMakeURL(t *testing.T) {
	request := NewPDNSClient("stork").createRequest("localhost", 5380)
	require.NotNil(t, request)
	require.Equal(t, fmt.Sprintf("http://localhost:5380/api/v%d/servers/localhost/zones", pdnsAPIVersion), request.makeURL("servers/localhost/zones"))
	require.Equal(t, fmt.Sprintf("http://localhost:5380/api/v%d/servers/localhost/zones", pdnsAPIVersion), request.makeURL("/servers/localhost/zones"))
	require.Equal(t, fmt.Sprintf("http://localhost:5380/api/v%d/servers/localhost/zones", pdnsAPIVersion), request.makeURL("/servers/localhost/zones/"))
	require.Equal(t, fmt.Sprintf("http://localhost:5380/api/v%d", pdnsAPIVersion), request.makeURL("/"))
	require.Equal(t, fmt.Sprintf("http://localhost:5380/api/v%d", pdnsAPIVersion), request.makeURL(""))
}

// Test the GET / endpoint returning raw statistics.
func TestPDNSGetRawJSON(t *testing.T) {
	defer gock.Off()
	gock.New("http://localhost:5380/").
		Get("api/v1/servers/localhost/statistics").
		MatchHeader("X-API-Key", "stork").
		AddMatcher(func(r1 *http.Request, r2 *gock.Request) (bool, error) {
			// Require empty body
			return r1.Body == nil, nil
		}).
		Persist().
		Reply(200).
		AddHeader("Content-Type", "application/json").
		BodyString(string(pdnsStats))
	request := NewPDNSClient("stork").createRequest("localhost", 5380)
	gock.InterceptClient(request.innerClient.GetClient())

	response, rawJSON, err := request.getRawJSON("/servers/localhost/statistics")
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, http.StatusOK, response.StatusCode())
	require.NotNil(t, rawJSON)
	require.NoError(t, err)
	require.JSONEq(t, string(pdnsStats), string(rawJSON))
}

// Tests that the REST client correctly handles a non-success status code.
func TestPDNSGetRawJSON404(t *testing.T) {
	defer gock.Off()
	gock.New("http://localhost:5380/").
		Get("api/v1/servers/localhost/statistics").
		MatchHeader("X-API-Key", "stork").
		AddMatcher(func(r1 *http.Request, r2 *gock.Request) (bool, error) {
			// Require empty body
			return r1.Body == nil, nil
		}).
		Persist().
		Reply(404).
		AddHeader("Content-Type", "application/json").
		BodyString("No such URL")
	request := NewPDNSClient("stork").createRequest("localhost", 5380)
	gock.InterceptClient(request.innerClient.GetClient())

	response, rawJSON, err := request.getRawJSON("/servers/localhost/statistics")
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, http.StatusNotFound, response.StatusCode())
	require.NotNil(t, rawJSON)
	require.NoError(t, err)
	require.Equal(t, "No such URL", string(rawJSON))
}

// Tests that the REST client correctly handles errors returned while
// making a GET request.
func TestPDNSGetRawJSONError(t *testing.T) {
	defer gock.Off()
	gock.New("http://localhost:5380/").
		Get("api/v1/servers/localhost/statistics").
		MatchHeader("X-API-Key", "stork").
		AddMatcher(func(r1 *http.Request, r2 *gock.Request) (bool, error) {
			// Require empty body
			return r1.Body == nil, nil
		}).
		Persist().
		ReplyError(pkgerrors.New("error during the HTTP request"))
	request := NewBind9StatsClient().createRequest("localhost", 5380)
	gock.InterceptClient(request.innerClient.GetClient())

	response, rawJSON, err := request.getRawJSON("/servers/localhost/statistics")
	require.Error(t, err)
	require.Nil(t, response)
	require.Nil(t, rawJSON)
}

// Test the GET /zones endpoint returning a list of views and the zones.
func TestPDNSGetViews(t *testing.T) {
	defer gock.Off()
	gock.New("http://localhost:5380/").
		Get("api/v1/servers/localhost/zones").
		MatchHeader("X-API-Key", "stork").
		AddMatcher(func(r1 *http.Request, r2 *gock.Request) (bool, error) {
			// Require empty body
			return r1.Body == nil, nil
		}).
		Persist().
		Reply(200).
		AddHeader("Content-Type", "application/json").
		BodyString(string(pdnsZones))
	client := NewPDNSClient("stork")
	gock.InterceptClient(client.innerClient.GetClient())

	response, views, err := client.getViews("localhost", 5380)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, http.StatusOK, response.StatusCode())
	require.NotNil(t, views)

	viewNames := views.GetViewNames()
	require.Len(t, viewNames, 1)
	require.Contains(t, viewNames, "localhost")

	view := views.GetView("localhost")
	require.NotNil(t, view)

	zoneNames := view.GetZoneNames()
	require.Len(t, zoneNames, 10)
	require.Contains(t, zoneNames, "pdns.example.com")

	zone := view.GetZone("pdns.example.com")
	require.NotNil(t, zone)

	require.Equal(t, "pdns.example.com", zone.Name())
	require.Equal(t, "Master", zone.Type)
}

// Tests that the REST client correctly handles a non-success status code
// when listing views.
func TestPDNSGetViews404(t *testing.T) {
	defer gock.Off()
	gock.New("http://localhost:5380/").
		Get("api/v1/servers/localhost/zones").
		MatchHeader("X-API-Key", "stork").
		AddMatcher(func(r1 *http.Request, r2 *gock.Request) (bool, error) {
			// Require empty body
			return r1.Body == nil, nil
		}).
		Persist().
		Reply(404).
		AddHeader("Content-Type", "application/json").
		BodyString("No such URL")
	request := NewPDNSClient("stork").createRequest("localhost", 5380)
	gock.InterceptClient(request.innerClient.GetClient())

	response, views, err := request.getViews()
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, http.StatusNotFound, response.StatusCode())
	require.Equal(t, "No such URL", response.String())
	require.Nil(t, views)
}

// Tests that the REST client correctly handles an error while getting
// views from the PowerDNS server.
func TestPDNSGetViewsError(t *testing.T) {
	defer gock.Off()
	gock.New("http://localhost:5380/").
		Get("api/v1/servers/localhost/zones").
		MatchHeader("X-API-Key", "stork").
		AddMatcher(func(r1 *http.Request, r2 *gock.Request) (bool, error) {
			// Require empty body
			return r1.Body == nil, nil
		}).
		Persist().
		ReplyError(pkgerrors.New("error making HTTP request"))
	request := NewBind9StatsClient().createRequest("localhost", 5380)
	gock.InterceptClient(request.innerClient.GetClient())

	response, views, err := request.getViews()
	require.Error(t, err)
	require.Nil(t, response)
	require.Nil(t, views)
}

// Test that the client returns with a timeout if the server doesn't
// respond.
func TestPDNSClientTimeout(t *testing.T) {
	wgServer := &sync.WaitGroup{}
	wgServer.Add(1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow/blocked response.
		wgServer.Wait()
	}))
	defer func() {
		wgServer.Done()
		ts.Close()
	}()

	client := NewPDNSClient("stork")
	// Set very short timeout for the testing purposes.
	client.SetRequestTimeout(100 * time.Millisecond)
	var (
		err        error
		clientDone bool
		mutex      sync.RWMutex
		wgClient   sync.WaitGroup
	)
	// Ensure that the client returned before we check an error code.
	wgClient.Add(1)
	go func() {
		// Use the client to communicate with the server. This call
		// should return with a timeout because the server response
		// is blocked.
		request := client.createRequestFromURL(ts.URL)
		_, _, err = request.getRawJSON(ts.URL)
		defer func() {
			// Indicate that the client returned.
			mutex.Lock()
			defer mutex.Unlock()
			clientDone = true
		}()
		// Indicate that the client has returned so we can now check
		// an error code returned.
		wgClient.Done()
	}()
	// The timeout is 100ms. Let's wait up to 2 seconds for the timeout.
	require.Eventually(t, func() bool {
		mutex.RLock()
		defer mutex.RUnlock()
		return clientDone
	}, 2*time.Second, 100*time.Millisecond)

	// Ensure that the client has returned and we can safely access the
	// returned error.
	wgClient.Wait()
	require.NotNil(t, err)
	require.ErrorContains(t, err, "context deadline exceeded")
}
