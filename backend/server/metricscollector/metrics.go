package metricscollector

// Functions to manage the Prometheus metrics.
//
// To add new statistic you should:
// 1. Update the Metrics structure.
// 2. Prepare the metric instance in the NewMetrics function.
// 3. Update SQL query (if needed) in the database/model/metrics.go file.
// 4. Change the UpdateMetrics function to collect new metric values.

import (
	"reflect"

	"github.com/go-pg/pg/v9"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dbmodel "isc.org/stork/server/database/model"
)

// Set of Stork server metrics.
type Metrics struct {
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
func NewMetrics(registry *prometheus.Registry) Metrics {
	factory := promauto.With(registry)

	namespace := "storkserver"

	metrics := Metrics{
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
			Help:      "Subnet pd utilization",
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
			Help:      "Shared network pd utilization",
		}, []string{"name"}),
	}

	return metrics
}

// Calculate current metric values from the database.
func UpdateMetrics(db *pg.DB, metrics Metrics) error {
	// statistics, err := dbmodel.GetAllStats(db)
	// if err != nil {
	// 	return err
	// }

	calculatedMetrics, err := dbmodel.GetCalculatedMetrics(db)
	if err != nil {
		return err
	}

	metrics.AuthorizedMachineTotal.Set(float64(calculatedMetrics.AuthorizedMachines))
	metrics.UnauthorizedMachineTotal.Set(float64(calculatedMetrics.UnauthorizedMachines))
	metrics.UnreachableMachineTotal.Set(float64(calculatedMetrics.UnreachableMachines))

	for _, networkMetrics := range calculatedMetrics.SubnetMetrics {
		metrics.SubnetAddressUtilization.
			With(prometheus.Labels{"subnet": networkMetrics.Label}).
			Set(float64(networkMetrics.AddrUtilization) / 1000.)
		metrics.SubnetPdUtilization.
			With(prometheus.Labels{"subnet": networkMetrics.Label}).
			Set(float64(networkMetrics.PdUtilization) / 1000.)
	}

	for _, networkMetrics := range calculatedMetrics.SharedNetworkMetrics {
		metrics.SharedNetworkAddressUtilization.
			With(prometheus.Labels{"name": networkMetrics.Label}).
			Set(float64(networkMetrics.AddrUtilization) / 1000.)
		metrics.SharedNetworkPdUtilization.
			With(prometheus.Labels{"name": networkMetrics.Label}).
			Set(float64(networkMetrics.PdUtilization) / 1000.)
	}

	return nil
}

// Unregister all metrics from the Prometheus registry.
func UnregisterAllMetrics(registry *prometheus.Registry, metrics Metrics) {
	v := reflect.ValueOf(metrics)
	typeMetrics := v.Type()
	for i := 0; i < typeMetrics.NumField(); i++ {
		rawField := v.Field(i).Interface()
		collector, ok := rawField.(prometheus.Collector)
		if !ok {
			continue
		}
		registry.Unregister(collector)
	}
}
