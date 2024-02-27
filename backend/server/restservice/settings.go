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
	dbSettingsMap, err := dbmodel.GetAllSettings(r.DB)
	if err != nil {
		msg := "Cannot get global settings"
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
		MetricsCollectorInterval: dbSettingsMap["metrics_collector_interval"].(int64),
	}
	rsp := settings.NewGetSettingsOK().WithPayload(s)

	return rsp
}

// Update global settings.
func (r *RestAPI) UpdateSettings(ctx context.Context, params settings.UpdateSettingsParams) middleware.Responder {
	s := params.Settings
	if s == nil {
		msg := "Missing settings"
		log.Error(msg)
		rsp := settings.NewGetSettingsDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	msg := "Problem updating settings"
	errRsp := settings.NewGetSettingsDefault(http.StatusBadRequest).WithPayload(&models.APIError{
		Message: &msg,
	})

	err := dbmodel.SetSettingInt(r.DB, "bind9_stats_puller_interval", s.Bind9StatsPullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingStr(r.DB, "grafana_url", s.GrafanaURL)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.DB, "kea_hosts_puller_interval", s.KeaHostsPullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.DB, "kea_stats_puller_interval", s.KeaStatsPullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.DB, "kea_status_puller_interval", s.KeaStatusPullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.DB, "apps_state_puller_interval", s.AppsStatePullerInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingStr(r.DB, "prometheus_url", s.PrometheusURL)
	if err != nil {
		log.Error(err)
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.DB, "metrics_collector_interval", s.MetricsCollectorInterval)
	if err != nil {
		log.Error(err)
		return errRsp
	}

	rsp := settings.NewUpdateSettingsOK()
	return rsp
}
