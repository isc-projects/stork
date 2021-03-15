package restservice

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
)

// This call searches for leases allocated by monitored DHCP servers.
// The text parameter may contain an IP address, delegated prefix,
// MAC address, client identifier, or hostname. The Stork server
// tries to identify the specified value type and sends queries to
// the Kea servers to find a lease or multiple leases.
func (r *RestAPI) GetLeases(ctx context.Context, params dhcp.GetLeasesParams) middleware.Responder {
	leases := &models.Leases{
		Total: 0,
	}
	var text string
	if params.Text != nil {
		text = strings.TrimSpace(*params.Text)
	}
	if len(text) == 0 {
		// There is nothing to do if search text is empty.
		rsp := dhcp.NewGetLeasesOK().WithPayload(leases)
		return rsp
	}

	// Try to find the leases from monitored Kea servers.
	keaLeases, err := kea.FindLeases(r.DB, r.Agents, text)
	if err != nil {
		msg := "problem with fetching leases from the database"
		log.Error(err)
		rsp := dhcp.NewGetLeasesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Return leases over the REST API.
	for _, l := range keaLeases {
		lease := models.Lease{
			AppID:             l.AppID,
			Cltt:              int64(l.Cltt),
			Duid:              l.DUID,
			Hostname:          l.Hostname,
			HwAddress:         l.HWAddress,
			Iaid:              int64(l.IAID),
			IPAddress:         l.IPAddress,
			LeaseType:         l.Type,
			PreferredLifetime: int64(l.PreferredLifetime),
			PrefixLength:      int64(l.PrefixLength),
			State:             int64(l.State),
			SubnetID:          int64(l.SubnetID),
			ValidLifetime:     int64(l.ValidLifetime),
		}
		leases.Items = append(leases.Items, &lease)
	}

	leases.Total = int64(len(leases.Items))

	rsp := dhcp.NewGetLeasesOK().WithPayload(leases)
	return rsp
}
