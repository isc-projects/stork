package storktestdbmodel

import (
	"isc.org/stork/server/configreview"
	dbmodel "isc.org/stork/server/database/model"
)

// Represents a single call to FakeDispatcher.
type FakeDispatcherCall struct {
	CallName string
	DaemonID int64
	Trigger  configreview.Trigger
}

// Mock implementation of the configuration review dispatcher.
// It substitutes the default dispatcher implementation in the
// unit tests.
type FakeDispatcher struct {
	CallLog    []FakeDispatcherCall
	Signature  string
	InProgress bool
}

var _ configreview.Dispatcher = (*FakeDispatcher)(nil)

func (d *FakeDispatcher) RegisterChecker(selector configreview.DispatchGroupSelector, checkerName string, triggers configreview.Triggers, checkFn func(*configreview.ReviewContext) (*configreview.Report, error)) {
	d.CallLog = append(d.CallLog, FakeDispatcherCall{CallName: "RegisterChecker"})
}

func (d *FakeDispatcher) UnregisterChecker(selector configreview.DispatchGroupSelector, checkerName string) bool {
	d.CallLog = append(d.CallLog, FakeDispatcherCall{CallName: "UnregisterChecker"})
	return true
}

func (d *FakeDispatcher) GetCheckersMetadata(daemonID int64, daemonName string) []*configreview.CheckerMetadata {
	d.CallLog = append(d.CallLog, FakeDispatcherCall{CallName: "GetCheckersMetadata"})
	return []*configreview.CheckerMetadata{}
}

func (d *FakeDispatcher) GetSignature() string {
	d.CallLog = append(d.CallLog, FakeDispatcherCall{CallName: "GetSignature"})
	return d.Signature
}

func (d *FakeDispatcher) Start() {
	d.CallLog = append(d.CallLog, FakeDispatcherCall{CallName: "Start"})
}

func (d *FakeDispatcher) Shutdown() {
	d.CallLog = append(d.CallLog, FakeDispatcherCall{CallName: "Shutdown"})
}

func (d *FakeDispatcher) BeginReview(daemon *dbmodel.Daemon, trigger configreview.Trigger, callback configreview.CallbackFunc) bool {
	d.CallLog = append(d.CallLog, FakeDispatcherCall{CallName: "BeginReview", DaemonID: daemon.ID, Trigger: trigger})
	return true
}

func (d *FakeDispatcher) ReviewInProgress(daemonID int64) bool {
	d.CallLog = append(d.CallLog, FakeDispatcherCall{CallName: "ReviewInProgress", DaemonID: daemonID})
	return d.InProgress
}
