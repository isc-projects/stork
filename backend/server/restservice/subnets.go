package restservice

import (
	"context"
	"fmt"
	"strconv"

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
	dbSubnets, err := dbmodel.GetSubnetsByPage(r.Db, start, limit, appID, dhcpVer, params.Text)
	if err != nil {
		msg := fmt.Sprintf("cannot get subnets from db")
		log.Error(err)
		rsp := dhcp.NewGetSubnetsDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// prepare response
	subnets := models.Subnets{}
	for _, sn := range dbSubnets {
		snID, _ := strconv.ParseInt(sn.ID, 10, 64)
		pools := []string{}
		for _, poolDetails := range sn.Pools {
			pool, ok := poolDetails["pool"].(string)
			if ok {
				pools = append(pools, pool)
			}
		}
		subnet := &models.Subnet{
			AppID:  int64(sn.AppID),
			ID:     snID,
			Pools:  pools,
			Subnet: sn.Subnet,
		}
		subnets.Items = append(subnets.Items, subnet)
	}

	rsp := dhcp.NewGetSubnetsOK().WithPayload(&subnets)
	return rsp
}
