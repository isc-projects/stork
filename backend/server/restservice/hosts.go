package restservice

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storkutil "isc.org/stork/util"
)

// Converts host reservation fetched from the database to the format
// used in REST API.
func hostToRestAPI(dbHost *dbmodel.Host) *models.Host {
	host := &models.Host{
		ID:       dbHost.ID,
		SubnetID: dbHost.SubnetID,
		Hostname: dbHost.Hostname,
	}
	// Include subnet prefix if this is subnet specific host.
	if dbHost.Subnet != nil {
		host.SubnetPrefix = dbHost.Subnet.Prefix
	}
	// Convert DHCP host identifiers.
	for _, dbHostID := range dbHost.HostIdentifiers {
		hostID := models.HostIdentifier{
			IDType:     dbHostID.Type,
			IDHexValue: dbHostID.ToHex(":"),
		}
		host.HostIdentifiers = append(host.HostIdentifiers, &hostID)
	}
	// Convert IP reservations.
	for _, dbHostIP := range dbHost.IPReservations {
		ip, prefix, ok := storkutil.ParseIP(dbHostIP.Address)
		if !ok {
			continue
		}
		hostIP := models.IPReservation{
			Address: ip,
		}
		if prefix {
			host.PrefixReservations = append(host.PrefixReservations, &hostIP)
		} else {
			host.AddressReservations = append(host.AddressReservations, &hostIP)
		}
	}
	// Append local hosts containing associations of the host with
	// apps.
	for _, dbLocalHost := range dbHost.LocalHosts {
		localHost := models.LocalHost{
			AppID:      dbLocalHost.AppID,
			AppName:    dbLocalHost.App.Name,
			DataSource: dbLocalHost.DataSource,
		}
		host.LocalHosts = append(host.LocalHosts, &localHost)
	}
	return host
}

// Fetches host reservations from the database and converts to the data formats
// used in REST API.
func (r *RestAPI) getHosts(offset, limit, appID int64, subnetID *int64, filterText *string, global *bool, sortField string, sortDir dbmodel.SortDirEnum) (*models.Hosts, error) {
	// Get the hosts from the database.
	dbHosts, total, err := dbmodel.GetHostsByPage(r.DB, offset, limit, appID, subnetID, filterText, global, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	hosts := &models.Hosts{
		Total: total,
	}

	// Convert hosts fetched from the database to REST.
	for i := range dbHosts {
		host := hostToRestAPI(&dbHosts[i])
		hosts.Items = append(hosts.Items, host)
	}

	return hosts, nil
}

// Get list of hosts with specifying an offset and a limit. The hosts can be fetched
// for a given subnet and with filtering by search text.
func (r *RestAPI) GetHosts(ctx context.Context, params dhcp.GetHostsParams) middleware.Responder {
	var start int64 = 0
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	var appID int64 = 0
	if params.AppID != nil {
		appID = *params.AppID
	}

	// get hosts from db
	hosts, err := r.getHosts(start, limit, appID, params.SubnetID, params.Text, params.Global, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "problem with fetching hosts from the database"
		log.Error(err)
		rsp := dhcp.NewGetHostsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Everything fine.
	rsp := dhcp.NewGetHostsOK().WithPayload(hosts)
	return rsp
}

// Get a host by ID.
func (r *RestAPI) GetHost(ctx context.Context, params dhcp.GetHostParams) middleware.Responder {
	// Find a host in the database.
	dbHost, err := dbmodel.GetHost(r.DB, params.ID)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("problem with fetching host reservation with id %d from db", params.ID)
		log.Error(err)
		rsp := dhcp.NewGetHostDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbHost == nil {
		// Host not found.
		msg := fmt.Sprintf("cannot find host reservation with id %d", params.ID)
		rsp := dhcp.NewGetHostDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Host found. Convert it to the format used in REST API.
	host := hostToRestAPI(dbHost)
	rsp := dhcp.NewGetHostOK().WithPayload(host)
	return rsp
}
