package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
)

type ZoneInventoryStateRelation string

const ZoneInventoryStateRelationApp ZoneInventoryStateRelation = "Daemon.App"

// Represents a status returned by a zone inventory on an agent.
// When server attempts to fetch the zone information from the
// agents it collects status codes they return. The agents may
// signal different kind of errors which the server stores in
// the database for individual agents. These different kinds of
// errors have the ZoneInventoryStatus type.
type ZoneInventoryStatus string

const (
	// Zone inventory on an agent performs a long lasting operation and
	// cannot perform the requested operation at this time.
	ZoneInventoryStatusBusy ZoneInventoryStatus = "busy"
	// There was an unclassified error when communicating with a zone
	// inventory on an agent.
	ZoneInventoryStatusErred ZoneInventoryStatus = "erred"
	// Communication with the zone inventory was successful.
	ZoneInventoryStatusOK ZoneInventoryStatus = "ok"
	// Zone inventory was not initialized (neither populated nor loaded).
	// Zones cannot be fetched until the zone inventory is initialized.
	ZoneInventoryStatusUninitialized ZoneInventoryStatus = "uninitialized"
)

// Represents a zone inventory state for a daemon in the database.
// The zone inventory state indicates whether or not fetching the zones
// was successful, what was the last error etc. The server can make it
// available to a user in the UI. It may also be used by the server to
// make a decision to retry fetching the zones or take some other action.
type ZoneInventoryState struct {
	ID        int64
	DaemonID  int64
	CreatedAt time.Time
	State     *ZoneInventoryStateDetails
	Daemon    *Daemon `pg:"rel:has-one"`
}

// Represents a zone inventory state in the database for a daemon. It is
// a part of a larger structure (i.e., ZoneInventoryState). It is held in
// the "state" column of the "zone_inventory_state" table as JSONB. Storing
// this data as JSONB makes this column flexible and extensible without a
// need to update the database schema.
type ZoneInventoryStateDetails struct {
	Status    ZoneInventoryStatus
	Error     *string
	ZoneCount *int64
}

// Instantiates the inventory state details.
func NewZoneInventoryStateDetails() *ZoneInventoryStateDetails {
	return &ZoneInventoryStateDetails{
		Status: ZoneInventoryStatusOK,
	}
}

// Sets the status and error. Note that the specified error can be nil which clears
// the error (e.g., in case of the OK status).
func (state *ZoneInventoryStateDetails) SetStatus(status ZoneInventoryStatus, err error) {
	state.Status = status
	state.Error = nil
	if err != nil {
		s := err.Error()
		state.Error = &s
	}
}

// Sets the total number of zones.
func (state *ZoneInventoryStateDetails) SetTotalZones(totalZones int64) {
	state.ZoneCount = &totalZones
}

// Instantiates the zone inventory state for a given daemon.
func NewZoneInventoryState(daemonID int64, state *ZoneInventoryStateDetails) *ZoneInventoryState {
	return &ZoneInventoryState{
		DaemonID: daemonID,
		State:    state,
	}
}

// Upsers zone inventory state in a database.
func AddZoneInventoryState(db pg.DBI, state *ZoneInventoryState) error {
	_, err := db.Model(state).OnConflict("(daemon_id) DO UPDATE").
		Set("created_at = EXCLUDED.created_at").
		Set("state = EXCLUDED.state").
		Insert()
	if err != nil {
		return err
	}
	return nil
}

// Returns zone inventory state for a daemon or nil if it doesn't exist.
func GetZoneInventoryState(db pg.DBI, daemonID int64, relations ...ZoneInventoryStateRelation) (*ZoneInventoryState, error) {
	state := &ZoneInventoryState{}
	err := db.Model(state).
		Where("daemon_id = ?", daemonID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "failed to get zone inventory state for daemon %d", daemonID)
		return nil, err
	}
	return state, err
}

// Returns zone inventory states for all daemons.
func GetZoneInventoryStates(db pg.DBI, relations ...ZoneInventoryStateRelation) ([]ZoneInventoryState, int, error) {
	var states []ZoneInventoryState
	q := db.Model(&states)
	for _, relation := range relations {
		q.Relation(string(relation))
	}
	count, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, count, nil
		}
		err = pkgerrors.Wrap(err, "failed to get zone inventory states")
		return nil, count, err
	}
	return states, count, nil
}
