package metricscollector

// Controller of the metric collector.
// It allows to start and stop the collector.
// It is responsible for creating HTTP handler to access
// the metrics.

import (
	"net/http"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Interface of the metrics collector.
type Control interface {
	// It returns the metrics on HTTP request.
	SetupHandler(next http.Handler) http.Handler
	// Shutdown metrics collecting.
	Shutdown()
}

// Metrics collector created on top of
// Prometheus library.
type PrometheusControl struct {
	registry *prometheus.Registry
	metrics  Metrics
	puller   *storkutil.PeriodicExecutor
}

// Constructor of the metrics collector.
func NewControl(db *pg.DB) (Control, error) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	intervalSettingName := "metrics_collector_interval"

	// Initialize the metrics
	err := UpdateMetrics(db, metrics)
	if err != nil {
		return nil, errors.WithMessage(err, "error during metrics initialization")
	}

	// Starts collecting the metrics periodically.
	metricPuller := storkutil.NewPeriodicExecutor("metrics",
		func() error {
			return UpdateMetrics(db, metrics)
		},
		// ToDo: Setup the interval
		func(prev int64) int64 {
			interval, err := dbmodel.GetSettingInt(db, intervalSettingName)
			if err != nil {
				log.Errorf("problem with getting interval setting %s from db: %+v",
					intervalSettingName, err)
				interval = prev
			}
			return interval
		},
	)

	return &PrometheusControl{
		metrics:  metrics,
		registry: registry,
		puller:   metricPuller,
	}, nil
}

// Creates standard Prometheus HTTP handler.
func (c *PrometheusControl) SetupHandler(next http.Handler) http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})
}

// Stops periodically collecting the metrics and unregisters
// all metrics.
func (c *PrometheusControl) Shutdown() {
	c.puller.Shutdown()
	UnregisterAllMetrics(c.registry, c.metrics)
}
