package storktestdbmodel

import (
	"net/http"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Helper struct to mock EventCenter behavior.
type FakeEventCenter struct {
	Events []*dbmodel.Event
}

// Creates and aggregates an info event.
func (fec *FakeEventCenter) AddInfoEvent(text string, objects ...interface{}) {
	fec.AddEvent(eventcenter.CreateEvent(dbmodel.EvInfo, text, objects...))
}

// Creates and aggregates an warning event.
func (fec *FakeEventCenter) AddWarningEvent(text string, objects ...interface{}) {
	fec.AddEvent(eventcenter.CreateEvent(dbmodel.EvWarning, text, objects...))
}

// Creates and aggregates an error event.
func (fec *FakeEventCenter) AddErrorEvent(text string, objects ...interface{}) {
	fec.AddEvent(eventcenter.CreateEvent(dbmodel.EvError, text, objects...))
}

// Aggregates an event.
func (fec *FakeEventCenter) AddEvent(event *dbmodel.Event) {
	fec.Events = append(fec.Events, event)
}

// Do nothing.
func (fec *FakeEventCenter) Shutdown() {
}

// Do nothing.
func (fec *FakeEventCenter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
}
