package storktest

import "net/http"

// Fake metrics collector. It collects nothing, but
// counts received requests.
type FakeMetricsCollector struct {
	IsRunning    bool
	RequestCount int
}

// Constructs a new fake collector instance.
func NewFakeMetricsCollector() *FakeMetricsCollector {
	return &FakeMetricsCollector{
		IsRunning:    true,
		RequestCount: 0,
	}
}

// Returns a mock handler that counts the requests.
func (c *FakeMetricsCollector) GetHTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.RequestCount++
	})
}

// Changes the internal running state.
func (c *FakeMetricsCollector) Shutdown() {
	c.IsRunning = false
}
