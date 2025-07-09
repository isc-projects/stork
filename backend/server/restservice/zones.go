package restservice

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dnsop"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/dns"
	storkutil "isc.org/stork/util"
)

// Returns a list DNS zones with paging.
func (r *RestAPI) GetZones(ctx context.Context, params dns.GetZonesParams) middleware.Responder {
	// Set paging parameters.
	var offset int
	if params.Start != nil {
		offset = int(*params.Start)
	}
	limit := 10
	if params.Limit != nil {
		limit = int(*params.Limit)
	}
	// Apply paging parameters and zone-specific filters.
	filter := &dbmodel.GetZonesFilter{
		AppID:   params.AppID,
		AppType: params.AppType,
		Class:   params.Class,
		Serial:  params.Serial,
		Text:    params.Text,
		Offset:  storkutil.Ptr(offset),
		Limit:   storkutil.Ptr(limit),
	}
	for _, zoneType := range params.ZoneType {
		filter.EnableZoneType(dbmodel.ZoneType(zoneType))
	}
	// Get the zones from the database.
	zones, total, err := dbmodel.GetZones(r.DB, filter, dbmodel.ZoneRelationLocalZonesApp)
	if err != nil {
		msg := "Failed to get zones from the database"
		log.WithError(err).Error(msg)
		rsp := dns.NewGetZonesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Convert the zones to the REST API format.
	var restZones []*models.Zone
	for _, zone := range zones {
		var restLocalZones []*models.LocalZone
		for _, localZone := range zone.LocalZones {
			restLocalZones = append(restLocalZones, &models.LocalZone{
				AppID:    localZone.Daemon.App.ID,
				AppName:  localZone.Daemon.App.Name,
				Class:    localZone.Class,
				DaemonID: localZone.DaemonID,
				LoadedAt: strfmt.DateTime(localZone.LoadedAt),
				Serial:   localZone.Serial,
				Rpz:      localZone.RPZ,
				View:     localZone.View,
				ZoneType: localZone.Type,
			})
		}
		restZones = append(restZones, &models.Zone{
			ID:         zone.ID,
			Name:       zone.Name,
			Rname:      zone.Rname,
			LocalZones: restLocalZones,
		})
	}
	// Return the zones.
	payload := models.Zones{
		Items: restZones,
		Total: int64(total),
	}
	rsp := dns.NewGetZonesOK().WithPayload(&payload)
	return rsp
}

// Get the states of fetching the DNS zone information from the remote zone inventories.
func (r *RestAPI) GetZonesFetch(ctx context.Context, params dns.GetZonesFetchParams) middleware.Responder {
	isFetching, appsCount, completedAppsCount := r.DNSManager.GetFetchZonesProgress()
	if isFetching {
		payload := models.ZonesFetchStatus{
			CompletedAppsCount: int64(completedAppsCount),
			AppsCount:          int64(appsCount),
		}
		rsp := dns.NewGetZonesFetchAccepted().WithPayload(&payload)
		return rsp
	}
	states, count, err := dbmodel.GetZoneInventoryStates(r.DB, dbmodel.ZoneInventoryStateRelationApp)
	if err != nil {
		msg := "Failed to get zones fetch states from the database"
		log.WithError(err).Error(msg)
		rsp := dns.NewGetZonesFetchDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if count == 0 {
		rsp := dns.NewGetZonesFetchNoContent()
		return rsp
	}
	var restStates []*models.ZoneInventoryState
	for _, state := range states {
		restStates = append(restStates, &models.ZoneInventoryState{
			AppID:              state.Daemon.AppID,
			AppName:            state.Daemon.App.Name,
			CreatedAt:          strfmt.DateTime(state.CreatedAt),
			DaemonID:           state.DaemonID,
			Error:              state.State.Error,
			Status:             string(state.State.Status),
			ZoneConfigsCount:   state.State.ZoneCount,
			DistinctZonesCount: state.State.DistinctZoneCount,
			BuiltinZonesCount:  state.State.BuiltinZoneCount,
		})
	}
	payload := models.ZoneInventoryStates{
		Items: restStates,
		Total: int64(count),
	}
	rsp := dns.NewGetZonesFetchOK().WithPayload(&payload)
	return rsp
}

// Begins fetching the zones from the zone inventories into the Stork server.
func (r *RestAPI) PutZonesFetch(ctx context.Context, params dns.PutZonesFetchParams) middleware.Responder {
	var alreadyFetchingError *dnsop.ManagerAlreadyFetchingError
	_, err := r.DNSManager.FetchZones(10, 1000, false)
	switch {
	case err == nil, errors.As(err, &alreadyFetchingError):
		return dns.NewPutZonesFetchAccepted()
	default:
		msg := "Failed to start fetching the zones"
		log.WithError(err).Error(msg)
		rsp := dns.NewPutZonesFetchDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
}

// Returns the zone contents (RRs) for a specified view, zone and daemon.
// If the zone RRs are not present in the database, the zone transfer is
// initiated. The transferred data is cached in the database and returned.
// Future calls to this endpoint will return the cached data.

func (r *RestAPI) GetZoneRRs(ctx context.Context, params dns.GetZoneRRsParams) middleware.Responder {
	var (
		restRrs               []*models.ZoneRR
		alreadyRequestedError *dnsop.ManagerRRsAlreadyRequestedError
		busyError             *agentcomm.ZoneInventoryBusyError
		notInitedError        *agentcomm.ZoneInventoryNotInitedError
	)
	var (
		cached         bool
		zoneTransferAt time.Time
	)
	for rrResponse := range r.DNSManager.GetZoneRRs(params.ZoneID, params.DaemonID, params.ViewName) {
		if rrResponse.Err != nil {
			msg := "Failed to get zone contents from the database"
			log.WithError(rrResponse.Err).Error(msg)
			switch {
			case errors.As(rrResponse.Err, &alreadyRequestedError):
				// There is another request in progress for the same zone.
				rsp := dns.NewGetZoneRRsDefault(http.StatusConflict).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(alreadyRequestedError, msg).Error()),
				})
				return rsp
			case errors.As(rrResponse.Err, &busyError):
				// The zone inventory is busy populating or sending zones to the server.
				rsp := dns.NewGetZoneRRsDefault(http.StatusConflict).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(busyError, msg).Error()),
				})
				return rsp
			case errors.As(rrResponse.Err, &notInitedError):
				// The zone inventory is not initialized.
				rsp := dns.NewGetZoneRRsDefault(http.StatusServiceUnavailable).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(notInitedError, msg).Error()),
				})
				return rsp
			default:
				// An unknown error occurred.
				rsp := dns.NewGetZoneRRsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(rrResponse.Err, msg).Error()),
				})
				return rsp
			}
		}
		for _, rr := range rrResponse.RRs {
			// Convert the RR to the REST API format.
			restRrs = append(restRrs, &models.ZoneRR{
				Name:    rr.Name,
				TTL:     rr.TTL,
				RrClass: rr.Class,
				RrType:  rr.Type,
				Data:    rr.Rdata,
			})
		}
		cached = rrResponse.Cached
		zoneTransferAt = rrResponse.ZoneTransferAt
	}
	// Return the zone contents.
	payload := models.ZoneRRs{
		Cached:         cached,
		Items:          restRrs,
		ZoneTransferAt: strfmt.DateTime(zoneTransferAt),
	}
	rsp := dns.NewGetZoneRRsOK().WithPayload(&payload)
	return rsp
}

// Refreshes the resource for a zone using zone transfer, and return the newly
// cached RRs. The zone transfer is initiated regardless of whether the zone
// RRs are present in the database or not.
func (r *RestAPI) PutZoneRRsCache(ctx context.Context, params dns.PutZoneRRsCacheParams) middleware.Responder {
	var (
		restRrs               []*models.ZoneRR
		alreadyRequestedError *dnsop.ManagerRRsAlreadyRequestedError
		busyError             *agentcomm.ZoneInventoryBusyError
		notInitedError        *agentcomm.ZoneInventoryNotInitedError
		cached                bool
		zoneTransferAt        time.Time
	)
	for rrResponse := range r.DNSManager.GetZoneRRs(params.ZoneID, params.DaemonID, params.ViewName, dnsop.GetZoneRRsOptionForceZoneTransfer) {
		if rrResponse.Err != nil {
			msg := "Failed to refresh zone contents using zone transfer"
			log.WithError(rrResponse.Err).Error(msg)
			switch {
			case errors.As(rrResponse.Err, &alreadyRequestedError):
				// There is another request in progress for the same zone.
				rsp := dns.NewPutZoneRRsCacheDefault(http.StatusAccepted).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(alreadyRequestedError, msg).Error()),
				})
				return rsp
			case errors.As(rrResponse.Err, &busyError):
				// The zone inventory is busy populating or sending zones to the server.
				rsp := dns.NewPutZoneRRsCacheDefault(http.StatusConflict).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(busyError, msg).Error()),
				})
				return rsp
			case errors.As(rrResponse.Err, &notInitedError):
				// The zone inventory is not initialized.
				rsp := dns.NewPutZoneRRsCacheDefault(http.StatusServiceUnavailable).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(notInitedError, msg).Error()),
				})
				return rsp
			default:
				// An unknown error occurred.
				rsp := dns.NewPutZoneRRsCacheDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
					Message: &msg,
				})
				return rsp
			}
		}
		for _, rr := range rrResponse.RRs {
			// Convert the RR to the REST API format.
			restRrs = append(restRrs, &models.ZoneRR{
				Name:    rr.Name,
				TTL:     rr.TTL,
				RrClass: rr.Class,
				RrType:  rr.Type,
				Data:    rr.Rdata,
			})
		}
		cached = rrResponse.Cached
		zoneTransferAt = rrResponse.ZoneTransferAt
	}
	// Return the zone contents.
	payload := models.ZoneRRs{
		Cached:         cached,
		Items:          restRrs,
		ZoneTransferAt: strfmt.DateTime(zoneTransferAt),
	}
	rsp := dns.NewPutZoneRRsCacheOK().WithPayload(&payload)
	return rsp
}
