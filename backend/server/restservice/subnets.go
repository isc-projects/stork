package restservice

import (
	"context"
	"fmt"

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
		rsp := dhcp.NewGetSubnetsDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// prepare response
	subnets := models.Subnets{
		Total: total,
	}
	for _, sn := range dbSubnets {
		pools := []string{}
		for _, poolDetails := range sn.Pools {
			pool, ok := poolDetails["pool"].(string)
			if ok {
				pools = append(pools, pool)
			}
		}
		subnet := &models.Subnet{
			AppID:          int64(sn.AppID),
			ID:             int64(sn.ID),
			Pools:          pools,
			Subnet:         sn.Subnet,
			SharedNetwork:  sn.SharedNetwork,
			MachineAddress: fmt.Sprintf("%s:%d", sn.MachineAddress, sn.AgentPort),
		}
		subnets.Items = append(subnets.Items, subnet)
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
		rsp := dhcp.NewGetSharedNetworksDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// prepare response
	sharedNetworks := models.SharedNetworks{
		Total: total,
	}
	for _, net := range dbSharedNetworks {
		subnets := []string{}
		for _, snDetails := range net.Subnets {
			subnet, ok := snDetails["subnet"].(string)
			if ok {
				subnets = append(subnets, subnet)
			}
		}
		sharedNetwork := &models.SharedNetwork{
			Name:           net.Name,
			AppID:          int64(net.AppID),
			Subnets:        subnets,
			MachineAddress: fmt.Sprintf("%s:%d", net.MachineAddress, net.AgentPort),
		}
		sharedNetworks.Items = append(sharedNetworks.Items, sharedNetwork)
	}

	rsp := dhcp.NewGetSharedNetworksOK().WithPayload(&sharedNetworks)
	return rsp
}
