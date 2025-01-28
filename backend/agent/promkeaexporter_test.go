package agent

import (
	_ "embed"
	"encoding/json"
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
	httpClient := newHTTPClientWithDefaults()
	fam := &FakeAppMonitor{
		Apps: []App{
			&KeaApp{
				BaseApp: BaseApp{
					Type:         AppTypeKea,
					AccessPoints: makeAccessPoint(AccessPointControl, "0.1.2.3", "", 1234, false),
				},
				HTTPClient:        httpClient,
				ConfiguredDaemons: []string{"dhcp4", "dhcp6"},
				ActiveDaemons:     []string{"dhcp4", "dhcp6"},
			},
		},
		HTTPClient: httpClient,
	}
	return fam
}

// Fake app monitor that returns some predefined list of apps with only
// DHCPv4 daemon configured and active.
func newFakeMonitorWithDefaultsDHCPv4Only() *FakeAppMonitor {
	fam := newFakeMonitorWithDefaults()
	fam.Apps[0].(*KeaApp).ConfiguredDaemons = []string{"dhcp4"}
	fam.Apps[0].(*KeaApp).ActiveDaemons = []string{"dhcp4"}
	return fam
}

// Fake app monitor that returns some predefined list of apps with only
// DHCPv6 daemon configured and active.
func newFakeMonitorWithDefaultsDHCPv6Only() *FakeAppMonitor {
	fam := newFakeMonitorWithDefaults()
	fam.Apps[0].(*KeaApp).ConfiguredDaemons = []string{"dhcp6"}
	fam.Apps[0].(*KeaApp).ActiveDaemons = []string{"dhcp6"}
	return fam
}

// Check creating PromKeaExporter, check if prometheus stats are set up.
func TestNewPromKeaExporterBasic(t *testing.T) {
	fam := newFakeMonitorWithDefaults()
	pke := NewPromKeaExporter("foo", 42, 24*time.Millisecond, true, fam)
	defer pke.Shutdown()

	require.NotNil(t, pke.HTTPServer)

	require.Equal(t, "foo", pke.Host)
	require.Equal(t, 42, pke.Port)
	require.Equal(t, 24*time.Millisecond, pke.Interval)
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
			"service":   []string{"dhcp4"},
			"arguments": map[string]string{},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 0,
			"arguments": {
				"subnet[7].assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ],
				"pkt4-nak-received": [ [ 19, "2019-07-30 10:04:28.386733" ] ]
            }
		}]`)

	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command":   "subnet4-list",
			"service":   []string{"dhcp4"},
			"arguments": map[string]string{},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 3,
			"text": "Command not supported"
		}]`)

	fam := newFakeMonitorWithDefaultsDHCPv4Only()

	pke := NewPromKeaExporter("foo", 1234, 1*time.Millisecond, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

	// start exporter
	pke.Start()
	require.NotNil(t, pke.Ticker)

	// wait for collecting is invoked at least once
	require.Eventually(t, func() bool {
		metric, _ := pke.Adr4StatsMap["assigned-addresses"].GetMetricWith(
			prometheus.Labels{
				"subnet":    "7",
				"subnet_id": "7",
				"prefix":    "",
			},
		)
		return testutil.ToFloat64(metric) == 13.0
	}, 100*time.Millisecond, 5*time.Millisecond)

	// check if pkt4-nak-received is 19
	metric, _ := pke.PktStatsMap["pkt4-nak-received"].Stat.GetMetricWith(prometheus.Labels{"operation": "nak"})
	require.Equal(t, 19.0, testutil.ToFloat64(metric))

	require.False(t, gock.HasUnmatchedRequest())
}

// The Kea statistic-get-all response fetched from the Kea DHCPv6 demo container.
//
//go:embed testdata/kea-dhcp6-statistic-get-all-rsp.json
var kea6ResponseFromDemo []byte

// Check starting PromKeaExporter and collecting stats using the real Kea
// response.
func TestPromKeaExporterStartDemoResponse(t *testing.T) {
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
		BodyString(string(kea6ResponseFromDemo))

	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command":   "subnet6-list",
			"service":   []string{"dhcp6"},
			"arguments": map[string]string{},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 3,
			"text": "Command not supported"
		}]`)

	fam := newFakeMonitorWithDefaultsDHCPv6Only()

	pke := NewPromKeaExporter("foo", 1234, 5*time.Millisecond, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

	// start exporter
	pke.Start()
	require.NotNil(t, pke.Ticker)

	require.Eventually(t, func() bool {
		// check if assigned-addresses is 13
		metric, _ := pke.Adr6StatsMap["total-nas"].GetMetricWith(
			prometheus.Labels{
				"subnet":    "6",
				"subnet_id": "6",
				"prefix":    "",
			},
		)
		return testutil.ToFloat64(metric) == 36893488147419103000
	}, 500*time.Millisecond, 10*time.Millisecond)

	// The response is pretty big, so some metrics are available earlier than
	// others.
	require.Eventually(t, func() bool {
		metric, _ := pke.PktStatsMap["pkt6-reply-sent"].Stat.GetMetricWith(prometheus.Labels{"operation": "reply"})
		return testutil.ToFloat64(metric) == 4489.0
	}, 500*time.Millisecond, 10*time.Millisecond)

	require.False(t, gock.HasUnmatchedRequest())
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
	require.Len(t, response, 2)
	require.Len(t, response[0], 26)
	require.Nil(t, response[1])
	require.EqualValues(t, 200, response[0]["subnet[1].total-addresses"].Value)
	require.NotNil(t, response[0]["reclaimed-leases"].Timestamp)
	require.EqualValues(t, "2021-10-14 10:44:18.687243", *response[0]["reclaimed-leases"].Timestamp)
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
	            }}, { "result": 1, "text": "server is likely to be offline" }]`)

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
		}, { "result": 1, "text": "server is likely to be offline" }]`)

	fam := newFakeMonitorWithDefaults()

	pke := NewPromKeaExporter("foo", 1234, 1*time.Millisecond, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)
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
	}, 100*time.Millisecond, 5*time.Millisecond)

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
func (s *FakeKeaCASender) sendCommandRaw(request []byte) ([]byte, error) {
	s.callCount++
	return s.payload, s.err
}

// Test that the lazy subnet name lookup is constructed properly.
func TestNewLazySubnetPrefixLookup(t *testing.T) {
	// Arrange
	sender := newFakeKeaCASender()

	// Act
	lookup := newLazySubnetPrefixLookup(sender)

	// Assert
	require.NotNil(t, lookup)
	require.Zero(t, sender.callCount)
}

// Test that the subnet names are retrieved.
func TestLazySubnetNameLookupFetchesNames(t *testing.T) {
	// Arrange
	sender := newFakeKeaCASender()
	lookup := newLazySubnetPrefixLookup(sender)

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
	lookup := newLazySubnetPrefixLookup(sender)

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
	lookup := newLazySubnetPrefixLookup(sender)

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
	lookup := newLazySubnetPrefixLookup(sender)

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
			"service":   []string{"dhcp4"},
			"arguments": map[string]string{},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{"result":0, "arguments": {
                    "subnet[7].assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ],
                    "pkt4-nak-received": [ [ 19, "2019-07-30 10:04:28.386733" ] ]
                }}]`)

	fam := newFakeMonitorWithDefaultsDHCPv4Only()

	// Act
	pke := NewPromKeaExporter("foo", 1234, 1*time.Millisecond, false, fam)
	defer pke.Shutdown()
	gock.InterceptClient(fam.HTTPClient.client)
	pke.Start()

	// Assert
	// Wait for collecting.
	require.Eventually(t, func() bool {
		metric, _ := pke.PktStatsMap["pkt4-nak-received"].Stat.GetMetricWith(prometheus.Labels{"operation": "nak"})
		// Check if pkt4-nak-received has expected value.
		return testutil.ToFloat64(metric) == 19.0
	}, 100*time.Millisecond, 5*time.Millisecond)

	require.Nil(t, pke.Adr4StatsMap)

	// Has no unnecessary calls.
	require.False(t, gock.HasUnmatchedRequest())
}

// Test that the global statistics are collected properly.
func TestCollectingGlobalStatistics(t *testing.T) {
	// Arrange
	defer gock.Off()
	gock.CleanUnmatchedRequest()
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

	pke := NewPromKeaExporter("foo", 1234, 1*time.Millisecond, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)
	pke.Start()

	// Act & Assert
	// Wait for collecting.
	require.Eventually(t, func() bool {
		return testutil.ToFloat64(pke.Global4StatMap["cumulative-assigned-addresses"]) > 0
	}, 100*time.Millisecond, 5*time.Millisecond)

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

	fam := newFakeMonitorWithDefaults()
	fam.Apps[0].(*KeaApp).ConfiguredDaemons = []string{"dhcp6"}
	fam.Apps[0].(*KeaApp).ActiveDaemons = []string{"dhcp6"}

	pke := NewPromKeaExporter("foo", 1234, 1*time.Millisecond, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

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
		}, { "result": 1, "text": "server is likely to be offline" }]`)

	fam := newFakeMonitorWithDefaults()

	pke := NewPromKeaExporter("foo", 1234, 1*time.Millisecond, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

	// Act
	err := pke.collectStats()

	// Assert
	require.NoError(t, err)
	require.Contains(t, pke.ignoredStats, "foo")
}
