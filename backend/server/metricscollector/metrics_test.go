package metricscollector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// All metrics should be properly constructed.
func TestNewMetrics(t *testing.T) {
	// Arrange
	registry := prometheus.NewRegistry()

	// Act
	metrics := NewMetrics(registry)
	mfs, _ := registry.Gather()

	// Arrange
	require.NotNil(t, metrics)
	// Prometheus doesn't include the vector metics
	// until it has no value.
	require.Len(t, mfs, 3)
}

// All metrics should be unregistered.
func TestUnregisterAllMetrics(t *testing.T) {
	// Arrange
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	// Act
	UnregisterAllMetrics(registry, metrics)
	mfs, _ := registry.Gather()

	// Arrange
	require.Len(t, mfs, 0)
}
