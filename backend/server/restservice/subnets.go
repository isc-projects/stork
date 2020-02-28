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
)

// Get list of DHCP subnets. The list can be filtered by app ID, DHCP version and text.
func (r *RestAPI) GetSubnets(ctx context.Context, params dhcp.GetSubnetsParams) middleware.Responder {
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

	var dhcpVer int64 = 0
	if params.DhcpVersion != nil {
		dhcpVer = *params.DhcpVersion
	}

	// get subnets from db
	dbSubnets, total, err := dbmodel.GetSubnetsByPage(r.Db, start, limit, appID, dhcpVer, params.Text)
	if err != nil {
		msg := "cannot get subnets from db"
		log.Error(err)
		rsp := dhcp.NewGetSubnetsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// prepare response
	subnets := models.Subnets{
		Total: total,
	}

	// todo: This logic has to change. According to the new data model, there is
	// a single instance of a subnet and multiple apps attached to it. The way
	// we do it currently is to iterate over the local subnets (and apps)
	// associated with the global subnet and return them individually. Changing
	// the current logic requires reworking the UI.
	for _, sn := range dbSubnets {
		pools := []string{}
		for _, poolDetails := range sn.AddressPools {
			pool := poolDetails.LowerBound + "-" + poolDetails.UpperBound
			pools = append(pools, pool)
		}
		var sharedNetworkName string
		if sn.SharedNetwork != nil {
			sharedNetworkName = sn.SharedNetwork.Name
		}

		for _, ls := range sn.LocalSubnets {
			subnet := &models.Subnet{
				AppID:          ls.App.ID,
				ID:             ls.LocalSubnetID,
				Pools:          pools,
				Subnet:         sn.Prefix,
				SharedNetwork:  sharedNetworkName,
				MachineAddress: fmt.Sprintf("%s:%d", ls.App.CtrlAddress, ls.App.CtrlPort),
			}
			subnets.Items = append(subnets.Items, subnet)
		}
	}

	rsp := dhcp.NewGetSubnetsOK().WithPayload(&subnets)
	return rsp
}

// Get list of DHCP shared networks. The list can be filtered by app ID, DHCP version and text.
func (r *RestAPI) GetSharedNetworks(ctx context.Context, params dhcp.GetSharedNetworksParams) middleware.Responder {
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

	var dhcpVer int64 = 0
	if params.DhcpVersion != nil {
		dhcpVer = *params.DhcpVersion
	}

	// get shared networks from db
	dbSharedNetworks, total, err := dbmodel.GetSharedNetworksByPage(r.Db, start, limit, appID, dhcpVer, params.Text)
	if err != nil {
		msg := fmt.Sprintf("cannot get shared network from db")
		log.Error(err)
		rsp := dhcp.NewGetSharedNetworksDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// prepare response
	sharedNetworks := models.SharedNetworks{
		Total: total,
	}

	// todo: This logic has to change. According to the new data model, there is
	// a single instance of a shared network and multiple apps attached to it.
	// Currently we mostly assume that each shared network is served by individual
	// server and we map the app id to the shared network. This will be reworked
	// but changes to the UI are required.
	for _, net := range dbSharedNetworks {
		if len(net.Subnets) == 0 || len(net.Subnets[0].LocalSubnets) == 0 {
			continue
		}
		subnets := []string{}
		// Exclude the subnets that are not attached to any app. This shouldn't
		// be the case but let's be safe.
		for _, subnet := range net.Subnets {
			if len(subnet.LocalSubnets) == 0 {
				continue
			}
			subnets = append(subnets, subnet.Prefix)
		}
		// Create shared network and use the app id of the first subnet found.
		sharedNetwork := &models.SharedNetwork{
			Name:    net.Name,
			AppID:   net.Subnets[0].LocalSubnets[0].AppID,
			Subnets: subnets,
			MachineAddress: fmt.Sprintf("%s:%d", net.Subnets[0].LocalSubnets[0].App.CtrlAddress,
				net.Subnets[0].LocalSubnets[0].App.CtrlPort),
		}
		sharedNetworks.Items = append(sharedNetworks.Items, sharedNetwork)
	}

	rsp := dhcp.NewGetSharedNetworksOK().WithPayload(&sharedNetworks)
	return rsp
}
