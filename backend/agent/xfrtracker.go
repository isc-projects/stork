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

func parseTime(tokens []string) (time.Time, xfrTimeFormat) {
	if len(tokens) == 0 {
		return time.Time{}, xfrTimeFormatUnknown
	}
	if len(tokens) > 1 {
		localTime := strings.Join(tokens[:2], " ")
		parsedTime, err := time.ParseInLocation("02-Jan-2006 15:04:05.000", localTime, time.Local)
		if err == nil {
			return parsedTime, xfrTimeFormatBind9
		}
	}
	parsedTime, err := time.Parse(time.RFC3339, tokens[0])
	if err == nil {
		return parsedTime, xfrTimeFormatRFC3339
	}
	return time.Time{}, xfrTimeFormatUnknown
}

func pull(tokens []string) (func(n int) (string, bool, int), func()) {
	seq := slices.Values(tokens)
	next, stop := iter.Pull(seq)
	index := 0
	return func(n int) (string, bool, int) {
		var (
			token string
			ok    bool
		)
		for i := 0; i < n; i++ {
			token, ok = next()
			if !ok {
				return "", false, index
			}
			index++
		}
		token = strings.TrimFunc(token, func(r rune) bool {
			return strings.ContainsRune("():',", r)
		})
		return token, ok, index
	}, stop
}

func (t *xfrTracker) parse(logLine string) *xfrState {
	// Limit the length of the log line to 1024 characters to avoid
	// parsing excessively long log lines.
	runes := []rune(logLine)
	if len(runes) > 1024 {
		runes = runes[:1024]
		logLine = string(runes)
	}

	// Remove any trailing dot from the log line. Some BIND9 logs contain
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

	var (
		parsedTime time.Time
		s          xfrState
	)

	// The log line typically begins with a timestamp. Parse it and determine
	// the time format used by BIND 9.
	parsedTime, s.timeFormat = parseTime(tokens)

	// Go over the tokens.
	next, stop := pull(tokens)
	defer stop()

	var (
		prevToken string
		msgIndex  int
	)
	for {
		// Get the next token.
		token, ok, _ := next(1)
		if !ok {
			// No more tokens to process.
			break
		}
		// Process the token.
		switch token {
		case "zone":
			parseZone(next, &s)
		case "view":
			parseView(next, &s)
		case "transfer":
			if index := parseTransfer(next, &s); index >= 0 {
				msgIndex = index
				s.status = xfrStatusMessage
			}
		case "serial":
			if err := parseSerial(next, &s); err != nil {
				log.WithError(err).Debugf("Failed to parse serial %s", token)
			}
		case "from":
			parseFrom(next, &s)
		case "client":
			parseClient(next, &s)
		case "started":
			s.status = xfrStatusStarted
			s.startTime = parsedTime
		case "completed", "ended":
			s.status = xfrStatusCompleted
			s.completionTime = parsedTime
		case "messages", "records", "bytes":
			if err := parseMessagesRecordsBytes(prevToken, parsedTime, token, &s); err != nil {
				log.WithError(err).Debugf("Failed to parse the number of %s", token)
			}
		case "secs":
			if err := parseSecs(prevToken, &s); err != nil {
				log.WithError(err).Debugf("Failed to parse the duration %s", prevToken)
			}
		}
		prevToken = token
	}
	if s.zoneName == "" {
		log.Debug("zone name not found in the log message")
		return nil
	}
	if msgIndex > 0 {
		s.message = strings.Join(tokens[msgIndex:], " ")
	}
	return &s
}

func parseTransfer(next func(n int) (string, bool, int), s *xfrState) int {
	var (
		token string
		ok    bool
		index int
	)
	if token, ok, index = next(1); !ok || token != "of" {
		return -1
	}
	index = parseZone(next, s)
	return index
}

func parseZone(next func(n int) (string, bool, int), s *xfrState) (index int) {
	if s.zoneName != "" {
		return
	}
	var (
		token string
		ok    bool
	)
	if token, ok, index = next(1); ok {
		split := strings.Split(token, "/")
		if len(split) > 1 {
			s.zoneName = split[0]
		}
		if len(split) > 2 {
			s.viewName = split[2]
		}
	}
	return
}

func parseView(next func(n int) (string, bool, int), s *xfrState) {
	if s.viewName != "" {
		return
	}
	if token, ok, _ := next(1); ok {
		s.viewName = token
	}
}

func parseFrom(next func(n int) (string, bool, int), s *xfrState) {
	if token, ok, _ := next(1); ok {
		split := strings.Split(token, "#")
		if len(split) > 0 {
			s.server = split[0]
		}
	}
}

func parseClient(next func(n int) (string, bool, int), s *xfrState) {
	if token, ok, _ := next(2); ok {
		split := strings.Split(token, "#")
		if len(split) > 0 {
			s.client = split[0]
		}
	}
}

func parseSerial(next func(n int) (string, bool, int), s *xfrState) (err error) {
	if s.serial != 0 {
		return
	}
	if token, ok, _ := next(1); ok {
		if s.serial, err = strconv.ParseInt(token, 10, 64); err != nil {
			err = errors.WithStack(err)
			return
		}
	}
	return err
}

func parseMessagesRecordsBytes(prevToken string, parsedTime time.Time, token string, s *xfrState) (err error) {
	if prevToken == "" {
		return
	}
	s.status = xfrStatusCompleted
	s.completionTime = parsedTime
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

func parseSecs(prevToken string, s *xfrState) (err error) {
	if prevToken == "" {
		return
	}
	duration, err := strconv.ParseFloat(prevToken, 64)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	s.duration = time.Duration(duration * float64(time.Second))
	return
}

func (t *xfrTracker) putState(key *xfrStateKey, s *xfrState) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	element, ok := t.startedMap[*key]
	if ok {
		element.Value = s
		t.startedList.MoveToBack(element)
	} else {
		element := t.startedList.PushBack(s)
		t.startedMap[*key] = element
	}
	if t.startedList.Len() > t.maxStates {
		element := t.startedList.Remove(t.startedList.Front())
		state := element.(*xfrState)
		key := xfrStateKey{
			viewName: state.viewName,
			zoneName: state.zoneName,
			client:   state.client,
		}
		delete(t.startedMap, key)
	}
}

func (t *xfrTracker) updateState(key *xfrStateKey, s *xfrState) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	element, ok := t.startedMap[*key]
	if !ok {
		return
	}
	state := element.Value.(*xfrState)
	if state.startTime.IsZero() {
		return
	}
	s.startTime = state.startTime
	t.startedList.MoveToBack(element)
	element.Value = s
}

func (t *xfrTracker) moveStateToCompleted(key *xfrStateKey, s *xfrState) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	element, ok := t.startedMap[*key]
	if ok {
		t.startedList.Remove(element)
		delete(t.startedMap, *key)
	}
	t.completedList.PushBack(s)
	if t.completedList.Len() > t.maxStates {
		t.completedList.Remove(t.completedList.Front())
	}
}

func (t *xfrTracker) feed(logLine string) {
	s := t.parse(logLine)
	if s == nil {
		return
	}
	key := xfrStateKey{
		viewName: s.viewName,
		zoneName: s.zoneName,
		client:   s.client,
	}
	switch s.status {
	case xfrStatusStarted:
		t.putState(&key, s)
	case xfrStatusMessage:
		t.updateState(&key, s)
	case xfrStatusCompleted:
		t.moveStateToCompleted(&key, s)
	default:
	}
}

func (t *xfrTracker) getNotCompleted() []*xfrState {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	states := make([]*xfrState, 0, t.startedList.Len())
	for element := t.startedList.Front(); element != nil; element = element.Next() {
		states = append(states, element.Value.(*xfrState))
	}
	return states
}

func (t *xfrTracker) getCompleted() []*xfrState {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	states := make([]*xfrState, 0, t.completedList.Len())
	for element := t.completedList.Front(); element != nil; element = element.Next() {
		states = append(states, element.Value.(*xfrState))
	}
	return states
}
