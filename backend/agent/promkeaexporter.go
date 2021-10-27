package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	storkutil "isc.org/stork/util"
)

// Parsed subnet list from Kea `subnet4-list` and `subnet6-list` response.
type SubnetList map[int]string

// Constructor of the SubnetList type.
func NewSubnetList() SubnetList {
	return make(SubnetList)
}

// UnmarshalJSON implements json.Unmarshaler. It unpacks the Kea response
// to map.
func (l *SubnetList) UnmarshalJSON(b []byte) error {
	// JSON structures of Kea `subnet4-list` and `subnet6-list` response.
	type SubnetListJSONArgumentsSubnet struct {
		ID     int
		Subnet string
	}

	type SubnetListJSONArguments struct {
		Subnets []SubnetListJSONArgumentsSubnet
	}

	type SubnetListJSON struct {
		Result    int
		Text      *string
		Arguments *SubnetListJSONArguments
	}

	// Unmarshal must be called with existing instance.
	if l == nil {
		return pkgerrors.New("receiver is nil")
	}

	// Standard unmarshal
	var dhcpLabelsJSONs []SubnetListJSON
	err := json.Unmarshal(b, &dhcpLabelsJSONs)
	// Parse JSON content
	if err != nil {
		return pkgerrors.Wrap(err, "problem with parsing DHCP4 labels from kea")
	}

	if len(dhcpLabelsJSONs) == 0 {
		return pkgerrors.New("empty JSON list")
	}

	dhcpLabelsJSON := dhcpLabelsJSONs[0]

	// Hook not installed. Return empty mapping
	if dhcpLabelsJSON.Result == 2 {
		return nil
	}

	if dhcpLabelsJSON.Result != 0 {
		reason := "unknown"
		if dhcpLabelsJSON.Text != nil {
			reason = *dhcpLabelsJSON.Text
		}
		return pkgerrors.Errorf("problem with content of DHCP4 labels response from kea: %s", reason)
	}

	// Result is OK, parse the mapping content

	// No entries
	if dhcpLabelsJSON.Arguments == nil {
		return nil
	}

	for _, subnet := range dhcpLabelsJSON.Arguments.Subnets {
		(*l)[subnet.ID] = subnet.Subnet
	}

	return nil
}

// JSON get-all-statistic response returned from Kea CA.
type GetAllStatisticsResponse struct {
	Dhcp4 map[string]GetAllStatisticResponseItemValue
	Dhcp6 map[string]GetAllStatisticResponseItemValue
}

// JSON get-all-statistic single value response returned from Kea CA.
type GetAllStatisticResponseItemValue struct {
	Value float64
	// Timestamp is not used.
	Timestamp *string
}

// UnmarshalJSON implements json.Unmarshaler. It unpacks the Kea response
// to simpler Go-friendly form.
func (r *GetAllStatisticsResponse) UnmarshalJSON(b []byte) error {
	// Raw structures - corresponding to real received JSON.
	type ResponseRawItem struct {
		Result int64
		Text   *string
		// In Go you cannot describe the JSON array with mixed-type items.
		Arguments *map[string][][]interface{}
	}
	type ResponseRaw = []ResponseRawItem

	// Standard GO unmarshal
	var obj ResponseRaw
	err := json.Unmarshal(b, &obj)
	if err != nil {
		return pkgerrors.Wrapf(err, "failed to parse responses from Kea")
	}

	// Retrieve values of mixed-type arrays.
	// Unpack the complex structure to simpler form.
	for daemonIdx, item := range obj {
		if item.Result != 0 {
			if item.Text != nil {
				text := *item.Text
				if strings.Contains(text, "server is likely to be offline") || strings.Contains(text, "forwarding socket is not configured for the server type") {
					log.Warnf("problem with connecting to dhcp daemon: %s", text)
					continue
				}
				return pkgerrors.Errorf("response result from Kea != 0: %d, text: %s", item.Result, text)
			}
			return pkgerrors.Errorf("response result from Kea != 0: %d", item.Result)
		}

		// daemon 0 is dhcp4, 1 is dhcp6
		isDhcp4 := daemonIdx == 0
		var statMap map[string]GetAllStatisticResponseItemValue
		if isDhcp4 {
			r.Dhcp4 = make(map[string]GetAllStatisticResponseItemValue)
			statMap = r.Dhcp4
		} else {
			r.Dhcp6 = make(map[string]GetAllStatisticResponseItemValue)
			statMap = r.Dhcp6
		}

		if item.Arguments == nil {
			return pkgerrors.Errorf("problem with arguments: %+v", item)
		}

		for statName, statValueOutterList := range *item.Arguments {
			if len(statValueOutterList) == 0 {
				log.Errorf("empty list of stat values")
				continue
			}
			statValueInnerList := statValueOutterList[0]

			if len(statValueInnerList) == 0 {
				log.Errorf("empty list of stat values")
				continue
			}

			statValue, ok := statValueInnerList[0].(float64)
			if !ok {
				log.Errorf("problem with casting statValueInnerList[0]: %+v", statValueInnerList[0])
				continue
			}

			var statTimestamp *string
			if len(statValueInnerList) > 1 {
				castedTimestamp, ok := statValueInnerList[1].(string)
				if ok {
					statTimestamp = &castedTimestamp
				}
			}

			item := GetAllStatisticResponseItemValue{
				Value:     statValue,
				Timestamp: statTimestamp,
			}

			statMap[statName] = item
		}
	}
	return nil
}

// Stats descriptor that holds reference to prometheus stats
// and its 'operation' label.
type statDescr struct {
	Stat      *prometheus.GaugeVec
	Operation string
}

// Main structure for Prometheus Kea Exporter. It holds its settings,
// references to app monitor, CA client, HTTP server, and main loop
// controlling elements like ticker, and mappings between kea stats
// names to prometheus stats.
type PromKeaExporter struct {
	Settings *cli.Context

	AppMonitor AppMonitor
	HTTPClient *HTTPClient
	HTTPServer *http.Server

	Ticker        *time.Ticker
	DoneCollector chan bool
	Wg            *sync.WaitGroup

	Registry     *prometheus.Registry
	PktStatsMap  map[string]statDescr
	Adr4StatsMap map[string]*prometheus.GaugeVec
	Adr6StatsMap map[string]*prometheus.GaugeVec
}

// Create new Prometheus Kea Exporter.
func NewPromKeaExporter(settings *cli.Context, appMonitor AppMonitor) *PromKeaExporter {
	pke := &PromKeaExporter{
		Settings:      settings,
		AppMonitor:    appMonitor,
		HTTPClient:    NewHTTPClient(settings.Bool("skip-tls-cert-verification")),
		DoneCollector: make(chan bool),
		Wg:            &sync.WaitGroup{},
		Registry:      prometheus.NewRegistry(),
	}

	factory := promauto.With(pke.Registry)

	// packets dhcp4
	packets4SentTotal := factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp4",
		Name:      "packets_sent_total",
		Help:      "Packets sent",
	}, []string{"operation"})
	packets4ReceivedTotal := factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp4",
		Name:      "packets_received_total",
		Help:      "Packets received",
	}, []string{"operation"})

	// packets dhcp6
	packets6SentTotal := factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "packets_sent_total",
		Help:      "Packets sent",
	}, []string{"operation"})
	packets6ReceivedTotal := factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "packets_received_total",
		Help:      "Packets received",
	}, []string{"operation"})
	packets4o6SentTotal := factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "packets_sent_dhcp4_total",
		Help:      "DHCPv4-over-DHCPv6 Packets received",
	}, []string{"operation"})
	packets4o6ReceivedTotal := factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "packets_received_dhcp4_total",
		Help:      "DHCPv4-over-DHCPv6 Packets received",
	}, []string{"operation"})

	pktStatsMap := make(map[string]statDescr)

	// packets4ReceivedTotal
	pktStatsMap["pkt4-nak-received"] = statDescr{Stat: packets4ReceivedTotal, Operation: "nak"}
	pktStatsMap["pkt4-ack-received"] = statDescr{Stat: packets4ReceivedTotal, Operation: "ack"}
	pktStatsMap["pkt4-decline-received"] = statDescr{Stat: packets4ReceivedTotal, Operation: "decline"}
	pktStatsMap["pkt4-discover-received"] = statDescr{Stat: packets4ReceivedTotal, Operation: "discover"}
	pktStatsMap["pkt4-inform-received"] = statDescr{Stat: packets4ReceivedTotal, Operation: "inform"}
	pktStatsMap["pkt4-offer-received"] = statDescr{Stat: packets4ReceivedTotal, Operation: "offer"}
	pktStatsMap["pkt4-receive-drop"] = statDescr{Stat: packets4ReceivedTotal, Operation: "drop"}
	pktStatsMap["pkt4-parse-failed"] = statDescr{Stat: packets4ReceivedTotal, Operation: "parse-failed"}
	pktStatsMap["pkt4-release-received"] = statDescr{Stat: packets4ReceivedTotal, Operation: "release"}
	pktStatsMap["pkt4-request-received"] = statDescr{Stat: packets4ReceivedTotal, Operation: "request"}
	pktStatsMap["pkt4-unknown-received"] = statDescr{Stat: packets4ReceivedTotal, Operation: "unknown"}

	// packets4SentTotal
	pktStatsMap["pkt4-offer-sent"] = statDescr{Stat: packets4SentTotal, Operation: "offer"}
	pktStatsMap["pkt4-nak-sent"] = statDescr{Stat: packets4SentTotal, Operation: "nak"}
	pktStatsMap["pkt4-ack-sent"] = statDescr{Stat: packets4SentTotal, Operation: "ack"}

	// packets6ReceivedTotal
	pktStatsMap["pkt6-receive-drop"] = statDescr{Stat: packets6ReceivedTotal, Operation: "drop"}
	pktStatsMap["pkt6-parse-failed"] = statDescr{Stat: packets6ReceivedTotal, Operation: "parse-failed"}
	pktStatsMap["pkt6-solicit-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "solicit"}
	pktStatsMap["pkt6-advertise-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "advertise"}
	pktStatsMap["pkt6-request-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "request"}
	pktStatsMap["pkt6-reply-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "reply"}
	pktStatsMap["pkt6-renew-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "renew"}
	pktStatsMap["pkt6-rebind-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "rebind"}
	pktStatsMap["pkt6-release-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "release"}
	pktStatsMap["pkt6-decline-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "decline"}
	pktStatsMap["pkt6-infrequest-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "infrequest"}
	pktStatsMap["pkt6-unknown-received"] = statDescr{Stat: packets6ReceivedTotal, Operation: "unknown"}

	// packets6SentTotal
	pktStatsMap["pkt6-advertise-sent"] = statDescr{Stat: packets6SentTotal, Operation: "advertise"}
	pktStatsMap["pkt6-reply-sent"] = statDescr{Stat: packets6SentTotal, Operation: "reply"}

	// packets4o6SentTotal & packets4o6ReceivedTotal
	pktStatsMap["pkt6-dhcpv4-response-sent"] = statDescr{Stat: packets4o6SentTotal, Operation: "response"}
	pktStatsMap["pkt6-dhcpv4-query-received"] = statDescr{Stat: packets4o6ReceivedTotal, Operation: "query"}
	pktStatsMap["pkt6-dhcpv4-response-received"] = statDescr{Stat: packets4o6ReceivedTotal, Operation: "response"}

	// addresses dhcp4
	adr4StatsMap := make(map[string]*prometheus.GaugeVec)
	adr4StatsMap["assigned-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp4",
		Name:      "addresses_assigned_total",
		Help:      "Assigned addresses",
	}, []string{"subnet"})
	adr4StatsMap["declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp4",
		Name:      "addresses_declined_total",
		Help:      "Declined counts",
	}, []string{"subnet"})
	adr4StatsMap["reclaimed-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp4",
		Name:      "addresses_declined_reclaimed_total",
		Help:      "Declined addresses that were reclaimed",
	}, []string{"subnet"})
	adr4StatsMap["reclaimed-leases"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp4",
		Name:      "addresses_reclaimed_total",
		Help:      "Expired addresses that were reclaimed",
	}, []string{"subnet"})
	adr4StatsMap["total-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp4",
		Name:      "addresses_total",
		Help:      "Size of subnet address pool",
	}, []string{"subnet"})
	adr4StatsMap["cumulative-assigned-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp4",
		Name:      "cumulative_addresses_assigned_total",
		Help:      "Cumulative number of assigned addresses since server startup",
	}, []string{"subnet"})

	// addresses dhcp6
	adr6StatsMap := make(map[string]*prometheus.GaugeVec)
	adr6StatsMap["total-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "na_total",
		Help:      "'Size of non-temporary address pool",
	}, []string{"subnet"})
	adr6StatsMap["assigned-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "na_assigned_total",
		Help:      "Assigned non-temporary addresses (IA_NA)",
	}, []string{"subnet"})
	adr6StatsMap["total-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "pd_total",
		Help:      "Size of prefix delegation pool",
	}, []string{"subnet"})
	adr6StatsMap["assigned-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "pd_assigned_total",
		Help:      "Assigned prefix delegations (IA_PD)",
	}, []string{"subnet"})
	adr6StatsMap["reclaimed-leases"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "addresses_reclaimed_total",
		Help:      "Expired addresses that were reclaimed",
	}, []string{"subnet"})
	adr6StatsMap["declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "addresses_declined_total",
		Help:      "Declined counts",
	}, []string{"subnet"})
	adr6StatsMap["reclaimed-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "addresses_declined_reclaimed_total",
		Help:      "Declined addresses that were reclaimed",
	}, []string{"subnet"})
	adr6StatsMap["cumulative-assigned-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "cumulative_nas_assigned_total",
		Help:      "Cumulative number of assigned NA addresses since server startup",
	}, []string{"subnet"})
	adr6StatsMap["cumulative-assigned-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "cumulative_pds_assigned_total",
		Help:      "Cumulative number of assigned PD prefixes since server startup",
	}, []string{"subnet"})

	pke.PktStatsMap = pktStatsMap
	pke.Adr4StatsMap = adr4StatsMap
	pke.Adr6StatsMap = adr6StatsMap

	// prepare http handler
	mux := http.NewServeMux()
	hdlr := promhttp.HandlerFor(pke.Registry, promhttp.HandlerOpts{})
	mux.Handle("/metrics", hdlr)
	pke.HTTPServer = &http.Server{
		Handler: mux,
	}

	return pke
}

// Start goroutine with main loop for collecting stats
// and http server for exposing them to Prometheus.
func (pke *PromKeaExporter) Start() {
	// set address for listening from config
	addrPort := net.JoinHostPort(pke.Settings.String("prometheus-kea-exporter-address"), strconv.Itoa(pke.Settings.Int("prometheus-kea-exporter-port")))
	pke.HTTPServer.Addr = addrPort

	log.Printf("Prometheus Kea Exporter listening on %s, stats pulling interval: %d seconds",
		addrPort, pke.Settings.Int("prometheus-kea-exporter-interval"))

	// start http server for metrics
	go func() {
		err := pke.HTTPServer.ListenAndServe()
		if err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Errorf("problem with serving Prometheus Kea Exporter: %s", err.Error())
		}
	}()

	// set ticker time for collecting loop from config
	pke.Ticker = time.NewTicker(time.Duration(pke.Settings.Int("prometheus-kea-exporter-interval")) * time.Second)

	// start collecting loop as goroutine and increment WaitGroup (which is used later
	// for stopping this goroutine)
	pke.Wg.Add(1)
	go pke.statsCollectorLoop()
}

// Shutdown exporter goroutines and unregister prometheus stats.
func (pke *PromKeaExporter) Shutdown() {
	log.Printf("Stopping Prometheus Kea Exporter")

	// stop http server
	if pke.HTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		pke.HTTPServer.SetKeepAlivesEnabled(false)
		if err := pke.HTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("Could not gracefully shutdown the kea exporter: %v\n", err)
		}
	}

	// stop stats collector
	if pke.Ticker != nil {
		pke.Ticker.Stop()
		pke.DoneCollector <- true
		pke.Wg.Wait()
	}

	// unregister kea counters from prometheus framework
	pke.Registry.Unregister(pke.PktStatsMap["pkt4-nak-received"].Stat)
	pke.Registry.Unregister(pke.PktStatsMap["pkt4-offer-sent"].Stat)
	pke.Registry.Unregister(pke.PktStatsMap["pkt6-receive-drop"].Stat)
	pke.Registry.Unregister(pke.PktStatsMap["pkt6-advertise-sent"].Stat)
	pke.Registry.Unregister(pke.PktStatsMap["pkt6-dhcpv4-response-sent"].Stat)
	pke.Registry.Unregister(pke.PktStatsMap["pkt6-dhcpv4-query-received"].Stat)
	for _, stat := range pke.Adr4StatsMap {
		pke.Registry.Unregister(stat)
	}
	for _, stat := range pke.Adr6StatsMap {
		pke.Registry.Unregister(stat)
	}

	log.Printf("Stopped Prometheus Kea Exporter")
}

// Main loop for collecting stats periodically.
func (pke *PromKeaExporter) statsCollectorLoop() {
	defer pke.Wg.Done()
	for {
		select {
		// every N seconds do stats collection from all kea and its active daemons
		case <-pke.Ticker.C:
			err := pke.collectStats()
			if err != nil {
				log.Errorf("some errors were encountered while collecting stats from kea: %+v", err)
			}
		// wait for done signal from shutdown function
		case <-pke.DoneCollector:
			return
		}
	}
}

// setDaemonStats stores the stat values from a daemon in the proper prometheus object.
func (pke *PromKeaExporter) setDaemonStats(isDhcp4 bool, response map[string]GetAllStatisticResponseItemValue, ignoredStats map[string]bool, subnetLabelMap map[int]string) {
	for statName, statEntry := range response {
		// skip ignored stats
		if ignoredStats[statName] {
			continue
		}

		// store stat value in proper prometheus object
		switch {
		case strings.HasPrefix(statName, "pkt"):
			// if this is pkt stat
			statDescr, ok := pke.PktStatsMap[statName]
			if ok {
				statDescr.Stat.With(prometheus.Labels{"operation": statDescr.Operation}).Set(statEntry.Value)
			} else {
				log.Printf("encountered unsupported stat: %s", statName)
			}
		case strings.HasPrefix(statName, "subnet["):
			// if this is address per subnet stat
			re := regexp.MustCompile(`subnet\[(\d+)\]\.(.+)`)
			matches := re.FindStringSubmatch(statName)
			subnetIDRaw := matches[1]
			metricName := matches[2]

			subnetID, err := strconv.Atoi(subnetIDRaw)
			subnetName := subnetIDRaw
			if err == nil {
				label, ok := subnetLabelMap[subnetID]
				if ok {
					subnetName = label
				}
			}

			var stat *prometheus.GaugeVec
			var ok bool
			if isDhcp4 {
				stat, ok = pke.Adr4StatsMap[metricName]
			} else {
				stat, ok = pke.Adr6StatsMap[metricName]
			}

			if ok {
				stat.With(prometheus.Labels{"subnet": subnetName}).Set(statEntry.Value)
			} else {
				log.Printf("encountered unsupported stat: %s", metricName)
			}
		default:
			log.Printf("encountered unsupported stat: %s", statName)
		}
	}
}

// Collect stats from all Kea apps.
func (pke *PromKeaExporter) collectStats() error {
	var lastErr error
	// these stats are ignored as they are estimated by summing sub-stats (like ack, nak, etc)
	ignoredStats := map[string]bool{
		"pkt4-received": true,
		"pkt4-sent":     true,
		"pkt6-received": true,
		"pkt6-sent":     true,
	}

	// Request to kea dhcp daemons for getting all stats. Both v4 and v6 is queried because
	// here we do not have knowledge which are active.
	requestData := `{
             "command":"statistic-get-all",
             "service":["dhcp4", "dhcp6"],
             "arguments": {}
        }`

	// Request to subnet labels. The above query returns only sequential, numeric IDs that aren't human-friendly.
	requestDhcp4Labels := `{
			"command":"subnet4-list",
			"service":["dhcp4"],
			"arguments": {}
		}`
	requestDhcp6Labels := `{
		"command":"subnet6-list",
		"service":["dhcp6"],
		"arguments": {}
	}`

	// go through all kea apps discovered by monitor and query them for stats
	apps := pke.AppMonitor.GetApps()
	for _, app := range apps {
		// ignore non-kea apps
		if app.GetBaseApp().Type != AppTypeKea {
			continue
		}

		// get stats from kea
		ctrl, err := getAccessPoint(app, AccessPointControl)
		if err != nil {
			lastErr = err
			log.Errorf("problem with getting stats from kea, bad Kea access control point: %+v", err)
			continue
		}

		// Fetching DHCP subnet prefixes. It may fail if Kea doesn't support
		// required commands.
		dhcp4Labels := NewSubnetList()
		responseDhcp4Labels, err := pke.sendCommandToKeaCA(ctrl, requestDhcp4Labels)
		if err == nil {
			err = json.Unmarshal(responseDhcp4Labels, &dhcp4Labels)
			if err != nil {
				log.Errorf("problem with parsing DHCP4 labels from kea: %+v", err)
			}
		}

		dhcp6Labels := NewSubnetList()
		responseDhcp6Labels, err := pke.sendCommandToKeaCA(ctrl, requestDhcp6Labels)
		if err == nil {
			err = json.Unmarshal(responseDhcp6Labels, &dhcp6Labels)
			if err != nil {
				log.Errorf("problem with parsing DHCP6 labels from kea: %+v", err)
			}
		}

		// Fetching statistics
		responseData, err := pke.sendCommandToKeaCA(ctrl, requestData)
		if err != nil {
			log.Errorf("problem with fetching stats from kea: %+v", err)
			continue
		}

		// parse response
		var response GetAllStatisticsResponse
		err = json.Unmarshal(responseData, &response)
		if err != nil {
			lastErr = err
			log.Errorf("failed to parse responses from Kea: %s", err)
			continue
		}

		// Go though responses from daemons (it can have none or some responses from dhcp4/dhcp6)
		// and store collected stats in Prometheus structures.
		if response.Dhcp4 != nil {
			pke.setDaemonStats(true, response.Dhcp4, ignoredStats, dhcp4Labels)
		}
		if response.Dhcp6 != nil {
			pke.setDaemonStats(false, response.Dhcp6, ignoredStats, dhcp6Labels)
		}
	}
	return lastErr
}

// Send any command to Kea CA and returns body content.
func (pke *PromKeaExporter) sendCommandToKeaCA(ctrl *AccessPoint, request string) ([]byte, error) {
	caURL := storkutil.HostWithPortURL(ctrl.Address, ctrl.Port, ctrl.UseSecureProtocol)
	httpRsp, err := pke.HTTPClient.Call(caURL, bytes.NewBuffer([]byte(request)))
	if err != nil {
		return nil, pkgerrors.Wrap(err, "problem with getting stats from kea")
	}
	body, err := ioutil.ReadAll(httpRsp.Body)
	httpRsp.Body.Close()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "problem with reading stats response from kea")
	}
	return body, nil
}
