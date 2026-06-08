package bind9xfr

import "time"

// The zone transfer status type.
type Status int

const (
	// This is a default status indicating that the parsed log message was unrecognized
	// and could not be used to determine the actual zone transfer status. Messages with
	// this status are discarded.
	StatusUnknown Status = iota
	// The zone transfer has started.
	StatusStarted
	// The incoming zone transfer has started on the secondary server, and the last
	// log message indicated that the secondary server successfully connected to the
	// primary server.
	StatusConnected
	// The zone transfer has completed.
	StatusCompleted
	// The last received log message neither marks the beginning nor the end of the zone
	// transfer. It is typically a message received during the zone transfer indicating
	// some kind of problem.
	StatusMessage
)

// A time format used in the parsed log messages.
type TimeFormat int

const (
	// The time format is in the RFC3339 format (e.g., 2026-02-23T10:41:27.071Z).
	TimeFormatRFC3339 TimeFormat = iota
	// The time format is in the BIND 9 format (e.g., 23-Feb-2026 10:41:27.071).
	TimeFormatBind9
	// The time format is unknown/unrecognized.
	TimeFormatUnknown
)

// The zone transfer state. An instance of this structure is returned for
// each parsed log message pertaining to a zone transfer. The state includes
// the data extracted from the log message such as the zone name, view name,
// client address (for outgoing zone transfers) and server address (for incoming
// zone transfers). It also includes suitable timestamps and zone transfer
// statistics.
type State struct {
	ViewName       string
	ZoneName       string
	Serial         int64
	Client         string
	Server         string
	MessagesCount  int64
	RecordsCount   int64
	BytesCount     int64
	Duration       time.Duration
	Status         Status
	StartTime      time.Time
	CompletionTime time.Time
	TimeFormat     TimeFormat
	Message        string
}

// The key used to index the started zone transfers in the LRU cache.
// The client is optional - it is empty for incoming zone transfers.
type StateKey struct {
	ViewName string
	ZoneName string
	Client   string
}
