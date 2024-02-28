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
			Text:  "some text",
			Level: dbmodel.EvInfo,
		}
		ec.(*eventCenter).sseBroker.dispatchEvent(ev)
	}()

	ec.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "data: {\"ID\":0,\"CreatedAt\":\"0001-01-01T00:00:00Z\",\"Text\":\"some text\",\"Level\":0,\"Relations\":null,\"Details\":\"\",\"SSEStreams\":[\"message\"]}\n\n", string(body))
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
	require.Contains(t, string(body), "data: {\"ID\":0,\"CreatedAt\":\"0001-01-01T00:00:00Z\",\"Text\":\"some text\",\"Level\":0,\"Relations\":null,\"Details\":\"\",\"SSEStreams\":[\"connectivity\"]}\n\n")
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
	require.Contains(t, string(body), "data: {\"ID\":0,\"CreatedAt\":\"0001-01-01T00:00:00Z\",\"Text\":\"some text\",\"Level\":0,\"Relations\":null,\"Details\":\"\",\"SSEStreams\":[\"message\",\"connectivity\",\"ha\"]}\n\n")
	require.Contains(t, string(body), "event: connectivity\ndata: {\"ID\":0,\"CreatedAt\":\"0001-01-01T00:00:00Z\",\"Text\":\"some text\",\"Level\":0,\"Relations\":null,\"Details\":\"\",\"SSEStreams\":[\"message\",\"connectivity\",\"ha\"]}\n\n")
	require.Contains(t, string(body), "event: ha\ndata: {\"ID\":0,\"CreatedAt\":\"0001-01-01T00:00:00Z\",\"Text\":\"some text\",\"Level\":0,\"Relations\":null,\"Details\":\"\",\"SSEStreams\":[\"message\",\"connectivity\",\"ha\"]}\n\n")
}
