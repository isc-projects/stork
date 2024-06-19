package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// The easy to mock metrics source.
type mockMetricsSource struct {
	calculatedMetrics dbmodel.CalculatedMetrics
}

// Creates an instance of the mock metrics source.
func newMockMetricsSource() *mockMetricsSource {
	return &mockMetricsSource{}
}

// Creates an instance of the mock metrics source.
func (s *mockMetricsSource) GetCalculatedMetrics() (*dbmodel.CalculatedMetrics, error) {
	return &s.calculatedMetrics, nil
}

// Sets the calculated metrics for the mock source.
func (s *mockMetricsSource) Set(metrics dbmodel.CalculatedMetrics) {
	s.calculatedMetrics = metrics
}

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
func TestCollectorConstruct(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = dbmodel.InitializeSettings(db, 0)

	// Act
	collector, err := NewCollector(NewDatabaseMetricsSource(db))
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
func TestCollectorCreateHttpHandler(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	collector, _ := NewCollector(NewDatabaseMetricsSource(db))
	defer collector.Shutdown()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Act
	handler := collector.GetHTTPHandler(nextHandler)

	// Assert
	require.NotNil(t, handler)
}

// Test that the handler responses with proper content.
func TestCollectorHandlerResponse(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	collector, _ := NewCollector(NewDatabaseMetricsSource(db))
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
func TestCollectorCollectUsingDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	collector, _ := NewCollector(NewDatabaseMetricsSource(db))
	defer collector.Shutdown()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := collector.GetHTTPHandler(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/abc", nil)

	_ = dbmodel.AddMachine(db, &dbmodel.Machine{
		Address:    "127.0.0.1",
		AgentPort:  8000,
		Authorized: true,
	})

	// Act
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	authorizedCount, _ := parseAuthorizedMachinesFromPrometheus(resp.Body)

	// Assert
	require.EqualValues(t, 1, authorizedCount)
}

// Test that the metrics are described.
func TestCollectorDescribe(t *testing.T) {
	// Arrange
	source := newMockMetricsSource()
	collector, _ := NewCollector(source)
	promCollector := collector.(prometheus.Collector)
	expectedDescriptionCount := 11

	t.Run("initial metrics values", func(t *testing.T) {
		source.Set(dbmodel.CalculatedMetrics{})
		descriptionsChannel := make(chan *prometheus.Desc, 100)

		// Act
		promCollector.Describe(descriptionsChannel)

		// Assert
		close(descriptionsChannel)
		require.Len(t, descriptionsChannel, expectedDescriptionCount)
	})

	t.Run("metrics values with data", func(t *testing.T) {
		source.Set(dbmodel.CalculatedMetrics{
			AuthorizedMachines:   1,
			UnauthorizedMachines: 2,
			UnreachableMachines:  3,
			SubnetMetrics: []dbmodel.CalculatedNetworkMetrics{{
				Label:           "subnet4",
				AddrUtilization: 5,
				PdUtilization:   6,
				Family:          7,
			}},
			SharedNetworkMetrics: []dbmodel.CalculatedNetworkMetrics{{
				Label:           "shared8",
				AddrUtilization: 9,
				PdUtilization:   10,
				Family:          11,
				SharedNetworkStats: dbmodel.SubnetStats{
					dbmodel.SubnetStatsNameTotalNAs:    uint64(12),
					dbmodel.SubnetStatsNameAssignedNAs: uint64(13),
					dbmodel.SubnetStatsNameDeclinedNAs: uint64(14),
					dbmodel.SubnetStatsNameTotalPDs:    uint64(15),
					dbmodel.SubnetStatsNameAssignedPDs: uint64(16),
				},
			}},
		})

		descriptionsChannel := make(chan *prometheus.Desc, 100)

		// Act
		promCollector.Describe(descriptionsChannel)

		// Assert
		close(descriptionsChannel)
		require.Len(t, descriptionsChannel, expectedDescriptionCount)
	})
}

// Test that the metrics are collected.
func TestCollectorCollect(t *testing.T) {
	// Arrange
	source := newMockMetricsSource()
	collector, _ := NewCollector(source)
	promCollector := collector.(prometheus.Collector)

	t.Run("initial metrics values", func(t *testing.T) {
		source.Set(dbmodel.CalculatedMetrics{})
		metricsChannel := make(chan prometheus.Metric, 100)

		// Act
		promCollector.Collect(metricsChannel)

		// Assert
		close(metricsChannel)
		// Only the machine counters are initialized with 0 value.
		// The other metrics are vectors, they have no value at the beginning.
		require.Len(t, metricsChannel, 3)
	})

	t.Run("metrics values with data", func(t *testing.T) {
		source.Set(dbmodel.CalculatedMetrics{
			AuthorizedMachines:   1,
			UnauthorizedMachines: 2,
			UnreachableMachines:  3,
			SubnetMetrics: []dbmodel.CalculatedNetworkMetrics{{
				Label: "subnet",
				// The utilizations are stored in DB multiplied by 1000.
				// 1000 (in DB) = 100% = 1.0 (in real)
				AddrUtilization: 4000,
				PdUtilization:   5000,
				Family:          4,
			}},
			SharedNetworkMetrics: []dbmodel.CalculatedNetworkMetrics{{
				Label:           "shared",
				AddrUtilization: 6000,
				PdUtilization:   7000,
				Family:          6,
				SharedNetworkStats: dbmodel.SubnetStats{
					dbmodel.SubnetStatsNameTotalNAs:    uint64(8),
					dbmodel.SubnetStatsNameAssignedNAs: uint64(9),
					dbmodel.SubnetStatsNameTotalPDs:    uint64(10),
					dbmodel.SubnetStatsNameAssignedPDs: uint64(11),
				},
			}},
		})

		metricsChannel := make(chan prometheus.Metric, 100)

		// Act
		promCollector.Collect(metricsChannel)

		// Assert
		close(metricsChannel)
		require.Len(t, metricsChannel, 11)
		i := 0
		for metric := range metricsChannel {
			i++
			metricDTO := &dto.Metric{}
			err := metric.Write(metricDTO)
			require.NoError(t, err)
			require.EqualValues(t, i, *metricDTO.Gauge.Value)
		}
	})

	t.Run("metrics values with many subnets and shared networks", func(t *testing.T) {
		source.Set(dbmodel.CalculatedMetrics{
			AuthorizedMachines:   1,
			UnauthorizedMachines: 2,
			UnreachableMachines:  3,
			SubnetMetrics: []dbmodel.CalculatedNetworkMetrics{
				{
					Label: "subnetA",
					// The utilizations are stored in DB multiplied by 1000.
					// 1000 (in DB) = 100% = 1.0 (in real)
					AddrUtilization: 4000,
					PdUtilization:   5000,
					Family:          4,
				},
				{
					Label:           "subnetB",
					AddrUtilization: 6000,
					PdUtilization:   7000,
					Family:          6,
				},
			},
			SharedNetworkMetrics: []dbmodel.CalculatedNetworkMetrics{
				{
					Label:           "sharedA",
					AddrUtilization: 8000,
					PdUtilization:   0,
					Family:          4,
					SharedNetworkStats: dbmodel.SubnetStats{
						dbmodel.SubnetStatsNameTotalNAs:    uint64(9),
						dbmodel.SubnetStatsNameAssignedNAs: uint64(10),
						dbmodel.SubnetStatsNameTotalPDs:    uint64(0),
						dbmodel.SubnetStatsNameAssignedPDs: uint64(0),
					},
				},
				{
					Label:           "sharedB",
					AddrUtilization: 11000,
					PdUtilization:   12000,
					Family:          6,
					SharedNetworkStats: dbmodel.SubnetStats{
						dbmodel.SubnetStatsNameTotalNAs:    uint64(13),
						dbmodel.SubnetStatsNameAssignedNAs: uint64(14),
						dbmodel.SubnetStatsNameTotalPDs:    uint64(15),
						dbmodel.SubnetStatsNameAssignedPDs: uint64(16),
					},
				},
			},
		})

		metricsChannel := make(chan prometheus.Metric, 100)

		// Act
		promCollector.Collect(metricsChannel)

		// Assert
		close(metricsChannel)
		require.Len(t, metricsChannel, 16)
		i := 0
		for metric := range metricsChannel {
			i++
			metricDTO := &dto.Metric{}
			err := metric.Write(metricDTO)
			require.NoError(t, err)
			require.EqualValues(t, i, *metricDTO.Gauge.Value)
		}
	})

	t.Run("metrics values with a shared network containing IPv4 and IPv6 addresses", func(t *testing.T) {
		source.Set(dbmodel.CalculatedMetrics{
			SharedNetworkMetrics: []dbmodel.CalculatedNetworkMetrics{
				{
					Label:           "shared",
					AddrUtilization: 1000,
					PdUtilization:   2000,
					Family:          4,
					SharedNetworkStats: dbmodel.SubnetStats{
						dbmodel.SubnetStatsNameTotalNAs:    uint64(3),
						dbmodel.SubnetStatsNameAssignedNAs: uint64(4),
						dbmodel.SubnetStatsNameTotalPDs:    uint64(5),
						dbmodel.SubnetStatsNameAssignedPDs: uint64(6),
					},
				},
				{
					Label:           "shared",
					AddrUtilization: 7000,
					PdUtilization:   8000,
					Family:          6,
					SharedNetworkStats: dbmodel.SubnetStats{
						dbmodel.SubnetStatsNameTotalNAs:    uint64(10),
						dbmodel.SubnetStatsNameAssignedNAs: uint64(11),
						dbmodel.SubnetStatsNameTotalPDs:    uint64(12),
						dbmodel.SubnetStatsNameAssignedPDs: uint64(13),
					},
				},
			},
		})

		metricsChannel := make(chan prometheus.Metric, 100)

		// Act
		promCollector.Collect(metricsChannel)

		// Assert
		close(metricsChannel)
		i := 0
		for metric := range metricsChannel {
			i++

			metricDTO := &dto.Metric{}
			err := metric.Write(metricDTO)
			require.NoError(t, err)

			var labelNames []string
			for _, label := range metricDTO.Label {
				labelNames = append(labelNames, *label.Name)
			}

			switch i {
			case 1, 2, 3:
				// Server-related metrics.
				continue
			case 4:
				// Address utilization IPv4.
				require.EqualValues(t, 1, *metricDTO.Gauge.Value)
				require.Len(t, metricDTO.Label, 2)
				require.Contains(t, labelNames, "name")
				require.Contains(t, labelNames, "family")
			case 5, 6:
				// Total and assigned addresses IPv4.
				require.EqualValues(t, i-2, *metricDTO.Gauge.Value)
				require.Len(t, metricDTO.Label, 2)
				require.Contains(t, labelNames, "name")
				require.Contains(t, labelNames, "family")
			case 7:
				// Address utilization IPv6.
				require.EqualValues(t, 7, *metricDTO.Gauge.Value)
				require.Len(t, metricDTO.Label, 2)
				require.Contains(t, labelNames, "name")
				require.Contains(t, labelNames, "family")
			case 8:
				// PD utilization IPv6.
				require.EqualValues(t, 8, *metricDTO.Gauge.Value)
				// PD utilization hasn't the family label.
				require.Len(t, metricDTO.Label, 1)
				require.Contains(t, labelNames, "name")
				require.NotContains(t, labelNames, "family")
			case 9, 10:
				// Total and assigned addresses IPv6.
				require.EqualValues(t, i+1, *metricDTO.Gauge.Value)
				require.Len(t, metricDTO.Label, 2)
				require.Contains(t, labelNames, "name")
				require.Contains(t, labelNames, "family")
			case 11, 12:
				// Total and assigned PDs IPv6.
				require.EqualValues(t, i+1, *metricDTO.Gauge.Value)
				require.Len(t, metricDTO.Label, 1)
				require.Contains(t, labelNames, "name")
				require.NotContains(t, labelNames, "family")
			}
		}
	})
}

// All metrics should be unregistered.
func TestCollectorUnregisterAllMetrics(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	collector, _ := NewCollector(NewDatabaseMetricsSource(db))
	defer collector.Shutdown()

	// Act
	collector.Shutdown()
	mfs, _ := collector.(*prometheusCollector).registry.Gather()

	// Arrange
	require.Empty(t, mfs)
}
