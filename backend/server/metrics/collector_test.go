package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Helper function to extract number of authorized machines
// from Prometheus metrics.
// Source: https://stackoverflow.com/a/65388822.
func parseAuthorizedMachinesFromPrometheus(input io.Reader) (int64, error) {
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(input)
	if err != nil {
		return 0, err
	}

	metric, ok := mf["storkserver_auth_authorized_machine_total"]
	if !ok {
		return 0, errors.Errorf("missing metric")
	}

	if len(metric.Metric) != 1 {
		return 0, errors.Errorf("too many metrics")
	}

	gauge := metric.Metric[0].GetGauge()

	if gauge == nil {
		return 0, errors.Errorf("unexpected metric type")
	}

	return int64(*gauge.Value), nil
}

// Test that the collector is properly created.
func TestConstructController(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)

	// Act
	collector, err := NewCollector(db)
	defer collector.Shutdown()

	// Assert
	require.NotNil(t, collector)
	require.NoError(t, err)
	mfs, _ := collector.(*prometheusCollector).registry.Gather()
	// Prometheus has lazy-initialization of the metrics.
	// Only the metrics with at least one value are
	// enumerated by the gather.
	// The 3 metrics are single counters (Gauge), they
	// are initialized with 0 value at the beginning.
	// Other metrics are vectors (GaugeVectors), they have
	// no value at the beginning.
	require.Len(t, mfs, 3)
}

// Test that the HTTP handler is created.
func TestCreateHttpHandler(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	collector, _ := NewCollector(db)
	defer collector.Shutdown()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Act
	handler := collector.GetHTTPHandler(nextHandler)

	// Assert
	require.NotNil(t, handler)
}

// Test that the handler responses with proper content.
func TestHandlerResponse(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	collector, _ := NewCollector(db)
	defer collector.Shutdown()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := collector.GetHTTPHandler(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/abc", nil)
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	authorizedCount, err := parseAuthorizedMachinesFromPrometheus(resp.Body)

	// Assert
	require.EqualValues(t, 200, resp.StatusCode)
	require.NoError(t, err)
	require.Zero(t, authorizedCount)
}

// Test that the metrics are updated on demand.
func TestCollect(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	collector, _ := NewCollector(db)
	defer collector.Shutdown()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := collector.GetHTTPHandler(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/abc", nil)

	// Act
	_ = dbmodel.AddMachine(db, &dbmodel.Machine{
		Address:    "127.0.0.1",
		AgentPort:  8000,
		Authorized: true,
	})

	require.Eventually(t, func() bool {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		authorizedCount, _ := parseAuthorizedMachinesFromPrometheus(resp.Body)

		// Assert
		return authorizedCount == 1
	}, 5*time.Second, 100*time.Millisecond)
}

// All metrics should be unregistered.
func TestUnregisterAllMetrics(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	collector, _ := NewCollector(db)
	defer collector.Shutdown()

	// Act
	collector.Shutdown()
	mfs, _ := collector.(*prometheusCollector).registry.Gather()

	// Arrange
	require.Empty(t, mfs)
}
