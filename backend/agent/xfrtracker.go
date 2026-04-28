package agent

import (
	"container/list"
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	storkutil "isc.org/stork/util"
)

// The default number of days to look into the past for the XFR tracking
// when the tracker starts analyzing the systemd logs.
const (
	defaultXfrTrackingSinceDaysAgo = 1
	defaultXfrTrackingMaxStates    = 1000
)

type xfrStatus int

const (
	xfrStatusUnknown xfrStatus = iota
	xfrStatusStarted
	xfrStatusConnected
	xfrStatusCompleted
	xfrStatusMessage
)

type xfrTimeFormat int

const (
	xfrTimeFormatRFC3339 xfrTimeFormat = iota
	xfrTimeFormatBind9
	xfrTimeFormatUnknown
)

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
	cancelCh      chan struct{}
	startedList   list.List
	startedMap    map[xfrStateKey]*list.Element
	completedList list.List
	maxStates     int
	mutex         sync.RWMutex
}

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
	// Failed to parse.
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
func (t *xfrTracker) parse(logLine string) *xfrState {
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
	if state.zoneName == "" || state.status == xfrStatusUnknown {
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
		split := strings.Split(token, "#")
		if len(split) > 0 {
			// The first part is the server address.
			server = split[0]
		}
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
	split := strings.Split(token, "#")
	if len(split) > 0 {
		// The first part is the client address.
		s.client = split[0]
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

// Puts new state into the zone transfer containers. It is specifically
// called when the zone transfer has started. It creates a new state entry
// for the zone or it replaces an existing one. This function is safe for
// concurrent use.
func (t *xfrTracker) putState(key *xfrStateKey, s *xfrState) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	// Check if the given transfer already exists.
	element, ok := t.startedMap[*key]
	if ok {
		// It exists, so let's replace it with a new one.
		element.Value = s
		// Move the element to the back of the list to keep the most recent one
		// at the end.
		t.startedList.MoveToBack(element)
	} else {
		// It does not exist, so let's append it at the end. Also.
		// remember the element in the map for quick access using the key.
		element := t.startedList.PushBack(s)
		t.startedMap[*key] = element
	}
	// Ensure that the number of started transfers does not exceed the maximum
	// allowed number. If it does, remove the oldest one.
	if t.startedList.Len() > t.maxStates {
		// Remove the oldest one from the front of the list.
		element := t.startedList.Remove(t.startedList.Front())
		state := element.(*xfrState)
		key := xfrStateKey{
			viewName: state.viewName,
			zoneName: state.zoneName,
			client:   state.client,
		}
		// Remove the element from the map.
		delete(t.startedMap, key)
	}
}

// Updates the existing zone transfer state. It is no-op if the
// specified transfer does not exist. This function is safe for concurrent use.
// It is typically called to update from xfrStatusStarted to xfrStatusMessage
// (when the started transfer fails). However, it also handles the case when
// the new state is xfrStatusConnected. This new state indicates that the
// transfer is starting up but successful connection was established to the
// primary server. So, we preserve the xfrStatusStarted status in the existing
// state, but update the rest of the fields. Note that we only want to track
// started, completed and failed transfers. The connected status indicates that
// the transfer is still started. The original transfer start time is preserved
// during the update.
func (t *xfrTracker) updateState(key *xfrStateKey, s *xfrState) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	// Check if the given transfer exists.
	element, ok := t.startedMap[*key]
	if !ok {
		// It does not exist. Nothing to update.
		return
	}
	state := element.Value.(*xfrState)
	// Preserve the original start time.
	s.startTime = state.startTime
	if s.status == xfrStatusConnected && state.status == xfrStatusStarted {
		// If the status is connected it indicates that we have received an
		// intermediate message between starting and completing the transfer.
		// This message has some useful information concerning the server from
		// which the transfer is received. Let's hold this new status but still
		// keep it as started rather than connected. The user is not interested
		// in distinguishing between started and connected transfers.
		s.status = xfrStatusStarted
	}
	// If the transfer was updated we can move it to the back of the list to keep
	// the most recent one at the end.
	t.startedList.MoveToBack(element)
	// Update the element with the new state.
	element.Value = s
}

// Moves the zone transfer state to the completed container. It is called when
// the zone transfer has completed. It is safe for concurrent use. It removes
// the state from the started container and adds it to the completed container.
// If the number of completed transfers exceeds the maximum allowed number, the
// oldest one is removed.
func (t *xfrTracker) moveStateToCompleted(key *xfrStateKey, s *xfrState) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	// Check if the given transfer exists.
	element, ok := t.startedMap[*key]
	if ok {
		// It exists, so let's remove it from the started container.
		t.startedList.Remove(element)
		delete(t.startedMap, *key)
	}
	// Add the state to the completed container.
	t.completedList.PushBack(s)
	// Ensure that the number of completed transfers does not exceed the maximum
	// allowed number. If it does, remove the oldest one.
	if t.completedList.Len() > t.maxStates {
		t.completedList.Remove(t.completedList.Front())
	}
}

// Feeds the log line to the XFR tracker. It parses the log line and updates
// the zone transfer state. It is safe for concurrent use.
func (t *xfrTracker) feed(logLine string) {
	s := t.parse(logLine)
	if s == nil {
		// The log line was not related to a zone transfer or we did not
		// consider it useful.
		return
	}
	key := xfrStateKey{
		viewName: s.viewName,
		zoneName: s.zoneName,
		client:   s.client,
	}
	switch s.status {
	case xfrStatusStarted:
		// This is a new zone transfer. Add it or replace the existing one for
		// the given zone and client.
		t.putState(&key, s)
	case xfrStatusMessage, xfrStatusConnected:
		// This is an intermediate message between starting and completing the transfer.
		// It may indicate an error or that the secondary established a connection
		// to the primary server.
		t.updateState(&key, s)
	case xfrStatusCompleted:
		// This is a completed zone transfer. Move it to the completed container.
		t.moveStateToCompleted(&key, s)
	default:
	}
}

// Returns the list of ongoing or stuck zone transfers. It is safe for concurrent use.
func (t *xfrTracker) getNotCompleted() []*xfrState {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	states := make([]*xfrState, 0, t.startedList.Len())
	for element := t.startedList.Front(); element != nil; element = element.Next() {
		states = append(states, element.Value.(*xfrState))
	}
	return states
}

// Returns the list of completed zone transfers. It is safe for concurrent use.
func (t *xfrTracker) getCompleted() []*xfrState {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	states := make([]*xfrState, 0, t.completedList.Len())
	for element := t.completedList.Front(); element != nil; element = element.Next() {
		states = append(states, element.Value.(*xfrState))
	}
	return states
}
