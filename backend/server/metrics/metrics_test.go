package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// All metrics should be properly constructed.
func TestNewMetrics(t *testing.T) {
	// Act
	metrics := newMetrics(nil)
	mfs, _ := metrics.Registry.Gather()

	// Arrange
	require.NotNil(t, metrics)
	// Prometheus has lazy-initialization of the metrics.
	// Only the metrics with at least one value are
	// enumerated by the gather.
	// The 3 metrics are a single counters (Gauge), they
	// are initialized with 0 value at the beginning.
	// Rest of metrics are vectors (GaugeVectors), they have
	// no value at the beginning.
	require.Len(t, mfs, 3)
}

// All metrics should be unregistered.
func TestUnregisterAllMetrics(t *testing.T) {
	// Arrange
	metrics := newMetrics(nil)

	// Act
	metrics.UnregisterAll()
	mfs, _ := metrics.Registry.Gather()

	// Arrange
	require.Empty(t, mfs)
}
