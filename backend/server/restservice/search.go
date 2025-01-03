package restservice

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/search"
)

func handleSearchError(err error, text string) middleware.Responder {
	log.Error(err)
	rsp := search.NewSearchRecordsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
		Message: &text,
	})
	return rsp
}

// Search through different tables in database. Currently supported tables are:
// machines, apps, subnets, shared networks, hosts, users, groups.
// If filter text is empty then empty result is returned.
func (r *RestAPI) SearchRecords(ctx context.Context, params search.SearchRecordsParams) middleware.Responder {
	// if empty text is provided then empty result is returned
	if params.Text == nil || strings.TrimSpace(*params.Text) == "" {
		result := &models.SearchResult{
			Subnets:        &models.Subnets{},
			SharedNetworks: &models.SharedNetworks{},
			Hosts:          &models.Hosts{},
			Machines:       &models.Machines{},
			Apps:           &models.Apps{},
			Users:          &models.Users{},
			Groups:         &models.Groups{},
		}
		rsp := search.NewSearchRecordsOK().WithPayload(result)
		return rsp
	}
	text := strings.TrimSpace(*params.Text)
	filters := &dbmodel.SubnetsByPageFilters{Text: &text}

	// get list of subnets
	subnets, err := r.getSubnets(0, 5, filters, "", dbmodel.SortDirAny)
	if err != nil {
		return handleSearchError(err, "Cannot get subnets from the db")
	}

	// get list of shared networks
	sharedNetworks, err := r.getSharedNetworks(0, 5, 0, 0, &text, "", dbmodel.SortDirAny)
	if err != nil {
		return handleSearchError(err, "Cannot get shared networks from the db")
	}

	// get list of hosts
	hosts, err := r.getHosts(0, 5, dbmodel.HostsByPageFilters{FilterText: &text}, "", dbmodel.SortDirAny)
	if err != nil {
		return handleSearchError(err, "Cannot get hosts from the db")
	}

	// get list of machines no matter if authorized or unauthorized
	machines, err := r.getMachines(0, 5, &text, nil, "", dbmodel.SortDirAny)
	if err != nil {
		return handleSearchError(err, "Cannot get machines from the db")
	}

	// get list of apps
	apps, err := r.getApps(0, 5, &text, "", "", dbmodel.SortDirAny)
	if err != nil {
		return handleSearchError(err, "Cannot get apps from the db")
	}

	// get list of users
	users, err := r.getUsers(0, 5, &text, "", dbmodel.SortDirAny)
	if err != nil {
		return handleSearchError(err, "Cannot get users from the db")
	}

	// get list of groups
	groups, err := r.getGroups(0, 5, &text, "", dbmodel.SortDirAny)
	if err != nil {
		return handleSearchError(err, "Cannot get groups from the db")
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
