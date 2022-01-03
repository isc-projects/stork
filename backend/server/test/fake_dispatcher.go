package storktest

import (
	"isc.org/stork/server/configreview"
	dbmodel "isc.org/stork/server/database/model"
)

// Mock implementation of the configuration review dispatcher.
// It substitutes the default dispatcher implementation in the
// unit tests.
type FakeDispatcher struct {
	CallLog    []string
	Signature  string
	InProgress bool
}

func (d *FakeDispatcher) RegisterChecker(selector configreview.DispatchGroupSelector, checkerName string, triggers configreview.Triggers, checkFn func(*configreview.ReviewContext) (*configreview.Report, error)) {
	d.CallLog = append(d.CallLog, "RegisterChecker")
}

func (d *FakeDispatcher) UnregisterChecker(selector configreview.DispatchGroupSelector, checkerName string) bool {
	d.CallLog = append(d.CallLog, "UnregisterChecker")
	return true
}

func (d *FakeDispatcher) GetSignature() string {
	d.CallLog = append(d.CallLog, "GetSignature")
	return d.Signature
}

func (d *FakeDispatcher) Start() {
	d.CallLog = append(d.CallLog, "Start")
}

func (d *FakeDispatcher) Shutdown() {
	d.CallLog = append(d.CallLog, "Shutdown")
}

func (d *FakeDispatcher) BeginReview(daemon *dbmodel.Daemon, trigger configreview.Trigger, callback configreview.CallbackFunc) bool {
	d.CallLog = append(d.CallLog, "BeginReview")
	return true
}

func (d *FakeDispatcher) ReviewInProgress(daemonID int64) bool {
	d.CallLog = append(d.CallLog, "ReviewInProgress")
	return d.InProgress
}
