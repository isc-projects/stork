package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
)

// The event severity level.
type EventLevel int64

// Event levels.
const (
	EvInfo    EventLevel = 0 // informational
	EvWarning EventLevel = 1 // someone should look into this
	EvError   EventLevel = 2 // there is a serious problem
)

// Returns a human-readable representation of the event level.
func (t EventLevel) String() string {
	switch t {
	case EvInfo:
		return "info"
	case EvWarning:
		return "warning"
	case EvError:
		return "error"
	default:
		return "unknown"
	}
}

// SSE stream type used in the event relations.
type SSEStream string

// Typical messages sent in the SSE stream.
const (
	SSERegularMessage = SSEStream("message")
	SSEConnectivity   = SSEStream("connectivity")
	SSERegistration   = SSEStream("registration")
)

// Relations between the event and other entities.
type Relations struct {
	MachineID int64 `json:",omitempty"`
	AppID     int64 `json:",omitempty"`
	SubnetID  int64 `json:",omitempty"`
	DaemonID  int64 `json:",omitempty"`
	UserID    int64 `json:",omitempty"`
}

// Represents an event held in event table in the database.
type Event struct {
	ID         int64
	CreatedAt  time.Time
	Text       string
	Level      EventLevel `pg:",use_zero"`
	Relations  *Relations
	Details    string
	SSEStreams []SSEStream `json:",omitempty" pg:"sse_streams,array"`
}

// Add given event to the database.
func AddEvent(db *pg.DB, event *Event) error {
	_, err := db.Model(event).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem inserting event %+v", event)
	}
	return err
}

// Fetches a collection of events from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. Limit has to be greater then 0, otherwise error is returned.
// The level indicates the lowest level of events that should be
// returned (0 - info, 1 - warning, 2 - error). daemonType and appType
// allows selecting events only from given type of app ('kea',
// 'bind9') or daemon (e.g. 'named' or 'dhcp4'. machineID and userID
// allows selecting events connected with indicated machine or
// user. sortField allows indicating sort column in database and
// sortDir allows selection the order of sorting. If sortField is
// empty then id is used for sorting. If SortDirAny is used then ASC
// order is used.
func GetEventsByPage(db *pg.DB, offset int64, limit int64, level EventLevel, daemonType *string, appType *string, machineID *int64, userID *int64, sortField string, sortDir SortDirEnum) ([]Event, int64, error) {
	if limit == 0 {
		return nil, 0, pkgerrors.New("limit should be greater than 0")
	}
	events := []Event{}

	// prepare query
	q := db.Model(&events)
	if level > 0 {
		q = q.Where("level >= ?", level)
	}
	if daemonType != nil {
		q = q.Join("JOIN daemon ON daemon.id = CAST (relations->>'DaemonID' AS INTEGER)")
		q = q.Where("daemon.name = ?", daemonType)
	}
	if appType != nil {
		q = q.Join("JOIN app ON app.id = CAST (relations->>'AppID' AS INTEGER)")
		q = q.Where("app.type = ?", appType)
	}
	if machineID != nil {
		q = q.Where("CAST (relations->>'MachineID' AS INTEGER) = ?", *machineID)
	}
	if userID != nil {
		q = q.Where("CAST (relations->>'UserID' AS INTEGER) = ?", *userID)
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("event", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return []Event{}, 0, nil
		}
		return nil, 0, pkgerrors.Wrapf(err, "problem getting events")
	}
	return events, int64(total), nil
}
