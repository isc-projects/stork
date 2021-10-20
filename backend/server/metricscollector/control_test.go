package metricscollector

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

	metric, ok := mf["stork_server_auth_authorized_machine_total"]
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

// Test that the control is properly created.
func TestConstructControler(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	control := NewControl(db)
	defer control.Shutdown()

	// Assert
	require.NotNil(t, control)
}

// Test that the HTTP handler is created.
func TestCreateHttpHandler(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	control := NewControl(db)
	defer control.Shutdown()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Act
	handler := control.SetupHandler(nextHandler)

	// Assert
	require.NotNil(t, handler)
}

// Test that the handler responses with proper content.
func TestHandlerResponse(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	control := NewControl(db)
	defer control.Shutdown()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := control.SetupHandler(nextHandler)
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
	require.EqualValues(t, 0, authorizedCount)
}

// Test that the metrics are updated periodically.
func TestPeriodicallyMetricsUpdate(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db)
	_ = dbmodel.SetSettingInt(db, "metrics_collector_interval", 1)

	control := NewControl(db)
	defer control.Shutdown()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := control.SetupHandler(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/abc", nil)
	w := httptest.NewRecorder()

	// Act
	_ = dbmodel.AddMachine(db, &dbmodel.Machine{
		Address:    "127.0.0.1",
		AgentPort:  8000,
		Authorized: true,
	})
	time.Sleep(5 * time.Second)

	handler.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	authorizedCount, _ := parseAuthorizedMachinesFromPrometheus(resp.Body)

	// Assert
	require.EqualValues(t, 1, authorizedCount)
}
