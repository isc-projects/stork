package restservice

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/services"
)

// Get daemon config. Only Kea daemon supported.
func (r *RestAPI) GetDaemonConfig(ctx context.Context, params services.GetDaemonConfigParams) middleware.Responder {
	dbDaemon, err := dbmodel.GetDaemonByID(r.DB, params.ID)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("cannot get daemon with id %d from db", params.ID)
		rsp := services.NewGetDaemonConfigDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbDaemon == nil {
		msg := fmt.Sprintf("cannot find daemon with id %d", params.ID)
		rsp := services.NewGetDaemonConfigDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbDaemon.KeaDaemon == nil {
		msg := fmt.Sprintf("daemon with id %d isn't Kea daemon", params.ID)
		rsp := services.NewGetDaemonConfigDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbDaemon.KeaDaemon.Config == nil {
		msg := fmt.Sprintf("config not assigned for daemon with id %d", params.ID)
		rsp := services.NewGetDaemonConfigDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	_, dbUser := r.SessionManager.Logged(ctx)
	if !dbUser.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		hideSensitiveData((*map[string]interface{})(dbDaemon.KeaDaemon.Config))
	}

	rsp := services.NewGetDaemonConfigOK().WithPayload(dbDaemon.KeaDaemon.Config)
	return rsp
}

// Get configuration review reports for a specified daemon. Only Kea
// daemons are supported currently.
func (r *RestAPI) GetDaemonConfigReports(ctx context.Context, params services.GetDaemonConfigReportsParams) middleware.Responder {
	rsp := services.NewGetDaemonConfigReportsOK()
	return rsp
}
