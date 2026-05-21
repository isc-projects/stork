package agent

import (
	"container/list"
	"context"
	"iter"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	storkutil "isc.org/stork/util"
)

const (
	// The default number of days to look into the past for the XFR tracking
	// when the tracker starts analyzing the systemd logs.
	defaultXfrTrackingSinceDaysAgo = 1
	// The default maximum number of zone transfers to track.
	defaultXfrTrackingMaxStates = 1000
)

// The zone transfer status type.
type xfrStatus int

const (
	// This is a default status indicating that the parsed log message was unrecognized
	// and could not be used to determine the actual zone transfer status. Messages with
	// this status are discarded.
	xfrStatusUnknown xfrStatus = iota
	// The zone transfer has started.
	xfrStatusStarted
	// The incoming zone transfer has started on the secondary server, and the last
	// log message indicated that the secondary server successfully connected to the
	// primary server.
	xfrStatusConnected
	// The zone transfer has completed.
	xfrStatusCompleted
	// The last received log message neither marks the beginning nor the end of the zone
	// transfer. It is typically a message received during the zone transfer indicating
	// some kind of problem.
	xfrStatusMessage
)

// A time format used in the parsed log messages.
type xfrTimeFormat int

const (
	// The time format is in the RFC3339 format (e.g., 2026-02-23T10:41:27.071Z).
	xfrTimeFormatRFC3339 xfrTimeFormat = iota
	// The time format is in the BIND 9 format (e.g., 23-Feb-2026 10:41:27.071).
	xfrTimeFormatBind9
	// The time format is unknown/unrecognized.
	xfrTimeFormatUnknown
)

// Zone transfer tracker uses the underlying log trackers to subscribe to the
// logs containing messages marking the beginning and end of the zone transfers
// initiated by BIND.
//
// Note that the log tracker is a common component owned by the monitor. It can manage
// many subscriptions to various logs. The XFR tracker is associated with a BIND 9
// daemon instance and it establishes a single subscription over the log tracker.
//
// The tracker uses the LRU (least recently used) cache to track the started zone
// transfers. The cache limits the number of tracked zone transfers to a default
// value. The most recently accessed zone transfer entry is moved to the end of the
// list. The oldest entries are removed if the length of the list exceeds the maximum
// allowed number. The LRU cache includes a slice (to order the zone transfers by from
// least recently accessed) and a map (for fast lookup by the zone transfer state key).
// Completed transfers are moved to a separate list.
//
// In order to start tracking the zone transfers, run the trackFiles() or trackSystemdUnit()
// functions, depending on the log locations. In order to stop tracking the zone transfers,
// run the stop() function. Additional calls to the trackFiles() or trackSystemdUnit()
// will restart the tracking. Note that calling these functions is not concurrent safe.
// The tracker instance belongs to the BIND 9 daemon and should be started after the daemon
// is detected. It should ensure that the calls to start/stop tracking are serialized.
//
// Getting the ongoing and completed zone transfers is safe for concurrent use.
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
	cancelCh       chan struct{}
	followChan     chan xfrState
	followCtx      context.Context
	followCancelFn context.CancelFunc
	// The list of started zone transfers.
	startedList list.List
	// The map of started zone transfers indexed by the state key.
	startedMap map[xfrStateKey]*list.Element
	// The list of completed zone transfers.
	completedList list.List
	// The maximum number of zone transfers to track.
	maxStates int
	// The mutex to protect the tracker state from concurrent access.
	mutex sync.RWMutex
}

// The zone transfer state. An instance of this structure is returned for
// each parsed log message pertaining to a zone transfer. The state includes
// the data extracted from the log message such as the zone name, view name,
// client address (for outgoing zone transfers) and server address (for incoming
// zone transfers). It also includes suitable timestamps and zone transfer
// statistics.
type xfrState struct {
	viewName       string
	zoneName       string
	serial         int64
	client         string
	server         string
	messagesCount  int64
	recordsCount   int64
	bytesCount     int64
	duration       time.Duration
	status         xfrStatus
	startTime      time.Time
	completionTime time.Time
	timeFormat     xfrTimeFormat
	message        string
}

// The key used to index the started zone transfers in the LRU cache.
// The client is optional - it is empty for incoming zone transfers.
type xfrStateKey struct {
	viewName string
	zoneName string
	client   string
}

// Instantiates a new XFR tracker instance. It is associated with the log
// tracker instance specified as an argument. The log tracker instance
// must be non-nil.
func newXfrTracker(logTracker *logTracker) *xfrTracker {
	return &xfrTracker{
		logTracker: logTracker,
		startedMap: make(map[xfrStateKey]*list.Element),
		maxStates:  defaultXfrTrackingMaxStates,
	}
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
	t.track()
	log.WithFields(log.Fields{
		"filenames": strings.Join(filenames, ", "),
	}).Info("DNS zone transfer tracking successfully started using log file(s)")
	return nil
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
	t.track()
	log.WithFields(log.Fields{
		"unit": unitName,
	}).Info("DNS zone transfer tracking successfully started using systemd unit")
	return nil
}

// Stops the XFR tracker. It stops the subscriptions, cancels the goroutine that
// consumes the log lines, and releases the resources.
func (t *xfrTracker) stop() {
	// Stop following the zone transfers.
	t.cancelFollow()
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

// Tracks the log file or systemd logs using created subscriptions.
func (t *xfrTracker) track() {
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
			case line, ok := <-chan0:
				if !ok {
					return
				}
				t.feed(line.text)
			case line, ok := <-chan1:
				if !ok {
					return
				}
				t.feed(line.text)
			}
		}
	}()
	// Remember the cancellation function and the channel, so we can use them to
	// stop the subscriptions in the stop() function.
	t.cancelFn = cancel
	t.cancelCh = cancelCh
}

// Feeds the log line to the XFR tracker. It parses the log line and updates
// the zone transfer state. It is safe for concurrent use.
func (t *xfrTracker) feed(logLine string) {
	newState := parseTransferLogLine(logLine)
	if newState == nil {
		// The log line was not related to a zone transfer or we did not
		// consider it useful.
		return
	}
	key := xfrStateKey{
		viewName: newState.viewName,
		zoneName: newState.zoneName,
		client:   newState.client,
	}

	t.mutex.Lock()

	// Check if the zone transfer state already exists.
	var currState *xfrState
	element, ok := t.startedMap[key]
	if ok && element != nil {
		st, ok := element.Value.(*xfrState)
		if !ok {
			t.mutex.Unlock()
			return
		}
		currState = st
	}

	switch newState.status {
	case xfrStatusStarted, xfrStatusMessage, xfrStatusConnected:
		if currState != nil {
			// It exists, so let's replace it with a new one.
			element.Value = newState
			// Move the element to the back of the list to keep the most recent one
			// at the end.
			t.startedList.MoveToBack(element)
			if !currState.startTime.IsZero() && newState.status != xfrStatusStarted {
				// Preserve the original start time.
				newState.startTime = currState.startTime
			}
			break
		}
		// The state does not exist. Let's add it to the container.
		element := t.startedList.PushBack(newState)
		t.startedMap[key] = element

		// Ensure that the number of started transfers does not exceed the maximum
		// allowed number. If it does, remove the oldest one.
		if t.startedList.Len() > t.maxStates {
			// Remove the oldest one from the front of the list.
			if element := t.startedList.Remove(t.startedList.Front()); element != nil {
				if state, ok := element.(*xfrState); ok {
					key := xfrStateKey{
						viewName: state.viewName,
						zoneName: state.zoneName,
						client:   state.client,
					}
					// Remove the element from the map.
					delete(t.startedMap, key)
				}
			}
		}
	default:
		if currState != nil {
			// It exists, so let's remove it from the started container.
			t.startedList.Remove(element)
			delete(t.startedMap, key)
			newState.startTime = currState.startTime
		}
		// Add the state to the completed container.
		t.completedList.PushBack(newState)
		// Ensure that the number of completed transfers does not exceed the maximum
		// allowed number. If it does, remove the oldest one.
		if t.completedList.Len() > t.maxStates {
			t.completedList.Remove(t.completedList.Front())
		}
	}
	t.mutex.Unlock()

	t.mutex.Lock()
	if t.followChan != nil && t.followCtx != nil {
		select {
		case t.followChan <- *newState:
		case <-t.followCtx.Done():
		}
	}
	t.mutex.Unlock()
}

func (t *xfrTracker) getNotCompletedUnsafe() []xfrState {
	states := make([]xfrState, 0, t.startedList.Len())
	for element := t.startedList.Front(); element != nil; element = element.Next() {
		states = append(states, *element.Value.(*xfrState))
	}
	return states
}

// Returns the list of ongoing or stuck zone transfers. It is safe for concurrent use.
func (t *xfrTracker) getNotCompleted() []xfrState {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.getNotCompletedUnsafe()
}

func (t *xfrTracker) getCompletedUnsafe() []xfrState {
	states := make([]xfrState, 0, t.completedList.Len())
	for element := t.completedList.Front(); element != nil; element = element.Next() {
		states = append(states, *element.Value.(*xfrState))
	}
	return states
}

// Returns the list of completed zone transfers. It is safe for concurrent use.
func (t *xfrTracker) getCompleted() []xfrState {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.getCompletedUnsafe()
}

// Returns the collection of completed and ongoing zone transfers as well as the
// channel to receive live updates about future zone transfers. The output from this
// function is meant to be consumed by the gRPC API handler returning the stream of
// zone transfers to the server. There may be only one caller receiving the updates
// over the channel. If the function is called again while the updates are being consumed,
// the channel is closed and the new channel is returned. In order to stop receiving the
// updates, the caller should cancel the context passed as an argument.
func (t *xfrTracker) follow(ctx context.Context) (completed iter.Seq[xfrState], ongoing iter.Seq[xfrState], followChan <-chan xfrState) {
	t.mutex.Lock()
	if t.followChan != nil {
		// We are already following the zone transfers. Let's cancel the
		// existing session, so it can be restarted.
		t.cancelFollowUnsafe()
	}
	// Create a new channel to receive the live updates about future zone transfers.
	ch := make(chan xfrState)
	t.followChan = ch
	followChan = ch

	// Wrap the context with another cancel context, so we can stop the goroutine
	// in cancelFollowUnsafe(), if needed.
	t.followCtx, t.followCancelFn = context.WithCancel(ctx)
	followCtx := t.followCtx

	// Get the list of ongoing and completed zone transfers before the lock is released.
	ongoing = slices.Values(t.getNotCompletedUnsafe())
	completed = slices.Values(t.getCompletedUnsafe())
	t.mutex.Unlock()

	// Start the goutine to cancel the session if the context is cancelled.
	go func() {
		<-followCtx.Done()
		t.mutex.Lock()
		defer t.mutex.Unlock()
		if t.followChan == followChan {
			// We're here when the caller cancelled the context (e.g., gRPC stream is closed).
			// Let's close the channel and release the resources.
			t.cancelFollowUnsafe()
		}
	}()

	return
}

// Cancels the zone transfer following session. It is not safe for concurrent use
// and should be called under the mutex.
func (t *xfrTracker) cancelFollowUnsafe() {
	if t.followCancelFn != nil {
		// Make sure that the child context is cancelled.
		t.followCancelFn()
	}
	if t.followChan != nil {
		// Close the channel to signal the caller that the session is cancelled.
		close(t.followChan)
	}
	// Release the resources.
	t.followChan = nil
	t.followCtx = nil
	t.followCancelFn = nil
}

// Cancels the zone transfer following session. It is safe for concurrent use.
func (t *xfrTracker) cancelFollow() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.cancelFollowUnsafe()
}

// Parses the time at the position of the iterator. It recognizes both RFC3339 and
// BIND 9 time formats. In case of a different time format, it returns the
// xfrTimeFormatUnknown.
func parseTime(iterator *storkutil.PeekingIterator[string]) (parsedTime time.Time, timeFormat xfrTimeFormat) {
	timeFormat = xfrTimeFormatUnknown
	firstToken, _ := iterator.Peek()
	if firstToken == "" {
		// If there are no more tokens, there is nothing to do.
		return
	}
	var err error
	if strings.Contains(firstToken, "T") {
		// The data in the RFC3339 format is typically in the form of
		// 2026-02-23T10:41:27.071Z. It contains the letter T to separate
		// the date and the time.
		parsedTime, err = time.Parse(time.RFC3339, firstToken)
		if err == nil {
			// If parsing was successful, set the time format and consume the token.
			timeFormat = xfrTimeFormatRFC3339
			iterator.Next()
		}
		// No point in trying to parse the time in the BIND 9 format.
		return
	}
	// Before we consume the first token, let's see if it is even recognized
	// as a date. Otherwise, we don't want to consume the token because it
	// may belong to some other statement we want to parse (e.g., zone name).
	// Note that some logs may lack timestamps.
	if _, err = time.Parse("02-Jan-2006", firstToken); err != nil {
		return
	}
	// It is apparently a date in the BIND 9 format. Let's consume the
	// first token.
	iterator.Next()
	secondToken, _ := iterator.Peek()
	if secondToken == "" {
		// No more tokens.
		return
	}
	// The BIND 9 format is in the form of "23-Feb-2026 10:41:27.071". Let's
	// re-create it from two consecutive tokens.
	localTime := firstToken + " " + secondToken
	parsedTime, err = time.ParseInLocation("02-Jan-2006 15:04:05.000", localTime, time.Local)
	if err == nil {
		// Parsing was successful. Set the time format and consume the second token.
		timeFormat = xfrTimeFormatBind9
		iterator.Next()
	}
	return
}

// Sanitizes the token so it can be used in switch statements. It removes
// any parentheses, commas, semicolons and single quotes. Note that all of
// these extra characters may appear in the log messages. For example, zone
// names are often enclosed in quotes. The serial number is often enclosed in
// parentheses. The message blocks are terminated with a semicolon.
func sanitizeToken(token string) string {
	return strings.TrimFunc(token, func(r rune) bool {
		return strings.ContainsRune("():',", r)
	})
}

// Parses the single log line and returns the corresponding state.
func parseTransferLogLine(logLine string) *xfrState {
	// Limit the length of the log line to 1024 characters to avoid
	// parsing excessively long log lines.
	runes := []rune(logLine)
	if len(runes) > 1024 {
		runes = runes[:1024]
		logLine = string(runes)
	}

	// Remove any trailing dot from the log line. Some BIND 9 logs contain
	// trailing dots, other don't.
	logLine = strings.TrimRight(logLine, ".")

	// Tokenize the log line and ensure it has reasonable length.
	tokens := strings.Fields(logLine)
	if len(tokens) > 50 {
		tokens = tokens[:50]
	}

	// We're only interested in the log lines containing the "transfer" keyword.
	// All other log lines are ignored. It is possible that we get some unrelated
	// ones but they will be filtered out once we parse them.
	if !slices.ContainsFunc(tokens, func(token string) bool {
		return strings.ToLower(token) == "transfer"
	}) {
		return nil
	}

	// Create a peeking iterator over the tokens.
	iterator := storkutil.NewPeekingIterator(tokens)

	var (
		parsedTime time.Time
		state      xfrState
	)

	// The log line typically begins with a timestamp. Parse it and determine
	// the time format used by BIND 9.
	parsedTime, state.timeFormat = parseTime(iterator)

	// Parse the tokens starting after the timestamp.
	for {
		// Get the next token.
		currToken, ok := iterator.Next()
		if !ok {
			// No more tokens to process.
			break
		}
		// Tokens are parsed case-insensitively.
		token := strings.ToLower(sanitizeToken(currToken))
		switch token {
		case "zone":
			// The secondary server logs the beginning of the zone transfer with a short
			// message beginning with the "zone" keyword (i.e., zone <name>: Transfer started.).
			// Parse the zone name and record the rest of the message.
			if parseZone(iterator, &state) {
				state.message = strings.Join(iterator.PeekSubsequent(), " ")
			}
		case "view":
			// View name is sometimes logged as view <name>. More often it is logged within
			// the zone name string, but sometimes it is logged separately.
			parseView(iterator, &state)
		case "transfer":
			// In most cases, the log message contains the "transfer of". It may also contain
			// "Transfer started". This call handles both cases.
			parseTransfer(iterator, parsedTime, &state)
		case "connected":
			// The secondary server logs a useful message when it connects to the primary server
			// including the server address. Note that the server address is not logged when the
			// zone transfer is started. Let's capture the "connected using <address>" messages
			// to set the server address for a started zone transfer.
			parseConnectedUsing(iterator, &state)
		case "axfr", "ixfr":
			// Parse the "AXFR started" and "IXFR started" messages to capture when a zone transfer
			// has started. Mark the zone transfer completed when we come across the "AXFR ended"
			// or "IXFR ended" messages. This call handles all these cases.
			parseXFR(iterator, parsedTime, &state)
		case "serial":
			// Zone serial is logged in many messages. It is very important for zone transfer
			// monitoring.
			if err := parseSerial(iterator, &state); err != nil {
				log.WithError(err).Debugf("Failed to parse serial %s", token)
			}
		case "client":
			// The client address is logged on the primary server when the secondary connects.
			parseClient(iterator, &state)
		case "messages", "records", "bytes":
			// Parse zone transfer statistics. The actual value should have been logged before
			// the current token.
			prevToken, ok := iterator.PeekBack()
			if !ok {
				log.WithField("token", currToken).Debug("Previous token not found")
			}
			if err := parseMessagesRecordsBytes(prevToken, token, &state); err != nil {
				log.WithError(err).Debugf("Failed to parse the number of %s", currToken)
			}
		case "secs":
			// Parse the zone transfer duration. The actual value should have been logged before
			// the current token.
			prevToken, ok := iterator.PeekBack()
			if !ok {
				log.WithField("token", currToken).Debug("Previous token not found")
			}
			if err := parseSecs(prevToken, &state); err != nil {
				log.WithError(err).Debugf("Failed to parse the duration %s", prevToken)
			}
		}
	}
	if state.zoneName == "" || state.status == xfrStatusUnknown || (state.status == xfrStatusConnected && state.server == "") {
		// Zone name and status are required. If they are not set, the log message is
		// malformed or not related to a zone transfer.
		return nil
	}
	if state.recordsCount > 0 {
		// The presence of these counters indicates that the zone transfer has completed.
		state.status = xfrStatusCompleted
		state.completionTime = parsedTime
	}
	return &state
}

// Parses the "transfer of" and "Transfer started" statements.
func parseTransfer(iterator *storkutil.PeekingIterator[string], parsedTime time.Time, s *xfrState) {
	token, _ := iterator.Peek()
	token = strings.ToLower(sanitizeToken(token))
	switch token {
	case "of":
		// Consume the token and parse the "transfer of" statement.
		iterator.Next()
		parseTransferOf(iterator, s)
	case "started":
		// Consume the token. We came across the "Transfer started" statement.
		// Let's mark the zone transfer started and record the start time.
		iterator.Next()
		s.status = xfrStatusStarted
		s.startTime = parsedTime
	default:
		// Unknown statement. The log message simply contains the "transfer" keyword
		// in some unrecognized context.
	}
}

// Parses the "transfer of" statement contents.
func parseTransferOf(iterator *storkutil.PeekingIterator[string], s *xfrState) {
	// The "transfer of" is followed by the zone name. Parse it.
	if !parseZone(iterator, s) {
		// Parsing the zone name failed. Perhaps the "transfer of" was used in
		// some other context.
		return
	}
	// Get next token because the zone name is optionally followed by the server
	// address. The server address is preceded by the "from" keyword.
	token, _ := iterator.Peek()
	if token == "from" {
		// The server address follows. Let's consume the "from" token.
		iterator.Next()
		// Parse the server address.
		parseFrom(iterator, s)
	}
	// The "transfer of" statement can be followed by different messages indicating
	// the status of the zone transfer (including errors). Some of these messages may
	// mark the beginning or end of the zone transfer but we don't know what exactly.
	// This will be known later as we process the following tokens. For now, let's just
	// assume it is a general message (hence set the xfrStatusMessage).
	s.message = strings.Join(iterator.PeekSubsequent(), " ")
	s.status = xfrStatusMessage
}

// Parses the "connected using" statement containing the server address.
func parseConnectedUsing(iterator *storkutil.PeekingIterator[string], s *xfrState) {
	token, _ := iterator.Next()
	if token != "using" {
		// It is not the "connected using" statement.
		return
	}
	// The "connected using" statement is followed by the server address in the
	// same format as the "from <address>" statement
	if parseFrom(iterator, s) {
		s.status = xfrStatusConnected
	}
}

// Parses the "AXFR/IXFR started/ended" statements.
func parseXFR(iterator *storkutil.PeekingIterator[string], parsedTime time.Time, s *xfrState) {
	token, ok := iterator.Next()
	if !ok {
		// No more tokens to process.
		return
	}
	token = sanitizeToken(token)
	switch token {
	case "started":
		// AXFR or IXFR started.
		s.status = xfrStatusStarted
		s.startTime = parsedTime
	case "completed", "ended":
		// AXFR or IXFR ended.
		s.status = xfrStatusCompleted
		s.completionTime = parsedTime
	}
}

// Parses the zone name and view name from the log message.
func parseZone(iterator *storkutil.PeekingIterator[string], s *xfrState) bool {
	var (
		zoneName string
		viewName string
	)
	if token, ok := iterator.Next(); ok {
		token = sanitizeToken(token)
		// The zone name is followed by the class name IN, and may be followed
		// by the view name. For example, "example.com/IN" or "example.com/IN/trusted".
		split := strings.Split(token, "/")
		if len(split) > 1 {
			// The first part is always the zone name.
			zoneName = split[0]
		}
		if len(split) > 2 {
			// The third part is the view name.
			viewName = split[2]
		}
		if viewName != "" && s.viewName == "" {
			// Only override the view name if it is not already set.
			s.viewName = viewName
		}
		if zoneName != "" && s.zoneName == "" {
			// Only override the zone name if it is not already set.
			s.zoneName = zoneName
		}
	}
	// Indicate whether or not we have been successful. At least there must be
	// a zone name parsed.
	return zoneName != ""
}

// Parses the view name from the log message.
func parseView(iterator *storkutil.PeekingIterator[string], s *xfrState) {
	if token, ok := iterator.Next(); ok {
		if s.viewName == "" {
			// Only override the view name if it is not already set.
			s.viewName = strings.Trim(token, ":")
		}
	}
}

// Parses the server address following the "from" keyword.
func parseFrom(iterator *storkutil.PeekingIterator[string], s *xfrState) bool {
	var server string
	if token, ok := iterator.Next(); ok {
		// The server address is followed by the port number (e.g., 192.5.5.241#53).
		server, _, _ = strings.Cut(token, "#")
	}
	if server != "" && s.server == "" {
		// Only override the server address if it is not already set.
		s.server = server
	}
	// Indicate if parsing was successful.
	return server != ""
}

// Parses the client address from the log message.
func parseClient(iterator *storkutil.PeekingIterator[string], s *xfrState) {
	// The first token is not interesting. It holds the pointer similar to @0x7ffffaa28c00.
	iterator.Next()
	token, ok := iterator.Next()
	if s.client != "" || !ok {
		// If the client address is already set or there are no more tokens to process,
		// there is nothing to parse.
		return
	}
	// The second token is the client address.
	split, _, ok := strings.Cut(token, "#")
	if ok {
		// The first part is the client address.
		s.client = split
	}
}

// Parses the zone serial number.
func parseSerial(iterator *storkutil.PeekingIterator[string], s *xfrState) (err error) {
	token, ok := iterator.Next()
	if s.serial != 0 || !ok {
		// If the serial number is already set or there are no more tokens to process,
		// there is nothing to parse.
		return
	}
	// Make sure that the token has no junk like parentheses.
	token = sanitizeToken(token)
	if s.serial, err = strconv.ParseInt(token, 10, 64); err != nil {
		// It is not a valid number.
		err = errors.WithStack(err)
	}
	return
}

// Parses the number of messages, records or bytes. The value to parse is
// specified as a prevToken. The current token should be one of the following:
// "messages", "records", "bytes".
func parseMessagesRecordsBytes(prevToken, token string, s *xfrState) (err error) {
	if prevToken == "" {
		return
	}
	v, err := strconv.ParseInt(prevToken, 10, 64)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	switch token {
	case "messages":
		s.messagesCount = v
	case "records":
		s.recordsCount = v
	case "bytes":
		s.bytesCount = v
	default:
		return errors.Errorf("unexpected token: '%s'", token)
	}
	return
}

// Parses the duration as float number of seconds (e.g., 0.014 secs).
func parseSecs(secs string, s *xfrState) (err error) {
	if secs == "" {
		return
	}
	duration, err := strconv.ParseFloat(secs, 64)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	s.duration = time.Duration(duration * float64(time.Second))
	return
}
