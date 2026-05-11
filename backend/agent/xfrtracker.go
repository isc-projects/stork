package agent

import (
	"context"
	"fmt"
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
	// The log tracker instance used to create the subscriptions.
	logTracker *logTracker
	// Subscriptions to the XFR-related logs. The number of subscriptions is determined
	// by the number of log files to track. The incoming and outgoing XFR requests can
	// be logged in the same log file, or in different log files. Also, tracking any
	// of them can be disabled.
	subscribers []*logTrackingSubscriber
	// The cancellation function used to stop the goroutine that consumes the log lines.
	cancelFn context.CancelFunc
	// The channel used to wait for the cancellation of the goroutine that consumes the
	// log lines.
	cancelCh chan struct{}
}

// Instantiates a new XFR tracker instance. It is associated with the log
// tracker instance specified as an argument. The log tracker instance
// must be non-nil.
func newXfrTracker(logTracker *logTracker) *xfrTracker {
	return &xfrTracker{
		logTracker: logTracker,
	}
}

// Tracks the log file or systemd logs using created subscriptions.
func (t *xfrTracker) track() error {
	// Create the cancellation context, so we can stop the goroutine that consumes
	// the log lines.
	ctx, cancel := context.WithCancel(context.Background())
	// Create the cancellation channel that will be waited after stopping the goroutine.
	cancelCh := make(chan struct{})
	// Get the channels from the subscriptions. It is ok if some channels are nil.
	// Reading from nil channel is supported in go. It will block never returning
	// a value.
	var chan0, chan1 chan logReaderLine
	for _, subscriber := range t.subscribers {
		if subscriber != nil {
			switch {
			case chan0 == nil:
				chan0 = subscriber.dataChan
			case chan1 == nil:
				chan1 = subscriber.dataChan
			}
		}
	}
	go func() {
		defer close(cancelCh)
		for {
			select {
			case <-ctx.Done():
				return
			case logLine, ok := <-chan0:
				if !ok {
					return
				}
				fmt.Println("Incoming XFR request:", logLine.text)
			case logLine, ok := <-chan1:
				if !ok {
					return
				}
				fmt.Println("Outgoing XFR request:", logLine.text)
			}
		}
	}()
	// Remember the cancellation function and the channel, so we can use them to
	// stop the subscriptions in the stop() function.
	t.cancelFn = cancel
	t.cancelCh = cancelCh
	return nil
}

// Tracks the log files specified as arguments. It is supported to track up to two log files.
// The XFR tracker will read the log files from the start, and then follow the new lines.
// If file names are the same, only one subscription is created. No subscription is created
// for the empty file name.
func (t *xfrTracker) trackFiles(filename1, filename2 string) (err error) {
	// Make sure that the old subscriptions are stopped, if any.
	t.stop()
	var filenames []string
	if filename1 != "" {
		filenames = append(filenames, filename1)
	}
	if filename2 != "" && filename2 != filename1 {
		filenames = append(filenames, filename2)
	}
	for _, filename := range filenames {
		// Create the subscription to the XFR requests.
		subscriber, err := t.logTracker.subscribe(logReaderCaptureOptionFileName(filename), logReaderCaptureOptionFollow())
		if err != nil {
			return err
		}
		t.subscribers = append(t.subscribers, subscriber)
	}
	return t.track()
}

// Tracks the logs of the systemd unit specified as an argument. The XFR tracker will read the
// logs starting from "yesterday" and then follow the new lines.
func (t *xfrTracker) trackSystemdUnit(unitName string) (err error) {
	// Make sure that the old subscriptions are stopped, if any.
	t.stop()
	subscriber, err := t.logTracker.subscribe(logReaderCaptureOptionUnitName(unitName), logReaderCaptureOptionFollow(), logReaderCaptureOptionSinceDaysAgo(defaultXfrTrackingSinceDaysAgo))
	if err != nil {
		return err
	}
	t.subscribers = append(t.subscribers, subscriber)
	return t.track()
}

// Stops the XFR tracker. It stops the subscriptions, cancels the goroutine that
// consumes the log lines, and releases the resources.
func (t *xfrTracker) stop() {
	// Stop the subscriptions.
	if t.cancelFn != nil {
		t.cancelFn()
		t.cancelFn = nil
	}
	if t.cancelCh != nil {
		<-t.cancelCh
		t.cancelCh = nil
	}
	for _, subscriber := range t.subscribers {
		if subscriber != nil {
			subscriber.stop()
		}
	}
	t.subscribers = nil
}
