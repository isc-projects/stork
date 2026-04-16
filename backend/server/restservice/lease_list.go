package restservice

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
)

func (r *RestAPI) convertLeaseFromRestAPI(dbLease *dbmodel.Lease) *models.Lease {
	// TODO: implement conversion
	return nil
}

// Fetches leases from the database and converts to the data formats
// used in the REST API.
func (r *RestAPI) getLeases(offset, limit int64, filters dbmodel.LeasesByPageFilters, sortField string, sortDir dbmodel.SortDirEnum) (*models.Leases, error) {
	// Get the hosts from the database.
	dbLeases, total, err := dbmodel.GetLeasesByPage(r.DB, offset, limit, filters, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	hosts := &models.Leases{
		Total: total,
	}

	// Convert hosts fetched from the database to REST.
	for i := range dbLeases {
		host := r.convertLeaseFromRestAPI(&dbLeases[i])
		hosts.Items = append(hosts.Items, host)
	}

	return hosts, nil
}

func (r *RestAPI) GetLeaseList(ctx context.Context, params dhcp.GetLeaseListParams) middleware.Responder {
	var start int64
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	sortField := ""
	if params.SortField != nil {
		sortField = *params.SortField
	}

	sortDir := dbmodel.SortDirAny
	if params.SortDir != nil {
		sortDir = dbmodel.SortDirEnum(*params.SortDir)
	}

	// Get leases from DB.
	filters := dbmodel.LeasesByPageFilters{
		MachineID:     params.MachineID,
		DaemonID:      params.DaemonID,
		SubnetID:      params.SubnetID,
		LocalSubnetID: params.LocalSubnetID,
		FilterText:    params.Text,
	}
	leases, err := r.getLeases(start, limit, filters, sortField, sortDir)
	if err != nil {
		msg := "Problem fetching hosts from the database"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetHostsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	rsp := dhcp.NewGetLeasesOK().WithPayload(leases)
	return rsp
}
