package restservice

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/search"
)

// Search through different tables in database.
func (r *RestAPI) SearchRecords(ctx context.Context, params search.SearchRecordsParams) middleware.Responder {
	// get list of subnets
	subnets, err := r.getSubnets(0, 5, 0, 0, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "cannot get subnets from the db"
		log.Error(err)
		rsp := search.NewSearchRecordsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get list of shared networks
	sharedNetworks, err := r.getSharedNetworks(0, 5, 0, 0, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "cannot get shared networks from the db"
		log.Error(err)
		rsp := search.NewSearchRecordsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get list of hosts
	hosts, err := r.getHosts(0, 5, 0, nil, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "cannot get hosts from the db"
		log.Error(err)
		rsp := search.NewSearchRecordsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get list of machines
	machines, err := r.getMachines(0, 5, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "cannot get machines from the db"
		log.Error(err)
		rsp := search.NewSearchRecordsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get list of apps
	apps, err := r.getApps(0, 5, params.Text, "", "", dbmodel.SortDirAny)
	if err != nil {
		msg := "cannot get apps from the db"
		log.Error(err)
		rsp := search.NewSearchRecordsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get list of users
	users, err := r.getUsers(0, 5, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "cannot get users from the db"
		log.Error(err)
		rsp := search.NewSearchRecordsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get list of groups
	groups, err := r.getGroups(0, 5, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "cannot get groups from the db"
		log.Error(err)
		rsp := search.NewSearchRecordsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// combine gathered information
	result := &models.SearchResult{
		Subnets:        subnets,
		SharedNetworks: sharedNetworks,
		Hosts:          hosts,
		Machines:       machines,
		Apps:           apps,
		Users:          users,
		Groups:         groups,
	}

	rsp := search.NewSearchRecordsOK().WithPayload(result)
	return rsp
}
