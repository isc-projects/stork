package restservice

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
	dbmodel "isc.org/stork/server/database/model"

	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
)

func subnetToRestAPI(sn *dbmodel.Subnet) *models.Subnet {
	subnet := &models.Subnet{
		ID:              sn.ID,
		Subnet:          sn.Prefix,
		ClientClass:     sn.ClientClass,
		AddrUtilization: float64(sn.AddrUtilization) / 10,
	}

	for _, poolDetails := range sn.AddressPools {
		pool := poolDetails.LowerBound + "-" + poolDetails.UpperBound
		subnet.Pools = append(subnet.Pools, pool)
	}

	if sn.SharedNetwork != nil {
		subnet.SharedNetwork = sn.SharedNetwork.Name
	}

	for _, lsn := range sn.LocalSubnets {
		localSubnet := &models.LocalSubnet{
			AppID:            lsn.App.ID,
			AppName:          lsn.App.Name,
			ID:               lsn.LocalSubnetID,
			MachineAddress:   lsn.App.Machine.Address,
			MachineHostname:  lsn.App.Machine.State.Hostname,
			Stats:            lsn.Stats,
			StatsCollectedAt: strfmt.DateTime(lsn.StatsCollectedAt),
		}
		subnet.LocalSubnets = append(subnet.LocalSubnets, localSubnet)
	}
	return subnet
}

func (r *RestAPI) getSubnets(offset, limit, appID, family int64, filterText *string, sortField string, sortDir dbmodel.SortDirEnum) (*models.Subnets, error) {
	// get subnets from db
	dbSubnets, total, err := dbmodel.GetSubnetsByPage(r.DB, offset, limit, appID, family, filterText, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	// prepare response
	subnets := &models.Subnets{
		Total: total,
	}

	// go through subnets from db and change their format to ReST one
	for _, snTmp := range dbSubnets {
		sn := snTmp
		subnet := subnetToRestAPI(&sn)
		subnets.Items = append(subnets.Items, subnet)
	}

	return subnets, nil
}

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
	subnets, err := r.getSubnets(start, limit, appID, dhcpVer, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "cannot get subnets from db"
		log.Error(err)
		rsp := dhcp.NewGetSubnetsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewGetSubnetsOK().WithPayload(subnets)
	return rsp
}

func (r *RestAPI) getSharedNetworks(offset, limit, appID, family int64, filterText *string, sortField string, sortDir dbmodel.SortDirEnum) (*models.SharedNetworks, error) {
	// get shared networks from db
	dbSharedNetworks, total, err := dbmodel.GetSharedNetworksByPage(r.DB, offset, limit, appID, family, filterText, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	// prepare response
	sharedNetworks := &models.SharedNetworks{
		Total: total,
	}

	// go through shared networks and their subnets from db and change their format to ReST one
	for _, net := range dbSharedNetworks {
		if len(net.Subnets) == 0 || len(net.Subnets[0].LocalSubnets) == 0 {
			continue
		}
		subnets := []*models.Subnet{}
		// Exclude the subnets that are not attached to any app. This shouldn't
		// be the case but let's be safe.
		for _, snTmp := range net.Subnets {
			sn := snTmp
			subnet := subnetToRestAPI(&sn)
			subnets = append(subnets, subnet)
		}
		// Create shared network.
		sharedNetwork := &models.SharedNetwork{
			ID:              net.ID,
			Name:            net.Name,
			Subnets:         subnets,
			AddrUtilization: float64(net.AddrUtilization) / 10,
		}
		sharedNetworks.Items = append(sharedNetworks.Items, sharedNetwork)
	}

	return sharedNetworks, nil
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
	sharedNetworks, err := r.getSharedNetworks(start, limit, appID, dhcpVer, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "cannot get shared network from db"
		log.Error(err)
		rsp := dhcp.NewGetSharedNetworksDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	rsp := dhcp.NewGetSharedNetworksOK().WithPayload(sharedNetworks)
	return rsp
}
