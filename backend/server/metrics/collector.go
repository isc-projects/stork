package metrics

import (
	"fmt"
	"net/http"

	"github.com/go-pg/pg/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	dbmodel "isc.org/stork/server/database/model"
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

	authorizedMachineTotalDesc          *prometheus.Desc
	unauthorizedMachineTotalDesc        *prometheus.Desc
	unreachableMachineTotalDesc         *prometheus.Desc
	subnetAddressUtilizationDesc        *prometheus.Desc
	subnetPdUtilizationDesc             *prometheus.Desc
	sharedNetworkAddressUtilizationDesc *prometheus.Desc
	sharedNetworkPdUtilizationDesc      *prometheus.Desc
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

		authorizedMachineTotalDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "auth", "authorized_machine_total"),
			"Authorized machines",
			nil, nil,
		),
		unauthorizedMachineTotalDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "auth", "unauthorized_machine_total"),
			"Unauthorized machines",
			nil, nil,
		),
		unreachableMachineTotalDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "auth", "unreachable_machine_total"),
			"Unreachable machines",
			nil, nil,
		),
		subnetAddressUtilizationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "subnet", "address_utilization"),
			"Subnet address utilization",
			[]string{"subnet"}, nil,
		),
		subnetPdUtilizationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "subnet", "pd_utilization"),
			"Subnet delegated-prefix utilization",
			[]string{"subnet"}, nil,
		),
		sharedNetworkAddressUtilizationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "shared_network", "address_utilization"),
			"Shared-network address utilization",
			[]string{"name", "family"}, nil,
		),
		sharedNetworkPdUtilizationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "shared_network", "pd_utilization"),
			"Shared-network delegated-prefix utilization",
			[]string{"name"}, nil,
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
	ch <- c.authorizedMachineTotalDesc
	ch <- c.unauthorizedMachineTotalDesc
	ch <- c.unreachableMachineTotalDesc
	ch <- c.subnetAddressUtilizationDesc
	ch <- c.subnetPdUtilizationDesc
	ch <- c.sharedNetworkAddressUtilizationDesc
	ch <- c.sharedNetworkPdUtilizationDesc
}

// Collect implements the prometheus.Collector interface. Converts the metrics
// from the database to Prometheus format.
func (c *prometheusCollector) Collect(ch chan<- prometheus.Metric) {
	calculatedMetrics, err := c.source.GetCalculatedMetrics()
	if err != nil {
		log.WithError(err).Error("Failed to fetch metrics from the database")
		return
	}

	ch <- prometheus.MustNewConstMetric(c.authorizedMachineTotalDesc,
		prometheus.GaugeValue, float64(calculatedMetrics.AuthorizedMachines))
	ch <- prometheus.MustNewConstMetric(c.unauthorizedMachineTotalDesc,
		prometheus.GaugeValue, float64(calculatedMetrics.UnauthorizedMachines))
	ch <- prometheus.MustNewConstMetric(c.unreachableMachineTotalDesc,
		prometheus.GaugeValue, float64(calculatedMetrics.UnreachableMachines))

	for _, networkMetrics := range calculatedMetrics.SubnetMetrics {
		ch <- prometheus.MustNewConstMetric(c.subnetAddressUtilizationDesc,
			prometheus.GaugeValue, float64(networkMetrics.AddrUtilization)/1000.,
			networkMetrics.Label)
		ch <- prometheus.MustNewConstMetric(c.subnetPdUtilizationDesc,
			prometheus.GaugeValue,
			float64(networkMetrics.PdUtilization)/1000.,
			networkMetrics.Label)
	}

	for _, networkMetrics := range calculatedMetrics.SharedNetworkMetrics {
		ch <- prometheus.MustNewConstMetric(c.sharedNetworkAddressUtilizationDesc,
			prometheus.GaugeValue,
			float64(networkMetrics.AddrUtilization)/1000.,
			networkMetrics.Label, fmt.Sprint(networkMetrics.Family))
		if networkMetrics.Family == 6 {
			ch <- prometheus.MustNewConstMetric(c.sharedNetworkPdUtilizationDesc,
				prometheus.GaugeValue,
				float64(networkMetrics.PdUtilization)/1000.,
				networkMetrics.Label)
		}
	}
}
