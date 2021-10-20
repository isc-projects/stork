package metricscollector

// Functions to manage the Prometheus metrics.
//
// For add new statistic you should:
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
	AuthorizedMachineTotal   prometheus.Gauge
	UnauthorizedMachineTotal prometheus.Gauge
	UnreachableMachineTotal  prometheus.Gauge
}

// Constructor of the metrics. They are automatically
// registered in the Prometheus.
func NewMetrics(registry *prometheus.Registry) Metrics {
	factory := promauto.With(registry)

	namespace := "stork_server"

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
	}

	return metrics
}

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
	return nil
}

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
