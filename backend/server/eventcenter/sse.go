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
	subscribers      map[chan dbmodel.Event]*Subscriber
	subscribersMutex *sync.RWMutex
}

// Create a new SSE Broker.
func NewSSEBroker(db *dbops.PgDB) *SSEBroker {
	sb := &SSEBroker{
		db:               db,
		subscribers:      map[chan dbmodel.Event]*Subscriber{},
		subscribersMutex: &sync.RWMutex{},
	}
	return sb
}

// Server SSE request for new session.
func (sb *SSEBroker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s := newSubscriber(req.URL, req.RemoteAddr)

	if err := s.applyFiltersFromQuery(sb.db); err != nil {
		log.Errorf("Failed to accept new SSE connection because query parameters are invalid: %+v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("New SSE subscriber from %s", req.RemoteAddr)

	// prepare proper HTTP headers for SSE response
	h := w.Header()
	h.Set("Connection", "keep-alive")
	h.Set("Cache-Control", "no-cache")
	h.Set("Content-Type", "text/event-stream")
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("X-Accel-Buffering", "no") // make nginx working: https://blog.icod.de/2018/12/17/angular-eventsource-go-and-wasted-lifetime/

	// create a subscriber and a channel which is used
	// to dispatch an event to this subscriber
	ch := make(chan dbmodel.Event)

	// store subscriber and its channel in a map, protect the map with mutex
	sb.subscribersMutex.Lock()
	sb.subscribers[ch] = s
	sb.subscribersMutex.Unlock()

	// and now listen for events dispatched to this subscriber
	for {
		select {
		case event := <-ch:
			evJSON, err := json.Marshal(event)
			if err != nil {
				log.WithError(err).Error("Problem serializing event to json")
				continue
			}

			// send received event to subscriber and flush the connection
			log.WithFields(log.Fields{
				"subscriber": s,
				"event":      event.Text,
			}).Info("Sending an event to the subscriber")
			for _, message := range event.SSEStreams {
				if message != dbmodel.SSERegularMessage {
					fmt.Fprintf(w, "event: %s\n", message)
				}
				fmt.Fprintf(w, "data: %s\n\n", evJSON)
			}

			// Not all ResponseWriter instances implement http.Flusher interface.
			// Test if this instance implement it before attempting to use it.
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-s.done:
			// The server is shutting down.
			log.Printf("Shutting down connection from %p", s)
			return
		case <-req.Context().Done():
			// connection is closed so unsubscribe subscriber
			log.Printf("Connection with %p closed", s)
			sb.subscribersMutex.Lock()
			delete(sb.subscribers, ch)
			sb.subscribersMutex.Unlock()
			return
		}
	}
}

// Dispatch event to subscribers using filtering.
func (sb *SSEBroker) dispatchEvent(event *dbmodel.Event) {
	sb.subscribersMutex.RLock()
	defer sb.subscribersMutex.RUnlock()

	for ch := range sb.subscribers {
		streams := sb.subscribers[ch].findMatchingEventStreams(event)
		if len(streams) > 0 {
			event.SSEStreams = streams
			ch <- *event
		}
	}
}

// Shuts down the all SSE broker connections from subscribers.
func (sb *SSEBroker) shutdown() {
	sb.subscribersMutex.Lock()
	defer sb.subscribersMutex.Unlock()
	for ch, s := range sb.subscribers {
		s.done <- struct{}{}
		close(ch)
	}
	for ch := range sb.subscribers {
		close(sb.subscribers[ch].done)
		delete(sb.subscribers, ch)
	}
}

// Get count of subscribers.
func (sb *SSEBroker) getSubscribersCount() int {
	sb.subscribersMutex.RLock()
	defer sb.subscribersMutex.RUnlock()
	return len(sb.subscribers)
}
