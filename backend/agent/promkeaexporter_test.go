package agent

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	keactrl "isc.org/stork/appctrl/kea"
)

// Fake app monitor that returns some predefined list of apps.
func newFakeMonitorWithDefaults() *FakeAppMonitor {
	fam := &FakeAppMonitor{
		Apps: []App{
			&KeaApp{
				BaseApp: BaseApp{
					Type:         AppTypeKea,
					AccessPoints: makeAccessPoint(AccessPointControl, "0.1.2.3", "", 1234, false),
				},
				HTTPClient:        nil,
				ConfiguredDaemons: []string{"dhcp4", "dhcp6"},
			},
		},
	}
	return fam
}

// Check creating PromKeaExporter, check if prometheus stats are set up.
func TestNewPromKeaExporterBasic(t *testing.T) {
	fam := newFakeMonitorWithDefaults()
	httpClient := NewHTTPClient()
	pke := NewPromKeaExporter("foo", 42, 24*time.Second, true, fam, httpClient)
	defer pke.Shutdown()

	require.NotNil(t, pke.HTTPClient)
	require.NotNil(t, pke.HTTPServer)

	require.Equal(t, "foo", pke.Host)
	require.Equal(t, 42, pke.Port)
	require.Equal(t, 24*time.Second, pke.Interval)
	require.Len(t, pke.PktStatsMap, 31)
	require.Len(t, pke.Adr4StatsMap, 6)
	require.Len(t, pke.Adr6StatsMap, 9)
}

// Check starting PromKeaExporter and collecting stats.
func TestPromKeaExporterStart(t *testing.T) {
	defer gock.Off()
	gock.CleanUnmatchedRequest()
	defer gock.CleanUnmatchedRequest()
	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command":   "statistic-get-all",
			"service":   []string{"dhcp4", "dhcp6"},
			"arguments": map[string]string{},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result":0,
			"arguments": {
                    "subnet[7].assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ],
                    "pkt4-nak-received": [ [ 19, "2019-07-30 10:04:28.386733" ] ]
            }
		}]`)

	fam := newFakeMonitorWithDefaults()
	httpClient := NewHTTPClient()
	pke := NewPromKeaExporter("foo", 1234, 1*time.Second, true, fam, httpClient)
	defer pke.Shutdown()

	gock.InterceptClient(pke.HTTPClient.client)

	// start exporter
	pke.Start()
	require.NotNil(t, pke.Ticker)

	// wait 1.5 seconds that collecting is invoked at least once
	time.Sleep(1500 * time.Millisecond)

	// check if assigned-addresses is 13
	metric, _ := pke.Adr4StatsMap["assigned-addresses"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "7",
			"subnet_id": "7",
			"prefix":    "",
		},
	)
	require.Equal(t, 13.0, testutil.ToFloat64(metric))

	// check if pkt4-nak-received is 19
	metric, _ = pke.PktStatsMap["pkt4-nak-received"].Stat.GetMetricWith(prometheus.Labels{"operation": "nak"})
	require.Equal(t, 19.0, testutil.ToFloat64(metric))

	require.True(t, gock.HasUnmatchedRequest())
	unmatchedRequests := gock.GetUnmatchedRequests()
	require.Len(t, unmatchedRequests, 1)
	unmatchedRequest := unmatchedRequests[0]
	require.NotNil(t, unmatchedRequest)
	body, err := io.ReadAll(unmatchedRequest.Body)
	require.NoError(t, err)
	var request map[string]interface{}
	err = json.Unmarshal(body, &request)
	require.NoError(t, err)
	require.Contains(t, request, "command")
	require.EqualValues(t, "subnet4-list", request["command"])
}

// Test if the Kea JSON get-all-stats response is unmarshal correctly.
func TestUnmarshalKeaGetAllStatisticsResponse(t *testing.T) {
	// Arrange
	rawResponse := `
	[
		{
			"arguments": {
				"cumulative-assigned-addresses": [ [0, "2021-10-14 10:44:18.687247"] ],
				"declined-addresses": [ [0, "2021-10-14 10:44:18.687235"] ],
				"pkt4-ack-received": [ [0, "2021-10-14 10:44:18.672377"] ],
				"pkt4-ack-sent": [ [0, "2021-10-14 10:44:18.672378"] ],
				"pkt4-decline-received": [ [0, "2021-10-14 10:44:18.672379"] ],
				"pkt4-discover-received": [ [0, "2021-10-14 10:44:18.672380"] ],
				"pkt4-inform-received": [ [0, "2021-10-14 10:44:18.672380"] ],
				"pkt4-nak-received": [ [0, "2021-10-14 10:44:18.672381"] ],
				"pkt4-nak-sent": [ [0, "2021-10-14 10:44:18.672382"] ],
				"pkt4-offer-received": [ [0, "2021-10-14 10:44:18.672382"] ],
				"pkt4-offer-sent": [ [0, "2021-10-14 10:44:18.672383"] ],
				"pkt4-parse-failed": [ [0, "2021-10-14 10:44:18.672384"] ],
				"pkt4-receive-drop": [ [0, "2021-10-14 10:44:18.672389"] ],
				"pkt4-received": [ [0, "2021-10-14 10:44:18.672390"] ],
				"pkt4-release-received": [ [0, "2021-10-14 10:44:18.672390"] ],
				"pkt4-request-received": [ [0, "2021-10-14 10:44:18.672391"] ],
				"pkt4-sent": [ [0, "2021-10-14 10:44:18.672392"] ],
				"pkt4-unknown-received": [ [0, "2021-10-14 10:44:18.672392"] ],
				"reclaimed-declined-addresses": [ [0, "2021-10-14 10:44:18.687239"] ],
				"reclaimed-leases": [ [0, "2021-10-14 10:44:18.687243"] ],
				"subnet[1].assigned-addresses": [ [0, "2021-10-14 10:44:18.687253"] ],
				"subnet[1].cumulative-assigned-addresses": [ [0, "2021-10-14 10:44:18.687229"] ],
				"subnet[1].declined-addresses": [ [0, "2021-10-14 10:44:18.687266"] ],
				"subnet[1].reclaimed-declined-addresses": [ [0, "2021-10-14 10:44:18.687274"] ],
				"subnet[1].reclaimed-leases": [ [0, "2021-10-14 10:44:18.687282"] ],
				"subnet[1].total-addresses": [ [200, "2021-10-14 10:44:18.687221"] ]
			},
			"result": 0
		},
		{
			"result": 1,
			"text": "Unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline"
		}
	]`

	// Act
	var response GetAllStatisticsResponse
	err := json.Unmarshal([]byte(rawResponse), &response)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, response.Dhcp4)
	require.Nil(t, response.Dhcp6)
	require.Len(t, response.Dhcp4, 26)
	require.EqualValues(t, 200, response.Dhcp4["subnet[1].total-addresses"].Value)
	require.NotNil(t, response.Dhcp4["reclaimed-leases"].Timestamp)
	require.EqualValues(t, "2021-10-14 10:44:18.687243", *response.Dhcp4["reclaimed-leases"].Timestamp)
}

// Test if the Kea JSON subnet4-list or subnet6-list response in unmarshal correctly.
func TestUnmarshalSubnetListOKResponse(t *testing.T) {
	// Arrange
	rawResponse := `[{
		"result": 0,
		"text": "2 IPv4 subnets found",
		"arguments": {
			"subnets": [
				{
					"id": 10,
					"subnet": "10.0.0.0/8"
				},
				{
					"id": 100,
					"subnet": "192.0.2.0/24"
				}
			]
		}
	}]`

	// Act
	response := NewSubnetList()
	err := json.Unmarshal([]byte(rawResponse), &response)

	// Assert
	require.NoError(t, err)
	require.Len(t, response, 2)
	require.EqualValues(t, "192.0.2.0/24", response[100])
}

// Test the Kea JSON subnet4-list or subnet6-list response when the hook is not installed.
func TestUnmarshalSubnetListUnsupportedResponse(t *testing.T) {
	// Arrange
	rawResponse := `[
		{ 
			"result": 2,
			"text": "'subnet4-list' command not supported."
		}
	]`

	// Act
	response := NewSubnetList()
	err := json.Unmarshal([]byte(rawResponse), &response)

	// Assert
	require.NoError(t, err)
	require.Len(t, response, 0)
}

// Test the Kea JSON subnet4-list or subnet6-list response when error occurs.
func TestUnmarshalSubnetListErrorResponse(t *testing.T) {
	// Arrange
	rawResponse := ""

	// Act
	response := NewSubnetList()
	err := json.Unmarshal([]byte(rawResponse), &response)

	// Assert
	require.Error(t, err)
	require.Len(t, response, 0)
}

// Test that the Prometheus metrics use the subnet prefix if available.
func TestSubnetPrefixInPrometheusMetrics(t *testing.T) {
	// Arrange
	defer gock.Off()
	gock.New("http://0.1.2.3:1234/").
		Post("/").
		JSON(map[string]interface{}{
			"command":   "statistic-get-all",
			"service":   []string{"dhcp4", "dhcp6"},
			"arguments": map[string]interface{}{},
		}).
		Persist().
		Reply(200).
		BodyString(`[{"result":0, "arguments": {
					"cumulative-assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ],
	                "subnet[7].assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ],
	                "pkt4-nak-received": [ [ 19, "2019-07-30 10:04:28.386733" ] ]
	            }}]`)

	gock.New("http://0.1.2.3:1234/").
		Post("/").
		JSON(map[string]interface{}{
			"command":   "subnet4-list",
			"service":   []string{"dhcp4"},
			"arguments": map[string]interface{}{},
		}).
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 0,
			"text": "1 IPv4 subnets found",
			"arguments": {
				"subnets": [
					{
						"id": 7,
						"subnet": "10.0.0.0/8"
					}
				]
			}
		}]`)

	fam := newFakeMonitorWithDefaults()

	httpClient := NewHTTPClient()
	pke := NewPromKeaExporter("foo", 1234, 1*time.Second, true, fam, httpClient)
	defer pke.Shutdown()

	gock.InterceptClient(pke.HTTPClient.client)
	pke.Start()

	// Act & Assert
	// Wait for collecting.
	require.Eventually(t, func() bool {
		metric, _ := pke.Adr4StatsMap["assigned-addresses"].GetMetricWith(
			prometheus.Labels{
				"subnet_id": "7",
				"prefix":    "10.0.0.0/8",
				"subnet":    "10.0.0.0/8",
			},
		)

		return testutil.ToFloat64(metric) == 13.0
	}, 10*time.Second, 500*time.Millisecond)

	require.NotZero(t, testutil.ToFloat64(pke.Global4StatMap["cumulative-assigned-addresses"]))
}

// Fake Kea CA request sender.
type FakeKeaCASender struct {
	payload   []byte
	err       error
	callCount int64
}

// Construct the fake Kea CA sender with default response.
func newFakeKeaCASender() *FakeKeaCASender {
	defaultResponse := []subnetListJSON{
		{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
			},
			Arguments: &subnetListJSONArguments{
				Subnets: []subnetListJSONArgumentsSubnet{
					{
						ID:     1,
						Subnet: "foo",
					},
					{
						ID:     42,
						Subnet: "bar",
					},
				},
			},
		},
	}
	data, _ := json.Marshal(defaultResponse)

	return &FakeKeaCASender{data, nil, 0}
}

// Increment call counter and return fixed data.
func (s *FakeKeaCASender) sendCommandToKeaCA(ctrl *AccessPoint, request string) ([]byte, error) {
	s.callCount++
	return s.payload, s.err
}

// Test that the lazy subnet name lookup is constructed properly.
func TestNewLazySubnetPrefixLookup(t *testing.T) {
	// Arrange
	sender := newFakeKeaCASender()
	accessPoint := &AccessPoint{Address: "foo"}

	// Act
	lookup := newLazySubnetPrefixLookup(sender, accessPoint)

	// Assert
	require.NotNil(t, lookup)
	require.Zero(t, sender.callCount)
}

// Test that the subnet names are retrivied.
func TestLazySubnetNameLookupFetchesNames(t *testing.T) {
	// Arrange
	sender := newFakeKeaCASender()
	accessPoint := &AccessPoint{Address: "foo"}
	lookup := newLazySubnetPrefixLookup(sender, accessPoint)

	// Act
	name1, ok1 := lookup.getPrefix(1)
	name42, ok42 := lookup.getPrefix(42)
	name0, ok0 := lookup.getPrefix(0)
	nameMinus1, okMinus1 := lookup.getPrefix(-1)

	// Assert
	require.True(t, ok1)
	require.EqualValues(t, "foo", name1)

	require.True(t, ok42)
	require.EqualValues(t, "bar", name42)

	require.False(t, ok0)
	require.Empty(t, name0)

	require.False(t, okMinus1)
	require.Empty(t, nameMinus1)
}

// Test that subnet names are fetched only once.
func TestLazySubnetNameLookupFetchesOnlyOnce(t *testing.T) {
	// Arrange
	sender := newFakeKeaCASender()
	accessPoint := &AccessPoint{Address: "foo"}
	lookup := newLazySubnetPrefixLookup(sender, accessPoint)

	// Act
	_, _ = lookup.getPrefix(1)
	_, _ = lookup.getPrefix(1)
	_, _ = lookup.getPrefix(1)
	_, _ = lookup.getPrefix(42)
	_, _ = lookup.getPrefix(100)

	// Assert
	require.EqualValues(t, 1, sender.callCount)
}

// Test that subnet names are fetched only once even if an error occurs.
func TestLazySubnetNameLookupFetchesOnlyOnceEvenIfError(t *testing.T) {
	// Arrange
	sender := newFakeKeaCASender()
	sender.payload = nil
	sender.err = errors.New("baz")
	accessPoint := &AccessPoint{Address: "foo"}
	lookup := newLazySubnetPrefixLookup(sender, accessPoint)

	for _, subnetID := range []int{1, 1, 1, 42, 100} {
		// Act
		_, ok := lookup.getPrefix(subnetID)
		// Assert
		require.False(t, ok)
	}
	require.EqualValues(t, 1, sender.callCount)
}

// Test that subnet names are fetched again after changing the family.
func TestLazySubnetNameLookupFetchesAgainWhenFamilyChanged(t *testing.T) {
	// Arrange
	sender := newFakeKeaCASender()
	accessPoint := &AccessPoint{Address: "foo"}
	lookup := newLazySubnetPrefixLookup(sender, accessPoint)

	// Act
	_, _ = lookup.getPrefix(1)
	lookup.setFamily(6)
	sender.payload = nil
	name, ok := lookup.getPrefix(1)

	// Assert
	require.False(t, ok)
	require.Empty(t, name)
	require.EqualValues(t, 2, sender.callCount)
}

// Test that is possible to disable per-subnet stats collecting.
func TestDisablePerSubnetStatsCollecting(t *testing.T) {
	// Arrange
	defer gock.Off()
	gock.CleanUnmatchedRequest()
	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command":   "statistic-get-all",
			"service":   []string{"dhcp4", "dhcp6"},
			"arguments": map[string]string{},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{"result":0, "arguments": {
                    "subnet[7].assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ],
                    "pkt4-nak-received": [ [ 19, "2019-07-30 10:04:28.386733" ] ]
                }}]`)

	fam := newFakeMonitorWithDefaults()

	// Act
	httpClient := NewHTTPClient()
	pke := NewPromKeaExporter("foo", 1234, 1*time.Second, false, fam, httpClient)
	defer pke.Shutdown()
	gock.InterceptClient(pke.HTTPClient.client)
	pke.Start()
	// wait 1.5 seconds that collecting is invoked at least once
	time.Sleep(1500 * time.Millisecond)

	// Assert
	require.Nil(t, pke.Adr4StatsMap)

	// check if pkt4-nak-received is 19
	metric, _ := pke.PktStatsMap["pkt4-nak-received"].Stat.GetMetricWith(prometheus.Labels{"operation": "nak"})
	require.Equal(t, 19.0, testutil.ToFloat64(metric))

	// Has no unnecessary calls
	require.False(t, gock.HasUnmatchedRequest())
}

// Test that the global statistics are collected properly.
func TestCollectingGlobalStatistics(t *testing.T) {
	// Arrange
	defer gock.Off()
	gock.New("http://0.1.2.3:1234/").
		Post("/").
		JSON(map[string]interface{}{
			"command":   "statistic-get-all",
			"service":   []string{"dhcp4", "dhcp6"},
			"arguments": map[string]interface{}{},
		}).
		Persist().
		Reply(200).
		BodyString(`[{"result":0, "arguments": {
			"cumulative-assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ],
			"declined-addresses": [ [ 14, "2019-07-29 10:04:28.386740" ] ],
			"reclaimed-leases": [ [ 15, "2019-07-28 10:04:28.386740" ] ],
			"reclaimed-declined-addresses": [ [ 16, "2019-07-27 10:04:28.386740" ] ]
		}}, {"result":0, "arguments": {
			"declined-addresses": [ [ 17, "2019-07-26 10:04:28.386740" ] ],
			"cumulative-assigned-nas": [ [ 18, "2019-07-25 10:04:28.386740" ] ],
			"cumulative-assigned-pds": [ [ 19, "2019-07-24 10:04:28.386740" ] ],
			"reclaimed-leases": [ [ 20, "2019-07-23 10:04:28.386740" ] ],
			"reclaimed-declined-addresses": [ [ 21, "2019-07-22 10:04:28.386740" ] ]
		}}]`)

	fam := newFakeMonitorWithDefaults()

	httpClient := NewHTTPClient()
	pke := NewPromKeaExporter("foo", 1234, 1*time.Second, true, fam, httpClient)
	defer pke.Shutdown()

	gock.InterceptClient(pke.HTTPClient.client)
	pke.Start()

	// Act & Assert
	// Wait for collecting.
	require.Eventually(t, func() bool {
		return testutil.ToFloat64(pke.Global4StatMap["cumulative-assigned-addresses"]) > 0
	}, 2*time.Second, 500*time.Millisecond)

	require.Equal(t, 13.0, testutil.ToFloat64(pke.Global4StatMap["cumulative-assigned-addresses"]))
	require.Equal(t, 14.0, testutil.ToFloat64(pke.Global4StatMap["declined-addresses"]))
	require.Equal(t, 15.0, testutil.ToFloat64(pke.Global4StatMap["reclaimed-leases"]))
	require.Equal(t, 16.0, testutil.ToFloat64(pke.Global4StatMap["reclaimed-declined-addresses"]))

	require.Equal(t, 17.0, testutil.ToFloat64(pke.Global6StatMap["declined-addresses"]))
	require.Equal(t, 18.0, testutil.ToFloat64(pke.Global6StatMap["cumulative-assigned-nas"]))
	require.Equal(t, 19.0, testutil.ToFloat64(pke.Global6StatMap["cumulative-assigned-pds"]))
	require.Equal(t, 20.0, testutil.ToFloat64(pke.Global6StatMap["reclaimed-leases"]))
	require.Equal(t, 21.0, testutil.ToFloat64(pke.Global6StatMap["reclaimed-declined-addresses"]))
}

// Test that the Prometheus exporter sends the get-statics-all request only
// to the configured daemons.
func TestSendRequestOnlyToDetectedDaemons(t *testing.T) {
	// Arrange
	defer gock.Off()
	gock.CleanUnmatchedRequest()
	defer gock.CleanUnmatchedRequest()
	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command":   "statistic-get-all",
			"service":   []string{"dhcp6"},
			"arguments": map[string]string{},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result":0,
			"arguments": {
				"pkt6-nak-received": [ [ 19, "2019-07-30 10:04:28.386733" ] ]
            }
		}]`)

	fam := &FakeAppMonitor{}
	fam.Apps = append(fam.Apps, &KeaApp{
		BaseApp: BaseApp{
			Type:         AppTypeKea,
			AccessPoints: makeAccessPoint(AccessPointControl, "0.1.2.3", "", 1234, false),
		},
		HTTPClient: nil,
		// Reduced list of the configured daemons.
		ConfiguredDaemons: []string{"dhcp6"},
	})

	httpClient := NewHTTPClient()
	pke := NewPromKeaExporter("foo", 1234, 1*time.Second, true, fam, httpClient)
	defer pke.Shutdown()

	gock.InterceptClient(pke.HTTPClient.client)

	// Act
	err := pke.collectStats()

	// Assert
	require.NoError(t, err)
}

// Test that the encountered unsupported Kea statistics are appended to the
// ignore list. It avoids producing a lot of duplicated log entries that grow
// the log file significantly.
func TestEncounteredUnsupportedStatisticsAreAppendedToIgnoreList(t *testing.T) {
	// Arrange
	defer gock.Off()
	gock.CleanUnmatchedRequest()
	defer gock.CleanUnmatchedRequest()
	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command":   "statistic-get-all",
			"service":   []string{"dhcp4", "dhcp6"},
			"arguments": map[string]string{},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result":0,
			"arguments": {
				"foo": [ [ 19, "2019-07-30 10:04:28.386733" ] ]
            }
		}]`)

	fam := newFakeMonitorWithDefaults()

	httpClient := NewHTTPClient()
	pke := NewPromKeaExporter("foo", 1234, 1*time.Second, true, fam, httpClient)
	defer pke.Shutdown()

	gock.InterceptClient(pke.HTTPClient.client)

	// Act
	err := pke.collectStats()

	// Assert
	require.NoError(t, err)
	require.Contains(t, pke.ignoredStats, "foo")
}
