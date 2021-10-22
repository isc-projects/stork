package storktest

import (
	"isc.org/stork/server/configreview"
	dbmodel "isc.org/stork/server/database/model"
)

// Mock implementation of the configuration review dispatcher.
// It substitutes the default dispatcher implementation in the
// unit tests.
type FakeDispatcher struct {
	CallLog []string
}

func (d *FakeDispatcher) RegisterProducer(selector configreview.DispatchGroupSelector, producerName string, produceFn func(*configreview.ReviewContext) (*configreview.Report, error)) {
	d.CallLog = append(d.CallLog, "RegisterProducer")
}

func (d *FakeDispatcher) RegisterDefaultProducers() {
	d.CallLog = append(d.CallLog, "RegisterProducer")
}

func (d *FakeDispatcher) Start() {
	d.CallLog = append(d.CallLog, "Start")
}

func (d *FakeDispatcher) Shutdown() {
	d.CallLog = append(d.CallLog, "Shutdown")
}

func (d *FakeDispatcher) BeginReview(daemon *dbmodel.Daemon, callback configreview.CallbackFunc) bool {
	d.CallLog = append(d.CallLog, "BeginReview")
	return true
}

