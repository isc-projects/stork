package restservice

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/events"
	storkutil "isc.org/stork/util"
)

func (r *RestAPI) getEvents(offset, limit int64, level dbmodel.EventLevel, daemonName *string, machineID *int64, userID *int64, sortField string, sortDir dbmodel.SortDirEnum) (*models.Events, error) {
	// Get the events from the database.
	dbEvents, total, err := dbmodel.GetEventsByPage(r.DB, offset, limit, level, daemonName, machineID, userID, sortField, sortDir)
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

	sortField := "created_at"
	if params.SortField != nil {
		sortField = *params.SortField
	}

	sortDir := dbmodel.SortDirDesc
	if params.SortDir != nil {
		sortDir = dbmodel.SortDirEnum(*params.SortDir)
	}

	// get events from db
	eventRecs, err := r.getEvents(start, limit, level, params.DaemonType, params.Machine, params.User, sortField, sortDir)
	if err != nil {
		msg := "Problem fetching events from the database"
		log.WithError(err).Error(msg)
		rsp := events.NewGetEventsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Everything fine.
	rsp := events.NewGetEventsOK().WithPayload(eventRecs)
	return rsp
}

func (r *RestAPI) DeleteEvents(ctx context.Context, params events.DeleteEventsParams) middleware.Responder {
	_, user := r.SessionManager.Logged(ctx)
	if user == nil {
		msg := "Unable to identify the user requesting deletion (for logging purposes)"
		rsp := events.NewDeleteEventsDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if !user.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		msg := "User is forbidden to clear the event log"
		rsp := events.NewDeleteEventsDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	deleteCount, err := dbmodel.DeleteAllEvents(r.DB)
	if err != nil {
		msg := "Problem deleting events from the database"
		log.WithError(err).Error(msg)
		rsp := events.NewDeleteEventsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	eventsStr := storkutil.FormatNoun(int64(deleteCount), "event", "s")
	r.EventCenter.AddInfoEvent(fmt.Sprintf("[%d] %s cleared the event log and removed %s", user.ID, user.Login, eventsStr), user)
	rsp := events.NewDeleteEventsNoContent()
	return rsp
}
