package restservice

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/events"
)

func (r *RestAPI) getEvents(offset, limit int64, level dbmodel.EventLevel, daemonType *string, appType *string, machineID *int64, userID *int64, sortField string, sortDir dbmodel.SortDirEnum) (*models.Events, error) {
	// Get the events from the database.
	dbEvents, total, err := dbmodel.GetEventsByPage(r.DB, offset, limit, level, daemonType, appType, machineID, userID, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	events := &models.Events{
		Total: total,
	}

	// Convert events fetched from the database to REST.
	for _, dbEvent := range dbEvents {
		event := models.Event{
			ID:        dbEvent.ID,
			CreatedAt: strfmt.DateTime(dbEvent.CreatedAt),
			Text:      dbEvent.Text,
			Level:     int64(dbEvent.Level),
			Details:   dbEvent.Details,
		}
		events.Items = append(events.Items, &event)
	}

	return events, nil
}

// Get list of events with specifying an offset and a limit.
func (r *RestAPI) GetEvents(ctx context.Context, params events.GetEventsParams) middleware.Responder {
	var start int64
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	var level dbmodel.EventLevel
	if params.Level != nil {
		level = dbmodel.EventLevel(*params.Level)
	}

	// get events from db
	eventRecs, err := r.getEvents(start, limit, level, params.DaemonType, params.AppType, params.Machine, params.User, "created_at", dbmodel.SortDirDesc)
	if err != nil {
		msg := "Problem fetching events from the database"
		log.Error(err)
		rsp := events.NewGetEventsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Everything fine.
	rsp := events.NewGetEventsOK().WithPayload(eventRecs)
	return rsp
}
