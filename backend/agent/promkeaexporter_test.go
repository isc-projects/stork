package agent

import (
	_ "embed"
	"encoding/json"
	"testing"

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
	pke := NewPromKeaExporter("foo", 42, true, fam)
	defer pke.Shutdown()

	require.NotNil(t, pke.HTTPServer)

	require.Equal(t, "foo", pke.Host)
	require.Equal(t, 42, pke.Port)
	require.Len(t, pke.PktStatsMap, 31)
	require.Len(t, pke.Addr4StatsMap, 12)
	require.Len(t, pke.Addr6StatsMap, 19)
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

	pke := NewPromKeaExporter("foo", 1234, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

	// Start exporter.
	pke.Start()

	// Trigger the stats collection.
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Check the collected stats.
	metric, _ := pke.Addr4StatsMap["assigned-addresses"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "7",
			"subnet_id": "7",
			"prefix":    "",
		},
	)
	require.EqualValues(t, 13.0, testutil.ToFloat64(metric))

	// check if pkt4-nak-received is 19
	metric, _ = pke.PktStatsMap["pkt4-nak-received"].Stat.GetMetricWith(prometheus.Labels{"operation": "nak"})
	require.Equal(t, 19.0, testutil.ToFloat64(metric))

	require.False(t, gock.HasUnmatchedRequest())
}

// The Kea statistic-get-all response fetched from the Kea demo containers.
//
//go:embed testdata/kea-prior-2.4.0-dhcp6-statistic-get-all-rsp.json
var kea6ResponsePrior2_4_0 []byte

//go:embed testdata/kea-2.4.0-dhcp4-statistic-get-all-rsp.json
var kea4Response2_4_0 []byte

//go:embed testdata/kea-2.4.0-dhcp6-statistic-get-all-rsp.json
var kea6Response2_4_0 []byte

// Check starting PromKeaExporter and collecting stats using the real Kea
// response returned by the Kea prior to 2.4.0 version.
func TestPromKeaExporterStartKeaPrior2_4_0(t *testing.T) {
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
		BodyString(string(kea6ResponsePrior2_4_0))

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

	pke := NewPromKeaExporter("foo", 1234, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

	// Start exporter and trigger the stats collection.
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Check the collected stats.
	metric, _ := pke.Addr6StatsMap["total-nas"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "6",
			"subnet_id": "6",
			"prefix":    "",
		},
	)
	require.EqualValues(t, 36893488147419103000., testutil.ToFloat64(metric))

	// The response is pretty big, so some metrics are available earlier than
	// others.
	metric, _ = pke.PktStatsMap["pkt6-reply-sent"].Stat.GetMetricWith(prometheus.Labels{"operation": "reply"})
	require.EqualValues(t, 4489.0, testutil.ToFloat64(metric))

	require.False(t, gock.HasUnmatchedRequest())
}

// Check starting PromKeaExporter and collecting stats using the real Kea
// response returned by the Kea DHCPv4 in 2.4.0 version.
func TestPromKeaExporterStartKea2_4_0DHCPv4(t *testing.T) {
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
		BodyString(string(kea4Response2_4_0))

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

	pke := NewPromKeaExporter("foo", 1234, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

	// Start exporter and trigger the stats collection.
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Check the collected stats.
	metric, _ := pke.Addr4StatsMap["total-addresses"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "22",
			"subnet_id": "22",
			"prefix":    "",
		},
	)
	require.EqualValues(t, 150., testutil.ToFloat64(metric))

	metric, _ = pke.PktStatsMap["pkt4-ack-received"].Stat.GetMetricWith(prometheus.Labels{"operation": "ack"})
	require.EqualValues(t, 42.0, testutil.ToFloat64(metric))

	metric, _ = pke.Addr4StatsMap["pool-total-addresses"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "11",
			"subnet_id": "11",
			"prefix":    "",
			"pool_id":   "0",
		},
	)
	require.EqualValues(t, 50., testutil.ToFloat64(metric))

	require.False(t, gock.HasUnmatchedRequest())
}

// Check starting PromKeaExporter and collecting stats using the real Kea
// response returned by the Kea DHCPv6 in 2.4.0 version.
func TestPromKeaExporterStartKea2_4_0DHCPv6(t *testing.T) {
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
		BodyString(string(kea6Response2_4_0))

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

	pke := NewPromKeaExporter("foo", 1234, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

	// Start exporter and trigger the stats collection.
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Check the collected stats.
	metric, _ := pke.Addr6StatsMap["total-nas"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "1",
			"subnet_id": "1",
			"prefix":    "",
		},
	)
	require.EqualValues(t, 844424930131968., testutil.ToFloat64(metric))

	metric, _ = pke.Addr6StatsMap["total-nas"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "1",
			"subnet_id": "1",
			"prefix":    "",
		},
	)
	require.EqualValues(t, 844424930131968., testutil.ToFloat64(metric))

	metric, _ = pke.Addr6StatsMap["total-pds"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "1",
			"subnet_id": "1",
			"prefix":    "",
		},
	)
	require.EqualValues(t, 512., testutil.ToFloat64(metric))

	// The response is pretty big, so some metrics are available earlier than
	// others.
	metric, _ = pke.PktStatsMap["pkt6-reply-sent"].Stat.GetMetricWith(prometheus.Labels{"operation": "reply"})
	require.EqualValues(t, 42.0, testutil.ToFloat64(metric))

	metric, _ = pke.Addr6StatsMap["pool-total-nas"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "1",
			"subnet_id": "1",
			"prefix":    "",
			"pool_id":   "0",
		},
	)
	require.EqualValues(t, 844424930131968., testutil.ToFloat64(metric))

	metric, _ = pke.Addr6StatsMap["pool-pd-total-pds"].GetMetricWith(
		prometheus.Labels{
			"subnet":    "1",
			"subnet_id": "1",
			"prefix":    "",
			"pool_id":   "0",
		},
	)
	require.EqualValues(t, 512., testutil.ToFloat64(metric))

	require.False(t, gock.HasUnmatchedRequest())
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

	pke := NewPromKeaExporter("foo", 1234, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

	// Act
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Assert
	metric, err := pke.Addr4StatsMap["assigned-addresses"].GetMetricWith(
		prometheus.Labels{
			"subnet_id": "7",
			"prefix":    "10.0.0.0/8",
			"subnet":    "10.0.0.0/8",
		},
	)

	require.NoError(t, err)
	require.Equal(t, 13.0, testutil.ToFloat64(metric))
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
                    "pkt4-nak-received": [ [ 19, "2019-07-30 10:04:28.386733" ] ],
					"subnet[7].pool[0].assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ],
					"subnet[7].pd-pool[0].assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ]
                }}]`)

	fam := newFakeMonitorWithDefaultsDHCPv4Only()

	// Act
	pke := NewPromKeaExporter("foo", 1234, false, fam)
	defer pke.Shutdown()
	gock.InterceptClient(fam.HTTPClient.client)
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Assert
	metric, _ := pke.PktStatsMap["pkt4-nak-received"].Stat.GetMetricWith(prometheus.Labels{"operation": "nak"})
	// Check if pkt4-nak-received has expected value.
	require.EqualValues(t, 19.0, testutil.ToFloat64(metric))

	require.Nil(t, pke.Addr4StatsMap)

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

	pke := NewPromKeaExporter("foo", 1234, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Act & Assert
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

	pke := NewPromKeaExporter("foo", 1234, true, fam)
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

	pke := NewPromKeaExporter("foo", 1234, true, fam)
	defer pke.Shutdown()

	gock.InterceptClient(fam.HTTPClient.client)

	// Act
	err := pke.collectStats()

	// Assert
	require.NoError(t, err)
	require.Contains(t, pke.ignoredStats, "foo")
}

// Test that the Describe method does nothing.
func TestDescribe(t *testing.T) {
	// Arrange
	fam := newFakeMonitorWithDefaults()
	pke := NewPromKeaExporter("foo", 1234, true, fam)
	ch := make(chan *prometheus.Desc, 1)
	defer close(ch)
	defer pke.Shutdown()

	// Act
	pke.Describe(ch)

	// Assert
	require.Empty(t, ch)
}
