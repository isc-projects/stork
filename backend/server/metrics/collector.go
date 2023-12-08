package metrics

import (
	"net/http"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

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
	metrics *metrics
	puller  *storkutil.PeriodicExecutor
}

// Creates an instance of the metrics collector and starts
// collecting the metrics according to the interval
// specified in the database.
func NewCollector(db *pg.DB) (Collector, error) {
	metrics := newMetrics(db)
	intervalSettingName := "metrics_collector_interval"

	// Initialize the metrics
	err := metrics.Update()
	if err != nil {
		return nil, errors.WithMessage(err, "error during metrics initialization")
	}

	// Starts collecting the metrics periodically.
	metricPuller, err := storkutil.NewPeriodicExecutor("metrics collector",
		metrics.Update,
		func() (int64, error) {
			interval, err := dbmodel.GetSettingInt(db, intervalSettingName)
			return interval, errors.WithMessagef(err, "problem getting interval setting %s from db",
				intervalSettingName)
		},
	)
	if err != nil {
		return nil, err
	}

	return &prometheusCollector{
		metrics: metrics,
		puller:  metricPuller,
	}, nil
}

// Creates standard Prometheus HTTP handler.
func (c *prometheusCollector) GetHTTPHandler(next http.Handler) http.Handler {
	return promhttp.HandlerFor(c.metrics.Registry, promhttp.HandlerOpts{
		ErrorLog: logrus.StandardLogger(),
	})
}

// Stops periodically collecting the metrics and unregisters
// all metrics.
func (c *prometheusCollector) Shutdown() {
	c.puller.Shutdown()
	c.metrics.UnregisterAll()
}
