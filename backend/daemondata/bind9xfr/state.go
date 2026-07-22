package bind9xfr

import (
	"fmt"
	"reflect"
	"slices"
	"time"
)

var _ fmt.Stringer = (*Status)(nil)

// The zone transfer status type.
type Status string

const (
	// The zone transfer has started.
	StatusStarted Status = "started"
	// The zone transfer has completed.
	StatusCompleted Status = "completed"
	// The zone is up-to-date.
	StatusUpToDate Status = "up-to-date"
	// The last received log message neither marks the beginning nor the end of the zone
	// transfer. It is also not clear whether or not it was a failure. It may indicate
	// some kind of problem, though.
	StatusMessage Status = "message"
	// The zone transfer has failed.
	StatusFailed Status = "failed"
)

// Returns the string representation of the status.
func (s Status) String() string {
	return string(s)
}

// A time format used in the parsed log messages.
type TimeFormat int

const (
	// The time format is unknown/unrecognized.
	TimeFormatUnknown TimeFormat = iota
	// The time format is in the RFC3339 format (e.g., 2026-02-23T10:41:27.071Z).
	TimeFormatRFC3339
	// The time format is in the BIND 9 format (e.g., 23-Feb-2026 10:41:27.071).
	TimeFormatBind9
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
	Serial         *int64
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

// Checks if the state has any of the specified statuses.
func (s *State) HasAnyStatus(statuses ...Status) bool {
	return s != nil && slices.Contains(statuses, s.Status)
}

// Checks if the state describes an outgoing zone transfer.
func (s *State) IsXFROut() bool {
	return s.Client != ""
}

// Derives selected fields from the source state to the destination state.
// The field value is copied to the destination of the field is not set
// in the destination but is set in the source. This function is useful when
// there is an existing state and the new state should inherit selected fields
// from it.
func (s *State) Derive(newState *State, fieldNames ...string) {
	if s == nil || newState == nil {
		return
	}
	source := reflect.ValueOf(s).Elem()
	dest := reflect.ValueOf(newState).Elem()
	for field, sourceValue := range source.Fields() {
		if !slices.Contains(fieldNames, field.Name) {
			continue
		}
		destValue := dest.FieldByName(field.Name)
		if !sourceValue.IsZero() && destValue.IsZero() {
			destValue.Set(sourceValue)
		}
	}
}

// The key used to index the started zone transfers in the LRU cache.
// The client is optional - it is empty for incoming zone transfers.
type StateKey struct {
	ViewName string
	ZoneName string
	Client   string
}
