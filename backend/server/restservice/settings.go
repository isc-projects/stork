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
		log.WithError(err).Error(msg)
		rsp := settings.NewGetSettingsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	s := &models.Settings{
		Bind9StatsPullerInterval:     dbSettingsMap["bind9_stats_puller_interval"].(int64),
		GrafanaURL:                   dbSettingsMap["grafana_url"].(string),
		GrafanaDhcp4DashboardID:      dbSettingsMap["grafana_dhcp4_dashboard_id"].(string),
		GrafanaDhcp6DashboardID:      dbSettingsMap["grafana_dhcp6_dashboard_id"].(string),
		KeaHostsPullerInterval:       dbSettingsMap["kea_hosts_puller_interval"].(int64),
		KeaStatsPullerInterval:       dbSettingsMap["kea_stats_puller_interval"].(int64),
		KeaStatusPullerInterval:      dbSettingsMap["kea_status_puller_interval"].(int64),
		AppsStatePullerInterval:      dbSettingsMap["state_puller_interval"].(int64),
		EnableMachineRegistration:    dbSettingsMap["enable_machine_registration"].(bool),
		EnableOnlineSoftwareVersions: dbSettingsMap["enable_online_software_versions"].(bool),
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
		log.WithError(err).Error("Cannot update bind9_stats_puller_interval")
		return errRsp
	}
	err = dbmodel.SetSettingStr(r.DB, "grafana_url", s.GrafanaURL)
	if err != nil {
		log.WithError(err).Error("Cannot update grafana_url")
		return errRsp
	}
	err = dbmodel.SetSettingStr(r.DB, "grafana_dhcp4_dashboard_id", s.GrafanaDhcp4DashboardID)
	if err != nil {
		log.WithError(err).Error("Cannot update grafana_dhcp4_dashboard_id")
		return errRsp
	}
	err = dbmodel.SetSettingStr(r.DB, "grafana_dhcp6_dashboard_id", s.GrafanaDhcp6DashboardID)
	if err != nil {
		log.WithError(err).Error("Cannot update grafana_dhcp6_dashboard_id")
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.DB, "kea_hosts_puller_interval", s.KeaHostsPullerInterval)
	if err != nil {
		log.WithError(err).Error("Cannot update kea_hosts_puller_interval")
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.DB, "kea_stats_puller_interval", s.KeaStatsPullerInterval)
	if err != nil {
		log.WithError(err).Error("Cannot update kea_stats_puller_interval")
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.DB, "kea_status_puller_interval", s.KeaStatusPullerInterval)
	if err != nil {
		log.WithError(err).Error("Cannot update kea_status_puller_interval")
		return errRsp
	}
	err = dbmodel.SetSettingInt(r.DB, "state_puller_interval", s.AppsStatePullerInterval)
	if err != nil {
		log.WithError(err).Error("Cannot update state_puller_interval")
		return errRsp
	}
	err = dbmodel.SetSettingBool(r.DB, "enable_machine_registration", s.EnableMachineRegistration)
	if err != nil {
		log.WithError(err).Error("Cannot update enable_machine_registration")
		return errRsp
	}
	err = dbmodel.SetSettingBool(r.DB, "enable_online_software_versions", s.EnableOnlineSoftwareVersions)
	if err != nil {
		log.WithError(err).Error("Cannot update enable_online_software_versions")
		return errRsp
	}
	r.EndpointControl.SetEnabled(EndpointOpCreateNewMachine, s.EnableMachineRegistration)

	rsp := settings.NewUpdateSettingsOK()
	return rsp
}
