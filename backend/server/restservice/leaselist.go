package restservice

import (
	"context"
	"math"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
)

// convertLeaseToRestAPI converts a [dbmodel.Lease] (the database
// representation) into a [models.Lease] (the REST API representation). It can
// fail when the input is nil, when the input's CLTT is larger than an int64 can
// hold, and when the [dbmodel.Daemon] or [dbmodel.Subnet] are nil.
func convertLeaseToRestAPI(dbLease *dbmodel.Lease) (*models.Lease, error) {
	if dbLease == nil {
		return nil, errors.New("cannot convert a nil dbmodel.Lease to a models.Lease")
	}
	if dbLease.CLTT > math.MaxInt64 {
		return nil, errors.New("CLTT is greater than math.MaxInt64, so no safe conversion is possible")
	}
	if dbLease.Daemon == nil {
		return nil, errors.New("database did not return a Daemon for this Lease")
	}
	if dbLease.Subnet == nil {
		return nil, errors.New("database did not return a Subnet for this Lease")
	}
	cltt := dbLease.CLTT
	state := uint32(dbLease.State)
	daemonLabel := dbLease.Daemon.GetLabel()
	validLifetime := int64(dbLease.ValidLifetime)
	return &models.Lease{
			ClientID:          dbLease.ClientID.String(),
			Cltt:              &cltt,
			DaemonID:          &dbLease.DaemonID,
			DaemonLabel:       &daemonLabel,
			Duid:              dbLease.DUID.String(),
			FqdnFwd:           dbLease.FqdnFwd,
			FqdnRev:           dbLease.FqdnRev,
			Hostname:          dbLease.Hostname,
			HwAddress:         dbLease.HWAddress,
			Iaid:              int64(dbLease.IAID),
			ID:                &dbLease.ID,
			IPAddress:         &dbLease.IPAddress,
			LeaseType:         dbLease.Type,
			PreferredLifetime: int64(dbLease.PreferredLifetime),
			PrefixLength:      int64(dbLease.PrefixLength),
			State:             &state,
			SubnetID:          &dbLease.Subnet.ID,
			SubnetPrefix:      dbLease.Subnet.Prefix,
			UserContext:       dbLease.UserContext,
			ValidLifetime:     &validLifetime,
		},
		nil
}

// convertSortFieldToColumnName converts the friendly REST API field names
// to the underlying database column names. If the sortField is unrecognized,
// it defaults to [dbmodel.GetLeasesByPageSortColumnNameNone].
func convertSortFieldToColumnName(sortField string) dbmodel.GetLeasesByPageSortColumnName {
	switch models.LeaseListSortField(sortField) {
	case models.LeaseListSortFieldHwAddress:
		return dbmodel.GetLeasesByPageSortColumnNameHwAddress
	case models.LeaseListSortFieldIPAddress:
		return dbmodel.GetLeasesByPageSortColumnNameIPAddress
	case models.LeaseListSortFieldHostname:
		return dbmodel.GetLeasesByPageSortColumnNameHostname
	case models.LeaseListSortFieldClientID:
		return dbmodel.GetLeasesByPageSortColumnNameClientID
	case models.LeaseListSortFieldDuid:
		return dbmodel.GetLeasesByPageSortColumnNameDuid
	case models.LeaseListSortFieldCltt:
		return dbmodel.GetLeasesByPageSortColumnNameCltt
	case models.LeaseListSortFieldValidLifetime:
		return dbmodel.GetLeasesByPageSortColumnNameValidLifetime
	case models.LeaseListSortFieldPrefixLength:
		return dbmodel.GetLeasesByPageSortColumnNamePrefixLength
	default:
		return dbmodel.GetLeasesByPageSortColumnNameNone
	}
}

// Fetches leases from the database and converts to the data formats
// used in the REST API.
func (r *RestAPI) getLeases(offset, limit int64, filters dbmodel.LeasesByPageFilters, sortField string, sortDir dbmodel.SortDirEnum) (*models.Leases, error) {
	dbSortCol := convertSortFieldToColumnName(sortField)
	// Get the leases from the database.
	dbLeases, total, err := dbmodel.GetLeasesByPage(r.DB, offset, limit, filters, dbSortCol, sortDir)
	if err != nil {
		return nil, err
	}

	leasesResponse := &models.Leases{
		Total: total,
		Items: make([]*models.Lease, 0, total),
	}

	// Convert leases fetched from the database to REST.
	for i := range dbLeases {
		lease, err := convertLeaseToRestAPI(&dbLeases[i])
		if err != nil {
			continue
		}
		leasesResponse.Items = append(leasesResponse.Items, lease)
	}

	return leasesResponse, nil
}

// GetLeaseList retreives a list of [dbmodel.Lease] from the database, converts
// them all to [model.Lease], and supports several filtering and sorting
// options.  It implements the /api/dhcp/lease-list endpoint.
func (r *RestAPI) GetLeaseList(ctx context.Context, params dhcp.GetLeaseListParams) middleware.Responder {
	_, user := r.SessionManager.Logged(ctx)
	if user == nil {
		msg := "Unable to identify the user requesting leases (for access control purposes)"
		rsp := dhcp.NewGetLeaseListDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if !user.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) &&
		!user.InGroup(&dbmodel.SystemGroup{ID: dbmodel.AdminGroupID}) {
		msg := "User is forbidden to access lease information"
		rsp := dhcp.NewGetLeaseListDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
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
		msg := "Problem fetching leases from the database"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetLeaseListDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	rsp := dhcp.NewGetLeaseListOK().WithPayload(leases)
	return rsp
}
