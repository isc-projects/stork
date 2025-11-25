package restservice

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/daemons/kea"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
)

// This call searches for leases allocated by monitored DHCP servers.
// The text parameter may contain an IP address, delegated prefix,
// MAC address, client identifier, hostname or the text state:declined.
// The Stork Server tries to identify the specified value type and
// sends queries to the Kea servers to find a lease or multiple leases.
func (r *RestAPI) GetLeases(ctx context.Context, params dhcp.GetLeasesParams) middleware.Responder {
	leases := &models.Leases{
		Total: 0,
	}
	var (
		text   string
		hostID int64
	)
	if params.Text != nil {
		text = strings.TrimSpace(*params.Text)
	}
	if params.HostID != nil {
		hostID = *params.HostID
	}
	if len(text) > 0 && hostID > 0 {
		msg := "Text and host identifier are mutually exclusive when searching for leases"
		log.Error(msg)
		rsp := dhcp.NewGetLeasesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if len(text) == 0 && hostID == 0 {
		// There is nothing to do if none if the parameters were specified.
		rsp := dhcp.NewGetLeasesOK().WithPayload(leases)
		return rsp
	}

	// Try to find the leases from monitored Kea servers.
	var (
		keaLeases    []dbmodel.Lease
		conflicts    []int64
		erredDaemons []*dbmodel.Daemon
		err          error
	)
	if len(text) > 0 {
		// Handle a special case when user specified state:declined search text
		// to find declined leases.
		if ok, _ := regexp.MatchString(`^state:\s*declined$`, text); ok {
			keaLeases, erredDaemons, err = kea.FindDeclinedLeases(r.DB, r.Agents)
		} else {
			keaLeases, erredDaemons, err = kea.FindLeases(r.DB, r.Agents, text)
		}
	} else {
		keaLeases, conflicts, erredDaemons, err = kea.FindLeasesByHostID(r.DB, r.Agents, hostID)
	}
	if err != nil {
		msg := "Problem searching leases on Kea servers due to Stork database errors"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetLeasesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Return leases over the REST API.
	for i := range keaLeases {
		l := keaLeases[i]
		var appName string
		var appID int64
		if l.Daemon != nil {
			app := l.Daemon.GetVirtualApp()
			appName = app.Name
			appID = app.ID
		}
		cltt := int64(l.CLTT)
		state := int64(l.State)
		subnetID := int64(l.SubnetID)
		validLifetime := int64(l.ValidLifetime)

		// Handle a special case when returned DUID is equal to 00. Kea returns such DUID
		// in declined DHCPv6 leases. We treat is as empty DUID.
		duid := ""
		if len(l.DUID) > 0 && l.DUID != "00" {
			duid = l.DUID
		}
		lease := models.Lease{
			ID:                &l.ID,
			AppID:             &appID,
			AppName:           &appName,
			ClientID:          l.ClientID,
			Cltt:              &cltt,
			Duid:              duid,
			FqdnFwd:           l.FqdnFwd,
			FqdnRev:           l.FqdnRev,
			Hostname:          l.Hostname,
			HwAddress:         l.HWAddress,
			Iaid:              int64(l.IAID),
			IPAddress:         &l.IPAddress,
			LeaseType:         l.Type,
			PreferredLifetime: int64(l.PreferredLifetime),
			PrefixLength:      int64(l.PrefixLength),
			State:             &state,
			SubnetID:          &subnetID,
			ValidLifetime:     &validLifetime,
			UserContext:       l.UserContext,
		}
		leases.Items = append(leases.Items, &lease)
	}

	// Record conflicting leases and leases count.
	leases.Conflicts = append(leases.Conflicts, conflicts...)
	leases.Total = int64(len(leases.Items))

	// Record apps for which there was an error communicating with the Kea servers.
	erredApps := map[int64]string{}
	for _, daemon := range erredDaemons {
		app := daemon.GetVirtualApp()
		erredApps[app.ID] = app.Name
	}

	for id, name := range erredApps {
		leases.ErredApps = append(leases.ErredApps, &models.LeasesSearchErredApp{
			ID:   &id,
			Name: &name,
		})
	}

	rsp := dhcp.NewGetLeasesOK().WithPayload(leases)
	return rsp
}
