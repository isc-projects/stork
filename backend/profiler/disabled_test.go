package profiler_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/profiler"
)

// Test that the disabled placeholder for the start profiler function doesn't
// panic.
func TestStartDoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		teardown := profiler.Start(2024)
		teardown()
	})
}

// Test that the disabled placeholder for the Start profiler function doesn't
// panic even if the port is wrong.
func TestStartDoesNotPanicOnWrongPort(t *testing.T) {
	require.NotPanics(t, func() {
		teardown := profiler.Start(0)
		teardown()
	})
}
