package storktest

import (
	"net/http"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Helper struct to mock EventCenter behavior.
type FakeEventCenter struct {
	Events []*dbmodel.Event
}

func (fec *FakeEventCenter) AddInfoEvent(text string, objects ...interface{}) {
	fec.AddEvent(eventcenter.CreateEvent(dbmodel.EvInfo, text, objects...))
}

func (fec *FakeEventCenter) AddWarningEvent(text string, objects ...interface{}) {
	fec.AddEvent(eventcenter.CreateEvent(dbmodel.EvWarning, text, objects...))
}

func (fec *FakeEventCenter) AddErrorEvent(text string, objects ...interface{}) {
	fec.AddEvent(eventcenter.CreateEvent(dbmodel.EvError, text, objects...))
}

func (fec *FakeEventCenter) AddEvent(event *dbmodel.Event) {
	fec.Events = append(fec.Events, event)
}

func (fec *FakeEventCenter) Shutdown() {
}

func (fec *FakeEventCenter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
}
