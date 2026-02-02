package eventcenter

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Check SSEBroker.
func TestSSEBroker(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ec := NewEventCenter(db)

	req := httptest.NewRequest("GET", "http://localhost/sse?stream=message", nil)
	w := httptest.NewRecorder()
	context, cancel := context.WithCancel(context.Background())
	req = req.WithContext(context)

	go func() {
		defer cancel()
		assert.Eventually(t, func() bool {
			return ec.(*eventCenter).sseBroker.getSubscribersCount() > 0
		}, 100*time.Millisecond, 10*time.Millisecond)

		ev := &dbmodel.Event{
			ID:        42,
			Text:      "some text",
			Level:     dbmodel.EvWarning,
			CreatedAt: time.Date(2000, time.February, 3, 4, 5, 6, 7, time.UTC),
			Relations: &dbmodel.Relations{MachineID: 24},
			Details:   "detail text",
		}
		ec.(*eventCenter).sseBroker.dispatchEvent(ev)
	}()

	ec.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "data: {\"createdAt\":\"2000-02-03T04:05:06.000Z\",\"details\":\"detail text\",\"id\":42,\"level\":1,\"text\":\"some text\"}\n\n", string(body))
}

// Check SSEBroker shutdown.
func TestSSEBrokerShutdown(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ec := NewEventCenter(db)

	// Serve the request in background.
	req := httptest.NewRequest("GET", "http://localhost/sse?stream=message", nil)
	w := httptest.NewRecorder()
	go ec.ServeHTTP(w, req)

	// We should establish new connection eventually.
	require.Eventually(t, func() bool {
		return ec.(*eventCenter).sseBroker.getSubscribersCount() > 0
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Shutdown event center.
	ec.Shutdown()

	// It should eventually remove the connection.
	require.Eventually(t, func() bool {
		return ec.(*eventCenter).sseBroker.getSubscribersCount() == 0
	}, 100*time.Millisecond, 10*time.Millisecond)
}

// Check that SSEBroker dispatches events to non-main streams.
func TestSSEBrokerNonMainStream(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ec := NewEventCenter(db)

	req := httptest.NewRequest("GET", "http://localhost/sse?stream=connectivity", nil)
	w := httptest.NewRecorder()
	context, cancel := context.WithCancel(context.Background())
	req = req.WithContext(context)

	go func() {
		defer cancel()
		assert.Eventually(t, func() bool {
			return ec.(*eventCenter).sseBroker.getSubscribersCount() > 0
		}, 100*time.Millisecond, 10*time.Millisecond)

		ev := &dbmodel.Event{
			Text:       "some text",
			Level:      dbmodel.EvInfo,
			SSEStreams: []dbmodel.SSEStream{"connectivity", "ha"},
		}
		ec.(*eventCenter).sseBroker.dispatchEvent(ev)
	}()

	ec.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	require.Contains(t, string(body), "event: connectivity\n")
	require.Contains(t, string(body), "data: {\"createdAt\":\"0001-01-01T00:00:00.000Z\",\"text\":\"some text\"}\n\n")
}

// Check that SSEBroker dispatches events to multiple streams.
func TestSSEBrokerWithDifferentStreams(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ec := NewEventCenter(db)

	req := httptest.NewRequest("GET", "http://localhost/sse?stream=message&stream=connectivity&stream=ha", nil)
	w := httptest.NewRecorder()
	context, cancel := context.WithCancel(context.Background())
	req = req.WithContext(context)

	go func() {
		defer cancel()
		assert.Eventually(t, func() bool {
			return ec.(*eventCenter).sseBroker.getSubscribersCount() > 0
		}, 100*time.Millisecond, 10*time.Millisecond)

		ev := &dbmodel.Event{
			ID:         42,
			Text:       "some text",
			Level:      dbmodel.EvInfo,
			SSEStreams: []dbmodel.SSEStream{"connectivity", "ha"},
		}
		ec.(*eventCenter).sseBroker.dispatchEvent(ev)
	}()

	ec.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	require.Contains(t, string(body), "data: {\"createdAt\":\"0001-01-01T00:00:00.000Z\",\"id\":42,\"text\":\"some text\"}\n\n")
	require.Contains(t, string(body), "event: connectivity\ndata: {\"createdAt\":\"0001-01-01T00:00:00.000Z\",\"id\":42,\"text\":\"some text\"}\n\n")
	require.Contains(t, string(body), "event: ha\ndata: {\"createdAt\":\"0001-01-01T00:00:00.000Z\",\"id\":42,\"text\":\"some text\"}\n\n")
}
