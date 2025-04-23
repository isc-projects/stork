package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
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

// Main structure for Prometheus Kea Exporter.
type PromKeaExporter struct {
	Host string
	Port int

	EnablePerSubnetStats bool

	AppMonitor AppMonitor
	HTTPServer *http.Server

	StartTime time.Time

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
func NewPromKeaExporter(host string, port int, enablePerSubnetStats bool, appMonitor AppMonitor) *PromKeaExporter {
	pke := &PromKeaExporter{
		Host:                 host,
		Port:                 port,
		EnablePerSubnetStats: enablePerSubnetStats,
		AppMonitor:           appMonitor,
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
		subnetLabels := []string{"subnet", "subnet_id", "prefix"}
		// addresses dhcp4
		adr4StatsMap := make(map[string]*prometheus.GaugeVec)
		adr4StatsMap["assigned-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_assigned_total",
			Help:      "Assigned addresses",
		}, subnetLabels)
		adr4StatsMap["declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_declined_total",
			Help:      "Declined counts",
		}, subnetLabels)
		adr4StatsMap["reclaimed-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_declined_reclaimed_total",
			Help:      "Declined addresses that were reclaimed",
		}, subnetLabels)
		adr4StatsMap["reclaimed-leases"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_reclaimed_total",
			Help:      "Expired addresses that were reclaimed",
		}, subnetLabels)
		adr4StatsMap["total-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "addresses_total",
			Help:      "Size of subnet address pool",
		}, subnetLabels)
		adr4StatsMap["cumulative-assigned-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "cumulative_addresses_assigned_total",
			Help:      "Cumulative number of assigned addresses since server startup",
		}, subnetLabels)

		// addresses dhcp6
		adr6StatsMap := make(map[string]*prometheus.GaugeVec)
		adr6StatsMap["total-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "na_total",
			Help:      "'Size of non-temporary address pool",
		}, subnetLabels)
		adr6StatsMap["assigned-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "na_assigned_total",
			Help:      "Assigned non-temporary addresses (IA_NA)",
		}, subnetLabels)
		adr6StatsMap["total-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pd_total",
			Help:      "Size of prefix delegation pool",
		}, subnetLabels)
		adr6StatsMap["assigned-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pd_assigned_total",
			Help:      "Assigned prefix delegations (IA_PD)",
		}, subnetLabels)
		adr6StatsMap["reclaimed-leases"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "addresses_reclaimed_total",
			Help:      "Expired addresses that were reclaimed",
		}, subnetLabels)
		adr6StatsMap["declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "addresses_declined_total",
			Help:      "Declined counts",
		}, subnetLabels)
		adr6StatsMap["reclaimed-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "addresses_declined_reclaimed_total",
			Help:      "Declined addresses that were reclaimed",
		}, subnetLabels)
		adr6StatsMap["cumulative-assigned-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "cumulative_nas_assigned_total",
			Help:      "Cumulative number of assigned NA addresses since server startup",
		}, subnetLabels)
		adr6StatsMap["cumulative-assigned-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "cumulative_pds_assigned_total",
			Help:      "Cumulative number of assigned PD prefixes since server startup",
		}, subnetLabels)

		poolLabels := []string{"pool_id"}
		poolLabels = append(poolLabels, subnetLabels...)

		adr4StatsMap["pool-assigned-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "pool_addresses_assigned_total",
			Help:      "Total number of assigned addresses in the DHCPv4 pool",
		}, poolLabels)
		adr4StatsMap["pool-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "pool_addresses_declined_total",
			Help:      "Total number of declined addresses in the DHCPv4 pool",
		}, poolLabels)
		adr4StatsMap["pool-reclaimed-leases"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "pool_addresses_reclaimed_total",
			Help:      "Total number of reclaimed leases in the DHCPv4 pool",
		}, poolLabels)
		adr4StatsMap["pool-reclaimed-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "pool_addresses_declined_reclaimed_total",
			Help:      "Total number of reclaimed declined addresses in the DHCPv4 pool",
		}, poolLabels)
		adr4StatsMap["pool-cumulative-assigned-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "pool_addresses_cumulative_assigned_total",
			Help:      "Total number of cumulative assigned addresses in the DHCPv4 pool",
		}, poolLabels)
		adr4StatsMap["pool-total-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp4",
			Name:      "pool_addresses_total",
			Help:      "Total number of addresses in the DHCPv4 pool",
		}, poolLabels)

		adr6StatsMap["pool-assigned-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_na_assigned_total",
			Help:      "Total number of assigned non-temporary addresses in the DHCPv6 pool",
		}, poolLabels)
		adr6StatsMap["pool-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_addresses_declined_total",
			Help:      "Total number of declined addresses in the DHCPv6 pool",
		}, poolLabels)
		adr6StatsMap["pool-reclaimed-leases"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_addresses_reclaimed_total",
			Help:      "Total number of reclaimed leases in the DHCPv6 pool",
		}, poolLabels)
		adr6StatsMap["pool-reclaimed-declined-addresses"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_addresses_declined_reclaimed_total",
			Help:      "Total number of reclaimed declined addresses in the DHCPv6 pool",
		}, poolLabels)
		adr6StatsMap["pool-cumulative-assigned-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_cumulative_nas_assigned_total",
			Help:      "Cumulative number of assigned addresses in the DHCPv6 pool",
		}, poolLabels)
		adr6StatsMap["pool-total-nas"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_na_total",
			Help:      "Size of non-temporary address pool",
		}, poolLabels)

		adr6StatsMap["pool-pd-assigned-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_pd_assigned_total",
			Help:      "Total number of assigned prefixes in the DHCPv6 pool",
		}, poolLabels)
		// There is a metric with the same name ("reclaimed-leases") in the
		// (NA) pools and pd-pools.
		adr6StatsMap["pool-pd-reclaimed-leases"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_pd_addresses_reclaimed_total",
			Help:      "Total number of reclaimed leases in the DHCPv6 pool",
		}, poolLabels)
		adr6StatsMap["pool-pd-cumulative-assigned-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_cumulative_pds_assigned_total",
			Help:      "Cumulative number of assigned prefixes in the DHCPv6 pool",
		}, poolLabels)
		adr6StatsMap["pool-pd-total-pds"] = factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: AppTypeKea,
			Subsystem: "dhcp6",
			Name:      "pool_pd_total",
			Help:      "Size of prefix delegation pool",
		}, poolLabels)

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
	// Register collectors.
	version.Version = stork.Version
	pke.Registry.MustRegister(pke, versioncollector.NewCollector("kea_exporter"))

	// Set address for listening from config.
	addrPort := net.JoinHostPort(pke.Host, strconv.Itoa(pke.Port))
	pke.HTTPServer.Addr = addrPort

	log.Printf("Prometheus Kea Exporter listening on %s", addrPort)

	// Start HTTP server for metrics.
	go func() {
		err := pke.HTTPServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).
				Error("Problem serving Prometheus Kea Exporter")
		}
	}()
}

// Shutdown exporter goroutines and unregister prometheus stats.
func (pke *PromKeaExporter) Shutdown() {
	log.Printf("Stopping Prometheus Kea Exporter")

	// Stop HTTP server.
	if pke.HTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		pke.HTTPServer.SetKeepAlivesEnabled(false)
		if err := pke.HTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("Could not gracefully shut down the Kea exporter: %v\n", err)
		}
	}

	// Unregister kea counters from prometheus framework.
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
	for _, stat := range pke.ExporterStatMap {
		pke.Registry.Unregister(stat)
	}

	log.Printf("Stopped Prometheus Kea Exporter")
}

// Collect fetches the stats from configured location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (pke *PromKeaExporter) Collect(ch chan<- prometheus.Metric) {
	err := pke.collectStats()
	if err != nil {
		log.WithError(err).Error("Some errors were encountered while collecting stats from Kea")
	}
}

// Describe describes all exported metrics. It implements prometheus.Collector.
func (pke *PromKeaExporter) Describe(ch chan<- *prometheus.Desc) {
	// Nothing is put in the channel in the Collect() method, so there is
	// nothing to describe.
}

// setDaemonStats stores the stat values from a daemon in the proper prometheus object.
func (pke *PromKeaExporter) setDaemonStats(dhcpStatMap map[string]*prometheus.GaugeVec, globalStatMap map[string]prometheus.Gauge, response keactrl.GetAllStatisticArguments, ignoredStats map[string]bool, prefixLookup subnetPrefixLookup) {
	for _, statEntry := range response {
		// store stat value in proper prometheus object
		switch {
		case strings.HasPrefix(statEntry.Name, "pkt"):
			// skip ignored stats
			if ignoredStats[statEntry.Name] {
				continue
			}

			// if this is pkt stat
			statisticDescriptor, ok := pke.PktStatsMap[statEntry.Name]
			if ok {
				// Go Prometheus library does not support big integers.
				value, _ := statEntry.Value.Float64()
				statisticDescriptor.Stat.With(prometheus.Labels{"operation": statisticDescriptor.Operation}).Set(value)
			} else {
				log.Warningf("Encountered unsupported stat: %s", statEntry.Name)
				ignoredStats[statEntry.Name] = true
			}
		case statEntry.SubnetID != 0:
			// Check if collecting the per-subnet metrics is enabled.
			// Processing subnet metrics for the Kea instance with
			// thousands of subnets causes a significant CPU overhead.
			// It's possible to disable collecting these metrics to limit
			// CPU consumption.
			if dhcpStatMap == nil {
				continue
			}
			legacyLabel := fmt.Sprint(statEntry.SubnetID) // Subnet ID or prefix if available.
			labels := prometheus.Labels{"subnet_id": legacyLabel, "prefix": ""}
			subnetPrefix, ok := prefixLookup.getPrefix(int(statEntry.SubnetID))
			if ok {
				labels["prefix"] = subnetPrefix
				legacyLabel = subnetPrefix
			}
			labels["subnet"] = legacyLabel

			statName := statEntry.Name
			// skip ignored stats
			if ignoredStats[statName] {
				continue
			}

			switch {
			case statEntry.PoolID != 0:
				labels["pool_id"] = fmt.Sprint(statEntry.PoolID)
				statName = "pool-" + statName
			case statEntry.PrefixPoolID != 0:
				labels["pool_id"] = fmt.Sprint(statEntry.PrefixPoolID)
				statName = "pool-pd-" + statName
			default:
				// It isn't a pool stat. Just a subnet stat.
			}

			if stat, ok := dhcpStatMap[statName]; ok {
				value, _ := statEntry.Value.Float64()
				stat.With(labels).Set(value)
			} else {
				log.Warningf("Encountered unsupported stat: %s", statName)
				ignoredStats[statName] = true
			}
		default:
			// skip ignored stats
			if ignoredStats[statEntry.Name] {
				continue
			}

			if globalGauge, ok := globalStatMap[statEntry.Name]; ok {
				value, _ := statEntry.Value.Float64()
				globalGauge.Set(value)
			} else {
				log.Warningf("Encountered unsupported stat: %s", statEntry.Name)
				ignoredStats[statEntry.Name] = true
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
		var response keactrl.GetAllStatisticsResponse
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
			adrStatsMap := pke.Adr4StatsMap
			globalStatMap := pke.Global4StatMap
			activeDaemonsCount := &activeDHCP4DaemonsCount
			family := 4
			if service == dhcp6 {
				adrStatsMap = pke.Adr6StatsMap
				globalStatMap = pke.Global6StatMap
				activeDaemonsCount = &activeDHCP6DaemonsCount
				family = 6
			}

			*activeDaemonsCount++

			serviceResponse := response[i]
			if err := serviceResponse.GetError(); err != nil {
				if !errors.As(err, &keactrl.NumberOverflowKeaError{}) {
					*activeDaemonsCount--
				}
				log.WithError(err).
					WithField("service", service).
					Error("Problem fetching stats from Kea")
				continue
			}

			subnetPrefixLookup.setFamily(int8(family))
			pke.setDaemonStats(adrStatsMap, globalStatMap, serviceResponse.Arguments, pke.ignoredStats, subnetPrefixLookup)
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
