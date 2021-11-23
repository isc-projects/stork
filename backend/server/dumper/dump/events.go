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

// Executes the event dump. It fetches the latest events from the database
// for a specific machine.
func (d *EventsDump) Execute() error {
	events, _, err := dbmodel.GetEventsByPage(d.db, 0, 1000, 1, nil, nil, &d.machineID, nil, "", dbmodel.SortDirAny)
	if err != nil {
		return err
	}

	d.AppendArtifact(NewBasicStructArtifact(
		"all", events,
	))
	return nil
}
