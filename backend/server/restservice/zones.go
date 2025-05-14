package restservice

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	dnslib "github.com/miekg/dns"
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
			AppID:            state.Daemon.AppID,
			AppName:          state.Daemon.App.Name,
			CreatedAt:        strfmt.DateTime(state.CreatedAt),
			DaemonID:         state.DaemonID,
			Error:            state.State.Error,
			Status:           string(state.State.Status),
			ZoneConfigsCount: state.State.ZoneCount,
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
func (r *RestAPI) GetZoneRRs(ctx context.Context, params dns.GetZoneRRsParams) middleware.Responder {
	var (
		restRrs               []*models.ZoneRR
		alreadyRequestedError *dnsop.ManagerRRsAlreadyRequestedError
		busyError             *agentcomm.ZoneInventoryBusyError
		notInitedError        *agentcomm.ZoneInventoryNotInitedError
	)
	for rrs, err := range r.DNSManager.GetZoneRRs(params.ZoneID, params.DaemonID, params.ViewName) {
		if err != nil {
			msg := "Failed to get zone contents using zone transfer"
			log.WithError(err).Error(msg)
			switch {
			case errors.As(err, &alreadyRequestedError):
				// There is another request in progress for the same zone.
				rsp := dns.NewGetZoneRRsDefault(http.StatusConflict).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(alreadyRequestedError, msg).Error()),
				})
				return rsp
			case errors.As(err, &busyError):
				// The zone inventory is busy populating or sending zones to the server.
				rsp := dns.NewGetZoneRRsDefault(http.StatusConflict).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(busyError, msg).Error()),
				})
				return rsp
			case errors.As(err, &notInitedError):
				// The zone inventory is not initialized.
				rsp := dns.NewGetZoneRRsDefault(http.StatusServiceUnavailable).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(notInitedError, msg).Error()),
				})
				return rsp
			default:
				// An unknown error occurred.
				rsp := dns.NewGetZoneRRsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
					Message: storkutil.Ptr(errors.WithMessage(err, msg).Error()),
				})
				return rsp
			}
		}
		for _, rr := range rrs {
			// Extract the RR data.
			var data string
			fields := strings.Fields(rr.String())
			// The full RR record has the following format:
			// <name> <ttl> <class> <type> <data>
			// We are interested in extracting the <data> field.
			if len(fields) > 4 {
				data = strings.Join(fields[4:], " ")
			}
			// Convert the RR to the REST API format.
			restRrs = append(restRrs, &models.ZoneRR{
				Name:    rr.Header().Name,
				TTL:     int64(rr.Header().Ttl),
				RrClass: dnslib.ClassToString[rr.Header().Class],
				RrType:  dnslib.TypeToString[rr.Header().Rrtype],
				Data:    data,
			})
		}
	}
	// Return the zone contents.
	payload := models.ZoneRRs{
		Items: restRrs,
	}
	rsp := dns.NewGetZoneRRsOK().WithPayload(&payload)
	return rsp
}
