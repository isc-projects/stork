package dump

import (
	"github.com/go-pg/pg/v9"
	dbmodel "isc.org/stork/server/database/model"
)

// Dumps the events related to the machine.
type EventsDump struct {
	BasicDump
	db        *pg.DB
	machineID int64
}

// Constructs new events dump instance.
func NewEventsDump(db *pg.DB, machine *dbmodel.Machine) *EventsDump {
	return &EventsDump{
		*NewBasicDump("events"),
		db, machine.ID,
	}
}

// Executes the event dump. It fetches at most 1000 the latest error and warning
// events from the database for a specific machine.
func (d *EventsDump) Execute() error {
	events, _, err := dbmodel.GetEventsByPage(
		d.db,
		// Limit and offset
		0, 1000,
		// Severity
		1,
		// Filters
		nil, nil, &d.machineID, nil,
		// Sorting
		"created_at", dbmodel.SortDirDesc)
	if err != nil {
		return err
	}

	d.AppendArtifact(NewBasicStructArtifact(
		"latest", events,
	))
	return nil
}
