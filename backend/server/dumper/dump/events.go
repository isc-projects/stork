package dump

import (
	"github.com/go-pg/pg/v10"
	dbmodel "isc.org/stork/server/database/model"
)

// Dumps the events related to the machine.
type EventsDump struct {
	BasicDump
	db        *pg.DB
	machineID int64
}

// Extended event structure with additional, derived members to improve the
// dump clarity.
type EventExtended struct {
	dbmodel.Event
	LevelText string
}

// Constructs new events dump instance.
func NewEventsDump(db *pg.DB, machine *dbmodel.Machine) *EventsDump {
	return &EventsDump{
		*NewBasicDump("events"),
		db, machine.ID,
	}
}

// Executes the event dump. It fetches at most 1000 the latest events from the
// database for a specific machine.
func (d *EventsDump) Execute() error {
	events, _, err := dbmodel.GetEventsByPage(
		d.db,
		// Limit and offset
		0, 1000,
		// Severity - accepts all events
		0,
		// Filters
		nil, nil, &d.machineID, nil,
		// Sorting
		"created_at", dbmodel.SortDirDesc)
	if err != nil {
		return err
	}

	eventsExtended := make([]EventExtended, len(events))
	for i, event := range events {
		eventsExtended[i] = EventExtended{
			Event:     event,
			LevelText: event.Level.String(),
		}
	}

	d.AppendArtifact(NewBasicStructArtifact(
		"latest", eventsExtended,
	))
	return nil
}
