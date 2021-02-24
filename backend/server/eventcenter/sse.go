package eventcenter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

// SSE Broker. It stores subscribers in a map which is protected by mutex.
type SSEBroker struct {
	db               *dbops.PgDB
	subscribers      map[chan []byte]*Subscriber
	subscribersMutex *sync.Mutex
}

// Create a new SSE Broker.
func NewSSEBroker(db *dbops.PgDB) *SSEBroker {
	sb := &SSEBroker{
		db:               db,
		subscribers:      map[chan []byte]*Subscriber{},
		subscribersMutex: &sync.Mutex{},
	}
	return sb
}

// Server SSE request for new session.
func (sb *SSEBroker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s := newSubscriber(req.URL)

	if err := s.applyFiltersFromQuery(sb.db); err != nil {
		log.Errorf("failed to accept new SSE connection because query parameters are invalid: %+v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("new SSE subscriber from %s", req.RemoteAddr)

	// prepare proper HTTP headers for SSE response
	h := w.Header()
	h.Set("Connection", "keep-alive")
	h.Set("Cache-Control", "no-cache")
	h.Set("Content-Type", "text/event-stream")
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("X-Accel-Buffering", "no") // make nginx working: https://blog.icod.de/2018/12/17/angular-eventsource-go-and-wasted-lifetime/

	// create a subscriber and a channel which is used
	// to dispatch an event to this subscriber
	ch := make(chan []byte)

	// store subscriber and its channel in a map, protect the map with mutex
	sb.subscribersMutex.Lock()
	sb.subscribers[ch] = s
	sb.subscribersMutex.Unlock()

	// and now listen for events dispatched to this subscriber
	for {
		select {
		case event := <-ch:
			// send received event to subscriber and flush the connection
			log.Printf("to %p sent %s", s, event)
			fmt.Fprintf(w, "data: %s\n\n", event)
			// Not all ResponseWriter instances implement http.Flusher interface.
			// Test if this instance implement it before attempting to use it.
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

		case <-req.Context().Done():
			// connection is closed so unsubscribe subscriber
			log.Printf("connection with %p closed", s)
			sb.subscribersMutex.Lock()
			delete(sb.subscribers, ch)
			sb.subscribersMutex.Unlock()
			return
		}
	}
}

// Dispatch event to subscribers using filtering.
func (sb *SSEBroker) dispatchEvent(event *dbmodel.Event) {
	sb.subscribersMutex.Lock()
	defer sb.subscribersMutex.Unlock()

	evJSON, err := json.Marshal(event)
	if err != nil {
		log.Errorf("problem with serializing event to json: %+v", err)
		return
	}

	for ch := range sb.subscribers {
		if sb.subscribers[ch].AcceptsEvent(event) {
			ch <- evJSON
		}
	}
}

// Get count of subscribers.
func (sb *SSEBroker) getSubscribersCount() int {
	sb.subscribersMutex.Lock()
	defer sb.subscribersMutex.Unlock()
	return len(sb.subscribers)
}
