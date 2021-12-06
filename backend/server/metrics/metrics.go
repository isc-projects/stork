package metrics

// Functions to manage the Prometheus metrics.
//
// To add new statistic you should:
// 1. Update the metrics structure.
// 2. Prepare the metric instance in the newMetrics function.
// 3. Update SQL query (if needed) in the database/model/metrics.go file.
// 4. Change the updateMetrics function to collect new metric values.

import (
	"reflect"

	"github.com/go-pg/pg/v9"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dbmodel "isc.org/stork/server/database/model"
)

// Set of Stork Server metrics.
type metrics struct {
	Registry *prometheus.Registry
	db       *pg.DB

	AuthorizedMachineTotal          prometheus.Gauge
	UnauthorizedMachineTotal        prometheus.Gauge
	UnreachableMachineTotal         prometheus.Gauge
	SubnetAddressUtilization        *prometheus.GaugeVec
	SubnetPdUtilization             *prometheus.GaugeVec
	SharedNetworkAddressUtilization *prometheus.GaugeVec
	SharedNetworkPdUtilization      *prometheus.GaugeVec
}

// Constructor of the metrics. They are automatically
// registered in the Prometheus.
func newMetrics(db *pg.DB) *metrics {
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)

	namespace := "storkserver"

	metrics := metrics{
		Registry: registry,
		db:       db,

		AuthorizedMachineTotal: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "authorized_machine_total",
			Subsystem: "auth",
			Help:      "Authorized machines",
		}),
		UnauthorizedMachineTotal: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "unauthorized_machine_total",
			Subsystem: "auth",
			Help:      "Unauthorized machines",
		}),
		UnreachableMachineTotal: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "unreachable_machine_total",
			Subsystem: "auth",
			Help:      "Unreachable machines",
		}),
		SubnetAddressUtilization: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "address_utilization",
			Subsystem: "subnet",
			Help:      "Subnet address utilization",
		}, []string{"subnet"}),
		SubnetPdUtilization: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "pd_utilization",
			Subsystem: "subnet",
			Help:      "Subnet delegated prefix utilization",
		}, []string{"subnet"}),
		SharedNetworkAddressUtilization: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "address_utilization",
			Subsystem: "shared_network",
			Help:      "Shared network address utilization",
		}, []string{"name"}),
		SharedNetworkPdUtilization: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "pd_utilization",
			Subsystem: "shared_network",
			Help:      "Shared network delegated prefix utilization",
		}, []string{"name"}),
	}

	return &metrics
}

// Calculate current metric values from the database.
func (m *metrics) Update() error {
	calculatedMetrics, err := dbmodel.GetCalculatedMetrics(m.db)
	if err != nil {
		return err
	}

	m.AuthorizedMachineTotal.Set(float64(calculatedMetrics.AuthorizedMachines))
	m.UnauthorizedMachineTotal.Set(float64(calculatedMetrics.UnauthorizedMachines))
	m.UnreachableMachineTotal.Set(float64(calculatedMetrics.UnreachableMachines))

	for _, networkMetrics := range calculatedMetrics.SubnetMetrics {
		m.SubnetAddressUtilization.
			With(prometheus.Labels{"subnet": networkMetrics.Label}).
			Set(float64(networkMetrics.AddrUtilization) / 1000.)
		m.SubnetPdUtilization.
			With(prometheus.Labels{"subnet": networkMetrics.Label}).
			Set(float64(networkMetrics.PdUtilization) / 1000.)
	}

	for _, networkMetrics := range calculatedMetrics.SharedNetworkMetrics {
		m.SharedNetworkAddressUtilization.
			With(prometheus.Labels{"name": networkMetrics.Label}).
			Set(float64(networkMetrics.AddrUtilization) / 1000.)
		m.SharedNetworkPdUtilization.
			With(prometheus.Labels{"name": networkMetrics.Label}).
			Set(float64(networkMetrics.PdUtilization) / 1000.)
	}

	return nil
}

// Unregister all metrics from the Prometheus registry.
func (m *metrics) UnregisterAll() {
	v := reflect.ValueOf(*m)
	typeMetrics := v.Type()
	for i := 0; i < typeMetrics.NumField(); i++ {
		fieldObj := v.Field(i)
		if !fieldObj.CanInterface() {
			// Field is not exported.
			continue
		}
		rawField := fieldObj.Interface()
		collector, ok := rawField.(prometheus.Collector)
		if !ok {
			continue
		}
		m.Registry.Unregister(collector)
	}
}
