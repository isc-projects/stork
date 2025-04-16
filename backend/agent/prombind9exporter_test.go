package agent

import (
	_ "embed"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

//go:embed testdata/bind9-prom-server-stats.json
var bind9PromServerStats []byte

//go:embed testdata/bind9-prom-traffic-stats.json
var bind9PromTrafficStats []byte

// Fake app monitor that returns some predefined list of apps.
type PromFakeBind9AppMonitor struct{}

func (fam *PromFakeBind9AppMonitor) GetApps() []App {
	accessPoints := makeAccessPoint(AccessPointStatistics, "localhost", "", 1234, false)
	accessPoints = append(accessPoints, AccessPoint{
		Type:    AccessPointControl,
		Address: "1.9.5.3",
		Port:    1953,
		Key:     "abcd",
	})
	ba := &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient: nil,
	}
	return []App{ba}
}

func (fam *PromFakeBind9AppMonitor) GetApp(appType, apType, address string, port int64) App {
	return nil
}

func (fam *PromFakeBind9AppMonitor) Shutdown() {
}

func (fam *PromFakeBind9AppMonitor) Start(storkAgent *StorkAgent) {
}

// Check creating PromBind9Exporter, check if prometheus stats are set up.
func TestNewPromBind9ExporterBasic(t *testing.T) {
	fam := &PromFakeBind9AppMonitor{}
	httpClient := NewBind9StatsClient()
	pbe := NewPromBind9Exporter("foo", 42, fam, httpClient)
	defer pbe.Shutdown()

	require.Equal(t, "foo", pbe.Host)
	require.Equal(t, 42, pbe.Port)
	require.NotNil(t, pbe.HTTPClient)
	require.NotNil(t, pbe.HTTPServer)
	require.Len(t, pbe.serverStatsDesc, 20)
	require.Len(t, pbe.viewStatsDesc, 18)
}

// Check starting PromBind9Exporter and collecting stats.
func TestPromBind9ExporterStart(t *testing.T) {
	defer gock.Off()
	gock.New("http://localhost:1234/").
		Get("json/(.+)/server").
		AddMatcher(func(r1 *http.Request, r2 *gock.Request) (bool, error) {
			// Require empty body
			return r1.Body == nil, nil
		}).
		Reply(200).
		AddHeader("Content-Type", "application/json").
		BodyString(string(bind9PromServerStats))

	gock.New("http://localhost:1234/").
		Get("json/(.+)/traffic").
		AddMatcher(func(r1 *http.Request, r2 *gock.Request) (bool, error) {
			// Require empty body
			return r1.Body == nil, nil
		}).
		Reply(200).
		AddHeader("Content-Type", "application/json").
		BodyString(string(bind9PromTrafficStats))

	fam := &PromFakeBind9AppMonitor{}
	httpClient := NewBind9StatsClient()
	pbe := NewPromBind9Exporter("localhost", 1234, fam, httpClient)
	defer pbe.Shutdown()

	gock.InterceptClient(pbe.HTTPClient.innerClient.GetClient())

	// start exporter
	pbe.Start()
	require.EqualValues(t, 1, pbe.up)

	// boot_time_seconds
	expect, _ := time.Parse(time.RFC3339, "2020-04-21T07:13:08.888Z")
	require.EqualValues(t, expect, pbe.stats.BootTime)
	// config_time_seconds
	expect, _ = time.Parse(time.RFC3339, "2020-04-21T07:13:09.989Z")
	require.EqualValues(t, expect, pbe.stats.ConfigTime)
	// current_time_seconds
	expect, _ = time.Parse(time.RFC3339, "2020-04-21T07:19:28.258Z")
	require.EqualValues(t, expect, pbe.stats.CurrentTime)

	// incoming_queries_total
	require.EqualValues(t, 201.0, pbe.stats.IncomingQueries["A"])
	require.EqualValues(t, 200.0, pbe.stats.IncomingQueries["AAAA"])
	require.EqualValues(t, 53.0, pbe.stats.IncomingQueries["DNSKEY"])
	// incoming_requests_total
	require.EqualValues(t, 454.0, pbe.stats.IncomingRequests["QUERY"])
	require.EqualValues(t, 1.0, pbe.stats.IncomingRequests["UPDATE"])
	require.EqualValues(t, 0.0, pbe.stats.IncomingRequests["IQUERY"])

	// incoming_queries_tcp
	require.EqualValues(t, 303.0, pbe.stats.NsStats["QryTCP"])
	// incoming_queries_udp
	require.EqualValues(t, 404.0, pbe.stats.NsStats["QryUDP"])

	// query_duplicates_total
	require.EqualValues(t, 15.0, pbe.stats.NsStats["QryDuplicate"])
	// query_errors_total
	require.EqualValues(t, 9.0, pbe.stats.NsStats["QryDropped"])
	require.EqualValues(t, 3.0, pbe.stats.NsStats["QryFailure"])
	// query_recursion_total
	require.EqualValues(t, 303.0, pbe.stats.NsStats["QryRecursion"])
	// recursive_clients (unset value)
	require.EqualValues(t, 0.0, pbe.stats.NsStats["RecursClients"])

	// resolver_cache_rrsets
	require.EqualValues(t, 37.0, pbe.stats.Views["_default"].ResolverCache["A"])
	require.EqualValues(t, 38.0, pbe.stats.Views["_default"].ResolverCache["AAAA"])
	require.EqualValues(t, 2.0, pbe.stats.Views["_default"].ResolverCache["DS"])

	// resolver_cache_hit_ratio
	require.EqualValues(t, 0.8, pbe.stats.Views["_default"].ResolverCachestats["CacheHitRatio"])
	// resolver_cache_hits
	require.EqualValues(t, 40.0, pbe.stats.Views["_default"].ResolverCachestats["CacheHits"])
	// resolver_cache_misses
	require.EqualValues(t, 10.0, pbe.stats.Views["_default"].ResolverCachestats["CacheMisses"])
	// resolver_query_hit_ratio
	require.EqualValues(t, 0.6, pbe.stats.Views["_default"].ResolverCachestats["QueryHitRatio"])
	// resolver_query_hits
	require.EqualValues(t, 30.0, pbe.stats.Views["_default"].ResolverCachestats["QueryHits"])
	// resolver_query_misses
	require.EqualValues(t, 20.0, pbe.stats.Views["_default"].ResolverCachestats["QueryMisses"])

	// resolver_dnssec_validation_errors_total
	require.EqualValues(t, 5.0, pbe.stats.Views["_default"].ResolverStats["ValFail"])
	// resolver_dnssec_validation_success_total
	require.EqualValues(t, 3.0, pbe.stats.Views["_default"].ResolverStats["ValNegOk"])
	require.EqualValues(t, 17.0, pbe.stats.Views["_default"].ResolverStats["ValOk"])

	// resolver_queries_total
	require.EqualValues(t, 37.0, pbe.stats.Views["_default"].ResolverQtypes["A"])
	require.EqualValues(t, 36.0, pbe.stats.Views["_default"].ResolverQtypes["AAAA"])
	require.EqualValues(t, 4.0, pbe.stats.Views["_default"].ResolverQtypes["DNSKEY"])
	require.EqualValues(t, 6.0, pbe.stats.Views["_default"].ResolverQtypes["DS"])
	require.EqualValues(t, 7.0, pbe.stats.Views["_default"].ResolverQtypes["NS"])
	require.EqualValues(t, 21.0, pbe.stats.Views["_default"].ResolverQtypes["RRSIG"])

	// resolver_query_duration_seconds_bucket
	// resolver_query_duration_seconds_count
	// resolver_query_duration_seconds_sum
	require.EqualValues(t, 2.0, pbe.stats.Views["_default"].ResolverStats["QryRTT10"])
	require.EqualValues(t, 18.0, pbe.stats.Views["_default"].ResolverStats["QryRTT100"])
	require.EqualValues(t, 37.0, pbe.stats.Views["_default"].ResolverStats["QryRTT500"])
	require.EqualValues(t, 3.0, pbe.stats.Views["_default"].ResolverStats["QryRTT800"])
	require.EqualValues(t, 1.0, pbe.stats.Views["_default"].ResolverStats["QryRTT1600"])
	require.EqualValues(t, 4.0, pbe.stats.Views["_default"].ResolverStats["QryRTT1600+"])
	count, _, buckets, err := pbe.qryRTTHistogram(pbe.stats.Views["_default"].ResolverStats)
	require.EqualValues(t, 65.0, count)
	require.Len(t, buckets, 6)
	require.EqualValues(t, 2, buckets[0.01])
	require.EqualValues(t, 20, buckets[0.1])
	require.EqualValues(t, 57, buckets[0.5])
	require.EqualValues(t, 60, buckets[0.8])
	require.EqualValues(t, 61, buckets[1.6])
	require.EqualValues(t, 65, buckets[math.Inf(0)])
	require.Nil(t, err)

	// resolver_query_edns0_errors_total
	// resolver_query_errors_total
	// resolver_query_retries_total
	require.EqualValues(t, 5.0, pbe.stats.Views["_default"].ResolverStats["EDNS0Fail"])
	require.EqualValues(t, 1.0, pbe.stats.Views["_default"].ResolverStats["QueryAbort"])
	require.EqualValues(t, 10.0, pbe.stats.Views["_default"].ResolverStats["QueryTimeout"])
	require.EqualValues(t, 0.0, pbe.stats.Views["_default"].ResolverStats["QuerySockFail"])
	require.EqualValues(t, 71.0, pbe.stats.Views["_default"].ResolverStats["Retry"])

	// resolver_response_errors_total
	require.EqualValues(t, 13.0, pbe.stats.Views["_default"].ResolverStats["FORMERR"])
	require.EqualValues(t, 50.0, pbe.stats.Views["_default"].ResolverStats["NXDOMAIN"])
	require.EqualValues(t, 404.0, pbe.stats.Views["_default"].ResolverStats["SERVFAIL"])
	require.EqualValues(t, 42.0, pbe.stats.Views["_default"].ResolverStats["OtherError"])
	// resolver_response_lame_total
	// resolver_response_mismatch_total
	// resolver_response_truncated_total
	require.EqualValues(t, 9.0, pbe.stats.Views["_default"].ResolverStats["Lame"])
	require.EqualValues(t, 10.0, pbe.stats.Views["_default"].ResolverStats["Mismatch"])
	require.EqualValues(t, 7.0, pbe.stats.Views["_default"].ResolverStats["Truncated"])

	// responses_total
	require.EqualValues(t, 5.0, pbe.stats.NsStats["QryNxrrset"])
	require.EqualValues(t, 55.0, pbe.stats.NsStats["QryNXDOMAIN"])
	require.EqualValues(t, 555.0, pbe.stats.NsStats["QrySERVFAIL"])
	require.EqualValues(t, 111.0, pbe.stats.NsStats["QrySuccess"])
	require.EqualValues(t, 0.0, pbe.stats.NsStats["QryFORMERR"])
	require.EqualValues(t, 0.0, pbe.stats.NsStats["QryReferral"])

	// tasks_running
	require.EqualValues(t, 1.0, pbe.stats.TaskMgr["tasks-running"])
	// worker_threads
	require.EqualValues(t, 4.0, pbe.stats.TaskMgr["worker-threads"])

	// traffic_incoming_requests_udp4_size
	require.EqualValues(t, 206.0, pbe.stats.TrafficStats["dns-udp-requests-sizes-received-ipv4"].SizeCount["32-47"])
	require.EqualValues(t, 24.0, pbe.stats.TrafficStats["dns-udp-requests-sizes-received-ipv4"].SizeCount["128+"])
	count, _, buckets, err = pbe.trafficSizesHistogram(pbe.stats.TrafficStats["dns-udp-requests-sizes-received-ipv4"].SizeCount)
	require.EqualValues(t, 230.0, count)
	require.Len(t, buckets, 2)
	require.EqualValues(t, 206, buckets[47])
	require.EqualValues(t, 230, buckets[math.Inf(0)])
	require.Nil(t, err)
	// traffic_responses_udp4_size
	require.EqualValues(t, 196.0, pbe.stats.TrafficStats["dns-udp-responses-sizes-sent-ipv4"].SizeCount["96-111"])
	require.EqualValues(t, 10.0, pbe.stats.TrafficStats["dns-udp-responses-sizes-sent-ipv4"].SizeCount["112-127"])
	count, _, buckets, err = pbe.trafficSizesHistogram(pbe.stats.TrafficStats["dns-udp-responses-sizes-sent-ipv4"].SizeCount)
	require.EqualValues(t, 206.0, count)
	require.Len(t, buckets, 3)
	require.EqualValues(t, 196, buckets[111])
	require.EqualValues(t, 206, buckets[127])
	require.EqualValues(t, 206, buckets[math.Inf(0)])
	require.Nil(t, err)
	// traffic_incoming_requests_tcp4_size
	require.EqualValues(t, 12.0, pbe.stats.TrafficStats["dns-tcp-requests-sizes-received-ipv4"].SizeCount["32-47"])
	count, _, buckets, err = pbe.trafficSizesHistogram(pbe.stats.TrafficStats["dns-tcp-requests-sizes-received-ipv4"].SizeCount)
	require.EqualValues(t, 12.0, count)
	require.Len(t, buckets, 2)
	require.EqualValues(t, 12, buckets[47])
	require.EqualValues(t, 12, buckets[math.Inf(0)])
	require.Nil(t, err)
	// traffic_responses_tcp4_size
	require.EqualValues(t, 12.0, pbe.stats.TrafficStats["dns-tcp-responses-sizes-sent-ipv4"].SizeCount["128-143"])
	count, _, buckets, err = pbe.trafficSizesHistogram(pbe.stats.TrafficStats["dns-tcp-responses-sizes-sent-ipv4"].SizeCount)
	require.EqualValues(t, 12.0, count)
	require.Len(t, buckets, 2)
	require.EqualValues(t, 12, buckets[143])
	require.EqualValues(t, 12, buckets[math.Inf(0)])
	require.Nil(t, err)
	// traffic_incoming_requests_udp6_size
	require.Len(t, pbe.stats.TrafficStats["dns-udp-requests-sizes-received-ipv6"].SizeCount, 0)
	count, _, buckets, err = pbe.trafficSizesHistogram(pbe.stats.TrafficStats["dns-udp-requests-sizes-received-ipv6"].SizeCount)
	require.Zero(t, count)
	require.Len(t, buckets, 1)
	require.Zero(t, buckets[math.Inf(0)])
	require.Nil(t, err)
	// traffic_responses_udp6_size
	require.Len(t, pbe.stats.TrafficStats["dns-udp-responses-sizes-sent-ipv6"].SizeCount, 0)
	count, _, buckets, err = pbe.trafficSizesHistogram(pbe.stats.TrafficStats["dns-udp-responses-sizes-sent-ipv6"].SizeCount)
	require.Zero(t, count)
	require.Len(t, buckets, 1)
	require.Zero(t, buckets[math.Inf(0)])
	require.Nil(t, err)
	// traffic_incoming_requests_tcp6_size
	require.Len(t, pbe.stats.TrafficStats["dns-tcp-requests-sizes-received-ipv6"].SizeCount, 0)
	count, _, buckets, err = pbe.trafficSizesHistogram(pbe.stats.TrafficStats["dns-tcp-requests-sizes-received-ipv6"].SizeCount)
	require.Zero(t, count)
	require.Len(t, buckets, 1)
	require.Zero(t, buckets[math.Inf(0)])
	require.Nil(t, err)
	// traffic_responses_tcp6_size
	require.Len(t, pbe.stats.TrafficStats["dns-tcp-responses-sizes-sent-ipv6"].SizeCount, 0)
	count, _, buckets, err = pbe.trafficSizesHistogram(pbe.stats.TrafficStats["dns-tcp-responses-sizes-sent-ipv6"].SizeCount)
	require.Zero(t, count)
	require.Len(t, buckets, 1)
	require.Zero(t, buckets[math.Inf(0)])
	require.Nil(t, err)

	// zone_transfer_failure_total
	require.EqualValues(t, 2.0, pbe.stats.NsStats["XfrFail"])
	// zone_transfer_rejected_total
	require.EqualValues(t, 11.0, pbe.stats.NsStats["XfrRej"])
	// zone_transfer_success_total
	require.EqualValues(t, 22.0, pbe.stats.NsStats["XfrSuccess"])
}
