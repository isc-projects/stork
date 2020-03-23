package restservice

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storkutil "isc.org/stork/util"
)

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

	// Get the hosts from the database.
	dbHosts, total, err := dbmodel.GetHostsByPage(r.Db, start, limit, params.SubnetID, params.Text)
	if err != nil {
		msg := "problem with fetching hosts from the database"
		log.Error(err)
		rsp := dhcp.NewGetHostsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	hosts := models.Hosts{
		Total: total,
	}

	// Convert hosts fetched from the database to REST.
	for _, dbHost := range dbHosts {
		host := models.Host{
			ID:       dbHost.ID,
			SubnetID: dbHost.SubnetID,
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
		hosts.Items = append(hosts.Items, &host)
	}

	// Evernything fine.
	rsp := dhcp.NewGetHostsOK().WithPayload(&hosts)
	return rsp
}
