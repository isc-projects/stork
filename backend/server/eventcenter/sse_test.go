package eventcenter

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Check SSEBroker.
func TestSSEBroker(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ec := NewEventCenter(db)

	req := httptest.NewRequest("GET", "http://localhost/sse", nil)
	w := httptest.NewRecorder()
	context, cancel := context.WithCancel(context.Background())
	req = req.WithContext(context)

	go func() {
		for i := 1; i <= 10; i++ {
			time.Sleep(10 * time.Millisecond)
			cnt := ec.(*eventCenter).sseBroker.getSubscribersCount()
			if cnt > 0 {
				break
			}
		}
		ev := &dbmodel.Event{
			Text:  "some text",
			Level: dbmodel.EvInfo,
		}
		ec.(*eventCenter).sseBroker.dispatchEvent(ev)

		cancel()
	}()

	ec.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "data: {\"ID\":0,\"CreatedAt\":\"0001-01-01T00:00:00Z\",\"Text\":\"some text\",\"Level\":0,\"Relations\":null,\"Details\":\"\"}\n\n", string(body))
}
