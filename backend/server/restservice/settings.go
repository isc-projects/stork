package restservice

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/settings"
)

// Get global settings.
func (r *RestAPI) GetSettings(ctx context.Context, params settings.GetSettingsParams) middleware.Responder {
	dbSettingsMap, err := dbmodel.GetAllSettings(r.Db)
	if err != nil {
		msg := "cannot get global settings"
		log.Error(err)
		rsp := settings.NewGetSettingsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	s := &models.Settings{
		Bind9StatsPullerInterval: dbSettingsMap["bind9_stats_puller_interval"].(int64),
		GrafanaURL:               dbSettingsMap["grafana_url"].(string),
		KeaHostsPullerInterval:   dbSettingsMap["kea_hosts_puller_interval"].(int64),
		KeaStatsPullerInterval:   dbSettingsMap["kea_stats_puller_interval"].(int64),
		KeaStatusPullerInterval:  dbSettingsMap["kea_status_puller_interval"].(int64),
		AppsStatePullerInterval:  dbSettingsMap["apps_state_puller_interval"].(int64),
		PrometheusURL:            dbSettingsMap["prometheus_url"].(string),
	}
	rsp := settings.NewGetSettingsOK().WithPayload(s)

	return rsp
}

// Update global settings.
func (r *RestAPI) UpdateSettings(ctx context.Context, params settings.UpdateSettingsParams) middleware.Responder {
	s := params.Settings
	if s == nil {
		msg := "missing settings"
		log.Error(msg)
		rsp := settings.NewGetSettingsDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	msg := "problem with updating settings"
	errRsp := settings.NewGetSettingsDefault(http.StatusBadRequest).WithPayload(&models.APIError{
		Message: &msg,
	})

	err := dbmodel.SetSettingInt(r.Db, "bind9_stats_puller_interval", s.Bind9StatsPullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingStr(r.Db, "grafana_url", s.GrafanaURL)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.Db, "kea_hosts_puller_interval", s.KeaHostsPullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.Db, "kea_stats_puller_interval", s.KeaStatsPullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.Db, "kea_status_puller_interval", s.KeaStatusPullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.Db, "apps_state_puller_interval", s.KeaStatusPullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingStr(r.Db, "prometheus_url", s.PrometheusURL)
	if err != nil {
		log.Error(err)
		return errRsp
	}

	rsp := settings.NewUpdateSettingsOK()
	return rsp
}
