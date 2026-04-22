package agent

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

// A subscriber represents a request to track the logs using specified options.
// The subscriber can be attached to a capture to follow the logs. It exposes the
// channel to receive the log lines from the capture by the caller.
type logTrackingSubscriber struct {
	// The channel used to send the captured log lines to the subscriber.
	dataChan chan logReaderLine
	// The context used to cancel the subscription.
	ctx context.Context
	// The function used to cancel the subscription.
	cancelFn context.CancelFunc
	// The guard to ensure that the signalReady function is called only once.
	signalReadyOnce sync.Once
	// The mutex to protect the subscriber state from concurrent access. Specifically,
	// it ensures that the data cannot be sent over the closed channel.
	mutex sync.RWMutex
	// The channel used to signal that the subscriber is ready to receive the log lines.
	readyChan chan struct{}
	// The channel used to signal that the subscriber is stopped.
	stoppedChan chan struct{}
	// The guard to ensure that the teardown function is called only once.
	teardownOnce sync.Once
}

// Instantiates a new subscriber. The context can be used to cancel subscription,
// and detach from the log capture.
func newLogTrackingSubscriber(ctx context.Context, channelSize int) *logTrackingSubscriber {
	ctx, cancel := context.WithCancel(ctx)
	return &logTrackingSubscriber{
		dataChan:    make(chan logReaderLine, channelSize),
		ctx:         ctx,
		cancelFn:    cancel,
		readyChan:   make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

// Sends the log line to the subscriber's receive channel. Returns true if the line
// was sent successfully, false if the subscriber was cancelled.
func (s *logTrackingSubscriber) consumeLogLine(line logReaderLine) bool {
	// Need to hold the mutex to prevent races between sending the log
	// line over the channel and closing the channel. Writing to the closed
	// channel will panic.
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.isStopped() {
		// Subscriber was stopped before we called this function.
		// It indicates that the subscriber channel was already closed.
		// Do not attempt to send to this channel to prevent panic.
		return false
	}
	select {
	case <-s.ctx.Done():
		// Subscriber context is cancelled (perhaps because the capture
		// is being cancelled). Stop sending the log lines to the subscriber.
		return false
	case s.dataChan <- line:
		// Otherwise, send it.
		return true
	}
}

// Stops the subscriber by closing its channel. It also marks the subscriber ready
// so any captures that are waiting for the subscriber to backfill the log before
// following the log are free to continue. This function is guaranteed to be executed
// only once. Subsequent calls are ignored.
func (s *logTrackingSubscriber) teardown() {
	s.teardownOnce.Do(func() {
		// Need to hold the mutex to prevent races between closing the channel
		// and sending the log lines to the subscriber.
		s.mutex.Lock()
		defer s.mutex.Unlock()
		// Cancel the subscriber context. It will ensure that the lines
		// will no longer be consumed/sent to the subscriber.
		s.cancelFn()
		// Release the semaphore in case the subscriber is waiting for
		// the log to be backfilled.
		s.signalReady()
		// Close the subscriber channel as we no longer send the log
		// to this subscriber.
		close(s.dataChan)
		// Signal to the stop function that it can now return.
		close(s.stoppedChan)
	})
}

// Stops the subscriber and waits for it to be stopped.
func (s *logTrackingSubscriber) stop() {
	s.cancelFn()
	<-s.stoppedChan
}

// Safely checks if the subscriber is stopped. It is used to check if the
// subscriber can be detached from the log capture.
func (s *logTrackingSubscriber) isStopped() bool {
	select {
	case <-s.stoppedChan:
		return true
	default:
		return false
	}
}

// Signals that the subscriber is ready to receive the log lines from the capture. It is
// called to signal that the subscriber backfilled the logs and is ready to follow the
// new logs. Until the subscriber is ready, the capture will hold from sending the
// log lines to the subscribers. This function is guaranteed to be executed only
// once. Subsequent calls are ignored.
func (s *logTrackingSubscriber) signalReady() {
	s.signalReadyOnce.Do(func() {
		close(s.readyChan)
	})
}

// Represents a log capture (with following) from a single log source (e.g.,
// a file or a systemd unit). Multiple subscribers can be attached to the
// capture. The capture is responsible for reading the log (using the dedicated
// reader), and sending the log lines to the subscribers.
type logTrackingCapture struct {
	// The function used to cancel the capture.
	cancelFn context.CancelFunc
	// The context used to cancel the capture.
	ctx context.Context
	// The mutex to protect the capture state from concurrent access.
	mutex sync.Mutex
	// The function used to clean up the capture when all subscribers are cancelled.
	emptyFn func()
	// The reader used to read the log contents from the log source.
	reader logReader
	// The guard to ensure that the start function is called only once.
	startOnce sync.Once
	// The error returned by the start function.
	startErr error
	// The subscribers attached to the capture.
	subscribers []*logTrackingSubscriber
}

// Instantiates a new log capture. The reader is used to read the log contents
// from the log source. The emptyFn callback is called when all subscribers are
// cancelled and the capture is no longer needed. The caller can use the emptyFn
// callback to garbage collect the capture.
func newLogTrackingCapture(ctx context.Context, reader logReader, emptyFn func()) *logTrackingCapture {
	ctx, cancel := context.WithCancel(ctx)
	return &logTrackingCapture{
		cancelFn: cancel,
		ctx:      ctx,
		emptyFn:  emptyFn,
		reader:   reader,
	}
}

// Starts the capture if it has not been already started.
func (c *logTrackingCapture) ensureStarted(options ...logReaderCaptureOption) error {
	c.startOnce.Do(func() {
		c.startErr = c.start(options...)
	})
	return c.startErr
}

// Stops all subscribers and removes the capture if it is no longer needed.
func (c *logTrackingCapture) stop() {
	c.mutex.Lock()
	for _, subscriber := range c.subscribers {
		subscriber.teardown()
	}
	empty := c.removeStoppedSubscribers()
	if empty {
		c.cancelFn()
	}
	c.mutex.Unlock()
	if empty && c.emptyFn != nil {
		// Remove the capture if it no longer has any subscribers.
		// It is important to call this function outside of the capture
		// lock to avoid deadlocks caused by locking order in logTrackingCapture
		// and logTracker.
		c.emptyFn()
	}
}

// Starts the new capture using the underlying reader. The capture runs in
// a goroutine that reads the log lines from the reader and sends them to the
// subscribers.
func (c *logTrackingCapture) start(options ...logReaderCaptureOption) error {
	ch, err := c.reader.capture(c.ctx, options...)
	if err != nil {
		return err
	}
	go func() {
		for {
			var (
				line logReaderLine
				ok   bool
				stop bool
			)
			select {
			case <-c.ctx.Done():
				// Stop the capture if the context is cancelled.
				stop = true
			case line, ok = <-ch:
				stop = !ok
			}
			if stop {
				// Stop the capture if the context is cancelled or the reader
				// channel is closed.
				c.stop()
				return
			}
			// Get the snapshot of the subscribers to avoid holding the lock
			// while sending the data to the subscribers.
			c.mutex.Lock()
			subscribers := append([]*logTrackingSubscriber{}, c.subscribers...)
			c.mutex.Unlock()

			var hadStoppedSubscribers bool
			// Send the log line to the subscribers.
			for _, subscriber := range subscribers {
				select {
				case <-subscriber.readyChan:
					// If this is a new subscriber, it may be backfilling the log.
					// We should not send any new log lines until the subscriber is ready
					// to receive. Otherwise, the subscribers will get out of sync.
					// Block here until the subscriber is ready.
				case <-c.ctx.Done():
					// Stop the capture if the context is cancelled.
					c.stop()
					return
				}
				if !subscriber.consumeLogLine(line) {
					// Something went wrong with sending the log line to the subscriber.
					// Possibly, the subscriber was cancelled.
					subscriber.teardown()
					hadStoppedSubscribers = true
				}
			}
			if hadStoppedSubscribers {
				// Remove all stopped subscribers.
				c.mutex.Lock()
				empty := c.removeStoppedSubscribers()
				if empty {
					// There are no more subscribers. Let's stop the capture.
					c.cancelFn()
				}
				c.mutex.Unlock()
				if empty {
					if c.emptyFn != nil {
						// Remove the capture if it no longer has any subscribers.
						// It is important to call this function outside of the capture
						// lock to avoid deadlocks caused by locking order in logTrackingCapture
						// and logTracker.
						c.emptyFn()
					}
					return
				}
			}
		}
	}()
	return nil
}

// Attaches new subscriber to the capture. It then starts a goroutine that waits
// for the subscriber context cancellation.
func (c *logTrackingCapture) addSubscriber(subscriber *logTrackingSubscriber) {
	c.mutex.Lock()
	c.subscribers = append(c.subscribers, subscriber)
	c.mutex.Unlock()
	go func() {
		// Wait here for the context cancellation that occurs when the subscriber
		// is detached from the capture.
		<-subscriber.ctx.Done()
		c.mutex.Lock()
		subscriber.teardown()
		// Make sure that the stopped subscriber is removed.
		empty := c.removeStoppedSubscribers()
		if empty {
			c.cancelFn()
		}
		c.mutex.Unlock()
		if empty && c.emptyFn != nil {
			// Remove the capture if it no longer has any subscribers.
			// It is important to call this function outside of the capture
			// lock to avoid deadlocks caused by locking order in logTrackingCapture
			// and logTracker.
			c.emptyFn()
		}
	}()
}

// Removes stopped subscribers from the capture. It returns boolean value indicating
// whether the capture has no more subscribers.
func (c *logTrackingCapture) removeStoppedSubscribers() bool {
	c.subscribers = slices.DeleteFunc(c.subscribers, func(subscriber *logTrackingSubscriber) bool {
		// Remove the subscriber if it is stopped.
		return subscriber.isStopped()
	})
	return len(c.subscribers) == 0
}

// A key for identifying log captures that follow the same log source.
type logTrackerCaptureKey struct {
	fileName string
	unitName string
}

// Configuration for the log tracker.
type logTrackerConfig struct {
	channelSize         int
	textLogReaderConfig textLogReaderConfig
}

// Implements the log tracker. It is a central point for log tracking operations.
// It allows for subscribing to the log streams. It supports multiple subscrptions
// per single log stream capture. For example, if multiple subscribers want to follow
// the systemd logs, this instance ensures that there is one systemd instance running
// and it sends the output to all subscribers. The only exception is when one subscriber
// is already following the tail of the log, and another subscriber requests reading the
// earlier logs, and then following the tail. In this case, the log tracker will open
// another reader to read the earlier logs, and then fall back to a single reader.
type logTracker struct {
	config   logTrackerConfig
	ctx      context.Context
	cancel   context.CancelFunc
	captures map[logTrackerCaptureKey]*logTrackingCapture
	executor storkutil.CommandExecutor
	stopped  bool
	stopOnce sync.Once
	mutex    sync.RWMutex
}

// Creates a new log tracker. It requires the executor instance that may be required
// by the systemd log readers. The textLogReaderConfig is used to configure the text
// file log readers (e.g., enable polling in the tests).
func newLogTracker(executor storkutil.CommandExecutor, config logTrackerConfig) *logTracker {
	ctx, cancel := context.WithCancel(context.Background())
	return &logTracker{
		ctx:      ctx,
		cancel:   cancel,
		captures: make(map[logTrackerCaptureKey]*logTrackingCapture),
		executor: executor,
		config:   config,
	}
}

// Instantiates a new log reader depending on whether the caller specified a file
// path or not. In the former case, the textFileReader is instantiated. In the latter
// case, the systemdLogReader is instantiated. If the systemdLogReader is not supported,
// an error is returned.
func (l *logTracker) createLogReader(filePath string) (logReader, error) {
	systemdLogReader := newSystemdLogReader(l.executor)
	switch {
	case filePath != "":
		return newTextFileLogReader(l.config.textLogReaderConfig), nil
	case systemdLogReader.isSupported():
		return systemdLogReader, nil
	default:
		return nil, errors.New("no supported log tracking method available")
	}
}

// Reads the initial part of the log according to the specified options, and returns
// the log lines via the returned channel. The subscriber must be instantiated by the
// caller and the onComplete callback typically includes a call on the subscriber to
// either cancel the subscription or mark it ready if the read function is used as
// a backfill function before tailing.
func (l *logTracker) read(subscriber *logTrackingSubscriber, onComplete func(), options ...logReaderCaptureOption) (*logTrackingSubscriber, error) {
	// Map specified options to the configuration structure.
	config := logReaderCaptureConfig{}
	for _, option := range options {
		option(&config)
	}
	// Create a new reader using specified options.
	reader, err := l.createLogReader(config.fileName)
	if err != nil {
		// In case of an error, we need to cancel the subscriber, so it is not left hanging.
		if onComplete != nil {
			onComplete()
		}
		return nil, err
	}
	ch, err := reader.capture(subscriber.ctx, slices.DeleteFunc(options, func(option logReaderCaptureOption) bool {
		var config logReaderCaptureConfig
		option(&config)
		// While backfilling, we need to ensure that we don't follow the log afterwards. Preserve
		// all options except following and reading from the end.
		return config.fromEnd || config.follow
	})...)
	if err != nil {
		return nil, err
	}
	// Log reader successfully created. Let's start backfilling the log.
	go func() {
		// Make sure we cancel the subscriber or mark it as ready to tail the log
		// after backfilling is complete.
		defer onComplete()
		for {
			select {
			case <-subscriber.ctx.Done():
				// Stop reading when the caller cancelled the context.
				return
			case line, ok := <-ch:
				// Send the log line to the subscriber or stop if the caller cancelled the context
				// in the meantime.
				if !ok || !subscriber.consumeLogLine(line) {
					return
				}
			}
		}
	}()
	return subscriber, nil
}

// Deletes a capture from the log tracker by key.
func (l *logTracker) removeCapture(captureKey logTrackerCaptureKey) {
	delete(l.captures, captureKey)
}

// Attaches new subscriber to a log tracking capture or reads the requested part of the log.
// This function returns a channel to be used by the caller to receive the log lines.
// The caller can request to follow or not follow the log. Regardless, the data is returned
// via the channel. If the caller requested to follow the log, the function will try to find
// an existing capture in case any other caller already created one. In that case, the new
// subscription is attached to the capture to prevent opening multiple readers. If the
// capture is not found, a new one is created. If two subscriptions are attached to the
// same capture, it is likely that the first subscriber is already reading the log from the
// tail, and the new subscriber wants to read the earlier logs before following the tail.
// In this case, the function will first read the earlier logs and return to the new
// subscriber. Once the subscriber has read the logs, it is attached to capture the tail
// together with other subscribers. It is not valid to read the log from the tail without
// following. This function must not be called after the log tracker is stopped.
func (l *logTracker) subscribe(options ...logReaderCaptureOption) (*logTrackingSubscriber, error) {
	// Map specified options to the configuration structure.
	config := logReaderCaptureConfig{}
	for _, option := range options {
		option(&config)
	}
	// Create a new subscriber. It holds a channel to receive the log lines.
	subscriber := newLogTrackingSubscriber(l.ctx, l.config.channelSize)

	if !config.follow {
		// If the user didn't request to follow the log, this is not real subscription.
		// We will just read the requested part of the log and return it via the
		// subscriber's channel. We wan't be creating or joining any live capture.
		return l.read(subscriber, subscriber.teardown, options...)
	}

	// The caller requested to follow the log. Let's create a candidate reader for
	// a capture. Creating the reader is cheap.
	candidateReader, err := l.createLogReader(config.fileName)
	if err != nil {
		return nil, err
	}
	// Create a key to identify the capture.
	captureKey := logTrackerCaptureKey{
		fileName: config.fileName,
		unitName: config.unitName,
	}
	// Create a candidate log capture using the reader. The second argument is
	// a callback function that is called when all subscribers are cancelled.
	// It is meant to garbage collect stale captures.
	candidateCapture := newLogTrackingCapture(l.ctx, candidateReader, func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		l.removeCapture(captureKey)
	})
	// Try to use the capture. If the capture with the same configuration already
	// exists, it is returned and loaded is set to true. Otherwise, the candidate
	// capture is stored and loaded is set to false.
	var (
		capture *logTrackingCapture
		loaded  bool
	)
	// Hold the mutex through the rest of the function to prevent stopping the
	// log tracker in the middle of creating the subscription.
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.stopped {
		// Prevent new subscriptions on the stopped log tracker.
		return nil, errors.New("log tracker was stopped")
	}
	capture, loaded = l.captures[captureKey]
	if !loaded {
		capture = candidateCapture
		l.captures[captureKey] = capture
	}

	// It doesn't matter whether it is a new capture or an existing one. Let's attach
	// the subscriber to it. The subscriber is going to receive the log lines from
	// the capture.
	capture.addSubscriber(subscriber)

	if !loaded || config.fromEnd {
		// If this is new capture or the caller wants to attach to the tail of the
		// capture, there is no need to backfill the missing log lines. Let's mark the
		// subscriber as ready, so the capture can continue feeding the subscribers'
		// channels with the log lines.
		subscriber.signalReady()
	} else if _, err := l.read(subscriber, subscriber.signalReady, options...); err != nil {
		// The capture was already started by another subscriber. Since we're not reading
		// from the tail, but rather want to read historical logs, we need to open a
		// separate backfill reader to get the missing log lines. If it fails we tear down
		// the subscriber but we leave the capture, as there must be other subscribers
		// attached to it.
		subscriber.teardown()
		return nil, err
	}
	// Ensure that the capture is started. This will only be fired if this is a new
	// capture with the first subscriber.
	err = capture.ensureStarted(options...)
	if err != nil {
		// Make sure that the subscriber is removed and the capture is removed.
		// Note that an error can only occur if we are starting the capture, so
		// this is the first subscriber. In that case the capture can be removed
		// without affecting any subscribers.
		subscriber.teardown()
		l.removeCapture(captureKey)
		return nil, err
	}
	return subscriber, nil
}

// Stops the log tracker and waits for all captures to be stopped.
func (l *logTracker) stop() {
	l.stopOnce.Do(func() {
		l.mutex.Lock()
		// Mark the log tracker as stopped, so no new subscriptions can be
		// created from now on.
		l.stopped = true
		captures := slices.Collect(maps.Values(l.captures))
		l.mutex.Unlock()
		// Release the mutex to ensure that the callback function cleaning
		// up the captures after stopping them is not blocked.
		for _, capture := range captures {
			capture.stop()
		}
	})
}

// Returns true if the log tracker is still busy. It is busy if it is not stopped
// and there are still captures running.
func (l *logTracker) isBusy() bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return !l.stopped && len(l.captures) > 0
}

// Returns true if the log tracker is stopped. The tracker cannot be resumed
// once it is stopped.
func (l *logTracker) isStopped() bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.stopped
}
