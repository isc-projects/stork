package agent

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
)

const (
	namespace = "bind"
	qryRTT    = "QryRTT"
)

// The traffic BIND 9 statistics.
type PromBind9TrafficStats struct {
	SizeCount map[string]float64
}

// The view BIND 9 statistics.
type PromBind9ViewStats struct {
	ResolverCache      map[string]float64
	ResolverCachestats map[string]float64
	ResolverQtypes     map[string]float64
	ResolverStats      map[string]float64
}

// Statistics to be exported.
type PromBind9ExporterStats struct {
	BootTime         time.Time
	ConfigTime       time.Time
	CurrentTime      time.Time
	IncomingQueries  map[string]float64
	IncomingRequests map[string]float64
	NsStats          map[string]float64
	TaskMgr          map[string]float64
	TrafficStats     map[string]PromBind9TrafficStats
	Views            map[string]PromBind9ViewStats
}

// Main structure for Prometheus BIND 9 Exporter. It holds its config,
// references to app monitor, HTTP client, HTTP server, and mappings
// between BIND 9 stats names to prometheus stats.
type PromBind9Exporter struct {
	Host string
	Port int

	StartTime time.Time

	AppMonitor AppMonitor
	HTTPClient *bind9StatsClient
	HTTPServer *http.Server

	up               int
	procID           int32
	procExporter     prometheus.Collector
	Registry         *prometheus.Registry
	serverStatsDesc  map[string]*prometheus.Desc
	trafficStatsDesc map[string]*prometheus.Desc
	viewStatsDesc    map[string]*prometheus.Desc

	stats PromBind9ExporterStats
}

// Create new Prometheus BIND 9 Exporter.
func NewPromBind9Exporter(host string, port int, appMonitor AppMonitor, httpClient *bind9StatsClient) *PromBind9Exporter {
	pbe := &PromBind9Exporter{
		Host:       host,
		Port:       port,
		StartTime:  time.Now(),
		AppMonitor: appMonitor,
		HTTPClient: httpClient,
		Registry:   prometheus.NewRegistry(),
	}

	// bind_exporter stats
	serverStatsDesc := make(map[string]*prometheus.Desc)
	trafficStatsDesc := make(map[string]*prometheus.Desc)
	viewStatsDesc := make(map[string]*prometheus.Desc)

	// uptime_seconds
	serverStatsDesc["uptime-seconds"] = prometheus.NewDesc(
		prometheus.BuildFQName("storkagent", "prombind9exporter", "uptime_seconds"),
		"Uptime of the Prometheus BIND 9 Exporter in seconds",
		nil, nil)
	// boot_time_seconds
	serverStatsDesc["boot-time"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "boot_time_seconds"),
		"Start time of the BIND process since unix epoch in seconds.",
		nil, nil)
	// config_time_seconds
	serverStatsDesc["config-time"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "config_time_seconds"),
		"Time of the last reconfiguration since unix epoch in seconds.",
		nil, nil)
	// current_time_seconds
	serverStatsDesc["current-time"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "current_time_seconds"),
		"Current time unix epoch in seconds as reported by named.",
		nil, nil)

	// incoming_queries_total
	serverStatsDesc["qtypes"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "incoming_queries_total"),
		"Number of incoming DNS queries.",
		[]string{"type"}, nil)
	// incoming_queries_tcp
	serverStatsDesc["QryTCP"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "incoming_queries_tcp"),
		"Number of incoming TCP queries.",
		nil, nil)
	// incoming_queries_udp
	serverStatsDesc["QryUDP"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "incoming_queries_udp"),
		"Number of incoming UDP queries.",
		nil, nil)

	// incoming_requests_total
	serverStatsDesc["opcodes"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "incoming_requests_total"),
		"Number of incoming DNS requests.",
		[]string{"opcode"}, nil)
	// incoming_requests_tcp
	serverStatsDesc["ReqTCP"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "incoming_requests_tcp"),
		"Number of incoming TCP requests.",
		nil, nil)

	// traffic_incoming_requests_udp4_size_bucket
	// traffic_incoming_requests_udp4_size_count
	// traffic_incoming_requests_udp4_size_sum
	trafficStatsDesc["dns-udp-requests-sizes-received-ipv4"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_incoming_requests_udp4_size"),
		"Size of DNS requests (UDP/IPv4).",
		nil, nil)
	// traffic_incoming_requests_udp6_size_bucket
	// traffic_incoming_requests_udp6_size_count
	// traffic_incoming_requests_udp6_size_sum
	trafficStatsDesc["dns-udp-requests-sizes-received-ipv6"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_incoming_requests_udp6_size"),
		"Size of DNS requests (UDP/IPv6).",
		nil, nil)
	// traffic_incoming_requests_tcp4_size_bucket
	// traffic_incoming_requests_tcp4_size_count
	// traffic_incoming_requests_tcp4_size_sum
	trafficStatsDesc["dns-tcp-requests-sizes-received-ipv4"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_incoming_requests_tcp4_size"),
		"Size of DNS requests (TCP/IPv4).",
		nil, nil)
	// traffic_incoming_requests_tcp6_size_bucket
	// traffic_incoming_requests_tcp6_size_count
	// traffic_incoming_requests_tcp6_size_sum
	trafficStatsDesc["dns-tcp-requests-sizes-received-ipv6"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_incoming_requests_tcp6_size"),
		"Size of DNS requests (TCP/IPv6).",
		nil, nil)
	// traffic_incoming_requests_total_size_bucket
	// traffic_incoming_requests_total_size_count
	// traffic_incoming_requests_total_size_sum
	trafficStatsDesc["dns-total-requests-sizes-sent"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_incoming_requests_total_size"),
		"Size of DNS requests (any transport).",
		nil, nil)

	// traffic_responses_udp4_size_bucket
	// traffic_responses_udp4_size_count
	// traffic_responses_udp4_size_sum
	trafficStatsDesc["dns-udp-responses-sizes-sent-ipv4"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_responses_udp4_size"),
		"Size of DNS responses (UDP/IPv4).",
		nil, nil)
	// traffic_responses_udp6_size_bucket
	// traffic_responses_udp6_size_count
	// traffic_responses_udp6_size_sum
	trafficStatsDesc["dns-udp-responses-sizes-sent-ipv6"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_responses_udp6_size"),
		"Size of DNS responses (UDP/IPv6).",
		nil, nil)
	// traffic_responses_tcp4_size_bucket
	// traffic_responses_tcp4_size_count
	// traffic_responses_tcp4_size_sum
	trafficStatsDesc["dns-tcp-responses-sizes-sent-ipv4"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_responses_tcp4_size"),
		"Size of DNS responses (TCP/IPv4).",
		nil, nil)
	// traffic_responses_tcp6_size_bucket
	// traffic_responses_tcp6_size_count
	// traffic_responses_tcp6_size_sum
	trafficStatsDesc["dns-tcp-responses-sizes-sent-ipv6"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_responses_tcp6_size"),
		"Size of DNS responses (TCP/IPv6).",
		nil, nil)
	// traffic_responses_total_size_bucket
	// traffic_responses_total_size_count
	// traffic_responses_total_size_sum
	trafficStatsDesc["dns-total-responses-sizes-sent"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "traffic_responses_total_size"),
		"Size of DNS responses (any transport).",
		nil, nil)

	// query_duplicates_total
	serverStatsDesc["QryDuplicate"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "query_duplicates_total"),
		"Number of duplicated queries received.",
		nil, nil)
	// query_errors_total
	serverStatsDesc["QryErrors"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "query_errors_total"),
		"Number of query failures.",
		[]string{"error"}, nil)
	// query_recursions_total
	serverStatsDesc["QryRecursion"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "query_recursions_total"),
		"Number of queries causing recursion.",
		nil, nil)
	// recursive_clients
	serverStatsDesc["RecursClients"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "recursive_clients"),
		"Number of current recursive clients.",
		nil, nil)

	// resolver_cache_hit_ratio
	viewStatsDesc["CacheHitRatio"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "cache_hit_ratio"),
		"Cache effectiveness (cache hit ratio).",
		[]string{"view"}, nil)
	// resolver_cache_hits
	viewStatsDesc["CacheHits"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "cache_hits"),
		"Total number of cache hits.",
		[]string{"view"}, nil)
	// resolver_cache_misses
	viewStatsDesc["CacheMisses"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "cache_misses"),
		"Total number of cache misses.",
		[]string{"view"}, nil)
	// resolver_cache_rrsets
	viewStatsDesc["cache"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "cache_rrsets"),
		"Number of RRsets in cache database.",
		[]string{"view", "type"}, nil)
	// resolver_query_hit_ratio
	viewStatsDesc["QueryHitRatio"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "query_hit_ratio"),
		"Query effectiveness (query hit ratio).",
		[]string{"view"}, nil)
	// resolver_query_hits
	viewStatsDesc["QueryHits"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "query_hits"),
		"Total number of queries that were answered from cache.",
		[]string{"view"}, nil)
	// resolver_query_misses
	viewStatsDesc["QueryMisses"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "query_misses"),
		"Total number of queries that were not in cache.",
		[]string{"view"}, nil)

	// resolver_dnssec_validation_errors_total
	viewStatsDesc["ValFail"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "dnssec_validation_errors_total"),
		"Number of DNSSEC validation attempt errors.",
		[]string{"view"}, nil)
	// resolver_dnssec_validation_success_total
	viewStatsDesc["ValSuccess"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "dnssec_validation_success_total"),
		"Number of successful DNSSEC validation attempts.",
		[]string{"view", "result"}, nil)

	// resolver_queries_total
	viewStatsDesc["ResolverQueries"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "queries_total"),
		"Number of outgoing DNS queries.",
		[]string{"view", "type"}, nil)

	// resolver_query_duration_seconds_bucket
	// resolver_query_duration_seconds_count
	// resolver_query_duration_seconds_sum
	viewStatsDesc["QueryDuration"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "query_duration_seconds"),
		"Resolver query round-trip time in seconds.",
		[]string{"view"}, nil)

	// resolver_query_edns0_errors_total
	viewStatsDesc["EDNS0Fail"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "query_edns0_errors_total"),
		"Number of EDNS(0) query errors.",
		[]string{"view"}, nil)
	// resolver_query_errors_total
	viewStatsDesc["ResolverQueryErrors"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "query_errors_total"),
		"Number of failed resolver queries.",
		[]string{"view", "error"}, nil)
	// resolver_query_retries_total
	viewStatsDesc["Retry"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "query_retries_total"),
		"Number of resolver query retries.",
		[]string{"view"}, nil)

	// resolver_response_errors_total
	viewStatsDesc["ResolverResponseErrors"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "response_errors_total"),
		"Number of resolver response errors received.",
		[]string{"view", "error"}, nil)
	// resolver_response_lame_total
	viewStatsDesc["Lame"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "response_lame_total"),
		"Number of lame delegation responses received.",
		[]string{"view"}, nil)
	// resolver_response_mismatch_total
	viewStatsDesc["Mismatch"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "response_mismatch_total"),
		"Number of mismatch responses received.",
		[]string{"view"}, nil)
	// resolver_response_truncated_total
	viewStatsDesc["Truncated"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "resolver", "response_truncated_total"),
		"Number of truncated responses received.",
		[]string{"view"}, nil)

	// responses_total
	serverStatsDesc["ServerResponses"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "responses_total"),
		"Number of responses sent.",
		[]string{"result"}, nil)

	// tasks_running
	serverStatsDesc["tasks-running"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "tasks_running"),
		"Number of running tasks.",
		nil, nil)
	// up
	serverStatsDesc["up"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the BIND instance query successful?",
		nil, nil)
	// worker_threads
	serverStatsDesc["worker-threads"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "worker_threads"),
		"Total number of available worker threads.",
		nil, nil)

	// zone_transfer_failure_total
	serverStatsDesc["XfrFail"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "zone_transfer_failure_total"),
		"Number of failed zone transfers.",
		nil, nil)
	// zone_transfer_rejected_total
	serverStatsDesc["XfrRej"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "zone_transfer_rejected_total"),
		"Number of rejected zone transfers.",
		nil, nil)
	// zone_transfer_success_total
	serverStatsDesc["XfrSuccess"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "zone_transfer_success_total"),
		"Number of successful zone transfers.",
		nil, nil)

	pbe.serverStatsDesc = serverStatsDesc
	pbe.trafficStatsDesc = trafficStatsDesc
	pbe.viewStatsDesc = viewStatsDesc

	incomingQueries := make(map[string]float64)
	views := make(map[string]PromBind9ViewStats)
	pbe.stats = PromBind9ExporterStats{
		IncomingQueries: incomingQueries,
		Views:           views,
	}

	// prepare http handler
	mux := http.NewServeMux()
	handler := promhttp.HandlerFor(pbe.Registry, promhttp.HandlerOpts{})
	mux.Handle("/metrics", handler)
	pbe.HTTPServer = &http.Server{
		Handler: mux,
		// Protection against Slowloris Attack (G112).
		ReadHeaderTimeout: 60 * time.Second,
	}

	return pbe
}

// Describe describes all exported metrics. It implements prometheus.Collector.
func (pbe *PromBind9Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range pbe.serverStatsDesc {
		ch <- m
	}
	for _, m := range pbe.viewStatsDesc {
		ch <- m
	}
}

// collectTime collects time stats.
func (pbe *PromBind9Exporter) collectTime(ch chan<- prometheus.Metric, key string, timeStat time.Time) {
	if !timeStat.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			pbe.serverStatsDesc[key],
			prometheus.GaugeValue,
			float64(timeStat.Unix()))
	}
}

// qryRTTHistogram collects a histogram from QryRTT statistics.
// RTT buckets are per second, for example bucket[0.8] stores how many query
// round trips took up to 800 milliseconds (cumulative counter).
// The total sum of all observed values is exposed with sum, but since named
// does not output the actual RTT values, this is not applicable.
// The count of events that have been observed is exposed with count and is
// identical to bucket[+Inf].
func (pbe *PromBind9Exporter) qryRTTHistogram(stats map[string]float64) (uint64, float64, map[float64]uint64, error) {
	buckets := map[float64]uint64{}

	for statName, statValue := range stats {
		// Find all statistics QryRTT<n>[+].
		// Each statistic represents a bucket with the number of
		// queries whose RTTs are up to <n> milliseconds, excluding
		// the count of previous buckets. Furthermore, if the
		// statistic ends in '+', this specifies the number of queries
		// whose RTT was is higher than <n> milliseconds.  So if we
		// have the following statistics:
		//
		//     QryRTT10: 5
		//     QryRTT50: 40
		//     QryRTT100: 10
		//     QryRTT100+: 1
		//
		// We have 5 queries whose RTT was below 10ms, 40 queries whose
		// RTT was between 10ms and 50ms, 10 queries whose RTT was
		// between 50ms and 100ms, and one query whose RTT was above
		// 100ms.
		// Each <n> represents a bucket and if the statistic ended in
		// a '+' we will consider that those queries took up to an
		// infinite time. Buckets are represented as seconds, so the
		// expected buckets to return are:
		//
		//    0.01: 5
		//    0.05: 45
		//    0.1 : 55
		//    Inf:  56
		if strings.HasPrefix(statName, qryRTT) {
			var bucket float64
			var err error
			if strings.HasSuffix(statName, "+") {
				bucket = math.Inf(0)
			} else {
				rtt := strings.TrimPrefix(statName, qryRTT)
				bucket, err = strconv.ParseFloat(rtt, 64)
				if err != nil {
					return 0, math.NaN(), buckets, pkgerrors.Errorf("could not parse RTT: %s", rtt)
				}
			}
			buckets[bucket/1000] = uint64(statValue)
		}
	}

	// cumulative count
	keys := make([]float64, 0, len(buckets))
	for b := range buckets {
		keys = append(keys, b)
	}
	sort.Float64s(keys)

	var count uint64
	for _, k := range keys {
		count += buckets[k]
		buckets[k] = count
	}

	return count, math.NaN(), buckets, nil
}

// trafficSizesHistogram collects a histogram from the traffic statistics as.
// 'buckets'.  Size buckets are in bytes, for example bucket[47] stores how
// many packets were at most 47 bytes long (cumulative counter).  The total
// sum of all observed values is exposed with 'sum', but since named does not
// output the actual sizes, this is not applicable. The count of events that
// have been observed is exposed with 'count' and is identical to bucket[+Inf].
func (pbe *PromBind9Exporter) trafficSizesHistogram(stats map[string]float64) (count uint64, sum float64, buckets map[float64]uint64, err error) {
	count = 0
	sum = math.NaN()
	buckets = map[float64]uint64{}

	buckets[math.Inf(0)] = 0
	for statName, statValue := range stats {
		// Find all traffic statistics.
		var bucket float64
		var err error
		if strings.HasSuffix(statName, "+") {
			bucket = math.Inf(0)
		} else {
			// The statistic name is of the format:
			//    <sizeMin>-<sizeMax>
			// Fetch the maximum size and put in corresponding
			// bucket.
			sizes := strings.SplitAfter(statName, "-")
			if len(sizes) != 2 {
				// bad format
				continue
			}
			bucket, err = strconv.ParseFloat(sizes[1], 64)
			if err != nil {
				return 0, math.NaN(), buckets, pkgerrors.Errorf("could not parse size: %s", sizes[1])
			}
		}
		buckets[bucket] = uint64(statValue)
	}

	// cumulative count
	keys := make([]float64, 0, len(buckets))
	for b := range buckets {
		keys = append(keys, b)
	}
	sort.Float64s(keys)

	for _, k := range keys {
		count += buckets[k]
		buckets[k] = count
	}

	return count, sum, buckets, nil
}

// collectResolverStat fetches a specific resolver view statistic.
func (pbe *PromBind9Exporter) collectResolverStat(statName, view string, viewStat PromBind9ViewStats, ch chan<- prometheus.Metric) {
	statValue, ok := viewStat.ResolverStats[statName]
	if !ok {
		statValue = 0
	}
	ch <- prometheus.MustNewConstMetric(
		pbe.viewStatsDesc[statName],
		prometheus.CounterValue, statValue, view)
}

// collectResolverLabelStat fetches a specific resolver view statistic
// with a label.
func (pbe *PromBind9Exporter) collectResolverLabelStat(statName, view string, viewStat PromBind9ViewStats, ch chan<- prometheus.Metric, labels []string) {
	for _, label := range labels {
		resolverStatValue, ok := viewStat.ResolverStats[label]
		if !ok {
			resolverStatValue = 0
		}
		ch <- prometheus.MustNewConstMetric(
			pbe.viewStatsDesc[statName],
			prometheus.CounterValue, resolverStatValue, view, label)
	}
}

// Collect fetches the stats from configured location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (pbe *PromBind9Exporter) Collect(ch chan<- prometheus.Metric) {
	var err error
	pbe.procID, err = pbe.collectStats()
	if pbe.procID == 0 {
		return
	}

	// uptime_seconds (Uptime of the Stork Agent)
	ch <- prometheus.MustNewConstMetric(
		pbe.serverStatsDesc["uptime-seconds"],
		prometheus.GaugeValue,
		time.Since(pbe.StartTime).Seconds())

	// up
	ch <- prometheus.MustNewConstMetric(pbe.serverStatsDesc["up"], prometheus.GaugeValue, float64(pbe.up))

	if err != nil {
		log.Errorf("Some errors were encountered while collecting stats from BIND 9: %+v", err)
	}

	// if not up or error encountered, don't bother collecting.
	if pbe.up == 0 || err != nil {
		return
	}

	// boot_time_seconds
	pbe.collectTime(ch, "boot-time", pbe.stats.BootTime)
	// config_time_seconds
	pbe.collectTime(ch, "config-time", pbe.stats.ConfigTime)
	// current_time_seconds
	pbe.collectTime(ch, "current-time", pbe.stats.CurrentTime)

	// incoming_queries_total
	for label, value := range pbe.stats.IncomingQueries {
		ch <- prometheus.MustNewConstMetric(
			pbe.serverStatsDesc["qtypes"],
			prometheus.CounterValue,
			value, label)
	}
	// incoming_requests_total
	for label, value := range pbe.stats.IncomingRequests {
		ch <- prometheus.MustNewConstMetric(
			pbe.serverStatsDesc["opcodes"],
			prometheus.CounterValue,
			value, label)
	}

	// incoming_requests_tcp
	value, ok := pbe.stats.NsStats["ReqTCP"]
	if !ok {
		value = 0
	}
	ch <- prometheus.MustNewConstMetric(
		pbe.serverStatsDesc["ReqTCP"],
		prometheus.CounterValue, value)
	// query_tcp_total
	value, ok = pbe.stats.NsStats["QryTCP"]
	if !ok {
		value = 0
	}
	ch <- prometheus.MustNewConstMetric(
		pbe.serverStatsDesc["QryTCP"],
		prometheus.CounterValue, value)
	// query_udp_total
	value, ok = pbe.stats.NsStats["QryUDP"]
	if !ok {
		value = 0
	}
	ch <- prometheus.MustNewConstMetric(
		pbe.serverStatsDesc["QryUDP"],
		prometheus.CounterValue, value)

	// query_duplicates_total
	value, ok = pbe.stats.NsStats["QryDuplicate"]
	if !ok {
		value = 0
	}
	ch <- prometheus.MustNewConstMetric(
		pbe.serverStatsDesc["QryDuplicate"],
		prometheus.CounterValue, value)
	// query_errors_total
	trimQryPrefix := func(name string) string {
		return strings.TrimPrefix(name, "Qry")
	}
	qryErrors := []string{"QryDropped", "QryFailure"}
	for _, label := range qryErrors {
		value, ok = pbe.stats.NsStats[label]
		if !ok {
			value = 0
		}

		ch <- prometheus.MustNewConstMetric(
			pbe.serverStatsDesc["QryErrors"],
			prometheus.CounterValue,
			value, trimQryPrefix(label))
	}
	// query_recursion_total
	value, ok = pbe.stats.NsStats["QryRecursion"]
	if !ok {
		value = 0
	}
	ch <- prometheus.MustNewConstMetric(
		pbe.serverStatsDesc["QryRecursion"],
		prometheus.CounterValue, value)
	// recursive_clients
	value, ok = pbe.stats.NsStats["RecursClients"]
	if ok {
		ch <- prometheus.MustNewConstMetric(
			pbe.serverStatsDesc["RecursClients"],
			prometheus.CounterValue, value)
	}

	// responses_total
	serverResponses := []string{
		"QrySuccess",
		"QryReferral",
		"QryNxrrset",
		"QrySERVFAIL",
		"QryFORMERR",
		"QryNXDOMAIN",
	}
	for _, label := range serverResponses {
		value, ok = pbe.stats.NsStats[label]
		if !ok {
			value = 0
		}

		ch <- prometheus.MustNewConstMetric(
			pbe.serverStatsDesc["ServerResponses"],
			prometheus.CounterValue,
			value, trimQryPrefix(label))
	}

	// tasks_running
	// worker_threads
	taskMgrStats := []string{"tasks-running", "worker-threads"}
	for _, label := range taskMgrStats {
		value, ok = pbe.stats.TaskMgr[label]
		if !ok {
			value = 0
		}

		ch <- prometheus.MustNewConstMetric(
			pbe.serverStatsDesc[label],
			prometheus.GaugeValue, value)
	}

	// zone_transfer_failure_total
	// zone_transfer_rejected_total
	// zone_transfer_success_total
	xfrStats := []string{"XfrFail", "XfrRej", "XfrSuccess"}
	for _, label := range xfrStats {
		value, ok = pbe.stats.NsStats[label]
		if !ok {
			value = 0
		}
		ch <- prometheus.MustNewConstMetric(
			pbe.serverStatsDesc[label],
			prometheus.CounterValue, value)
	}

	// Traffic metrics.

	// traffic_incoming_requests_udp4_size_{bucket,count,sum}
	// traffic_incoming_requests_udp6_size_{bucket,count,sum}
	// traffic_incoming_requests_tcp4_size_{bucket,count,sum}
	// traffic_incoming_requests_tcp6_size_{bucket,count,sum}
	// traffic_incoming_requests_total_size_{bucket,count,sum}
	// traffic_responses_udp4_size_{bucket,count,sum}
	// traffic_responses_udp6_size_{bucket,count,sum}
	// traffic_responses_tcp4_size_{bucket,count,sum}
	// traffic_responses_tcp6_size_{bucket,count,sum}
	// traffic_responses_total_size_{bucket,count,sum}
	for label, trafficStats := range pbe.stats.TrafficStats {
		if count, sum, buckets, err := pbe.trafficSizesHistogram(trafficStats.SizeCount); err == nil {
			ch <- prometheus.MustNewConstHistogram(
				pbe.trafficStatsDesc[label],
				count, sum, buckets)
		}
	}

	// View metrics.
	for view, viewStats := range pbe.stats.Views {
		// resolver_cache_rrsets
		for rrType, statValue := range viewStats.ResolverCache {
			ch <- prometheus.MustNewConstMetric(
				pbe.viewStatsDesc["cache"],
				prometheus.CounterValue,
				statValue, view, rrType)
		}

		// resolver_cache_hit_ratio
		// resolver_cache_hits
		// resolver_cache_misses
		// resolver_query_hit_ratio
		// resolver_query_hits
		// resolver_query_misses
		for statName, statValue := range viewStats.ResolverCachestats {
			if desc, ok := pbe.viewStatsDesc[statName]; ok {
				ch <- prometheus.MustNewConstMetric(
					desc, prometheus.CounterValue,
					statValue, view)
			}
		}

		// resolver_query_duration_seconds_bucket
		// resolver_query_duration_seconds_count
		// resolver_query_duration_seconds_sum
		if count, sum, buckets, err := pbe.qryRTTHistogram(viewStats.ResolverStats); err == nil {
			ch <- prometheus.MustNewConstHistogram(
				pbe.viewStatsDesc["QueryDuration"],
				count, sum, buckets, view)
		}

		// resolver_query_edns0_errors_total
		pbe.collectResolverStat("EDNS0Fail", view, viewStats, ch)

		// resolver_query_errors_total
		resolverQueryErrors := []string{"QueryAbort", "QuerySockFail", "QueryTimeout"}
		pbe.collectResolverLabelStat("ResolverQueryErrors", view, viewStats, ch, resolverQueryErrors)
		// resolver_query_retries_total
		pbe.collectResolverStat("Retry", view, viewStats, ch)
		// resolver_queries_total
		for statName, statValue := range viewStats.ResolverQtypes {
			ch <- prometheus.MustNewConstMetric(
				pbe.viewStatsDesc["ResolverQueries"],
				prometheus.CounterValue,
				statValue, view, statName)
		}

		// resolver_response_errors_total
		resolverResponseErrors := []string{"NXDOMAIN", "SERVFAIL", "FORMERR", "OtherError"}
		pbe.collectResolverLabelStat("ResolverResponseErrors", view, viewStats, ch, resolverResponseErrors)
		// resolver_response_lame_total
		pbe.collectResolverStat("Lame", view, viewStats, ch)
		// resolver_response_mismatch_total
		pbe.collectResolverStat("Mismatch", view, viewStats, ch)
		// resolver_response_truncated_total
		pbe.collectResolverStat("Truncated", view, viewStats, ch)

		// resolver_dnssec_validation_errors_total
		pbe.collectResolverStat("ValFail", view, viewStats, ch)
		// resolver_dnssec_validation_success_total
		valSuccess := []string{"ValOk", "ValNegOk"}
		pbe.collectResolverLabelStat("ValSuccess", view, viewStats, ch, valSuccess)
	}
}

// Start goroutine with main loop for collecting stats and HTTP server for
// exposing them to Prometheus.
func (pbe *PromBind9Exporter) Start() {
	// initial collect
	var err error
	pbe.procID, err = pbe.collectStats()
	if err != nil {
		log.WithError(err).Error("Some errors were encountered while collecting stats from BIND 9")
	}

	// register collectors
	version.Version = stork.Version
	pbe.Registry.MustRegister(pbe, versioncollector.NewCollector("bind_exporter"))
	pbe.procExporter = collectors.NewProcessCollector(
		collectors.ProcessCollectorOpts{
			PidFn: func() (int, error) {
				return int(pbe.procID), nil
			},
			Namespace: namespace,
		})
	pbe.Registry.MustRegister(pbe.procExporter)

	// set address for listening from config
	addrPort := net.JoinHostPort(pbe.Host, strconv.Itoa(pbe.Port))
	pbe.HTTPServer.Addr = addrPort

	log.Printf("Prometheus BIND 9 Exporter listening on %s", addrPort)

	// start HTTP server for metrics
	go func() {
		err := pbe.HTTPServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).
				Error("Problem serving Prometheus BIND 9 Exporter")
		}
	}()
}

// Shutdown exporter goroutines and unregister prometheus stats.
func (pbe *PromBind9Exporter) Shutdown() {
	log.Printf("Stopping Prometheus BIND 9 Exporter")

	// stop http server
	if pbe.HTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		pbe.HTTPServer.SetKeepAlivesEnabled(false)
		if err := pbe.HTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("Could not gracefully shut down the BIND 9 exporter: %v\n", err)
		}
	}

	// unregister bind9 counters from prometheus framework
	if pbe.procID > 0 {
		pbe.Registry.Unregister(pbe.procExporter)
	}
	pbe.Registry.Unregister(pbe)

	log.Printf("Stopped Prometheus BIND 9 Exporter")
}

// getStat is an utility to get a statistic from a map.
func getStat(statMap map[string]interface{}, statName string) interface{} {
	value, ok := statMap[statName]
	if !ok {
		log.Infof("No '%s' in response:", statName)
		return nil
	}
	return value
}

// scrapeServerStat is an utility to get a server statistic from a map.
func (pbe *PromBind9Exporter) scrapeServerStat(statMap map[string]interface{}, statName string) (map[string]float64, error) {
	storageMap := make(map[string]float64)

	statIfc := getStat(statMap, statName)
	if statIfc != nil {
		stats, ok := statIfc.(map[string]interface{})
		if !ok {
			return nil, pkgerrors.Errorf("problem casting '%s' interface", statName)
		}
		for labelName, labelValueIfc := range stats {
			// get value
			labelValue, ok := labelValueIfc.(float64)
			if !ok {
				continue
			}
			// store stat value
			storageMap[labelName] = labelValue
		}
	}
	return storageMap, nil
}

// scrapeTimeStats stores time related statistics from statMap.
func (pbe *PromBind9Exporter) scrapeTimeStats(statMap map[string]interface{}) (err error) {
	var timeVal time.Time
	var timeStr string

	// boot_time_seconds
	timeStr = getStat(statMap, "boot-time").(string)
	timeVal, err = time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return pkgerrors.Errorf("problem parsing time %+s: %+v", timeStr, err)
	}
	pbe.stats.BootTime = timeVal
	// config_time_seconds
	timeStr = getStat(statMap, "config-time").(string)
	timeVal, err = time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return pkgerrors.Errorf("problem parsing time %+s: %+v", timeStr, err)
	}
	pbe.stats.ConfigTime = timeVal
	// current_time_seconds
	timeStr = getStat(statMap, "current-time").(string)
	timeVal, err = time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return pkgerrors.Errorf("problem parsing time %+s: %+v", timeStr, err)
	}
	pbe.stats.CurrentTime = timeVal

	return nil
}

func (pbe *PromBind9Exporter) scrapeViewStats(viewName string, viewStatsIfc interface{}) {
	pbe.initViewStats(viewName)

	viewStats, ok := viewStatsIfc.(map[string]interface{})
	if !ok {
		log.Errorf("Problem casting viewStatsIfc: %+v", viewStatsIfc)
		return
	}

	// Parse resolver.
	resolverIfc, ok := viewStats["resolver"]
	if !ok {
		log.Infof("No 'resolver' in viewStats: %+v", viewStats)
		return
	}
	resolver, ok := resolverIfc.(map[string]interface{})
	if !ok {
		log.Errorf("Problem casting resolverIfc: %+v", resolverIfc)
		return
	}

	// Parse stats.
	statsIfc, ok := resolver["stats"]
	if !ok {
		log.Infof("No 'stats' in resolver: %+v", resolver)
		return
	}
	resolverStats, ok := statsIfc.(map[string]interface{})
	if !ok {
		log.Errorf("Problem casting statsIfc: %+v", statsIfc)
		return
	}

	// resolver_dnssec_validation_errors_total
	// resolver_dnssec_validation_success_total
	for statName, statValueIfc := range resolverStats {
		// get stat value
		statValue, ok := statValueIfc.(float64)
		if !ok {
			log.Errorf("Problem casting statValue: %+v", statValueIfc)
			continue
		}
		// store stat value
		pbe.stats.Views[viewName].ResolverStats[statName] = statValue
	}

	// Parse qtypes.
	qtypesIfc, ok := resolver["qtypes"]
	if !ok {
		log.Infof("No 'qtypes' in resolver: %+v", resolver)
		return
	}
	qtypes, ok := qtypesIfc.(map[string]interface{})
	if !ok {
		log.Errorf("Problem casting qtypesIfc: %+v", qtypesIfc)
		return
	}

	// resolver_queries_total
	for qtype, statValueIfc := range qtypes {
		// get stat value
		statValue, ok := statValueIfc.(float64)
		if !ok {
			log.Errorf("Problem casting statValue: %+v", statValueIfc)
			continue
		}
		// store stat value
		pbe.stats.Views[viewName].ResolverQtypes[qtype] = statValue
	}

	// Parse cache.
	cacheIfc, ok := resolver["cache"]
	if !ok {
		log.Infof("No 'cachestats' in resolver: %+v", resolver)
		return
	}
	cacheRRsets, ok := cacheIfc.(map[string]interface{})
	if !ok {
		log.Errorf("Problem casting cacheIfc: %+v", cacheIfc)
		return
	}

	// resolver_cache_rrsets
	for statName, statValueIfc := range cacheRRsets {
		// get stat value
		statValue, ok := statValueIfc.(float64)
		if !ok {
			log.Errorf("Problem casting statValue: %+v", statValueIfc)
			continue
		}
		// store stat value
		pbe.stats.Views[viewName].ResolverCache[statName] = statValue
	}

	// Parse cachestats.
	cachestatsIfc, ok := resolver["cachestats"]
	if !ok {
		log.Infof("No 'cachestats' in resolver: %+v", resolver)
		return
	}
	cachestats, ok := cachestatsIfc.(map[string]interface{})
	if !ok {
		log.Errorf("Problem casting cachestatsIfc: %+v", cachestatsIfc)
		return
	}

	// resolver_cache_hit_ratio
	// resolver_cache_hits
	// resolver_cache_misses
	// resolver_query_hit_ratio
	// resolver_query_hits
	// resolver_query_misses
	var cacheHits float64
	var cacheMisses float64
	var queryHits float64
	var queryMisses float64
	for statName, statValueIfc := range cachestats {
		// get stat value
		statValue, ok := statValueIfc.(float64)
		if !ok {
			log.Errorf("Problem casting statValue: %+v", statValueIfc)
			continue
		}
		switch statName {
		case "CacheHits":
			cacheHits = statValue
		case "CacheMisses":
			cacheMisses = statValue
		case "QueryHits":
			queryHits = statValue
		case "QueryMisses":
			queryMisses = statValue
		}

		// store stat value
		pbe.stats.Views[viewName].ResolverCachestats[statName] = statValue
	}

	total := cacheHits + cacheMisses
	if total > 0 {
		pbe.stats.Views[viewName].ResolverCachestats["CacheHitRatio"] = cacheHits / total
	}
	total = queryHits + queryMisses
	if total > 0 {
		pbe.stats.Views[viewName].ResolverCachestats["QueryHitRatio"] = queryHits / total
	}
}

// setDaemonStats stores the stat values from a daemon in the proper prometheus object.
func (pbe *PromBind9Exporter) setDaemonStats(rspIfc interface{}) (ret error) {
	rsp, ok := rspIfc.(map[string]interface{})
	if !ok {
		return pkgerrors.Errorf("problem casting rspIfc: %+v", rspIfc)
	}

	// boot_time_seconds
	// config_time_seconds
	// current_time_seconds
	err := pbe.scrapeTimeStats(rsp)
	if err != nil {
		return err
	}

	// incoming_queries_total
	pbe.stats.IncomingQueries, err = pbe.scrapeServerStat(rsp, "qtypes")
	if err != nil {
		return pkgerrors.Errorf("problem parsing 'qtypes': %+v", err)
	}
	// incoming_requests_total
	pbe.stats.IncomingRequests, err = pbe.scrapeServerStat(rsp, "opcodes")
	if err != nil {
		return pkgerrors.Errorf("problem parsing 'opcodes': %+v", err)
	}

	// query_duplicates_total
	// query_errors_total
	// query_recursion_total
	// recursive_clients
	// zone_transfer_failure_total
	// zone_transfer_rejected_total
	// zone_transfer_success_total
	pbe.stats.NsStats, err = pbe.scrapeServerStat(rsp, "nsstats")
	if err != nil {
		return pkgerrors.Errorf("problem parsing 'nsstats': %+v", err)
	}

	// tasks_running
	// worker_threads
	pbe.stats.TaskMgr, err = pbe.scrapeServerStat(rsp, "taskmgr")
	if err != nil {
		return pkgerrors.Errorf("problem parsing 'nsstats': %+v", err)
	}

	// Parse traffic stats.
	trafficIfc, ok := rsp["traffic"]
	if !ok {
		return pkgerrors.Errorf("no 'traffic' in response: %+v", rsp)
	}
	traffic, ok := trafficIfc.(map[string]interface{})
	if !ok {
		return pkgerrors.Errorf("problem casting trafficIfc: %+v", trafficIfc)
	}
	trafficMap := make(map[string]PromBind9TrafficStats)
	for trafficName, trafficStatsIfc := range traffic {
		sizeCounts := make(map[string]float64)
		trafficStats, ok := trafficStatsIfc.(map[string]interface{})
		if !ok {
			return pkgerrors.Errorf("problem casting '%s' interface", trafficName)
		}
		for labelName, labelValueIfc := range trafficStats {
			// get value
			labelValue, ok := labelValueIfc.(float64)
			if !ok {
				continue
			}
			// store stat value
			sizeCounts[labelName] = labelValue
		}
		trafficMap[trafficName] = PromBind9TrafficStats{
			SizeCount: sizeCounts,
		}
	}
	pbe.stats.TrafficStats = trafficMap

	// Parse views.
	viewsIfc, ok := rsp["views"]
	if !ok {
		return pkgerrors.Errorf("no 'views' in response: %+v", rsp)
	}

	views := viewsIfc.(map[string]interface{})
	if !ok {
		return pkgerrors.Errorf("problem casting viewsIfc: %+v", viewsIfc)
	}

	for viewName, viewStatsIfc := range views {
		pbe.scrapeViewStats(viewName, viewStatsIfc)
	}
	return nil
}

// collectStats collects stats from all bind9 apps.
func (pbe *PromBind9Exporter) collectStats() (bind9Pid int32, lastErr error) {
	pbe.up = 0

	// go through all bind9 apps discovered by monitor and query them for stats
	apps := pbe.AppMonitor.GetApps()
	for _, app := range apps {
		// ignore non-bind9 apps
		if app.GetBaseApp().Type != AppTypeBind9 {
			continue
		}
		bind9Pid = app.GetBaseApp().Pid

		// get stats from named
		sap, err := getAccessPoint(app, AccessPointStatistics)
		if err != nil {
			lastErr = err
			log.WithError(err).Error("Problem getting stats from BIND 9, bad access statistics point")
			continue
		}
		response, rspIfc, err := pbe.HTTPClient.createRequest(sap.Address, sap.Port).getRawStats()
		if err != nil {
			lastErr = err
			log.Errorf("Problem getting stats from BIND 9: %+v", err)
			continue
		}

		// Error HTTP response received.
		if response.IsError() {
			errorText := fmt.Sprintf("BIND9 stats returned error status code with message: %s", response.String())
			lastErr = pkgerrors.New(errorText)
			log.WithFields(log.Fields{
				"StatusCode": response.StatusCode(),
			}).Error(errorText)
			continue
		}

		// parse response
		err = pbe.setDaemonStats(rspIfc)
		if err != nil {
			lastErr = err
			log.Errorf("Cannot get stat from daemon: %+v", err)
			continue
		}

		pbe.up = 1
	}

	return bind9Pid, lastErr
}

// initViewStats initializes the maps for storing metrics.
func (pbe *PromBind9Exporter) initViewStats(viewName string) {
	_, ok := pbe.stats.Views[viewName]
	if !ok {
		resolverCache := make(map[string]float64)
		resolverCachestats := make(map[string]float64)
		resolverQtypes := make(map[string]float64)
		resolverStats := make(map[string]float64)

		pbe.stats.Views[viewName] = PromBind9ViewStats{
			ResolverCache:      resolverCache,
			ResolverCachestats: resolverCachestats,
			ResolverQtypes:     resolverQtypes,
			ResolverStats:      resolverStats,
		}
	}
}
