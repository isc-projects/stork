package agent

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	keactrl "isc.org/stork/appctrl/kea"
)

// Parsed subnet list from Kea `subnet4-list` and `subnet6-list` response.
type SubnetList map[int]string

// Constructor of the SubnetList type.
func NewSubnetList() SubnetList {
	return make(SubnetList)
}

// JSON structures of Kea `subnet4-list` and `subnet6-list` response.
type subnetListJSONArgumentsSubnet struct {
	ID     int
	Subnet string
}

type subnetListJSONArguments struct {
	Subnets []subnetListJSONArgumentsSubnet
}

type subnetListJSON struct {
	keactrl.ResponseHeader
	Arguments *subnetListJSONArguments
}

// UnmarshalJSON implements json.Unmarshaler. It unpacks the Kea response
// to map.
func (l *SubnetList) UnmarshalJSON(b []byte) error {
	// Unmarshal must be called with existing instance.
	if *l == nil {
		*l = NewSubnetList()
	}

	// Standard unmarshal
	var dhcpLabelsJSONs []subnetListJSON
	err := json.Unmarshal(b, &dhcpLabelsJSONs)
	// Parse JSON content
	if err != nil {
		return errors.Wrap(err, "problem parsing DHCP4 labels from Kea")
	}

	if len(dhcpLabelsJSONs) == 0 {
		return errors.New("empty JSON list")
	}

	dhcpLabelsJSON := dhcpLabelsJSONs[0]

	// Check the response error.
	if err := dhcpLabelsJSON.GetError(); err != nil {
		if errors.As(err, &keactrl.UnsupportedOperationKeaError{}) {
			// Hook not installed. Return empty mapping
			return nil
		}
		return errors.WithMessage(err, "problem with content of DHCP labels response from Kea")
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
// There is a response entry for each service. The order of entries is the
// same as the order of services in the request.
type GetAllStatisticsResponse []map[string]GetAllStatisticResponseItemValue

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
		keactrl.ResponseHeader
		// In Go you cannot describe the JSON array with mixed-type items.
		Arguments *map[string][][]interface{}
	}
	type ResponseRaw = []ResponseRawItem

	// Standard GO unmarshal
	var obj ResponseRaw
	err := json.Unmarshal(b, &obj)
	if err != nil {
		outerError := errors.Wrapf(err, "failed to parse responses from Kea")
		// Kea sends the error as a single item, not array,
		var singleItem ResponseRawItem
		err = json.Unmarshal(b, &singleItem)
		if err != nil {
			return outerError
		}

		err = singleItem.GetError()
		if err != nil {
			return err
		}
		return errors.Errorf("cannot parse response from Kea")
	}

	// Prepare the result. It is filled with nils to indicate that the daemon
	// is not responding.
	statMaps := make([]map[string]GetAllStatisticResponseItemValue, len(obj))

	// Retrieve values of mixed-type arrays.
	// Unpack the complex structure to simpler form.
	for i, item := range obj {
		// Indicates the situation when the daemon is active but the statistics
		// endpoint doesn't return the data.
		areStatisticsUnavailable := false

		err := item.GetError()
		switch {
		case errors.As(err, &keactrl.ConnectivityIssueKeaError{}):
			log.WithError(err).Warn("Problem connecting to dhcp daemon")
			continue
		case errors.As(err, &keactrl.NumberOverflowKeaError{}):
			log.WithError(err).Warnf("Number overflow in the statistics data")
			areStatisticsUnavailable = true
		case err != nil:
			return err
		}

		statMap := make(map[string]GetAllStatisticResponseItemValue)
		statMaps[i] = statMap

		if areStatisticsUnavailable {
			// The empty but not-nil stat map indicates that the daemon is
			// responding to the requests.
			continue
		}

		if item.Arguments == nil {
			return errors.Errorf("problem with arguments: %+v", item)
		}

		for statName, statValueOuterList := range *item.Arguments {
			if len(statValueOuterList) == 0 {
				log.Errorf("Empty list of stat values")
				continue
			}
			statValueInnerList := statValueOuterList[0]

			if len(statValueInnerList) == 0 {
				log.Errorf("Empty list of stat values")
				continue
			}

			statValue, ok := statValueInnerList[0].(float64)
			if !ok {
				log.Errorf("Problem casting statValueInnerList[0]: %+v", statValueInnerList[0])
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

	*r = statMaps
	return nil
}

// Stats descriptor that holds reference to prometheus stats
// and its 'operation' label.
type statisticDescriptor struct {
	Stat      *prometheus.GaugeVec
	Operation string
}

// subnetPrefixLookup is the interface that wraps the subnet prefix lookup methods.
type subnetPrefixLookup interface {
	// Returns the subnet prefix based on the subnet ID and IP family.
	// If the prefix isn't available returns the empty string and false value.
	getPrefix(subnetID int) (string, bool)
	// Sets the IP family to use during lookup (4 or 6).
	setFamily(int8)
}

// An object that implements this interface can send requests to the Kea CA.
type keaCommandSender interface {
	sendCommandRaw(command []byte) ([]byte, error)
}

// Subnet prefix lookup that fetches the subnet prefixes only if necessary.
// The subnet prefixes are fetched on the first call getPrefix() for an IP
// family. The results are cached; no more requests are made until IP family
// change. Therefore, the lifetime of instances should be short to avoid
// out-of-date prefixes in a cache.
type lazySubnetPrefixLookup struct {
	sender keaCommandSender
	// Cached subnet prefixes from current family.
	cachedPrefixes SubnetList
	// Indicates that the subnet prefixes were fetched for current family.
	cached bool
	// Family to use during lookups.
	family int8
}

// Constructs the lazySubnetPrefixLookup instance. It accepts the Kea CA request sender.
func newLazySubnetPrefixLookup(sender keaCommandSender) subnetPrefixLookup {
	return &lazySubnetPrefixLookup{sender, nil, false, 4}
}

// Fetches the subnet prefixes from Kea CA and stores the response in a cache.
// If any error occurs or prefixes are unavailable then the cache for specific
// family is set to nil. Returns fetched subnet prefixes.
// Family should be 4 or 6.
func (l *lazySubnetPrefixLookup) fetchAndCachePrefixes() SubnetList {
	// Request to subnet prefixes.
	var request string
	if l.family == 4 {
		request = `{
			"command":"subnet4-list",
			"service":["dhcp4"],
			"arguments": {}
		}`
	} else {
		request = `{
			"command":"subnet6-list",
			"service":["dhcp6"],
			"arguments": {}
		}`
	}

	response, err := l.sender.sendCommandRaw([]byte(request))
	var target SubnetList
	if err == nil {
		err = json.Unmarshal(response, &target)
		if err != nil {
			log.WithError(err).Errorf(
				"Problem parsing DHCPv%d prefixes from Kea",
				l.family,
			)
		}
	}

	// Cache results
	l.cachedPrefixes = target
	l.cached = true
	return target
}

// Returns the subnet prefix for specific subnet ID and IP family (4 or 6).
// If the prefix is unavailable then it returns empty string and false.
func (l *lazySubnetPrefixLookup) getPrefix(subnetID int) (string, bool) {
	prefixes := l.cachedPrefixes
	if !l.cached {
		prefixes = l.fetchAndCachePrefixes()
	}
	if prefixes == nil {
		return "", false
	}

	prefix, ok := prefixes[subnetID]
	return prefix, ok
}

// Sets the family used during prefix lookups.
func (l *lazySubnetPrefixLookup) setFamily(family int8) {
	l.family = family
	l.cached = false
}

// Main structure for Prometheus Kea Exporter. It holds its settings,
// references to app monitor, CA client, HTTP server, and main loop
// controlling elements like ticker, and mappings between kea stats
// names to prometheus stats.
type PromKeaExporter struct {
	Host     string
	Port     int
	Interval time.Duration

	EnablePerSubnetStats bool

	AppMonitor AppMonitor
	HTTPServer *http.Server

	Ticker        *time.Ticker
	DoneCollector chan bool
	Wg            *sync.WaitGroup
	StartTime     time.Time

	Registry        *prometheus.Registry
	PktStatsMap     map[string]statisticDescriptor
	Adr4StatsMap    map[string]*prometheus.GaugeVec
	Adr6StatsMap    map[string]*prometheus.GaugeVec
	Global4StatMap  map[string]prometheus.Gauge
	Global6StatMap  map[string]prometheus.Gauge
	ExporterStatMap map[string]prometheus.Gauge

	// Set of the ignored stats as they are estimated by summing sub-stats
	// (like ack, nak, etc) or not-supported.
	ignoredStats map[string]bool
}

// Create new Prometheus Kea Exporter.
func NewPromKeaExporter(host string, port int, interval time.Duration, enablePerSubnetStats bool, appMonitor AppMonitor) *PromKeaExporter {
	pke := &PromKeaExporter{
		Host:                 host,
		Port:                 port,
		Interval:             interval,
		EnablePerSubnetStats: enablePerSubnetStats,
		AppMonitor:           appMonitor,
		DoneCollector:        make(chan bool),
		Wg:                   &sync.WaitGroup{},
		Registry:             prometheus.NewRegistry(),
		StartTime:            time.Now(),
		Adr4StatsMap:         nil,
		Adr6StatsMap:         nil,
		Global4StatMap:       nil,
		Global6StatMap:       nil,
		ignoredStats: map[string]bool{
			// Stats estimated by summing sub-stats.
			"pkt4-received": true,
			"pkt4-sent":     true,
			"pkt6-received": true,
			"pkt6-sent":     true,
		},
	}

	factory := promauto.With(pke.Registry)

	// stork agent internal stats
	const storkAgentNamespace = "storkagent"
	pke.ExporterStatMap = map[string]prometheus.Gauge{
		"uptime_seconds": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: storkAgentNamespace,
			Subsystem: "promkeaexporter",
			Name:      "uptime_seconds",
			Help:      "Uptime of the Prometheus Kea Exporter in seconds",
		}),
		"monitored_kea_apps": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: storkAgentNamespace,
			Subsystem: "appmonitor",
			Name:      "monitored_kea_apps_total",
			Help:      "Number of currently monitored Kea applications",
		}),
		"active_dhcp4_daemons": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: storkAgentNamespace,
			Subsystem: "promkeaexporter",
			Name:      "active_dhcp4_daemons_total",
			Help:      "Number of DHCPv4 daemons providing statistics",
		}),
		"active_dhcp6_daemons": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: storkAgentNamespace,
			Subsystem: "promkeaexporter",
			Name:      "active_dhcp6_daemons_total",
			Help:      "Number of DHCPv6 daemons providing statistics",
		}),
		"configured_dhcp4_daemons": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: storkAgentNamespace,
			Subsystem: "promkeaexporter",
			Name:      "configured_dhcp4_daemons_total",
			Help:      "Number of configured DHCPv4 daemons in Kea CA",
		}),
		"configured_dhcp6_daemons": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: storkAgentNamespace,
			Subsystem: "promkeaexporter",
			Name:      "configured_dhcp6_daemons_total",
			Help:      "Number of configured DHCPv6 daemons in Kea CA",
		}),
	}

	// global stats
	pke.Global4StatMap = map[string]prometheus.Gauge{
		"cumulative-assigned-addresses": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "global4_cumulative_addresses_assigned_total",
			Help:      "Cumulative number of assigned addresses since server startup from all subnets",
		}),
		"declined-addresses": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "global4_addresses_declined_total",
			Help:      "Declined counts from all subnets",
		}),
		"reclaimed-declined-addresses": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "global4_addresses_declined_reclaimed_total",
			Help:      "Declined addresses that were reclaimed for all subnets",
		}),
		"reclaimed-leases": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "global4_addresses_reclaimed_total",
			Help:      "Expired addresses that were reclaimed for all subnets",
		}),
	}

	pke.Global6StatMap = map[string]prometheus.Gauge{
		"declined-addresses": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "global6_addresses_declined_total",
			Help:      "Declined counts from all subnets",
		}),
		"cumulative-assigned-nas": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "global6_cumulative_nas_assigned_total",
			Help:      "Cumulative number of assigned NA addresses since server startup from all subnets",
		}),
		"cumulative-assigned-pds": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "global6_cumulative_pds_assigned_total",
			Help:      "Cumulative number of assigned PD prefixes since server startup",
		}),
		"reclaimed-declined-addresses": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "global6_addresses_declined_reclaimed_total",
			Help:      "Declined addresses that were reclaimed for all subnets",
		}),
		"reclaimed-leases": factory.NewGauge(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "global6_addresses_reclaimed_total",
			Help:      "Expired addresses that were reclaimed for all subnets",
		}),
	}

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
		Help:      "DHCPv4-over-DHCPv6 packets received",
	}, []string{"operation"})
	packets4o6ReceivedTotal := factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeKea,
		Subsystem: "dhcp6",
		Name:      "packets_received_dhcp4_total",
		Help:      "DHCPv4-over-DHCPv6 packets received",
	}, []string{"operation"})

	pktStatsMap := make(map[string]statisticDescriptor)

	// packets4ReceivedTotal
	pktStatsMap["pkt4-nak-received"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "nak"}
	pktStatsMap["pkt4-ack-received"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "ack"}
	pktStatsMap["pkt4-decline-received"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "decline"}
	pktStatsMap["pkt4-discover-received"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "discover"}
	pktStatsMap["pkt4-inform-received"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "inform"}
	pktStatsMap["pkt4-offer-received"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "offer"}
	pktStatsMap["pkt4-receive-drop"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "drop"}
	pktStatsMap["pkt4-parse-failed"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "parse-failed"}
	pktStatsMap["pkt4-release-received"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "release"}
	pktStatsMap["pkt4-request-received"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "request"}
	pktStatsMap["pkt4-unknown-received"] = statisticDescriptor{Stat: packets4ReceivedTotal, Operation: "unknown"}

	// packets4SentTotal
	pktStatsMap["pkt4-offer-sent"] = statisticDescriptor{Stat: packets4SentTotal, Operation: "offer"}
	pktStatsMap["pkt4-nak-sent"] = statisticDescriptor{Stat: packets4SentTotal, Operation: "nak"}
	pktStatsMap["pkt4-ack-sent"] = statisticDescriptor{Stat: packets4SentTotal, Operation: "ack"}

	// packets6ReceivedTotal
	pktStatsMap["pkt6-receive-drop"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "drop"}
	pktStatsMap["pkt6-parse-failed"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "parse-failed"}
	pktStatsMap["pkt6-solicit-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "solicit"}
	pktStatsMap["pkt6-advertise-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "advertise"}
	pktStatsMap["pkt6-request-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "request"}
	pktStatsMap["pkt6-reply-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "reply"}
	pktStatsMap["pkt6-renew-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "renew"}
	pktStatsMap["pkt6-rebind-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "rebind"}
	pktStatsMap["pkt6-release-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "release"}
	pktStatsMap["pkt6-decline-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "decline"}
	pktStatsMap["pkt6-infrequest-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "infrequest"}
	pktStatsMap["pkt6-unknown-received"] = statisticDescriptor{Stat: packets6ReceivedTotal, Operation: "unknown"}

	// packets6SentTotal
	pktStatsMap["pkt6-advertise-sent"] = statisticDescriptor{Stat: packets6SentTotal, Operation: "advertise"}
	pktStatsMap["pkt6-reply-sent"] = statisticDescriptor{Stat: packets6SentTotal, Operation: "reply"}

	// packets4o6SentTotal & packets4o6ReceivedTotal
	pktStatsMap["pkt6-dhcpv4-response-sent"] = statisticDescriptor{Stat: packets4o6SentTotal, Operation: "response"}
	pktStatsMap["pkt6-dhcpv4-query-received"] = statisticDescriptor{Stat: packets4o6ReceivedTotal, Operation: "query"}
	pktStatsMap["pkt6-dhcpv4-response-received"] = statisticDescriptor{Stat: packets4o6ReceivedTotal, Operation: "response"}

	pke.PktStatsMap = pktStatsMap

	// Collecting per subnet stats is enabled by default. It can be explicitly disabled.
	if pke.EnablePerSubnetStats {
		log.Info(
			"Per-subnet statistics are enabled. You may consider turning it" +
				" off if you observe the performance problems for Kea" +
				" deployments with many subnets.")
		// addresses dhcp4
		adr4StatsMap := make(map[string]*prometheus.GaugeVec)
		adr4StatsMap["assigned-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_assigned_total",
			Help:      "Assigned addresses",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr4StatsMap["declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_declined_total",
			Help:      "Declined counts",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr4StatsMap["reclaimed-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_declined_reclaimed_total",
			Help:      "Declined addresses that were reclaimed",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr4StatsMap["reclaimed-leases"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_reclaimed_total",
			Help:      "Expired addresses that were reclaimed",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr4StatsMap["total-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_total",
			Help:      "Size of subnet address pool",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr4StatsMap["cumulative-assigned-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "cumulative_addresses_assigned_total",
			Help:      "Cumulative number of assigned addresses since server startup",
		}, []string{"subnet", "subnet_id", "prefix"})

		// addresses dhcp6
		adr6StatsMap := make(map[string]*prometheus.GaugeVec)
		adr6StatsMap["total-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "na_total",
			Help:      "'Size of non-temporary address pool",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr6StatsMap["assigned-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "na_assigned_total",
			Help:      "Assigned non-temporary addresses (IA_NA)",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr6StatsMap["total-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pd_total",
			Help:      "Size of prefix delegation pool",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr6StatsMap["assigned-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pd_assigned_total",
			Help:      "Assigned prefix delegations (IA_PD)",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr6StatsMap["reclaimed-leases"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "addresses_reclaimed_total",
			Help:      "Expired addresses that were reclaimed",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr6StatsMap["declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "addresses_declined_total",
			Help:      "Declined counts",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr6StatsMap["reclaimed-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "addresses_declined_reclaimed_total",
			Help:      "Declined addresses that were reclaimed",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr6StatsMap["cumulative-assigned-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "cumulative_nas_assigned_total",
			Help:      "Cumulative number of assigned NA addresses since server startup",
		}, []string{"subnet", "subnet_id", "prefix"})
		adr6StatsMap["cumulative-assigned-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "cumulative_pds_assigned_total",
			Help:      "Cumulative number of assigned PD prefixes since server startup",
		}, []string{"subnet", "subnet_id", "prefix"})

		pke.Adr4StatsMap = adr4StatsMap
		pke.Adr6StatsMap = adr6StatsMap
	} else {
		log.Info(
			"Per-subnet statistics are disabled. You may consider turning it" +
				" on if you want to collect more detailed statistics for each" +
				" subnet.")
	}

	// prepare http handler
	mux := http.NewServeMux()
	handler := promhttp.HandlerFor(pke.Registry, promhttp.HandlerOpts{})
	mux.Handle("/metrics", handler)
	pke.HTTPServer = &http.Server{
		Handler: mux,
		// Protection against Slowloris Attack (G112).
		ReadHeaderTimeout: 60 * time.Second,
	}

	return pke
}

// Start goroutine with main loop for collecting stats
// and http server for exposing them to Prometheus.
func (pke *PromKeaExporter) Start() {
	// set address for listening from config
	addrPort := net.JoinHostPort(pke.Host, strconv.Itoa(pke.Port))
	pke.HTTPServer.Addr = addrPort

	log.Printf("Prometheus Kea Exporter listening on %s, stats pulling interval: %.f seconds",
		addrPort, pke.Interval.Seconds())

	// start http server for metrics
	go func() {
		err := pke.HTTPServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).
				Error("Problem serving Prometheus Kea Exporter")
		}
	}()

	// set ticker time for collecting loop from config
	pke.Ticker = time.NewTicker(pke.Interval)

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
			log.Warnf("Could not gracefully shut down the Kea exporter: %v\n", err)
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
	for _, stat := range pke.Global4StatMap {
		pke.Registry.Unregister(stat)
	}
	for _, stat := range pke.Global6StatMap {
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
				log.WithError(err).Error("Some errors were encountered while collecting stats from Kea")
			}
		// wait for done signal from shutdown function
		case <-pke.DoneCollector:
			return
		}
	}
}

// setDaemonStats stores the stat values from a daemon in the proper prometheus object.
func (pke *PromKeaExporter) setDaemonStats(dhcpStatMap *map[string]*prometheus.GaugeVec, globalStatMap map[string]prometheus.Gauge, response map[string]GetAllStatisticResponseItemValue, ignoredStats map[string]bool, prefixLookup subnetPrefixLookup) {
	for statName, statEntry := range response {
		// skip ignored stats
		if ignoredStats[statName] {
			continue
		}

		// store stat value in proper prometheus object
		switch {
		case strings.HasPrefix(statName, "pkt"):
			// if this is pkt stat
			statisticDescriptor, ok := pke.PktStatsMap[statName]
			if ok {
				statisticDescriptor.Stat.With(prometheus.Labels{"operation": statisticDescriptor.Operation}).Set(statEntry.Value)
			} else {
				log.Warningf("Encountered unsupported stat: %s", statName)
				ignoredStats[statName] = true
			}
		case strings.HasPrefix(statName, "subnet["):
			// Check if collecting the per-subnet metrics is enabled.
			// Processing subnet metrics for the Kea instance with
			// thousands of subnets causes a significant CPU overhead.
			// It's possible to disable collecting these metrics to limit
			// CPU consumption.
			if *dhcpStatMap == nil {
				continue
			}
			// if this is address per subnet stat
			re := regexp.MustCompile(`subnet\[(\d+)\]\.(.+)`)
			matches := re.FindStringSubmatch(statName)
			subnetIDRaw := matches[1]
			metricName := matches[2]

			labels := prometheus.Labels{"subnet_id": subnetIDRaw, "prefix": ""}
			subnetID, err := strconv.Atoi(subnetIDRaw)
			legacyLabel := subnetIDRaw // Subnet ID or prefix if available.
			if err == nil {
				subnetPrefix, ok := prefixLookup.getPrefix(subnetID)
				if ok {
					labels["prefix"] = subnetPrefix
					legacyLabel = subnetPrefix
				}
			}
			labels["subnet"] = legacyLabel

			if stat, ok := (*dhcpStatMap)[metricName]; ok {
				stat.With(labels).Set(statEntry.Value)
			} else {
				log.Warningf("Encountered unsupported stat: %s", statName)
				ignoredStats[statName] = true
			}
		default:
			if globalGauge, ok := globalStatMap[statName]; ok {
				globalGauge.Set(statEntry.Value)
			} else {
				log.Warningf("Encountered unsupported stat: %s", statName)
				ignoredStats[statName] = true
			}
		}
	}
}

// Collect stats from all Kea apps.
func (pke *PromKeaExporter) collectStats() error {
	// Update uptime counter
	pke.ExporterStatMap["uptime_seconds"].Set(time.Since(pke.StartTime).Seconds())

	var lastErr error

	// Request to kea dhcp daemons for getting all stats.
	requestData := &keactrl.Command{
		Command: keactrl.StatisticGetAll,
		// Send the request only to the configured daemons.
		Daemons:   nil,
		Arguments: map[string]any{},
	}

	// Go through all kea apps discovered by monitor and query them for stats.
	apps := pke.AppMonitor.GetApps()
	keaAppsCount := 0
	activeDHCP4DaemonsCount := 0
	activeDHCP6DaemonsCount := 0
	configuredDHCP4DaemonsCount := 0
	configuredDHCP6DaemonsCount := 0

	const (
		dhcp4 = "dhcp4"
		dhcp6 = "dhcp6"
	)

	for _, app := range apps {
		// Ignore non-kea apps.
		if app.GetBaseApp().Type != AppTypeKea {
			continue
		}
		keaApp := app.(*KeaApp)
		keaAppsCount++

		// Count the configured daemons.
		for _, daemon := range keaApp.ConfiguredDaemons {
			if daemon == dhcp4 {
				configuredDHCP4DaemonsCount++
			} else if daemon == dhcp6 {
				configuredDHCP6DaemonsCount++
			}
		}

		// Collect the list of the active DHCP daemons in a given app to
		// avoid sending requests to non-existing daemons.
		var services []string
		for _, daemon := range keaApp.ActiveDaemons {
			// Select services (daemons) that support the get-statistics-all
			// command.
			if daemon == dhcp4 {
				services = append(services, daemon)
			} else if daemon == dhcp6 {
				services = append(services, daemon)
			}
		}
		if len(services) == 0 {
			err := errors.Errorf("missing configured daemons in the application: %+v", app.GetBaseApp())
			lastErr = err
			log.WithError(err).Error("The Kea application has no DHCP daemons configured")
			continue
		}
		requestData.Daemons = services

		// get stats from kea
		requestDataBytes, err := json.Marshal(requestData)
		if err != nil {
			err = errors.Wrap(err, "cannot serialize a request to JSON")
			lastErr = err
			log.WithError(err).Error("Problem serializing the statistics request to JSON")
			continue
		}

		// Fetching statistics
		responseData, err := keaApp.sendCommandRaw(requestDataBytes)
		if err != nil {
			lastErr = err
			log.WithError(err).Error("Problem fetching stats from Kea")
			continue
		}

		// Parse response
		var response GetAllStatisticsResponse
		err = json.Unmarshal(responseData, &response)
		if err != nil {
			lastErr = err
			log.WithError(err).
				WithField("request", requestData).
				Error("Failed to parse responses from Kea")
			continue
		}

		// The number of responses should match the number of services.
		if len(response) != len(services) {
			err = errors.Errorf("number of responses (%d) does not match the number of services (%d)", len(response), len(services))
			lastErr = err
			log.WithError(err).
				Error("Unexpected number of responses from Kea")
			continue
		}

		// Prepare subnet prefix lookup
		subnetPrefixLookup := newLazySubnetPrefixLookup(keaApp)

		// Go though responses from daemons (it can have none or some responses from dhcp4/dhcp6)
		// and store collected stats in Prometheus structures.
		// Fetching also DHCP subnet prefixes. It may fail if Kea doesn't support
		// required commands.
		for i, service := range services {
			serviceResponse := response[i]

			if service == dhcp4 {
				activeDHCP4DaemonsCount++
				subnetPrefixLookup.setFamily(4)
				pke.setDaemonStats(&pke.Adr4StatsMap, pke.Global4StatMap, serviceResponse, pke.ignoredStats, subnetPrefixLookup)
			} else if service == dhcp6 {
				activeDHCP6DaemonsCount++
				subnetPrefixLookup.setFamily(6)
				pke.setDaemonStats(&pke.Adr6StatsMap, pke.Global6StatMap, serviceResponse, pke.ignoredStats, subnetPrefixLookup)
			}
		}
	}

	// Set the number of monitored Kea applications and daemons.
	pke.ExporterStatMap["monitored_kea_apps"].Set(float64(keaAppsCount))
	pke.ExporterStatMap["active_dhcp4_daemons"].Set(float64(activeDHCP4DaemonsCount))
	pke.ExporterStatMap["active_dhcp6_daemons"].Set(float64(activeDHCP6DaemonsCount))
	pke.ExporterStatMap["configured_dhcp4_daemons"].Set(float64(configuredDHCP4DaemonsCount))
	pke.ExporterStatMap["configured_dhcp6_daemons"].Set(float64(configuredDHCP6DaemonsCount))

	return lastErr
}
