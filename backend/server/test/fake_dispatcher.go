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

func (d *FakeDispatcher) RegisterChecker(selector configreview.DispatchGroupSelector, checkerName string, checkFn func(*configreview.ReviewContext) (*configreview.Report, error)) {
	d.CallLog = append(d.CallLog, "RegisterChecker")
}

func (d *FakeDispatcher) UnregisterChecker(selector configreview.DispatchGroupSelector, checkerName string) bool {
	d.CallLog = append(d.CallLog, "UnregisterChecker")
	return true
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
