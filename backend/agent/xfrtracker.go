package agent

import (
	"context"
)

// The default number of days to look into the past for the XFR tracking
// when the tracker starts analyzing the systemd logs.
const defaultXfrTrackingSinceDaysAgo = 1

// Zone transfer tracker uses the underlying log tracker to subscribe to the
// logs containing messages marking the beginning and end of the zone transfers
// initiated by BIND.
//
// Note that the log tracker is a common component owned by the monitor. It can manage
// many subscriptions to various logs. The XFR tracker is associated with a BIND 9
// daemon instance and it establishes a single subscription over the log tracker.
//
// The XFR is not safe for concurrent use. It is intended to be used by the BIND 9
// daemon detection routines. These routines are executed sequentially.
//
// TODO: Current implementation is a stub. It subscribes to the log events, but does
// not process them. The XFR tracker is going to be extended to capture and collect
// the zone transfer logs. The logs will be converted to the zone transfer events,
// stored here, and will have proper indexes, so they can be retrieved over gRPC.
type xfrTracker struct {
	logTracker *logTracker
	subscriber *logTrackingSubscriber
	cancelFn   context.CancelFunc
	cancelCh   chan struct{}
}

// Instantiates a new XFR tracker instance. It is associated with the log
// tracker instance specified as an argument. The log tracker instance
// must be non-nil.
func newXfrTracker(logTracker *logTracker) *xfrTracker {
	return &xfrTracker{
		logTracker: logTracker,
	}
}

// Tracks the log file or systemd logs specified as options. If a subscription
// already exists, it is replaced with a new one. The old subscription is stopped.
func (t *xfrTracker) track(options ...logReaderCaptureOption) error {
	// Make sure that the old subscription is stopped, if any.
	t.stop()
	// Create new subscription using specified options.
	subscriber, err := t.logTracker.subscribe(options...)
	if err != nil {
		return err
	}
	// Create the cancellation context, so we can stop the goroutine that consumes
	// the log lines.
	ctx, cancel := context.WithCancel(context.Background())
	// Create the cancellation channel that will be waited after stopping the goroutine.
	cancelCh := make(chan struct{})
	go func() {
		defer close(cancelCh)
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-subscriber.dataChan:
				if !ok {
					return
				}
			}
		}
	}()
	// Remember the subscription, cancellation function and the channel, so we
	// can use them to stop the subscription in the stop() function.
	t.subscriber = subscriber
	t.cancelFn = cancel
	t.cancelCh = cancelCh
	return nil
}

// Tracks the log file specified as an argument. The XFR tracker will read the
// log file from the start, and then follow the new lines.
func (t *xfrTracker) trackFile(filename string) (err error) {
	return t.track(logReaderCaptureOptionFileName(filename), logReaderCaptureOptionFollow())
}

// Tracks the logs of the systemd unit specified as an argument. The XFR tracker will read the
// logs starting from "yesterday" and then follow the new lines.
func (t *xfrTracker) trackSystemdUnit(unitName string) (err error) {
	return t.track(logReaderCaptureOptionUnitName(unitName), logReaderCaptureOptionFollow(), logReaderCaptureOptionSinceDaysAgo(defaultXfrTrackingSinceDaysAgo))
}

// Stops the XFR tracker. It stops the subscription, cancels the goroutine that
// consumes the log lines, and releases the resources.
func (t *xfrTracker) stop() {
	// Stop the subscription.
	if t.cancelFn != nil {
		t.cancelFn()
		t.cancelFn = nil
	}
	if t.cancelCh != nil {
		<-t.cancelCh
		t.cancelCh = nil
	}
	if t.subscriber != nil {
		t.subscriber.stop()
		t.subscriber = nil
	}
}
