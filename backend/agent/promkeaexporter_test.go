package agent

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
)

// Fake daemon monitor that returns some predefined list of daemons.
func newFakeMonitorWithDefaults(interceptor func(client *http.Client)) *FakeMonitor {
	fdm := &FakeMonitor{
		Daemons: []Daemon{
			&keaDaemon{
				daemon: daemon{
					Name: daemonname.DHCPv4,
					AccessPoints: []AccessPoint{{
						Type:     AccessPointControl,
						Address:  "0.1.2.3",
						Port:     1234,
						Protocol: protocoltype.HTTP,
					}},
				},
				connector: newKeaConnector(AccessPoint{
					Type:     AccessPointControl,
					Address:  "0.1.2.3",
					Port:     1234,
					Protocol: protocoltype.HTTP,
				}, HTTPClientConfig{Interceptor: interceptor}),
			},
			&keaDaemon{
				daemon: daemon{
					Name: daemonname.DHCPv6,
					AccessPoints: []AccessPoint{{
						Type:     AccessPointControl,
						Address:  "0.1.2.3",
						Port:     1234,
						Protocol: protocoltype.HTTP,
					}},
				},
				connector: newKeaConnector(AccessPoint{
					Type:     AccessPointControl,
					Address:  "0.1.2.3",
					Port:     1234,
					Protocol: protocoltype.HTTP,
				}, HTTPClientConfig{Interceptor: interceptor}),
			},
		},
	}

	for i := range fdm.Daemons {
		gock.InterceptClient(fdm.Daemons[i].(*keaDaemon).connector.(*keaHTTPConnector).httpClient.client)
	}

	return fdm
}

// Fake daemon monitor that returns some predefined list of daemons with only
// DHCPv4 daemon configured and active.
func newFakeMonitorWithDefaultsDHCPv4Only(interceptor func(client *http.Client)) *FakeMonitor {
	fdm := newFakeMonitorWithDefaults(interceptor)
	// Keep only the DHCPv4 daemon
	fdm.Daemons = fdm.Daemons[:1]
	return fdm
}

// Fake daemon monitor that returns some predefined list of daemons with only
// DHCPv6 daemon configured and active.
func newFakeMonitorWithDefaultsDHCPv6Only(interceptor func(client *http.Client)) *FakeMonitor {
	fdm := newFakeMonitorWithDefaults(interceptor)
	// Keep only the DHCPv6 daemon
	fdm.Daemons = fdm.Daemons[1:]
	return fdm
}

// Check creating PromKeaExporter, check if prometheus stats are set up.
func TestNewPromKeaExporterBasic(t *testing.T) {
	fam := newFakeMonitorWithDefaults(nil)
	pke := NewPromKeaExporter(t.Context(), "foo", 42, true, fam)
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
			"command": "statistic-get-all",
			"service": []string{"dhcp4"},
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
			"command": "subnet4-list",
			"service": []string{"dhcp4"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 3,
			"text": "Command not supported"
		}]`)

	fdm := newFakeMonitorWithDefaultsDHCPv4Only(gock.InterceptClient)

	pke := NewPromKeaExporter(t.Context(), "foo", 1234, true, fdm)
	defer pke.Shutdown()

	// Start exporter.
	pke.Start()

	// Trigger the stats collection.
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Check the collected stats.
	metric, err := pke.Addr4StatsMap["assigned-addresses"].GetMetricWith(
		prometheus.Labels{
			"subnet":         "7",
			"subnet_id":      "7",
			"prefix":         "",
			"shared_network": "",
		},
	)
	require.NoError(t, err)
	require.EqualValues(t, 13.0, testutil.ToFloat64(metric))

	// check if pkt4-nak-received is 19
	metric, err = pke.PktStatsMap["pkt4-nak-received"].Stat.GetMetricWith(prometheus.Labels{"operation": "nak"})
	require.NoError(t, err)
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
			"command": "statistic-get-all",
			"service": []string{"dhcp6"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(string(kea6ResponsePrior2_4_0))

	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command": "subnet6-list",
			"service": []string{"dhcp6"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 3,
			"text": "Command not supported"
		}]`)

	fdm := newFakeMonitorWithDefaultsDHCPv6Only(gock.InterceptClient)

	pke := NewPromKeaExporter(t.Context(), "foo", 1234, true, fdm)
	defer pke.Shutdown()

	// Start exporter and trigger the stats collection.
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Check the collected stats.
	metric, _ := pke.Addr6StatsMap["total-nas"].GetMetricWith(
		prometheus.Labels{
			"subnet":         "6",
			"subnet_id":      "6",
			"prefix":         "",
			"shared_network": "",
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
			"command": "statistic-get-all",
			"service": []string{"dhcp4"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(string(kea4Response2_4_0))

	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command": "subnet4-list",
			"service": []string{"dhcp4"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 3,
			"text": "Command not supported"
		}]`)

	fdm := newFakeMonitorWithDefaultsDHCPv4Only(gock.InterceptClient)

	pke := NewPromKeaExporter(t.Context(), "foo", 1234, true, fdm)
	defer pke.Shutdown()

	// Start exporter and trigger the stats collection.
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Check the collected stats.
	metric, _ := pke.Addr4StatsMap["total-addresses"].GetMetricWith(
		prometheus.Labels{
			"subnet":         "22",
			"subnet_id":      "22",
			"prefix":         "",
			"shared_network": "",
		},
	)
	require.EqualValues(t, 150., testutil.ToFloat64(metric))

	metric, _ = pke.PktStatsMap["pkt4-ack-received"].Stat.GetMetricWith(prometheus.Labels{"operation": "ack"})
	require.EqualValues(t, 42.0, testutil.ToFloat64(metric))

	metric, _ = pke.Addr4StatsMap["pool-total-addresses"].GetMetricWith(
		prometheus.Labels{
			"subnet":         "11",
			"subnet_id":      "11",
			"prefix":         "",
			"pool_id":        "0",
			"shared_network": "",
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
			"command": "statistic-get-all",
			"service": []string{"dhcp6"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(string(kea6Response2_4_0))

	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command": "subnet6-list",
			"service": []string{"dhcp6"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 3,
			"text": "Command not supported"
		}]`)

	fdm := newFakeMonitorWithDefaultsDHCPv6Only(gock.InterceptClient)

	pke := NewPromKeaExporter(t.Context(), "foo", 1234, true, fdm)
	defer pke.Shutdown()

	// Start exporter and trigger the stats collection.
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Check the collected stats.
	metric, _ := pke.Addr6StatsMap["total-nas"].GetMetricWith(
		prometheus.Labels{
			"subnet":         "1",
			"subnet_id":      "1",
			"prefix":         "",
			"shared_network": "",
		},
	)
	require.EqualValues(t, 844424930131968., testutil.ToFloat64(metric))

	metric, _ = pke.Addr6StatsMap["total-nas"].GetMetricWith(
		prometheus.Labels{
			"subnet":         "1",
			"subnet_id":      "1",
			"prefix":         "",
			"shared_network": "",
		},
	)
	require.EqualValues(t, 844424930131968., testutil.ToFloat64(metric))

	metric, _ = pke.Addr6StatsMap["total-pds"].GetMetricWith(
		prometheus.Labels{
			"subnet":         "1",
			"subnet_id":      "1",
			"prefix":         "",
			"shared_network": "",
		},
	)
	require.EqualValues(t, 512., testutil.ToFloat64(metric))

	// The response is pretty big, so some metrics are available earlier than
	// others.
	metric, _ = pke.PktStatsMap["pkt6-reply-sent"].Stat.GetMetricWith(prometheus.Labels{"operation": "reply"})
	require.EqualValues(t, 42.0, testutil.ToFloat64(metric))

	metric, _ = pke.Addr6StatsMap["pool-total-nas"].GetMetricWith(
		prometheus.Labels{
			"subnet":         "1",
			"subnet_id":      "1",
			"prefix":         "",
			"pool_id":        "0",
			"shared_network": "",
		},
	)
	require.EqualValues(t, 844424930131968., testutil.ToFloat64(metric))

	metric, _ = pke.Addr6StatsMap["pool-pd-total-pds"].GetMetricWith(
		prometheus.Labels{
			"subnet":         "1",
			"subnet_id":      "1",
			"prefix":         "",
			"pool_id":        "0",
			"shared_network": "",
		},
	)
	require.EqualValues(t, 512., testutil.ToFloat64(metric))

	require.False(t, gock.HasUnmatchedRequest())
}

// Test if the Kea JSON subnet4-list or subnet6-list response in unmarshal
// correctly. Uses the response from Kea prior to 2.7.8 version.
func TestUnmarshalSubnetListOKResponsePriorKea2_7_8(t *testing.T) {
	// Arrange
	rawResponse := `{
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
	}`

	// Act
	var response subnetListJSON
	err := json.Unmarshal([]byte(rawResponse), &response)

	// Assert
	require.NoError(t, err)
	idx := slices.IndexFunc(response.Arguments.Subnets, func(item subnetListJSONArgumentsSubnet) bool {
		return item.ID == 100
	})
	require.GreaterOrEqual(t, idx, 0)
	subnet := response.Arguments.Subnets[idx]
	require.EqualValues(t, "192.0.2.0/24", subnet.Subnet)
	require.Empty(t, subnet.SharedNetworkName)
}

// Test if the Kea JSON subnet4-list or subnet6-list response in unmarshal
// correctly. Uses the response from Kea 2.7.8 version.
func TestUnmarshalSubnetListOKResponseKea2_7_8(t *testing.T) {
	// Arrange
	rawResponse := `{
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
					"subnet": "192.0.2.0/24",
					"shared-network-name": "foo"
				}
			]
		}
	}`

	// Act
	var response subnetListJSON
	err := json.Unmarshal([]byte(rawResponse), &response)

	// Assert
	require.NoError(t, err)
	require.Len(t, response.Arguments.Subnets, 2)
	idx := slices.IndexFunc(response.Arguments.Subnets, func(item subnetListJSONArgumentsSubnet) bool {
		return item.ID == 100
	})
	require.GreaterOrEqual(t, idx, 0)
	subnet := response.Arguments.Subnets[idx]
	require.EqualValues(t, "192.0.2.0/24", subnet.Subnet)
	require.EqualValues(t, "foo", subnet.SharedNetworkName)
}

// Test the Kea JSON subnet4-list or subnet6-list response when the hook is not installed.
func TestUnmarshalSubnetListUnsupportedResponse(t *testing.T) {
	// Arrange
	rawResponse := `
		{
			"result": 2,
			"text": "'subnet4-list' command not supported."
		}
	`

	// Act
	var response subnetListJSON
	err := json.Unmarshal([]byte(rawResponse), &response)

	// Assert
	require.NoError(t, err)
	require.Len(t, response.Arguments.Subnets, 0)
}

// Test the Kea JSON subnet4-list or subnet6-list response when error occurs.
func TestUnmarshalSubnetListErrorResponse(t *testing.T) {
	// Arrange
	rawResponse := ""

	// Act
	var response subnetListJSON
	err := json.Unmarshal([]byte(rawResponse), &response)

	// Assert
	require.Error(t, err)
	require.Len(t, response.Arguments.Subnets, 0)
}

// Test that the Prometheus metrics use the subnet prefix if available.
func TestSubnetPrefixInPrometheusMetrics(t *testing.T) {
	// Arrange
	defer gock.Off()
	gock.New("http://0.1.2.3:1234/").
		Post("/").
		JSON(map[string]interface{}{
			"command": "statistic-get-all",
			"service": []string{"dhcp4"},
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
			"command": "subnet4-list",
			"service": []string{"dhcp4"},
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
						"subnet": "10.0.0.0/8",
						"shared-network-name": "foobar"
					}
				]
			}
		}]`)

	fam := newFakeMonitorWithDefaults(gock.InterceptClient)

	pke := NewPromKeaExporter(t.Context(), "foo", 1234, true, fam)
	defer pke.Shutdown()

	// Act
	pke.Start()
	c := make(chan prometheus.Metric)
	pke.Collect(c)

	// Assert
	metric, err := pke.Addr4StatsMap["assigned-addresses"].GetMetricWith(
		prometheus.Labels{
			"subnet_id":      "7",
			"prefix":         "10.0.0.0/8",
			"subnet":         "10.0.0.0/8",
			"shared_network": "foobar",
		},
	)

	require.NoError(t, err)
	require.Equal(t, 13.0, testutil.ToFloat64(metric))
	require.NotZero(t, testutil.ToFloat64(pke.Global4StatMap["cumulative-assigned-addresses"]))
}

// Fake Kea request sender.
type fakeKeaSender struct {
	payload   []byte
	err       error
	callCount int64
}

// Construct the fake Kea sender with default response.
func newFakeKeaSender() *fakeKeaSender {
	defaultResponse := subnetListJSON{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
		},
		Arguments: subnetListJSONArguments{
			Subnets: []subnetListJSONArgumentsSubnet{
				{
					ID:     1,
					Subnet: "foo",
				},
				{
					ID:                42,
					Subnet:            "bar",
					SharedNetworkName: "baz",
				},
			},
		},
	}

	data, _ := json.Marshal(defaultResponse)

	return &fakeKeaSender{data, nil, 0}
}

// Increment call counter and return fixed data.
func (s *fakeKeaSender) sendCommand(ctx context.Context, command keactrl.SerializableCommand, response any) error {
	s.callCount++
	if s.err != nil {
		return s.err
	}
	err := json.Unmarshal(s.payload, response)
	return errors.Wrap(err, "unmarshaling fake Kea response")
}

// Test that the lazy subnet lookup is constructed properly.
func TestNewLazySubnetLookup(t *testing.T) {
	// Arrange
	sender := newFakeKeaSender()

	// Act
	lookup := newLazySubnetLookup(t.Context(), sender)

	// Assert
	require.NotNil(t, lookup)
	require.Zero(t, sender.callCount)
}

// Test that the subnet data are retrieved.
func TestLazySubnetLookupFetchesData(t *testing.T) {
	// Arrange
	sender := newFakeKeaSender()
	lookup := newLazySubnetLookup(t.Context(), sender)

	// Act
	info1, ok1 := lookup.getSubnetInfo(1)
	info42, ok42 := lookup.getSubnetInfo(42)
	info0, ok0 := lookup.getSubnetInfo(0)
	infoMinus1, okMinus1 := lookup.getSubnetInfo(-1)

	// Assert
	require.True(t, ok1)
	require.EqualValues(t, "foo", info1.prefix)
	require.Empty(t, info1.sharedNetwork)

	require.True(t, ok42)
	require.EqualValues(t, "bar", info42.prefix)
	require.EqualValues(t, "baz", info42.sharedNetwork)

	require.False(t, ok0)
	require.Empty(t, info0.prefix)

	require.False(t, okMinus1)
	require.Empty(t, infoMinus1.prefix)
}

// Test that subnets are fetched only once.
func TestLazySubnetLookupFetchesOnlyOnce(t *testing.T) {
	// Arrange
	sender := newFakeKeaSender()
	lookup := newLazySubnetLookup(t.Context(), sender)

	// Act
	_, _ = lookup.getSubnetInfo(1)
	_, _ = lookup.getSubnetInfo(1)
	_, _ = lookup.getSubnetInfo(1)
	_, _ = lookup.getSubnetInfo(42)
	_, _ = lookup.getSubnetInfo(100)

	// Assert
	require.EqualValues(t, 1, sender.callCount)
}

// Test that subnets are fetched only once even if an error occurs.
func TestLazySubnetLookupFetchesOnlyOnceEvenIfError(t *testing.T) {
	// Arrange
	sender := newFakeKeaSender()
	sender.payload = nil
	sender.err = errors.New("baz")
	lookup := newLazySubnetLookup(t.Context(), sender)

	for _, subnetID := range []int{1, 1, 1, 42, 100} {
		// Act
		_, ok := lookup.getSubnetInfo(subnetID)
		// Assert
		require.False(t, ok)
	}
	require.EqualValues(t, 1, sender.callCount)
}

// Test that subnets are fetched again after changing the family.
func TestLazySubnetLookupFetchesAgainWhenFamilyChanged(t *testing.T) {
	// Arrange
	sender := newFakeKeaSender()
	lookup := newLazySubnetLookup(t.Context(), sender)

	// Act
	_, _ = lookup.getSubnetInfo(1)
	lookup.setFamily(6)
	sender.payload = nil
	info, ok := lookup.getSubnetInfo(1)

	// Assert
	require.False(t, ok)
	require.Empty(t, info.prefix)
	require.EqualValues(t, 2, sender.callCount)
}

// Test that is possible to disable per-subnet stats collecting.
func TestDisablePerSubnetStatsCollecting(t *testing.T) {
	// Arrange
	defer gock.Off()
	gock.CleanUnmatchedRequest()
	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command": "statistic-get-all",
			"service": []string{"dhcp4"},
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

	fam := newFakeMonitorWithDefaultsDHCPv4Only(gock.InterceptClient)

	// Act
	pke := NewPromKeaExporter(t.Context(), "foo", 1234, false, fam)
	defer pke.Shutdown()
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
			"command": "statistic-get-all",
			"service": []string{"dhcp4"},
		}).
		Persist().
		Reply(200).
		BodyString(`[{"result":0, "arguments": {
			"cumulative-assigned-addresses": [ [ 13, "2019-07-30 10:04:28.386740" ] ],
			"declined-addresses": [ [ 14, "2019-07-29 10:04:28.386740" ] ],
			"reclaimed-leases": [ [ 15, "2019-07-28 10:04:28.386740" ] ],
			"reclaimed-declined-addresses": [ [ 16, "2019-07-27 10:04:28.386740" ] ]
		}}]`)
	gock.New("http://0.1.2.3:1234/").
		Post("/").
		JSON(map[string]interface{}{
			"command": "statistic-get-all",
			"service": []string{"dhcp6"},
		}).
		Persist().
		Reply(200).
		BodyString(`[{"result":0, "arguments": {
			"declined-addresses": [ [ 17, "2019-07-26 10:04:28.386740" ] ],
			"cumulative-assigned-nas": [ [ 18, "2019-07-25 10:04:28.386740" ] ],
			"cumulative-assigned-pds": [ [ 19, "2019-07-24 10:04:28.386740" ] ],
			"reclaimed-leases": [ [ 20, "2019-07-23 10:04:28.386740" ] ],
			"reclaimed-declined-addresses": [ [ 21, "2019-07-22 10:04:28.386740" ] ]
		}}]`)

	fam := newFakeMonitorWithDefaults(gock.InterceptClient)

	pke := NewPromKeaExporter(t.Context(), "foo", 1234, true, fam)
	defer pke.Shutdown()

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
			"command": "statistic-get-all",
			"service": []string{"dhcp4"},
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

	gock.New("http://0.1.2.3:1234/").
		JSON(map[string]interface{}{
			"command": "statistic-get-all",
			"service": []string{"dhcp6"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result":0,
			"arguments": {
				"bar": [ [ 19, "2019-07-30 10:04:28.386733" ] ]
            }
		}]`)

	fam := newFakeMonitorWithDefaults(gock.InterceptClient)

	pke := NewPromKeaExporter(t.Context(), "my-host", 1234, true, fam)
	defer pke.Shutdown()

	// Act
	err := pke.collectStats()

	// Assert
	require.NoError(t, err)
	require.Contains(t, pke.ignoredStats, "foo")
	require.Contains(t, pke.ignoredStats, "bar")
}

// Test that the Describe method does nothing.
func TestDescribe(t *testing.T) {
	// Arrange
	fam := newFakeMonitorWithDefaults(nil)
	pke := NewPromKeaExporter(t.Context(), "foo", 1234, true, fam)
	ch := make(chan *prometheus.Desc, 1)
	defer close(ch)
	defer pke.Shutdown()

	// Act
	pke.Describe(ch)

	// Assert
	require.Empty(t, ch)
}
