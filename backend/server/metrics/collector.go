package metrics

import (
	"fmt"
	"net/http"

	"github.com/go-pg/pg/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Interface of the metrics source. It is responsible for returning the
// current metric values. It is used to allow testing the collector without
// extensive seeding of the database.
type MetricsSource interface {
	GetCalculatedMetrics() (*dbmodel.CalculatedMetrics, error)
}

// The production implementation of the metrics source based on the database.
type databaseMetricsSource struct {
	db *pg.DB
}

// Creates an instance of the metrics source for a database.
func NewDatabaseMetricsSource(db *pg.DB) MetricsSource {
	return &databaseMetricsSource{db: db}
}

// Returns the current metric values from the database.
func (s *databaseMetricsSource) GetCalculatedMetrics() (*dbmodel.CalculatedMetrics, error) {
	return dbmodel.GetCalculatedMetrics(s.db)
}

// Interface of the metrics collector. Metric collector is
// a background worker which collect various metrics
// about the application.
//
// It is responsible for creating HTTP handler to access
// the metrics.
type Collector interface {
	// It returns the metrics on HTTP request.
	GetHTTPHandler(next http.Handler) http.Handler
	// Shutdown metrics collecting.
	Shutdown()
}

// Metrics collector created on top of
// Prometheus library.
type prometheusCollector struct {
	source   MetricsSource
	registry *prometheus.Registry

	authorizedMachineTotalDescriptor          *prometheus.Desc
	unauthorizedMachineTotalDescriptor        *prometheus.Desc
	unreachableMachineTotalDescriptor         *prometheus.Desc
	subnetAddressUtilizationDescriptor        *prometheus.Desc
	subnetPdUtilizationDescriptor             *prometheus.Desc
	sharedNetworkAddressUtilizationDescriptor *prometheus.Desc
	sharedNetworkPdUtilizationDescriptor      *prometheus.Desc
	// The statistics are stored as a map in the dbmodel.SharedNetwork
	// structure. So, it is possible to handle all of them in the same way and
	// convert them to the Prometheus metrics using for-loop. The collector
	// must store them in an iterable structure, such as a map, to achieve
	// this.
	//
	// The metrics must be iterated in the strict order.
	// The default map's iteration order in Golang is indeterministic. It isn't
	// specified if the order is preserved for subsequent iterations. The
	// collector iterates the metrics twice: in the Desc and the Collect
	// methods. The iteration order must be the same. Otherwise, the samples
	// will be assigned to the wrong metrics. Therefore, the OrderedMap is used
	// here.
	sharedNetworkStatisticDescriptors *storkutil.OrderedMap[dbmodel.SubnetStatsName, *prometheus.Desc]
}

var _ prometheus.Collector = (*prometheusCollector)(nil)

// Creates an instance of the metrics collector and starts
// collecting the metrics according to the interval
// specified in the database.
func NewCollector(source MetricsSource) (Collector, error) {
	registry := prometheus.NewRegistry()

	namespace := "storkserver"

	collector := &prometheusCollector{
		source:   source,
		registry: registry,

		authorizedMachineTotalDescriptor: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "auth", "authorized_machine_total"),
			"Authorized machines",
			nil, nil,
		),
		unauthorizedMachineTotalDescriptor: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "auth", "unauthorized_machine_total"),
			"Unauthorized machines",
			nil, nil,
		),
		unreachableMachineTotalDescriptor: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "auth", "unreachable_machine_total"),
			"Unreachable machines",
			nil, nil,
		),
		subnetAddressUtilizationDescriptor: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "subnet", "address_utilization"),
			"Subnet address utilization",
			[]string{"subnet"}, nil,
		),
		subnetPdUtilizationDescriptor: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "subnet", "pd_utilization"),
			"Subnet delegated-prefix utilization",
			[]string{"subnet"}, nil,
		),
		sharedNetworkAddressUtilizationDescriptor: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "shared_network", "address_utilization"),
			"Shared-network address utilization",
			[]string{"name", "family"}, nil,
		),
		sharedNetworkPdUtilizationDescriptor: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "shared_network", "pd_utilization"),
			"Shared-network delegated-prefix utilization",
			[]string{"name"}, nil,
		),
		sharedNetworkStatisticDescriptors: storkutil.NewOrderedMapFromEntries(
			[]dbmodel.SubnetStatsName{
				dbmodel.SubnetStatsNameTotalNAs,
				dbmodel.SubnetStatsNameAssignedNAs,
				dbmodel.SubnetStatsNameTotalPDs,
				dbmodel.SubnetStatsNameAssignedPDs,
			},
			[]*prometheus.Desc{
				prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "shared_network", "total_na"),
					"Shared-network total number of assigned NAs",
					[]string{"name", "family"}, nil,
				),
				prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "shared_network", "assigned_na"),
					"Shared-network number of assigned NAs",
					[]string{"name", "family"}, nil,
				),
				prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "shared_network", "total_pd"),
					"Shared-network total number of assigned PDs",
					[]string{"name"}, nil,
				),
				prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "shared_network", "assigned_pd"),
					"Shared-network number of assigned PDs",
					[]string{"name"}, nil,
				),
			},
		),
	}

	registry.MustRegister(collector)
	return collector, nil
}

// Creates standard Prometheus HTTP handler.
func (c *prometheusCollector) GetHTTPHandler(next http.Handler) http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{
		ErrorLog: log.StandardLogger(),
	})
}

// Stops periodically collecting the metrics and unregisters
// all metrics.
func (c *prometheusCollector) Shutdown() {
	c.unregisterAll()
}

// Unregister all metrics from the Prometheus registry.
func (c *prometheusCollector) unregisterAll() {
	c.registry.Unregister(c)
}

// Describe implements the prometheus.Collector interface. Returns the
// descriptors of all metrics.
func (c *prometheusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.authorizedMachineTotalDescriptor
	ch <- c.unauthorizedMachineTotalDescriptor
	ch <- c.unreachableMachineTotalDescriptor
	ch <- c.subnetAddressUtilizationDescriptor
	ch <- c.subnetPdUtilizationDescriptor
	ch <- c.sharedNetworkAddressUtilizationDescriptor
	ch <- c.sharedNetworkPdUtilizationDescriptor
	for _, descriptor := range c.sharedNetworkStatisticDescriptors.GetValues() {
		ch <- descriptor
	}
}

// Collect implements the prometheus.Collector interface. Converts the metrics
// from the database to Prometheus format.
func (c *prometheusCollector) Collect(ch chan<- prometheus.Metric) {
	calculatedMetrics, err := c.source.GetCalculatedMetrics()
	if err != nil {
		log.WithError(err).Error("Failed to fetch metrics from the database")
		return
	}

	ch <- prometheus.MustNewConstMetric(c.authorizedMachineTotalDescriptor,
		prometheus.GaugeValue, float64(calculatedMetrics.AuthorizedMachines))
	ch <- prometheus.MustNewConstMetric(c.unauthorizedMachineTotalDescriptor,
		prometheus.GaugeValue, float64(calculatedMetrics.UnauthorizedMachines))
	ch <- prometheus.MustNewConstMetric(c.unreachableMachineTotalDescriptor,
		prometheus.GaugeValue, float64(calculatedMetrics.UnreachableMachines))

	for _, networkMetrics := range calculatedMetrics.SubnetMetrics {
		ch <- prometheus.MustNewConstMetric(c.subnetAddressUtilizationDescriptor,
			prometheus.GaugeValue, float64(networkMetrics.AddrUtilization)/1000.,
			networkMetrics.Label)
		ch <- prometheus.MustNewConstMetric(c.subnetPdUtilizationDescriptor,
			prometheus.GaugeValue,
			float64(networkMetrics.PdUtilization)/1000.,
			networkMetrics.Label)
	}

	for _, networkMetrics := range calculatedMetrics.SharedNetworkMetrics {
		ch <- prometheus.MustNewConstMetric(c.sharedNetworkAddressUtilizationDescriptor,
			prometheus.GaugeValue,
			float64(networkMetrics.AddrUtilization)/1000.,
			networkMetrics.Label, fmt.Sprint(networkMetrics.Family))

		if networkMetrics.Family == 6 {
			ch <- prometheus.MustNewConstMetric(c.sharedNetworkPdUtilizationDescriptor,
				prometheus.GaugeValue,
				float64(networkMetrics.PdUtilization)/1000.,
				networkMetrics.Label)
		}

		// Statistics.
		v6OnlyStats := map[dbmodel.SubnetStatsName]struct{}{
			dbmodel.SubnetStatsNameAssignedPDs: {},
			dbmodel.SubnetStatsNameTotalPDs:    {},
		}

		for _, entry := range c.sharedNetworkStatisticDescriptors.GetEntries() {
			name, descriptor := entry.Key, entry.Value
			counter := networkMetrics.SharedNetworkStats.GetBigCounter(name)
			if counter == nil {
				continue
			}

			labels := []string{networkMetrics.Label}
			if _, ok := v6OnlyStats[name]; !ok {
				labels = append(labels, fmt.Sprint(networkMetrics.Family))
			} else if networkMetrics.Family != 6 {
				continue
			}

			ch <- prometheus.MustNewConstMetric(
				descriptor, prometheus.GaugeValue, counter.ToFloat64(),
				labels...,
			)
		}
	}
}
