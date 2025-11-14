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

// Returns the formatted configuration for a BIND 9 daemon with optional
// filtering of the configuration elements returned.
func (r *RestAPI) GetBind9FormattedConfig(ctx context.Context, params services.GetBind9FormattedConfigParams) middleware.Responder {
	var (
		filter       *bind9config.Filter
		fileSelector *bind9config.FileTypeSelector
	)
	if len(params.Filter) > 0 {
		filter = bind9config.NewFilter()
		for _, filterType := range params.Filter {
			filter.Enable(bind9config.FilterType(filterType))
		}
	}
	if len(params.FileSelector) > 0 {
		fileSelector = bind9config.NewFileTypeSelector()
		for _, fileType := range params.FileSelector {
			fileSelector.Enable(bind9config.FileType(fileType))
		}
	}
	var bind9FormattedConfigFiles []*models.Bind9FormattedConfigFile
	for rsp := range r.DNSManager.GetBind9FormattedConfig(ctx, params.ID, fileSelector, filter) {
		if rsp.Err != nil {
			msg := fmt.Sprintf("Cannot get BIND 9 configuration for daemon with ID %d", params.ID)
			log.WithError(rsp.Err).Error(msg)
			rsp := services.NewGetBind9FormattedConfigDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
		if rsp.File != nil {
			bind9FormattedConfigFiles = append(bind9FormattedConfigFiles, &models.Bind9FormattedConfigFile{
				SourcePath: rsp.File.SourcePath,
				FileType:   string(rsp.File.FileType),
			})
		} else if rsp.Contents != nil {
			if len(bind9FormattedConfigFiles) > 0 {
				contents := bind9FormattedConfigFiles[len(bind9FormattedConfigFiles)-1].Contents
				contents = append(contents, *rsp.Contents)
				bind9FormattedConfigFiles[len(bind9FormattedConfigFiles)-1].Contents = contents
			}
		}
	}
	rsp := services.NewGetBind9FormattedConfigOK().WithPayload(&models.Bind9FormattedConfig{
		Files: bind9FormattedConfigFiles,
	})
	return rsp
}
