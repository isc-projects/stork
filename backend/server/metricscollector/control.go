package metricscollector

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	storkutil "isc.org/stork/util"
)

type Control interface {
	SetupHandler(next http.Handler) http.Handler
	Shutdown()
}

type PrometheusControl struct {
	registry *prometheus.Registry
	metrics  Metrics
	puller   *storkutil.PeriodicExecutor
}

func NewControl() Control {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metricPuller := storkutil.NewPeriodicExecutor("metrics",
		func() error {
			UpdateMetrics(metrics)
			return nil
		},
		func(prev int64) int64 { return 10 },
	)

	return &PrometheusControl{
		metrics:  metrics,
		registry: registry,
		puller:   metricPuller,
	}
}

func (c *PrometheusControl) SetupHandler(next http.Handler) http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})
}

func (c *PrometheusControl) Shutdown() {
	c.puller.Shutdown()
	UnregisterAllMetrics(c.registry, c.metrics)
}
