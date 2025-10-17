package restservice

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"
	bind9config "isc.org/stork/appcfg/bind9"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/services"
)

// Returns the raw configuration for a BIND 9 daemon with optional
// filtering of the configuration elements returned.
func (r *RestAPI) GetBind9RawConfig(ctx context.Context, params services.GetBind9RawConfigParams) middleware.Responder {
	var filter *bind9config.Filter
	if len(params.Filter) > 0 {
		filter = bind9config.NewFilter()
		for _, filterType := range params.Filter {
			filter.Enable(bind9config.FilterType(filterType))
		}
	}
	configFiles, err := r.DNSManager.GetBind9RawConfig(ctx, params.ID, filter)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot get BIND 9 configuration for daemon with ID %d", params.ID)
		rsp := services.NewGetBind9RawConfigDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	var bind9RawConfigFiles []*models.Bind9RawConfigFile
	for _, configFile := range configFiles.Files {
		bind9RawConfigFiles = append(bind9RawConfigFiles, &models.Bind9RawConfigFile{
			SourcePath: configFile.SourcePath,
			FileType:   string(configFile.FileType),
			Contents:   configFile.Contents,
		})
	}
	rsp := services.NewGetBind9RawConfigOK().WithPayload(&models.Bind9RawConfig{
		Files: bind9RawConfigFiles,
	})
	return rsp
}
