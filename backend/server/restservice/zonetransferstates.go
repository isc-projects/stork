package restservice

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/daemondata/bind9xfr"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/dns"
	storkutil "isc.org/stork/util"
)

// Implements the GET call to retrieve the list of the zone transfer states.
func (r *RestAPI) GetZoneTransferStates(ctx context.Context, params dns.GetZoneTransferStatesParams) middleware.Responder {
	var offset int
	if params.Start != nil {
		offset = int(*params.Start)
	}
	limit := 10
	if params.Limit != nil {
		limit = int(*params.Limit)
	}
	includeLocal := false
	if params.IncludeLocal != nil {
		includeLocal = *params.IncludeLocal
	}
	filter := &dbmodel.GetZoneTransferStatesFilter{
		Offset:          &offset,
		Limit:           &limit,
		Serial:          params.Serial,
		ServerMachineID: params.ServerMachineID,
		ClientMachineID: params.ClientMachineID,
		Text:            params.Text,
		ExcludeLocal:    !includeLocal,
	}
	for _, status := range params.Status {
		filter.EnableStatus(bind9xfr.Status(status))
	}

	sortField := "started_at"
	if params.SortField != nil {
		sortField = *params.SortField
	}
	sortDir := dbmodel.SortDirDesc
	if params.SortDir != nil {
		sortDir = dbmodel.SortDirEnum(*params.SortDir)
	}
	states, total, err := dbmodel.GetZoneTransferStatesByPage(r.DB, filter, sortField, sortDir, dbmodel.ZoneTransferStateRelationClientMachine, dbmodel.ZoneTransferStateRelationServerMachine)
	if err != nil {
		msg := "Failed to get zone transfer states from the database"
		log.WithError(err).Error(msg)
		rsp := dns.NewGetZoneTransferStatesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	var restStates []*models.ZoneTransferState
	for _, state := range states {
		restState := &models.ZoneTransferState{
			ID:              state.ID,
			CreatedAt:       strfmt.DateTime(state.CreatedAt),
			ViewName:        state.ViewName,
			ZoneName:        state.ZoneName,
			Serial:          state.Serial,
			Client:          state.Client,
			Server:          state.Server,
			MessagesCount:   state.MessagesCount,
			RecordsCount:    state.RecordsCount,
			BytesCount:      state.BytesCount,
			BytesPerSecond:  state.BytesPerSecond,
			Duration:        strfmt.Duration(state.EffectiveDuration),
			Status:          state.Status.String(),
			StartedAt:       strfmt.DateTime(state.StartedAt),
			Message:         state.Message,
			ClientMachineID: state.ClientMachineID,
			ServerMachineID: state.ServerMachineID,
		}
		if !state.CompletedAt.IsZero() {
			restState.CompletedAt = storkutil.Ptr(strfmt.DateTime(state.CompletedAt))
		}
		if state.ClientMachine != nil {
			restState.ClientMachineAddress = state.ClientMachine.Address
			restState.ClientMachineAgentPort = state.ClientMachine.AgentPort
		}
		if state.ServerMachine != nil {
			restState.ServerMachineAddress = state.ServerMachine.Address
			restState.ServerMachineAgentPort = state.ServerMachine.AgentPort
		}
		restStates = append(restStates, restState)
	}
	return dns.NewGetZoneTransferStatesOK().WithPayload(&models.ZoneTransferStates{
		Items: restStates,
		Total: total,
	})
}
