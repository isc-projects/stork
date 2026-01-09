package restservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dnsop"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/dns"
	storkutil "isc.org/stork/util"
)

// Returns a single DNS zone.
func (r *RestAPI) GetZone(ctx context.Context, params dns.GetZoneParams) middleware.Responder {
	// Find the zone in the database.
	dbZone, err := dbmodel.GetZoneByID(r.DB, params.ZoneID, dbmodel.ZoneRelationLocalZonesDaemon, dbmodel.ZoneRelationLocalZonesMachine)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("Problem fetching DNS zone with ID %d from db", params.ZoneID)
		log.WithError(err).Error(msg)
		rsp := dns.NewGetZoneDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbZone == nil {
		// Zone not found.
		msg := fmt.Sprintf("Cannot find DNS zone with ID %d", params.ZoneID)
		rsp := dns.NewGetZoneDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Zone found. Convert it to the format used in REST API.
	var restLocalZones []*models.LocalZone
	for _, localZone := range dbZone.LocalZones {
		virtualApp := localZone.Daemon.GetVirtualApp()
		restLocalZones = append(restLocalZones, &models.LocalZone{
			AppID:    virtualApp.ID,
			AppName:  virtualApp.Name,
			Class:    localZone.Class,
			DaemonID: localZone.DaemonID,
			LoadedAt: strfmt.DateTime(localZone.LoadedAt),
			Serial:   localZone.Serial,
			Rpz:      localZone.RPZ,
			View:     localZone.View,
			ZoneType: localZone.Type,
		})
	}
	restZone := models.Zone{
		ID:         dbZone.ID,
		Name:       dbZone.Name,
		Rname:      dbZone.Rname,
		LocalZones: restLocalZones,
	}
	rsp := dns.NewGetZoneOK().WithPayload(&restZone)
	return rsp
}

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
	sortField := "rname"
	if params.SortField != nil {
		sortField = *params.SortField
	}
	sortDir := dbmodel.SortDirAsc
	if params.SortDir != nil {
		sortDir = dbmodel.SortDirEnum(*params.SortDir)
	}

	var daemonName *daemonname.Name
	if params.AppType != nil {
		switch *params.AppType {
		case "bind9":
			daemonName = storkutil.Ptr(daemonname.Bind9)
		case "pdns":
			daemonName = storkutil.Ptr(daemonname.PDNS)
		default:
			// Unknown app type, return empty result.
			payload := models.Zones{
				Items: []*models.Zone{},
				Total: 0,
			}
			rsp := dns.NewGetZonesOK().WithPayload(&payload)
			return rsp
		}
	}

	var machineIDPtr *int64
	if params.AppID != nil {
		machineID, err := dbmodel.GetMachineIDByVirtualAppID(r.DB, *params.AppID)
		if err != nil {
			msg := "Failed to get machine ID by virtual app ID"
			log.WithError(err).Error(msg)
			rsp := dns.NewGetZonesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
		if machineID == 0 {
			// No machine with the specified virtual app ID exist, return empty result.
			payload := models.Zones{
				Items: []*models.Zone{},
				Total: 0,
			}
			rsp := dns.NewGetZonesOK().WithPayload(&payload)
			return rsp
		}
		machineIDPtr = &machineID
	}

	// Apply paging parameters and zone-specific filters.
	filter := &dbmodel.GetZonesFilter{
		MachineID:  machineIDPtr,
		DaemonName: daemonName,
		Class:      params.Class,
		RPZ:        params.Rpz,
		Serial:     params.Serial,
		Text:       params.Text,
		Offset:     storkutil.Ptr(offset),
		Limit:      storkutil.Ptr(limit),
	}
	for _, zoneType := range params.ZoneType {
		filter.EnableZoneType(dbmodel.ZoneType(zoneType))
	}
	// Get the zones from the database.
	zones, total, err := dbmodel.GetZones(r.DB, filter, sortField, sortDir, dbmodel.ZoneRelationLocalZonesDaemon, dbmodel.ZoneRelationLocalZonesAccessPoints, dbmodel.ZoneRelationLocalZonesMachine)
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
			app := localZone.Daemon.GetVirtualApp()
			restLocalZones = append(restLocalZones, &models.LocalZone{
				AppID:    app.ID,
				AppName:  app.Name,
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
	states, count, err := dbmodel.GetZoneInventoryStates(r.DB, dbmodel.ZoneInventoryStateRelationDaemon, dbmodel.ZoneInventoryStateRelationAccessPoints, dbmodel.ZoneInventoryStateRelationMachine)
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
		app := state.Daemon.GetVirtualApp()
		restStates = append(restStates, &models.ZoneInventoryState{
			AppID:              app.ID,
			AppName:            app.Name,
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
	_, err := r.DNSManager.FetchZones(10, 1000)
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
// The returned data is filtered according to the parameters. However, all
// RRs are cached regardless of the filtering. Future calls to this endpoint
// will return the cached data.
func (r *RestAPI) GetZoneRRs(ctx context.Context, params dns.GetZoneRRsParams) middleware.Responder {
	var (
		restRrs               []*models.ZoneRR
		alreadyRequestedError *dnsop.ManagerRRsAlreadyRequestedError
		busyError             *agentcomm.ZoneInventoryBusyError
		notInitedError        *agentcomm.ZoneInventoryNotInitedError
		cached                bool
		zoneTransferAt        time.Time
		total                 int
		filter                *dbmodel.GetZoneRRsFilter
	)
	// Apply filtering if requested.
	if params.Start != nil || params.Limit != nil || len(params.RrType) > 0 || params.Text != nil {
		filter = dbmodel.NewGetZoneRRsFilterWithParams(params.Start, params.Limit, params.RrType, params.Text)
	}
	for rrResponse := range r.DNSManager.GetZoneRRs(params.ZoneID, params.DaemonID, params.ViewName, filter, dnsop.GetZoneRRsOptionExcludeTrailingSOA) {
		if rrResponse.Err != nil {
			msg := "Failed to get zone contents"
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
		// If the records are not cached, the returned total number is increasing as the
		// new records are returned by the agent. We don't know until the last record how many
		// records are to be returned. Therefore, we track the total number and take the
		// highest value.
		total = max(total, rrResponse.Total)
	}
	// Return the zone contents.
	payload := models.ZoneRRs{
		Cached:         cached,
		Items:          restRrs,
		Total:          int64(total),
		ZoneTransferAt: strfmt.DateTime(zoneTransferAt),
	}
	rsp := dns.NewGetZoneRRsOK().WithPayload(&payload)
	return rsp
}

// Refreshes the resource for a zone using zone transfer, and return the newly
// cached RRs with filtering. The zone transfer is initiated regardless of whether
// the zone RRs are present in the database or not.
func (r *RestAPI) PutZoneRRsCache(ctx context.Context, params dns.PutZoneRRsCacheParams) middleware.Responder {
	var (
		restRrs               []*models.ZoneRR
		alreadyRequestedError *dnsop.ManagerRRsAlreadyRequestedError
		busyError             *agentcomm.ZoneInventoryBusyError
		notInitedError        *agentcomm.ZoneInventoryNotInitedError
		cached                bool
		zoneTransferAt        time.Time
		total                 int
		filter                *dbmodel.GetZoneRRsFilter
	)
	// Apply filtering if requested.
	if params.Start != nil || params.Limit != nil || len(params.RrType) > 0 || params.Text != nil {
		filter = dbmodel.NewGetZoneRRsFilterWithParams(params.Start, params.Limit, params.RrType, params.Text)
	}
	for rrResponse := range r.DNSManager.GetZoneRRs(params.ZoneID, params.DaemonID, params.ViewName, filter, dnsop.GetZoneRRsOptionForceZoneTransfer, dnsop.GetZoneRRsOptionExcludeTrailingSOA) {
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
		// The records are not cached because we're forcing the zone transfer. The returned
		// total number is increasing as the new records are returned by the agent. We don't
		// know until the last record how many records are to be returned. Therefore, we
		// track the total number and take the highest value.
		total = max(total, rrResponse.Total)
	}
	// Return the zone contents.
	payload := models.ZoneRRs{
		Cached:         cached,
		Items:          restRrs,
		Total:          int64(total),
		ZoneTransferAt: strfmt.DateTime(zoneTransferAt),
	}
	rsp := dns.NewPutZoneRRsCacheOK().WithPayload(&payload)
	return rsp
}
