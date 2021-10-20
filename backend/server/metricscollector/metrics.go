package metricscollector

import (
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	AuthorizedMachineTotal   prometheus.Gauge
	UnauthorizedMachineTotal prometheus.Gauge
	UnreachableMachineTotal  prometheus.Gauge
}

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

func UpdateMetrics(metrics Metrics) {
	authorizedMachines := 10
	unauthorizedMachines := 20
	unreachableMachines := 5

	metrics.AuthorizedMachineTotal.Set(float64(authorizedMachines))
	metrics.UnauthorizedMachineTotal.Set(float64(unauthorizedMachines))
	metrics.UnreachableMachineTotal.Set(float64(unreachableMachines))
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
